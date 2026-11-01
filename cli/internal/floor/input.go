package floor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"mint/internal/execstate"
	"mint/internal/snapshot"
	"mint/internal/specschema"
	"mint/internal/unitstore"
	"mint/internal/verify"
)

var safeRefRE = regexp.MustCompile(`^[A-Za-z0-9_./~^@{}+-]+$`)

type BuildOptions struct {
	SpecPath      string
	Slug          string
	SpecID        string
	VerdictPath   string
	TerminalState string
	AttemptID     string
	Base          string
}

func BuildInput(root string, opts BuildOptions) (Input, error) {
	base := opts.Base
	if base == "" {
		base = "HEAD"
	}
	if base != "HEAD" {
		if err := validateBase(root, base); err != nil {
			return Input{}, err
		}
	}

	specBytes, err := os.ReadFile(opts.SpecPath)
	if err != nil {
		return Input{}, err
	}
	state, _ := execstate.Read(root, opts.Slug, opts.SpecID, opts.AttemptID)
	var changed []string
	var patch, current string
	if opts.Base == "" && state != nil && state.BaselineTree != nil {
		changed, patch, current, err = snapshot.ChangedSince(root, *state.BaselineTree)
	} else {
		changed, err = changedFilePaths(root, base)
		if err != nil {
			return Input{}, err
		}
		patch, err = changedPatch(root, base)
	}
	if err != nil {
		return Input{}, err
	}
	var aside []string
	if current != "" {
		if siblings := siblingLanes(root, opts.Slug, opts.SpecID, state.StartedAt); len(siblings) > 0 {
			changed, aside = SplitSiblingFiles(changed, specschema.ResolveCanModify(string(specBytes)), siblings)
			if len(aside) > 0 {
				if patch, err = snapshot.PatchFor(root, *state.BaselineTree, current, changed); err != nil {
					return Input{}, err
				}
			}
		}
	}

	gates := verify.Run(root, opts.Slug, opts.SpecID, opts.AttemptID, opts.SpecPath)
	if gates.Error != "" {
		return Input{}, fmt.Errorf("verify: %s", gates.Error)
	}
	var inconclusive, failed []string
	for label, result := range gates.Results {
		switch result {
		case "inconclusive":
			inconclusive = append(inconclusive, label+" ("+gates.Reasons[label]+")")
		case "fail":
			failed = append(failed, label+" ("+gates.Reasons[label]+")")
		}
	}
	sort.Strings(inconclusive)
	sort.Strings(failed)
	verdict := ReadVerdict(opts.VerdictPath)

	input := Input{
		Spec:            string(specBytes),
		ChangedFiles:    changed,
		Patch:           patch,
		Gates:           &GateResult{OK: gates.OK, GateMode: gates.Mode, Reverted: gates.Reverted, Inconclusive: inconclusive, Failed: failed},
		Verdict:         verdict,
		TerminalState:   opts.TerminalState,
		RequiredReviews: resolveRequiredReviews(opts.SpecPath),
		Reviews:         attachedReviews(root, opts.Slug, opts.SpecID, opts.AttemptID),
		SetAside:        aside,
	}
	if state != nil && state.Maker != nil {
		input.MakerExecutor = state.Maker.Executor
		input.MakerVendor = state.Maker.Vendor
		input.MakerModel = state.Maker.Model
		input.MakerLocality = state.Maker.Locality
		input.MakerExecutionRef = state.Maker.ExecutionRef
	}
	return input, nil
}

// siblingLanes is the can-modify of every other unit in this worktree with an attempt
// that was open at some point since this attempt started: still running, or completed
// after it. Unit state is worktree-isolated, so List only sees this worktree's units.
// ponytail: an attempt left "running" forever keeps its lane set aside for every later
// attempt in the worktree; mint exec set-status closes it.
func siblingLanes(root, slug, specID, startedAt string) []string {
	since, err := time.Parse(time.RFC3339Nano, startedAt)
	if err != nil {
		return nil
	}
	var lanes []string
	for _, unit := range unitstore.List(root) {
		if unit.Slug == slug && unit.SpecID == specID {
			continue
		}
		open := false
		for _, attempt := range execstate.Attempts(root, unit.Slug, unit.SpecID) {
			st, ok := execstate.Read(root, unit.Slug, unit.SpecID, attempt)
			if !ok {
				continue
			}
			if st.CompletedAt == nil {
				open = open || st.Status == "running"
			} else if done, err := time.Parse(time.RFC3339Nano, *st.CompletedAt); err == nil && done.After(since) {
				open = true
			}
		}
		if !open {
			continue
		}
		if path, ok := unitstore.ResolveSpec(root, unit.Slug, unit.SpecID); ok {
			if spec, err := os.ReadFile(path); err == nil {
				lanes = append(lanes, specschema.ResolveCanModify(string(spec))...)
			}
		}
	}
	return lanes
}

func ReadVerdict(path string) map[string]any {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var parsed any
	if err := json.Unmarshal(b, &parsed); err != nil {
		return nil
	}
	obj, ok := parsed.(map[string]any)
	if !ok {
		return nil
	}
	version, ok := obj["schemaVersion"].(float64)
	if !ok || version != 1 {
		return nil
	}
	if _, ok := obj["accepted"].(bool); !ok {
		return nil
	}
	if _, ok := verdictProvenance(obj); !ok {
		return nil
	}
	return obj
}

func validateBase(root, base string) error {
	if base == "" || strings.HasPrefix(base, "-") || !safeRefRE.MatchString(base) {
		return fmt.Errorf("Invalid --base ref: %s", base)
	}
	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", base+"^{commit}")
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Unresolvable --base ref: %s", base)
	}
	return nil
}

func changedFilePaths(root, base string) ([]string, error) {
	seen := map[string]bool{}
	var files []string

	nameOnly, err := gitOutput(root, "diff", "--name-only", base)
	if err != nil {
		if base != "HEAD" {
			return nil, fmt.Errorf("--base diff failed for ref: %s", base)
		}
		nameOnly = ""
	}
	for _, line := range strings.Split(nameOnly, "\n") {
		p := strings.TrimSpace(line)
		if p != "" && !seen[p] {
			seen[p] = true
			files = append(files, p)
		}
	}

	others, err := gitOutput(root, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		others = ""
	}
	for _, line := range strings.Split(others, "\n") {
		p := strings.TrimSpace(line)
		if p != "" && !seen[p] {
			seen[p] = true
			files = append(files, p)
		}
	}
	return files, nil
}

func changedPatch(root, base string) (string, error) {
	out, err := gitOutput(root, "diff", base)
	if err != nil {
		if base != "HEAD" {
			return "", fmt.Errorf("--base diff failed for ref: %s", base)
		}
		return "", nil
	}
	return out, nil
}

func gitOutput(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func resolveRequiredReviews(specPath string) []string {
	if specPath != "" {
		if b, err := os.ReadFile(specPath); err == nil {
			if reviews := specschema.ResolveSpecReviews(string(b)); len(reviews) > 0 {
				return reviews
			}
		}
	}
	return nil
}

func attachedReviews(root, slug, specID, attemptID string) map[string]execstate.Review {
	state, ok := execstate.Read(root, slug, specID, attemptID)
	if !ok || state.Reviews == nil {
		return map[string]execstate.Review{}
	}
	return state.Reviews
}

func verdictProvenance(verdict map[string]any) (execstate.Provenance, bool) {
	if verdict == nil {
		return execstate.Provenance{}, false
	}
	stringField := func(key string) string {
		value, _ := verdict[key].(string)
		return value
	}
	p, err := execstate.ValidateProvenance(execstate.Provenance{
		Executor: stringField("executor"), Vendor: stringField("vendor"),
		Model: stringField("model"), Locality: stringField("locality"),
		ExecutionRef: stringField("executionRef"), ObservedBy: stringField("observedBy"),
		Attestation: stringField("attestation"),
	})
	return p, err == nil
}

func IsInvalidBaseError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Invalid --base ref")
}

func IsUnresolvableBaseError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Unresolvable --base ref")
}

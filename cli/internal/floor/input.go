package floor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"mint/internal/execstate"
	"mint/internal/specschema"
	"mint/internal/verify"
)

var safeRefRE = regexp.MustCompile(`^[A-Za-z0-9_./~^@{}+-]+$`)

type BuildOptions struct {
	SpecPath      string
	Slug          string
	SpecID        string
	VerdictPath   string
	TerminalState string
	MakerEngine   string
	MakerSession  string
	SessionID     string
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
	changed, err := changedFilePaths(root, base)
	if err != nil {
		return Input{}, err
	}
	patch, err := changedPatch(root, base)
	if err != nil {
		return Input{}, err
	}

	gates := verify.Run(root, opts.Slug, opts.SpecID, opts.SessionID, opts.SpecPath)

	input := Input{
		Spec:            string(specBytes),
		ChangedFiles:    changed,
		Patch:           patch,
		Gates:           &GateResult{OK: gates.OK},
		Verdict:         ReadVerdict(opts.VerdictPath),
		TerminalState:   opts.TerminalState,
		RequiredReviews: resolveRequiredReviews(root, opts.SpecPath, opts.SessionID),
		Reviews:         attachedReviews(root, opts.Slug, opts.SpecID, opts.SessionID),
	}
	if state, ok := execstate.Read(root, opts.Slug, opts.SpecID, opts.SessionID); ok && state.Maker != nil {
		input.MakerEngine = state.Maker.Engine
		input.MakerVendor = state.Maker.Vendor
		input.MakerModel = state.Maker.Model
		input.MakerLocality = state.Maker.Locality
		input.MakerSession = state.Maker.Session
	}
	return input, nil
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
	if _, ok := obj["accepted"].(bool); !ok {
		return nil
	}
	if _, ok := obj["byEngine"].(string); !ok {
		return nil
	}
	if _, ok := obj["bySession"].(string); !ok {
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

func resolveRequiredReviews(root, specPath, sessionID string) []string {
	if specPath != "" {
		if b, err := os.ReadFile(specPath); err == nil {
			if reviews := specschema.ResolveSpecReviews(string(b)); len(reviews) > 0 {
				return reviews
			}
		}
	}
	session, ok := readJSONObject(sessionPath(root, sessionID))
	if !ok {
		return nil
	}
	raw, ok := session["reviews"].([]any)
	if !ok {
		return nil
	}
	var reviews []string
	for _, item := range raw {
		if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
			reviews = append(reviews, s)
		}
	}
	return reviews
}

func attachedReviews(root, slug, specID, sessionID string) map[string]execstate.Review {
	state, ok := execstate.Read(root, slug, specID, sessionID)
	if !ok || state.Reviews == nil {
		return map[string]execstate.Review{}
	}
	return state.Reviews
}

func readJSONObject(path string) (map[string]any, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, false
	}
	obj, ok := v.(map[string]any)
	return obj, ok
}

func sessionPath(root, sessionID string) string {
	if sessionID == "" {
		return filepath.Join(root, ".mint", "sessions", ".json")
	}
	return filepath.Join(root, ".mint", "sessions", sessionID+".json")
}

func IsInvalidBaseError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Invalid --base ref")
}

func IsUnresolvableBaseError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Unresolvable --base ref")
}

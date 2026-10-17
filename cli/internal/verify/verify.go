package verify

import (
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"mint/internal/docclassify"
	"mint/internal/execstate"
	"mint/internal/replay"
	"mint/internal/snapshot"
	"mint/internal/specschema"
)

type ChangedFile struct {
	Path   string
	Status string
}

type Result struct {
	SchemaVersion int               `json:"schemaVersion"`
	Tier          string            `json:"tier"`
	Results       map[string]string `json:"results"`
	OK            bool              `json:"ok"`
	Declared      bool              `json:"declared"`
	Error         string            `json:"error,omitempty"`
	Mode          string            `json:"mode,omitempty"`
	Reverted      []string          `json:"reverted,omitempty"`
}

func Run(root, slug, specID, attemptID, specPath string) (result Result) {
	result = Result{SchemaVersion: 1, Results: map[string]string{}, Mode: "in-place"}
	state, exists := execstate.Read(root, slug, specID, attemptID)
	if !exists {
		result.Error = "attempt does not exist; run mint exec init"
		return result
	}

	var paths []string
	if state.BaselineTree != nil {
		result.Mode = "replay"
		var err error
		paths, _, _, err = snapshot.ChangedSince(root, *state.BaselineTree)
		if err != nil {
			result.Error = err.Error()
			return result
		}
	} else {
		for _, file := range ChangedFiles(root) {
			paths = append(paths, file.Path)
		}
	}
	if docclassify.IsDocsOnly(paths) {
		if _, err := execstate.RecordGate(root, slug, specID, "tier", "skip", attemptID); err != nil {
			result.Error = err.Error()
			return result
		}
		result.Tier, result.OK, result.Declared = "skip", true, true
		return result
	}

	specBytes, _ := os.ReadFile(specPath)
	spec := string(specBytes)
	gates := specschema.ResolveSpecGates(spec)
	labels := make([]string, 0, len(gates))
	for label := range gates {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	if len(labels) == 0 {
		if _, err := execstate.RecordGate(root, slug, specID, "tier", "none", attemptID); err != nil {
			result.Error = err.Error()
			return result
		}
		result.Tier = "none"
		return result
	}

	if _, err := execstate.RecordGate(root, slug, specID, "tier", "full", attemptID); err != nil {
		result.Error = err.Error()
		return result
	}
	result.Tier = "full"
	gateRoot := root
	var prepareErr error
	if state.BaselineTree != nil {
		worktree, err := replay.Prepare(root, *state.BaselineTree, specschema.ResolveProtected(spec))
		if err != nil {
			result.Error = err.Error()
			return result
		}
		defer func() {
			if err := worktree.Close(); err != nil {
				result.OK = false
				result.Error = err.Error()
			}
		}()
		gateRoot, result.Reverted = worktree.Dir, worktree.Reverted
		if prepare := specschema.ResolvePrepare(spec); prepare != "" {
			prepareErr = runGate(gateRoot, prepare, "MINT_SOURCE_ROOT="+root)
		}
	}
	result.OK = true
	for _, label := range labels {
		command := gates[label]
		if strings.TrimSpace(command) == "" {
			continue
		}
		gateResult := "pass"
		if prepareErr != nil || runGate(gateRoot, command) != nil {
			gateResult = "fail"
			result.OK = false
		}
		result.Results[label] = gateResult
		if _, err := execstate.RecordGate(root, slug, specID, label, gateResult, attemptID); err != nil {
			result.OK = false
			result.Error = err.Error()
			return result
		}
	}
	result.Declared = true
	return result
}

func ChangedFiles(root string) []ChangedFile {
	seen := map[string]bool{}
	files := []ChangedFile{}
	for _, line := range strings.Split(gitOutput(root, "diff", "--name-only", "HEAD"), "\n") {
		p := strings.TrimSpace(line)
		if p != "" && !seen[p] {
			seen[p] = true
			files = append(files, ChangedFile{Path: p, Status: "modified"})
		}
	}
	for _, line := range strings.Split(gitOutput(root, "ls-files", "--others", "--exclude-standard"), "\n") {
		p := strings.TrimSpace(line)
		if p != "" && !seen[p] {
			seen[p] = true
			files = append(files, ChangedFile{Path: p, Status: "new"})
		}
	}
	return files
}

func ResolveGates(specPath string) map[string]string {
	if specPath != "" {
		if b, err := os.ReadFile(specPath); err == nil {
			if gates := specschema.ResolveSpecGates(string(b)); len(gates) > 0 {
				return gates
			}
		}
	}
	return map[string]string{}
}

func runGate(root, command string, env ...string) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Dir = root
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	return cmd.Run()
}

func gitOutput(root string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

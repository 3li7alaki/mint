package verify

import (
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"mint/internal/docclassify"
	"mint/internal/execstate"
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
}

func Run(root, slug, specID, attemptID, specPath string) Result {
	if _, ok := execstate.Read(root, slug, specID, attemptID); !ok {
		return Result{SchemaVersion: 1, Results: map[string]string{}, Error: "attempt does not exist; run mint exec init"}
	}

	files := ChangedFiles(root)
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	if docclassify.IsDocsOnly(paths) {
		if _, err := execstate.RecordGate(root, slug, specID, "tier", "skip", attemptID); err != nil {
			return Result{SchemaVersion: 1, Results: map[string]string{}, Error: err.Error()}
		}
		return Result{SchemaVersion: 1, Tier: "skip", Results: map[string]string{}, OK: true, Declared: true}
	}

	gates := ResolveGates(specPath)
	labels := make([]string, 0, len(gates))
	for label := range gates {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	if len(labels) == 0 {
		if _, err := execstate.RecordGate(root, slug, specID, "tier", "none", attemptID); err != nil {
			return Result{SchemaVersion: 1, Results: map[string]string{}, Error: err.Error()}
		}
		return Result{SchemaVersion: 1, Tier: "none", Results: map[string]string{}, OK: false, Declared: false}
	}

	if _, err := execstate.RecordGate(root, slug, specID, "tier", "full", attemptID); err != nil {
		return Result{SchemaVersion: 1, Results: map[string]string{}, Error: err.Error()}
	}
	results := map[string]string{}
	ok := true
	for _, label := range labels {
		cmd := gates[label]
		if strings.TrimSpace(cmd) == "" {
			continue
		}
		result := "pass"
		if err := runGate(root, cmd); err != nil {
			result = "fail"
			ok = false
		}
		results[label] = result
		if _, err := execstate.RecordGate(root, slug, specID, label, result, attemptID); err != nil {
			return Result{SchemaVersion: 1, Tier: "full", Results: results, Error: err.Error()}
		}
	}
	return Result{SchemaVersion: 1, Tier: "full", Results: results, OK: ok, Declared: true}
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

func runGate(root, command string) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Dir = root
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

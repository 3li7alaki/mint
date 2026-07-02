package verify

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"mint/internal/docclassify"
	"mint/internal/execstate"
	"mint/internal/session"
	"mint/internal/specschema"
)

type ChangedFile struct {
	Path   string
	Status string
}

type Result struct {
	Tier     string            `json:"tier"`
	Results  map[string]string `json:"results"`
	OK       bool              `json:"ok"`
	Declared bool              `json:"declared"`
}

func Run(root, slug, specID, sessionID, specPath string) Result {
	if _, ok := execstate.Read(root, slug, specID, sessionID); !ok {
		_, _ = execstate.Init(root, slug, specID, sessionID, nil)
	}

	files := ChangedFiles(root)
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	if docclassify.IsDocsOnly(paths) {
		_, _ = execstate.RecordGate(root, slug, specID, "tier", "skip", sessionID)
		return Result{Tier: "skip", Results: map[string]string{}, OK: true, Declared: true}
	}

	gates := ResolveGates(root, sessionID, specPath)
	labels := make([]string, 0, len(gates))
	for label := range gates {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	if len(labels) == 0 {
		_, _ = execstate.RecordGate(root, slug, specID, "tier", "none", sessionID)
		return Result{Tier: "none", Results: map[string]string{}, OK: false, Declared: false}
	}

	_, _ = execstate.RecordGate(root, slug, specID, "tier", "full", sessionID)
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
		_, _ = execstate.RecordGate(root, slug, specID, label, result, sessionID)
	}
	return Result{Tier: "full", Results: results, OK: ok, Declared: true}
}

func ChangedFiles(root string) []ChangedFile {
	seen := map[string]bool{}
	files := []ChangedFile{}
	for _, line := range strings.Split(gitOutput(root, "diff", "--name-only", "HEAD"), "\n") {
		p := strings.TrimSpace(line)
		if p != "" && !isMintState(p) && !seen[p] {
			seen[p] = true
			files = append(files, ChangedFile{Path: p, Status: "modified"})
		}
	}
	for _, line := range strings.Split(gitOutput(root, "ls-files", "--others", "--exclude-standard"), "\n") {
		p := strings.TrimSpace(line)
		if p != "" && !isMintState(p) && !seen[p] {
			seen[p] = true
			files = append(files, ChangedFile{Path: p, Status: "new"})
		}
	}
	return files
}

func ResolveGates(root, sessionID, specPath string) map[string]string {
	if specPath != "" {
		if b, err := os.ReadFile(specPath); err == nil {
			if gates := specschema.ResolveSpecGates(string(b)); len(gates) > 0 {
				return gates
			}
		}
	}
	state, ok := session.ReadState(root, sessionID)
	if !ok {
		return map[string]string{}
	}
	return gatesFromState(state)
}

func gatesFromState(state session.State) map[string]string {
	raw, ok := state["gates"]
	if !ok {
		return map[string]string{}
	}
	switch v := raw.(type) {
	case map[string]string:
		return v
	case map[string]any:
		out := map[string]string{}
		for key, value := range v {
			if s, ok := value.(string); ok && strings.TrimSpace(key) != "" && strings.TrimSpace(s) != "" {
				out[key] = s
			}
		}
		return out
	case json.RawMessage:
		var parsed map[string]string
		if err := json.Unmarshal(v, &parsed); err == nil {
			return parsed
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

func isMintState(path string) bool {
	clean := filepath.ToSlash(path)
	return clean == ".mint" || strings.HasPrefix(clean, ".mint/")
}

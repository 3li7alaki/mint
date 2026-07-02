package verify

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"mint/internal/execstate"
	"mint/internal/session"
)

func TestChangedFilesExcludesMintAndIncludesTrackedAndUntracked(t *testing.T) {
	root := newRepo(t)
	writeFile(t, filepath.Join(root, "README.md"), "changed\n")
	writeFile(t, filepath.Join(root, "app.go"), "package main\n")
	writeFile(t, filepath.Join(root, ".mint", "state.json"), "{}")
	files := ChangedFiles(root)
	got := map[string]string{}
	for _, file := range files {
		got[file.Path] = file.Status
	}
	if got["README.md"] != "modified" || got["app.go"] != "new" {
		t.Fatalf("ChangedFiles = %#v", files)
	}
	if _, ok := got[".mint/state.json"]; ok {
		t.Fatalf("mint state was included: %#v", files)
	}
}

func TestRunDocsOnlySkipsGates(t *testing.T) {
	root := newRepo(t)
	if err := session.WriteState(root, "sid-a", session.State{"gates": map[string]string{"tests": "false"}}); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "README.md"), "changed\n")
	result := Run(root, "feat", "001", "sid-a", "")
	if result.Tier != "skip" || !result.OK || !result.Declared {
		t.Fatalf("Run docs-only = %#v", result)
	}
	state, ok := execstate.Read(root, "feat", "001", "sid-a")
	if !ok || state.Gates["tier"] != "skip" {
		t.Fatalf("state = %#v ok=%v", state, ok)
	}
}

func TestRunNoDeclaredGates(t *testing.T) {
	root := newRepo(t)
	writeFile(t, filepath.Join(root, "app.go"), "package main\n")
	result := Run(root, "feat", "002", "sid-a", "")
	if result.Tier != "none" || result.OK || result.Declared {
		t.Fatalf("Run no gates = %#v", result)
	}
	state, ok := execstate.Read(root, "feat", "002", "sid-a")
	if !ok || state.Gates["tier"] != "none" {
		t.Fatalf("state = %#v ok=%v", state, ok)
	}
}

func TestRunSpecGatesOverrideSessionGates(t *testing.T) {
	root := newRepo(t)
	if err := session.WriteState(root, "sid-a", session.State{"gates": map[string]string{"session": "false"}}); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "app.go"), "package main\n")
	spec := filepath.Join(root, "spec.xml")
	writeFile(t, spec, `<task><gates>
  spec: `+successCommand()+`
</gates></task>`)
	result := Run(root, "feat", "003", "sid-a", spec)
	if result.Tier != "full" || !result.OK || result.Results["spec"] != "pass" {
		t.Fatalf("Run spec gates = %#v", result)
	}
	state, _ := execstate.Read(root, "feat", "003", "sid-a")
	if state.Gates["spec"] != "pass" {
		t.Fatalf("state gates = %#v", state.Gates)
	}
	if _, ok := state.Gates["session"]; ok {
		t.Fatalf("session gate should not run when spec overrides: %#v", state.Gates)
	}
}

func TestRunSessionGateFailure(t *testing.T) {
	root := newRepo(t)
	if err := session.WriteState(root, "sid-a", session.State{"gates": map[string]string{"tests": failureCommand()}}); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "app.go"), "package main\n")
	result := Run(root, "feat", "004", "sid-a", "")
	if result.Tier != "full" || result.OK || result.Results["tests"] != "fail" {
		t.Fatalf("Run failing gate = %#v", result)
	}
	state, _ := execstate.Read(root, "feat", "004", "sid-a")
	if state.Gates["tier"] != "full" || state.Gates["tests"] != "fail" {
		t.Fatalf("state gates = %#v", state.Gates)
	}
}

func newRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	gitRun(t, root, "init", "-q")
	gitRun(t, root, "config", "user.email", "t@t.t")
	gitRun(t, root, "config", "user.name", "t")
	writeFile(t, filepath.Join(root, "README.md"), "base\n")
	gitRun(t, root, "add", "-A")
	gitRun(t, root, "commit", "-q", "-m", "base")
	return root
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func gitRun(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func successCommand() string {
	if runtime.GOOS == "windows" {
		return "cmd /C exit 0"
	}
	return "true"
}

func failureCommand() string {
	if runtime.GOOS == "windows" {
		return "cmd /C exit 1"
	}
	return "false"
}

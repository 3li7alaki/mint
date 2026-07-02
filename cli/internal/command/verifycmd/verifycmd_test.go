package verifycmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"mint/internal/session"
)

func TestVerifyCommandRunsDeclaredGate(t *testing.T) {
	root := newRepo(t)
	if err := session.WriteState(root, "sid-a", session.State{"gates": map[string]string{"tests": successCommand()}}); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "app.go"), "package main\n")
	var out bytes.Buffer
	code, err := Run(root, []string{"feat", "001"}, Flags{Session: "sid-a"}, &out)
	if err != nil || code != 0 || !strings.Contains(out.String(), "ok tests") {
		t.Fatalf("code=%d err=%v out=%q", code, err, out.String())
	}
}

func TestVerifyCommandNoGatesIsNonzero(t *testing.T) {
	root := newRepo(t)
	writeFile(t, filepath.Join(root, "app.go"), "package main\n")
	var out bytes.Buffer
	code, err := Run(root, []string{"feat", "001"}, Flags{Session: "sid-a"}, &out)
	if err != nil || code != 1 || !strings.Contains(out.String(), "no gates declared") {
		t.Fatalf("code=%d err=%v out=%q", code, err, out.String())
	}
}

func TestVerifyCommandDocsOnlySkip(t *testing.T) {
	root := newRepo(t)
	writeFile(t, filepath.Join(root, "README.md"), "changed\n")
	var out bytes.Buffer
	code, err := Run(root, []string{"feat", "001"}, Flags{Session: "sid-a"}, &out)
	if err != nil || code != 0 || !strings.Contains(out.String(), "tier: skip") {
		t.Fatalf("code=%d err=%v out=%q", code, err, out.String())
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
	if err := os.MkdirAll(filepath.Join(root, ".mint"), 0o755); err != nil {
		t.Fatal(err)
	}
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

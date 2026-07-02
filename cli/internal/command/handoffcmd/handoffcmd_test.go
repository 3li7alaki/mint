package handoffcmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mint/internal/execstate"
	"mint/internal/session"
)

func TestHandoffWritesSeedFromSessionAndExecState(t *testing.T) {
	root := t.TempDir()
	if err := session.WriteState(root, "sid-a", session.State{"task": "ship", "mode": "full", "autoCommitOverride": false}); err != nil {
		t.Fatal(err)
	}
	specDir := filepath.Join(root, ".mint", "tasks", "sid-a", "feat", "001")
	writeFile(t, filepath.Join(specDir, "001-feat.xml"), `<task><scope><can-modify><path>app.go</path></can-modify></scope></task>`)
	if _, err := execstate.Init(root, "feat", "001", "sid-a", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := execstate.RecordGate(root, "feat", "001", "tests", "pass", "sid-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := execstate.RecordReview(root, "feat", "001", "quality", "passed", "sid-a"); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	code, err := Run(root, []string{"sid-a"}, Flags{Notes: []string{"keep scope tight"}}, &out)
	if err != nil || code != 0 {
		t.Fatalf("Run code=%d err=%v", code, err)
	}
	seedPath := filepath.Join(root, ".mint", "sessions", "sid-a", "handoff.md")
	b, err := os.ReadFile(seedPath)
	if err != nil {
		t.Fatalf("read seed: %v", err)
	}
	seed := string(b)
	for _, want := range []string{"# Handoff - sid-a", "- Task: ship", "- Mode: full", "- Autocommit: OFF", "## feat/001", "app.go", "tests=pass", "quality=passed", "keep scope tight"} {
		if !strings.Contains(seed, want) {
			t.Fatalf("seed missing %q:\n%s", want, seed)
		}
	}
	if !strings.Contains(out.String(), seedPath) {
		t.Fatalf("stdout missing seed path: %q", out.String())
	}
}

func TestHandoffNoSpecs(t *testing.T) {
	root := t.TempDir()
	if err := session.WriteState(root, "sid-a", session.State{}); err != nil {
		t.Fatal(err)
	}
	seed := Build(root, "sid-a", session.State{}, nil, nil)
	if !strings.Contains(seed, "(no spec state recorded yet)") {
		t.Fatalf("seed = %s", seed)
	}
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

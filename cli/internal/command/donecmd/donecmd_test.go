package donecmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"mint/internal/execstate"
	"mint/internal/receipt"
	"mint/internal/snapshot"
	"mint/internal/statehome"
	"mint/internal/unitstore"
)

func TestDoneIssuesAttemptAndSnapshotBoundReceipt(t *testing.T) {
	root := doneRepo(t)
	t.Setenv("MINT_STATE_HOME", t.TempDir())
	if err := unitstore.Ensure(root); err != nil {
		t.Fatal(err)
	}
	spec := unitstore.SpecPath(root, "unit", "001")
	os.MkdirAll(filepath.Dir(spec), 0o700)
	xml := `<task><id>001</id><title>x</title><goal>x</goal><scope><can-modify>app.go</can-modify></scope><acceptance>WHEN x, THE system SHALL y</acceptance><gates>tests: true</gates></task>`
	if err := os.WriteFile(spec, []byte(xml), 0o600); err != nil {
		t.Fatal(err)
	}
	maker := &execstate.Maker{Executor: "codex", Vendor: "openai", Model: "gpt", Locality: "remote", ExecutionRef: "maker"}
	if _, err := execstate.Init(root, "unit", "001", "a1", maker); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "app.go"), []byte("package app\nvar X = 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	verdict := map[string]any{"schemaVersion": 1, "accepted": true, "executor": "opencode", "vendor": "zai", "model": "glm", "locality": "remote", "executionRef": "checker"}
	b, _ := json.Marshal(verdict)
	if err := statehome.Write(unitstore.VerdictPath(root, "unit", "001", "a1"), b); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	code, err := Run(root, []string{"unit", "001"}, Flags{Attempt: "a1", JSON: true}, &out, &bytes.Buffer{})
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v out=%s", code, err, out.String())
	}
	paths := receipt.List(root, "unit", "001")
	if len(paths) != 1 {
		t.Fatalf("receipts=%v", paths)
	}
	record, err := receipt.Read(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	if record.AttemptID != "a1" || !receipt.Validate(root, record).Current {
		t.Fatalf("record=%#v", record)
	}
	os.WriteFile(filepath.Join(root, "app.go"), []byte("package app\nvar X = 3\n"), 0o644)
	if receipt.Validate(root, record).Current {
		t.Fatal("receipt stayed current after source change")
	}
}

func TestDoneRejectsSourceChangeDuringFloorEvaluation(t *testing.T) {
	root := doneRepo(t)
	t.Setenv("MINT_STATE_HOME", t.TempDir())
	if err := unitstore.Ensure(root); err != nil {
		t.Fatal(err)
	}
	spec := unitstore.SpecPath(root, "unit", "001")
	if err := os.MkdirAll(filepath.Dir(spec), 0o700); err != nil {
		t.Fatal(err)
	}
	xml := `<task><id>001</id><title>x</title><goal>x</goal><scope><can-modify>app.go</can-modify></scope><acceptance>WHEN x, THE system SHALL y</acceptance><gates>tests: true</gates></task>`
	if err := os.WriteFile(spec, []byte(xml), 0o600); err != nil {
		t.Fatal(err)
	}
	maker := &execstate.Maker{Executor: "codex", Vendor: "openai", Model: "gpt", Locality: "remote", ExecutionRef: "maker"}
	if _, err := execstate.Init(root, "unit", "001", "a1", maker); err != nil {
		t.Fatal(err)
	}
	verdict := map[string]any{"schemaVersion": 1, "accepted": true, "executor": "opencode", "vendor": "zai", "model": "glm", "locality": "remote", "executionRef": "checker"}
	b, _ := json.Marshal(verdict)
	if err := statehome.Write(unitstore.VerdictPath(root, "unit", "001", "a1"), b); err != nil {
		t.Fatal(err)
	}
	originalCapture := captureSnapshot
	t.Cleanup(func() { captureSnapshot = originalCapture })
	calls := 0
	captureSnapshot = func(snapshotRoot, base string) (snapshot.Source, error) {
		calls++
		if calls == 2 {
			if err := os.WriteFile(filepath.Join(root, "app.go"), []byte("package app\nvar X = 9\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return originalCapture(snapshotRoot, base)
	}
	var out, stderr bytes.Buffer
	code, err := Run(root, []string{"unit", "001"}, Flags{Attempt: "a1"}, &out, &stderr)
	if err != nil || code == 0 || !bytes.Contains(stderr.Bytes(), []byte("source changed")) {
		t.Fatalf("code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if paths := receipt.List(root, "unit", "001"); len(paths) != 0 {
		t.Fatalf("receipt issued for raced source: %v", paths)
	}
}

func doneRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "t@example.com"}, {"config", "user.name", "T"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v %s", args, err, out)
		}
	}
	os.WriteFile(filepath.Join(root, "app.go"), []byte("package app\nvar X = 1\n"), 0o644)
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = root
	cmd.Run()
	cmd = exec.Command("git", "commit", "-qm", "base")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit: %v %s", err, out)
	}
	return root
}

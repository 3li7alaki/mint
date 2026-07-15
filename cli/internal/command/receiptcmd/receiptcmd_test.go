package receiptcmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"mint/internal/floor"
	"mint/internal/receipt"
	"mint/internal/snapshot"
)

func TestVerifyReportsCurrentThenStale(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MINT_STATE_HOME", t.TempDir())
	git(t, root, "init", "-q")
	git(t, root, "config", "user.email", "test@example.com")
	git(t, root, "config", "user.name", "Test")
	write(t, filepath.Join(root, "app.txt"), "one\n")
	git(t, root, "add", "app.txt")
	source, err := snapshot.Capture(root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	record, err := receipt.New(receipt.NewOptions{
		Slug: "unit", SpecID: "001", AttemptID: "a1", Terminal: "done-verified", Snapshot: source,
		Result: floor.Result{Pass: true}, Input: floor.Input{Verdict: map[string]any{"executor": "codex", "vendor": "openai", "model": "gpt", "locality": "remote", "executionRef": "checker"}}, IssuedAt: time.Unix(1, 0),
	})
	if err != nil {
		t.Fatal(err)
	}
	path, err := receipt.Store(root, record)
	if err != nil {
		t.Fatal(err)
	}
	rel, _ := filepath.Rel(root, path)
	var out bytes.Buffer
	if code, err := Run(root, []string{"verify", rel}, Flags{JSON: true}, &out); err != nil || code != 0 {
		t.Fatalf("fresh code=%d err=%v out=%s", code, err, out.String())
	}
	write(t, filepath.Join(root, "app.txt"), "two\n")
	out.Reset()
	if code, err := Run(root, []string{"verify", rel}, Flags{JSON: true}, &out); err != nil || code != 1 {
		t.Fatalf("stale code=%d err=%v out=%s", code, err, out.String())
	}
}

func git(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

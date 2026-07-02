package cleancmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mint/internal/session"
)

func TestCleanReportsNothing(t *testing.T) {
	var out bytes.Buffer
	code, err := Run(t.TempDir(), Flags{}, &out)
	if err != nil || code != 0 {
		t.Fatalf("Run code=%d err=%v", code, err)
	}
	if !strings.Contains(out.String(), "Nothing to clean.") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestCleanRequiresYesForWorktrees(t *testing.T) {
	root := t.TempDir()
	worktreePath := filepath.Join(root, ".mint", "worktrees", "feat")
	if err := os.MkdirAll(worktreePath, 0o755); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	code, err := Run(root, Flags{}, &out)
	if err != nil || code != 0 {
		t.Fatalf("Run code=%d err=%v", code, err)
	}
	if _, err := os.Stat(worktreePath); err != nil {
		t.Fatalf("worktree removed without --yes: %v", err)
	}
	if !strings.Contains(out.String(), "re-run `mint clean --yes`") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestCleanYesRemovesWorktrees(t *testing.T) {
	root := t.TempDir()
	worktreePath := filepath.Join(root, ".mint", "worktrees", "feat")
	if err := os.MkdirAll(worktreePath, 0o755); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	code, err := Run(root, Flags{Yes: true}, &out)
	if err != nil || code != 0 {
		t.Fatalf("Run code=%d err=%v", code, err)
	}
	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Fatalf("worktree still exists or stat failed unexpectedly: %v", err)
	}
	if !strings.Contains(out.String(), "Removed 1 worktree") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestCleanReclaimsOrphanedSessionAndTasks(t *testing.T) {
	root := t.TempDir()
	if err := session.WriteState(root, "orphan", session.State{
		"invokedAt":     time.Now().Format(time.RFC3339),
		"pid":           -1,
		"lastHeartbeat": time.Now().Add(-2 * time.Hour).Format(time.RFC3339Nano),
	}); err != nil {
		t.Fatal(err)
	}
	taskDir := filepath.Join(root, ".mint", "tasks", "orphan")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	code, err := Run(root, Flags{}, &out)
	if err != nil || code != 0 {
		t.Fatalf("Run code=%d err=%v", code, err)
	}
	if _, ok := session.ReadState(root, "orphan"); ok {
		t.Fatal("orphan session still present")
	}
	if _, err := os.Stat(taskDir); !os.IsNotExist(err) {
		t.Fatalf("task dir still exists or stat failed unexpectedly: %v", err)
	}
	if !strings.Contains(out.String(), "Reclaimed 1 orphaned session") {
		t.Fatalf("stdout = %q", out.String())
	}
}

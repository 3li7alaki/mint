package replay

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"mint/internal/snapshot"
)

func TestPrepareRestoresProtectedPathsAndCloseRemovesWorktree(t *testing.T) {
	root := t.TempDir()
	run(t, root, "init", "-q")
	write(t, root, ".gitignore", "ignored.txt\n")
	write(t, root, "scripts/gate.txt", "ORIGINAL\n")
	write(t, root, "scripts/deleted.txt", "restore deletion\n")
	run(t, root, "add", ".")
	run(t, root, "-c", "user.name=Test", "-c", "user.email=t@example.com", "commit", "-qm", "base")
	// The baseline includes dispatcher changes that HEAD does not contain.
	write(t, root, "scripts/gate.txt", "ORIGINAL baseline\n")
	baseline, err := snapshot.CaptureTree(root)
	if err != nil {
		t.Fatal(err)
	}
	write(t, root, "scripts/gate.txt", "WEAKENED\n")
	write(t, root, "scripts/new.txt", "worker addition\n")
	write(t, root, "worker.go", "package worker\n")
	write(t, root, "ignored.txt", "ignored\n")
	if err := os.Remove(filepath.Join(root, "scripts/deleted.txt")); err != nil {
		t.Fatal(err)
	}
	worktree, err := Prepare(root, baseline, []string{"scripts/**", "scripts/gate.txt"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := os.Stat(worktree.Dir); err == nil {
			_ = worktree.Close()
		}
	}()
	if want := []string{"scripts/deleted.txt", "scripts/gate.txt", "scripts/new.txt"}; !reflect.DeepEqual(worktree.Reverted, want) {
		t.Fatalf("Reverted=%v want %v", worktree.Reverted, want)
	}
	for path, want := range map[string]string{"scripts/gate.txt": "ORIGINAL baseline\n", "scripts/deleted.txt": "restore deletion\n", "worker.go": "package worker\n"} {
		body, err := os.ReadFile(filepath.Join(worktree.Dir, path))
		if err != nil || string(body) != want {
			t.Fatalf("%s=%q want %q err=%v", path, body, want, err)
		}
	}
	for _, path := range []string{"scripts/new.txt", "ignored.txt"} {
		if _, err := os.Stat(filepath.Join(worktree.Dir, path)); !os.IsNotExist(err) {
			t.Fatalf("%s exists in replay: %v", path, err)
		}
	}
	if body, err := os.ReadFile(filepath.Join(root, "scripts/gate.txt")); err != nil || string(body) != "WEAKENED\n" {
		t.Fatalf("live protected file=%q err=%v", body, err)
	}
	if err := worktree.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(worktree.Dir); !os.IsNotExist(err) {
		t.Fatalf("replay directory remains: %v", err)
	}
	if got := run(t, root, "worktree", "list", "--porcelain"); strings.Count(got, "worktree ") != 1 {
		t.Fatalf("worktrees remain: %s", got)
	}
}

func TestPrepareNeedsCommit(t *testing.T) {
	root := t.TempDir()
	run(t, root, "init", "-q")
	baseline, err := snapshot.CaptureTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Prepare(root, baseline, nil); err == nil || !strings.Contains(err.Error(), "replay needs a commit") {
		t.Fatalf("Prepare error=%v", err)
	}
}

func run(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func write(t *testing.T, root, path, body string) {
	t.Helper()
	abs := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

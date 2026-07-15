package snapshot

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCaptureStableAcrossCommitAndChangesWithSource(t *testing.T) {
	root := newRepo(t)
	write(t, filepath.Join(root, "app.txt"), "one\n")
	gitRun(t, root, "add", "app.txt")
	before := capture(t, root)
	gitRun(t, root, "commit", "-m", "add app")
	afterCommit := capture(t, root)
	if before.Digest != afterCommit.Digest {
		t.Fatalf("content-identical commit changed digest: %s != %s", before.Digest, afterCommit.Digest)
	}
	write(t, filepath.Join(root, "app.txt"), "two\n")
	afterChange := capture(t, root)
	if before.Digest == afterChange.Digest {
		t.Fatal("source change did not change digest")
	}
}

func TestCaptureDeletionStableAcrossCommit(t *testing.T) {
	root := newRepo(t)
	write(t, filepath.Join(root, "keep.txt"), "keep\n")
	write(t, filepath.Join(root, "remove.txt"), "remove\n")
	gitRun(t, root, "add", "keep.txt", "remove.txt")
	gitRun(t, root, "commit", "-m", "base")
	if err := os.Remove(filepath.Join(root, "remove.txt")); err != nil {
		t.Fatal(err)
	}
	before := capture(t, root)
	gitRun(t, root, "add", "-u")
	gitRun(t, root, "commit", "-m", "remove file")
	after := capture(t, root)
	if before.Digest != after.Digest {
		t.Fatalf("reviewed deletion changed digest on commit: %s != %s", before.Digest, after.Digest)
	}
}

func TestCaptureIncludesAllRelevantUntrackedSource(t *testing.T) {
	root := newRepo(t)
	write(t, filepath.Join(root, "tracked.txt"), "tracked\n")
	gitRun(t, root, "add", "tracked.txt")
	first := capture(t, root)
	write(t, filepath.Join(root, "untracked.txt"), "new\n")
	second := capture(t, root)
	if first.Digest == second.Digest {
		t.Fatal("untracked source was not included")
	}
	write(t, filepath.Join(root, ".mint", "receipts", "local.json"), "evidence\n")
	third := capture(t, root)
	if second.Digest == third.Digest {
		t.Fatal("repository-local .mint content was incorrectly excluded")
	}
}

func TestCaptureWorksFromLinkedWorktree(t *testing.T) {
	root := newRepo(t)
	write(t, filepath.Join(root, "app.txt"), "one\n")
	gitRun(t, root, "add", "app.txt")
	gitRun(t, root, "commit", "-m", "base")
	worktree := filepath.Join(t.TempDir(), "linked")
	gitRun(t, root, "worktree", "add", "-b", "feature", worktree)
	source := capture(t, worktree)
	if source.WorktreeRoot != worktree {
		t.Fatalf("worktreeRoot=%q want %q", source.WorktreeRoot, worktree)
	}
	if source.CommonGitDir == "" || source.CommonGitDir == filepath.Join(worktree, ".git") {
		t.Fatalf("commonGitDir not resolved for linked worktree: %q", source.CommonGitDir)
	}
}

func capture(t *testing.T, root string) Source {
	t.Helper()
	source, err := Capture(root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func newRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	gitRun(t, root, "init", "-q")
	gitRun(t, root, "config", "user.email", "test@example.com")
	gitRun(t, root, "config", "user.name", "Test")
	return root
}

func gitRun(t *testing.T, root string, args ...string) {
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

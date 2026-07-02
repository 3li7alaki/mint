package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateCopiesEnvAndListRemove(t *testing.T) {
	root := newRepo(t)
	writeFile(t, filepath.Join(root, ".env"), "SECRET=1\n")

	created, err := Create(root, "spec-001", CreateOptions{})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Branch != "mint/spec-001" {
		t.Fatalf("branch = %q", created.Branch)
	}
	if _, err := os.Stat(filepath.Join(created.WorktreePath, ".env")); err != nil {
		t.Fatalf("env not copied: %v", err)
	}

	listed := List(root)
	if len(listed) != 1 || listed[0].Slug != "spec-001" || listed[0].Branch != "mint/spec-001" {
		t.Fatalf("List() = %#v", listed)
	}

	Remove(root, "spec-001")
	if _, err := os.Stat(created.WorktreePath); !os.IsNotExist(err) {
		t.Fatalf("worktree still exists/stat err = %v", err)
	}
	if listed := List(root); len(listed) != 0 {
		t.Fatalf("List() after remove = %#v", listed)
	}
}

func TestCreateRemovesStaleWorktree(t *testing.T) {
	root := newRepo(t)
	first, err := Create(root, "spec-002", CreateOptions{})
	if err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	writeFile(t, filepath.Join(first.WorktreePath, "stale.txt"), "stale")
	second, err := Create(root, "spec-002", CreateOptions{})
	if err != nil {
		t.Fatalf("second Create() error = %v", err)
	}
	if first.WorktreePath != second.WorktreePath {
		t.Fatalf("path changed: %q vs %q", first.WorktreePath, second.WorktreePath)
	}
	if _, err := os.Stat(filepath.Join(second.WorktreePath, "stale.txt")); !os.IsNotExist(err) {
		t.Fatalf("stale file survived/stat err = %v", err)
	}
}

func TestCommitChangesNoopAndCommitExcludesMint(t *testing.T) {
	root := newRepo(t)
	created, err := Create(root, "spec-003", CreateOptions{})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if got := CommitChanges(created.WorktreePath, ""); got.Committed || got.Reason != "clean" {
		t.Fatalf("clean CommitChanges() = %#v", got)
	}

	writeFile(t, filepath.Join(created.WorktreePath, "worker-output.txt"), "built")
	writeFile(t, filepath.Join(created.WorktreePath, ".mint", "transient.txt"), "ignore")
	got := CommitChanges(created.WorktreePath, "feat: worker output")
	if !got.Committed {
		t.Fatalf("CommitChanges() = %#v", got)
	}
	tree := gitOut(t, created.WorktreePath, "ls-tree", "-r", "--name-only", "HEAD")
	if !strings.Contains(tree, "worker-output.txt") {
		t.Fatalf("committed tree missing worker output:\n%s", tree)
	}
	if strings.Contains(tree, ".mint/transient.txt") {
		t.Fatalf("committed tree included .mint transient:\n%s", tree)
	}
	log := gitOut(t, created.WorktreePath, "log", "--oneline", "-1")
	if !strings.Contains(log, "feat: worker output") {
		t.Fatalf("commit message not used: %s", log)
	}
}

func TestMergeWorktree(t *testing.T) {
	root := newRepo(t)
	created, err := Create(root, "spec-004", CreateOptions{})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	writeFile(t, filepath.Join(created.WorktreePath, "worker-output.txt"), "built")
	if got := CommitChanges(created.WorktreePath, "feat: worker output"); !got.Committed {
		t.Fatalf("CommitChanges() = %#v", got)
	}
	merge := Merge(root, "spec-004", "HEAD")
	if !merge.Merged || merge.Commits != 1 || merge.Conflicts {
		t.Fatalf("Merge() = %#v", merge)
	}
	tree := gitOut(t, root, "ls-tree", "-r", "--name-only", "HEAD")
	if !strings.Contains(tree, "worker-output.txt") {
		t.Fatalf("merged tree missing worker output:\n%s", tree)
	}
}

func TestCleanAll(t *testing.T) {
	root := newRepo(t)
	if _, err := Create(root, "spec-005", CreateOptions{}); err != nil {
		t.Fatalf("Create spec-005: %v", err)
	}
	if _, err := Create(root, "spec-006", CreateOptions{}); err != nil {
		t.Fatalf("Create spec-006: %v", err)
	}
	if got := CleanAll(root); got != 2 {
		t.Fatalf("CleanAll() = %d, want 2", got)
	}
	if listed := List(root); len(listed) != 0 {
		t.Fatalf("List() after clean = %#v", listed)
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
	t.Cleanup(func() {
		_ = exec.Command("git", "-C", root, "worktree", "prune").Run()
	})
	return root
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func gitRun(t *testing.T, cwd string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func gitOut(t *testing.T, cwd string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

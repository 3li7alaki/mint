package statehome

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestResolveLivesOutsideRepoAndSeparatesLinkedWorktrees(t *testing.T) {
	state := t.TempDir()
	t.Setenv("MINT_STATE_HOME", state)
	root := t.TempDir()
	gitRun(t, root, "init", "-q")
	gitRun(t, root, "config", "user.email", "test@example.com")
	gitRun(t, root, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(root, "file"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, root, "add", "file")
	gitRun(t, root, "commit", "-m", "base")
	linked := filepath.Join(t.TempDir(), "linked")
	gitRun(t, root, "worktree", "add", "-b", "feature", linked)

	a := Resolve(root)
	b := Resolve(linked)
	if a.RepositoryID != b.RepositoryID || a.RepositoryDir != b.RepositoryDir {
		t.Fatalf("worktrees did not share repository identity: a=%#v b=%#v", a, b)
	}
	if a.WorktreeID == b.WorktreeID || a.Dir == b.Dir {
		t.Fatalf("linked worktrees were not isolated: a=%#v b=%#v", a, b)
	}
	if rel, _ := filepath.Rel(root, a.Dir); rel == "." || (len(rel) > 0 && rel[0] != '.') {
		t.Fatalf("state directory unexpectedly inside repository: %q", a.Dir)
	}
	if filepath.Dir(filepath.Dir(a.RepositoryDir)) != state {
		t.Fatalf("state dir=%q want under %q", a.Dir, state)
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

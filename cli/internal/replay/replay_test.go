package replay

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
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
	if got := run(t, root, "worktree", "list", "--porcelain"); !strings.Contains(got, "locked mint pid "+strconv.Itoa(os.Getpid())+"\n") {
		t.Fatalf("replay worktree is not owned: %s", got)
	}
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

func TestPrepareSweepsOnlyDeadMintOwners(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows conservatively treats every owner as alive")
	}
	child := exec.Command("git", "--version")
	if err := child.Run(); err != nil {
		t.Fatal(err)
	}
	dead := strconv.Itoa(child.Process.Pid)
	for _, tc := range []struct {
		name    string
		reason  string
		missing bool
		removed bool
	}{
		{"dead owner", "mint pid " + dead, false, true},
		{"missing directory", "mint pid " + dead, true, true},
		{"live owner", "mint pid " + strconv.Itoa(os.Getpid()), false, false},
		{"foreign lock", "my own", false, false},
		{"unlocked", "", false, false},
		{"zero pid", "mint pid 0", false, false},
		{"negative pid", "mint pid -" + dead, false, false},
		{"signed pid", "mint pid +" + dead, false, false},
		{"extra text", "mint pid " + dead + " extra", false, false},
		{"missing pid", "mint pid ", false, false},
		{"overflow pid", "mint pid 99999999999999999999", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			run(t, root, "init", "-q")
			run(t, root, "-c", "user.name=Test", "-c", "user.email=t@example.com", "commit", "--allow-empty", "-qm", "base")
			baseline := strings.TrimSpace(run(t, root, "rev-parse", "HEAD^{tree}"))
			parent, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			orphan := filepath.Join(parent, "orphan\nquoted\" path")
			args := []string{"worktree", "add", "--detach"}
			if tc.reason != "" {
				args = append(args, "--lock", "--reason", tc.reason)
			}
			run(t, root, append(args, orphan, "HEAD")...)
			t.Cleanup(func() { _ = (Worktree{Dir: orphan, root: root}).Close() })
			if tc.missing {
				if err := os.RemoveAll(orphan); err != nil {
					t.Fatal(err)
				}
			}
			worktree, err := Prepare(root, baseline, nil)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = worktree.Close() })
			got := run(t, root, "worktree", "list", "--porcelain", "-z")
			if remains := strings.Contains(got, "worktree "+orphan+"\x00"); remains == tc.removed {
				t.Fatalf("orphan registered=%v, want %v: %q", remains, !tc.removed, got)
			}
			if _, err := os.Stat(orphan); tc.removed && !os.IsNotExist(err) || !tc.removed && err != nil {
				t.Fatalf("orphan directory: removed=%v err=%v", tc.removed, err)
			}
			if _, err := os.Stat(worktree.Dir); err != nil {
				t.Fatalf("new replay directory: %v", err)
			}
			want := 2
			if !tc.removed {
				want++
			}
			if strings.Count(got, "worktree ") != want {
				t.Fatalf("unexpected worktrees: %q", got)
			}
		})
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

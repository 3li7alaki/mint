package snapshot

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCaptureTreePreservesWorktreeAndIndex(t *testing.T) {
	for _, kind := range []string{"unborn", "committed", "linked"} {
		t.Run(kind, func(t *testing.T) {
			root := newRepo(t)
			write(t, filepath.Join(root, ".gitignore"), "ignored.txt\n")
			write(t, filepath.Join(root, "tracked.txt"), "original\n")
			write(t, filepath.Join(root, "remove.txt"), "remove\n")
			gitRun(t, root, "add", ".")
			if kind != "unborn" {
				gitRun(t, root, "commit", "-qm", "base")
			}
			if kind == "linked" {
				linked := filepath.Join(t.TempDir(), "linked")
				gitRun(t, root, "worktree", "add", "-b", "feature", linked)
				root = linked
			}
			write(t, filepath.Join(root, "tracked.txt"), "staged\n")
			gitRun(t, root, "add", "tracked.txt")
			write(t, filepath.Join(root, "tracked.txt"), "working\n")
			write(t, filepath.Join(root, "nested", "untracked.txt"), "untracked\n")
			write(t, filepath.Join(root, "ignored.txt"), "ignored\n")
			if err := os.Remove(filepath.Join(root, "remove.txt")); err != nil {
				t.Fatal(err)
			}
			statusBefore, err := gitBytes(root, "--no-optional-locks", "status", "--porcelain")
			if err != nil {
				t.Fatal(err)
			}
			indexPath, err := git(root, "rev-parse", "--path-format=absolute", "--git-path", "index")
			if err != nil {
				t.Fatal(err)
			}
			indexBefore, err := os.ReadFile(strings.TrimSpace(indexPath))
			if err != nil {
				t.Fatal(err)
			}
			sourceBefore := capture(t, root)
			tree, err := CaptureTree(filepath.Join(root, "nested"))
			if err != nil {
				t.Fatal(err)
			}
			paths, err := git(root, "ls-tree", "-r", "--name-only", tree)
			if err != nil || paths != ".gitignore\nnested/untracked.txt\ntracked.txt\n" {
				t.Fatalf("tree paths=%q err=%v", paths, err)
			}
			for path, want := range map[string]string{"tracked.txt": "working\n", "nested/untracked.txt": "untracked\n"} {
				body, err := git(root, "show", tree+":"+path)
				if err != nil || body != want {
					t.Fatalf("%s contents=%q want %q err=%v", path, body, want, err)
				}
			}
			statusAfter, err := gitBytes(root, "--no-optional-locks", "status", "--porcelain")
			if err != nil || !bytes.Equal(statusBefore, statusAfter) {
				t.Fatalf("status changed: before=%q after=%q err=%v", statusBefore, statusAfter, err)
			}
			indexAfter, err := os.ReadFile(strings.TrimSpace(indexPath))
			if err != nil || !bytes.Equal(indexBefore, indexAfter) {
				t.Fatalf("real index changed: err=%v", err)
			}
			if sourceAfter := capture(t, root); sourceBefore.Digest != sourceAfter.Digest {
				t.Fatal("capture changed worktree contents")
			}
			if body, err := os.ReadFile(filepath.Join(root, "ignored.txt")); err != nil || string(body) != "ignored\n" {
				t.Fatalf("ignored file changed: body=%q err=%v", body, err)
			}
		})
	}
}

func TestCaptureTreeEmptyUnbornRepo(t *testing.T) {
	root := newRepo(t)
	tree, err := CaptureTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if paths, err := git(root, "ls-tree", "-r", "--name-only", tree); err != nil || paths != "" {
		t.Fatalf("empty tree paths=%q err=%v", paths, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "index")); !os.IsNotExist(err) {
		t.Fatalf("capture created real index: err=%v", err)
	}
}

func TestChangedSinceReturnsBaselineDiffAndCurrentTree(t *testing.T) {
	root := newRepo(t)
	write(t, filepath.Join(root, "tracked.txt"), "HEAD\n")
	write(t, filepath.Join(root, ".gitignore"), "ignored.txt\n")
	gitRun(t, root, "add", ".")
	gitRun(t, root, "commit", "-qm", "base")
	write(t, filepath.Join(root, "tracked.txt"), "dispatcher\n")
	baseline, err := CaptureTree(root)
	if err != nil {
		t.Fatal(err)
	}
	path := "new file\nwith newline.txt"
	write(t, filepath.Join(root, path), "worker\n")
	write(t, filepath.Join(root, "ignored.txt"), "ignored\n")
	files, patch, current, err := ChangedSince(root, baseline)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(files, []string{path}) || !strings.Contains(patch, "+worker") || strings.Contains(patch, "dispatcher") {
		t.Fatalf("files=%q patch=%q", files, patch)
	}
	if body, err := git(root, "show", current+":"+path); err != nil || body != "worker\n" {
		t.Fatalf("current tree file=%q err=%v", body, err)
	}
	if _, _, _, err := ChangedSince(root, "missing-tree"); err == nil {
		t.Fatal("missing baseline accepted")
	}
}

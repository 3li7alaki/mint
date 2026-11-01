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

// The sibling-lanes checker's case: a kept path that starts with a set-aside file's name and
// " b/", and glob characters, each mean only themselves.
func TestPatchForLimitsToLiteralFiles(t *testing.T) {
	root := newRepo(t)
	write(t, filepath.Join(root, "base.txt"), "base\n")
	gitRun(t, root, "add", ".")
	gitRun(t, root, "commit", "-qm", "base")
	baseline, err := CaptureTree(root)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, "sibling.go"), "ASIDE\n")
	write(t, filepath.Join(root, "sibling.go b", "tests_test.go"), "OWN-TEST\n")
	write(t, filepath.Join(root, "[a].go"), "BRACKET\n")
	write(t, filepath.Join(root, "a.go"), "PLAIN\n")
	_, _, current, err := ChangedSince(root, baseline)
	if err != nil {
		t.Fatal(err)
	}
	patch, err := PatchFor(root, baseline, current, []string{"sibling.go b/tests_test.go", "[a].go"})
	if err != nil {
		t.Fatal(err)
	}
	for want, in := range map[string]bool{"OWN-TEST": true, "BRACKET": true, "ASIDE": false, "PLAIN": false} {
		if strings.Contains(patch, want) != in {
			t.Fatalf("%s in patch=%v want %v:\n%s", want, !in, in, patch)
		}
	}
	if patch, err := PatchFor(root, baseline, current, nil); err != nil || patch != "" {
		t.Fatalf("no files: patch=%q err=%v", patch, err)
	}
}

// A pathspec selects a directory's contents too: a kept deletion of foo.go must not pull in
// a set-aside foo.go/child.txt, and a kept file replacing a directory still lands.
func TestPatchForFileDirectorySwaps(t *testing.T) {
	root := newRepo(t)
	write(t, filepath.Join(root, "foo.go"), "OLD-FOO\n")
	write(t, filepath.Join(root, "dir", "inner.txt"), "OLD-INNER\n")
	gitRun(t, root, "add", ".")
	gitRun(t, root, "commit", "-qm", "base")
	baseline, err := CaptureTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "foo.go")); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, "foo.go", "child.txt"), "ASIDE-CHILD\n")
	if err := os.RemoveAll(filepath.Join(root, "dir")); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, "dir"), "NEW-DIR-FILE\n")
	_, _, current, err := ChangedSince(root, baseline)
	if err != nil {
		t.Fatal(err)
	}
	patch, err := PatchFor(root, baseline, current, []string{"foo.go", "dir", "dir/inner.txt"})
	if err != nil {
		t.Fatal(err)
	}
	for want, in := range map[string]bool{"-OLD-FOO": true, "+NEW-DIR-FILE": true, "-OLD-INNER": true, "ASIDE-CHILD": false} {
		if strings.Contains(patch, want) != in {
			t.Fatalf("%s in patch=%v want %v:\n%s", want, !in, in, patch)
		}
	}
}

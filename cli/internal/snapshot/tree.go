package snapshot

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CaptureTree records tracked and non-ignored untracked files without changing
// the worktree or its index.
// ponytail: the tree is unreferenced, so gc may prune it after gc.pruneExpire
// (2 weeks by default) and a longer attempt fails closed at done; pin it under a
// ref if attempts ever outlive that.
func CaptureTree(root string) (string, error) {
	worktreeRoot, err := git(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("Resolve worktree root: %w", err)
	}
	indexDir, err := os.MkdirTemp("", "mint-index-*")
	if err != nil {
		return "", fmt.Errorf("Create temporary Git index: %w", err)
	}
	defer os.RemoveAll(indexDir)
	run := func(args ...string) (string, error) {
		cmd := exec.Command("git", args...)
		cmd.Dir = strings.TrimSpace(worktreeRoot)
		cmd.Env = append(os.Environ(), "GIT_INDEX_FILE="+filepath.Join(indexDir, "index"))
		out, err := cmd.Output()
		return string(out), err
	}
	readTree := "HEAD"
	if _, err := run("rev-parse", "--verify", "--quiet", "HEAD"); err != nil {
		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
			return "", fmt.Errorf("Resolve HEAD: %w", err)
		}
		readTree = "--empty"
	}
	if _, err := run("read-tree", readTree); err != nil {
		return "", fmt.Errorf("Read Git tree: %w", err)
	}
	if _, err := run("add", "-A"); err != nil {
		return "", fmt.Errorf("Add worktree files: %w", err)
	}
	tree, err := run("write-tree")
	if err != nil {
		return "", fmt.Errorf("Write Git tree: %w", err)
	}
	return strings.TrimSpace(tree), nil
}

// ChangedSince compares the current worktree contents with a captured baseline.
func ChangedSince(root, baseline string) (files []string, patch string, current string, err error) {
	current, err = CaptureTree(root)
	if err != nil {
		return nil, "", "", fmt.Errorf("Capture current tree: %w", err)
	}
	paths, err := git(root, "diff-tree", "-r", "--name-only", "-z", baseline, current)
	if err != nil {
		return nil, "", current, fmt.Errorf("Baseline tree diff failed: %w", err)
	}
	for _, path := range strings.Split(paths, "\x00") {
		if path != "" {
			files = append(files, path)
		}
	}
	patch, err = git(root, "diff", baseline, current)
	if err != nil {
		return nil, "", current, fmt.Errorf("Baseline tree patch failed: %w", err)
	}
	return files, patch, current, nil
}

// PatchFor is the baseline-to-current patch of exactly files. It builds a tree that is the
// baseline with only those paths taken from current, then diffs the two with no pathspec.
// Both shortcuts leaked: cutting the full patch apart misreads a header once a path holds
// " b/", and a pathspec, even :(literal), also selects everything below a directory of the
// same name, so a kept deleted file foo.go dragged in a set-aside foo.go/child.txt (mint
// sibling-lanes checker, 2026-11-01). files must hold both sides of any file/directory swap
// (floor.SplitSiblingFiles keeps them together): staging a file over a baseline directory
// drops that directory's entries, which then show in the patch as deletions.
func PatchFor(root, baseline, current string, files []string) (string, error) {
	if len(files) == 0 {
		return "", nil
	}
	indexDir, err := os.MkdirTemp("", "mint-index-*")
	if err != nil {
		return "", fmt.Errorf("Create temporary Git index: %w", err)
	}
	defer os.RemoveAll(indexDir)
	run := func(stdin string, args ...string) (string, error) {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "GIT_INDEX_FILE="+filepath.Join(indexDir, "index"))
		cmd.Stdin = strings.NewReader(stdin)
		out, err := cmd.Output()
		return string(out), err
	}
	if _, err := run("", "read-tree", baseline); err != nil {
		return "", fmt.Errorf("Read baseline tree: %w", err)
	}
	// One --index-info line per path: its exact entry in current, or mode 0 to drop it when
	// current holds no file there (deleted, or a directory now).
	var info strings.Builder
	for _, file := range files {
		entry, err := run("", "ls-tree", "-z", "--full-tree", current, "--", ":(literal)"+file)
		if err != nil {
			return "", fmt.Errorf("Read current entry for %s: %w", file, err)
		}
		meta, name, _ := strings.Cut(strings.TrimSuffix(entry, "\x00"), "\t")
		fields := strings.Fields(meta)
		if name == file && len(fields) == 3 && fields[1] != "tree" {
			fmt.Fprintf(&info, "%s %s\t%s\x00", fields[0], fields[2], file)
		} else {
			fmt.Fprintf(&info, "0 %s\t%s\x00", strings.Repeat("0", len(baseline)), file)
		}
	}
	if _, err := run(info.String(), "update-index", "-z", "--replace", "--index-info"); err != nil {
		return "", fmt.Errorf("Stage kept files: %w", err)
	}
	kept, err := run("", "write-tree")
	if err != nil {
		return "", fmt.Errorf("Write kept tree: %w", err)
	}
	patch, err := git(root, "diff", baseline, strings.TrimSpace(kept))
	if err != nil {
		return "", fmt.Errorf("Baseline tree patch failed: %w", err)
	}
	return patch, nil
}

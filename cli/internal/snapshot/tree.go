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

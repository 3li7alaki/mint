// Package replay prepares disposable worktrees for gate execution.
package replay

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"mint/internal/scopematch"
	"mint/internal/snapshot"
)

type Worktree struct {
	Dir      string
	Reverted []string
	root     string
}

func Prepare(root, baseline string, protected []string) (Worktree, error) {
	if _, err := git(root, "", "rev-parse", "--verify", "HEAD^{commit}"); err != nil {
		return Worktree{}, fmt.Errorf("replay needs a commit: %w", err)
	}
	files, _, current, err := snapshot.ChangedSince(root, baseline)
	if err != nil {
		return Worktree{}, err
	}
	indexDir, err := os.MkdirTemp("", "mint-replay-index-*")
	if err != nil {
		return Worktree{}, err
	}
	defer os.RemoveAll(indexDir)
	index := filepath.Join(indexDir, "index")
	if _, err := git(root, index, "read-tree", current); err != nil {
		return Worktree{}, err
	}
	var reverted []string
	for _, path := range files {
		for _, pattern := range protected {
			if !scopematch.MatchesAllowed(path, pattern) {
				continue
			}
			entry, err := git(root, index, "--literal-pathspecs", "ls-tree", "-z", baseline, "--", path)
			if err != nil {
				return Worktree{}, err
			}
			if entry == "" {
				_, err = git(root, index, "update-index", "--force-remove", "--", path)
			} else {
				metadata, _, _ := strings.Cut(entry, "\t")
				fields := strings.Fields(metadata)
				_, err = git(root, index, "update-index", "--add", "--cacheinfo", fields[0]+","+fields[2]+","+path)
			}
			if err != nil {
				return Worktree{}, err
			}
			reverted = append(reverted, path)
			break
		}
	}
	graded, err := git(root, index, "write-tree")
	if err != nil {
		return Worktree{}, err
	}
	dir, err := os.MkdirTemp("", "mint-replay-*")
	if err != nil {
		return Worktree{}, err
	}
	worktree := Worktree{Dir: dir, Reverted: reverted, root: root}
	if _, err := git(root, "", "worktree", "add", "--detach", "--no-checkout", dir, "HEAD"); err != nil {
		return Worktree{}, errors.Join(err, os.RemoveAll(dir))
	}
	if _, err := git(dir, "", "read-tree", "-u", "--reset", strings.TrimSpace(graded)); err != nil {
		return Worktree{}, errors.Join(err, worktree.Close())
	}
	return worktree, nil
}

func (w Worktree) Close() error {
	_, err := git(w.root, "", "worktree", "remove", "--force", w.Dir)
	return errors.Join(err, os.RemoveAll(w.Dir))
}

func git(root, index string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	if index != "" {
		cmd.Env = append(os.Environ(), "GIT_INDEX_FILE="+index)
	}
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", err
	}
	return string(out), nil
}

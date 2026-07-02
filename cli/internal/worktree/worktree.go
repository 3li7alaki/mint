package worktree

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type CreateOptions struct {
	BaseBranch string
}

type Created struct {
	WorktreePath string
	Branch       string
}

type Listed struct {
	Slug   string
	Path   string
	Branch string
}

type CommitResult struct {
	Committed bool
	Reason    string
}

type MergeResult struct {
	Merged    bool
	Commits   int
	Conflicts bool
}

func Create(projectRoot, slug string, options CreateOptions) (Created, error) {
	worktreeDir := filepath.Join(projectRoot, ".mint", "worktrees")
	worktreePath := filepath.Join(worktreeDir, slug)
	branch := "mint/" + slug
	if err := os.MkdirAll(worktreeDir, 0o755); err != nil {
		return Created{}, err
	}
	if _, err := os.Stat(worktreePath); err == nil {
		if err := git(projectRoot, "worktree", "remove", worktreePath, "--force"); err != nil {
			_ = os.RemoveAll(worktreePath)
			_ = git(projectRoot, "worktree", "prune")
		}
	}
	_ = git(projectRoot, "branch", "-D", branch)

	base := options.BaseBranch
	if base == "" {
		base = "HEAD"
	}
	if err := git(projectRoot, "worktree", "add", worktreePath, "-b", branch, base); err != nil {
		return Created{}, err
	}
	copyEnvFiles(projectRoot, worktreePath)
	return Created{WorktreePath: worktreePath, Branch: branch}, nil
}

func Remove(projectRoot, slug string) {
	worktreePath := filepath.Join(projectRoot, ".mint", "worktrees", slug)
	branch := "mint/" + slug
	if err := git(projectRoot, "worktree", "remove", worktreePath, "--force"); err != nil {
		if _, statErr := os.Stat(worktreePath); statErr == nil {
			_ = os.RemoveAll(worktreePath)
		}
		_ = git(projectRoot, "worktree", "prune")
	}
	_ = git(projectRoot, "branch", "-D", branch)
}

func List(projectRoot string) []Listed {
	worktreeDir := filepath.Join(projectRoot, ".mint", "worktrees")
	entries, err := os.ReadDir(worktreeDir)
	if err != nil {
		return nil
	}
	out := make([]Listed, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		slug := entry.Name()
		out = append(out, Listed{
			Slug:   slug,
			Path:   filepath.Join(worktreeDir, slug),
			Branch: "mint/" + slug,
		})
	}
	return out
}

func CleanAll(projectRoot string) int {
	worktrees := List(projectRoot)
	count := 0
	for _, wt := range worktrees {
		Remove(projectRoot, wt.Slug)
		count++
	}
	_ = git(projectRoot, "worktree", "prune")
	return count
}

func CommitChanges(worktreePath, message string) CommitResult {
	dirty, err := gitOutput(worktreePath, "status", "--porcelain", "--", ".", ":!.mint")
	if err != nil {
		return CommitResult{Committed: false, Reason: firstLine(err.Error())}
	}
	if strings.TrimSpace(dirty) == "" {
		return CommitResult{Committed: false, Reason: "clean"}
	}
	if err := git(worktreePath, "add", "-A", "--", ".", ":!.mint"); err != nil {
		return CommitResult{Committed: false, Reason: firstLine(err.Error())}
	}
	if message == "" {
		message = "mint: worker output"
	}
	if err := gitWithInput(worktreePath, message, "commit", "-F", "-"); err != nil {
		return CommitResult{Committed: false, Reason: firstLine(err.Error())}
	}
	return CommitResult{Committed: true}
}

func Merge(projectRoot, slug, baseBranch string) MergeResult {
	branch := "mint/" + slug
	log, err := gitOutput(projectRoot, "log", baseBranch+".."+branch, "--oneline")
	if err != nil {
		return MergeResult{Merged: false, Commits: 0, Conflicts: false}
	}
	log = strings.TrimSpace(log)
	if log == "" {
		return MergeResult{Merged: true, Commits: 0, Conflicts: false}
	}
	commitCount := len(strings.Split(log, "\n"))
	if err := git(projectRoot, "merge", branch, "--no-ff", "-m", "merge: "+slug); err != nil {
		_ = git(projectRoot, "merge", "--abort")
		return MergeResult{Merged: false, Commits: commitCount, Conflicts: true}
	}
	return MergeResult{Merged: true, Commits: commitCount, Conflicts: false}
}

func copyEnvFiles(projectRoot, worktreePath string) {
	for _, name := range []string{".env", ".env.local", ".env.development", ".env.development.local", ".env.test"} {
		src := filepath.Join(projectRoot, name)
		dest := filepath.Join(worktreePath, name)
		content, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		_ = os.WriteFile(dest, content, 0o644)
	}
}

func git(cwd string, args ...string) error {
	_, err := gitCombined(cwd, nil, args...)
	return err
}

func gitOutput(cwd string, args ...string) (string, error) {
	out, err := gitCombined(cwd, nil, args...)
	return string(out), err
}

func gitWithInput(cwd, input string, args ...string) error {
	_, err := gitCombined(cwd, []byte(input), args...)
	return err
}

func gitCombined(cwd string, input []byte, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

func firstLine(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if i := strings.IndexByte(value, '\n'); i >= 0 {
		return value[:i]
	}
	return value
}

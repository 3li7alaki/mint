package cleancmd

import (
	"fmt"
	"io"
	"time"

	"mint/internal/session"
	"mint/internal/worktree"
)

type Flags struct {
	Yes bool
}

func Run(root string, flags Flags, stdout io.Writer) (int, error) {
	fmt.Fprintln(stdout, "mint clean")

	sessionsCleaned := session.CleanStale(root, 24*time.Hour, time.Now())
	if sessionsCleaned > 0 {
		fmt.Fprintf(stdout, "Removed %d stale session%s\n", sessionsCleaned, plural(sessionsCleaned))
	}

	orphansReclaimed := session.ReclaimOrphaned(root, time.Hour, time.Now())
	if orphansReclaimed > 0 {
		fmt.Fprintf(stdout, "Reclaimed %d orphaned session%s\n", orphansReclaimed, plural(orphansReclaimed))
	}

	worktrees := worktree.List(root)
	if len(worktrees) == 0 {
		if sessionsCleaned == 0 && orphansReclaimed == 0 {
			fmt.Fprintln(stdout, "Nothing to clean.")
		}
		fmt.Fprintln(stdout, "Clean")
		return 0, nil
	}

	fmt.Fprintf(stdout, "Found %d worktree%s:\n", len(worktrees), plural(len(worktrees)))
	for _, wt := range worktrees {
		fmt.Fprintf(stdout, "  %s -> %s\n", wt.Slug, wt.Path)
	}
	if !flags.Yes {
		fmt.Fprintf(stdout, "%d worktree%s found - re-run `mint clean --yes` to remove.\n", len(worktrees), plural(len(worktrees)))
		fmt.Fprintln(stdout, "Clean")
		return 0, nil
	}

	count := worktree.CleanAll(root)
	fmt.Fprintf(stdout, "Removed %d worktree%s\n", count, plural(count))
	fmt.Fprintln(stdout, "Clean")
	return 0, nil
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

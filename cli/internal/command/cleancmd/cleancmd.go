package cleancmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"mint/internal/receipt"
	"mint/internal/statehome"
)

type Flags struct{ Yes bool }

func Run(root string, flags Flags, stdout io.Writer) (int, error) {
	dir := statehome.Resolve(root).Dir
	var locks []string
	_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && strings.HasSuffix(entry.Name(), ".lock") {
			locks = append(locks, path)
		}
		return nil
	})
	claims := receipt.OrphanClaims(root)
	if len(locks) == 0 && len(claims) == 0 {
		fmt.Fprintln(stdout, "nothing to clean")
		return 0, nil
	}
	fmt.Fprintf(stdout, "%d orphanable lock file(s) and %d incomplete completion claim(s) in this worktree state\n", len(locks), len(claims))
	if !flags.Yes {
		fmt.Fprintln(stdout, "re-run with --yes after confirming no mint process is writing this worktree")
		return 0, nil
	}
	for _, path := range append(locks, claims...) {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return 1, err
		}
	}
	fmt.Fprintf(stdout, "removed %d orphanable state file(s)\n", len(locks)+len(claims))
	return 0, nil
}

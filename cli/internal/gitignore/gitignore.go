package gitignore

import (
	"os"
	"path/filepath"
	"strings"
)

const marker = "# mint local state"

var MintIgnoreEntries = []string{
	".mint/tasks/",
	".mint/worktrees/",
	".mint/sessions/",
	".mint/verdicts/",
	"scratchpad/",
}

type Result struct {
	Added          []string
	AlreadyPresent []string
}

func Ensure(projectRoot string, entries []string) (Result, error) {
	if entries == nil {
		entries = MintIgnoreEntries
	}

	gitignorePath := filepath.Join(projectRoot, ".gitignore")
	b, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return Result{}, err
	}
	content := string(b)

	var result Result
	for _, entry := range entries {
		if strings.Contains(content, entry) {
			result.AlreadyPresent = append(result.AlreadyPresent, entry)
		} else {
			result.Added = append(result.Added, entry)
		}
	}

	if len(result.Added) == 0 {
		return result, nil
	}

	var lines []string
	if !strings.Contains(content, marker) {
		lines = append(lines, "", marker)
	}
	lines = append(lines, result.Added...)
	block := strings.Join(lines, "\n") + "\n"

	f, err := os.OpenFile(gitignorePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return Result{}, err
	}
	defer f.Close()
	if _, err := f.WriteString(block); err != nil {
		return Result{}, err
	}
	return result, nil
}

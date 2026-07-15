package docclassify

import (
	"path/filepath"
	"strings"

	"mint/internal/scopematch"
)

var DocPatterns = []string{
	"*.md", "*.txt", "*.mdx", "LICENSE*", "CHANGELOG*",
	".gitignore", ".gitattributes",
	"*.png", "*.jpg", "*.jpeg", "*.gif", "*.svg", "*.ico",
	"*.woff", "*.woff2", "*.ttf", "*.eot",
	"docs/**",
}

func IsDocPath(path string) bool {
	for _, pattern := range DocPatterns {
		if !strings.Contains(pattern, "/") {
			if scopematch.MatchesAllowed(filepath.Base(path), pattern) {
				return true
			}
			continue
		}
		if scopematch.MatchesAllowed(path, pattern) {
			return true
		}
	}
	return false
}

func IsDocsOnly(changedFiles []string) bool {
	if len(changedFiles) == 0 {
		return false
	}
	for _, file := range changedFiles {
		if !IsDocPath(file) {
			return false
		}
	}
	return true
}

// Package snapshot fingerprints the source contents visible from the current
// Git worktree. It understands linked worktrees but never creates, switches,
// merges, or removes them.
package snapshot

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const Version = 1

type Source struct {
	Version      int    `json:"version"`
	Digest       string `json:"digest"`
	RepositoryID string `json:"repositoryId"`
	WorktreeRoot string `json:"worktreeRoot"`
	CommonGitDir string `json:"commonGitDir"`
	Head         string `json:"head,omitempty"`
	Base         string `json:"base"`
	FileCount    int    `json:"fileCount"`
}

// Capture hashes tracked and relevant untracked contents, including deletions,
// file kinds, and executable bits. Git-ignored files are excluded. mint has no
// repository-local state to special-case.
func Capture(root, base string) (Source, error) {
	worktreeRoot, err := git(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return Source{}, fmt.Errorf("resolve worktree root: %w", err)
	}
	worktreeRoot = strings.TrimSpace(worktreeRoot)
	commonGitDir, err := git(worktreeRoot, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return Source{}, fmt.Errorf("resolve common git dir: %w", err)
	}
	commonGitDir = strings.TrimSpace(commonGitDir)
	canonicalGitDir, err := filepath.EvalSymlinks(commonGitDir)
	if err == nil {
		commonGitDir = canonicalGitDir
	}
	repositoryID := repositoryIdentity(commonGitDir)
	if base == "" {
		base = "HEAD"
	}

	paths, err := sourcePaths(worktreeRoot)
	if err != nil {
		return Source{}, err
	}
	h := sha256.New()
	writeField(h, "mint-source-snapshot-v1")
	writeField(h, repositoryID)
	for _, path := range paths {
		if err := hashPath(h, worktreeRoot, path); err != nil {
			return Source{}, err
		}
	}

	head, _ := git(worktreeRoot, "rev-parse", "--verify", "HEAD")
	return Source{
		Version:      Version,
		Digest:       "sha256:" + hex.EncodeToString(h.Sum(nil)),
		RepositoryID: repositoryID,
		WorktreeRoot: worktreeRoot,
		CommonGitDir: commonGitDir,
		Head:         strings.TrimSpace(head),
		Base:         base,
		FileCount:    len(paths),
	}, nil
}

func sourcePaths(root string) ([]string, error) {
	out, err := gitBytes(root, "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	if err != nil {
		return nil, fmt.Errorf("list source files: %w", err)
	}
	seen := map[string]bool{}
	var paths []string
	for _, raw := range bytes.Split(out, []byte{0}) {
		path := string(raw)
		if path == "" || excluded(path) || seen[path] {
			continue
		}
		// A deletion is absence from the current source tree, not an index
		// tombstone. This keeps the digest stable when a reviewed deletion is
		// committed without changing source contents.
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(path))); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("stat source path %s: %w", path, err)
		}
		seen[path] = true
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func repositoryIdentity(commonGitDir string) string {
	sum := sha256.Sum256([]byte(filepath.Clean(commonGitDir)))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func excluded(path string) bool {
	clean := filepath.ToSlash(filepath.Clean(path))
	return clean == ".git" || strings.HasPrefix(clean, ".git/")
}

func hashPath(w io.Writer, root, path string) error {
	writeField(w, filepath.ToSlash(path))
	abs := filepath.Join(root, filepath.FromSlash(path))
	info, err := os.Lstat(abs)
	if os.IsNotExist(err) {
		writeField(w, "deleted")
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat snapshot path %s: %w", path, err)
	}

	mode := info.Mode()
	switch {
	case mode.IsRegular():
		writeField(w, fmt.Sprintf("file:%o", mode.Perm()&0o111))
		f, err := os.Open(abs)
		if err != nil {
			return fmt.Errorf("open snapshot path %s: %w", path, err)
		}
		defer f.Close()
		if err := writeSizedReader(w, f, info.Size()); err != nil {
			return fmt.Errorf("hash snapshot path %s: %w", path, err)
		}
	case mode&os.ModeSymlink != 0:
		target, err := os.Readlink(abs)
		if err != nil {
			return fmt.Errorf("read snapshot symlink %s: %w", path, err)
		}
		writeField(w, "symlink")
		writeField(w, target)
	default:
		return fmt.Errorf("unsupported source file kind for %s (%s)", path, mode.String())
	}
	return nil
}

func writeField(w io.Writer, value string) {
	_ = binary.Write(w, binary.BigEndian, uint64(len(value)))
	_, _ = io.WriteString(w, value)
}

func writeSizedReader(w io.Writer, r io.Reader, size int64) error {
	if err := binary.Write(w, binary.BigEndian, uint64(size)); err != nil {
		return err
	}
	_, err := io.Copy(w, bufio.NewReader(r))
	return err
}

func git(root string, args ...string) (string, error) {
	out, err := gitBytes(root, args...)
	return string(out), err
}

func gitBytes(root string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return out, nil
}

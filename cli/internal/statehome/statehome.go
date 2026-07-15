// Package statehome locates mint's private operational state outside the
// repository. Linked worktrees share a repository identity while keeping
// units, attempts, receipts, and notes isolated by worktree identity.
package statehome

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Location struct {
	RepositoryID  string `json:"repositoryId"`
	WorktreeID    string `json:"worktreeId"`
	WorktreeRoot  string `json:"worktreeRoot"`
	CommonGitDir  string `json:"commonGitDir"`
	GitDir        string `json:"gitDir"`
	RepositoryDir string `json:"repositoryDir"`
	Dir           string `json:"dir"`
}

func Resolve(root string) Location {
	worktreeRoot := canonical(git(root, "rev-parse", "--show-toplevel"))
	if worktreeRoot == "" {
		worktreeRoot = canonical(root)
	}
	commonGitDir := canonical(git(worktreeRoot, "rev-parse", "--path-format=absolute", "--git-common-dir"))
	gitDir := canonical(git(worktreeRoot, "rev-parse", "--path-format=absolute", "--git-dir"))
	if commonGitDir == "" {
		commonGitDir = worktreeRoot
	}
	if gitDir == "" {
		gitDir = commonGitDir
	}
	repositoryID := digest(commonGitDir)
	worktreeID := digest(worktreeRoot + "\x00" + gitDir)
	repositoryDir := filepath.Join(BaseDir(), "repos", repositoryID)
	return Location{
		RepositoryID:  "sha256:" + repositoryID,
		WorktreeID:    "sha256:" + worktreeID,
		WorktreeRoot:  worktreeRoot,
		CommonGitDir:  commonGitDir,
		GitDir:        gitDir,
		RepositoryDir: repositoryDir,
		Dir:           filepath.Join(repositoryDir, "worktrees", worktreeID),
	}
}

func Ensure(root string) (Location, error) {
	loc := Resolve(root)
	if err := os.MkdirAll(loc.Dir, 0o700); err != nil {
		return Location{}, err
	}
	for _, dir := range []string{BaseDir(), filepath.Join(BaseDir(), "repos"), loc.RepositoryDir, filepath.Join(loc.RepositoryDir, "worktrees"), loc.Dir} {
		if err := os.Chmod(dir, 0o700); err != nil {
			return Location{}, err
		}
	}
	return loc, nil
}

func WriteJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return Write(path, append(b, '\n'))
}

func Write(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.Chmod(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".write-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func BaseDir() string {
	if explicit := strings.TrimSpace(os.Getenv("MINT_STATE_HOME")); explicit != "" {
		if abs, err := filepath.Abs(explicit); err == nil {
			return abs
		}
	}
	if xdg := strings.TrimSpace(os.Getenv("XDG_STATE_HOME")); xdg != "" && filepath.IsAbs(xdg) {
		return filepath.Join(xdg, "mint")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "state", "mint")
	}
	return filepath.Join(os.TempDir(), "mint-state")
}

func canonical(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	if evaluated, err := filepath.EvalSymlinks(abs); err == nil {
		return filepath.Clean(evaluated)
	}
	return filepath.Clean(abs)
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func git(root string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

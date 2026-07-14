package atomic

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// IsLiteralSegment reports whether s is a single, in-place path segment — not
// empty, ".", "..", nor containing a separator. Callers that join a
// caller-controlled identifier (slug, spec-id, session-id) into a filesystem
// path use this to fail closed rather than let a crafted value traverse out of
// the intended directory. It lives here because both execstate and session
// build paths from untrusted identifiers and neither imports the other.
func IsLiteralSegment(s string) bool {
	return s != "" && s != "." && s != ".." && !strings.ContainsRune(s, '/') && !strings.ContainsRune(s, filepath.Separator)
}

func Write(path string, content []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	suffix, err := randomHex(4)
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, "."+filepath.Base(path)+"."+suffix+".tmp")
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func WriteString(path, content string) error {
	return Write(path, []byte(content))
}

func WriteJSON(path string, data any) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return Write(path, b)
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

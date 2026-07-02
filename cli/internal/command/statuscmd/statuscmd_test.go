package statuscmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mint/internal/session"
)

func TestStatusNoSessions(t *testing.T) {
	var out bytes.Buffer
	code, err := Run(t.TempDir(), &out)
	if err != nil || code != 0 {
		t.Fatalf("Run code=%d err=%v", code, err)
	}
	// Assert against the Version var (not a literal) so this survives ldflags injection.
	if !strings.Contains(out.String(), "mint v"+Version) || !strings.Contains(out.String(), "none active") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestStatusShowsSessionsAndWorktrees(t *testing.T) {
	root := t.TempDir()
	if err := session.WriteState(root, "1234567890abcdef", session.State{"task": "ship", "mode": "full"}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".mint", "worktrees", "feat"), 0o755); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	code, err := Run(root, &out)
	if err != nil || code != 0 {
		t.Fatalf("Run code=%d err=%v", code, err)
	}
	text := out.String()
	for _, want := range []string{"Sessions", "1234567890ab...", "full - ship", "Worktrees:  1 active"} {
		if !strings.Contains(text, want) {
			t.Fatalf("stdout missing %q:\n%s", want, text)
		}
	}
}

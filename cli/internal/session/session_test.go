package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSessionPathsAndCapturedID(t *testing.T) {
	root := t.TempDir()
	if got := GetSessionsDir(root); got != filepath.Join(root, ".mint", "sessions") {
		t.Fatalf("GetSessionsDir = %q", got)
	}
	if got := Path(root, "sess-a"); got != filepath.Join(root, ".mint", "sessions", "sess-a.json") {
		t.Fatalf("Path = %q", got)
	}
	current := filepath.Join(root, ".mint", "sessions", ".current-session-id")
	if err := os.MkdirAll(filepath.Dir(current), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(current, []byte(" captured \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ReadCapturedID(root); got != "captured" {
		t.Fatalf("ReadCapturedID = %q", got)
	}
}

func TestWriteReadListDeleteState(t *testing.T) {
	root := t.TempDir()
	if err := WriteState(root, "sess-a", State{"reviews": []string{"security"}, "invokedAt": "2026-06-30T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	state, ok := ReadState(root, "sess-a")
	if !ok {
		t.Fatal("ReadState failed")
	}
	if _, ok := state["pid"].(float64); !ok {
		t.Fatalf("pid was not stamped: %#v", state)
	}
	if state["lastHeartbeat"] == "" {
		t.Fatalf("lastHeartbeat was not stamped: %#v", state)
	}
	reviews, ok := state["reviews"].([]any)
	if !ok || reviews[0] != "security" {
		t.Fatalf("reviews = %#v", state["reviews"])
	}
	if content, err := os.ReadFile(filepath.Join(root, ".gitignore")); err != nil || !strings.Contains(string(content), ".mint/tasks/") {
		t.Fatalf("gitignore not ensured: content=%q err=%v", content, err)
	}
	listed := List(root)
	if len(listed) != 1 || listed[0].ID != "sess-a" {
		t.Fatalf("List = %#v", listed)
	}
	if !DeleteState(root, "sess-a") {
		t.Fatal("DeleteState should delete existing state")
	}
	if _, ok := ReadState(root, "sess-a"); ok {
		t.Fatal("deleted session still readable")
	}
}

func TestCleanStale(t *testing.T) {
	root := t.TempDir()
	old := "2026-06-28T00:00:00Z"
	fresh := "2026-06-30T00:00:00Z"
	if err := WriteState(root, "old", State{"invokedAt": old}); err != nil {
		t.Fatal(err)
	}
	if err := WriteState(root, "fresh", State{"invokedAt": fresh}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 6, 30, 1, 0, 0, 0, time.UTC)
	if got := CleanStale(root, 24*time.Hour, now); got != 1 {
		t.Fatalf("CleanStale = %d, want 1", got)
	}
	if _, ok := ReadState(root, "old"); ok {
		t.Fatal("old session should be cleaned")
	}
	if _, ok := ReadState(root, "fresh"); !ok {
		t.Fatal("fresh session should remain")
	}
}

func TestGenerateID(t *testing.T) {
	id, err := GenerateID(time.UnixMilli(0x195e3a1b2c0))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(id, "0195e3a1b2c0-") || len(id) != len("0195e3a1b2c0-a1b2c3d4") {
		t.Fatalf("GenerateID = %q", id)
	}
}

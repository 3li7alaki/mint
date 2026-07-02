package gitignore

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestEnsure(t *testing.T) {
	t.Run("creates .gitignore with one entry per line", func(t *testing.T) {
		dir := t.TempDir()
		if _, err := Ensure(dir, nil); err != nil {
			t.Fatal(err)
		}
		content := read(t, filepath.Join(dir, ".gitignore"))
		lines := strings.Split(content, "\n")
		for _, entry := range MintIgnoreEntries {
			if !contains(lines, entry) {
				t.Fatalf("missing entry %q in %q", entry, content)
			}
		}
		for _, line := range lines {
			matches := 0
			for _, entry := range MintIgnoreEntries {
				if strings.Contains(line, entry) {
					matches++
				}
			}
			if matches > 1 {
				t.Fatalf("line contains multiple entries: %q", line)
			}
		}
		if !strings.Contains(content, marker) {
			t.Fatalf("missing marker in %q", content)
		}
	})

	t.Run("appends and does not duplicate", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".gitignore")
		if err := os.WriteFile(path, []byte("node_modules/\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		first, err := Ensure(dir, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(first.Added, MintIgnoreEntries) || len(first.AlreadyPresent) != 0 {
			t.Fatalf("first result = %#v", first)
		}
		second, err := Ensure(dir, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(second.Added) != 0 || !reflect.DeepEqual(second.AlreadyPresent, MintIgnoreEntries) {
			t.Fatalf("second result = %#v", second)
		}
		content := read(t, path)
		if strings.Count(content, ".mint/tasks/") != 1 {
			t.Fatalf("duplicated .mint/tasks/: %q", content)
		}
		if !strings.Contains(content, "node_modules/") {
			t.Fatalf("lost existing content: %q", content)
		}
	})

	t.Run("adds only missing entries and supports custom entries", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".gitignore")
		if err := os.WriteFile(path, []byte("# mint local state\n.mint/tasks/\n.mint/sessions/\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		result, err := Ensure(dir, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(result.AlreadyPresent, ".mint/tasks/") || !contains(result.AlreadyPresent, ".mint/sessions/") {
			t.Fatalf("alreadyPresent = %#v", result.AlreadyPresent)
		}
		if contains(result.Added, ".mint/tasks/") || contains(result.Added, ".mint/sessions/") {
			t.Fatalf("added already-present entries: %#v", result.Added)
		}

		custom := t.TempDir()
		if _, err := Ensure(custom, []string{".env", ".env.local"}); err != nil {
			t.Fatal(err)
		}
		content := read(t, filepath.Join(custom, ".gitignore"))
		if !strings.Contains(content, ".env") || !strings.Contains(content, ".env.local") {
			t.Fatalf("missing custom entries: %q", content)
		}
	})
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func contains[T comparable](items []T, want T) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

package atomic

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWrite(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.txt")

	if err := WriteString(file, "hello world"); err != nil {
		t.Fatal(err)
	}
	assertFile(t, file, "hello world")

	if err := WriteString(file, "second"); err != nil {
		t.Fatal(err)
	}
	assertFile(t, file, "second")

	nested := filepath.Join(dir, "nested", "dir", "test.txt")
	if err := WriteString(nested, "deep"); err != nil {
		t.Fatal(err)
	}
	assertFile(t, nested, "deep")

	clean := filepath.Join(dir, "clean.txt")
	if err := WriteString(clean, "data"); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Type().IsRegular() && entry.Name() != "test.txt" && entry.Name() != "clean.txt" {
			t.Fatalf("unexpected temp file left behind: %s", entry.Name())
		}
	}
}

func TestWriteJSON(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "data.json")
	data := map[string]any{"key": "value", "nested": map[string]any{"a": float64(1)}}
	if err := WriteJSON(file, data); err != nil {
		t.Fatal(err)
	}
	assertFile(t, file, "{\n  \"key\": \"value\",\n  \"nested\": {\n    \"a\": 1\n  }\n}\n")

	var parsed map[string]any
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(parsed, data) {
		t.Fatalf("parsed JSON = %#v, want %#v", parsed, data)
	}
}

func assertFile(t *testing.T, path, want string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != want {
		t.Fatalf("%s = %q, want %q", path, string(b), want)
	}
}

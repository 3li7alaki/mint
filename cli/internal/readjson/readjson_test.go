package readjson

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadSafe(t *testing.T) {
	dir := t.TempDir()

	valid := filepath.Join(dir, "valid.json")
	if err := os.WriteFile(valid, []byte(`{"a":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got := ReadSafe(valid)
	m, ok := got.(map[string]any)
	if !ok || m["a"] != float64(1) {
		t.Fatalf("ReadSafe(valid) = %#v", got)
	}

	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ReadSafe(bad); got != nil {
		t.Fatalf("ReadSafe(invalid) = %#v, want nil", got)
	}
	if got := ReadSafe(filepath.Join(dir, "missing.json")); got != nil {
		t.Fatalf("ReadSafe(missing) = %#v, want nil", got)
	}
}

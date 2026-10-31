package unitstore

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPathsRejectTraversalAndListUnits(t *testing.T) {
	t.Setenv("MINT_STATE_HOME", t.TempDir())
	root := t.TempDir()
	for _, bad := range []string{"", ".", "..", "a/b"} {
		if got := UnitDir(root, bad, "001"); got != "" {
			t.Fatalf("hostile slug %q resolved %q", bad, got)
		}
		if got := AttemptPath(root, "unit", "001", bad); got != "" {
			t.Fatalf("hostile attempt %q resolved %q", bad, got)
		}
	}
	path := SpecPath(root, "unit", "001")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("<task/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	units := List(root)
	if len(units) != 1 || units[0].Slug != "unit" || units[0].SpecID != "001" {
		t.Fatalf("units=%#v", units)
	}
}

func TestAttemptIDsAreLiteralAndUnique(t *testing.T) {
	t.Setenv("MINT_STATE_HOME", t.TempDir())
	now := time.Unix(1, 0)
	a, err := GenerateAttemptID(now)
	if err != nil {
		t.Fatal(err)
	}
	b, err := GenerateAttemptID(now)
	if err != nil {
		t.Fatal(err)
	}
	if a == b || AttemptPath(t.TempDir(), "unit", "001", a) == "" {
		t.Fatalf("attempt ids a=%q b=%q", a, b)
	}
}

func TestGateLogDir(t *testing.T) {
	t.Setenv("MINT_STATE_HOME", t.TempDir())
	root := t.TempDir()
	want := filepath.Join(filepath.Dir(AttemptPath(root, "unit", "001", "a1")), "a1.logs")
	if got := GateLogDir(root, "unit", "001", "a1"); got != want || !filepath.IsAbs(got) {
		t.Fatalf("log dir=%q, want %q", got, want)
	}
	for _, tc := range []struct{ slug, specID, attemptID string }{
		{"../unit", "001", "a1"},
		{"unit", "..", "a1"},
		{"unit", "001", "a/b"},
		{"unit", "001", ""},
	} {
		if got := GateLogDir(root, tc.slug, tc.specID, tc.attemptID); got != "" {
			t.Fatalf("invalid path components %#v resolved %q", tc, got)
		}
	}
}

func TestVerifyLockPath(t *testing.T) {
	t.Setenv("MINT_STATE_HOME", t.TempDir())
	root := t.TempDir()
	want := filepath.Join(filepath.Dir(AttemptPath(root, "unit", "001", "a1")), "a1.verify.lock")
	if got := VerifyLockPath(root, "unit", "001", "a1"); got != want || !filepath.IsAbs(got) {
		t.Fatalf("verify lock=%q, want %q", got, want)
	}
	for _, tc := range []struct{ slug, specID, attemptID string }{
		{"../unit", "001", "a1"},
		{"unit", "..", "a1"},
		{"unit", "001", "a/b"},
		{"unit", "001", ""},
	} {
		if got := VerifyLockPath(root, tc.slug, tc.specID, tc.attemptID); got != "" {
			t.Fatalf("invalid path components %#v resolved %q", tc, got)
		}
	}
}

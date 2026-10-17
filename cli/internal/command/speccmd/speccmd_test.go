package speccmd

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"mint/internal/specschema"
)

func TestNewUsesGlobalStateAndCarriesOpaqueParent(t *testing.T) {
	root := t.TempDir()
	state := t.TempDir()
	t.Setenv("MINT_STATE_HOME", state)
	t.Setenv("MINT_PARENT_SYSTEM", "driver-x")
	t.Setenv("MINT_PARENT_ID", "card-7")
	result, err := New(root, "Atomic unit", Flags{Goal: "G", Scope: "cli/**", Acceptance: "WHEN x, THE system SHALL y", Gates: map[string]string{"tests": "go test ./..."}, Reviews: []string{"quality"}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(result.SpecPath, root) || !strings.HasPrefix(result.SpecPath, state) {
		t.Fatalf("path=%s", result.SpecPath)
	}
	if _, err := os.Stat(root + "/.gitignore"); !os.IsNotExist(err) {
		t.Fatalf("gitignore was created: %v", err)
	}
	b, err := os.ReadFile(result.SpecPath)
	if err != nil {
		t.Fatal(err)
	}
	xml := string(b)
	parent, ok := specschema.ResolveParent(xml)
	if !ok || parent.System != "driver-x" || parent.ID != "card-7" {
		t.Fatalf("parent=%#v", parent)
	}
	if specschema.ResolveSpecGates(xml)["tests"] == "" {
		t.Fatal("gate missing")
	}
	var out bytes.Buffer
	if code, err := Run(root, []string{"validate", result.SpecPath}, Flags{}, &out, &bytes.Buffer{}); err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
}

func TestNewProtectedPaths(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MINT_STATE_HOME", t.TempDir())
	var out bytes.Buffer
	flags := Flags{Goal: "G", Scope: "cli/**", Protected: "a,b", Acceptance: "WHEN x, THE system SHALL y"}
	if code, err := Run(root, []string{"new", "Protected unit"}, flags, &out, &bytes.Buffer{}); err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	var result NewResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(result.SpecPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "<protected>a,b</protected>") {
		t.Fatalf("protected block missing: %s", b)
	}
	want := append([]string{"a", "b"}, specschema.DefaultProtected...)
	if got := specschema.ResolveProtected(string(b)); !reflect.DeepEqual(got, want) {
		t.Fatalf("protected=%#v, want %#v", got, want)
	}
}

func TestSetProtectedPaths(t *testing.T) {
	for _, initial := range []string{"", "old/path"} {
		t.Run("initial="+initial, func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("MINT_STATE_HOME", t.TempDir())
			result, err := New(root, "Protected unit", Flags{Goal: "G", Scope: "cli/**", Protected: initial, Acceptance: "WHEN x, THE system SHALL y"})
			if err != nil {
				t.Fatal(err)
			}
			if code, err := Run(root, []string{"set", result.SpecPath}, Flags{Protected: "c, d"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil || code != 0 {
				t.Fatalf("code=%d err=%v", code, err)
			}
			b, err := os.ReadFile(result.SpecPath)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Count(string(b), "<protected>") != 1 || !strings.Contains(string(b), "<protected>c,d</protected>") {
				t.Fatalf("protected block not replaced: %s", b)
			}
			want := append([]string{"c", "d"}, specschema.DefaultProtected...)
			if got := specschema.ResolveProtected(string(b)); !reflect.DeepEqual(got, want) {
				t.Fatalf("protected=%#v, want %#v", got, want)
			}
		})
	}
}

func TestPrepareNewAndSet(t *testing.T) {
	for _, initial := range []string{"", `cp "$MINT_SOURCE_ROOT/.env" . && echo '$HOME' > ready`} {
		t.Run("initial="+initial, func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("MINT_STATE_HOME", t.TempDir())
			result, err := New(root, "Prepare unit", Flags{Goal: "G", Scope: "cli/**", Prepare: initial, Acceptance: "WHEN x, THE system SHALL y"})
			if err != nil {
				t.Fatal(err)
			}
			b, err := os.ReadFile(result.SpecPath)
			if err != nil || specschema.ResolvePrepare(string(b)) != initial {
				t.Fatalf("new prepare=%q err=%v", specschema.ResolvePrepare(string(b)), err)
			}
			command := `test -f "$MINT_SOURCE_ROOT/.env" && echo '<ready>' > ready`
			if code, err := Run(root, []string{"set", result.SpecPath}, Flags{Prepare: command}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil || code != 0 {
				t.Fatalf("code=%d err=%v", code, err)
			}
			b, err = os.ReadFile(result.SpecPath)
			if err != nil || strings.Count(string(b), "<prepare>") != 1 || specschema.ResolvePrepare(string(b)) != command {
				t.Fatalf("set prepare=%q err=%v", specschema.ResolvePrepare(string(b)), err)
			}
		})
	}
}

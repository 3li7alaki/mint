package speccmd

import (
	"bytes"
	"os"
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

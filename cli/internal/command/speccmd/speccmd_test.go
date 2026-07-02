package speccmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mint/internal/session"
	"mint/internal/specschema"
)

func TestSpecNewCreatesSpecAndBakesSessionState(t *testing.T) {
	root := t.TempDir()
	if err := session.WriteState(root, "sid-a", session.State{
		"gates":   map[string]string{"tests": "go test ./...", "vet": "go vet ./..."},
		"reviews": []string{"security", "quality"},
	}); err != nil {
		t.Fatalf("WriteState: %v", err)
	}
	var out bytes.Buffer
	code, err := Run(root, []string{"new", "Add billing checks"}, Flags{Session: "sid-a"}, &out, &bytes.Buffer{})
	if err != nil || code != 0 {
		t.Fatalf("Run new code=%d err=%v", code, err)
	}
	var result NewResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("parse json: %v\n%s", err, out.String())
	}
	if result.ID != "001" || result.Branch != "feat/add-billing-checks" {
		t.Fatalf("result = %#v", result)
	}
	b, err := os.ReadFile(result.SpecPath)
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	xml := string(b)
	if !strings.Contains(xml, "<title>Add billing checks</title>") {
		t.Fatalf("title not scaffolded:\n%s", xml)
	}
	if gates := specschema.ResolveSpecGates(xml); gates["tests"] != "go test ./..." || gates["vet"] != "go vet ./..." {
		t.Fatalf("gates = %#v", gates)
	}
	if reviews := specschema.ResolveSpecReviews(xml); strings.Join(reviews, ",") != "security,quality" {
		t.Fatalf("reviews = %#v", reviews)
	}
	if content, err := os.ReadFile(filepath.Join(root, ".gitignore")); err != nil || !strings.Contains(string(content), ".mint/tasks/") {
		t.Fatalf("gitignore not ensured: %q %v", content, err)
	}
}

func TestSpecNewAllocatesNextIDAndSlugOverride(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".mint", "tasks", "sid-a", "custom")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "001-custom.xml"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := New(root, "!!!", Flags{Session: "sid-a", Slug: "custom"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if result.ID != "002" || !strings.HasSuffix(result.SpecPath, filepath.Join(".mint", "tasks", "sid-a", "custom", "002-custom.xml")) {
		t.Fatalf("result = %#v", result)
	}
}

func TestSpecValidateAndScope(t *testing.T) {
	root := t.TempDir()
	spec := `<task>
  <id>001</id><title>T</title><goal>G</goal>
  <scope><can-modify><path>src/a.go</path><path>README.md</path></can-modify></scope>
  <steps>1. Do it</steps>
  <acceptance>- WHEN x happens, THE system SHALL y</acceptance>
  <commit>feat: x</commit>
</task>`
	path := filepath.Join(root, "spec.xml")
	if err := os.WriteFile(path, []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	code, err := Run(root, []string{"validate", "spec.xml"}, Flags{}, &out, &stderr)
	if err != nil || code != 0 || !strings.Contains(out.String(), "is valid") {
		t.Fatalf("validate code=%d err=%v out=%q stderr=%q", code, err, out.String(), stderr.String())
	}
	out.Reset()
	code, err = Run(root, []string{"scope", "spec.xml"}, Flags{}, &out, &stderr)
	if err != nil || code != 0 || out.String() != "src/a.go\nREADME.md\n" {
		t.Fatalf("scope code=%d err=%v out=%q", code, err, out.String())
	}
}

func TestSpecValidateWarningsAreNonzero(t *testing.T) {
	root := t.TempDir()
	spec := `<task>
  <id>001</id><title>T</title><goal>G</goal>
  <scope><can-modify>src</can-modify></scope>
  <steps>1. Do it</steps>
  <acceptance>- WHEN x happens, THE system SHALL y</acceptance>
  <risk>medium</risk>
  <commit>feat: x</commit>
</task>`
	path := filepath.Join(root, "spec.xml")
	if err := os.WriteFile(path, []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code, err := Run(root, []string{"validate", "spec.xml"}, Flags{}, &bytes.Buffer{}, &stderr)
	if err != nil || code != 1 || !strings.Contains(stderr.String(), "unrecognized <risk>") {
		t.Fatalf("validate warning code=%d err=%v stderr=%q", code, err, stderr.String())
	}
}

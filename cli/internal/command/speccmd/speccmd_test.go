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

func TestSpecNewFieldFlags(t *testing.T) {
	tests := []struct {
		name  string
		flags Flags
		check func(t *testing.T, xml string)
	}{
		{
			name: "complete fields replace placeholders",
			flags: Flags{
				Session:    "sid-a",
				Goal:       "G",
				Scope:      "a.go,b.go",
				Acceptance: "A",
			},
			check: func(t *testing.T, xml string) {
				t.Helper()
				if got := elementText(t, xml, "goal"); got != "G" {
					t.Fatalf("goal = %q", got)
				}
				if got := specschema.ResolveCanModify(xml); strings.Join(got, ",") != "a.go,b.go" {
					t.Fatalf("can-modify = %#v", got)
				}
				if got := strings.TrimSpace(elementText(t, xml, "acceptance")); got != "A" {
					t.Fatalf("acceptance = %q", got)
				}
				for _, placeholder := range []string{
					"One sentence - what this unit achieves when done",
					"exact/paths/to/files.js, other/file.js",
					"- WHEN the change runs, THE result SHALL satisfy the goal",
				} {
					if strings.Contains(xml, placeholder) {
						t.Fatalf("placeholder %q still present:\n%s", placeholder, xml)
					}
				}
			},
		},
		{
			name:  "omitted fields keep placeholders",
			flags: Flags{Session: "sid-a", Goal: "G"},
			check: func(t *testing.T, xml string) {
				t.Helper()
				if got := elementText(t, xml, "goal"); got != "G" {
					t.Fatalf("goal = %q", got)
				}
				for _, placeholder := range []string{
					"exact/paths/to/files.js, other/file.js",
					"1. Concrete step with exact file/function reference",
					"- WHEN the change runs, THE result SHALL satisfy the goal",
					"type(scope): short description",
				} {
					if !strings.Contains(xml, placeholder) {
						t.Fatalf("placeholder %q missing:\n%s", placeholder, xml)
					}
				}
			},
		},
		{
			name:  "scope entries are trimmed",
			flags: Flags{Session: "sid-a", Scope: "a.go, b.go"},
			check: func(t *testing.T, xml string) {
				t.Helper()
				if got := specschema.ResolveCanModify(xml); strings.Join(got, ",") != "a.go,b.go" {
					t.Fatalf("can-modify = %#v", got)
				}
				if strings.Contains(elementText(t, xml, "can-modify"), " b.go") {
					t.Fatalf("scope entry was not trimmed:\n%s", xml)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			var out bytes.Buffer
			code, err := Run(root, []string{"new", "T"}, tt.flags, &out, &bytes.Buffer{})
			if err != nil || code != 0 {
				t.Fatalf("Run new code=%d err=%v", code, err)
			}
			var result NewResult
			if err := json.Unmarshal(out.Bytes(), &result); err != nil {
				t.Fatalf("parse json: %v\n%s", err, out.String())
			}
			b, err := os.ReadFile(result.SpecPath)
			if err != nil {
				t.Fatalf("read spec: %v", err)
			}
			tt.check(t, string(b))
		})
	}
}

func TestSpecSetFieldFlags(t *testing.T) {
	original := `<task>
  <id>001</id>
  <title>T</title>
  <goal>OLD</goal>
  <scope>
    <can-modify>a.go, b.go</can-modify>
    <cannot-modify>everything else</cannot-modify>
  </scope>
  <steps>1. Do it</steps>
  <acceptance>- WHEN x happens, THE system SHALL y</acceptance>
  <commit>feat: x</commit>
</task>`
	tests := []struct {
		name      string
		flags     Flags
		wantCode  int
		wantErr   string
		checkFile func(t *testing.T, before, after string)
	}{
		{
			name:     "goal only preserves other bytes",
			flags:    Flags{Goal: "NEW"},
			wantCode: 0,
			checkFile: func(t *testing.T, before, after string) {
				t.Helper()
				if got := elementText(t, after, "goal"); got != "NEW" {
					t.Fatalf("goal = %q", got)
				}
				if strings.Replace(after, "<goal>NEW</goal>", "<goal>OLD</goal>", 1) != before {
					t.Fatalf("non-goal bytes changed\nbefore:\n%s\nafter:\n%s", before, after)
				}
			},
		},
		{
			name:     "invalid update reports validation",
			flags:    Flags{Scope: ","},
			wantCode: 1,
			wantErr:  "<can-modify> resolves to no paths",
			checkFile: func(t *testing.T, before, after string) {
				t.Helper()
				if before == after {
					t.Fatalf("spec was not updated")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "spec.xml")
			if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
				t.Fatal(err)
			}
			var out, stderr bytes.Buffer
			code, err := Run(root, []string{"set", "spec.xml"}, tt.flags, &out, &stderr)
			if code != tt.wantCode {
				t.Fatalf("code = %d, want %d err=%v out=%q stderr=%q", code, tt.wantCode, err, out.String(), stderr.String())
			}
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("Run set error = %v", err)
			}
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read spec: %v", err)
			}
			tt.checkFile(t, original, string(b))
		})
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

func elementText(t *testing.T, xml, name string) string {
	t.Helper()
	open := "<" + name + ">"
	close := "</" + name + ">"
	start := strings.Index(xml, open)
	if start == -1 {
		t.Fatalf("missing %s in:\n%s", open, xml)
	}
	start += len(open)
	end := strings.Index(xml[start:], close)
	if end == -1 {
		t.Fatalf("missing %s in:\n%s", close, xml)
	}
	return xml[start : start+end]
}

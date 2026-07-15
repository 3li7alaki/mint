package verify

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"mint/internal/execstate"
	"mint/internal/unitstore"
)

func TestRunUsesOnlyUnitDeclaredGates(t *testing.T) {
	root := repo(t)
	t.Setenv("MINT_STATE_HOME", t.TempDir())
	if err := unitstore.Ensure(root); err != nil {
		t.Fatal(err)
	}
	spec := unitstore.SpecPath(root, "unit", "001")
	if err := os.MkdirAll(filepath.Dir(spec), 0o700); err != nil {
		t.Fatal(err)
	}
	xml := `<task><id>001</id><title>x</title><goal>x</goal><scope><can-modify>app.go</can-modify></scope><acceptance>WHEN x, THE system SHALL y</acceptance><gates>tests: true</gates></task>`
	if err := os.WriteFile(spec, []byte(xml), 0o600); err != nil {
		t.Fatal(err)
	}
	maker := &execstate.Maker{Executor: "codex", Vendor: "openai", Model: "gpt", Locality: "remote", ExecutionRef: "maker"}
	if _, err := execstate.Init(root, "unit", "001", "a1", maker); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "app.go"), []byte("package app\nvar X = 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := Run(root, "unit", "001", "a1", spec)
	if !result.OK || result.Results["tests"] != "pass" {
		t.Fatalf("result=%#v", result)
	}
}

func repo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	run(t, root, "init", "-q")
	run(t, root, "config", "user.email", "t@example.com")
	run(t, root, "config", "user.name", "T")
	os.WriteFile(filepath.Join(root, "app.go"), []byte("package app\nvar X = 1\n"), 0o644)
	run(t, root, "add", ".")
	run(t, root, "commit", "-qm", "base")
	return root
}
func run(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
}

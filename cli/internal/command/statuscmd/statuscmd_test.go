package statuscmd

import (
	"os"
	"path/filepath"
	"testing"

	"mint/internal/execstate"
	"mint/internal/statehome"
	"mint/internal/unitstore"
)

func TestBuildReportsVersionedWorktreeLedgerAndMissingEvidence(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MINT_STATE_HOME", t.TempDir())
	if err := unitstore.Ensure(root); err != nil {
		t.Fatal(err)
	}
	spec := unitstore.SpecPath(root, "unit", "001")
	if err := os.MkdirAll(filepath.Dir(spec), 0o700); err != nil {
		t.Fatal(err)
	}
	xml := `<task><id>001</id><title>x</title><goal>x</goal><scope><can-modify>**</can-modify></scope><acceptance>WHEN x, THE system SHALL y</acceptance><gates>tests: true</gates><reviews>quality</reviews></task>`
	if err := os.WriteFile(spec, []byte(xml), 0o600); err != nil {
		t.Fatal(err)
	}
	maker := &execstate.Maker{Executor: "codex", Vendor: "openai", Model: "gpt", Locality: "remote", ExecutionRef: "maker"}
	if _, err := execstate.Init(root, "unit", "001", "a1", maker); err != nil {
		t.Fatal(err)
	}
	report := Build(root)
	loc := statehome.Resolve(root)
	if report.SchemaVersion != 1 || report.RepositoryID != loc.RepositoryID || report.WorktreeID != loc.WorktreeID || len(report.Units) != 1 {
		t.Fatalf("report=%#v", report)
	}
	missing := report.Units[0].Attempts[0].Missing
	if len(missing) != 3 {
		t.Fatalf("missing=%v", missing)
	}
}

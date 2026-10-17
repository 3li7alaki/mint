package floor

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"mint/internal/execstate"
	"mint/internal/statehome"
	"mint/internal/unitstore"
)

func TestBuildInputBaselineAndLegacyDiffs(t *testing.T) {
	cases := []struct {
		name      string
		base      string
		legacy    bool
		worker    string
		wantFiles []string
		wantOld   bool
		wantScope bool
	}{
		{"baseline", "", false, "src/new.go", []string{"src/new.go"}, false, true},
		{"out of scope", "", false, "outside.go", []string{"outside.go"}, false, false},
		{"explicit HEAD", "HEAD", false, "src/new.go", []string{"tracked.go", "README.md", "src/new.go"}, true, false},
		{"explicit ref", "HEAD~0", false, "src/new.go", []string{"tracked.go", "README.md", "src/new.go"}, true, false},
		{"legacy state", "", true, "src/new.go", []string{"tracked.go", "README.md", "src/new.go"}, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("MINT_STATE_HOME", t.TempDir())
			inputGit(t, root, "init", "-q")
			inputWrite(t, filepath.Join(root, "tracked.go"), "package app\nvar X = 1\n")
			inputGit(t, root, "add", ".")
			inputGit(t, root, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-qm", "base")
			inputWrite(t, filepath.Join(root, "tracked.go"), "package app\nvar X = 42\n")
			inputWrite(t, filepath.Join(root, "README.md"), "dispatcher notes\n")
			spec := unitstore.SpecPath(root, "unit", "001")
			inputWrite(t, spec, `<task><scope><can-modify>src/**</can-modify></scope><protected>`+tc.worker+`</protected><gates>tests: true</gates></task>`)
			maker := &execstate.Maker{Executor: "codex", Vendor: "openai", Model: "gpt", Locality: "remote", ExecutionRef: "maker"}
			if tc.legacy {
				state := &execstate.State{SchemaVersion: execstate.SchemaVersion, AttemptID: "a1", Status: "running", Maker: maker}
				if err := statehome.WriteJSON(execstate.Path(root, "unit", "001", "a1"), state); err != nil {
					t.Fatal(err)
				}
			} else if _, err := execstate.Init(root, "unit", "001", "a1", maker); err != nil {
				t.Fatal(err)
			}
			inputWrite(t, filepath.Join(root, tc.worker), "package worker\n")
			input, err := BuildInput(root, BuildOptions{SpecPath: spec, Slug: "unit", SpecID: "001", AttemptID: "a1", Base: tc.base})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(input.ChangedFiles, tc.wantFiles) {
				t.Fatalf("ChangedFiles=%v want %v", input.ChangedFiles, tc.wantFiles)
			}
			wantMode := "replay"
			wantReverted := []string{tc.worker}
			if tc.legacy {
				wantMode, wantReverted = "in-place", nil
			}
			if input.Gates.GateMode != wantMode || !reflect.DeepEqual(input.Gates.Reverted, wantReverted) {
				t.Fatalf("gate evidence=%#v", input.Gates)
			}
			if strings.Contains(input.Patch, "tracked.go") != tc.wantOld || strings.Contains(input.Patch, "var X = 42") != tc.wantOld {
				t.Fatalf("pre-existing modification in patch=%q want %v", input.Patch, tc.wantOld)
			}
			if strings.Contains(input.Patch, "dispatcher notes") {
				t.Fatalf("pre-existing untracked file in patch: %s", input.Patch)
			}
			if strings.Contains(input.Patch, "package worker") == tc.wantOld {
				t.Fatalf("worker addition in patch=%q want %v", input.Patch, !tc.wantOld)
			}
			if c5 := clauseResult(t, Enforce(input), 5); c5.Pass != tc.wantScope {
				t.Fatalf("scope pass=%v want %v: %s", c5.Pass, tc.wantScope, reason(c5))
			}
		})
	}
}

func inputGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func inputWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

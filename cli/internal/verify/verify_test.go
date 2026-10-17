package verify

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

func TestRunReplayAndLegacy(t *testing.T) {
	cases := []struct {
		name     string
		prepare  string
		gate     string
		legacy   bool
		wantOK   bool
		reverted []string
	}{
		{"protected baseline", "", `sh -c 'grep -q ORIGINAL scripts/gate.txt'`, false, true, []string{"scripts/gate.txt", "scripts/new.txt"}},
		{"new protected absent", "", `test ! -e scripts/new.txt`, false, true, []string{"scripts/gate.txt", "scripts/new.txt"}},
		{"ignored absent", "", `test ! -e local.env`, false, true, []string{"scripts/gate.txt", "scripts/new.txt"}},
		{"prepare copies ignored", `cp "$MINT_SOURCE_ROOT/local.env" .`, `grep -q LOCAL local.env`, false, true, []string{"scripts/gate.txt", "scripts/new.txt"}},
		{"prepare fails", `false`, `printf ran > "$MINT_GATE_MARKER"`, false, false, []string{"scripts/gate.txt", "scripts/new.txt"}},
		{"gate fails", "", `false`, false, false, []string{"scripts/gate.txt", "scripts/new.txt"}},
		{"legacy in place", `false`, `grep -q WEAKENED scripts/gate.txt && test -f scripts/new.txt && test -f local.env`, true, true, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := repo(t)
			t.Setenv("MINT_STATE_HOME", t.TempDir())
			pathFile := filepath.Join(t.TempDir(), "gate-dir")
			marker := filepath.Join(t.TempDir(), "gate-ran")
			t.Setenv("MINT_REPLAY_PATH", pathFile)
			t.Setenv("MINT_GATE_MARKER", marker)
			prepare := ""
			if tc.prepare != "" {
				prepare = `<prepare>pwd > "$MINT_REPLAY_PATH"; ` + tc.prepare + `</prepare>`
			}
			spec := unitstore.SpecPath(root, "unit", "001")
			write(t, spec, `<task><protected>scripts/**</protected>`+prepare+`<gates>
first: pwd > "$MINT_REPLAY_PATH"; `+tc.gate+`
second: printf ran > "$MINT_GATE_MARKER"
</gates></task>`)
			maker := &execstate.Maker{Executor: "codex", Vendor: "openai", Model: "gpt", Locality: "remote", ExecutionRef: "maker"}
			state, err := execstate.Init(root, "unit", "001", "a1", maker)
			if err != nil {
				t.Fatal(err)
			}
			if tc.legacy {
				state.BaselineTree = nil
				if err := statehome.WriteJSON(execstate.Path(root, "unit", "001", "a1"), state); err != nil {
					t.Fatal(err)
				}
			}
			write(t, filepath.Join(root, "scripts/gate.txt"), "WEAKENED\n")
			write(t, filepath.Join(root, "scripts/new.txt"), "worker addition\n")
			write(t, filepath.Join(root, "app.go"), "package app\nvar X = 2\n")
			write(t, filepath.Join(root, "local.env"), "LOCAL\n")
			result := Run(root, "unit", "001", "a1", spec)
			wantMode := "replay"
			if tc.legacy {
				wantMode = "in-place"
			}
			if result.Error != "" || result.OK != tc.wantOK || result.Mode != wantMode || !reflect.DeepEqual(result.Reverted, tc.reverted) {
				t.Fatalf("result=%#v", result)
			}
			wantFirst, wantSecond := "pass", "pass"
			if !tc.wantOK {
				wantFirst = "fail"
			}
			if tc.name == "prepare fails" {
				wantSecond = "fail"
				if _, err := os.Stat(marker); !os.IsNotExist(err) {
					t.Fatalf("gate ran after prepare failed: %v", err)
				}
			}
			if result.Results["first"] != wantFirst || result.Results["second"] != wantSecond {
				t.Fatalf("gate results=%v", result.Results)
			}
			state, ok := execstate.Read(root, "unit", "001", "a1")
			if !ok || state.Gates["tier"] != "full" || state.Gates["first"] != wantFirst || state.Gates["second"] != wantSecond {
				t.Fatalf("recorded state=%#v", state)
			}
			body, err := os.ReadFile(filepath.Join(root, "scripts/gate.txt"))
			if err != nil || string(body) != "WEAKENED\n" {
				t.Fatalf("live gate=%q err=%v", body, err)
			}
			dirBytes, err := os.ReadFile(pathFile)
			if err != nil {
				t.Fatal(err)
			}
			gateDir := strings.TrimSpace(string(dirBytes))
			if tc.legacy {
				canonicalRoot, err := filepath.EvalSymlinks(root)
				canonicalDir, dirErr := filepath.EvalSymlinks(gateDir)
				if err != nil || dirErr != nil || canonicalDir != canonicalRoot {
					t.Fatalf("gate dir=%q root=%q errors=%v, %v", gateDir, canonicalRoot, err, dirErr)
				}
			} else if _, err := os.Stat(gateDir); !os.IsNotExist(err) {
				t.Fatalf("replay directory remains: %q err=%v", gateDir, err)
			}
			if got := gitOutput(root, "worktree", "list", "--porcelain"); strings.Count(got, "worktree ") != 1 {
				t.Fatalf("replay worktree remains: %s", got)
			}
		})
	}
}

func TestRunDocsOnlyUsesBaseline(t *testing.T) {
	root := repo(t)
	t.Setenv("MINT_STATE_HOME", t.TempDir())
	write(t, filepath.Join(root, "app.go"), "package app\nvar X = 42\n")
	spec := unitstore.SpecPath(root, "unit", "001")
	write(t, spec, `<task><gates>tests: false</gates></task>`)
	maker := &execstate.Maker{Executor: "codex", Vendor: "openai", Model: "gpt", Locality: "remote", ExecutionRef: "maker"}
	if _, err := execstate.Init(root, "unit", "001", "a1", maker); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, "README.md"), "worker docs\n")
	result := Run(root, "unit", "001", "a1", spec)
	if !result.OK || result.Tier != "skip" || result.Mode != "replay" || result.Error != "" {
		t.Fatalf("result=%#v", result)
	}
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func repo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	run(t, root, "init", "-q")
	run(t, root, "config", "user.email", "t@example.com")
	run(t, root, "config", "user.name", "T")
	os.WriteFile(filepath.Join(root, "app.go"), []byte("package app\nvar X = 1\n"), 0o644)
	write(t, filepath.Join(root, "scripts/gate.txt"), "ORIGINAL\n")
	write(t, filepath.Join(root, ".gitignore"), "local.env\n")
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

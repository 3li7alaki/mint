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

func TestBuildInputInconclusiveGates(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MINT_STATE_HOME", t.TempDir())
	inputGit(t, root, "init", "-q")
	inputWrite(t, filepath.Join(root, "app.go"), "package app\n")
	inputGit(t, root, "add", ".")
	inputGit(t, root, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-qm", "base")
	spec := unitstore.SpecPath(root, "unit", "001")
	inputWrite(t, spec, `<task><gates>
zeta: nosuchbin-mint-xyz
red: exit 3
passed: true
alpha: nosuchbin-mint-xyz
</gates></task>`)
	maker := &execstate.Maker{Executor: "codex", Vendor: "openai", Model: "gpt", Locality: "remote", ExecutionRef: "maker"}
	if _, err := execstate.Init(root, "unit", "001", "a1", maker); err != nil {
		t.Fatal(err)
	}
	inputWrite(t, filepath.Join(root, "app.go"), "package app\nvar X = 2\n")
	input, err := BuildInput(root, BuildOptions{SpecPath: spec, Slug: "unit", SpecID: "001", AttemptID: "a1"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"alpha (exit 127: command not found)", "zeta (exit 127: command not found)"}
	if input.Gates.OK || !reflect.DeepEqual(input.Gates.Inconclusive, want) || !reflect.DeepEqual(input.Gates.Failed, []string{"red (exit 3)"}) {
		t.Fatalf("gate evidence=%#v, want %v", input.Gates, want)
	}
	c1 := clauseResult(t, Enforce(input), 1)
	if c1.Pass || reason(c1) != "gates failed: red (exit 3). Also inconclusive, they could not run: "+strings.Join(want, "; ") {
		t.Fatalf("clause 1=%#v", c1)
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

// Two lanes in ONE worktree (barada, 2026-10-24): each lane's baseline diff holds the
// other's files. A file in a sibling's lane is set aside and named, a file in no lane
// still fails clause 5, and a sibling that finished before this attempt started counts
// for nothing.
func TestBuildInputSetsAsideSiblingLaneFiles(t *testing.T) {
	cases := []struct {
		name       string
		siblingRan string // "open": running beside this one; "before": done before it started
		stray      bool
		wantFiles  []string
		wantScope  bool
	}{
		{"sibling open", "open", false, []string{"src/a/x.go"}, true},
		{"sibling open plus stray", "open", true, []string{"src/a/x.go", "stray.go"}, false},
		{"sibling done before", "before", false, []string{"src/a/x.go", "src/b/y.go"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("MINT_STATE_HOME", t.TempDir())
			inputGit(t, root, "init", "-q")
			inputWrite(t, filepath.Join(root, "base.go"), "package app\n")
			inputGit(t, root, "add", ".")
			inputGit(t, root, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-qm", "base")
			maker := &execstate.Maker{Executor: "codex", Vendor: "openai", Model: "gpt", Locality: "remote", ExecutionRef: "maker"}
			for _, unit := range []string{"lane-a", "lane-b"} {
				lane := "src/a/**"
				if unit == "lane-b" {
					lane = "src/b/**"
				}
				inputWrite(t, unitstore.SpecPath(root, unit, "001"), `<task><scope><can-modify>`+lane+`</can-modify></scope><gates>tests: true</gates></task>`)
			}
			if _, err := execstate.Init(root, "lane-b", "001", "b1", maker); err != nil {
				t.Fatal(err)
			}
			if tc.siblingRan == "before" {
				if _, err := execstate.SetStatus(root, "lane-b", "001", "done-verified", "b1", nil); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := execstate.Init(root, "lane-a", "001", "a1", maker); err != nil {
				t.Fatal(err)
			}
			inputWrite(t, filepath.Join(root, "src/a/x.go"), "package a\n")
			inputWrite(t, filepath.Join(root, "src/b/y.go"), "package b // SIBLING-ONLY\n")
			if tc.stray {
				inputWrite(t, filepath.Join(root, "stray.go"), "package stray\n")
			}
			spec := unitstore.SpecPath(root, "lane-a", "001")
			input, err := BuildInput(root, BuildOptions{SpecPath: spec, Slug: "lane-a", SpecID: "001", AttemptID: "a1"})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(input.ChangedFiles, tc.wantFiles) {
				t.Fatalf("ChangedFiles=%v want %v", input.ChangedFiles, tc.wantFiles)
			}
			open := tc.siblingRan == "open"
			if strings.Contains(input.Patch, "SIBLING-ONLY") == open {
				t.Fatalf("sibling file in patch=%v want %v", !open, !open)
			}
			c5 := clauseResult(t, Enforce(input), 5)
			if c5.Pass != tc.wantScope {
				t.Fatalf("scope pass=%v want %v: %s", c5.Pass, tc.wantScope, reason(c5))
			}
			if tc.wantScope && !strings.Contains(reason(c5), "set aside 1 file(s)") || tc.wantScope && !strings.Contains(reason(c5), "src/b/y.go") {
				t.Fatalf("clause 5 does not name the set-aside file: %s", reason(c5))
			}
		})
	}
}

func TestSplitSiblingFilesKeepsOwnAndUnclaimed(t *testing.T) {
	kept, aside := SplitSiblingFiles([]string{"a/1", "b/2", "c/3", "ab/4"}, []string{"a/**", "ab/**"}, []string{"b/**", "ab/**"})
	if !reflect.DeepEqual(kept, []string{"a/1", "c/3", "ab/4"}) || !reflect.DeepEqual(aside, []string{"b/2"}) {
		t.Fatalf("kept=%v aside=%v", kept, aside)
	}
	// A file/directory swap across lanes is a collision: the sibling's side stays here.
	kept, aside = SplitSiblingFiles([]string{"foo.go", "foo.go/child.txt", "lib/y"}, []string{"*.go"}, []string{"foo.go/**", "lib/**"})
	if !reflect.DeepEqual(kept, []string{"foo.go", "foo.go/child.txt"}) || !reflect.DeepEqual(aside, []string{"lib/y"}) {
		t.Fatalf("swap: kept=%v aside=%v", kept, aside)
	}
	if kept, aside := SplitSiblingFiles([]string{"b/2"}, []string{"a/**"}, nil); len(aside) != 0 || len(kept) != 1 {
		t.Fatalf("no siblings must set nothing aside: kept=%v aside=%v", kept, aside)
	}
}

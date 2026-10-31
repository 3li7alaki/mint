package verify

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"mint/internal/execstate"
	"mint/internal/statehome"
	"mint/internal/unitstore"
)

func TestRunVerifyLock(t *testing.T) {
	for _, tc := range []struct {
		name    string
		body    string
		present bool
	}{
		{"old pid", strconv.Itoa(os.Getpid()), true},
		{"garbage", "not a pid", true},
		{"empty", "", true},
		{"normal", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := repo(t)
			t.Setenv("MINT_STATE_HOME", t.TempDir())
			spec := unitstore.SpecPath(root, "unit", "001")
			write(t, spec, `<task><gates>tests: cat "$MINT_VERIFY_LOCK"</gates></task>`)
			maker := &execstate.Maker{Executor: "codex", Vendor: "openai", Model: "gpt", Locality: "remote", ExecutionRef: "maker"}
			if _, err := execstate.Init(root, "unit", "001", "a1", maker); err != nil {
				t.Fatal(err)
			}
			if _, err := execstate.RecordGate(root, "unit", "001", "tests", "fail", "a1"); err != nil {
				t.Fatal(err)
			}
			logPath := filepath.Join(unitstore.GateLogDir(root, "unit", "001", "a1"), "old.log")
			write(t, logPath, "prior output")
			lockPath := unitstore.VerifyLockPath(root, "unit", "001", "a1")
			t.Setenv("MINT_VERIFY_LOCK", lockPath)
			if tc.present {
				write(t, lockPath, tc.body)
			}
			write(t, filepath.Join(root, "app.go"), "package app\nvar X = 2\n")
			result := Run(root, "unit", "001", "a1", spec)
			if result.Error != "" || !result.OK || result.Results["tests"] != "pass" {
				t.Fatalf("result=%#v", result)
			}
			body, err := os.ReadFile(result.Logs["tests"])
			if err != nil || string(body) != strconv.Itoa(os.Getpid()) {
				t.Fatalf("gate output=%q err=%v", body, err)
			}
			body, err = os.ReadFile(lockPath)
			if err != nil || string(body) != strconv.Itoa(os.Getpid()) {
				t.Fatalf("lock content=%q err=%v", body, err)
			}
			assertVerifyLockFree(t, lockPath)
		})
	}
}

func TestRunMissingAttemptReleasesVerifyLock(t *testing.T) {
	t.Setenv("MINT_STATE_HOME", t.TempDir())
	root := t.TempDir()
	result := Run(root, "unit", "001", "a1", "")
	if result.Error != "attempt does not exist; run mint exec init" {
		t.Fatalf("result=%#v", result)
	}
	assertVerifyLockFree(t, unitstore.VerifyLockPath(root, "unit", "001", "a1"))
}

func assertVerifyLockFree(t *testing.T, path string) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := lockVerifyFile(file); err != nil {
		t.Fatalf("verify lock still held: %v", err)
	}
	defer unlockVerifyFile(file)
}

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
				wantFirst, wantSecond = "inconclusive", "inconclusive"
				if !result.PrepareFailed || result.PrepareLog == "" || len(result.Logs) != 0 {
					t.Fatalf("prepare failure logs=%#v", result)
				}
				entries, err := os.ReadDir(unitstore.GateLogDir(root, "unit", "001", "a1"))
				if err != nil || len(entries) != 1 || entries[0].Name() != "prepare.log" {
					t.Fatalf("prepare failure log files=%v err=%v", entries, err)
				}
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
	if _, err := os.Stat(unitstore.GateLogDir(root, "unit", "001", "a1")); !os.IsNotExist(err) {
		t.Fatalf("docs-only run created logs: %v", err)
	}
}

func TestRunGateLogs(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		name := "replay"
		if legacy {
			name = "legacy"
		}
		t.Run(name, func(t *testing.T) {
			root := repo(t)
			t.Setenv("MINT_STATE_HOME", t.TempDir())
			spec := unitstore.SpecPath(root, "unit", "001")
			write(t, spec, `<task><prepare>printf 'PREP\n'</prepare><gates>
z/last._-9: printf 'LAST\n'
a space: printf 'OUT\n'; printf 'ERR\n' >&2
a/space: printf 'COLLISION\n'
é: printf 'UTF8\n'
</gates></task>`)
			maker := &execstate.Maker{Executor: "codex", Vendor: "openai", Model: "gpt", Locality: "remote", ExecutionRef: "maker"}
			state, err := execstate.Init(root, "unit", "001", "a1", maker)
			if err != nil {
				t.Fatal(err)
			}
			if legacy {
				state.BaselineTree = nil
				if err := statehome.WriteJSON(execstate.Path(root, "unit", "001", "a1"), state); err != nil {
					t.Fatal(err)
				}
			}
			write(t, filepath.Join(root, "app.go"), "package app\nvar X = 2\n")
			result := Run(root, "unit", "001", "a1", spec)
			if result.Error != "" || !result.OK || result.PrepareFailed || len(result.Logs) != 4 {
				t.Fatalf("result=%#v", result)
			}
			dir := unitstore.GateLogDir(root, "unit", "001", "a1")
			info, err := os.Stat(dir)
			if err != nil || info.Mode().Perm() != 0o700 {
				t.Fatalf("log dir info=%v err=%v", info, err)
			}
			logs := []struct{ label, file, body string }{
				{"a space", "01-a-space.log", "OUT\nERR\n"},
				{"a/space", "02-a-space.log", "COLLISION\n"},
				{"z/last._-9", "03-z-last._-9.log", "LAST\n"},
				{"é", "04---.log", "UTF8\n"},
			}
			if legacy {
				if result.PrepareLog != "" {
					t.Fatalf("legacy prepare log=%q", result.PrepareLog)
				}
			} else {
				if result.PrepareLog != filepath.Join(dir, "prepare.log") {
					t.Fatalf("prepare log=%q", result.PrepareLog)
				}
				logs = append(logs, struct{ label, file, body string }{"", "prepare.log", "PREP\n"})
			}
			for _, log := range logs {
				path := filepath.Join(dir, log.file)
				if log.label != "" && result.Logs[log.label] != path {
					t.Fatalf("log path for %q=%q, want %q", log.label, result.Logs[log.label], path)
				}
				body, err := os.ReadFile(path)
				if err != nil || string(body) != log.body {
					t.Fatalf("log %q=%q err=%v", path, body, err)
				}
				info, err := os.Stat(path)
				if err != nil || info.Mode().Perm() != 0o600 {
					t.Fatalf("log info=%v err=%v", info, err)
				}
			}
			write(t, spec, `<task><gates>a space: printf 'NEW\n'</gates></task>`)
			result = Run(root, "unit", "001", "a1", spec)
			if result.Error != "" || !result.OK || len(result.Logs) != 1 || result.PrepareLog != "" {
				t.Fatalf("second result=%#v", result)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 1 || entries[0].Name() != "01-a-space.log" {
				t.Fatalf("stale logs remain: entries=%v err=%v", entries, err)
			}
			body, err := os.ReadFile(result.Logs["a space"])
			if err != nil || string(body) != "NEW\n" {
				t.Fatalf("second log=%q err=%v", body, err)
			}
		})
	}
}

func TestRunGateResults(t *testing.T) {
	const hint = "gates ran in a replay worktree that holds only tracked files, so gitignored dependencies (node_modules, vendor, .env, generated code) are absent; declare them with <prepare>, e.g. mint spec set <spec> --prepare 'pnpm install --frozen-lockfile'"
	for _, tc := range []struct {
		name       string
		gate       string
		prepare    string
		legacy     bool
		wantResult string
		wantReason string
		wantHint   string
	}{
		{"missing command", "nosuchbin-mint-xyz", "", false, "inconclusive", "exit 127: command not found", hint},
		{"not executable", "./script.sh", "", false, "inconclusive", "exit 126: command not executable", hint},
		{"signal", "kill -9 $$", "", false, "inconclusive", "killed by a signal", hint},
		{"failed", "exit 3", "", false, "fail", "exit 3", hint},
		{"child signal exit", "exit 137", "", false, "fail", "exit 137", hint},
		{"prepare failed", "true", "false", false, "inconclusive", "not run: prepare failed (exit 1)", ""},
		{"prepare missing command", "true", "nosuchbin-mint-xyz", false, "inconclusive", "not run: prepare failed (exit 127: command not found)", ""},
		{"prepare declared", "nosuchbin-mint-xyz", "true", false, "inconclusive", "exit 127: command not found", ""},
		{"passed", "true", "", false, "pass", "", ""},
		{"legacy", "nosuchbin-mint-xyz", "", true, "inconclusive", "exit 127: command not found", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := repo(t)
			t.Setenv("MINT_STATE_HOME", t.TempDir())
			if tc.gate == "./script.sh" {
				write(t, filepath.Join(root, "script.sh"), "#!/bin/sh\nexit 0\n")
				run(t, root, "add", "script.sh")
			}
			prepare := ""
			if tc.prepare != "" {
				prepare = "<prepare>" + tc.prepare + "</prepare>"
			}
			spec := unitstore.SpecPath(root, "unit", "001")
			write(t, spec, "<task>"+prepare+"<gates>\ntests: "+tc.gate+"\nother: true\n</gates></task>")
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
			write(t, filepath.Join(root, "app.go"), "package app\nvar X = 2\n")
			result := Run(root, "unit", "001", "a1", spec)
			if result.Error != "" || result.OK != (tc.wantResult == "pass") || !result.Declared || result.Tier != "full" {
				t.Fatalf("result=%#v", result)
			}
			if result.Results["tests"] != tc.wantResult || result.Reasons["tests"] != tc.wantReason || result.Hint != tc.wantHint {
				t.Fatalf("result=%#v, want %s, %q, hint %q", result, tc.wantResult, tc.wantReason, tc.wantHint)
			}
			wantOther, wantOtherReason := "pass", ""
			wantReasons := 0
			if tc.wantResult != "pass" {
				wantReasons++
			}
			if strings.HasPrefix(tc.wantReason, "not run: prepare failed") {
				wantOther, wantOtherReason = "inconclusive", tc.wantReason
				wantReasons++
				if !result.PrepareFailed || len(result.Logs) != 0 {
					t.Fatalf("gates ran after prepare failed: %#v", result)
				}
			}
			if len(result.Results) != 2 || result.Results["other"] != wantOther || result.Reasons["other"] != wantOtherReason || len(result.Reasons) != wantReasons {
				t.Fatalf("gate results=%#v", result)
			}
			state, ok := execstate.Read(root, "unit", "001", "a1")
			if !ok || state.Gates["tests"] != tc.wantResult || state.Gates["other"] != wantOther {
				t.Fatalf("recorded state=%#v", state)
			}
		})
	}
}

func TestClassifyGateErrors(t *testing.T) {
	exitErr := runGate(t.TempDir(), "exit 127", io.Discard)
	if exitErr == nil {
		t.Fatal("exit 127 succeeded")
	}
	for _, tc := range []struct {
		name   string
		err    error
		reason string
	}{
		{"could not start", errors.New("shell unavailable"), "could not start: shell unavailable"},
		{"wrapped exit", fmt.Errorf("wrapped: %w", exitErr), "exit 127: command not found"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, reason := classifyGate(tc.err)
			if result != "inconclusive" || reason != tc.reason {
				t.Fatalf("result=%q reason=%q, want inconclusive, %q", result, reason, tc.reason)
			}
		})
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

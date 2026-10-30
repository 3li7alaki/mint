package verifycmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"mint/internal/execstate"
	"mint/internal/unitstore"
	"mint/internal/verify"
)

func TestTail(t *testing.T) {
	var lines []string
	for i := 1; i <= 25; i++ {
		lines = append(lines, fmt.Sprintf("line %02d", i))
	}
	for _, tc := range []struct {
		name    string
		body    string
		missing bool
		want    []string
	}{
		{"last 20", strings.Join(lines, "\n") + "\n", false, lines[5:]},
		{"no final newline", strings.Join(lines, "\n"), false, lines[5:]},
		{"empty", "", false, []string{"(no output)"}},
		{"missing", "", true, []string{"(no output)"}},
		{"long line", strings.Repeat("x", 128*1024), false, []string{strings.Repeat("x", 128*1024)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "gate.log")
			if !tc.missing {
				if err := os.WriteFile(path, []byte(tc.body), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if got := tail(path); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("tail=%q, want %q", got, tc.want)
			}
		})
	}
}

func TestRunLogOutput(t *testing.T) {
	for _, tc := range []struct {
		name    string
		prepare string
		gate    string
		json    bool
		wantOK  bool
	}{
		{"failed gates", "", "printf 'OUT\\n'; printf 'ERR\\n' >&2; false", false, false},
		{"failed prepare", "printf 'PREP\\n'; false", "true", false, false},
		{"passed gates", "", "printf 'PASS\\n'", false, true},
		{"json", "", "printf 'OUT\\n'; false", true, false},
		{"inconclusive gates", "", "printf 'OUT\\n'; printf 'ERR\\n' >&2; exit 127", false, false},
		{"json inconclusive", "", "exit 127", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("MINT_STATE_HOME", t.TempDir())
			root := t.TempDir()
			for _, args := range [][]string{
				{"init", "-q"},
				{"-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "--allow-empty", "-qm", "base"},
			} {
				cmd := exec.Command("git", args...)
				cmd.Dir = root
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("git %v: %v %s", args, err, out)
				}
			}
			spec := unitstore.SpecPath(root, "unit", "001")
			if err := os.MkdirAll(filepath.Dir(spec), 0o700); err != nil {
				t.Fatal(err)
			}
			xml := "<task><prepare>" + tc.prepare + "</prepare><gates>\nzeta: " + tc.gate + "\nmiddle: true\nalpha: " + tc.gate + "\n</gates></task>"
			if err := os.WriteFile(spec, []byte(xml), 0o600); err != nil {
				t.Fatal(err)
			}
			maker := &execstate.Maker{Executor: "codex", Vendor: "openai", Model: "gpt", Locality: "remote", ExecutionRef: "maker"}
			if _, err := execstate.Init(root, "unit", "001", "a1", maker); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "app.go"), []byte("package app\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			var stdout bytes.Buffer
			code, err := Run(root, []string{"unit", "001"}, Flags{JSON: tc.json}, &stdout)
			if err != nil || (code == 0) != tc.wantOK {
				t.Fatalf("exit=%d err=%v output=%s", code, err, stdout.String())
			}
			dir := unitstore.GateLogDir(root, "unit", "001", "a1")
			if tc.json {
				var result verify.Result
				if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if result.OK || result.Logs["alpha"] != filepath.Join(dir, "01-alpha.log") {
					t.Fatalf("json result=%#v", result)
				}
				wantResult, wantReason := "fail", "exit 1"
				if tc.name == "json inconclusive" {
					wantResult, wantReason = "inconclusive", "exit 127: command not found"
				}
				if result.Results["alpha"] != wantResult || result.Reasons["alpha"] != wantReason || result.Hint == "" {
					t.Fatalf("json reasons and hint=%#v", result)
				}
				return
			}
			want := "  tier: full - ok alpha  ok middle  ok zeta\n"
			if tc.prepare != "" {
				want = "  tier: full - inconclusive alpha  inconclusive middle  inconclusive zeta\n" +
					fmt.Sprintf("  prepare: last 20 lines of %s\n    | PREP\n", filepath.Join(dir, "prepare.log")) +
					"  alpha: inconclusive, not run, prepare failed\n  middle: inconclusive, not run, prepare failed\n  zeta: inconclusive, not run, prepare failed\n"
			} else if !tc.wantOK {
				status, reason := "fail", "exit 1"
				if tc.name == "inconclusive gates" {
					status, reason = "inconclusive", "exit 127: command not found"
				}
				want = fmt.Sprintf("  tier: full - %s alpha  ok middle  %s zeta\n", status, status) +
					fmt.Sprintf("  alpha: %s, %s; last 20 lines of %s\n    | OUT\n    | ERR\n", status, reason, filepath.Join(dir, "01-alpha.log")) +
					fmt.Sprintf("  zeta: %s, %s; last 20 lines of %s\n    | OUT\n    | ERR\n", status, reason, filepath.Join(dir, "03-zeta.log")) +
					"  hint: gates ran in a replay worktree that holds only tracked files, so gitignored dependencies (node_modules, vendor, .env, generated code) are absent; declare them with <prepare>, e.g. mint spec set <spec> --prepare 'pnpm install --frozen-lockfile'\n"
			}
			if got := stdout.String(); got != want {
				t.Fatalf("output=%q, want %q", got, want)
			}
		})
	}
}

func TestGateLine(t *testing.T) {
	for _, tc := range []struct {
		name    string
		results map[string]string
		want    string
	}{
		{"empty", nil, "no gates to run"},
		{"inconclusive", map[string]string{"lint": "inconclusive"}, "inconclusive lint"},
		{"mixed sorted", map[string]string{"zeta": "fail", "middle": "inconclusive", "alpha": "pass"}, "ok alpha  inconclusive middle  fail zeta"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := gateLine(tc.results); got != tc.want {
				t.Fatalf("gateLine=%q, want %q", got, tc.want)
			}
		})
	}
}

package donecmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"mint/internal/execstate"
	"mint/internal/floor"
	"mint/internal/receipt"
	"mint/internal/snapshot"
	"mint/internal/statehome"
	"mint/internal/unitstore"
)

func TestDoneIssuesAttemptAndSnapshotBoundReceipt(t *testing.T) {
	root := doneRepo(t)
	t.Setenv("MINT_STATE_HOME", t.TempDir())
	if err := unitstore.Ensure(root); err != nil {
		t.Fatal(err)
	}
	spec := unitstore.SpecPath(root, "unit", "001")
	os.MkdirAll(filepath.Dir(spec), 0o700)
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
	verdict := map[string]any{"schemaVersion": 1, "accepted": true, "executor": "opencode", "vendor": "zai", "model": "glm", "locality": "remote", "executionRef": "checker"}
	b, _ := json.Marshal(verdict)
	if err := statehome.Write(unitstore.VerdictPath(root, "unit", "001", "a1"), b); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	code, err := Run(root, []string{"unit", "001"}, Flags{Attempt: "a1", JSON: true}, &out, &bytes.Buffer{})
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v out=%s", code, err, out.String())
	}
	paths := receipt.List(root, "unit", "001")
	if len(paths) != 1 {
		t.Fatalf("receipts=%v", paths)
	}
	record, err := receipt.Read(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	if record.AttemptID != "a1" || record.Evidence.GateMode != "replay" || !receipt.Validate(root, record).Current {
		t.Fatalf("record=%#v", record)
	}
	if record.VerdictSource != "declared" || record.Evidence.Acceptance["source"] != "declared" {
		t.Fatalf("verdict source=%q acceptance=%v", record.VerdictSource, record.Evidence.Acceptance)
	}
	os.WriteFile(filepath.Join(root, "app.go"), []byte("package app\nvar X = 3\n"), 0o644)
	if receipt.Validate(root, record).Current {
		t.Fatal("receipt stayed current after source change")
	}
}

func TestDoneRejectsSourceChangeDuringFloorEvaluation(t *testing.T) {
	root := doneRepo(t)
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
	verdict := map[string]any{"schemaVersion": 1, "accepted": true, "executor": "opencode", "vendor": "zai", "model": "glm", "locality": "remote", "executionRef": "checker"}
	b, _ := json.Marshal(verdict)
	if err := statehome.Write(unitstore.VerdictPath(root, "unit", "001", "a1"), b); err != nil {
		t.Fatal(err)
	}
	originalCapture := captureSnapshot
	t.Cleanup(func() { captureSnapshot = originalCapture })
	calls := 0
	captureSnapshot = func(snapshotRoot, base string) (snapshot.Source, error) {
		calls++
		if calls == 2 {
			if err := os.WriteFile(filepath.Join(root, "app.go"), []byte("package app\nvar X = 9\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return originalCapture(snapshotRoot, base)
	}
	var out, stderr bytes.Buffer
	code, err := Run(root, []string{"unit", "001"}, Flags{Attempt: "a1"}, &out, &stderr)
	if err != nil || code == 0 || !bytes.Contains(stderr.Bytes(), []byte("source changed")) {
		t.Fatalf("code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if paths := receipt.List(root, "unit", "001"); len(paths) != 0 {
		t.Fatalf("receipt issued for raced source: %v", paths)
	}
}

func TestDoneCheckerAttestsProtectedEdit(t *testing.T) {
	root := checkerUnit(t, true, "true")
	flags := checkerFlags("printf 'ALLOW: looks right\\n'")
	var out, stderr bytes.Buffer
	code, err := Run(root, []string{"unit", "001"}, flags, &out, &stderr)
	if err != nil || code == 0 || !strings.Contains(out.String(), "verifier-tampering") {
		t.Fatalf("unattested protected edit: code=%d err=%v out=%s stderr=%s", code, err, &out, &stderr)
	}
	flags.Checker = "printf '\\nALLOW: looks right\\nATTEST tampering: reviewed the workflow edit\\n'"
	out.Reset()
	stderr.Reset()
	code, err = Run(root, []string{"unit", "001"}, flags, &out, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v out=%s stderr=%s", code, err, &out, &stderr)
	}
	paths := receipt.List(root, "unit", "001")
	if len(paths) != 1 {
		t.Fatalf("receipts=%v", paths)
	}
	record, err := receipt.Read(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	verdict := record.Evidence.Acceptance
	if record.VerdictSource != "checker" || verdict["source"] != "checker" || verdict["accepted"] != true || verdict["checkerReason"] != "looks right" || verdict["tamperingReviewed"] != true || verdict["tamperingReason"] != "reviewed the workflow edit" {
		t.Fatalf("source=%q verdict=%v", record.VerdictSource, verdict)
	}
	if !record.Clauses[3].Pass {
		t.Fatalf("anti-gaming=%#v", record.Clauses[3])
	}
	stored := floor.ReadVerdict(unitstore.VerdictPath(root, "unit", "001", "a1"))
	if stored["source"] != "checker" || stored["accepted"] != true {
		t.Fatalf("stored verdict=%v", stored)
	}
}

func TestDoneCheckerFailuresLeaveNoReceipt(t *testing.T) {
	for _, tc := range []struct {
		name, command, reason string
	}{
		{"block", "printf 'BLOCK: missing test\\n'", "missing test"},
		{"malformed", "printf 'looks right\\nALLOW: too late\\n'", "malformed checker first line"},
		{"missing reason", "printf 'ALLOW: \\n'", "malformed checker first line"},
		{"case sensitive", "printf 'allow: looks right\\n'", "malformed checker first line"},
		{"nonzero", "printf 'ALLOW: looks right\\n'; printf 'model failed' >&2; exit 7", "checker failed: exit status 7: model failed"},
		{"empty", "printf '\\n  \\n'", "checker produced empty output"},
		{"stderr only", "printf 'ALLOW: looks right\\n' >&2", "checker produced empty output"},
		{"oversized", "printf 'ALLOW: looks right\\n'; head -c 1048576 /dev/zero", "checker stdout exceeded 1 MiB"},
		{"unbounded stdout", "printf 'ALLOW: looks right\\n'; cat /dev/zero", "checker stdout exceeded 1 MiB"},
		{"timeout", "sleep 30", "checker timed out after 1s"},
		{"timeout children", "sh -c 'sleep 30 & wait'", "checker timed out after 1s"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := checkerUnit(t, false, "true")
			flags := checkerFlags(tc.command)
			flags.CheckerTimeout = time.Second
			var out, stderr bytes.Buffer
			started := time.Now()
			code, err := Run(root, []string{"unit", "001"}, flags, &out, &stderr)
			if elapsed := time.Since(started); elapsed > 8*time.Second {
				t.Fatalf("checker took %s", elapsed)
			}
			if err != nil || code == 0 || !strings.Contains(stderr.String(), tc.reason) {
				t.Fatalf("code=%d err=%v out=%s stderr=%s", code, err, &out, &stderr)
			}
			if paths := receipt.List(root, "unit", "001"); len(paths) != 0 {
				t.Fatalf("failed checker issued receipts: %v", paths)
			}
			state, ok := execstate.Read(root, "unit", "001", "a1")
			if !ok || state.Status != "running" {
				t.Fatalf("failed checker terminated attempt: %#v", state)
			}
		})
	}
}

func TestDoneCheckerReceivesFloorInputAfterGates(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CHECKER_INPUT", filepath.Join(dir, "input.json"))
	t.Setenv("CHECKER_GATE", filepath.Join(dir, "gate-ran"))
	root := checkerUnit(t, false, `touch "$CHECKER_GATE"`)
	flags := checkerFlags(`tee "$CHECKER_INPUT" | grep -q '"changedFiles":\["app.go"\]' && test -f "$CHECKER_GATE" && test -f app.go && printf 'ALLOW: checked stdin\n'`)
	var out, stderr bytes.Buffer
	code, err := Run(root, []string{"unit", "001"}, flags, &out, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v out=%s stderr=%s", code, err, &out, &stderr)
	}
	b, err := os.ReadFile(filepath.Join(dir, "input.json"))
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		SchemaVersion int
		Spec          string
		ChangedFiles  []string
		Patch         string
		Gates         struct {
			OK       bool
			Mode     string
			Reverted []string
		}
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.SchemaVersion != 1 || !strings.Contains(payload.Spec, "<goal>x</goal>") || len(payload.ChangedFiles) != 1 || payload.ChangedFiles[0] != "app.go" || !strings.Contains(payload.Patch, "+var X = 2") || !payload.Gates.OK || payload.Gates.Mode != "replay" || payload.Gates.Reverted == nil {
		t.Fatalf("checker input=%s", b)
	}
}

func TestDoneCheckerValidatesFlagsBeforeRunning(t *testing.T) {
	for _, tc := range []struct {
		name, reason string
		change       func(*Flags)
	}{
		{"verdict conflict", "mutually exclusive", func(f *Flags) { f.Verdict = "verdict.json" }},
		{"missing executor", "checker provenance:", func(f *Flags) { f.CheckerExecutor = "" }},
		{"missing vendor", "checker provenance:", func(f *Flags) { f.CheckerVendor = "" }},
		{"missing model", "checker provenance:", func(f *Flags) { f.CheckerModel = "" }},
		{"missing locality", "checker provenance:", func(f *Flags) { f.CheckerLocality = "" }},
		{"missing execution ref", "checker provenance:", func(f *Flags) { f.CheckerExecutionRef = "" }},
		{"invalid locality", "checker provenance:", func(f *Flags) { f.CheckerLocality = "cloud" }},
		{"invalid provenance", "checker provenance:", func(f *Flags) { f.CheckerVendor = "bad\nvalue" }},
		{"invalid timeout", "--checker-timeout must be positive", func(f *Flags) { f.CheckerTimeout = -time.Second }},
		{"blank checker", "--checker requires a non-empty command", func(f *Flags) { f.Checker = " " }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			marker := filepath.Join(root, "checker-ran")
			t.Setenv("CHECKER_RAN", marker)
			flags := checkerFlags(`touch "$CHECKER_RAN"`)
			tc.change(&flags)
			code, err := Run(root, []string{"unit", "001"}, flags, &bytes.Buffer{}, &bytes.Buffer{})
			if code == 0 || err == nil || !strings.Contains(err.Error(), tc.reason) {
				t.Fatalf("invalid flags accepted: code=%d err=%v", code, err)
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatalf("checker ran with invalid flags: %v", err)
			}
		})
	}
}

func TestCheckerBuildsOnlyKnownAttestations(t *testing.T) {
	command := `printf 'ALLOW: looks right\nATTEST adversarial: exercised boundary inputs\nATTEST safety: checked trust boundaries\nATTEST tampering: reviewed workflow edits\nATTEST vendorDiversity: only one vendor available\nATTEST unknown: ignore this name\n'; printf 'BLOCK: stderr is not a verdict\nATTEST safety: stderr spoof\n' >&2`
	provenance := execstate.Provenance{Executor: "opencode", Vendor: "zai", Model: "glm", Locality: "remote", ExecutionRef: "checker"}
	got := runChecker(t.TempDir(), command, provenance, time.Second, floor.Input{})
	want := map[string]any{
		"schemaVersion": 1, "accepted": true, "source": "checker", "checkerReason": "looks right",
		"executor": "opencode", "vendor": "zai", "model": "glm", "locality": "remote", "executionRef": "checker",
		"adversarialReviewed": true, "adversarialReason": "exercised boundary inputs",
		"safetyReviewed": true, "safetyReason": "checked trust boundaries",
		"tamperingReviewed": true, "tamperingReason": "reviewed workflow edits",
		"vendorDiversityWaived": true, "vendorDiversityReason": "only one vendor available",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("verdict=%v want=%v", got, want)
	}
}

func TestDoneExplicitVerdictSource(t *testing.T) {
	for _, source := range []string{"", "external"} {
		t.Run(source, func(t *testing.T) {
			root := checkerUnit(t, false, "true")
			verdict := map[string]any{"schemaVersion": 1, "accepted": true, "executor": "opencode", "vendor": "zai", "model": "glm", "locality": "remote", "executionRef": "checker"}
			want := source
			if source != "" {
				verdict["source"] = source
			} else {
				want = "declared"
			}
			path := filepath.Join(t.TempDir(), "verdict.json")
			if err := statehome.WriteJSON(path, verdict); err != nil {
				t.Fatal(err)
			}
			var out, stderr bytes.Buffer
			code, err := Run(root, []string{"unit", "001"}, Flags{Verdict: path, Attempt: "a1", JSON: true}, &out, &stderr)
			if err != nil || code != 0 {
				t.Fatalf("code=%d err=%v out=%s stderr=%s", code, err, &out, &stderr)
			}
			var response struct{ Receipt receipt.Record }
			if err := json.Unmarshal(out.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Receipt.VerdictSource != want {
				t.Fatalf("source=%q want=%q", response.Receipt.VerdictSource, want)
			}
			stored := floor.ReadVerdict(unitstore.VerdictPath(root, "unit", "001", "a1"))
			if stored["source"] != want {
				t.Fatalf("stored verdict=%v", stored)
			}
		})
	}
}

func checkerFlags(command string) Flags {
	return Flags{
		Attempt: "a1", Checker: command,
		CheckerExecutor: "opencode", CheckerVendor: "zai", CheckerModel: "glm",
		CheckerLocality: "remote", CheckerExecutionRef: "checker",
	}
}

func checkerUnit(t *testing.T, protected bool, gate string) string {
	t.Helper()
	root := doneRepo(t)
	t.Setenv("MINT_STATE_HOME", t.TempDir())
	if err := unitstore.Ensure(root); err != nil {
		t.Fatal(err)
	}
	protection := ""
	if protected {
		protection = "<protected>app.go</protected>"
	}
	xml := `<task><id>001</id><title>x</title><goal>x</goal><scope><can-modify>app.go</can-modify></scope>` + protection + `<acceptance>WHEN x, THE system SHALL y</acceptance><gates>tests: ` + gate + `</gates></task>`
	if err := statehome.Write(unitstore.SpecPath(root, "unit", "001"), []byte(xml)); err != nil {
		t.Fatal(err)
	}
	maker := &execstate.Maker{Executor: "codex", Vendor: "openai", Model: "gpt", Locality: "remote", ExecutionRef: "maker"}
	if _, err := execstate.Init(root, "unit", "001", "a1", maker); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "app.go"), []byte("package app\nvar X = 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func doneRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "t@example.com"}, {"config", "user.name", "T"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v %s", args, err, out)
		}
	}
	os.WriteFile(filepath.Join(root, "app.go"), []byte("package app\nvar X = 1\n"), 0o644)
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = root
	cmd.Run()
	cmd = exec.Command("git", "commit", "-qm", "base")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit: %v %s", err, out)
	}
	return root
}

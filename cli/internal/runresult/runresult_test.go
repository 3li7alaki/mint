package runresult

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExitCodeConstantsDistinct(t *testing.T) {
	if Done != 0 || HardFailure != 1 || Escalation != 2 {
		t.Fatalf("codes = %d/%d/%d, want 0/1/2", Done, HardFailure, Escalation)
	}
	if Done == HardFailure || Done == Escalation || HardFailure == Escalation {
		t.Fatalf("exit codes must be distinct")
	}
}

func TestClassifyExit(t *testing.T) {
	passingGates := &Gates{Tier: "full", Results: map[string]string{"lint": "pass"}, OK: true}
	failingGates := &Gates{Tier: "full", Results: map[string]string{"lint": "fail"}, OK: false}
	cases := []struct {
		name   string
		report Report
		want   int
	}{
		{"clean pass", Report{Status: "passed", Gates: passingGates, ScopeOK: Bool(true)}, Done},
		{"scope escaped", Report{Status: "passed", Gates: passingGates, ScopeOK: Bool(false)}, HardFailure},
		{"gate failed", Report{Status: "passed", Gates: failingGates, ScopeOK: Bool(true)}, HardFailure},
		{"failed", Report{Status: "failed", Gates: passingGates, ScopeOK: Bool(true)}, HardFailure},
		{"timeout", Report{Status: "timeout", Gates: passingGates, ScopeOK: Bool(true)}, HardFailure},
		{"escalation", Report{Status: "escalation"}, Escalation},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyExit(tc.report); got != tc.want {
				t.Fatalf("ClassifyExit() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestBuildDoneResult(t *testing.T) {
	report := Report{
		Status:  "passed",
		Gates:   &Gates{Tier: "full", Results: map[string]string{"lint": "pass", "types": "pass", "tests": "pass"}, OK: true},
		ScopeOK: Bool(true),
	}
	result := Build(report, IDs{SpecID: "001", SessionID: "sid-1"})
	if result.SpecID == nil || *result.SpecID != "001" {
		t.Fatalf("spec id = %#v", result.SpecID)
	}
	if result.SessionID != "sid-1" || result.Status != "passed" || result.ExitCode != Done {
		t.Fatalf("result identity/status = %#v", result)
	}
	if result.Gates != report.Gates || result.OpenBlockers != 0 || result.Resumable {
		t.Fatalf("done result = %#v", result)
	}
}

func TestBuildEscalationResult(t *testing.T) {
	result := Build(Report{Status: "escalation"}, IDs{SessionID: "sid-2"})
	if result.SpecID != nil {
		t.Fatalf("spec id = %#v, want nil", result.SpecID)
	}
	if result.Gates != nil || result.OpenBlockers != 1 || !result.Resumable || result.ExitCode != Escalation {
		t.Fatalf("escalation result = %#v", result)
	}
}

func TestBuildOpenBlockers(t *testing.T) {
	result := Build(Report{
		Status:  "failed",
		Gates:   &Gates{Tier: "full", Results: map[string]string{"lint": "pass", "types": "fail", "tests": "fail"}, OK: false},
		ScopeOK: Bool(true),
	}, IDs{SpecID: "002", SessionID: "sid-3"})
	if result.OpenBlockers != 2 || result.ExitCode != HardFailure || !result.Resumable {
		t.Fatalf("gate blockers result = %#v", result)
	}

	result = Build(Report{
		Status:  "passed",
		Gates:   &Gates{Tier: "full", Results: map[string]string{"lint": "pass", "types": "pass", "tests": "pass"}, OK: true},
		ScopeOK: Bool(false),
	}, IDs{SpecID: "003", SessionID: "sid-4"})
	if result.OpenBlockers != 1 || result.ExitCode != HardFailure || !result.Resumable {
		t.Fatalf("scope blocker result = %#v", result)
	}

	result = Build(Report{
		Status:  "passed",
		Gates:   &Gates{Tier: "full", Results: map[string]string{"lint": "pass", "types": "fail", "tests": "pass"}, OK: false},
		ScopeOK: Bool(true),
	}, IDs{SpecID: "004", SessionID: "sid-5"})
	if result.OpenBlockers != 1 || result.ExitCode != HardFailure || !result.Resumable {
		t.Fatalf("passed-but-gate-failed result = %#v", result)
	}

	result = Build(Report{
		Status: "failed",
		Gates:  &Gates{Tier: "full", Results: map[string]string{"lint": "pass"}, OK: true},
	}, IDs{SpecID: "005", SessionID: "sid-6"})
	if result.OpenBlockers != 0 {
		t.Fatalf("absent scopeOk should not count as scope escape: %#v", result)
	}
}

func TestWriteResult(t *testing.T) {
	root := t.TempDir()
	report := Report{
		Status:  "passed",
		Gates:   &Gates{Tier: "full", Results: map[string]string{"lint": "pass", "types": "pass", "tests": "pass"}, OK: true},
		ScopeOK: Bool(true),
	}
	ids := IDs{SpecID: "010", SessionID: "sid-write"}
	returned, err := Write(root, report, ids)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	path := filepath.Join(root, ".mint", "sessions", ids.SessionID, "result.json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	var onDisk Result
	if err := json.Unmarshal(b, &onDisk); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if onDisk.ExitCode != Done || onDisk.OpenBlockers != 0 || onDisk.Resumable {
		t.Fatalf("on disk = %#v", onDisk)
	}
	if returned.ExitCode != onDisk.ExitCode || returned.SessionID != onDisk.SessionID {
		t.Fatalf("returned = %#v, on disk = %#v", returned, onDisk)
	}
}

func TestWriteEscalationSerializesNullGates(t *testing.T) {
	root := t.TempDir()
	ids := IDs{SpecID: "011", SessionID: "sid-escalation"}
	returned, err := Write(root, Report{Status: "escalation"}, ids)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	path := filepath.Join(root, ".mint", "sessions", ids.SessionID, "result.json")
	var raw map[string]any
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if raw["gates"] != nil {
		t.Fatalf("gates = %#v, want JSON null", raw["gates"])
	}
	if returned.ExitCode != Escalation || returned.OpenBlockers != 1 || !returned.Resumable {
		t.Fatalf("returned = %#v", returned)
	}
}

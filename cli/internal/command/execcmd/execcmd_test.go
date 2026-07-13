package execcmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mint/internal/execstate"
	"mint/internal/session"
)

func TestExecInitRecordsMaker(t *testing.T) {
	root := rootWithMint(t)
	var out bytes.Buffer
	code, err := Run(root, []string{"init", "feat", "001"}, Flags{Session: "sid-a", MakerEngine: "codex"}, &out, &bytes.Buffer{})
	if err != nil || code != 0 {
		t.Fatalf("init code=%d err=%v", code, err)
	}
	state, ok := execstate.Read(root, "feat", "001", "sid-a")
	if !ok {
		t.Fatal("execution state missing")
	}
	if state.Maker == nil || state.Maker.Engine != "codex" || state.Maker.Session != "sid-a" {
		t.Fatalf("maker = %#v", state.Maker)
	}
	if !strings.Contains(out.String(), `"status": "running"`) {
		t.Fatalf("output = %s", out.String())
	}
}

func TestExecRecordsGateReviewStatus(t *testing.T) {
	root := rootWithMint(t)
	if _, err := execstate.Init(root, "feat", "001", "sid-a", nil); err != nil {
		t.Fatal(err)
	}
	if code, err := Run(root, []string{"record-gate", "feat", "001", "tests", "pass"}, Flags{Session: "sid-a"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil || code != 0 {
		t.Fatalf("record-gate code=%d err=%v", code, err)
	}
	if code, err := Run(root, []string{"record-review", "feat", "001", "quality", "passed"}, Flags{Session: "sid-a", ByEngine: "codex", BySession: "reviewer"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil || code != 0 {
		t.Fatalf("record-review code=%d err=%v", code, err)
	}
	if code, err := Run(root, []string{"set-status", "feat", "001", "passed"}, Flags{Session: "sid-a", Commit: "abc123"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil || code != 0 {
		t.Fatalf("set-status code=%d err=%v", code, err)
	}
	state, _ := execstate.Read(root, "feat", "001", "sid-a")
	if state.Gates["tests"] != "pass" || state.Reviews["quality"].Verdict != "passed" || state.Status != "passed" || state.Commit == nil || *state.Commit != "abc123" {
		t.Fatalf("state = %#v", state)
	}
}

func TestExecRecordReviewRequiresProvenance(t *testing.T) {
	root := rootWithMint(t)
	if _, err := execstate.Init(root, "feat", "001", "sid-a", nil); err != nil {
		t.Fatal(err)
	}
	_, err := Run(root, []string{"record-review", "feat", "001", "quality", "passed"}, Flags{Session: "sid-a"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "provenance is required") {
		t.Fatalf("bare pass should fail fast, got %v", err)
	}
	state, _ := execstate.Read(root, "feat", "001", "sid-a")
	if len(state.Reviews) != 0 {
		t.Fatalf("bare pass was stored: %#v", state.Reviews)
	}
}

func TestExecStatusAndReviews(t *testing.T) {
	root := rootWithMint(t)
	if _, err := execstate.Init(root, "feat", "001", "sid-a", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := execstate.RecordReview(root, "feat", "001", "quality", "passed", "sid-a", &execstate.Provenance{Engine: "codex", Session: "reviewer"}); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	code, err := Run(root, []string{"status", "feat", "001"}, Flags{Session: "sid-a"}, &out, &bytes.Buffer{})
	if err != nil || code != 0 || out.String() != "running\n" {
		t.Fatalf("status code=%d err=%v out=%q", code, err, out.String())
	}
	out.Reset()
	code, err = Run(root, []string{"reviews", "feat", "001"}, Flags{Session: "sid-a"}, &out, &bytes.Buffer{})
	if err != nil || code != 0 {
		t.Fatalf("reviews code=%d err=%v", code, err)
	}
	var reviews map[string]execstate.Review
	if err := json.Unmarshal(out.Bytes(), &reviews); err != nil {
		t.Fatalf("parse reviews: %v", err)
	}
	if reviews["quality"].Verdict != "passed" {
		t.Fatalf("reviews = %#v", reviews)
	}
}

func TestExecRequiresMint(t *testing.T) {
	code, err := Run(t.TempDir(), []string{"init", "feat", "001"}, Flags{Session: "sid-a"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 1 || err == nil || !strings.Contains(err.Error(), "No mint session here") {
		t.Fatalf("code=%d err=%v", code, err)
	}
}

func TestExecUsesCapturedSession(t *testing.T) {
	root := rootWithMint(t)
	current := filepath.Join(root, ".mint", "sessions", ".current-session-id")
	if err := os.MkdirAll(filepath.Dir(current), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(current, []byte("captured\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, err := Run(root, []string{"init", "feat", "001"}, Flags{MakerEngine: "codex"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil || code != 0 {
		t.Fatalf("init code=%d err=%v", code, err)
	}
	state, ok := execstate.Read(root, "feat", "001", "captured")
	if !ok || state.Maker == nil || state.Maker.Session != "captured" {
		t.Fatalf("state = %#v ok=%v", state, ok)
	}
}

// A worker inits + records under one session, then a bare command (no --session,
// no pin) must still land on that owning session — not mint a fresh empty one.
func TestExecResolvesOwningSessionWithoutPin(t *testing.T) {
	root := rootWithMint(t)
	if _, err := execstate.Init(root, "feat", "001", "owner-sid", nil); err != nil {
		t.Fatal(err)
	}
	if code, err := Run(root, []string{"record-review", "feat", "001", "correctness", "passed"}, Flags{ByEngine: "codex", BySession: "reviewer"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil || code != 0 {
		t.Fatalf("record-review code=%d err=%v", code, err)
	}
	state, ok := execstate.Read(root, "feat", "001", "owner-sid")
	if !ok || state.Reviews["correctness"].Verdict != "passed" {
		t.Fatalf("review not recorded on owning session: state=%#v ok=%v", state, ok)
	}
}

func TestExecAmbiguousOwnersRequireSession(t *testing.T) {
	root := rootWithMint(t)
	if _, err := execstate.Init(root, "feat", "001", "sid-a", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := execstate.Init(root, "feat", "001", "sid-b", nil); err != nil {
		t.Fatal(err)
	}
	_, err := Run(root, []string{"record-review", "feat", "001", "correctness", "passed"}, Flags{}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--session") {
		t.Fatalf("expected ambiguity error, got %v", err)
	}
}

func rootWithMint(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".mint"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := session.WriteState(root, "sid-a", session.State{"task": "x"}); err != nil {
		t.Fatal(err)
	}
	return root
}

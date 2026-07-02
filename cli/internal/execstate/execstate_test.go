package execstate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testSession = "sess-exec"
	testSlug    = "feat"
)

func TestPathSessionIsolation(t *testing.T) {
	root := t.TempDir()
	a := Path(root, "feat-shared", "1.1", "session-aaa")
	b := Path(root, "feat-shared", "1.1", "session-bbb")
	if a == b {
		t.Fatal("different sessions produced same execution path")
	}
	if !containsPath(a, filepath.Join("tasks", "session-aaa", "feat-shared", "1.1")) {
		t.Fatalf("path %q does not include session segment", a)
	}
	if !containsPath(b, filepath.Join("tasks", "session-bbb", "feat-shared", "1.1")) {
		t.Fatalf("path %q does not include session segment", b)
	}
}

func TestInitExecStateMaker(t *testing.T) {
	root := t.TempDir()
	state, err := Init(root, testSlug, "001", testSession, &Maker{Engine: "claude", Session: "sX"})
	if err != nil {
		t.Fatal(err)
	}
	if state.Maker == nil || state.Maker.Engine != "claude" || state.Maker.Session != "sX" {
		t.Fatalf("maker = %#v", state.Maker)
	}
	raw := readRaw(t, root, "001")
	if raw["maker"].(map[string]any)["engine"] != "claude" {
		t.Fatalf("raw maker = %#v", raw["maker"])
	}

	noMaker, err := Init(root, testSlug, "002", testSession, nil)
	if err != nil {
		t.Fatal(err)
	}
	if noMaker.Maker != nil {
		t.Fatalf("maker should be omitted: %#v", noMaker.Maker)
	}
	raw = readRaw(t, root, "002")
	if _, ok := raw["maker"]; ok {
		t.Fatalf("raw maker key should be omitted: %#v", raw)
	}
	if raw["status"] != "running" || raw["commit"] != nil || raw["completedAt"] != nil {
		t.Fatalf("raw base shape = %#v", raw)
	}

	engineOnly, err := Init(root, testSlug, "003", testSession, &Maker{Engine: "codex", Session: "  "})
	if err != nil {
		t.Fatal(err)
	}
	if engineOnly.Maker == nil || engineOnly.Maker.Engine != "codex" || engineOnly.Maker.Session != "" {
		t.Fatalf("engine-only maker = %#v", engineOnly.Maker)
	}
	sessionOnly, err := Init(root, testSlug, "004", testSession, &Maker{Session: "sOnly"})
	if err != nil {
		t.Fatal(err)
	}
	if sessionOnly.Maker == nil || sessionOnly.Maker.Session != "sOnly" || sessionOnly.Maker.Engine != "" {
		t.Fatalf("session-only maker = %#v", sessionOnly.Maker)
	}
}

func TestRecordGateReviewAndStatus(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root, testSlug, "020", testSession, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordGate(root, testSlug, "020", "e2e", "pass", testSession); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordGate(root, testSlug, "020", "typecheck", "fail", testSession); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordGate(root, testSlug, "020", "tier", "full", testSession); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordGate(root, testSlug, "020", "lint", "green", testSession); err == nil {
		t.Fatal("invalid gate result should fail")
	}
	if _, err := RecordGate(root, testSlug, "020", "", "pass", testSession); err == nil {
		t.Fatal("empty gate label should fail")
	}
	if _, err := RecordReview(root, testSlug, "020", "security", "passed", testSession); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordReview(root, testSlug, "020", "quality", "failed", testSession); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordReview(root, testSlug, "020", "security", "green", testSession); err == nil {
		t.Fatal("invalid review verdict should fail")
	}
	commit := "abc123"
	state, err := SetStatus(root, testSlug, "020", "passed", testSession, &commit)
	if err != nil {
		t.Fatal(err)
	}
	if state.Gates["e2e"] != "pass" || state.Gates["typecheck"] != "fail" || state.Gates["tier"] != "full" {
		t.Fatalf("gates = %#v", state.Gates)
	}
	if state.Reviews["security"] != "passed" || state.Reviews["quality"] != "failed" {
		t.Fatalf("reviews = %#v", state.Reviews)
	}
	if state.CompletedAt == nil || state.Commit == nil || *state.Commit != "abc123" {
		t.Fatalf("status state = %#v", state)
	}
	if _, err := SetStatus(root, testSlug, "020", "done", testSession, nil); err == nil {
		t.Fatal("invalid status should fail")
	}
}

func TestReadAndSessionIsolation(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root, "feat-shared", "1.1", "session-aaa", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(root, "feat-shared", "1.1", "session-bbb", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordGate(root, "feat-shared", "1.1", "tests", "pass", "session-aaa"); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordGate(root, "feat-shared", "1.1", "tests", "fail", "session-bbb"); err != nil {
		t.Fatal(err)
	}
	a, ok := Read(root, "feat-shared", "1.1", "session-aaa")
	if !ok || a.Gates["tests"] != "pass" {
		t.Fatalf("session a = %#v ok=%v", a, ok)
	}
	b, ok := Read(root, "feat-shared", "1.1", "session-bbb")
	if !ok || b.Gates["tests"] != "fail" {
		t.Fatalf("session b = %#v ok=%v", b, ok)
	}
	if _, ok := Read(root, "feat-shared", "1.1", "captured-current"); ok {
		t.Fatal("explicit session path should not write captured-current")
	}
}

func readRaw(t *testing.T, root, specID string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(Path(root, testSlug, specID, testSession))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	return raw
}

func containsPath(path, sub string) bool {
	return filepath.ToSlash(path) != "" && filepath.ToSlash(sub) != "" &&
		(len(path) >= len(sub)) && (filepath.ToSlash(path) == filepath.ToSlash(sub) ||
		strings.Contains(filepath.ToSlash(path), filepath.ToSlash(sub)))
}

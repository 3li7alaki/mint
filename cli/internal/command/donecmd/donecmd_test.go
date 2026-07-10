package donecmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"mint/internal/execstate"
	"mint/internal/notelist"
	"mint/internal/session"
)

func TestDoneProvenanceLessUnitFailsClause2AndRecordsDod(t *testing.T) {
	root := newRepo(t)
	writeFile(t, filepath.Join(root, "app.js"), "x\n")
	writeSpec(t, root, "sid-a", "feat", "001", []string{"app.js"}, "")
	writeVerdict(t, root, "feat", "001", map[string]any{"accepted": true, "byEngine": "codex", "bySession": "s2"})
	var out bytes.Buffer
	code, err := Run(root, []string{"feat", "001"}, Flags{Session: "sid-a", Terminal: "done-verified", JSON: true}, &out, &bytes.Buffer{})
	if err != nil || code != 1 {
		t.Fatalf("Run code=%d err=%v out=%q", code, err, out.String())
	}
	var result map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("parse result: %v\n%s", err, out.String())
	}
	if result["pass"] != false {
		t.Fatalf("result = %#v", result)
	}
	failed := result["failed"].([]any)
	if len(failed) != 1 || failed[0].(float64) != 2 {
		t.Fatalf("failed = %#v", failed)
	}
	state, ok := execstate.Read(root, "feat", "001", "sid-a")
	if !ok || state.Reviews["dod"] != "failed" {
		t.Fatalf("state = %#v ok=%v", state, ok)
	}
}

// Regression: done with no --session and no pin must resolve to the session that
// owns the unit's execution.json (where reviews were recorded), not mint a fresh
// empty one that leaves clause-1 unsatisfiable. This was the worktree/unpinned bug.
func TestDoneResolvesOwningSessionWithoutFlag(t *testing.T) {
	root := newRepo(t)
	writeFile(t, filepath.Join(root, "app.js"), "x\n")
	writeSpec(t, root, "owner-sid", "feat", "010", []string{"app.js"}, "")
	writeVerdict(t, root, "feat", "010", map[string]any{"accepted": true, "byEngine": "codex", "bySession": "s2"})
	if _, err := execstate.Init(root, "feat", "010", "owner-sid", &execstate.Maker{Engine: "claude", Session: "owner-sid"}); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	code, err := Run(root, []string{"feat", "010"}, Flags{Terminal: "done-verified", JSON: true}, &out, &bytes.Buffer{})
	if err != nil || code != 0 {
		t.Fatalf("Run code=%d err=%v out=%q", code, err, out.String())
	}
	state, ok := execstate.Read(root, "feat", "010", "owner-sid")
	if !ok || state.Reviews["dod"] != "passed" {
		t.Fatalf("dod not recorded on owning session: state=%#v ok=%v", state, ok)
	}
}

func TestDonePassesWithRecordedDifferentEngineMaker(t *testing.T) {
	root := newRepo(t)
	writeFile(t, filepath.Join(root, "app.js"), "x\n")
	writeSpec(t, root, "sid-a", "feat", "002", []string{"app.js"}, "")
	writeVerdict(t, root, "feat", "002", map[string]any{"accepted": true, "byEngine": "codex", "bySession": "s2"})
	if _, err := execstate.Init(root, "feat", "002", "sid-a", &execstate.Maker{Engine: "claude", Session: "sid-a"}); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	code, err := Run(root, []string{"feat", "002"}, Flags{Session: "sid-a", Terminal: "done-verified", JSON: true}, &out, &bytes.Buffer{})
	if err != nil || code != 0 {
		t.Fatalf("Run code=%d err=%v out=%q", code, err, out.String())
	}
	var result map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if result["pass"] != true {
		t.Fatalf("result = %#v", result)
	}
	state, _ := execstate.Read(root, "feat", "002", "sid-a")
	if state.Reviews["dod"] != "passed" {
		t.Fatalf("state = %#v", state)
	}
}

func TestDoneRecordedMakerOverridesHostileFlags(t *testing.T) {
	root := newRepo(t)
	writeFile(t, filepath.Join(root, "app.js"), "x\n")
	writeSpec(t, root, "sid-a", "feat", "003", []string{"app.js"}, "")
	writeVerdict(t, root, "feat", "003", map[string]any{"accepted": true, "byEngine": "codex", "bySession": "s2"})
	if _, err := execstate.Init(root, "feat", "003", "sid-a", &execstate.Maker{Engine: "claude", Session: "sid-a"}); err != nil {
		t.Fatal(err)
	}
	code, err := Run(root, []string{"feat", "003"}, Flags{Session: "sid-a", Terminal: "done-verified", MakerEngine: "codex", MakerSession: "s2", JSON: true}, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil || code != 0 {
		t.Fatalf("recorded maker should win: code=%d err=%v", code, err)
	}
}

func TestDoneMissingVerdictFailsClause1(t *testing.T) {
	root := newRepo(t)
	writeFile(t, filepath.Join(root, "app.js"), "x\n")
	writeSpec(t, root, "sid-a", "feat", "004", []string{"app.js"}, "")
	code, err := Run(root, []string{"feat", "004"}, Flags{Session: "sid-a", Terminal: "done-verified", JSON: true}, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil || code != 1 {
		t.Fatalf("Run code=%d err=%v", code, err)
	}
}

func TestDoneInvalidBaseFailsClosed(t *testing.T) {
	root := newRepo(t)
	writeFile(t, filepath.Join(root, "app.js"), "x\n")
	writeSpec(t, root, "sid-a", "feat", "005", []string{"app.js"}, "")
	writeVerdict(t, root, "feat", "005", map[string]any{"accepted": true, "byEngine": "codex", "bySession": "s2"})
	var stderr bytes.Buffer
	code, err := Run(root, []string{"feat", "005"}, Flags{Session: "sid-a", Terminal: "done-verified", Base: "-s", JSON: true}, &bytes.Buffer{}, &stderr)
	if err != nil || code != 1 || !strings.Contains(stderr.String(), "Invalid --base ref") {
		t.Fatalf("code=%d err=%v stderr=%q", code, err, stderr.String())
	}
}

func TestDoneExplicitSpecPath(t *testing.T) {
	root := newRepo(t)
	writeFile(t, filepath.Join(root, "app.js"), "x\n")
	spec := filepath.Join(root, "custom-spec.xml")
	writeFile(t, spec, specXML("009", []string{"app.js"}, passGateXML()))
	writeVerdict(t, root, "feat", "009", map[string]any{"accepted": true, "byEngine": "codex", "bySession": "s2"})
	code, err := Run(root, []string{"feat", "009"}, Flags{Session: "sid-a", Spec: "custom-spec.xml", Terminal: "done-verified", MakerEngine: "claude", MakerSession: "s1", JSON: true}, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil || code != 0 {
		t.Fatalf("Run code=%d err=%v", code, err)
	}
}

// runFailingDone sets up a provenance-less unit (fails floor clause 2) and runs done once.
func runFailingDone(t *testing.T, root, slug, specID string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "app.js"), "x\n")
	writeSpec(t, root, "sid-a", slug, specID, []string{"app.js"}, "")
	writeVerdict(t, root, slug, specID, map[string]any{"accepted": true, "byEngine": "codex", "bySession": "s2"})
	code, err := Run(root, []string{slug, specID}, Flags{Session: "sid-a", Terminal: "done-verified", JSON: true}, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil || code != 1 {
		t.Fatalf("expected failing done: code=%d err=%v", code, err)
	}
}

func noteFor(t *testing.T, root, topic string) (notelist.Note, bool) {
	t.Helper()
	notes, err := notelist.Read(root)
	if err != nil {
		t.Fatalf("notelist.Read: %v", err)
	}
	for _, n := range notes {
		if n.Topic == topic {
			return n, true
		}
	}
	return notelist.Note{}, false
}

// TestDoneFailWritesAccumulatingNote covers the note-wire acceptance: a failing
// done appends one spec-keyed note naming the failed clause; retries accumulate
// under the same topic; a passing done writes nothing.
func TestDoneFailWritesAccumulatingNote(t *testing.T) {
	root := newRepo(t)
	topic := failNoteTopic("feat", "010")

	// (a) fail once -> exactly one note, Entries==1, body names the failed clause + reason.
	runFailingDone(t, root, "feat", "010")
	n, ok := noteFor(t, root, topic)
	if !ok {
		t.Fatalf("no note under topic %q", topic)
	}
	if n.Entries != 1 {
		t.Fatalf("Entries = %d, want 1", n.Entries)
	}
	body, err := notelist.Body(root, topic)
	if err != nil {
		t.Fatalf("Body: %v", err)
	}
	// (d) text names the specific failed clause (2) with a Reason substring, not generic.
	if !strings.Contains(body, "clause 2") {
		t.Fatalf("body does not name failed clause 2:\n%s", body)
	}
	if strings.Contains(body, "done failed") || !strings.Contains(body, "failed clauses:") {
		t.Fatalf("body is generic, expected clause detail:\n%s", body)
	}

	// (b) fail again on the SAME spec -> same topic accumulates (Entries==2), no new topic.
	runFailingDone(t, root, "feat", "010")
	notes, _ := notelist.Read(root)
	count := 0
	for _, x := range notes {
		if x.Topic == topic {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one topic row, got %d", count)
	}
	n2, _ := noteFor(t, root, topic)
	if n2.Entries != 2 {
		t.Fatalf("after 2 fails Entries = %d, want 2", n2.Entries)
	}

	// (c) a passing done writes no note.
	writeFile(t, filepath.Join(root, "app.js"), "x\n")
	writeSpec(t, root, "sid-a", "feat", "011", []string{"app.js"}, "")
	writeVerdict(t, root, "feat", "011", map[string]any{"accepted": true, "byEngine": "codex", "bySession": "s2"})
	if _, err := execstate.Init(root, "feat", "011", "sid-a", &execstate.Maker{Engine: "claude", Session: "sid-a"}); err != nil {
		t.Fatal(err)
	}
	code, err := Run(root, []string{"feat", "011"}, Flags{Session: "sid-a", Terminal: "done-verified", JSON: true}, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil || code != 0 {
		t.Fatalf("expected passing done: code=%d err=%v", code, err)
	}
	if _, ok := noteFor(t, root, failNoteTopic("feat", "011")); ok {
		t.Fatalf("passing done should write no note")
	}
}

func newRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".mint", "sessions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := session.WriteState(root, "sid-a", session.State{"gates": map[string]string{"tests": "true"}}); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, ".gitignore"), ".mint/\ncustom-spec.xml\n")
	gitRun(t, root, "init", "-q")
	gitRun(t, root, "config", "user.email", "t@t.t")
	gitRun(t, root, "config", "user.name", "t")
	writeFile(t, filepath.Join(root, "README.md"), "base\n")
	gitRun(t, root, "add", "-A")
	gitRun(t, root, "commit", "-q", "-m", "base")
	return root
}

func writeSpec(t *testing.T, root, sessionID, slug, specID string, canModify []string, extra string) {
	t.Helper()
	path := filepath.Join(root, ".mint", "tasks", sessionID, slug, specID+"-"+slug+".xml")
	writeFile(t, path, specXML(specID, canModify, extra+passGateXML()))
}

func specXML(specID string, canModify []string, extra string) string {
	paths := strings.Builder{}
	for _, p := range canModify {
		paths.WriteString("<path>")
		paths.WriteString(p)
		paths.WriteString("</path>")
	}
	return "<task><id>" + specID + "</id><scope><can-modify>" + paths.String() + "</can-modify></scope><acceptance>WHEN x, THE app SHALL y.</acceptance>" + extra + "</task>"
}

func passGateXML() string {
	return "<gates>\n  tests: true\n</gates>"
}

func writeVerdict(t *testing.T, root, slug, specID string, obj map[string]any) {
	t.Helper()
	b, err := json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, ".mint", "verdicts", slug+"-"+specID+".json"), string(b))
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func gitRun(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

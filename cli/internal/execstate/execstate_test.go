package execstate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	if state.Maker == nil || state.Maker.Engine != "claude" || state.Maker.Vendor != "anthropic" || state.Maker.Model != "claude" || state.Maker.Locality != "remote" || state.Maker.Session != "sX" {
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

func TestInitExecStateConfigurableMakerProvenanceFailsClosedOnUnprovenLocality(t *testing.T) {
	root := t.TempDir()
	state, err := Init(root, testSlug, "005", testSession, &Maker{
		Engine: "opencode", Vendor: "openai", Model: "gpt", Locality: "remote", Session: "maker-session",
	})
	if err != nil {
		t.Fatal(err)
	}
	if state.Maker == nil || state.Maker.Vendor != "openai" || state.Maker.Model != "gpt" || state.Maker.Locality != "remote" {
		t.Fatalf("recorded maker = %#v", state.Maker)
	}
	if _, err := Init(root, testSlug, "006", testSession, &Maker{
		Engine: "opencode", Vendor: "zai", Model: "glm", Locality: "local", Session: "maker-session",
	}); err == nil || !strings.Contains(err.Error(), "cannot prove local execution") {
		t.Fatalf("caller-asserted local configurable engine should fail closed: %v", err)
	}
}

func TestInitCannotOverwriteRecordedMakerProvenance(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root, testSlug, "007", testSession, &Maker{Engine: "codex", Session: "maker"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(root, testSlug, "007", testSession, &Maker{Engine: "claude", Session: "checker"}); err == nil || !strings.Contains(err.Error(), "write-once") {
		t.Fatalf("checker re-initialized maker provenance: %v", err)
	}
	state, ok := Read(root, testSlug, "007", testSession)
	if !ok || state.Maker == nil || state.Maker.Engine != "codex" || state.Maker.Vendor != "openai" || state.Maker.Session != "maker" {
		t.Fatalf("recorded maker was overwritten: %#v", state)
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
	reviewer := &Provenance{Engine: "codex", Session: "review-session"}
	if _, err := RecordReview(root, testSlug, "020", "security", "passed", testSession, reviewer); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordReview(root, testSlug, "020", "quality", "failed", testSession, reviewer); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordReview(root, testSlug, "020", "security", "green", testSession, reviewer); err == nil {
		t.Fatal("invalid review verdict should fail")
	}
	if _, err := RecordReview(root, testSlug, "020", "quality", "passed", testSession, nil); err == nil {
		t.Fatal("review without provenance should fail at record time")
	}
	if _, err := RecordReview(root, testSlug, "020", "quality", "passed", testSession, &Provenance{Engine: "opencode", Session: "reviewer"}); err == nil {
		t.Fatal("configurable reviewer without vendor/model/locality should fail")
	}
	if _, err := RecordReview(root, testSlug, "020", "quality", "passed", testSession, &Provenance{Engine: "codex", Vendor: "anthropic", Session: "reviewer"}); err == nil {
		t.Fatal("reviewer provenance conflicting with fixed registry should fail")
	}
	commit := "abc123"
	state, err := SetStatus(root, testSlug, "020", "passed", testSession, &commit)
	if err != nil {
		t.Fatal(err)
	}
	if state.Gates["e2e"] != "pass" || state.Gates["typecheck"] != "fail" || state.Gates["tier"] != "full" {
		t.Fatalf("gates = %#v", state.Gates)
	}
	if state.Reviews["security"].Verdict != "passed" || state.Reviews["quality"].Verdict != "failed" {
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

func TestOwningSessions(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root, "feat", "001", "sid-a", nil); err != nil {
		t.Fatal(err)
	}
	if got := OwningSessions(root, "feat", "001"); len(got) != 1 || got[0] != "sid-a" {
		t.Fatalf("unique owner = %v, want [sid-a]", got)
	}
	if got := OwningSessions(root, "feat", "999"); len(got) != 0 {
		t.Fatalf("no owner = %v, want empty", got)
	}
	if _, err := Init(root, "feat", "001", "sid-b", nil); err != nil {
		t.Fatal(err)
	}
	if got := OwningSessions(root, "feat", "001"); len(got) != 2 || got[0] != "sid-a" || got[1] != "sid-b" {
		t.Fatalf("multi owner = %v, want sorted [sid-a sid-b]", got)
	}
}

// Adversarial: glob metacharacters and path traversal in slug/specID must never
// mis-match, error into a wrong zero-owner fallthrough, or escape .mint/tasks/.
func TestOwningSessionsRejectsHostileIdentifiers(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root, "feat", "001", "sid-a", nil); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"fe[a", "fe*t", "fe?t", "..", ".", "", "a/b"} {
		if got := OwningSessions(root, id, "001"); len(got) != 0 {
			t.Fatalf("hostile slug %q returned owners %v, want none", id, got)
		}
		if got := OwningSessions(root, "feat", id); len(got) != 0 {
			t.Fatalf("hostile specID %q returned owners %v, want none", id, got)
		}
	}
	// The literal unit is still resolvable — hostile inputs didn't corrupt state.
	if got := OwningSessions(root, "feat", "001"); len(got) != 1 {
		t.Fatalf("literal owner lost after hostile probes: %v", got)
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

func TestInitRejectsTraversalIdentifiers(t *testing.T) {
	root := t.TempDir()
	for _, id := range []string{"..", ".", "", "a/b", "../escape"} {
		if _, err := Init(root, id, "001", testSession, nil); err == nil {
			t.Fatalf("Init accepted hostile slug %q", id)
		}
		if _, err := Init(root, testSlug, id, testSession, nil); err == nil {
			t.Fatalf("Init accepted hostile spec-id %q", id)
		}
	}
	// A traversal identifier must not have written anything under .mint/tasks.
	if entries, _ := os.ReadDir(filepath.Join(root, ".mint", "tasks", testSession)); len(entries) != 0 {
		t.Fatalf("hostile identifier wrote state: %#v", entries)
	}
}

func TestMutatorsRejectTraversalIdentifiers(t *testing.T) {
	root := t.TempDir()
	// Path joins sessionID/slug/specID, so all three are traversal-controlling.
	// No mutation entry point may accept a hostile value in ANY of the three
	// positions, and Init must refuse them too.
	hostile := []string{"..", ".", "", "a/b", "../../escape"}
	for _, bad := range hostile {
		for _, pos := range []struct{ slug, specID, session string }{
			{bad, "001", testSession},
			{testSlug, bad, testSession},
			{testSlug, "001", bad},
		} {
			if _, err := Init(root, pos.slug, pos.specID, pos.session, &Maker{Engine: "codex", Session: "s"}); err == nil {
				t.Fatalf("Init accepted hostile identifier %q/%q/%q", pos.slug, pos.specID, pos.session)
			}
			if _, err := RecordGate(root, pos.slug, pos.specID, "tests", "pass", pos.session); err == nil {
				t.Fatalf("RecordGate accepted hostile identifier %q/%q/%q", pos.slug, pos.specID, pos.session)
			}
			if _, err := RecordReview(root, pos.slug, pos.specID, "security", "passed", pos.session, &Provenance{Engine: "codex", Session: "s"}); err == nil {
				t.Fatalf("RecordReview accepted hostile identifier %q/%q/%q", pos.slug, pos.specID, pos.session)
			}
			if _, err := SetStatus(root, pos.slug, pos.specID, "passed", pos.session, nil); err == nil {
				t.Fatalf("SetStatus accepted hostile identifier %q/%q/%q", pos.slug, pos.specID, pos.session)
			}
		}
	}
	// A traversal target one level above root must never be created.
	if _, err := os.Stat(filepath.Join(root, "..", "escape")); err == nil {
		t.Fatal("hostile identifier escaped the repo root")
	}
}

func TestConcurrentMakerUpgradeStaysWriteOnce(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root, testSlug, "020", testSession, nil); err != nil {
		t.Fatal(err)
	}
	// Two concurrent upgrades race to fill the maker-less placeholder. Exactly
	// one must win; the other must see write-once, never last-writer-win.
	engines := []string{"codex", "claude"}
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := range engines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = Init(root, testSlug, "020", testSession, &Maker{Engine: engines[i], Session: "s"})
		}(i)
	}
	wg.Wait()
	wins, writeOnce := 0, 0
	for _, err := range errs {
		if err == nil {
			wins++
		} else if strings.Contains(err.Error(), "write-once") {
			writeOnce++
		} else {
			t.Fatalf("unexpected upgrade error: %v", err)
		}
	}
	if wins != 1 || writeOnce != 1 {
		t.Fatalf("concurrent upgrade not write-once: wins=%d writeOnce=%d", wins, writeOnce)
	}
	state, _ := Read(root, testSlug, "020", testSession)
	if state.Maker == nil || state.Maker.Engine == "" {
		t.Fatalf("no maker recorded after race: %#v", state)
	}
}

func TestInitUpgradesMakerLessPlaceholder(t *testing.T) {
	root := t.TempDir()
	// verify/done create a maker-less placeholder before exec init runs.
	if _, err := Init(root, testSlug, "010", testSession, nil); err != nil {
		t.Fatal(err)
	}
	// A later init records the real maker instead of failing write-once.
	state, err := Init(root, testSlug, "010", testSession, &Maker{Engine: "codex", Session: "maker"})
	if err != nil {
		t.Fatalf("maker-less placeholder was not upgradable: %v", err)
	}
	if state.Maker == nil || state.Maker.Engine != "codex" || state.Maker.Vendor != "openai" {
		t.Fatalf("maker not recorded on upgrade: %#v", state.Maker)
	}
	// Once a real maker is recorded, it is write-once again.
	if _, err := Init(root, testSlug, "010", testSession, &Maker{Engine: "claude", Session: "checker"}); err == nil || !strings.Contains(err.Error(), "write-once") {
		t.Fatalf("recorded maker was not write-once after upgrade: %v", err)
	}
}

func containsPath(path, sub string) bool {
	return filepath.ToSlash(path) != "" && filepath.ToSlash(sub) != "" &&
		(len(path) >= len(sub)) && (filepath.ToSlash(path) == filepath.ToSlash(sub) ||
		strings.Contains(filepath.ToSlash(path), filepath.ToSlash(sub)))
}

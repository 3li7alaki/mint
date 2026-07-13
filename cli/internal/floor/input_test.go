package floor

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"mint/internal/execstate"
)

func TestReadVerdict(t *testing.T) {
	dir := t.TempDir()
	valid := writeJSON(t, filepath.Join(dir, "valid.json"), map[string]any{
		"accepted":  true,
		"byEngine":  "codex",
		"bySession": "s2",
		"criteria":  []any{map[string]any{"id": "c1", "met": true}},
	})
	v := ReadVerdict(valid)
	if v == nil || v["accepted"] != true || v["byEngine"] != "codex" || v["bySession"] != "s2" {
		t.Fatalf("ReadVerdict(valid) = %#v", v)
	}
	if _, ok := v["criteria"].([]any); !ok {
		t.Fatalf("criteria should be carried through untouched: %#v", v)
	}

	if ReadVerdict(filepath.Join(dir, "missing.json")) != nil {
		t.Fatal("missing verdict should return nil")
	}
	if ReadVerdict("") != nil {
		t.Fatal("empty verdict path should return nil")
	}
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte("{ not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ReadVerdict(bad) != nil {
		t.Fatal("malformed JSON should return nil")
	}
	for name, obj := range map[string]any{
		"array.json":     []any{1, 2},
		"missing.json":   map[string]any{"accepted": true, "byEngine": "codex"},
		"wrongbool.json": map[string]any{"accepted": "yes", "byEngine": "codex", "bySession": "s"},
		"wrongeng.json":  map[string]any{"accepted": true, "byEngine": 5, "bySession": "s"},
		"wrongsess.json": map[string]any{"accepted": true, "byEngine": "codex", "bySession": nil},
	} {
		if got := ReadVerdict(writeJSON(t, filepath.Join(dir, name), obj)); got != nil {
			t.Fatalf("ReadVerdict(%s) = %#v, want nil", name, got)
		}
	}
	rejected := ReadVerdict(writeJSON(t, filepath.Join(dir, "rejected.json"), map[string]any{
		"accepted": false, "byEngine": "codex", "bySession": "s",
	}))
	if rejected == nil || rejected["accepted"] != false {
		t.Fatalf("accepted:false verdict is well-shaped and should be returned: %#v", rejected)
	}
}

func TestBuildInputCollectsFacts(t *testing.T) {
	root := newGitRepo(t)
	writeFile(t, filepath.Join(root, "app.js"), "console.log(1)\n")
	specPath := writeSpecFile(t, root, "sess-x", "feat", "001", []string{"app.js"}, passGateXML())
	verdictPath := writeJSON(t, filepath.Join(root, "verdict.json"), map[string]any{
		"accepted": true, "byEngine": "codex", "bySession": "s2",
	})
	if _, err := execstate.Init(root, "feat", "001", "sess-x", &execstate.Maker{Engine: "claude", Session: "s1"}); err != nil {
		t.Fatal(err)
	}

	input, err := BuildInput(root, BuildOptions{
		SpecPath:      specPath,
		Slug:          "feat",
		SpecID:        "001",
		VerdictPath:   verdictPath,
		TerminalState: "done-verified",
		SessionID:     "sess-x",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(input.ChangedFiles, "app.js") {
		t.Fatalf("changed files = %#v, want app.js", input.ChangedFiles)
	}
	if input.Patch != "" {
		t.Fatalf("untracked files should not appear in git diff patch, got %q", input.Patch)
	}
	if input.Gates == nil || !input.Gates.OK {
		t.Fatalf("gates = %#v", input.Gates)
	}
	if input.Verdict == nil || input.Verdict["byEngine"] != "codex" {
		t.Fatalf("verdict = %#v", input.Verdict)
	}
	if input.MakerEngine != "claude" || input.MakerVendor != "anthropic" || input.MakerModel != "claude" || input.MakerLocality != "remote" || input.MakerSession != "s1" || input.TerminalState != "done-verified" {
		t.Fatalf("pass-through facts wrong: %#v", input)
	}
	if len(input.RequiredReviews) != 0 || len(input.Reviews) != 0 {
		t.Fatalf("unexpected reviews: required=%#v attached=%#v", input.RequiredReviews, input.Reviews)
	}
}

func TestBuildInputMakerProvenanceComesOnlyFromExecutionState(t *testing.T) {
	root := newGitRepo(t)
	specPath := writeSpecFile(t, root, "sess-maker", "feat", "maker", []string{"README.md"}, passGateXML())
	verdictPath := writeJSON(t, filepath.Join(root, "verdict.json"), map[string]any{
		"accepted": true, "byEngine": "opencode", "bySession": "checker",
		"byVendor": "openai", "byModel": "gpt", "byLocality": "remote",
		"makerVendor": "anthropic", "makerModel": "claude", "makerLocality": "local",
	})
	if _, err := execstate.Init(root, "feat", "maker", "sess-maker", &execstate.Maker{Engine: "codex", Session: "maker"}); err != nil {
		t.Fatal(err)
	}
	input, err := BuildInput(root, BuildOptions{
		SpecPath: specPath, Slug: "feat", SpecID: "maker", SessionID: "sess-maker", VerdictPath: verdictPath,
		MakerEngine: "claude", MakerSession: "forged-option",
	})
	if err != nil {
		t.Fatal(err)
	}
	if input.MakerEngine != "codex" || input.MakerVendor != "openai" || input.MakerModel != "gpt" || input.MakerLocality != "remote" || input.MakerSession != "maker" {
		t.Fatalf("maker provenance was not loaded from execution state: %#v", input)
	}
	if c2 := clauseResult(t, Enforce(input), 2); c2.Pass || !strings.Contains(reason(c2), "different chassis do not establish independence") {
		t.Fatalf("checker forged maker identity through verdict/options: %#v", c2)
	}
}

func TestBuildInputPatchAndCleanTree(t *testing.T) {
	root := newGitRepo(t)
	specPath := writeSpecFile(t, root, "sess-p", "feat", "002", []string{"README.md"}, passGateXML())
	writeFile(t, filepath.Join(root, "README.md"), "base\nchanged\n")
	input, err := BuildInput(root, BuildOptions{SpecPath: specPath, Slug: "feat", SpecID: "002", SessionID: "sess-p"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(input.Patch, "+++ b/README.md") || !strings.Contains(input.Patch, "+changed") {
		t.Fatalf("patch = %q", input.Patch)
	}

	git(t, root, "checkout", "--", "README.md")
	clean, err := BuildInput(root, BuildOptions{SpecPath: specPath, Slug: "feat", SpecID: "002", SessionID: "sess-p"})
	if err != nil {
		t.Fatal(err)
	}
	if len(clean.ChangedFiles) != 0 || clean.Patch != "" {
		t.Fatalf("clean facts = changed=%#v patch=%q", clean.ChangedFiles, clean.Patch)
	}
}

func TestBuildInputBaseScopingAndSecurity(t *testing.T) {
	root := newGitRepo(t)
	base := strings.TrimSpace(gitOut(t, root, "rev-parse", "HEAD"))
	writeFile(t, filepath.Join(root, "later.js"), "later\n")
	git(t, root, "add", "later.js")
	git(t, root, "commit", "-q", "-m", "later")
	specPath := writeSpecFile(t, root, "sess-b", "feat", "003", []string{"later.js"}, "")

	scoped, err := BuildInput(root, BuildOptions{SpecPath: specPath, Slug: "feat", SpecID: "003", SessionID: "sess-b", Base: base})
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(scoped.ChangedFiles, "later.js") || !strings.Contains(scoped.Patch, "later.js") {
		t.Fatalf("base-scoped facts = changed=%#v patch=%q", scoped.ChangedFiles, scoped.Patch)
	}
	head, err := BuildInput(root, BuildOptions{SpecPath: specPath, Slug: "feat", SpecID: "003", SessionID: "sess-b"})
	if err != nil {
		t.Fatal(err)
	}
	if containsString(head.ChangedFiles, "later.js") {
		t.Fatalf("HEAD-scoped facts should be clean for later.js: %#v", head.ChangedFiles)
	}

	for _, hostile := range []string{"HEAD; touch pwned", "$(touch pwned2)", "-s", "--output=written.txt"} {
		_, err := BuildInput(root, BuildOptions{SpecPath: specPath, Slug: "feat", SpecID: "003", SessionID: "sess-b", Base: hostile})
		if !IsInvalidBaseError(err) {
			t.Fatalf("base %q error = %v, want invalid base", hostile, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "pwned")); !os.IsNotExist(err) {
		t.Fatalf("hostile base created side-effect file")
	}
	_, err = BuildInput(root, BuildOptions{SpecPath: specPath, Slug: "feat", SpecID: "003", SessionID: "sess-b", Base: "nonexistent-ref-xyz"})
	if !IsUnresolvableBaseError(err) {
		t.Fatalf("garbage base error = %v, want unresolvable", err)
	}
	blob := strings.TrimSpace(gitOut(t, root, "hash-object", "-w", "README.md"))
	_, err = BuildInput(root, BuildOptions{SpecPath: specPath, Slug: "feat", SpecID: "003", SessionID: "sess-b", Base: blob})
	if !IsUnresolvableBaseError(err) {
		t.Fatalf("blob base error = %v, want unresolvable", err)
	}
}

func TestBuildInputReviews(t *testing.T) {
	root := newGitRepo(t)
	specPath := writeSpecFile(t, root, "sess-r", "feat", "004", []string{"README.md"}, "<reviews>security, quality</reviews>"+passGateXML())
	writeJSON(t, filepath.Join(root, ".mint", "sessions", "sess-r.json"), map[string]any{
		"reviews": []string{"performance"},
	})
	writeJSON(t, filepath.Join(root, ".mint", "tasks", "sess-r", "feat", "004", "execution.json"), map[string]any{
		"reviews": map[string]string{"security": "passed", "quality": "failed"},
	})
	input, err := BuildInput(root, BuildOptions{SpecPath: specPath, Slug: "feat", SpecID: "004", SessionID: "sess-r"})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(input.RequiredReviews, ","); got != "security,quality" {
		t.Fatalf("required reviews = %#v", input.RequiredReviews)
	}
	if input.Reviews["security"].Verdict != "passed" || input.Reviews["quality"].Verdict != "failed" {
		t.Fatalf("attached reviews = %#v", input.Reviews)
	}
	if input.Reviews["security"].Provenance.Engine != "" {
		t.Fatalf("legacy string review must remain unattributed: %#v", input.Reviews["security"])
	}

	sessionOnlySpec := writeSpecFile(t, root, "sess-r", "feat", "005", []string{"README.md"}, passGateXML())
	sessionOnly, err := BuildInput(root, BuildOptions{SpecPath: sessionOnlySpec, Slug: "feat", SpecID: "005", SessionID: "sess-r"})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(sessionOnly.RequiredReviews, ","); got != "performance" {
		t.Fatalf("session required reviews = %#v", sessionOnly.RequiredReviews)
	}
}

func newGitRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".mint"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, ".gitignore"), ".mint/\nverdict.json\n")
	git(t, root, "init", "-q")
	git(t, root, "config", "user.email", "t@t.t")
	git(t, root, "config", "user.name", "t")
	writeFile(t, filepath.Join(root, "README.md"), "base\n")
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", "base")
	return root
}

func writeSpecFile(t *testing.T, root, sessionID, slug, specID string, canModify []string, extra string) string {
	t.Helper()
	var paths strings.Builder
	for _, p := range canModify {
		paths.WriteString("<path>")
		paths.WriteString(p)
		paths.WriteString("</path>")
	}
	p := filepath.Join(root, ".mint", "tasks", sessionID, slug, specID+".xml")
	writeFile(t, p, "<task><scope><can-modify>"+paths.String()+"</can-modify></scope>"+extra+"</task>")
	return p
}

func passGateXML() string {
	return "<gates>\n  tests: true\n</gates>"
}

func writeJSON(t *testing.T, path string, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, string(b))
	return path
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

func git(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func gitOut(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("git %v failed: %v\n%s", args, err, ee.Stderr)
		}
		t.Fatalf("git %v failed: %v", args, err)
	}
	return string(out)
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

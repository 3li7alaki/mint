package sessioncmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mint/internal/session"
)

func TestSessionNewCreatesState(t *testing.T) {
	root := t.TempDir()
	var out bytes.Buffer
	code, err := Run(root, []string{"new"}, Flags{Session: "sid-a", Task: "ship it", Mode: "full", NoCommit: true}, &out)
	if err != nil || code != 0 {
		t.Fatalf("Run new code=%d err=%v", code, err)
	}
	var result map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if result["sessionId"] != "sid-a" {
		t.Fatalf("result = %#v", result)
	}
	state, ok := session.ReadState(root, "sid-a")
	if !ok {
		t.Fatal("state not written")
	}
	if state["task"] != "ship it" || state["mode"] != "full" || state["autoCommitOverride"] != false {
		t.Fatalf("state = %#v", state)
	}
	if content, err := os.ReadFile(filepath.Join(root, ".gitignore")); err != nil || !strings.Contains(string(content), ".mint/sessions/") {
		t.Fatalf("gitignore not ensured: %q %v", content, err)
	}
}

func TestSessionListShowResumeEnd(t *testing.T) {
	root := t.TempDir()
	if err := session.WriteState(root, "sid-a", session.State{"task": "task a", "mode": "quick"}); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	code, err := Run(root, []string{"list"}, Flags{}, &out)
	if err != nil || code != 0 || !strings.Contains(out.String(), "sid-a  quick  task a") {
		t.Fatalf("list code=%d err=%v out=%q", code, err, out.String())
	}
	out.Reset()
	code, err = Run(root, []string{"show", "sid-a"}, Flags{}, &out)
	if err != nil || code != 0 || !strings.Contains(out.String(), `"task": "task a"`) {
		t.Fatalf("show code=%d err=%v out=%q", code, err, out.String())
	}
	out.Reset()
	code, err = Run(root, []string{"resume", "sid-a"}, Flags{}, &out)
	if err != nil || code != 0 || !strings.Contains(out.String(), `"mode": "quick"`) {
		t.Fatalf("resume code=%d err=%v out=%q", code, err, out.String())
	}
	out.Reset()
	code, err = Run(root, []string{"end", "sid-a"}, Flags{}, &out)
	if err != nil || code != 0 || !strings.Contains(out.String(), `"deleted": true`) {
		t.Fatalf("end code=%d err=%v out=%q", code, err, out.String())
	}
	if _, ok := session.ReadState(root, "sid-a"); ok {
		t.Fatal("session still exists")
	}
}

func TestSessionSetters(t *testing.T) {
	root := t.TempDir()
	if err := session.WriteState(root, "sid-a", session.State{"task": "task a"}); err != nil {
		t.Fatal(err)
	}
	if code, err := Run(root, []string{"set-autocommit", "sid-a", "true"}, Flags{}, &bytes.Buffer{}); err != nil || code != 0 {
		t.Fatalf("set-autocommit code=%d err=%v", code, err)
	}
	if code, err := Run(root, []string{"set-gates", "sid-a", "tests: go test ./..., vet: go vet ./..."}, Flags{}, &bytes.Buffer{}); err != nil || code != 0 {
		t.Fatalf("set-gates code=%d err=%v", code, err)
	}
	if code, err := Run(root, []string{"set-reviews", "sid-a", "security, quality"}, Flags{}, &bytes.Buffer{}); err != nil || code != 0 {
		t.Fatalf("set-reviews code=%d err=%v", code, err)
	}
	state, ok := session.ReadState(root, "sid-a")
	if !ok {
		t.Fatal("state missing")
	}
	if state["autoCommitOverride"] != true {
		t.Fatalf("autoCommitOverride = %#v", state["autoCommitOverride"])
	}
	gates := state["gates"].(map[string]any)
	if gates["tests"] != "go test ./..." || gates["vet"] != "go vet ./..." {
		t.Fatalf("gates = %#v", gates)
	}
	reviews := state["reviews"].([]any)
	if len(reviews) != 2 || reviews[0] != "security" || reviews[1] != "quality" {
		t.Fatalf("reviews = %#v", reviews)
	}
}

func TestSessionRequiresMintForExistingVerbs(t *testing.T) {
	root := t.TempDir()
	code, err := Run(root, []string{"list"}, Flags{}, &bytes.Buffer{})
	if code != 1 || err == nil || !strings.Contains(err.Error(), "No mint session here") {
		t.Fatalf("code=%d err=%v", code, err)
	}
}

func TestParseGateSpec(t *testing.T) {
	got := ParseGateSpec("tests: go test ./..., vet: go vet ./...\nfmt: gofmt -l .")
	if got["tests"] != "go test ./..." || got["vet"] != "go vet ./..." || got["fmt"] != "gofmt -l ." {
		t.Fatalf("ParseGateSpec = %#v", got)
	}
}

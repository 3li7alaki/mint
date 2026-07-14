package main

import (
	"reflect"
	"testing"
)

func TestParseSpecArgsAcceptanceAccumulates(t *testing.T) {
	_, flags, err := parseSpecArgs([]string{
		"--acceptance", "WHEN a, THE s SHALL x",
		"--acceptance", "WHEN b, THE s SHALL y",
		"--acceptance", "WHEN c, THE s SHALL z",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "WHEN a, THE s SHALL x\nWHEN b, THE s SHALL y\nWHEN c, THE s SHALL z"
	if flags.Acceptance != want {
		t.Fatalf("repeated --acceptance did not accumulate:\n got %q\nwant %q", flags.Acceptance, want)
	}
}

func TestParseDoneArgsHasNoMakerFlags(t *testing.T) {
	// --maker-engine/--maker-session were dead (BuildInput reads maker from
	// execution.json), so done must not consume them as flags. This asserts only
	// that the parser no longer recognizes them — they fall through to the
	// positional-arg slice rather than being silently eaten as a flag+value pair.
	out, _, err := parseDoneArgs([]string{"slug", "001", "--maker-engine", "codex"})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, a := range out {
		if a == "--maker-engine" {
			found = true
		}
	}
	if !found {
		t.Fatalf("done still consumes --maker-engine as a flag: %#v", out)
	}
}

func TestParseExecArgsWitnessSentinel(t *testing.T) {
	// The `--` sentinel is the CLI trust boundary for witness: everything after
	// it is the checker argv, verbatim, and must NOT be consumed as flags. A flag
	// name appearing after `--` (here --review) must survive as a literal token,
	// not be interpreted.
	out, flags, err := parseExecArgs([]string{
		"witness", "slug", "001", "--review", "security", "--session", "sid-a",
		"--", "codex", "exec", "--review", "review the diff",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(out, []string{"witness", "slug", "001"}) {
		t.Fatalf("positional args = %#v", out)
	}
	if flags.Review != "security" || flags.Session != "sid-a" {
		t.Fatalf("flags before -- not parsed: %#v", flags)
	}
	wantCmd := []string{"codex", "exec", "--review", "review the diff"}
	if !reflect.DeepEqual(flags.CheckerCmd, wantCmd) {
		t.Fatalf("checker cmd = %#v, want verbatim %#v", flags.CheckerCmd, wantCmd)
	}
}

func TestParseExecArgsSentinelNilVsEmpty(t *testing.T) {
	// No `--` at all: CheckerCmd is nil, so witness can tell "no sentinel" apart
	// from "empty command after sentinel".
	_, flags, err := parseExecArgs([]string{"witness", "slug", "001", "--review", "security"})
	if err != nil {
		t.Fatal(err)
	}
	if flags.CheckerCmd != nil {
		t.Fatalf("no sentinel should leave CheckerCmd nil, got %#v", flags.CheckerCmd)
	}
	// `--` with nothing after it: present-but-empty, distinguishable from nil.
	_, flags, err = parseExecArgs([]string{"witness", "slug", "001", "--"})
	if err != nil {
		t.Fatal(err)
	}
	if flags.CheckerCmd == nil || len(flags.CheckerCmd) != 0 {
		t.Fatalf("empty sentinel should be non-nil empty slice, got %#v", flags.CheckerCmd)
	}
}

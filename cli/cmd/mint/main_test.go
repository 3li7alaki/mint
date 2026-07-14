package main

import "testing"

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

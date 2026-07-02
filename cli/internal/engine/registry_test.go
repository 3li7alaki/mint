package engine

import (
	"reflect"
	"testing"
)

func TestRegistryKeys(t *testing.T) {
	want := []string{"claude", "codex", "kimi", "opencode"}
	if got := Keys(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Keys() = %#v, want %#v", got, want)
	}
	if got := TrustedKeys(); !reflect.DeepEqual(got, want) {
		t.Fatalf("TrustedKeys() = %#v, want %#v", got, want)
	}
}

func TestReferenceSlotsUseManifestProbeOrder(t *testing.T) {
	want := []string{"claude", "codex", "kimi", "opencode"}
	if got := ReferenceSlots(); !reflect.DeepEqual(got, want) {
		t.Fatalf("ReferenceSlots() = %#v, want %#v", got, want)
	}
}

func TestAvailableProbesReferenceRows(t *testing.T) {
	got := Available(func(row Row) bool {
		return row.Binary == "codex" || row.StopFromExit
	})
	keys := make([]string, 0, len(got))
	for _, row := range got {
		keys = append(keys, row.Key)
	}
	want := []string{"codex", "kimi", "opencode"}
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("Available() keys = %#v, want %#v", keys, want)
	}
}

func TestManifestValidation(t *testing.T) {
	for name, raw := range map[string]string{
		"bad json":       `{`,
		"empty":          `{"trusted":[]}`,
		"noncanonical":   `{"trusted":[{"key":" CODEX ","binary":"codex"}]}`,
		"missing binary": `{"trusted":[{"key":"codex"}]}`,
		"duplicate":      `{"trusted":[{"key":"codex","binary":"codex"},{"key":"codex","binary":"codex"}]}`,
	} {
		if _, err := loadRegistry([]byte(raw)); err == nil {
			t.Fatalf("%s: loadRegistry returned nil error", name)
		}
	}
}

func TestIsKnown(t *testing.T) {
	for _, name := range []string{
		"claude",
		" CODEX ",
		"kimi",
		"opencode",
		"ｃｏｄｅｘ",
		"codex\u200e",
		"co\u200bdex",
		"codex\ufeff",
	} {
		if !IsKnown(name) {
			t.Fatalf("IsKnown(%q) = false, want true", name)
		}
	}
	for _, name := range []string{"phantom", "оpus", ""} {
		if IsKnown(name) {
			t.Fatalf("IsKnown(%q) = true, want false", name)
		}
	}
}

func TestCanonical(t *testing.T) {
	cases := map[string]string{
		" CODEX ":      "codex",
		"ｃｏｄｅｘ":        "codex",
		"co\u200bdex":  "codex",
		"codex\u200e":  "codex",
		"codex\ufeff":  "codex",
		"co\u00a0dex":  "co dex",
		"e\u0301ngine": "engine",
	}
	for input, want := range cases {
		if got := Canonical(input); got != want {
			t.Fatalf("Canonical(%q) = %q, want %q", input, got, want)
		}
	}
	if got := Canonical("оpus"); got == "opus" {
		t.Fatalf("Canonical should not fold Cyrillic confusable to Latin: got %q", got)
	}
}

func TestStopFromExitRows(t *testing.T) {
	if !Registry["kimi"].StopFromExit || !Registry["opencode"].StopFromExit {
		t.Fatalf("kimi/opencode should stop from exit: %#v", Registry)
	}
	if Registry["claude"].StopFromExit || Registry["codex"].StopFromExit {
		t.Fatalf("claude/codex should not stop from exit: %#v", Registry)
	}
}

func TestStrictSession(t *testing.T) {
	for _, session := range []string{"sess-maker", "2026-06-30_abc.123", "refs/heads/main@{1}+~^"} {
		if !IsStrictSession(session) {
			t.Fatalf("IsStrictSession(%q) = false, want true", session)
		}
	}
	for _, session := range []string{"", "sess maker", "sess:maker", "ѕess-maker"} {
		if IsStrictSession(session) {
			t.Fatalf("IsStrictSession(%q) = true, want false", session)
		}
	}
}

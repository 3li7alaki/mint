package engine

import (
	"errors"
	"reflect"
	"testing"
)

func TestResolveProvenanceFixedAndConfigurable(t *testing.T) {
	fixed, err := ResolveProvenance(" CODEX ", "", "", "")
	if err != nil || fixed.Vendor != "openai" || fixed.Model != "gpt" || fixed.Locality != "remote" {
		t.Fatalf("fixed provenance = %#v, %v", fixed, err)
	}
	if _, err := ResolveProvenance("codex", "anthropic", "gpt", "remote"); err == nil {
		t.Fatal("conflicting fixed provenance should fail")
	}
	for _, fields := range [][3]string{{"", "llama", "local"}, {"lab", "", "local"}, {"lab", "llama", ""}, {"lab", "llama", "onsite"}} {
		if _, err := ResolveProvenance("opencode", fields[0], fields[1], fields[2]); err == nil {
			t.Fatalf("invalid configurable provenance %#v should fail", fields)
		}
	}
	if _, err := ResolveProvenance("opencode", "оpenai", "gpt-5", "remote"); err == nil {
		t.Fatal("confusable non-ASCII vendor should fail")
	}
	got, err := ResolveProvenance("opencode", "Local Lab", "Llama 70B", "LOCAL")
	if err != nil || got.Vendor != "local lab" || got.Model != "llama 70b" || got.Locality != "local" {
		t.Fatalf("configurable provenance = %#v, %v", got, err)
	}
	if _, err := ResolveExecutionProvenance("opencode", "zai", "glm", "local"); err == nil {
		t.Fatal("caller-supplied local execution provenance should not be trusted")
	}
	if got, err := ResolveExecutionProvenance("opencode", "zai", "glm", "remote"); err != nil || got.Locality != "remote" {
		t.Fatalf("conservative remote execution provenance = %#v, %v", got, err)
	}
	if ProvesLocal("opencode") || ProvesLocal("codex") {
		t.Fatal("remote/configurable engines must not prove local execution")
	}
}

func TestByBinaryResolvesSpawnedProcessToEngine(t *testing.T) {
	reg := map[string]Row{
		"codex":  {Key: "codex", Binary: "codex", Vendor: "openai", Model: "gpt", Locality: "remote"},
		"claude": {Key: "claude", Binary: "claude", Vendor: "anthropic", Model: "claude", Locality: "remote"},
	}
	// Fake PATH: bare names and one absolute path all point at the same files.
	look := func(name string) (string, error) {
		switch name {
		case "codex", "/opt/bin/codex":
			return "/opt/bin/codex", nil
		case "claude":
			return "/usr/bin/claude", nil
		default:
			return "", errors.New("not found")
		}
	}
	if row, ok := byBinary("codex", reg, look); !ok || row.Key != "codex" {
		t.Fatalf("bare name should resolve to codex, got %#v ok=%v", row, ok)
	}
	if row, ok := byBinary("/opt/bin/codex", reg, look); !ok || row.Key != "codex" {
		t.Fatalf("absolute path should resolve to the same engine, got %#v ok=%v", row, ok)
	}
	if _, ok := byBinary("rogue", reg, look); ok {
		t.Fatal("unknown binary must resolve to no engine")
	}
	if _, ok := byBinary("", reg, look); ok {
		t.Fatal("empty name must resolve to no engine")
	}
}

func TestSameEngineIsCanonical(t *testing.T) {
	if !SameEngine(" CODEX ", "codex") {
		t.Fatal("SameEngine should canonicalize before comparing")
	}
	if SameEngine("codex", "claude") {
		t.Fatal("distinct engines must not match")
	}
}

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

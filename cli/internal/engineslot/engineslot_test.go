package engineslot

import (
	"errors"
	"reflect"
	"testing"

	"mint/internal/engine"
)

func TestResolveDefaultReturnsFirstAvailableReferenceSlot(t *testing.T) {
	calls := []string{}
	got, ok := ResolveDefault(Options{
		Probe: func(row engine.Row) error {
			calls = append(calls, row.Key)
			if row.Key == ReferenceSlots()[1] {
				return nil
			}
			return errors.New("not installed")
		},
	})
	if !ok || got != ReferenceSlots()[1] {
		t.Fatalf("ResolveDefault() = %q, %v; want %q, true", got, ok, ReferenceSlots()[1])
	}
	wantCalls := ReferenceSlots()[:2]
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("probe calls = %#v, want %#v", calls, wantCalls)
	}
}

func TestResolveDefaultReturnsFalseWhenNoReferenceSlotLoads(t *testing.T) {
	got, ok := ResolveDefault(Options{
		Probe: func(engine.Row) error {
			return errors.New("not installed")
		},
	})
	if ok || got != "" {
		t.Fatalf("ResolveDefault() = %q, %v; want empty, false", got, ok)
	}
}

func TestReferenceSlotsComeFromEngineManifest(t *testing.T) {
	want := []string{"claude", "codex", "kimi", "opencode"}
	if got := ReferenceSlots(); !reflect.DeepEqual(got, want) {
		t.Fatalf("ReferenceSlots() = %#v, want %#v", got, want)
	}
}

func TestResolveDefaultSkipsUnknownOverrideSlots(t *testing.T) {
	calls := []string{}
	got, ok := ResolveDefault(Options{
		Slots: []string{"missing", "codex"},
		Probe: func(row engine.Row) error {
			calls = append(calls, row.Key)
			return nil
		},
	})
	if !ok || got != "codex" {
		t.Fatalf("ResolveDefault() = %q, %v; want codex, true", got, ok)
	}
	if !reflect.DeepEqual(calls, []string{"codex"}) {
		t.Fatalf("probe calls = %#v, want codex only", calls)
	}
}

func TestResolveDefaultUsesInjectedSlotOrder(t *testing.T) {
	got, ok := ResolveDefault(Options{
		Slots: []string{"opencode", "claude"},
		Probe: func(row engine.Row) error {
			return nil
		},
	})
	if !ok || got != "opencode" {
		t.Fatalf("ResolveDefault() = %q, %v; want opencode, true", got, ok)
	}
}

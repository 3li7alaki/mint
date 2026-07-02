package termination

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"mint/internal/floor"
)

func progressing(overrides func(*History)) History {
	h := History{
		Done:              false,
		ExternalStop:      false,
		Budget:            &Budget{Spent: 10, Limit: 100},
		Attempts:          1,
		MaxAttempts:       5,
		FailureSignatures: []any{"clauseA"},
	}
	if overrides != nil {
		overrides(&h)
	}
	return h
}

func TestDecisionShapeAndVocabulary(t *testing.T) {
	d := Decide(progressing(nil))
	if d.State != Continue {
		t.Fatalf("state = %q, want continue", d.State)
	}
	if d.Reason == "" {
		t.Fatalf("reason is empty")
	}
	if d.Signals == nil {
		t.Fatalf("signals is nil")
	}
	terminal := []string{DoneVerified, BudgetExhausted, StuckEscalated, ExternalStop}
	want := floor.TerminalStates
	if !reflect.DeepEqual(terminal, want) {
		t.Fatalf("terminal states = %#v, want %#v", terminal, want)
	}
}

func TestPrecedence(t *testing.T) {
	d := Decide(History{
		ExternalStop:      true,
		Done:              true,
		Budget:            &Budget{Spent: 200, Limit: 100},
		Attempts:          9,
		MaxAttempts:       5,
		FailureSignatures: []any{"x", "x", "x"},
	})
	if d.State != ExternalStop || len(d.Signals) != 0 {
		t.Fatalf("external stop precedence got %#v", d)
	}

	d = Decide(History{
		Done:              true,
		Budget:            &Budget{Spent: 100, Limit: 100},
		Attempts:          5,
		MaxAttempts:       5,
		FailureSignatures: []any{"a", "a", "a"},
	})
	if d.State != DoneVerified || len(d.Signals) != 0 {
		t.Fatalf("done precedence got %#v", d)
	}

	d = Decide(History{
		Budget:            &Budget{Spent: 100, Limit: 100},
		Attempts:          2,
		MaxAttempts:       5,
		FailureSignatures: []any{"a", "b"},
	})
	if d.State != BudgetExhausted {
		t.Fatalf("budget precedence got %#v", d)
	}
}

func TestDoneVerified(t *testing.T) {
	d := Decide(progressing(func(h *History) { h.Done = true }))
	if d.State != DoneVerified || !strings.Contains(strings.ToLower(d.Reason), "done") {
		t.Fatalf("done got %#v", d)
	}

	d = Decide(History{
		Done:              true,
		Attempts:          5,
		MaxAttempts:       5,
		FailureSignatures: []any{"a", "b", "c", "d", "e"},
	})
	if d.State != DoneVerified {
		t.Fatalf("last allowed done got %#v", d)
	}
}

func TestBudgetExhausted(t *testing.T) {
	cases := []struct {
		name string
		h    History
		want string
	}{
		{"spent equal", progressing(func(h *History) { h.Budget = &Budget{Spent: 100, Limit: 100} }), BudgetExhausted},
		{"spent greater", progressing(func(h *History) { h.Budget = &Budget{Spent: 150, Limit: 100} }), BudgetExhausted},
		{"spent less", progressing(func(h *History) { h.Budget = &Budget{Spent: 50, Limit: 100} }), Continue},
		{"nil budget", progressing(func(h *History) { h.Budget = nil }), Continue},
		{"zero limit", History{Attempts: 0, MaxAttempts: 5, Budget: &Budget{Spent: 0, Limit: 0}}, Continue},
		{"negative limit", progressing(func(h *History) { h.Budget = &Budget{Spent: 5, Limit: -1} }), Continue},
		{"nan limit", progressing(func(h *History) { h.Budget = &Budget{Spent: 5, Limit: math.NaN()} }), Continue},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Decide(tc.h); got.State != tc.want {
				t.Fatalf("state = %q, want %q; decision %#v", got.State, tc.want, got)
			}
		})
	}
}

func TestStuckCapReached(t *testing.T) {
	d := Decide(History{Attempts: 5, MaxAttempts: 5, FailureSignatures: []any{"a", "b", "c", "d", "e"}})
	if d.State != StuckEscalated || !contains(d.Signals, "cap-reached") {
		t.Fatalf("cap reached got %#v", d)
	}
	d = Decide(History{Attempts: 7, MaxAttempts: 5, FailureSignatures: []any{"a", "b", "c"}})
	if !contains(d.Signals, "cap-reached") {
		t.Fatalf("over cap got %#v", d)
	}
	d = Decide(History{Attempts: 3, MaxAttempts: 5, FailureSignatures: []any{"a", "b", "c"}})
	if d.State != Continue {
		t.Fatalf("below cap got %#v", d)
	}
}

func TestStuckNoProgress(t *testing.T) {
	d := Decide(History{Attempts: 2, MaxAttempts: 5, FailureSignatures: []any{"gate:tests", "gate:tests"}})
	if d.State != StuckEscalated || !contains(d.Signals, "no-progress") {
		t.Fatalf("no-progress got %#v", d)
	}
	d = Decide(History{Attempts: 2, MaxAttempts: 5, FailureSignatures: []any{"gate:tests", "gate:lint"}})
	if contains(d.Signals, "no-progress") || d.State != Continue {
		t.Fatalf("different last two got %#v", d)
	}
	d = Decide(History{Attempts: 1, MaxAttempts: 5, FailureSignatures: []any{"gate:tests"}})
	if contains(d.Signals, "no-progress") {
		t.Fatalf("single attempt got %#v", d)
	}
}

func TestFailTwiceOscillation(t *testing.T) {
	d := Decide(History{Attempts: 5, MaxAttempts: 9, FailureSignatures: []any{"clause5", "clause1", "clause5", "clause1", "clause5"}})
	if d.State != StuckEscalated || !contains(d.Signals, "fail-twice") {
		t.Fatalf("oscillation got %#v", d)
	}

	for _, sigs := range [][]any{
		{"a", "b", "c", "a"},
		{"a", "b", "c", "d", "e", "a"},
		{"clause5", "", "clause5"},
	} {
		d = Decide(History{Attempts: float64(len(sigs)), MaxAttempts: 20, FailureSignatures: sigs})
		if contains(d.Signals, "fail-twice") || d.State != Continue {
			t.Fatalf("one-off regress %#v got %#v", sigs, d)
		}
	}

	d = Decide(History{Attempts: 2, MaxAttempts: 5, FailureSignatures: []any{"clause5", "clause5"}})
	if !contains(d.Signals, "no-progress") || contains(d.Signals, "fail-twice") {
		t.Fatalf("adjacent repeat got %#v", d)
	}

	d = Decide(History{Attempts: 4, MaxAttempts: 9, FailureSignatures: []any{"a", "b", "c", "d"}})
	if contains(d.Signals, "fail-twice") || d.State != Continue {
		t.Fatalf("steady different got %#v", d)
	}

	d = Decide(History{Attempts: 5, MaxAttempts: 9, FailureSignatures: []any{"clause5", "", "clause5", "", "clause5"}})
	if !contains(d.Signals, "fail-twice") {
		t.Fatalf("clean middle oscillation got %#v", d)
	}
}

func TestTypeNormalizationAndCapCoercion(t *testing.T) {
	d := Decide(History{Attempts: 2, MaxAttempts: 9, FailureSignatures: []any{map[string]string{"c": "x"}, map[string]string{"c": "x"}}})
	if !contains(d.Signals, "no-progress") {
		t.Fatalf("non-string normalization got %#v", d)
	}

	d = Decide(History{Attempts: 5, MaxAttempts: 5.5, FailureSignatures: []any{"a", "b", "c", "d", "e"}})
	if d.State != StuckEscalated {
		t.Fatalf("fractional cap at 5 got %#v", d)
	}
	d = Decide(History{Attempts: 4, MaxAttempts: 5.5, FailureSignatures: []any{"a", "b", "c", "d"}})
	if d.State != Continue {
		t.Fatalf("fractional cap at 4 got %#v", d)
	}
	for _, cap := range []float64{0.1, 0.5, 0.9, 0.999999, 0, -1, math.NaN()} {
		d = Decide(History{Attempts: 1, MaxAttempts: cap, FailureSignatures: []any{"x"}})
		if d.State != Continue {
			t.Fatalf("sub-1/default cap %v got %#v", cap, d)
		}
	}
	d = Decide(History{Attempts: 1, MaxAttempts: 1, FailureSignatures: []any{"x"}})
	if d.State != StuckEscalated {
		t.Fatalf("cap 1 got %#v", d)
	}
}

func TestMultipleStuckSignals(t *testing.T) {
	d := Decide(History{Attempts: 5, MaxAttempts: 5, FailureSignatures: []any{"a", "b", "c", "d", "d"}})
	if d.State != StuckEscalated || !contains(d.Signals, "cap-reached") || !contains(d.Signals, "no-progress") {
		t.Fatalf("combined signals got %#v", d)
	}
	if !strings.Contains(d.Reason, "d") {
		t.Fatalf("reason does not name repeated signature: %#v", d)
	}
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

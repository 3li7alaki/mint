package termination

import (
	"fmt"
	"math"
	"strings"

	"mint/internal/floor"
)

const (
	DefaultMaxAttempts = 5

	DoneVerified    = "done-verified"
	BudgetExhausted = "budget-exhausted"
	StuckEscalated  = "stuck-escalated"
	ExternalStop    = "external-stop"
	Continue        = "continue"
)

func init() {
	want := map[string]bool{
		DoneVerified:    true,
		BudgetExhausted: true,
		StuckEscalated:  true,
		ExternalStop:    true,
	}
	for _, state := range floor.TerminalStates {
		delete(want, state)
	}
	if len(want) > 0 {
		panic("termination terminal states drifted from floor terminal states")
	}
}

type Budget struct {
	Spent float64
	Limit float64
}

type History struct {
	Done              bool
	ExternalStop      bool
	Budget            *Budget
	Attempts          float64
	MaxAttempts       float64
	FailureSignatures []any
}

type Decision struct {
	State   string
	Reason  string
	Signals []string
}

func Decide(history History) Decision {
	attempts := attempts(history.Attempts)
	cap := resolveCap(history.MaxAttempts)
	sigs := normalizeSignatures(history.FailureSignatures)

	if history.ExternalStop {
		return Decision{State: ExternalStop, Reason: "external stop signaled by caller", Signals: []string{}}
	}
	if history.Done {
		return Decision{State: DoneVerified, Reason: "floor cleared - unit is verified done", Signals: []string{}}
	}
	if budgetExhausted(history.Budget) {
		return Decision{
			State:   BudgetExhausted,
			Reason:  fmt.Sprintf("budget exhausted - spent %v of %v before reaching done", history.Budget.Spent, history.Budget.Limit),
			Signals: []string{},
		}
	}

	signals := []string{}
	if capReached(attempts, cap) {
		signals = append(signals, "cap-reached")
	}
	if noProgress(sigs) {
		signals = append(signals, "no-progress")
	}
	if failTwice(sigs) {
		signals = append(signals, "fail-twice")
	}
	if len(signals) > 0 {
		return Decision{
			State:   StuckEscalated,
			Reason:  fmt.Sprintf("stuck escalated - signals: %s; repeated failure signature: %s", strings.Join(signals, ","), repeatedSignature(sigs)),
			Signals: signals,
		}
	}
	return Decision{State: Continue, Reason: "continue - no terminal condition met", Signals: []string{}}
}

func attempts(value float64) int {
	if !isFinite(value) || value <= 0 {
		return 0
	}
	return int(math.Floor(value))
}

func resolveCap(maxAttempts float64) int {
	if !isFinite(maxAttempts) {
		return DefaultMaxAttempts
	}
	floored := int(math.Floor(maxAttempts))
	if floored < 1 {
		return DefaultMaxAttempts
	}
	return floored
}

func capReached(attempts, cap int) bool {
	return attempts >= 1 && attempts >= cap
}

func budgetExhausted(budget *Budget) bool {
	return budget != nil &&
		isFinite(budget.Limit) &&
		budget.Limit > 0 &&
		isFinite(budget.Spent) &&
		budget.Spent >= budget.Limit
}

func noProgress(sigs []string) bool {
	if len(sigs) < 2 {
		return false
	}
	last := sigs[len(sigs)-1]
	prev := sigs[len(sigs)-2]
	return last != "" && last == prev
}

func failTwice(sigs []string) bool {
	if len(sigs) < 3 {
		return false
	}
	seen := map[string]bool{}
	gapped := map[string]bool{}
	regressCount := map[string]int{}
	for _, sig := range sigs {
		if sig == "" {
			for s := range seen {
				gapped[s] = true
			}
			continue
		}
		if seen[sig] && gapped[sig] {
			regressCount[sig]++
			if regressCount[sig] >= 2 {
				return true
			}
		}
		for s := range seen {
			if s != sig {
				gapped[s] = true
			}
		}
		delete(gapped, sig)
		seen[sig] = true
	}
	return false
}

func repeatedSignature(sigs []string) string {
	for i := len(sigs) - 1; i >= 0; i-- {
		if sigs[i] != "" {
			return sigs[i]
		}
	}
	return "(unknown)"
}

func normalizeSignatures(values []any) []string {
	if len(values) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == nil {
			out = append(out, "")
			continue
		}
		switch v := value.(type) {
		case string:
			out = append(out, v)
		default:
			out = append(out, fmt.Sprint(v))
		}
	}
	return out
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

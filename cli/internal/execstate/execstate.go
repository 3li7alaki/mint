package execstate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mint/internal/atomic"
	"mint/internal/engine"
)

type State struct {
	Status      string            `json:"status"`
	StartedAt   string            `json:"startedAt"`
	CompletedAt *string           `json:"completedAt"`
	Gates       map[string]string `json:"gates"`
	Reviews     map[string]Review `json:"reviews"`
	Commit      *string           `json:"commit"`
	Attempts    []Attempt         `json:"attempts"`
	Maker       *Maker            `json:"maker,omitempty"`
}

type Attempt struct {
	At   string `json:"at"`
	Note string `json:"note,omitempty"`
}

type Provenance struct {
	Engine   string `json:"engine,omitempty"`
	Vendor   string `json:"vendor,omitempty"`
	Model    string `json:"model,omitempty"`
	Locality string `json:"locality,omitempty"`
	Session  string `json:"session,omitempty"`
}

type Maker = Provenance

type Review struct {
	Verdict    string     `json:"verdict"`
	Provenance Provenance `json:"provenance"`
}

// UnmarshalJSON keeps old string-only review entries readable but unattributed.
// They deliberately fail the floor until the review is re-run and overwritten
// with registry-validated provenance; this is not a migration or a trust shim.
func (r *Review) UnmarshalJSON(data []byte) error {
	var legacy string
	if err := json.Unmarshal(data, &legacy); err == nil {
		r.Verdict = legacy
		return nil
	}
	type plain Review
	return json.Unmarshal(data, (*plain)(r))
}

var (
	statuses       = map[string]bool{"running": true, "passed": true, "failed": true, "interrupted": true}
	gateResults    = map[string]bool{"pass": true, "fail": true, "skip": true}
	reviewVerdicts = map[string]bool{"passed": true, "failed": true}
)

func Path(root, slug, specID, sessionID string) string {
	return filepath.Join(root, ".mint", "tasks", sessionID, slug, specID, "execution.json")
}

// OwningSessions returns the session ids whose tasks tree holds an
// execution.json for slug+specID. A unit's execution.json lives under exactly
// one session, so a single hit is the owner; zero means the unit is new; more
// than one is ambiguous and forces --session. This lets done/exec find the
// session that recorded the reviews regardless of cwd (worktree) or a missing
// current-session pin.
func OwningSessions(root, slug, specID string) []string {
	// Walk session dirs and test each candidate path literally. slug/specID are
	// untrusted (spec-derived) — never feed them to filepath.Glob, whose
	// metacharacters ('*', '?', '[') would mis-match or error them into a wrong
	// zero-owner fallthrough. os.ReadDir yields names in sorted order, so owners
	// come out deterministic.
	if !IsLiteralSegment(slug) || !IsLiteralSegment(specID) {
		return nil // path-traversal / separator in an identifier — no owner
	}
	entries, err := os.ReadDir(filepath.Join(root, ".mint", "tasks"))
	if err != nil {
		return nil
	}
	var owners []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if info, statErr := os.Stat(Path(root, slug, specID, e.Name())); statErr == nil && info.Mode().IsRegular() {
			owners = append(owners, e.Name())
		}
	}
	return owners
}

// IsLiteralSegment rejects identifiers that aren't a single, in-place path
// segment, so a crafted slug/specID/sessionID can't escape .mint/tasks/ when a
// command joins them into a read/write path (Path joins sessionID/slug/specID,
// so all three are traversal-controlling). It delegates to the shared primitive
// in atomic so session and execstate enforce identical rules.
func IsLiteralSegment(s string) bool {
	return atomic.IsLiteralSegment(s)
}

// validateSegments fails closed when any path-controlling identifier is not a
// literal segment. Every execstate filesystem boundary routes through it.
func validateSegments(slug, specID, sessionID string) error {
	if !IsLiteralSegment(slug) || !IsLiteralSegment(specID) || !IsLiteralSegment(sessionID) {
		return fmt.Errorf("invalid slug/spec-id/session %q/%q/%q — each must be a single path segment (no empty, '.', '..', or path separator)", slug, specID, sessionID)
	}
	return nil
}

func Read(root, slug, specID, sessionID string) (*State, bool) {
	if !IsLiteralSegment(slug) || !IsLiteralSegment(specID) || !IsLiteralSegment(sessionID) {
		return nil, false // hostile identifier — never resolve to a state
	}
	b, err := os.ReadFile(Path(root, slug, specID, sessionID))
	if err != nil {
		return nil, false
	}
	var state State
	if err := json.Unmarshal(b, &state); err != nil {
		return nil, false
	}
	if state.Gates == nil {
		state.Gates = map[string]string{}
	}
	if state.Reviews == nil {
		state.Reviews = map[string]Review{}
	}
	return &state, true
}

func Init(root, slug, specID, sessionID string, maker *Maker) (*State, error) {
	if err := validateSegments(slug, specID, sessionID); err != nil {
		return nil, err
	}
	if existing, exists := Read(root, slug, specID, sessionID); exists {
		return upgradeMaker(root, slug, specID, sessionID, existing, maker)
	}
	state := &State{
		Status:    "running",
		StartedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Gates:     map[string]string{},
		Reviews:   map[string]Review{},
		Attempts:  []Attempt{},
	}
	resolved, err := resolveMaker(maker)
	if err != nil {
		return nil, err
	}
	state.Maker = resolved
	if err := writeNew(root, slug, specID, sessionID, state); err != nil {
		return nil, err
	}
	return state, nil
}

// resolveMaker snapshots caller-supplied maker provenance against the registry,
// returning nil when nothing identifying was supplied.
func resolveMaker(maker *Maker) (*Maker, error) {
	if maker == nil {
		return nil, nil
	}
	recorded := Maker{Session: strings.TrimSpace(maker.Session)}
	if strings.TrimSpace(maker.Engine) != "" {
		provenance, err := engine.ResolveExecutionProvenance(maker.Engine, maker.Vendor, maker.Model, maker.Locality)
		if err != nil {
			return nil, fmt.Errorf("invalid maker execution provenance: %w", err)
		}
		recorded.Engine = provenance.Engine
		recorded.Vendor = provenance.Vendor
		recorded.Model = provenance.Model
		recorded.Locality = provenance.Locality
	}
	if recorded.Engine == "" && recorded.Session == "" {
		return nil, nil
	}
	return &recorded, nil
}

// upgradeMaker fills maker provenance on an existing state that has none — the
// case where verify/done auto-created a maker-less execution.json before the
// unit ran `exec init`. A state that already records a maker engine is
// write-once and refuses to be overwritten. An exclusive lock serializes the
// check-then-write so two concurrent upgrades can't both observe "no maker" and
// last-writer-win over a freshly recorded maker (the write-once invariant is a
// trust boundary, so it must hold under a race, not just sequential calls).
func upgradeMaker(root, slug, specID, sessionID string, existing *State, maker *Maker) (*State, error) {
	unlock, err := lockUnit(root, slug, specID, sessionID)
	if err != nil {
		return nil, err
	}
	defer unlock()
	// Re-read under the lock: another upgrade may have recorded a maker between
	// the caller's Read and our acquiring the lock.
	if fresh, ok := Read(root, slug, specID, sessionID); ok {
		existing = fresh
	}
	if existing.Maker != nil && strings.TrimSpace(existing.Maker.Engine) != "" {
		return nil, fmt.Errorf("execution state for %s/%s is already initialized — maker provenance is write-once", slug, specID)
	}
	resolved, err := resolveMaker(maker)
	if err != nil {
		return nil, err
	}
	if resolved == nil || resolved.Engine == "" {
		// Nothing to record and nothing recorded yet — a maker-less re-init
		// (e.g. verify re-running). Leave the placeholder untouched.
		return existing, nil
	}
	existing.Maker = resolved
	if err := write(root, slug, specID, sessionID, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func RecordGate(root, slug, specID, gate, result, sessionID string) (*State, error) {
	if strings.TrimSpace(gate) == "" {
		return nil, fmt.Errorf("Gate label must be a non-empty string")
	}
	if gate != "tier" && !gateResults[result] {
		return nil, fmt.Errorf("Invalid gate result %q for %s — expected one of pass, fail, skip", result, gate)
	}
	state, err := requireState(root, slug, specID, sessionID)
	if err != nil {
		return nil, err
	}
	state.Gates[gate] = result
	if err := write(root, slug, specID, sessionID, state); err != nil {
		return nil, err
	}
	return state, nil
}

func RecordReview(root, slug, specID, key, verdict, sessionID string, reviewer *Provenance) (*State, error) {
	if strings.TrimSpace(key) == "" {
		return nil, fmt.Errorf("Review key is required")
	}
	if !reviewVerdicts[verdict] {
		return nil, fmt.Errorf("Invalid review verdict %q — expected one of passed, failed", verdict)
	}
	if reviewer == nil || strings.TrimSpace(reviewer.Engine) == "" || strings.TrimSpace(reviewer.Session) == "" {
		return nil, fmt.Errorf("Review provenance is required — record which engine+session produced the review")
	}
	if !engine.IsStrictSession(reviewer.Session) {
		return nil, fmt.Errorf("Invalid review session %q — expected visible ASCII session/ref characters", reviewer.Session)
	}
	provenance, err := engine.ResolveProvenance(reviewer.Engine, reviewer.Vendor, reviewer.Model, reviewer.Locality)
	if err != nil {
		return nil, fmt.Errorf("invalid review provenance: %w", err)
	}
	state, err := requireState(root, slug, specID, sessionID)
	if err != nil {
		return nil, err
	}
	state.Reviews[key] = Review{
		Verdict: verdict,
		Provenance: Provenance{
			Engine: provenance.Engine, Vendor: provenance.Vendor, Model: provenance.Model,
			Locality: provenance.Locality, Session: strings.TrimSpace(reviewer.Session),
		},
	}
	if err := write(root, slug, specID, sessionID, state); err != nil {
		return nil, err
	}
	return state, nil
}

func SetStatus(root, slug, specID, status, sessionID string, commit *string) (*State, error) {
	if !statuses[status] {
		return nil, fmt.Errorf("Invalid status %q — expected one of running, passed, failed, interrupted", status)
	}
	state, err := requireState(root, slug, specID, sessionID)
	if err != nil {
		return nil, err
	}
	state.Status = status
	if status == "passed" || status == "failed" {
		now := time.Now().UTC().Format(time.RFC3339Nano)
		state.CompletedAt = &now
	}
	if commit != nil {
		state.Commit = commit
	}
	if err := write(root, slug, specID, sessionID, state); err != nil {
		return nil, err
	}
	return state, nil
}

func requireState(root, slug, specID, sessionID string) (*State, error) {
	state, ok := Read(root, slug, specID, sessionID)
	if !ok {
		return nil, fmt.Errorf("No execution.json for %s/%s — run \"mint exec init\" first", slug, specID)
	}
	return state, nil
}

func write(root, slug, specID, sessionID string, state *State) error {
	if err := validateSegments(slug, specID, sessionID); err != nil {
		return err
	}
	return atomic.WriteJSON(Path(root, slug, specID, sessionID), state)
}

func writeNew(root, slug, specID, sessionID string, state *State) error {
	if err := validateSegments(slug, specID, sessionID); err != nil {
		return err
	}
	path := Path(root, slug, specID, sessionID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if os.IsExist(err) {
		return fmt.Errorf("execution state for %s/%s is already initialized — maker provenance is write-once", slug, specID)
	}
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return err
	}
	return f.Close()
}

// lockUnit acquires an exclusive per-unit lock via an O_EXCL lockfile so a
// check-then-write (the maker upgrade) is serialized against a concurrent one.
// It retries briefly on contention, then fails closed rather than proceeding
// unlocked. The returned func releases the lock.
func lockUnit(root, slug, specID, sessionID string) (func(), error) {
	if err := validateSegments(slug, specID, sessionID); err != nil {
		return nil, err
	}
	lockPath := Path(root, slug, specID, sessionID) + ".lock"
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, err
	}
	for attempt := 0; attempt < 100; attempt++ {
		f, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			_ = f.Close()
			return func() { _ = os.Remove(lockPath) }, nil
		}
		if !os.IsExist(err) {
			return nil, err
		}
		time.Sleep(5 * time.Millisecond)
	}
	// Fail closed. We never age-out and reclaim a lock: a "stale" heuristic is
	// itself fail-open — a live-but-paused holder (GC/SIGSTOP/scheduler stall)
	// would let a waiter unlink its lock and both proceed, defeating the
	// write-once serialization this lock exists to guarantee. A lock orphaned by
	// a killed process is a rare, self-evident manual cleanup ("rm <unit>.lock"),
	// not a reason to weaken the invariant.
	return nil, fmt.Errorf("could not acquire execution lock for %s/%s — another init is in progress (if a prior run was killed, remove the stale %s.lock)", slug, specID, Path(root, slug, specID, sessionID))
}

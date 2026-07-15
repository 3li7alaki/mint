// Package execstate stores bounded attempts and their attributable evidence.
package execstate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"mint/internal/atomic"
	"mint/internal/statehome"
	"mint/internal/unitstore"
)

const SchemaVersion = 1

type State struct {
	SchemaVersion int               `json:"schemaVersion"`
	AttemptID     string            `json:"attemptId"`
	Status        string            `json:"status"`
	StartedAt     string            `json:"startedAt"`
	CompletedAt   *string           `json:"completedAt,omitempty"`
	Gates         map[string]string `json:"gates"`
	Reviews       map[string]Review `json:"reviews"`
	Commit        *string           `json:"commit,omitempty"`
	Maker         *Maker            `json:"maker,omitempty"`
}

type Provenance struct {
	Executor     string `json:"executor"`
	Vendor       string `json:"vendor"`
	Model        string `json:"model"`
	Locality     string `json:"locality"`
	ExecutionRef string `json:"executionRef"`
	ObservedBy   string `json:"observedBy,omitempty"`
	Attestation  string `json:"attestation,omitempty"`
}

type Maker = Provenance

type Review struct {
	Verdict    string     `json:"verdict"`
	Provenance Provenance `json:"provenance"`
}

var (
	statuses = map[string]bool{
		"running": true, "done-verified": true, "budget-exhausted": true,
		"stuck-escalated": true, "external-stop": true,
	}
	gateResults    = map[string]bool{"pass": true, "fail": true, "skip": true}
	reviewVerdicts = map[string]bool{"passed": true, "failed": true}
)

func Path(root, slug, specID, attemptID string) string {
	return unitstore.AttemptPath(root, slug, specID, attemptID)
}

func Attempts(root, slug, specID string) []string { return unitstore.Attempts(root, slug, specID) }

func IsLiteralSegment(value string) bool { return atomic.IsLiteralSegment(value) }

func Read(root, slug, specID, attemptID string) (*State, bool) {
	if validateSegments(slug, specID, attemptID) != nil {
		return nil, false
	}
	b, err := os.ReadFile(Path(root, slug, specID, attemptID))
	if err != nil {
		return nil, false
	}
	var state State
	if json.Unmarshal(b, &state) != nil || state.SchemaVersion != SchemaVersion || state.AttemptID != attemptID {
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

func Init(root, slug, specID, attemptID string, maker *Maker) (*State, error) {
	if err := validateSegments(slug, specID, attemptID); err != nil {
		return nil, err
	}
	if _, ok := unitstore.ResolveSpec(root, slug, specID); !ok {
		return nil, fmt.Errorf("unit %s/%s does not exist; create it with mint spec new", slug, specID)
	}
	if maker == nil {
		return nil, fmt.Errorf("maker provenance is required")
	}
	resolved, err := ValidateProvenance(*maker)
	if err != nil {
		return nil, fmt.Errorf("invalid maker provenance: %w", err)
	}
	unlock, err := lockAttempt(root, slug, specID, attemptID)
	if err != nil {
		return nil, err
	}
	defer unlock()
	if _, exists := Read(root, slug, specID, attemptID); exists {
		return nil, fmt.Errorf("attempt %s for %s/%s already exists; maker provenance is immutable", attemptID, slug, specID)
	}
	state := &State{
		SchemaVersion: SchemaVersion,
		AttemptID:     attemptID,
		Status:        "running",
		StartedAt:     time.Now().UTC().Format(time.RFC3339Nano),
		Gates:         map[string]string{},
		Reviews:       map[string]Review{},
		Maker:         &resolved,
	}
	if err := writeNew(root, slug, specID, attemptID, state); err != nil {
		return nil, err
	}
	return state, nil
}

func RecordGate(root, slug, specID, gate, result, attemptID string) (*State, error) {
	if strings.TrimSpace(gate) == "" {
		return nil, fmt.Errorf("gate label is required")
	}
	if gate != "tier" && !gateResults[result] {
		return nil, fmt.Errorf("invalid gate result %q", result)
	}
	return mutate(root, slug, specID, attemptID, func(state *State) error {
		state.Gates[gate] = result
		return nil
	})
}

func RecordReview(root, slug, specID, key, verdict, attemptID string, reviewer *Provenance) (*State, error) {
	if strings.TrimSpace(key) == "" {
		return nil, fmt.Errorf("review key is required")
	}
	if !reviewVerdicts[verdict] {
		return nil, fmt.Errorf("invalid review verdict %q", verdict)
	}
	if reviewer == nil {
		return nil, fmt.Errorf("review provenance is required")
	}
	resolved, err := ValidateProvenance(*reviewer)
	if err != nil {
		return nil, fmt.Errorf("invalid review provenance: %w", err)
	}
	return mutate(root, slug, specID, attemptID, func(state *State) error {
		state.Reviews[key] = Review{Verdict: verdict, Provenance: resolved}
		return nil
	})
}

func SetStatus(root, slug, specID, status, attemptID string, commit *string) (*State, error) {
	if !statuses[status] {
		return nil, fmt.Errorf("invalid status %q", status)
	}
	return mutate(root, slug, specID, attemptID, func(state *State) error {
		if state.CompletedAt != nil && state.Status != status {
			return fmt.Errorf("attempt %s is terminal (%s) and immutable", attemptID, state.Status)
		}
		state.Status = status
		if status != "running" {
			now := time.Now().UTC().Format(time.RFC3339Nano)
			state.CompletedAt = &now
		}
		if commit != nil {
			state.Commit = commit
		}
		return nil
	})
}

func ValidateProvenance(value Provenance) (Provenance, error) {
	value.Executor = strings.TrimSpace(value.Executor)
	value.Vendor = strings.TrimSpace(value.Vendor)
	value.Model = strings.TrimSpace(value.Model)
	value.Locality = strings.ToLower(strings.TrimSpace(value.Locality))
	value.ExecutionRef = strings.TrimSpace(value.ExecutionRef)
	value.ObservedBy = strings.TrimSpace(value.ObservedBy)
	value.Attestation = strings.TrimSpace(value.Attestation)
	for name, field := range map[string]string{
		"executor": value.Executor, "vendor": value.Vendor, "model": value.Model,
		"executionRef": value.ExecutionRef,
	} {
		if !visibleASCII(field) {
			return Provenance{}, fmt.Errorf("%s must be non-empty visible ASCII", name)
		}
	}
	if value.Locality != "local" && value.Locality != "remote" {
		return Provenance{}, fmt.Errorf("locality must be local or remote")
	}
	if (value.ObservedBy == "") != (value.Attestation == "") {
		return Provenance{}, fmt.Errorf("observedBy and attestation must be supplied together")
	}
	if value.ObservedBy != "" && (!visibleASCII(value.ObservedBy) || len(value.Attestation) < 8) {
		return Provenance{}, fmt.Errorf("observation attestation is malformed")
	}
	return value, nil
}

func IsVisibleASCII(value string) bool { return visibleASCII(strings.TrimSpace(value)) }

func mutate(root, slug, specID, attemptID string, apply func(*State) error) (*State, error) {
	if err := validateSegments(slug, specID, attemptID); err != nil {
		return nil, err
	}
	unlock, err := lockAttempt(root, slug, specID, attemptID)
	if err != nil {
		return nil, err
	}
	defer unlock()
	state, ok := Read(root, slug, specID, attemptID)
	if !ok {
		return nil, fmt.Errorf("attempt %s for %s/%s does not exist; run mint exec init", attemptID, slug, specID)
	}
	if state.CompletedAt != nil {
		return nil, fmt.Errorf("attempt %s is terminal (%s); evidence is immutable", attemptID, state.Status)
	}
	if err := apply(state); err != nil {
		return nil, err
	}
	if err := statehome.WriteJSON(Path(root, slug, specID, attemptID), state); err != nil {
		return nil, err
	}
	return state, nil
}

func writeNew(root, slug, specID, attemptID string, state *State) error {
	if err := unitstore.Ensure(root); err != nil {
		return err
	}
	path := Path(root, slug, specID, attemptID)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return err
	}
	return f.Close()
}

func lockAttempt(root, slug, specID, attemptID string) (func(), error) {
	if err := validateSegments(slug, specID, attemptID); err != nil {
		return nil, err
	}
	if err := unitstore.Ensure(root); err != nil {
		return nil, err
	}
	path := unitstore.AttemptLockPath(root, slug, specID, attemptID)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	for i := 0; i < 2000; i++ {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			_ = f.Close()
			return func() { _ = os.Remove(path) }, nil
		}
		if !os.IsExist(err) {
			return nil, err
		}
		time.Sleep(5 * time.Millisecond)
	}
	return nil, fmt.Errorf("attempt %s is locked by another writer", attemptID)
}

func validateSegments(slug, specID, attemptID string) error {
	if !atomic.IsLiteralSegment(slug) || !atomic.IsLiteralSegment(specID) || !atomic.IsLiteralSegment(attemptID) {
		return fmt.Errorf("invalid slug/spec-id/attempt %q/%q/%q", slug, specID, attemptID)
	}
	return nil
}

func visibleASCII(value string) bool {
	if value == "" || !utf8.ValidString(value) {
		return false
	}
	for _, r := range value {
		if r < 0x21 || r > 0x7e {
			return false
		}
	}
	return true
}

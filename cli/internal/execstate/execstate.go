package execstate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mint/internal/atomic"
)

type State struct {
	Status      string            `json:"status"`
	StartedAt   string            `json:"startedAt"`
	CompletedAt *string           `json:"completedAt"`
	Gates       map[string]string `json:"gates"`
	Reviews     map[string]string `json:"reviews"`
	Commit      *string           `json:"commit"`
	Attempts    []Attempt         `json:"attempts"`
	Maker       *Maker            `json:"maker,omitempty"`
}

type Attempt struct {
	At   string `json:"at"`
	Note string `json:"note,omitempty"`
}

type Maker struct {
	Engine  string `json:"engine,omitempty"`
	Session string `json:"session,omitempty"`
}

var (
	statuses       = map[string]bool{"running": true, "passed": true, "failed": true, "interrupted": true}
	gateResults    = map[string]bool{"pass": true, "fail": true, "skip": true}
	reviewVerdicts = map[string]bool{"passed": true, "failed": true}
)

func Path(root, slug, specID, sessionID string) string {
	return filepath.Join(root, ".mint", "tasks", sessionID, slug, specID, "execution.json")
}

func Read(root, slug, specID, sessionID string) (*State, bool) {
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
		state.Reviews = map[string]string{}
	}
	return &state, true
}

func Init(root, slug, specID, sessionID string, maker *Maker) (*State, error) {
	state := &State{
		Status:    "running",
		StartedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Gates:     map[string]string{},
		Reviews:   map[string]string{},
		Attempts:  []Attempt{},
	}
	if maker != nil {
		recorded := Maker{}
		if strings.TrimSpace(maker.Engine) != "" {
			recorded.Engine = maker.Engine
		}
		if strings.TrimSpace(maker.Session) != "" {
			recorded.Session = maker.Session
		}
		if recorded.Engine != "" || recorded.Session != "" {
			state.Maker = &recorded
		}
	}
	if err := write(root, slug, specID, sessionID, state); err != nil {
		return nil, err
	}
	return state, nil
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

func RecordReview(root, slug, specID, key, verdict, sessionID string) (*State, error) {
	if strings.TrimSpace(key) == "" {
		return nil, fmt.Errorf("Review key is required")
	}
	if !reviewVerdicts[verdict] {
		return nil, fmt.Errorf("Invalid review verdict %q — expected one of passed, failed", verdict)
	}
	state, err := requireState(root, slug, specID, sessionID)
	if err != nil {
		return nil, err
	}
	state.Reviews[key] = verdict
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
	return atomic.WriteJSON(Path(root, slug, specID, sessionID), state)
}

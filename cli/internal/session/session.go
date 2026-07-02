package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"mint/internal/atomic"
	"mint/internal/gitignore"
)

const (
	sessionsDir        = ".mint/sessions"
	currentSessionFile = ".current-session-id"
)

type State map[string]any

func GenerateID(now time.Time) (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// hex-ms timestamp (zero-padded to 12) + random suffix — sortable, unique.
	return fmt.Sprintf("%012x-%s", now.UnixMilli(), hex.EncodeToString(b)), nil
}

func GetSessionsDir(root string) string {
	return filepath.Join(root, sessionsDir)
}

func Path(root, sessionID string) string {
	return filepath.Join(GetSessionsDir(root), sessionID+".json")
}

func ReadCapturedID(root string) string {
	b, err := os.ReadFile(filepath.Join(GetSessionsDir(root), currentSessionFile))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func WriteState(root string, sessionID string, state State) error {
	if sessionID == "" {
		sessionID = ReadCapturedID(root)
	}
	if sessionID == "" {
		id, err := GenerateID(time.Now())
		if err != nil {
			return err
		}
		sessionID = id
	}
	if _, err := gitignore.Ensure(root, nil); err != nil {
		return err
	}
	stamped := State{
		"pid":           os.Getpid(),
		"startTime":     time.Now().UnixMilli(),
		"lastHeartbeat": time.Now().UTC().Format(time.RFC3339Nano),
	}
	for k, v := range state {
		stamped[k] = v
	}
	return atomic.WriteJSON(Path(root, sessionID), stamped)
}

func ReadState(root string, sessionID string) (State, bool) {
	if sessionID == "" {
		sessionID = ReadCapturedID(root)
	}
	if sessionID == "" {
		return nil, false
	}
	b, err := os.ReadFile(Path(root, sessionID))
	if err != nil {
		return nil, false
	}
	var state State
	if err := json.Unmarshal(b, &state); err != nil {
		return nil, false
	}
	return state, true
}

func DeleteState(root string, sessionID string) bool {
	if sessionID == "" {
		sessionID = ReadCapturedID(root)
	}
	if sessionID == "" {
		return false
	}
	return os.Remove(Path(root, sessionID)) == nil
}

type Listed struct {
	ID    string
	State State
}

func List(root string) []Listed {
	entries, err := os.ReadDir(GetSessionsDir(root))
	if err != nil {
		return nil
	}
	var out []Listed
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		state, ok := ReadState(root, id)
		if ok {
			out = append(out, Listed{ID: id, State: state})
		}
	}
	return out
}

func CleanStale(root string, maxAge time.Duration, now time.Time) int {
	cleaned := 0
	for _, item := range List(root) {
		raw, _ := item.State["invokedAt"].(string)
		at, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			at = time.Time{}
		}
		if now.Sub(at) > maxAge && DeleteState(root, item.ID) {
			cleaned++
		}
	}
	return cleaned
}

func ReclaimOrphaned(root string, staleHeartbeat time.Duration, now time.Time) int {
	currentID := ReadCapturedID(root)
	reclaimed := 0
	for _, item := range List(root) {
		if currentID != "" && item.ID == currentID {
			continue
		}
		if !isOrphaned(item.State, staleHeartbeat, now) {
			continue
		}
		if DeleteState(root, item.ID) {
			_ = os.RemoveAll(filepath.Join(root, ".mint", "tasks", item.ID))
			reclaimed++
		}
	}
	return reclaimed
}

func isOrphaned(state State, staleHeartbeat time.Duration, now time.Time) bool {
	pid := statePID(state)
	pidAlive := pid > 1 && syscall.Kill(pid, 0) == nil
	heartbeatFresh := false
	if raw, ok := state["lastHeartbeat"].(string); ok {
		if at, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			age := now.Sub(at)
			heartbeatFresh = age >= 0 && age <= staleHeartbeat
		}
	}
	return pid <= 1 || !pidAlive || !heartbeatFresh
}

func statePID(state State) int {
	switch value := state["pid"].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		if value == float64(int(value)) {
			return int(value)
		}
	}
	return 0
}

package handoffcmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"mint/internal/atomic"
	"mint/internal/execstate"
	"mint/internal/hitlist"
	"mint/internal/notelist"
	"mint/internal/session"
	"mint/internal/specschema"
)

type Flags struct {
	Session string
	Notes   []string
}

type SpecInfo struct {
	Slug    string
	SpecID  string
	SpecDir string
}

func Run(root string, args []string, flags Flags, stdout io.Writer) (int, error) {
	if _, err := os.Stat(filepath.Join(root, ".mint")); err != nil {
		return 1, fmt.Errorf("No mint session here - run `mint session new` first")
	}
	sessionID := flags.Session
	if len(args) > 0 {
		sessionID = args[0]
	}
	if sessionID == "" {
		sessionID = session.ReadCapturedID(root)
	}
	if sessionID == "" {
		return 1, fmt.Errorf("No session selected - pass a session id")
	}
	if !execstate.IsLiteralSegment(sessionID) {
		return 1, fmt.Errorf("invalid session %q — must be a single path segment (no empty, '.', '..', or path separator)", sessionID)
	}
	state, _ := session.ReadState(root, sessionID)
	specs := FindSpecs(root, sessionID)
	seed := Build(root, sessionID, state, specs, flags.Notes)
	seedPath := filepath.Join(session.GetSessionsDir(root), sessionID, "handoff.md")
	if err := atomic.WriteString(seedPath, seed); err != nil {
		return 1, err
	}
	fmt.Fprintln(stdout, seedPath)
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, seed)
	return 0, nil
}

func FindSpecs(root, sessionID string) []SpecInfo {
	tasksDir := filepath.Join(root, ".mint", "tasks", sessionID)
	slugs := dirs(tasksDir)
	found := []SpecInfo{}
	for _, slug := range slugs {
		for _, specID := range dirs(filepath.Join(tasksDir, slug)) {
			specDir := filepath.Join(tasksDir, slug, specID)
			if _, err := os.Stat(filepath.Join(specDir, "execution.json")); err == nil {
				found = append(found, SpecInfo{Slug: slug, SpecID: specID, SpecDir: specDir})
			}
		}
	}
	sort.Slice(found, func(i, j int) bool {
		if found[i].Slug == found[j].Slug {
			return found[i].SpecID < found[j].SpecID
		}
		return found[i].Slug < found[j].Slug
	})
	return found
}

func Build(root, sessionID string, state session.State, specs []SpecInfo, notes []string) string {
	lines := []string{"# Handoff - " + sessionID, ""}
	lines = append(lines, "- Task: "+stateString(state, "task", "(none)"))
	lines = append(lines, "- Mode: "+stateString(state, "mode", "(unknown)"))
	if state["autoCommitOverride"] == false {
		lines = append(lines, "- Autocommit: OFF (session override)")
	}
	lines = append(lines, "")
	if len(specs) == 0 {
		lines = append(lines, "## Specs", "(no spec state recorded yet)", "")
	}
	for _, spec := range specs {
		lines = append(lines, "## "+spec.Slug+"/"+spec.SpecID)
		if scope := scopeForSpec(spec.SpecDir); len(scope) > 0 {
			lines = append(lines, "- Scope (can-modify): "+strings.Join(scope, ", "))
		}
		if state, ok := execstate.Read(root, spec.Slug, spec.SpecID, sessionID); ok {
			lines = append(lines, "- Exec: status="+state.Status+" - gates: "+formatMap(state.Gates)+" - reviews: "+formatReviews(state.Reviews))
			commit := "(none)"
			if state.Commit != nil && *state.Commit != "" {
				commit = *state.Commit
			}
			lines = append(lines, fmt.Sprintf("- Commit: %s - attempts: %d", commit, len(state.Attempts)))
		}
		lines = append(lines, "")
	}
	cleanNotes := []string{}
	for _, note := range notes {
		if strings.TrimSpace(note) != "" {
			cleanNotes = append(cleanNotes, strings.TrimSpace(note))
		}
	}
	if len(cleanNotes) > 0 {
		lines = append(lines, "## Durable decisions")
		for _, note := range cleanNotes {
			lines = append(lines, "- "+note)
		}
		lines = append(lines, "")
	}
	// The open agenda rides into the seed — the marquee resurface point. Whatever work
	// spawned mid-session survives /clear and lands in the next session's continue.
	if open, err := hitlist.Open(root); err == nil && len(open) > 0 {
		lines = append(lines, "## Open hits (mint hit)")
		for _, it := range open {
			lines = append(lines, fmt.Sprintf("- %s [%s] %s", it.ID, it.Priority, it.Text))
		}
		lines = append(lines, "")
	}
	// Scratchpad topics ride in too — the reasoning-so-far a fresh context most needs. The
	// seed lists the topics + a `mint note show <topic>` pointer rather than inlining big
	// bodies (the next session pulls the full reasoning on demand).
	if notes, err := notelist.Read(root); err == nil && len(notes) > 0 {
		lines = append(lines, "## Scratchpad (mint note show <topic>)")
		for _, n := range notes {
			lines = append(lines, fmt.Sprintf("- %s (%d entries)", n.Topic, n.Entries))
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func dirs(path string) []string {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}
	out := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			out = append(out, entry.Name())
		}
	}
	sort.Strings(out)
	return out
}

func scopeForSpec(specDir string) []string {
	entries, err := os.ReadDir(specDir)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".xml") {
			b, err := os.ReadFile(filepath.Join(specDir, entry.Name()))
			if err == nil {
				return specschema.ResolveCanModify(string(b))
			}
		}
	}
	return nil
}

func formatMap(values map[string]string) string {
	if len(values) == 0 {
		return "(none)"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values[key])
	}
	return strings.Join(parts, " ")
}

func formatReviews(values map[string]execstate.Review) string {
	verdicts := make(map[string]string, len(values))
	for key, review := range values {
		verdicts[key] = review.Verdict
	}
	return formatMap(verdicts)
}

func stateString(state session.State, key, fallback string) string {
	if state == nil {
		return fallback
	}
	if value, ok := state[key].(string); ok && value != "" {
		return value
	}
	return fallback
}

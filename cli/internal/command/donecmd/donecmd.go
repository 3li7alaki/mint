package donecmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mint/internal/execstate"
	"mint/internal/floor"
	"mint/internal/notelist"
	"mint/internal/session"
)

type Flags struct {
	Verdict  string
	Terminal string
	Session  string
	Spec     string
	Base     string
	JSON     bool
}

func Run(root string, args []string, flags Flags, stdout, stderr io.Writer) (int, error) {
	if len(args) < 2 {
		return 1, fmt.Errorf("Usage: mint done <slug> <spec-id> [--verdict <path>] [--terminal <state>] [--session <id>] [--spec <path>] [--base <ref>] [--json]")
	}
	if _, err := os.Stat(filepath.Join(root, ".mint")); err != nil {
		return 1, fmt.Errorf("No mint session here - run `mint session new` first")
	}
	slug, specID := args[0], args[1]
	if !execstate.IsLiteralSegment(slug) || !execstate.IsLiteralSegment(specID) {
		return 1, fmt.Errorf("invalid slug/spec-id %q/%q — each must be a single path segment (no empty, '.', '..', or path separator)", slug, specID)
	}
	sessionID, err := resolveSessionID(root, slug, specID, flags.Session)
	if err != nil {
		return 1, err
	}
	specPath, err := ResolveSpecPath(root, slug, specID, sessionID, flags.Spec)
	if err != nil {
		return 1, err
	}
	verdictPath := flags.Verdict
	if verdictPath == "" {
		verdictPath = filepath.Join(root, ".mint", "verdicts", slug+"-"+specID+".json")
	} else if !filepath.IsAbs(verdictPath) {
		verdictPath = filepath.Join(root, verdictPath)
	}
	terminalState := flags.Terminal

	input, err := floor.BuildInput(root, floor.BuildOptions{
		SpecPath:      specPath,
		Slug:          slug,
		SpecID:        specID,
		VerdictPath:   verdictPath,
		TerminalState: terminalState,
		SessionID:     sessionID,
		Base:          flags.Base,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1, nil
	}
	result := floor.Enforce(input)
	if !result.Pass {
		topic := failNoteTopic(slug, specID)
		if _, err := notelist.Append(root, topic, failNoteText(result), nil, time.Now()); err != nil {
			fmt.Fprintln(stderr, "note: could not record done-fail note:", err)
		}
	}
	if flags.JSON {
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return 1, err
		}
		fmt.Fprintln(stdout, string(b))
	} else {
		fmt.Fprintln(stdout, FormatReport(result))
	}
	if result.Pass {
		return 0, nil
	}
	return 1, nil
}

func ResolveSpecPath(root, slug, specID, sessionID, explicit string) (string, error) {
	if explicit != "" {
		if filepath.IsAbs(explicit) {
			return explicit, nil
		}
		return filepath.Join(root, explicit), nil
	}
	dir := filepath.Join(root, ".mint", "tasks", sessionID, slug)
	entries, err := os.ReadDir(dir)
	if err != nil {
		entries = nil
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".xml") {
			continue
		}
		if name == specID+".xml" || strings.HasPrefix(name, specID+"-") {
			return filepath.Join(dir, name), nil
		}
	}
	return "", fmt.Errorf("Could not resolve a spec for %s/%s under %s - pass --spec <path>", slug, specID, dir)
}

func FormatReport(result floor.Result) string {
	headline := "PASS - floor clean, done"
	if !result.Pass {
		headline = fmt.Sprintf("FAIL - floor not clean (failed clauses: %s)", joinInts(result.Failed))
	}
	lines := []string{"  " + headline}
	for _, c := range result.Clauses {
		mark := "ok"
		why := ""
		if !c.Pass {
			mark = "fail"
			if c.Reason != nil {
				why = " - " + *c.Reason
			}
		}
		lines = append(lines, fmt.Sprintf("    %s clause %d: %s%s", mark, c.Clause, c.Name, why))
	}
	return strings.Join(lines, "\n")
}

// resolveSessionID resolves the session that OWNS this unit. done re-checks an
// existing execution.json, so an explicit --session wins, then the session that
// actually holds the unit's execution.json (survives a worktree cwd or a missing
// current-session pin), then the pinned id, and only a genuinely new unit falls
// through to a generated id. Minting a fresh id for an existing unit was the bug:
// done targeted an empty execution.json and clause-1 (declared reviews) could
// never be satisfied.
func resolveSessionID(root, slug, specID, explicit string) (string, error) {
	if id := strings.TrimSpace(explicit); id != "" {
		if !execstate.IsLiteralSegment(id) {
			return "", fmt.Errorf("invalid session %q — must be a single path segment (no empty, '.', '..', or path separator)", id)
		}
		return id, nil
	}
	switch owners := execstate.OwningSessions(root, slug, specID); len(owners) {
	case 1:
		return owners[0], nil
	case 0:
		// no execution.json yet — fall through to pin/generate (new unit)
	default:
		return "", fmt.Errorf("multiple sessions own %s/%s (%s) — pass --session <id> to pick one", slug, specID, strings.Join(owners, ", "))
	}
	if id := session.ReadCapturedID(root); id != "" {
		return id, nil
	}
	generated, err := session.GenerateID(time.Now())
	if err != nil {
		return "", err
	}
	return generated, nil
}

// failNoteTopic keys a done-fail note to the spec so retries accumulate under
// one topic. Deterministic: same slug+specID always yields the same topic.
func failNoteTopic(slug, specID string) string {
	return sanitizeTopic("done-fail-" + slug + "-" + specID)
}

// sanitizeTopic lowercases and replaces any char outside [a-z0-9._-] with '-'
// so the result satisfies notelist's topic regex.
func sanitizeTopic(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z':
			b = append(b, c+('a'-'A'))
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '.', c == '_', c == '-':
			b = append(b, c)
		default:
			b = append(b, '-')
		}
	}
	return string(b)
}

// failNoteText composes the note body from the floor result: which clauses
// failed and why. This is the evidence a sharper re-mint of the spec consumes.
func failNoteText(result floor.Result) string {
	lines := []string{"floor FAIL - failed clauses: " + joinInts(result.Failed)}
	for _, c := range result.Clauses {
		if c.Pass {
			continue
		}
		why := ""
		if c.Reason != nil {
			why = ": " + *c.Reason
		}
		lines = append(lines, fmt.Sprintf("clause %d (%s)%s", c.Clause, c.Name, why))
	}
	return strings.Join(lines, "\n")
}

func joinInts(values []int) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprint(value))
	}
	return strings.Join(parts, ", ")
}

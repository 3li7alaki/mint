package donecmd

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"mint/internal/execstate"
	"mint/internal/floor"
	"mint/internal/notelist"
	"mint/internal/receipt"
	"mint/internal/snapshot"
	"mint/internal/statehome"
	"mint/internal/unitstore"
)

type Flags struct {
	Verdict  string
	Terminal string
	Attempt  string
	Spec     string
	Base     string
	JSON     bool
}

var captureSnapshot = snapshot.Capture

func Run(root string, args []string, flags Flags, stdout, stderr io.Writer) (int, error) {
	if len(args) < 2 {
		return 1, fmt.Errorf("Usage: mint done <slug> <spec-id> [--attempt <id>] [--verdict <path>] [--terminal <state>] [--base <ref>] [--json]")
	}
	slug, specID := args[0], args[1]
	if !execstate.IsLiteralSegment(slug) || !execstate.IsLiteralSegment(specID) {
		return 1, fmt.Errorf("invalid slug/spec-id %q/%q: each must be a single path segment (no empty, '.', '..', or path separator)", slug, specID)
	}
	attemptID, err := resolveAttemptID(root, slug, specID, flags.Attempt)
	if err != nil {
		return 1, err
	}
	specPath, err := ResolveSpecPath(root, slug, specID, flags.Spec)
	if err != nil {
		return 1, err
	}
	verdictPath := flags.Verdict
	if verdictPath == "" {
		verdictPath = unitstore.VerdictPath(root, slug, specID, attemptID)
	} else if !filepath.IsAbs(verdictPath) {
		verdictPath = filepath.Join(root, verdictPath)
	}
	terminalState := flags.Terminal
	if terminalState == "" {
		terminalState = "done-verified"
	}
	before, err := captureSnapshot(root, flags.Base)
	if err != nil {
		fmt.Fprintln(stderr, "snapshot:", err)
		return 1, nil
	}

	input, err := floor.BuildInput(root, floor.BuildOptions{
		SpecPath:      specPath,
		Slug:          slug,
		SpecID:        specID,
		VerdictPath:   verdictPath,
		TerminalState: terminalState,
		AttemptID:     attemptID,
		Base:          flags.Base,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1, nil
	}
	result := floor.Enforce(input)
	var issued *receipt.Record
	var receiptPath string
	if result.Pass {
		after, snapshotErr := captureSnapshot(root, flags.Base)
		if snapshotErr != nil {
			fmt.Fprintln(stderr, "snapshot:", snapshotErr)
			return 1, nil
		}
		if before.Digest != after.Digest {
			fmt.Fprintln(stderr, "source changed while mint done was evaluating the floor; retry against a stable snapshot")
			return 1, nil
		}
		defaultVerdictPath := unitstore.VerdictPath(root, slug, specID, attemptID)
		if verdictPath != defaultVerdictPath {
			if err := statehome.WriteJSON(defaultVerdictPath, input.Verdict); err != nil {
				fmt.Fprintln(stderr, "verdict:", err)
				return 1, nil
			}
		}
		record, receiptErr := receipt.New(receipt.NewOptions{
			Slug: slug, SpecID: specID, AttemptID: attemptID, Terminal: terminalState,
			Snapshot: before, Result: result, Input: input, IssuedAt: time.Now(),
		})
		if receiptErr != nil {
			fmt.Fprintln(stderr, "receipt:", receiptErr)
			return 1, nil
		}
		receiptPath, receiptErr = receipt.Store(root, record)
		if receiptErr != nil {
			fmt.Fprintln(stderr, "receipt:", receiptErr)
			return 1, nil
		}
		issued = &record
		if _, statusErr := execstate.SetStatus(root, slug, specID, terminalState, attemptID, nil); statusErr != nil {
			fmt.Fprintln(stderr, "attempt:", statusErr)
			return 1, nil
		}
	}
	if !result.Pass {
		topic := failNoteTopic(slug, specID)
		if _, err := notelist.Append(root, topic, failNoteText(result), nil, time.Now()); err != nil {
			fmt.Fprintln(stderr, "note: could not record done-fail note:", err)
		}
	}
	if flags.JSON {
		output := struct {
			SchemaVersion int                  `json:"schemaVersion"`
			Pass          bool                 `json:"pass"`
			Clauses       []floor.ClauseResult `json:"clauses"`
			Failed        []int                `json:"failed"`
			Receipt       *receipt.Record      `json:"receipt,omitempty"`
			ReceiptPath   string               `json:"receiptPath,omitempty"`
		}{
			SchemaVersion: 1, Pass: result.Pass, Clauses: result.Clauses, Failed: result.Failed,
			Receipt: issued, ReceiptPath: relativePath(root, receiptPath),
		}
		b, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return 1, err
		}
		fmt.Fprintln(stdout, string(b))
	} else {
		fmt.Fprintln(stdout, FormatReport(result))
		if receiptPath != "" {
			fmt.Fprintln(stdout, "  receipt:", relativePath(root, receiptPath))
		}
	}
	if result.Pass {
		return 0, nil
	}
	return 1, nil
}

func relativePath(root, path string) string {
	if path == "" {
		return ""
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return path
	}
	return filepath.ToSlash(rel)
}

func ResolveSpecPath(root, slug, specID, explicit string) (string, error) {
	if explicit != "" {
		if filepath.IsAbs(explicit) {
			return explicit, nil
		}
		return filepath.Join(root, explicit), nil
	}
	if path, ok := unitstore.ResolveSpec(root, slug, specID); ok {
		return path, nil
	}
	return "", fmt.Errorf("Could not resolve unit %s/%s - pass --spec <path>", slug, specID)
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

func resolveAttemptID(root, slug, specID, explicit string) (string, error) {
	if id := strings.TrimSpace(explicit); id != "" {
		if !execstate.IsLiteralSegment(id) {
			return "", fmt.Errorf("invalid attempt %q", id)
		}
		return id, nil
	}
	switch owners := execstate.Attempts(root, slug, specID); len(owners) {
	case 1:
		return owners[0], nil
	case 0:
		return "", fmt.Errorf("no attempt for %s/%s; run mint exec init", slug, specID)
	default:
		return "", fmt.Errorf("multiple attempts exist for %s/%s (%s); pass --attempt <id>", slug, specID, strings.Join(owners, ", "))
	}
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

package execcmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mint/internal/engineslot"
	"mint/internal/execstate"
	"mint/internal/session"
)

type Flags struct {
	Session     string
	MakerEngine string
	Commit      string
}

func Run(root string, args []string, flags Flags, stdout, stderr io.Writer) (int, error) {
	if len(args) == 0 {
		fmt.Fprintln(stdout, "  Usage: mint exec init|record-gate|record-review|set-status|status|reviews <slug> <spec-id> ...")
		return 1, nil
	}
	if _, err := os.Stat(filepath.Join(root, ".mint")); err != nil {
		return 1, fmt.Errorf("No mint session here - run `mint session new` first")
	}
	switch args[0] {
	case "init":
		if len(args) < 3 {
			return 1, fmt.Errorf("Usage: mint exec init <slug> <spec-id>")
		}
		// init creates the unit, so there is no owning session to resolve to.
		sessionID, err := newUnitSessionID(root, flags.Session)
		if err != nil {
			return 1, err
		}
		makerEngine := flags.MakerEngine
		if makerEngine == "" {
			if slot, ok := engineslot.ResolveDefault(engineslot.Options{}); ok {
				makerEngine = slot
			}
		}
		state, err := execstate.Init(root, args[1], args[2], sessionID, &execstate.Maker{Engine: makerEngine, Session: sessionID})
		if err != nil {
			return 1, err
		}
		return printJSON(stdout, state)
	case "record-gate":
		if len(args) < 5 {
			return 1, fmt.Errorf("Usage: mint exec record-gate <slug> <spec-id> <gate> <result>")
		}
		sessionID, err := resolveSessionID(root, args[1], args[2], flags.Session)
		if err != nil {
			return 1, err
		}
		state, err := execstate.RecordGate(root, args[1], args[2], args[3], args[4], sessionID)
		if err != nil {
			return 1, err
		}
		return printJSON(stdout, state)
	case "record-review":
		if len(args) < 5 {
			return 1, fmt.Errorf("Usage: mint exec record-review <slug> <spec-id> <key> <verdict>")
		}
		sessionID, err := resolveSessionID(root, args[1], args[2], flags.Session)
		if err != nil {
			return 1, err
		}
		state, err := execstate.RecordReview(root, args[1], args[2], args[3], args[4], sessionID)
		if err != nil {
			return 1, err
		}
		return printJSON(stdout, state)
	case "set-status":
		if len(args) < 4 {
			return 1, fmt.Errorf("Usage: mint exec set-status <slug> <spec-id> <status> [--commit <hash>]")
		}
		sessionID, err := resolveSessionID(root, args[1], args[2], flags.Session)
		if err != nil {
			return 1, err
		}
		var commit *string
		if flags.Commit != "" {
			commit = &flags.Commit
		}
		state, err := execstate.SetStatus(root, args[1], args[2], args[3], sessionID, commit)
		if err != nil {
			return 1, err
		}
		return printJSON(stdout, state)
	case "status":
		if len(args) < 3 {
			return 1, fmt.Errorf("Usage: mint exec status <slug> <spec-id>")
		}
		sessionID, err := resolveSessionID(root, args[1], args[2], flags.Session)
		if err != nil {
			return 1, err
		}
		state, ok := execstate.Read(root, args[1], args[2], sessionID)
		if !ok {
			fmt.Fprintf(stderr, "No execution.json for %s/%s\n", args[1], args[2])
			return 1, nil
		}
		fmt.Fprintln(stdout, state.Status)
		return 0, nil
	case "reviews":
		if len(args) < 3 {
			return 1, fmt.Errorf("Usage: mint exec reviews <slug> <spec-id>")
		}
		sessionID, err := resolveSessionID(root, args[1], args[2], flags.Session)
		if err != nil {
			return 1, err
		}
		state, ok := execstate.Read(root, args[1], args[2], sessionID)
		if !ok {
			fmt.Fprintf(stderr, "No execution.json for %s/%s\n", args[1], args[2])
			return 1, nil
		}
		return printJSON(stdout, state.Reviews)
	default:
		fmt.Fprintln(stdout, "  Usage: mint exec init|record-gate|record-review|set-status|status|reviews <slug> <spec-id> ...")
		return 1, nil
	}
}

// resolveSessionID resolves the session that OWNS this unit for commands that
// read or mutate an existing execution.json (record-*, set-status, status,
// reviews). Explicit --session wins, then the session that actually holds the
// unit's execution.json — this is what survives a worktree cwd or a missing
// current-session pin — then the pin, then a generated id. Minting a fresh id
// for an existing unit was the bug that left done's clause-1 unsatisfiable.
func resolveSessionID(root, slug, specID, explicit string) (string, error) {
	if id := strings.TrimSpace(explicit); id != "" {
		return id, nil
	}
	switch owners := execstate.OwningSessions(root, slug, specID); len(owners) {
	case 1:
		return owners[0], nil
	case 0:
		// no execution.json yet — fall through to pin/generate
	default:
		return "", fmt.Errorf("multiple sessions own %s/%s (%s) — pass --session <id> to pick one", slug, specID, strings.Join(owners, ", "))
	}
	return newUnitSessionID(root, "")
}

// newUnitSessionID resolves the session for a brand-new unit (exec init): there
// is no owning execution.json to find, so explicit --session wins, then the pin,
// then a generated id.
func newUnitSessionID(root, explicit string) (string, error) {
	if id := strings.TrimSpace(explicit); id != "" {
		return id, nil
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

func printJSON(stdout io.Writer, value any) (int, error) {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return 1, err
	}
	fmt.Fprintln(stdout, string(b))
	return 0, nil
}

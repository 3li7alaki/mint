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
	sessionID, err := resolveSessionID(root, flags.Session)
	if err != nil {
		return 1, err
	}

	switch args[0] {
	case "init":
		if len(args) < 3 {
			return 1, fmt.Errorf("Usage: mint exec init <slug> <spec-id>")
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
		state, err := execstate.RecordGate(root, args[1], args[2], args[3], args[4], sessionID)
		if err != nil {
			return 1, err
		}
		return printJSON(stdout, state)
	case "record-review":
		if len(args) < 5 {
			return 1, fmt.Errorf("Usage: mint exec record-review <slug> <spec-id> <key> <verdict>")
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

func resolveSessionID(root, explicit string) (string, error) {
	id := strings.TrimSpace(explicit)
	if id != "" {
		return id, nil
	}
	id = session.ReadCapturedID(root)
	if id != "" {
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

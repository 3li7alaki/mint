package execcmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"mint/internal/engine"
	"mint/internal/engineslot"
	"mint/internal/execstate"
	"mint/internal/session"
)

type Flags struct {
	Session     string
	MakerEngine string
	Commit      string
	ByEngine    string
	ByVendor    string
	ByModel     string
	ByLocality  string
	BySession   string
	Review      string
	CheckerCmd  []string // checker argv after `--`; nil = no sentinel supplied
}

func Run(root string, args []string, flags Flags, stdout, stderr io.Writer) (int, error) {
	if len(args) == 0 {
		fmt.Fprintln(stdout, "  Usage: mint exec init|record-gate|record-review|witness|set-status|status|reviews <slug> <spec-id> ...")
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
			return 1, fmt.Errorf("Usage: mint exec record-review <slug> <spec-id> <key> <verdict> --by-engine <engine> --by-session <session> [--by-vendor <vendor> --by-model <model> --by-locality <local|remote>]")
		}
		sessionID, err := resolveSessionID(root, args[1], args[2], flags.Session)
		if err != nil {
			return 1, err
		}
		state, err := execstate.RecordReview(root, args[1], args[2], args[3], args[4], sessionID, &execstate.Provenance{
			Engine: flags.ByEngine, Vendor: flags.ByVendor, Model: flags.ByModel,
			Locality: flags.ByLocality, Session: flags.BySession,
		})
		if err != nil {
			return 1, err
		}
		return printJSON(stdout, state)
	case "witness":
		return runWitness(root, args, flags, stdout, stderr)
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
		fmt.Fprintln(stdout, "  Usage: mint exec init|record-gate|record-review|witness|set-status|status|reviews <slug> <spec-id> ...")
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

// runWitness spawns the checker process itself and records the review verdict
// from THAT process, closing the typed-identity hole: the recorded engine is
// reverse-resolved from the binary mint actually ran (engine.ByBinary), never
// from a caller-typed --by-engine. Verdict is the child's exit code (0=passed),
// so mint stays content-blind exactly like a gate. It is a notary, not a
// harness — the driver chooses the checker and writes its prompt; mint only
// witnesses that a registry engine ran and grades the provenance.
func runWitness(root string, args []string, flags Flags, stdout, stderr io.Writer) (int, error) {
	if len(args) < 3 {
		return 1, fmt.Errorf("Usage: mint exec witness <slug> <spec-id> --review <lens> [--by-vendor <v> --by-model <m> --by-locality <local|remote>] -- <checker command...>")
	}
	slug, specID := args[1], args[2]
	lens := strings.TrimSpace(flags.Review)
	if lens == "" {
		return 1, fmt.Errorf("--review <lens> is required — name the review lens this witnessed check satisfies")
	}
	if flags.CheckerCmd == nil {
		return 1, fmt.Errorf("no checker command — put the checker after `--`, e.g. mint exec witness %s %s --review %s -- codex exec ...", slug, specID, lens)
	}
	if len(flags.CheckerCmd) == 0 {
		return 1, fmt.Errorf("empty checker command after `--`")
	}

	// Resolve WHO ran from the binary mint is about to spawn, before spawning.
	// An unrecognized binary can never produce a witnessed verdict.
	row, ok := engine.ByBinary(flags.CheckerCmd[0])
	if !ok {
		return 1, fmt.Errorf("checker binary %q resolves to no registry engine — witness only records a verdict for a recognized engine", flags.CheckerCmd[0])
	}
	// A caller-supplied --by-engine may not contradict the binary that actually
	// runs; the process is the source of truth, not the flag.
	if e := strings.TrimSpace(flags.ByEngine); e != "" && !engine.SameEngine(e, row.Key) {
		return 1, fmt.Errorf("--by-engine %q conflicts with the spawned binary, which resolves to engine %q — witness records the engine that runs", e, row.Key)
	}

	sessionID, err := resolveSessionID(root, slug, specID, flags.Session)
	if err != nil {
		return 1, err
	}

	// Spawn the checker directly (no shell) so argv[0] is the exact binary we
	// reverse-resolved, and stream its output to the driver so they see it run.
	cmd := exec.Command(flags.CheckerCmd[0], flags.CheckerCmd[1:]...)
	cmd.Dir = root
	cmd.Stdout = stderr
	cmd.Stderr = stderr
	runErr := cmd.Run()
	verdict := "passed"
	var exitErr *exec.ExitError
	switch {
	case runErr == nil:
		verdict = "passed"
	case errors.As(runErr, &exitErr):
		verdict = "failed"
	default:
		// Could not spawn/observe the process (binary vanished, permission) —
		// record nothing rather than a verdict we didn't witness.
		return 1, fmt.Errorf("could not run witnessed checker %q: %w", flags.CheckerCmd[0], runErr)
	}

	// Provenance comes from the observed engine. Fixed engines carry registry
	// vendor/model/locality; a configurable chassis binary can't reveal its
	// model, so witness still needs caller-declared --by-vendor/--by-model there.
	reviewer := &execstate.Provenance{Engine: row.Key, Session: sessionID}
	if row.Configurable {
		reviewer.Vendor = flags.ByVendor
		reviewer.Model = flags.ByModel
		reviewer.Locality = flags.ByLocality
	}
	state, err := execstate.RecordReview(root, slug, specID, lens, verdict, sessionID, reviewer)
	if err != nil {
		return 1, err
	}
	fmt.Fprintf(stderr, "witnessed %s: engine=%s verdict=%s\n", lens, row.Key, verdict)
	return printJSON(stdout, state)
}

func printJSON(stdout io.Writer, value any) (int, error) {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return 1, err
	}
	fmt.Fprintln(stdout, string(b))
	return 0, nil
}

package execcmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"mint/internal/execstate"
	"mint/internal/unitstore"
)

type Flags struct {
	Attempt      string
	Executor     string
	Vendor       string
	Model        string
	Locality     string
	ExecutionRef string
	ObservedBy   string
	Attestation  string
	Commit       string
}

func Run(root string, args []string, flags Flags, stdout, stderr io.Writer) (int, error) {
	if len(args) == 0 {
		return usage(stdout)
	}
	switch args[0] {
	case "init":
		if len(args) != 3 {
			return 1, fmt.Errorf("Usage: mint exec init <slug> <spec-id> [--attempt <id>] --executor <name> --vendor <name> --model <name> --locality <local|remote> [--execution-ref <ref>]")
		}
		attemptID, err := newAttemptID(flags.Attempt)
		if err != nil {
			return 1, err
		}
		maker := provenance(flags, attemptID)
		state, err := execstate.Init(root, args[1], args[2], attemptID, &maker)
		if err != nil {
			return 1, err
		}
		return printJSON(stdout, state)

	case "record-gate":
		if len(args) != 5 {
			return 1, fmt.Errorf("Usage: mint exec record-gate <slug> <spec-id> <gate> <pass|fail|skip> --attempt <id>")
		}
		attemptID, err := resolveAttemptID(root, args[1], args[2], flags.Attempt)
		if err != nil {
			return 1, err
		}
		state, err := execstate.RecordGate(root, args[1], args[2], args[3], args[4], attemptID)
		if err != nil {
			return 1, err
		}
		return printJSON(stdout, state)

	case "record-review":
		if len(args) != 5 {
			return 1, fmt.Errorf("Usage: mint exec record-review <slug> <spec-id> <lens> <passed|failed> --attempt <id> --executor <name> --vendor <name> --model <name> --locality <local|remote> --execution-ref <ref>")
		}
		attemptID, err := resolveAttemptID(root, args[1], args[2], flags.Attempt)
		if err != nil {
			return 1, err
		}
		reviewer := provenance(flags, "")
		state, err := execstate.RecordReview(root, args[1], args[2], args[3], args[4], attemptID, &reviewer)
		if err != nil {
			return 1, err
		}
		return printJSON(stdout, state)

	case "set-status":
		if len(args) != 4 {
			return 1, fmt.Errorf("Usage: mint exec set-status <slug> <spec-id> <state> --attempt <id> [--commit <hash>]")
		}
		attemptID, err := resolveAttemptID(root, args[1], args[2], flags.Attempt)
		if err != nil {
			return 1, err
		}
		var commit *string
		if flags.Commit != "" {
			commit = &flags.Commit
		}
		state, err := execstate.SetStatus(root, args[1], args[2], args[3], attemptID, commit)
		if err != nil {
			return 1, err
		}
		return printJSON(stdout, state)

	case "show":
		if len(args) != 3 {
			return 1, fmt.Errorf("Usage: mint exec show <slug> <spec-id> [--attempt <id>]")
		}
		attemptID, err := resolveAttemptID(root, args[1], args[2], flags.Attempt)
		if err != nil {
			return 1, err
		}
		state, ok := execstate.Read(root, args[1], args[2], attemptID)
		if !ok {
			fmt.Fprintf(stderr, "attempt %s not found\n", attemptID)
			return 1, nil
		}
		return printJSON(stdout, state)
	default:
		return usage(stdout)
	}
}

func ResolveAttemptID(root, slug, specID, explicit string) (string, error) {
	return resolveAttemptID(root, slug, specID, explicit)
}

func resolveAttemptID(root, slug, specID, explicit string) (string, error) {
	if id := strings.TrimSpace(explicit); id != "" {
		if !execstate.IsLiteralSegment(id) {
			return "", fmt.Errorf("invalid attempt %q", id)
		}
		return id, nil
	}
	attempts := execstate.Attempts(root, slug, specID)
	if len(attempts) == 1 {
		return attempts[0], nil
	}
	if len(attempts) == 0 {
		return "", fmt.Errorf("no attempt for %s/%s; run mint exec init", slug, specID)
	}
	return "", fmt.Errorf("multiple attempts exist for %s/%s (%s); pass --attempt <id>", slug, specID, strings.Join(attempts, ", "))
}

func newAttemptID(explicit string) (string, error) {
	if id := strings.TrimSpace(explicit); id != "" {
		if !execstate.IsLiteralSegment(id) {
			return "", fmt.Errorf("invalid attempt %q", id)
		}
		return id, nil
	}
	return unitstore.GenerateAttemptID(time.Now())
}

func provenance(flags Flags, fallbackRef string) execstate.Provenance {
	return execstate.Provenance{
		Executor:     first(flags.Executor, os.Getenv("MINT_EXECUTOR")),
		Vendor:       first(flags.Vendor, os.Getenv("MINT_VENDOR")),
		Model:        first(flags.Model, os.Getenv("MINT_MODEL")),
		Locality:     first(flags.Locality, os.Getenv("MINT_LOCALITY")),
		ExecutionRef: first(flags.ExecutionRef, os.Getenv("MINT_EXECUTION_REF"), fallbackRef),
		ObservedBy:   flags.ObservedBy,
		Attestation:  flags.Attestation,
	}
}

func first(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func printJSON(stdout io.Writer, value any) (int, error) {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return 1, err
	}
	_, err = fmt.Fprintln(stdout, string(b))
	return 0, err
}

func usage(stdout io.Writer) (int, error) {
	fmt.Fprintln(stdout, "Usage: mint exec init|record-gate|record-review|set-status|show ...")
	return 1, nil
}

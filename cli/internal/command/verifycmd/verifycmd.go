package verifycmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"mint/internal/unitstore"
	"mint/internal/verify"
)

type Flags struct {
	Attempt string
	Spec    string
	JSON    bool
}

func Run(root string, args []string, flags Flags, stdout io.Writer) (int, error) {
	if len(args) < 2 {
		return 1, fmt.Errorf("Usage: mint verify <slug> <spec-id> [--attempt <id>]")
	}
	attemptID, err := resolveAttempt(root, args[0], args[1], flags.Attempt)
	if err != nil {
		return 1, err
	}
	specPath := flags.Spec
	if specPath == "" {
		var ok bool
		specPath, ok = unitstore.ResolveSpec(root, args[0], args[1])
		if !ok {
			return 1, fmt.Errorf("unit %s/%s does not exist", args[0], args[1])
		}
	} else if !filepath.IsAbs(specPath) {
		specPath = filepath.Join(root, specPath)
	}
	result := verify.Run(root, args[0], args[1], attemptID, specPath)
	if flags.JSON {
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return 1, err
		}
		fmt.Fprintln(stdout, string(b))
		if result.Error != "" || !result.OK {
			return 1, nil
		}
		return 0, nil
	}
	if result.Error != "" {
		return 1, fmt.Errorf("verify: %s", result.Error)
	}
	if result.Tier == "skip" {
		fmt.Fprintln(stdout, "  tier: skip - docs-only diff, no gates needed")
		return 0, nil
	}
	if !result.Declared {
		fmt.Fprintln(stdout, "  no gates declared for this unit - add them to the spec <gates> or run")
		fmt.Fprintln(stdout, "  mint runs unit-declared gates and never guesses commands.")
		return 1, nil
	}
	fmt.Fprintf(stdout, "  tier: %s - %s\n", result.Tier, gateLine(result.Results))
	if result.PrepareFailed {
		printLogTail(stdout, "prepare", "", result.PrepareLog)
	}
	labels := make([]string, 0, len(result.Results))
	for label := range result.Results {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	for _, label := range labels {
		if result.Results[label] == "pass" {
			continue
		}
		path := result.Logs[label]
		if result.PrepareFailed && path == "" {
			fmt.Fprintf(stdout, "  %s: inconclusive, not run, prepare failed\n", label)
			continue
		}
		printLogTail(stdout, label, result.Results[label]+", "+result.Reasons[label]+"; ", path)
	}
	if result.Hint != "" {
		fmt.Fprintf(stdout, "  hint: %s\n", result.Hint)
	}
	if result.OK {
		return 0, nil
	}
	return 1, nil
}

func printLogTail(stdout io.Writer, label, detail, path string) {
	fmt.Fprintf(stdout, "  %s: %slast 20 lines of %s\n", label, detail, path)
	for _, line := range tail(path) {
		fmt.Fprintf(stdout, "    | %s\n", line)
	}
}

func tail(path string) []string {
	body, err := os.ReadFile(path)
	if err != nil || len(body) == 0 {
		return []string{"(no output)"}
	}
	lines := strings.Split(strings.TrimSuffix(string(body), "\n"), "\n")
	if len(lines) > 20 {
		lines = lines[len(lines)-20:]
	}
	return lines
}

func resolveAttempt(root, slug, specID, explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return explicit, nil
	}
	attempts := unitstore.Attempts(root, slug, specID)
	if len(attempts) == 1 {
		return attempts[0], nil
	}
	if len(attempts) == 0 {
		return "", fmt.Errorf("no attempt for %s/%s; run mint exec init", slug, specID)
	}
	return "", fmt.Errorf("multiple attempts exist for %s/%s; pass --attempt", slug, specID)
}

func gateLine(results map[string]string) string {
	if len(results) == 0 {
		return "no gates to run"
	}
	labels := make([]string, 0, len(results))
	for label := range results {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	parts := make([]string, 0, len(labels))
	for _, label := range labels {
		prefix := "ok"
		if results[label] == "fail" || results[label] == "inconclusive" {
			prefix = results[label]
		}
		parts = append(parts, prefix+" "+label)
	}
	return strings.Join(parts, "  ")
}

package verifycmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"mint/internal/verify"
)

type Flags struct {
	Session string
	Spec    string
}

func Run(root string, args []string, flags Flags, stdout io.Writer) (int, error) {
	if len(args) < 2 {
		return 1, fmt.Errorf("Usage: mint verify <slug> <spec-id> [--session <id>]")
	}
	if _, err := os.Stat(filepath.Join(root, ".mint")); err != nil {
		return 1, fmt.Errorf("No mint session here - run `mint session new` first")
	}
	result := verify.Run(root, args[0], args[1], flags.Session, flags.Spec)
	if result.Tier == "skip" {
		fmt.Fprintln(stdout, "  tier: skip - docs-only diff, no gates needed")
		return 0, nil
	}
	if !result.Declared {
		fmt.Fprintln(stdout, "  no gates declared for this unit - add them to the spec <gates> or run")
		fmt.Fprintln(stdout, "  `mint session set-gates '<cmd>, <cmd>'`. mint runs declared gates, never guesses.")
		return 1, nil
	}
	fmt.Fprintf(stdout, "  tier: %s - %s\n", result.Tier, gateLine(result.Results))
	if result.OK {
		return 0, nil
	}
	return 1, nil
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
		if results[label] == "fail" {
			prefix = "fail"
		}
		parts = append(parts, prefix+" "+label)
	}
	return strings.Join(parts, "  ")
}

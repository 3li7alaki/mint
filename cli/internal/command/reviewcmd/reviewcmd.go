package reviewcmd

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

type Flags struct {
	List  bool
	Focus string
	Lens  string
}

type lens struct {
	Description string
	Body        string
}

var lenses = map[string]lens{
	"adversarial": {
		Description: "A review lens the driver selects (`mint review --adversarial`). Actively tries to break the implementation.",
		Body:        lensBody("adversarial", "You are the adversarial lens. Try to break this code with edge cases, malicious inputs, and boundary probes. Report reproducible failures as BLOCKING."),
	},
	"business": {
		Description: "A review lens the driver selects (`mint review --business`). Reviews business requirements and domain logic.",
		Body:        lensBody("business", "You are the business lens. Check whether the implementation solves the stated business problem and matches domain rules. Read-only: report findings and a verdict."),
	},
	"conventions": {
		Description: "A review lens the driver selects (`mint review --conventions`). Verifies project conventions.",
		Body:        lensBody("conventions", "You are the conventions lens. Compare the diff against AGENTS.md, docs, and existing code patterns. Read-only: report convention violations."),
	},
	"performance": {
		Description: "A review lens the driver selects (`mint review --performance`). Reviews for performance issues.",
		Body:        lensBody("performance", "You are the performance lens. Look for real costs: N+1 work, unnecessary synchronous work, memory leaks, excessive rendering, and bundle impact."),
	},
	"quality": {
		Description: "A review lens the driver selects (`mint review --quality`). Reviews code quality, readability, and type safety.",
		Body:        lensBody("quality", "You are the quality lens. Review patterns, readability, DRY, over-engineering, type safety, and fit with the existing codebase."),
	},
	"security": {
		Description: "A review lens the driver selects (`mint review --security`). Scans for security vulnerabilities.",
		Body:        lensBody("security", "You are the security lens. Trace data flow and check injection, XSS, auth, authorization, secrets, unsafe deserialization, SSRF, and data exposure."),
	},
	"test": {
		Description: "A review lens the driver selects (`mint review --test`). Reviews test quality.",
		Body:        lensBody("test", "You are the test lens. Check whether tests can fail when behavior breaks, avoid internal mocks, and cover meaningful edge cases."),
	},
}

func Run(args []string, flags Flags, stdout, stderr io.Writer) (int, error) {
	if flags.Focus != "" {
		printFocus(stdout, flags.Focus)
		return 0, nil
	}
	if flags.List {
		printList(stdout)
		return 0, nil
	}
	name := ""
	if len(args) > 0 {
		name = args[0]
	} else if flags.Lens != "" {
		name = flags.Lens
	}
	if name == "" {
		printList(stdout)
		return 0, nil
	}
	row, ok := lenses[name]
	if !ok {
		fmt.Fprintf(stderr, "No lens named %q.\n", name)
		fmt.Fprintln(stderr, "Run `mint review --list` for available lenses.")
		return 1, nil
	}
	fmt.Fprint(stdout, row.Body)
	return 0, nil
}

func printList(stdout io.Writer) {
	names := names()
	if len(names) == 0 {
		fmt.Fprintln(stdout, "No review lenses found.")
		return
	}
	fmt.Fprintln(stdout, "Review lenses (select with `mint review --<lens>`):")
	fmt.Fprintln(stdout)
	for _, name := range names {
		fmt.Fprintf(stdout, "  %s\n", name)
		if lenses[name].Description != "" {
			fmt.Fprintf(stdout, "      %s\n", lenses[name].Description)
		}
	}
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, `  Ad-hoc: mint review --focus "<what to look for>"`)
}

func printFocus(stdout io.Writer, focus string) {
	fmt.Fprintln(stdout, "# ad-hoc review lens")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "You are a review lens the driver selected for one specific concern. Look at the")
	fmt.Fprintln(stdout, "diff through exactly this lens and report findings ([BLOCKING]/[WARNING]/[INFO],")
	fmt.Fprintln(stdout, "with file:line) plus a one-line verdict. Read-only - report, do not fix.")
	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "Focus: %s\n", strings.TrimSpace(focus))
}

func names() []string {
	names := make([]string, 0, len(lenses))
	for name := range lenses {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func lensBody(name, body string) string {
	return "---\nname: " + name + "\ndescription: " + lensesafe(body) + "\n---\n\n" + body + "\n"
}

func lensesafe(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 120 {
		return s[:120]
	}
	return s
}

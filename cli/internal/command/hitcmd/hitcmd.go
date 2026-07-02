// Package hitcmd is the `mint hit` CLI surface over internal/hitlist — mint's durable
// mid-session agenda. Verbs:
//
//	mint hit add "<text>" [--priority now|next|later] [--session <id>]
//	mint hit list [--open]
//	mint hit done <id>
//	mint hit drop <id>
//
// AUTO-CAPTURE is orchestrator behavior, not code here: the driving agent recognizes
// "remember/do-later" intent in conversation and calls `mint hit add` ITSELF, announcing it
// visibly. This package just owns the deterministic storage + lifecycle.
package hitcmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"mint/internal/hitlist"
)

// Flags are the parsed `mint hit` options.
type Flags struct {
	Priority string
	Session  string
	OpenOnly bool
	Body     string   // --body <file>: read big markdown content from this file
	Files    []string // --file <path> (repeatable): repo paths this hit is about
}

// Run dispatches a `mint hit` subcommand. now is injected for deterministic tests.
func Run(root string, args []string, flags Flags, now time.Time, stdout io.Writer) (int, error) {
	if len(args) == 0 {
		return usage(stdout)
	}
	switch args[0] {
	case "add":
		if len(args) < 2 {
			return 1, fmt.Errorf(`usage: mint hit add "<text>" [--priority now|next|later] [--body <file>] [--file <path>]`)
		}
		// --body names a file whose contents become the hit's spilled markdown body (big
		// content lives as a real file, never crammed into the JSONL row).
		body := ""
		if flags.Body != "" {
			b, err := os.ReadFile(flags.Body)
			if err != nil {
				return 1, fmt.Errorf("reading --body %s: %w", flags.Body, err)
			}
			body = string(b)
		}
		it, err := hitlist.Add(root, args[1], flags.Priority, hitlist.AddOpts{
			Body: body, Files: flags.Files, Session: flags.Session,
		}, now)
		if err != nil {
			return 1, err
		}
		extra := ""
		if it.BodyPath != "" {
			extra += " (+body " + it.BodyPath + ")"
		}
		if len(it.Files) > 0 {
			extra += " (re: " + strings.Join(it.Files, ", ") + ")"
		}
		fmt.Fprintf(stdout, "📌 noted %s [%s]: %s%s\n", it.ID, it.Priority, it.Text, extra)
		return 0, nil

	case "show":
		if len(args) < 2 {
			return 1, fmt.Errorf("usage: mint hit show <id>")
		}
		return show(root, args[1], stdout)

	case "list":
		return list(root, flags.OpenOnly, stdout)

	case "done":
		if len(args) < 2 {
			return 1, fmt.Errorf("usage: mint hit done <id>")
		}
		it, err := hitlist.Done(root, args[1], now)
		if err != nil {
			return 1, err
		}
		fmt.Fprintf(stdout, "✓ done %s: %s\n", it.ID, it.Text)
		return 0, nil

	case "drop":
		if len(args) < 2 {
			return 1, fmt.Errorf("usage: mint hit drop <id>")
		}
		it, err := hitlist.Drop(root, args[1], now)
		if err != nil {
			return 1, err
		}
		fmt.Fprintf(stdout, "✗ dropped %s: %s\n", it.ID, it.Text)
		return 0, nil

	default:
		return usage(stdout)
	}
}

func list(root string, openOnly bool, stdout io.Writer) (int, error) {
	if openOnly {
		open, err := hitlist.Open(root)
		if err != nil {
			return 1, err
		}
		if len(open) == 0 {
			fmt.Fprintln(stdout, "  no open hits")
			return 0, nil
		}
		for _, it := range open {
			fmt.Fprintf(stdout, "  %s [%s] %s\n", it.ID, it.Priority, it.Text)
		}
		return 0, nil
	}

	items, err := hitlist.Read(root)
	if err != nil {
		return 1, err
	}
	if len(items) == 0 {
		fmt.Fprintln(stdout, "  no hits")
		return 0, nil
	}
	for _, it := range items {
		mark := map[string]string{"open": " ", "done": "✓", "dropped": "✗"}[it.Status]
		fmt.Fprintf(stdout, "  %s %s [%s] %s\n", mark, it.ID, it.Priority, it.Text)
	}
	return 0, nil
}

// show prints one hit in full: the row metadata, its file references, and its spilled
// markdown body if it has one.
func show(root, id string, stdout io.Writer) (int, error) {
	items, err := hitlist.Read(root)
	if err != nil {
		return 1, err
	}
	for _, it := range items {
		if it.ID != id {
			continue
		}
		fmt.Fprintf(stdout, "%s [%s] %s — %s\n", it.ID, it.Priority, it.Status, it.Text)
		if len(it.Files) > 0 {
			fmt.Fprintf(stdout, "  files: %s\n", strings.Join(it.Files, ", "))
		}
		body, err := hitlist.Body(root, it)
		if err != nil {
			return 1, err
		}
		if body != "" {
			fmt.Fprintf(stdout, "\n%s\n", body)
		}
		return 0, nil
	}
	return 1, fmt.Errorf("no hit %q (see `mint hit list`)", id)
}

func usage(stdout io.Writer) (int, error) {
	fmt.Fprintln(stdout, `usage: mint hit add "<text>" [--priority now|next|later] [--body <file>] [--file <path>] | list [--open] | show <id> | done <id> | drop <id>`)
	return 1, nil
}

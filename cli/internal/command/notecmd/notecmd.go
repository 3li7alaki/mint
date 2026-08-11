// Package notecmd is the `mint note` CLI surface over internal/notelist, mint's scratchpad
// for the AI's reasoning-in-progress. Verbs:
//
//	mint note add <topic> "<text>" [--body <file>] [--file <path>]
//	mint note show <topic>
//	mint note list
//
// A note appends to its topic (append-mostly working memory), so calling add on the same
// topic across turns accumulates the reasoning instead of overwriting it.
package notecmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"mint/internal/notelist"
)

// Flags are the parsed `mint note` options.
type Flags struct {
	Body  string   // --body <file>: read big markdown content from this file
	Files []string // --file <path> (repeatable): repo paths this note is about
}

// Run dispatches a `mint note` subcommand. now is injected for deterministic tests.
func Run(root string, args []string, flags Flags, now time.Time, stdout io.Writer) (int, error) {
	if len(args) == 0 {
		return usage(stdout)
	}
	switch args[0] {
	case "add":
		if len(args) < 2 {
			return 1, fmt.Errorf(`usage: mint note add <topic> "<text>" [--body <file>] [--file <path>]`)
		}
		topic := args[1]
		// Text is the optional positional after the topic; --body adds (or replaces) big content.
		text := ""
		if len(args) >= 3 {
			text = args[2]
		}
		if flags.Body != "" {
			b, err := os.ReadFile(flags.Body)
			if err != nil {
				return 1, fmt.Errorf("reading --body %s: %w", flags.Body, err)
			}
			if text != "" {
				text += "\n\n"
			}
			text += string(b)
		}
		n, err := notelist.Append(root, topic, text, flags.Files, now)
		if err != nil {
			return 1, err
		}
		fmt.Fprintf(stdout, "🗒 noted on %q (%d entr%s)\n", n.Topic, n.Entries, plural(n.Entries))
		return 0, nil

	case "show":
		if len(args) < 2 {
			return 1, fmt.Errorf("usage: mint note show <topic>")
		}
		body, err := notelist.Body(root, args[1])
		if err != nil {
			return 1, err
		}
		if body == "" {
			fmt.Fprintf(stdout, "  no note on %q\n", args[1])
			return 0, nil
		}
		fmt.Fprint(stdout, body)
		return 0, nil

	case "list":
		notes, err := notelist.Read(root)
		if err != nil {
			return 1, err
		}
		if len(notes) == 0 {
			fmt.Fprintln(stdout, "  no notes")
			return 0, nil
		}
		for _, n := range notes {
			line := fmt.Sprintf("  %s (%d)", n.Topic, n.Entries)
			if len(n.Files) > 0 {
				line += " - " + strings.Join(n.Files, ", ")
			}
			fmt.Fprintln(stdout, line)
		}
		return 0, nil

	default:
		return usage(stdout)
	}
}

func plural(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

func usage(stdout io.Writer) (int, error) {
	fmt.Fprintln(stdout, `usage: mint note add <topic> "<text>" [--body <file>] [--file <path>] | show <topic> | list`)
	return 1, nil
}

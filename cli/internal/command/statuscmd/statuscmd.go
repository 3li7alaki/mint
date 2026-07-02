package statuscmd

import (
	"fmt"
	"io"

	"mint/internal/hitlist"
	"mint/internal/notelist"
	"mint/internal/session"
	"mint/internal/worktree"
)

// Version is injected at build time via ldflags
// (-X 'mint/internal/command/statuscmd.Version=vX.Y.Z'); "dev" for a plain `go build`.
var Version = "dev"

func Run(root string, stdout io.Writer) (int, error) {
	fmt.Fprintf(stdout, "\n  mint v%s\n\n", Version)

	sessions := session.List(root)
	if len(sessions) > 0 {
		fmt.Fprintln(stdout, "  Sessions")
		for _, item := range sessions {
			fmt.Fprintf(stdout, "    %s %s - %s\n", shortID(item.ID), stateString(item.State, "mode", "session"), stateString(item.State, "task", "unknown task"))
		}
	} else {
		fmt.Fprintln(stdout, "  Sessions:   none active")
	}

	worktrees := worktree.List(root)
	if len(worktrees) > 0 {
		fmt.Fprintf(stdout, "  Worktrees:  %d active\n", len(worktrees))
	}
	// Resurface the open agenda — a glanceable nudge so durable intentions don't rot in
	// scrollback. Silent (prints nothing) when there are no open hits.
	if summary := hitlist.Summary(root); summary != "" {
		fmt.Fprintf(stdout, "  %s\n", summary)
	}
	if summary := notelist.Summary(root); summary != "" {
		fmt.Fprintf(stdout, "  %s\n", summary)
	}
	fmt.Fprintln(stdout)
	return 0, nil
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12] + "..."
	}
	return id
}

func stateString(state session.State, key, fallback string) string {
	if value, ok := state[key].(string); ok && value != "" {
		return value
	}
	return fallback
}

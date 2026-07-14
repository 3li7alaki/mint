package sessioncmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mint/internal/atomic"
	"mint/internal/session"
)

type Flags struct {
	Session  string
	Task     string
	Mode     string
	NoCommit bool
	Commit   bool
}

func Run(root string, args []string, flags Flags, stdout io.Writer) (int, error) {
	if len(args) == 0 {
		fmt.Fprintln(stdout, "  Usage: mint session new|list|show|set-autocommit|set-gates|set-reviews|resume|end ...")
		return 1, nil
	}
	if args[0] == "new" {
		return runNew(root, flags, stdout)
	}
	if _, err := os.Stat(filepath.Join(root, ".mint")); err != nil {
		return 1, fmt.Errorf("No mint session here - run `mint session new` first")
	}
	switch args[0] {
	case "list":
		sessions := session.List(root)
		if len(sessions) == 0 {
			fmt.Fprintln(stdout, "  No active sessions.")
			return 0, nil
		}
		for _, item := range sessions {
			fmt.Fprintf(stdout, "  %s  %s  %s\n", item.ID, stateString(item.State, "mode", "?"), stateString(item.State, "task", ""))
		}
		return 0, nil
	case "show", "resume":
		if len(args) < 2 {
			return 1, fmt.Errorf("Usage: mint session %s <id>", args[0])
		}
		state, err := requireSession(root, args[1])
		if err != nil {
			return 1, err
		}
		return printJSON(stdout, state)
	case "set-autocommit":
		if len(args) < 3 {
			return 1, fmt.Errorf("Usage: mint session set-autocommit <id> <true|false>")
		}
		raw := strings.ToLower(args[2])
		if raw != "true" && raw != "false" {
			return 1, fmt.Errorf("Usage: mint session set-autocommit <id> <true|false>")
		}
		state, err := requireSession(root, args[1])
		if err != nil {
			return 1, err
		}
		state["autoCommitOverride"] = raw == "true"
		if err := session.WriteState(root, args[1], state); err != nil {
			return 1, err
		}
		return printJSON(stdout, state)
	case "set-gates":
		if len(args) < 3 {
			return 1, fmt.Errorf(`Usage: mint session set-gates <id> "<label>: <cmd>, <label>: <cmd>"`)
		}
		state, err := requireSession(root, args[1])
		if err != nil {
			return 1, err
		}
		state["gates"] = ParseGateSpec(strings.Join(args[2:], " "))
		if err := session.WriteState(root, args[1], state); err != nil {
			return 1, err
		}
		return printJSON(stdout, state)
	case "set-reviews":
		if len(args) < 3 {
			return 1, fmt.Errorf("Usage: mint session set-reviews <id> <lens,lens,...>")
		}
		state, err := requireSession(root, args[1])
		if err != nil {
			return 1, err
		}
		state["reviews"] = parseReviews(strings.Join(args[2:], " "))
		if err := session.WriteState(root, args[1], state); err != nil {
			return 1, err
		}
		return printJSON(stdout, state)
	case "end":
		if len(args) < 2 {
			return 1, fmt.Errorf("Usage: mint session end <id>")
		}
		deleted := session.DeleteState(root, args[1])
		if deleted && session.ReadCapturedID(root) == args[1] {
			_ = session.ClearCapturedID(root)
		}
		return printJSON(stdout, map[string]any{"sessionId": args[1], "deleted": deleted})
	default:
		fmt.Fprintln(stdout, "  Usage: mint session new|list|show|set-autocommit|set-gates|set-reviews|resume|end ...")
		return 1, nil
	}
}

func runNew(root string, flags Flags, stdout io.Writer) (int, error) {
	id := strings.TrimSpace(flags.Session)
	if id == "" {
		id = session.ReadCapturedID(root)
	}
	if id == "" {
		generated, err := session.GenerateID(time.Now())
		if err != nil {
			return 1, err
		}
		id = generated
	}
	if !atomic.IsLiteralSegment(id) {
		return 1, fmt.Errorf("invalid session %q — must be a single path segment (no empty, '.', '..', or path separator)", id)
	}
	mode := flags.Mode
	if mode == "" {
		mode = "quick"
	}
	var autoCommitOverride any
	autoCommitOverride = nil
	if flags.NoCommit {
		autoCommitOverride = false
	} else if flags.Commit {
		autoCommitOverride = true
	}
	state := session.State{
		"mintInvoked":        true,
		"invokedAt":          time.Now().UTC().Format(time.RFC3339Nano),
		"task":               flags.Task,
		"mode":               mode,
		"autoCommitOverride": autoCommitOverride,
	}
	if err := session.WriteState(root, id, state); err != nil {
		return 1, err
	}
	if err := session.WriteCapturedID(root, id); err != nil {
		return 1, err
	}
	return printJSON(stdout, map[string]any{"sessionId": id, "dir": session.GetSessionsDir(root)})
}

func ParseGateSpec(text string) map[string]string {
	gates := map[string]string{}
	for _, item := range strings.FieldsFunc(text, func(r rune) bool { return r == ',' || r == '\n' }) {
		i := strings.Index(item, ":")
		if i == -1 {
			continue
		}
		label := strings.TrimSpace(item[:i])
		cmd := strings.TrimSpace(item[i+1:])
		if label != "" && cmd != "" {
			gates[label] = cmd
		}
	}
	return gates
}

func parseReviews(text string) []string {
	parts := strings.Split(text, ",")
	reviews := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			reviews = append(reviews, part)
		}
	}
	return reviews
}

func requireSession(root, id string) (session.State, error) {
	state, ok := session.ReadState(root, id)
	if !ok {
		return nil, fmt.Errorf("No session %q - check \"mint session list\"", id)
	}
	return state, nil
}

func printJSON(stdout io.Writer, value any) (int, error) {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return 1, err
	}
	fmt.Fprintln(stdout, string(b))
	return 0, nil
}

func stateString(state session.State, key, fallback string) string {
	if value, ok := state[key].(string); ok && value != "" {
		return value
	}
	return fallback
}

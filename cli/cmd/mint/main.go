package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"mint/internal/command/cleancmd"
	"mint/internal/command/donecmd"
	"mint/internal/command/execcmd"
	"mint/internal/command/handoffcmd"
	"mint/internal/command/hitcmd"
	"mint/internal/command/notecmd"
	"mint/internal/command/reviewcmd"
	"mint/internal/command/sessioncmd"
	"mint/internal/command/speccmd"
	"mint/internal/command/statuscmd"
	"mint/internal/command/verifycmd"
	"mint/internal/session"
)

func main() {
	code, err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if code != 0 {
		os.Exit(code)
	}
}

func run(args []string) (int, error) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stdout, "Usage: mint <command> [args]")
		return 1, nil
	}
	switch args[0] {
	case "--version", "-v", "version":
		fmt.Fprintln(os.Stdout, statuscmd.Version)
		return 0, nil
	case "done":
		doneArgs, flags, err := parseDoneArgs(args[1:])
		if err != nil {
			return 1, err
		}
		return donecmd.Run(mustGetwd(), doneArgs, flags, os.Stdout, os.Stderr)
	case "exec":
		execArgs, flags, err := parseExecArgs(args[1:])
		if err != nil {
			return 1, err
		}
		return execcmd.Run(mustGetwd(), execArgs, flags, os.Stdout, os.Stderr)
	case "review":
		reviewArgs, flags, err := parseReviewArgs(args[1:])
		if err != nil {
			return 1, err
		}
		return reviewcmd.Run(reviewArgs, flags, os.Stdout, os.Stderr)
	case "handoff":
		handoffArgs, flags, err := parseHandoffArgs(args[1:])
		if err != nil {
			return 1, err
		}
		return handoffcmd.Run(mustGetwd(), handoffArgs, flags, os.Stdout)
	case "hit":
		hitArgs, flags, err := parseHitArgs(args[1:])
		if err != nil {
			return 1, err
		}
		root := mustGetwd()
		if flags.Session == "" {
			flags.Session = session.ReadCapturedID(root)
		}
		return hitcmd.Run(root, hitArgs, flags, time.Now(), os.Stdout)
	case "note":
		noteArgs, flags, err := parseNoteArgs(args[1:])
		if err != nil {
			return 1, err
		}
		return notecmd.Run(mustGetwd(), noteArgs, flags, time.Now(), os.Stdout)
	case "clean":
		flags, err := parseCleanArgs(args[1:])
		if err != nil {
			return 1, err
		}
		return cleancmd.Run(mustGetwd(), flags, os.Stdout)
	case "status":
		if len(args) > 1 {
			return 1, fmt.Errorf("unknown status argument: %s", args[1])
		}
		return statuscmd.Run(mustGetwd(), os.Stdout)
	case "session":
		sessionArgs, flags, err := parseSessionArgs(args[1:])
		if err != nil {
			return 1, err
		}
		return sessioncmd.Run(mustGetwd(), sessionArgs, flags, os.Stdout)
	case "spec":
		specArgs, flags, err := parseSpecArgs(args[1:])
		if err != nil {
			return 1, err
		}
		return speccmd.Run(mustGetwd(), specArgs, flags, os.Stdout, os.Stderr)
	case "verify":
		verifyArgs, flags, err := parseVerifyArgs(args[1:])
		if err != nil {
			return 1, err
		}
		return verifycmd.Run(mustGetwd(), verifyArgs, flags, os.Stdout)
	default:
		fmt.Fprintln(os.Stdout, "Usage: mint <command> [args]")
		return 1, nil
	}
}

func parseCleanArgs(args []string) (cleancmd.Flags, error) {
	var flags cleancmd.Flags
	for _, arg := range args {
		switch arg {
		case "--yes":
			flags.Yes = true
		default:
			return flags, fmt.Errorf("unknown clean argument: %s", arg)
		}
	}
	return flags, nil
}

func parseHandoffArgs(args []string) ([]string, handoffcmd.Flags, error) {
	var flags handoffcmd.Flags
	out := []string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--session":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--session requires a value")
			}
			flags.Session = args[i+1]
			i++
		case "--note":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--note requires a value")
			}
			flags.Notes = append(flags.Notes, args[i+1])
			i++
		default:
			out = append(out, args[i])
		}
	}
	return out, flags, nil
}

func parseReviewArgs(args []string) ([]string, reviewcmd.Flags, error) {
	var flags reviewcmd.Flags
	out := []string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--list":
			flags.List = true
		case arg == "--focus":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--focus requires a value")
			}
			flags.Focus = args[i+1]
			i++
		case strings.HasPrefix(arg, "--"):
			flags.Lens = strings.TrimPrefix(arg, "--")
		default:
			out = append(out, arg)
		}
	}
	return out, flags, nil
}

func parseDoneArgs(args []string) ([]string, donecmd.Flags, error) {
	var flags donecmd.Flags
	out := []string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--verdict":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--verdict requires a value")
			}
			flags.Verdict = args[i+1]
			i++
		case "--terminal":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--terminal requires a value")
			}
			flags.Terminal = args[i+1]
			i++
		case "--session":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--session requires a value")
			}
			flags.Session = args[i+1]
			i++
		case "--spec":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--spec requires a value")
			}
			flags.Spec = args[i+1]
			i++
		case "--base":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--base requires a value")
			}
			flags.Base = args[i+1]
			i++
		case "--json":
			flags.JSON = true
		default:
			out = append(out, args[i])
		}
	}
	return out, flags, nil
}

func parseVerifyArgs(args []string) ([]string, verifycmd.Flags, error) {
	var flags verifycmd.Flags
	out := []string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--session":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--session requires a value")
			}
			flags.Session = args[i+1]
			i++
		case "--spec":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--spec requires a value")
			}
			flags.Spec = args[i+1]
			i++
		default:
			out = append(out, args[i])
		}
	}
	return out, flags, nil
}

func parseExecArgs(args []string) ([]string, execcmd.Flags, error) {
	var flags execcmd.Flags
	out := []string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--session":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--session requires a value")
			}
			flags.Session = args[i+1]
			i++
		case "--maker-engine":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--maker-engine requires a value")
			}
			flags.MakerEngine = args[i+1]
			i++
		case "--by-engine":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--by-engine requires a value")
			}
			flags.ByEngine = args[i+1]
			i++
		case "--by-vendor":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--by-vendor requires a value")
			}
			flags.ByVendor = args[i+1]
			i++
		case "--by-model":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--by-model requires a value")
			}
			flags.ByModel = args[i+1]
			i++
		case "--by-locality":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--by-locality requires a value")
			}
			flags.ByLocality = args[i+1]
			i++
		case "--by-session":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--by-session requires a value")
			}
			flags.BySession = args[i+1]
			i++
		case "--commit":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--commit requires a value")
			}
			flags.Commit = args[i+1]
			i++
		case "--review":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--review requires a value")
			}
			flags.Review = args[i+1]
			i++
		case "--":
			// Everything after -- is the checker command for `exec witness`,
			// passed through verbatim (never shell-wrapped) so mint spawns and
			// observes the exact binary. Present-but-empty ([]string{}) differs
			// from absent (nil) so witness can require the sentinel.
			flags.CheckerCmd = append([]string{}, args[i+1:]...)
			if flags.CheckerCmd == nil {
				flags.CheckerCmd = []string{}
			}
			return out, flags, nil
		default:
			out = append(out, args[i])
		}
	}
	return out, flags, nil
}

func parseHitArgs(args []string) ([]string, hitcmd.Flags, error) {
	var flags hitcmd.Flags
	out := []string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--priority":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--priority requires a value")
			}
			flags.Priority = args[i+1]
			i++
		case "--session":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--session requires a value")
			}
			flags.Session = args[i+1]
			i++
		case "--body":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--body requires a file path")
			}
			flags.Body = args[i+1]
			i++
		case "--file":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--file requires a path")
			}
			flags.Files = append(flags.Files, args[i+1])
			i++
		case "--open":
			flags.OpenOnly = true
		default:
			out = append(out, args[i])
		}
	}
	return out, flags, nil
}

func parseNoteArgs(args []string) ([]string, notecmd.Flags, error) {
	var flags notecmd.Flags
	out := []string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--body":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--body requires a file path")
			}
			flags.Body = args[i+1]
			i++
		case "--file":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--file requires a path")
			}
			flags.Files = append(flags.Files, args[i+1])
			i++
		default:
			out = append(out, args[i])
		}
	}
	return out, flags, nil
}

func parseSessionArgs(args []string) ([]string, sessioncmd.Flags, error) {
	var flags sessioncmd.Flags
	out := []string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--session":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--session requires a value")
			}
			flags.Session = args[i+1]
			i++
		case "--task":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--task requires a value")
			}
			flags.Task = args[i+1]
			i++
		case "--mode":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--mode requires a value")
			}
			flags.Mode = args[i+1]
			i++
		case "--no-commit":
			flags.NoCommit = true
		case "--commit":
			flags.Commit = true
		default:
			out = append(out, args[i])
		}
	}
	return out, flags, nil
}

func parseSpecArgs(args []string) ([]string, speccmd.Flags, error) {
	var flags speccmd.Flags
	out := []string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--session":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--session requires a value")
			}
			flags.Session = args[i+1]
			i++
		case "--slug":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--slug requires a value")
			}
			flags.Slug = args[i+1]
			i++
		case "--goal":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--goal requires a value")
			}
			flags.Goal = args[i+1]
			i++
		case "--scope":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--scope requires a value")
			}
			flags.Scope = args[i+1]
			i++
		case "--acceptance":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--acceptance requires a value")
			}
			if flags.Acceptance == "" {
				flags.Acceptance = args[i+1]
			} else {
				flags.Acceptance += "\n" + args[i+1]
			}
			i++
		case "--steps":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--steps requires a value")
			}
			flags.Steps = args[i+1]
			i++
		case "--commit":
			if i+1 >= len(args) {
				return nil, flags, fmt.Errorf("--commit requires a value")
			}
			flags.Commit = args[i+1]
			i++
		default:
			out = append(out, args[i])
		}
	}
	return out, flags, nil
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return wd
}

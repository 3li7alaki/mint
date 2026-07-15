package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"mint/internal/command/cleancmd"
	"mint/internal/command/donecmd"
	"mint/internal/command/execcmd"
	"mint/internal/command/notecmd"
	"mint/internal/command/receiptcmd"
	"mint/internal/command/reviewcmd"
	"mint/internal/command/speccmd"
	"mint/internal/command/statuscmd"
	"mint/internal/command/verifycmd"
	"mint/internal/statehome"
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
		return usage()
	}
	if args[0] == "--help" || args[0] == "help" {
		return help("")
	}
	if len(args) == 2 && args[1] == "--help" {
		return help(args[0])
	}
	root := mustGetwd()
	switch args[0] {
	case "--version", "-v", "version":
		fmt.Fprintln(os.Stdout, statuscmd.Version)
		return 0, nil
	case "spec":
		pos, flags, err := parseSpecArgs(args[1:])
		if err != nil {
			return 1, err
		}
		return speccmd.Run(root, pos, flags, os.Stdout, os.Stderr)
	case "exec":
		pos, flags, err := parseExecArgs(args[1:])
		if err != nil {
			return 1, err
		}
		return execcmd.Run(root, pos, flags, os.Stdout, os.Stderr)
	case "verify":
		pos, flags, err := parseVerifyArgs(args[1:])
		if err != nil {
			return 1, err
		}
		return verifycmd.Run(root, pos, flags, os.Stdout)
	case "done":
		pos, flags, err := parseDoneArgs(args[1:])
		if err != nil {
			return 1, err
		}
		return donecmd.Run(root, pos, flags, os.Stdout, os.Stderr)
	case "status":
		flags := statuscmd.Flags{}
		for _, arg := range args[1:] {
			if arg != "--json" {
				return 1, fmt.Errorf("unknown status argument: %s", arg)
			}
			flags.JSON = true
		}
		return statuscmd.Run(root, flags, os.Stdout)
	case "receipt":
		pos, flags, err := parseReceiptArgs(args[1:])
		if err != nil {
			return 1, err
		}
		return receiptcmd.Run(root, pos, flags, os.Stdout)
	case "review":
		pos, flags, err := parseReviewArgs(args[1:])
		if err != nil {
			return 1, err
		}
		return reviewcmd.Run(pos, flags, os.Stdout, os.Stderr)
	case "note":
		pos, flags, err := parseNoteArgs(args[1:])
		if err != nil {
			return 1, err
		}
		return notecmd.Run(root, pos, flags, time.Now(), os.Stdout)
	case "clean":
		flags := cleancmd.Flags{}
		for _, arg := range args[1:] {
			if arg != "--yes" {
				return 1, fmt.Errorf("unknown clean argument: %s", arg)
			}
			flags.Yes = true
		}
		return cleancmd.Run(root, flags, os.Stdout)
	default:
		return usage()
	}
}

func parseSpecArgs(args []string) ([]string, speccmd.Flags, error) {
	flags := speccmd.Flags{Gates: map[string]string{}}
	var out []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--gate" {
			value, err := next(args, &i, arg)
			if err != nil {
				return nil, flags, err
			}
			label, command, ok := strings.Cut(value, ":")
			if !ok || strings.TrimSpace(label) == "" || strings.TrimSpace(command) == "" {
				return nil, flags, fmt.Errorf("--gate requires 'label: command'")
			}
			flags.Gates[strings.TrimSpace(label)] = strings.TrimSpace(command)
			continue
		}
		if arg == "--reviews" {
			value, err := next(args, &i, arg)
			if err != nil {
				return nil, flags, err
			}
			for _, lens := range strings.Split(value, ",") {
				if lens = strings.TrimSpace(lens); lens != "" {
					flags.Reviews = append(flags.Reviews, lens)
				}
			}
			continue
		}
		var target *string
		switch arg {
		case "--slug":
			target = &flags.Slug
		case "--goal":
			target = &flags.Goal
		case "--scope":
			target = &flags.Scope
		case "--acceptance":
			target = &flags.Acceptance
		case "--steps":
			target = &flags.Steps
		case "--commit":
			target = &flags.Commit
		case "--parent-system":
			target = &flags.ParentSystem
		case "--parent-id":
			target = &flags.ParentID
		case "--parent-url":
			target = &flags.ParentURL
		}
		if target != nil {
			value, err := next(args, &i, arg)
			if err != nil {
				return nil, flags, err
			}
			if arg == "--acceptance" && *target != "" {
				*target += "\n" + value
			} else {
				*target = value
			}
			continue
		}
		if strings.HasPrefix(arg, "--") {
			return nil, flags, fmt.Errorf("unknown spec flag: %s", arg)
		}
		out = append(out, arg)
	}
	return out, flags, nil
}

func parseExecArgs(args []string) ([]string, execcmd.Flags, error) {
	var flags execcmd.Flags
	var out []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		var target *string
		switch arg {
		case "--attempt":
			target = &flags.Attempt
		case "--executor":
			target = &flags.Executor
		case "--vendor":
			target = &flags.Vendor
		case "--model":
			target = &flags.Model
		case "--locality":
			target = &flags.Locality
		case "--execution-ref":
			target = &flags.ExecutionRef
		case "--observed-by":
			target = &flags.ObservedBy
		case "--attestation":
			target = &flags.Attestation
		case "--commit":
			target = &flags.Commit
		}
		if target != nil {
			value, err := next(args, &i, arg)
			if err != nil {
				return nil, flags, err
			}
			*target = value
			continue
		}
		if strings.HasPrefix(arg, "--") {
			return nil, flags, fmt.Errorf("unknown exec flag: %s", arg)
		}
		out = append(out, arg)
	}
	return out, flags, nil
}

func parseDoneArgs(args []string) ([]string, donecmd.Flags, error) {
	var flags donecmd.Flags
	var out []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--json" {
			flags.JSON = true
			continue
		}
		var target *string
		switch arg {
		case "--attempt":
			target = &flags.Attempt
		case "--verdict":
			target = &flags.Verdict
		case "--terminal":
			target = &flags.Terminal
		case "--spec":
			target = &flags.Spec
		case "--base":
			target = &flags.Base
		}
		if target != nil {
			value, err := next(args, &i, arg)
			if err != nil {
				return nil, flags, err
			}
			*target = value
			continue
		}
		if strings.HasPrefix(arg, "--") {
			return nil, flags, fmt.Errorf("unknown done flag: %s", arg)
		}
		out = append(out, arg)
	}
	return out, flags, nil
}

func parseVerifyArgs(args []string) ([]string, verifycmd.Flags, error) {
	var flags verifycmd.Flags
	var out []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--json" {
			flags.JSON = true
			continue
		}
		var target *string
		if arg == "--attempt" {
			target = &flags.Attempt
		}
		if arg == "--spec" {
			target = &flags.Spec
		}
		if target != nil {
			value, err := next(args, &i, arg)
			if err != nil {
				return nil, flags, err
			}
			*target = value
			continue
		}
		if strings.HasPrefix(arg, "--") {
			return nil, flags, fmt.Errorf("unknown verify flag: %s", arg)
		}
		out = append(out, arg)
	}
	return out, flags, nil
}

func parseReceiptArgs(args []string) ([]string, receiptcmd.Flags, error) {
	var flags receiptcmd.Flags
	var out []string
	for _, arg := range args {
		if arg == "--json" {
			flags.JSON = true
		} else if strings.HasPrefix(arg, "--") {
			return nil, flags, fmt.Errorf("unknown receipt flag: %s", arg)
		} else {
			out = append(out, arg)
		}
	}
	return out, flags, nil
}
func parseReviewArgs(args []string) ([]string, reviewcmd.Flags, error) {
	var flags reviewcmd.Flags
	var out []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--list" {
			flags.List = true
			continue
		}
		if arg == "--focus" {
			value, err := next(args, &i, arg)
			if err != nil {
				return nil, flags, err
			}
			flags.Focus = value
			continue
		}
		if strings.HasPrefix(arg, "--") {
			flags.Lens = strings.TrimPrefix(arg, "--")
		} else {
			out = append(out, arg)
		}
	}
	return out, flags, nil
}
func parseNoteArgs(args []string) ([]string, notecmd.Flags, error) {
	var flags notecmd.Flags
	var out []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--body" {
			value, err := next(args, &i, arg)
			if err != nil {
				return nil, flags, err
			}
			flags.Body = value
			continue
		}
		if arg == "--file" {
			value, err := next(args, &i, arg)
			if err != nil {
				return nil, flags, err
			}
			flags.Files = append(flags.Files, value)
			continue
		}
		if strings.HasPrefix(arg, "--") {
			return nil, flags, fmt.Errorf("unknown note flag: %s", arg)
		}
		out = append(out, arg)
	}
	return out, flags, nil
}

func next(args []string, i *int, flag string) (string, error) {
	if *i+1 >= len(args) {
		return "", fmt.Errorf("%s requires a value", flag)
	}
	*i++
	return args[*i], nil
}
func usage() (int, error) {
	fmt.Fprintln(os.Stdout, "Usage: mint spec|exec|verify|review|done|status|receipt|note|clean")
	return 1, nil
}

func help(command string) (int, error) {
	lines := map[string]string{
		"":        "Usage: mint <command> [args]\nCommands: spec, exec, verify, review, done, status, receipt, note, clean",
		"spec":    "Usage: mint spec new \"<title>\" [--slug <slug>] --goal <text> --scope <paths> --acceptance <EARS> [--gate 'label: command'] [--reviews <list>]\n       mint spec set|validate|scope <spec-path>",
		"exec":    "Usage: mint exec init <slug> <spec-id> [--attempt <id>] --executor <name> --vendor <name> --model <name> --locality <local|remote> [--execution-ref <ref>]\n       mint exec record-review|record-gate|set-status|show ...",
		"verify":  "Usage: mint verify <slug> <spec-id> [--attempt <id>] [--spec <path>]",
		"review":  "Usage: mint review --list | mint review --<lens> [--focus <text>]",
		"done":    "Usage: mint done <slug> <spec-id> [--attempt <id>] [--verdict <path>] [--terminal <state>] [--base <ref>] [--json]",
		"status":  "Usage: mint status [--json]",
		"receipt": "Usage: mint receipt show|verify <path> [--json]",
		"note":    "Usage: mint note add <topic> <text> [--body <path>] [--file <path>] | show <topic> | list",
		"clean":   "Usage: mint clean [--yes]",
	}
	text, ok := lines[command]
	if !ok {
		return 1, fmt.Errorf("unknown command: %s", command)
	}
	fmt.Fprintln(os.Stdout, text)
	return 0, nil
}
func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return statehome.Resolve(wd).WorktreeRoot
}

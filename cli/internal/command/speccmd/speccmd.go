package speccmd

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mint/internal/atomic"
	"mint/internal/gitignore"
	"mint/internal/session"
	"mint/internal/specschema"
)

//go:embed template.xml
var templateText string

type Flags struct {
	Session string
	Slug    string
}

type NewResult struct {
	SpecPath string `json:"specPath"`
	Branch   string `json:"branch"`
	ID       string `json:"id"`
}

func Run(root string, args []string, flags Flags, stdout, stderr io.Writer) (int, error) {
	if len(args) == 0 {
		fmt.Fprintln(stdout, "  Usage: mint spec validate|scope <spec-path>")
		return 1, nil
	}
	switch args[0] {
	case "new":
		if len(args) < 2 {
			return 1, fmt.Errorf("Usage: mint spec new \"<title>\" [--session <id>] [--slug <slug>]")
		}
		result, err := New(root, args[1], flags)
		if err != nil {
			return 1, err
		}
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return 1, err
		}
		fmt.Fprintln(stdout, string(b))
		return 0, nil
	case "validate":
		if len(args) < 2 {
			return 1, fmt.Errorf("Usage: mint spec validate <spec-path>")
		}
		xml, err := readSpec(root, args[1])
		if err != nil {
			return 1, err
		}
		result := specschema.Validate(xml)
		for _, e := range result.Errors {
			fmt.Fprintf(stderr, "  error  %s\n", e)
		}
		for _, w := range result.Warnings {
			fmt.Fprintf(stderr, "  warn   %s\n", w)
		}
		if result.Valid && len(result.Warnings) == 0 {
			fmt.Fprintf(stdout, "  ok %s is valid\n", args[1])
		}
		if len(result.Errors) > 0 || len(result.Warnings) > 0 {
			return 1, nil
		}
		return 0, nil
	case "scope":
		if len(args) < 2 {
			return 1, fmt.Errorf("Usage: mint spec scope <spec-path>")
		}
		xml, err := readSpec(root, args[1])
		if err != nil {
			return 1, err
		}
		for _, p := range specschema.ResolveCanModify(xml) {
			fmt.Fprintln(stdout, p)
		}
		return 0, nil
	default:
		fmt.Fprintln(stdout, "  Usage: mint spec validate|scope <spec-path>")
		return 1, nil
	}
}

func New(root, title string, flags Flags) (NewResult, error) {
	sessionID := strings.TrimSpace(flags.Session)
	if sessionID == "" {
		sessionID = session.ReadCapturedID(root)
	}
	if sessionID == "" {
		id, err := session.GenerateID(time.Now())
		if err != nil {
			return NewResult{}, err
		}
		sessionID = id
	}
	slug := flags.Slug
	if slug == "" {
		slug = specschema.Slugify(title)
	}
	if slug == "" {
		return NewResult{}, fmt.Errorf("Could not derive a slug from the title - pass --slug")
	}

	dir := filepath.Join(root, ".mint", "tasks", sessionID, slug)
	id := specschema.AllocateSpecID(dir)
	specPath := filepath.Join(dir, id+"-"+slug+".xml")
	state, _ := session.ReadState(root, sessionID)
	fields := specschema.Fields{
		ID:      id,
		Title:   title,
		Gates:   gatesFromState(state),
		Reviews: reviewsFromState(state),
	}
	if _, err := gitignore.Ensure(root, nil); err != nil {
		return NewResult{}, err
	}
	if err := atomic.WriteString(specPath, specschema.Scaffold(templateText, fields)); err != nil {
		return NewResult{}, err
	}
	return NewResult{SpecPath: specPath, Branch: "feat/" + slug, ID: id}, nil
}

func readSpec(root, specPath string) (string, error) {
	abs := specPath
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(root, specPath)
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", fmt.Errorf("Spec not found: %s", specPath)
	}
	return string(b), nil
}

func gatesFromState(state session.State) map[string]string {
	raw, ok := state["gates"]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case map[string]string:
		return v
	case map[string]any:
		out := map[string]string{}
		for k, value := range v {
			if s, ok := value.(string); ok && strings.TrimSpace(k) != "" && strings.TrimSpace(s) != "" {
				out[k] = s
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return nil
}

func reviewsFromState(state session.State) []string {
	raw, ok := state["reviews"]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		out := []string{}
		for _, value := range v {
			if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		parts := strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' || r == '\n' })
		out := []string{}
		for _, part := range parts {
			if strings.TrimSpace(part) != "" {
				out = append(out, strings.TrimSpace(part))
			}
		}
		return out
	}
	return nil
}

package speccmd

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"mint/internal/atomic"
	"mint/internal/execstate"
	"mint/internal/gitignore"
	"mint/internal/session"
	"mint/internal/specschema"
)

//go:embed template.xml
var templateText string

type Flags struct {
	Session    string
	Slug       string
	Goal       string
	Scope      string
	Acceptance string
	Steps      string
	Commit     string
}

type NewResult struct {
	SpecPath string `json:"specPath"`
	Branch   string `json:"branch"`
	ID       string `json:"id"`
}

func Run(root string, args []string, flags Flags, stdout, stderr io.Writer) (int, error) {
	if len(args) == 0 {
		printUsage(stdout)
		return 1, nil
	}
	switch args[0] {
	case "new":
		if len(args) < 2 {
			return 1, fmt.Errorf("Usage: mint spec new \"<title>\" [--session <id>] [--slug <slug>] [--goal <text>] [--scope <paths>] [--acceptance <text>] [--steps <text>] [--commit <text>]")
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
	case "set":
		if len(args) < 2 {
			return 1, fmt.Errorf("Usage: mint spec set <spec-path> [--goal <text>] [--scope <paths>] [--acceptance <text>] [--steps <text>] [--commit <text>]")
		}
		if err := Set(root, args[1], flags); err != nil {
			return 1, err
		}
		fmt.Fprintf(stdout, "  ok %s updated\n", args[1])
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
		printUsage(stdout)
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
	if !execstate.IsLiteralSegment(slug) {
		return NewResult{}, fmt.Errorf("invalid slug %q - must be a single path segment (no empty, '.', '..', or path separator)", slug)
	}
	if !execstate.IsLiteralSegment(sessionID) {
		return NewResult{}, fmt.Errorf("invalid session %q - must be a single path segment (no empty, '.', '..', or path separator)", sessionID)
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
	scaffolded := applyFieldFlags(specschema.Scaffold(templateText, fields), flags)
	if err := atomic.WriteString(specPath, scaffolded); err != nil {
		return NewResult{}, err
	}
	return NewResult{SpecPath: specPath, Branch: "feat/" + slug, ID: id}, nil
}

func Set(root, specPath string, flags Flags) error {
	abs := resolveSpecPath(root, specPath)
	b, err := os.ReadFile(abs)
	if err != nil {
		return fmt.Errorf("Spec not found: %s", specPath)
	}
	updated := applyFieldFlags(string(b), flags)
	if err := atomic.WriteString(abs, updated); err != nil {
		return err
	}
	result := specschema.Validate(updated)
	if len(result.Errors) > 0 || len(result.Warnings) > 0 {
		var lines []string
		for _, e := range result.Errors {
			lines = append(lines, "error  "+e)
		}
		for _, w := range result.Warnings {
			lines = append(lines, "warn   "+w)
		}
		return fmt.Errorf("spec validation failed:\n  %s", strings.Join(lines, "\n  "))
	}
	return nil
}

func readSpec(root, specPath string) (string, error) {
	abs := resolveSpecPath(root, specPath)
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", fmt.Errorf("Spec not found: %s", specPath)
	}
	return string(b), nil
}

func resolveSpecPath(root, specPath string) string {
	if filepath.IsAbs(specPath) {
		return specPath
	}
	return filepath.Join(root, specPath)
}

func applyFieldFlags(xml string, flags Flags) string {
	out := xml
	if flags.Goal != "" {
		out = replaceElementContent(out, "goal", escapeXML(flags.Goal))
	}
	if flags.Scope != "" {
		out = replaceElementContent(out, "can-modify", renderScope(flags.Scope))
	}
	if flags.Steps != "" {
		out = replaceElementContent(out, "steps", escapeXML(flags.Steps))
	}
	if flags.Acceptance != "" {
		out = replaceElementContent(out, "acceptance", escapeXML(flags.Acceptance))
	}
	if flags.Commit != "" {
		out = replaceElementContent(out, "commit", escapeXML(flags.Commit))
	}
	return out
}

func replaceElementContent(xml, name, content string) string {
	re := regexp.MustCompile(`(<` + regexp.QuoteMeta(name) + `>)[\s\S]*?(</` + regexp.QuoteMeta(name) + `>)`)
	m := re.FindStringSubmatchIndex(xml)
	if m == nil {
		return xml
	}
	return xml[:m[3]] + content + xml[m[4]:]
}

func renderScope(scope string) string {
	var paths []string
	for _, path := range strings.Split(scope, ",") {
		if trimmed := strings.TrimSpace(path); trimmed != "" {
			paths = append(paths, escapeXML(trimmed))
		}
	}
	return strings.Join(paths, ",")
}

func escapeXML(s string) string {
	return html.EscapeString(s)
}

func printUsage(stdout io.Writer) {
	fmt.Fprintln(stdout, "  Usage: mint spec new \"<title>\" [--session <id>] [--slug <slug>] [--goal <text>] [--scope <paths>] [--acceptance <text>] [--steps <text>] [--commit <text>]")
	fmt.Fprintln(stdout, "         mint spec set <spec-path> [--goal <text>] [--scope <paths>] [--acceptance <text>] [--steps <text>] [--commit <text>]")
	fmt.Fprintln(stdout, "         mint spec validate|scope <spec-path>")
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

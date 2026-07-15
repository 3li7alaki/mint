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

	"mint/internal/specschema"
	"mint/internal/statehome"
	"mint/internal/unitstore"
)

//go:embed template.xml
var templateText string

type Flags struct {
	Slug         string
	Goal         string
	Scope        string
	Acceptance   string
	Steps        string
	Commit       string
	Gates        map[string]string
	Reviews      []string
	ParentSystem string
	ParentID     string
	ParentURL    string
}

type NewResult struct {
	SchemaVersion int    `json:"schemaVersion"`
	SpecPath      string `json:"specPath"`
	Slug          string `json:"slug"`
	ID            string `json:"id"`
}

func Run(root string, args []string, flags Flags, stdout, stderr io.Writer) (int, error) {
	if len(args) == 0 {
		printUsage(stdout)
		return 1, nil
	}
	switch args[0] {
	case "new":
		if len(args) < 2 {
			return 1, fmt.Errorf("Usage: mint spec new \"<title>\" [--slug <slug>] [--goal <text>] [--scope <paths>] [--acceptance <text>] [--gate 'label: command'] [--reviews <list>] [--parent-system <name> --parent-id <id>]")
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
	slug := flags.Slug
	if slug == "" {
		slug = specschema.Slugify(title)
	}
	if slug == "" {
		return NewResult{}, fmt.Errorf("Could not derive a slug from the title - pass --slug")
	}
	if !unitstore.ValidSegment(slug) {
		return NewResult{}, fmt.Errorf("invalid slug %q - must be a single path segment (no empty, '.', '..', or path separator)", slug)
	}
	if err := unitstore.Ensure(root); err != nil {
		return NewResult{}, err
	}
	dir := unitstore.SpecsDir(root, slug)
	id := specschema.AllocateSpecID(dir)
	specPath := unitstore.SpecPath(root, slug, id)
	gates := flags.Gates
	reviews := flags.Reviews
	fields := specschema.Fields{
		ID:      id,
		Title:   title,
		Gates:   gates,
		Reviews: reviews,
	}
	scaffolded := applyFieldFlags(specschema.Scaffold(templateText, fields), flags)
	scaffolded = applyParent(scaffolded, flags)
	validation := specschema.Validate(scaffolded)
	if len(validation.Errors) > 0 {
		return NewResult{}, fmt.Errorf("spec validation failed: %s", strings.Join(validation.Errors, "; "))
	}
	if err := statehome.Write(specPath, []byte(scaffolded)); err != nil {
		return NewResult{}, err
	}
	return NewResult{SchemaVersion: 1, SpecPath: specPath, Slug: slug, ID: id}, nil
}

func applyParent(xml string, flags Flags) string {
	system := firstNonEmpty(flags.ParentSystem, os.Getenv("MINT_PARENT_SYSTEM"))
	id := firstNonEmpty(flags.ParentID, os.Getenv("MINT_PARENT_ID"))
	url := firstNonEmpty(flags.ParentURL, os.Getenv("MINT_PARENT_URL"))
	if system == "" && id == "" && url == "" {
		return xml
	}
	lines := []string{"  <parent>", "    <system>" + escapeXML(system) + "</system>", "    <id>" + escapeXML(id) + "</id>"}
	if url != "" {
		lines = append(lines, "    <url>"+escapeXML(url)+"</url>")
	}
	lines = append(lines, "  </parent>")
	return regexp.MustCompile(`\n?</task>\s*$`).ReplaceAllString(xml, "\n\n"+strings.Join(lines, "\n")+"\n</task>\n")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func Set(root, specPath string, flags Flags) error {
	abs := resolveSpecPath(root, specPath)
	b, err := os.ReadFile(abs)
	if err != nil {
		return fmt.Errorf("Spec not found: %s", specPath)
	}
	updated := applyFieldFlags(string(b), flags)
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
	if err := statehome.Write(abs, []byte(updated)); err != nil {
		return err
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
	fmt.Fprintln(stdout, "  Usage: mint spec new \"<title>\" [--slug <slug>] [--goal <text>] [--scope <paths>] [--acceptance <text>] [--gate 'label: command'] [--reviews <list>] [--parent-system <name> --parent-id <id>]")
	fmt.Fprintln(stdout, "         mint spec set <spec-path> [--goal <text>] [--scope <paths>] [--acceptance <text>] [--steps <text>] [--commit <text>]")
	fmt.Fprintln(stdout, "         mint spec validate|scope <spec-path>")
}

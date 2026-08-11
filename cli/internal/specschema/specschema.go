package specschema

import (
	"fmt"
	"html"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"mint/internal/acceptance"
)

var required = []string{"id", "title", "goal", "scope", "steps", "acceptance", "commit"}

type ValidationResult struct {
	Valid    bool
	Errors   []string
	Warnings []string
}

type Parent struct {
	System string `json:"system"`
	ID     string `json:"id"`
	URL    string `json:"url,omitempty"`
}

func Slugify(title string) string {
	s := strings.ToLower(title)
	s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
	s = regexp.MustCompile(`^-+|-+$`).ReplaceAllString(s, "")
	return s
}

func AllocateSpecID(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "001"
	}
	maxID := 0
	re := regexp.MustCompile(`^(\d+)(?:-|$)`)
	for _, entry := range entries {
		m := re.FindStringSubmatch(entry.Name())
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err == nil && n > maxID {
			maxID = n
		}
	}
	return fmt.Sprintf("%03d", maxID+1)
}

func Scaffold(templateText string, fields Fields) string {
	out := regexp.MustCompile(`<id>[\s\S]*?</id>`).ReplaceAllString(templateText, "<id>"+escapeXML(fields.ID)+"</id>")
	out = regexp.MustCompile(`<title>[\s\S]*?</title>`).ReplaceAllString(out, "<title>"+escapeXML(fields.Title)+"</title>")

	gatesBlock := renderGatesBlock(fields.Gates)
	var reviews []string
	for _, review := range fields.Reviews {
		review = strings.TrimSpace(review)
		if review != "" {
			reviews = append(reviews, escapeXML(review))
		}
	}
	reviewsBlock := ""
	if len(reviews) > 0 {
		reviewsBlock = "  <reviews>" + strings.Join(reviews, ", ") + "</reviews>"
	}
	var baked []string
	if gatesBlock != "" {
		baked = append(baked, gatesBlock)
	}
	if reviewsBlock != "" {
		baked = append(baked, reviewsBlock)
	}
	if len(baked) == 0 {
		return out
	}
	return regexp.MustCompile(`(\n?)(</task>\s*)$`).ReplaceAllString(out, "\n\n"+strings.Join(baked, "\n")+"\n$2")
}

type Fields struct {
	ID      string
	Title   string
	Gates   map[string]string
	Reviews []string
}

func ResolveCanModify(xml string) []string {
	scope, ok := tag(xml, "scope")
	if !ok {
		return nil
	}
	inner, ok := tag(scope, "can-modify")
	if !ok {
		return nil
	}
	pathRE := regexp.MustCompile(`<path>([\s\S]*?)</path>`)
	matches := pathRE.FindAllStringSubmatch(inner, -1)
	var paths []string
	if len(matches) > 0 {
		for _, m := range matches {
			paths = append(paths, strings.TrimSpace(m[1]))
		}
	} else {
		for _, p := range strings.Split(inner, ",") {
			paths = append(paths, strings.TrimSpace(p))
		}
	}
	return compact(paths)
}

func ResolveSpecGates(xml string) map[string]string {
	inner, ok := tag(xml, "gates")
	if !ok {
		return map[string]string{}
	}
	gates := map[string]string{}
	for _, line := range strings.Split(inner, "\n") {
		i := strings.Index(line, ":")
		if i == -1 {
			continue
		}
		label := unescapeXML(strings.TrimSpace(line[:i]))
		cmd := unescapeXML(strings.TrimSpace(line[i+1:]))
		if label != "" && cmd != "" {
			gates[label] = cmd
		}
	}
	return gates
}

func ResolveSpecReviews(xml string) []string {
	inner, ok := tag(xml, "reviews")
	if !ok {
		return nil
	}
	fields := regexp.MustCompile(`[\s,]+`).Split(inner, -1)
	for i := range fields {
		fields[i] = unescapeXML(strings.TrimSpace(fields[i]))
	}
	return compact(fields)
}

func Validate(xml string) ValidationResult {
	var result ValidationResult
	if strings.TrimSpace(xml) == "" {
		return ValidationResult{Valid: false, Errors: []string{"spec is empty"}}
	}
	if !regexp.MustCompile(`<task>[\s\S]*</task>`).MatchString(xml) {
		result.Errors = append(result.Errors, "missing root <task> element")
	}
	for _, field := range required {
		content, ok := tag(xml, field)
		if !ok {
			result.Errors = append(result.Errors, fmt.Sprintf("missing required field <%s>", field))
		} else if field != "scope" && strings.TrimSpace(content) == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("<%s> is empty", field))
		}
	}
	if scope, ok := tag(xml, "scope"); ok {
		if !regexp.MustCompile(`<can-modify>[\s\S]*?</can-modify>`).MatchString(scope) {
			result.Errors = append(result.Errors, "<scope> is missing <can-modify>")
		} else if len(ResolveCanModify(xml)) == 0 {
			result.Errors = append(result.Errors, "<can-modify> resolves to no paths")
		}
	}
	if deps, ok := tag(xml, "depends-on"); ok {
		d := strings.TrimSpace(deps)
		if d != "" && d != "none" && len(compact(strings.Split(d, ","))) == 0 {
			result.Warnings = append(result.Warnings, `<depends-on> is non-empty but lists no ids: use "none" if there are no dependencies`)
		}
	}
	if risk, ok := tag(xml, "risk"); ok {
		r := strings.TrimSpace(risk)
		if r == "" {
			result.Warnings = append(result.Warnings, "empty <risk>: omit it or use <risk>safety</risk>")
		} else if !regexp.MustCompile(`(?i)^safety$`).MatchString(r) {
			result.Warnings = append(result.Warnings, fmt.Sprintf("unrecognized <risk> value '%s': the only floor-recognized value is 'safety' (forces a cross-vendor safety review); other values are ignored by the floor", r))
		}
	}
	if parent, ok := ResolveParent(xml); ok {
		if parent.System == "" || parent.ID == "" {
			result.Errors = append(result.Errors, "<parent> requires non-empty <system> and <id>")
		}
	}
	accept := acceptance.Parse(xml)
	result.Errors = append(result.Errors, accept.Errors...)
	result.Warnings = append(result.Warnings, accept.Warnings...)
	result.Valid = len(result.Errors) == 0
	return result
}

func ResolveParent(xml string) (Parent, bool) {
	inner, ok := tag(xml, "parent")
	if !ok {
		return Parent{}, false
	}
	system, _ := tag(inner, "system")
	id, _ := tag(inner, "id")
	url, _ := tag(inner, "url")
	return Parent{
		System: unescapeXML(strings.TrimSpace(system)),
		ID:     unescapeXML(strings.TrimSpace(id)),
		URL:    unescapeXML(strings.TrimSpace(url)),
	}, true
}

func tag(xml, name string) (string, bool) {
	live := regexp.MustCompile(`<!--[\s\S]*?-->`).ReplaceAllString(xml, "")
	re := regexp.MustCompile(`<` + regexp.QuoteMeta(name) + `>([\s\S]*?)</` + regexp.QuoteMeta(name) + `>`)
	m := re.FindStringSubmatch(live)
	if m == nil {
		return "", false
	}
	return m[1], true
}

func renderGatesBlock(gates map[string]string) string {
	if len(gates) == 0 {
		return ""
	}
	keys := make([]string, 0, len(gates))
	for key := range gates {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, "    "+escapeXML(key)+": "+escapeXML(gates[key]))
	}
	return "  <gates>\n" + strings.Join(lines, "\n") + "\n  </gates>"
}

func escapeXML(s string) string {
	return html.EscapeString(s)
}

func unescapeXML(s string) string {
	return html.UnescapeString(s)
}

func compact(in []string) []string {
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

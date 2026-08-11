package acceptance

import (
	"fmt"
	"regexp"
	"strings"
)

type Criterion struct {
	Raw      string
	Pattern  string
	Trigger  *string
	System   *string
	Response string
}

type ParseResult struct {
	Criteria []Criterion
	Errors   []string
	Warnings []string
}

var (
	markerRE  = regexp.MustCompile(`^\s*(?:[-*•]\s+|\d+[.)]\s+)`)
	gateWord  = regexp.MustCompile(`(?i)\b(lint|type|types|typecheck|test|tests|coverage|cover)\b`)
	gwtRE     = regexp.MustCompile(`(?i)^GIVEN\b([\s\S]*?),?\s*WHEN\b([\s\S]*?),?\s*THEN\b([\s\S]+)$`)
	ubiqRE    = regexp.MustCompile(`(?i)^THE\s+([\s\S]*?)\s+SHALL\s+([\s\S]+)$`)
	lineSplit = regexp.MustCompile(`\r?\n`)
)

type earsPattern struct {
	pattern string
	keyword string
}

var earsPatterns = []earsPattern{
	{"event", "WHEN"},
	{"state", "WHILE"},
	{"optional", "WHERE"},
	{"unwanted", "IF"},
}

func Parse(xml string) ParseResult {
	var result ParseResult
	inner, ok := extractTag(xml, "acceptance")
	if !ok {
		return result
	}

	for _, rawLine := range lineSplit.Split(inner, -1) {
		line := stripMarker(rawLine)
		if line == "" {
			continue
		}
		if IsGateRestatement(line) {
			result.Errors = append(result.Errors, fmt.Sprintf(`acceptance restates a deterministic gate (the declared gates already enforce it): "%s"`, line))
			continue
		}
		if ears, ok := classifyEars(line); ok {
			ears.Raw = line
			result.Criteria = append(result.Criteria, ears)
			continue
		}
		result.Criteria = append(result.Criteria, Criterion{Raw: line, Pattern: "prose", Response: line})
		result.Warnings = append(result.Warnings, fmt.Sprintf(`acceptance criterion is free prose, not EARS, not machine-structured: "%s"`, line))
	}

	if len(result.Criteria) == 0 {
		result.Errors = append(result.Errors, "acceptance has no criteria")
	}
	return result
}

func IsGateRestatement(line string) bool {
	raw := stripMarker(line)
	if raw == "" {
		return false
	}
	if _, ok := classifyEars(raw); ok {
		return false
	}
	if !gateWord.MatchString(raw) {
		return false
	}

	residue := regexp.MustCompile(`(?i)\b(lint(?:ing)?|types?|typecheck(?:ing)?|tests?|coverage|cover(?:age)?)\b`).ReplaceAllString(raw, " ")
	residue = regexp.MustCompile(`(?i)\b(pass(?:e[ds])?|passing|green|clean|ok|okay|✅)\b`).ReplaceAllString(residue, " ")
	residue = regexp.MustCompile(`(?i)\b(and|or|no|not|is|are|be|been|the|a|an|with|of|for|to|in|on|must|should|shall|ensure|ensures?|verify|verifies|verified|confirm|confirms?|make|sure|succeed(?:s|ed)?|require[ds]?|required|run|runs|ran|running|should[ds]?|errors?|threshold)\b`).ReplaceAllString(residue, " ")
	residue = regexp.MustCompile(`\d+(?:\.\d+)?\s*%?`).ReplaceAllString(residue, " ")
	residue = regexp.MustCompile(`[≥≤<>=]+`).ReplaceAllString(residue, " ")
	residue = regexp.MustCompile(`[^\p{L}\p{N}]+`).ReplaceAllString(residue, " ")
	return strings.TrimSpace(residue) == ""
}

func stripMarker(line string) string {
	return strings.TrimSpace(markerRE.ReplaceAllString(line, ""))
}

func classifyEars(text string) (Criterion, bool) {
	if m := gwtRE.FindStringSubmatch(text); m != nil {
		return Criterion{Pattern: "gwt", Trigger: strPtr(strings.TrimSpace(m[2])), Response: strings.TrimSpace(m[3])}, true
	}
	if m := ubiqRE.FindStringSubmatch(text); m != nil {
		return Criterion{Pattern: "ubiquitous", System: strPtr(strings.TrimSpace(m[1])), Response: strings.TrimSpace(m[2])}, true
	}
	for _, ep := range earsPatterns {
		re := regexp.MustCompile(`(?i)^` + ep.keyword + `\b([\s\S]*?),\s*(?:THEN\s+)?THE\s+([\s\S]*?)\s+SHALL\s+([\s\S]+)$`)
		if m := re.FindStringSubmatch(text); m != nil {
			return Criterion{Pattern: ep.pattern, Trigger: strPtr(strings.TrimSpace(m[1])), System: strPtr(strings.TrimSpace(m[2])), Response: strings.TrimSpace(m[3])}, true
		}
	}
	return Criterion{}, false
}

func extractTag(xml, name string) (string, bool) {
	re := regexp.MustCompile(`<` + regexp.QuoteMeta(name) + `>([\s\S]*?)</` + regexp.QuoteMeta(name) + `>`)
	m := re.FindStringSubmatch(xml)
	if m == nil {
		return "", false
	}
	return m[1], true
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

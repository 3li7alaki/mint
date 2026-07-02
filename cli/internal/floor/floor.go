package floor

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"mint/internal/docclassify"
	"mint/internal/engine"
	"mint/internal/scopematch"
	"mint/internal/specschema"
)

var TerminalStates = []string{
	"done-verified",
	"budget-exhausted",
	"stuck-escalated",
	"external-stop",
}

type Input struct {
	Spec            string
	ChangedFiles    []string
	Patch           string
	Gates           *GateResult
	Verdict         map[string]any
	MakerEngine     string
	MakerSession    string
	TerminalState   string
	RequiredReviews []string
	Reviews         map[string]string
}

type GateResult struct {
	OK bool
}

type ClauseResult struct {
	Clause int     `json:"clause"`
	Name   string  `json:"name"`
	Pass   bool    `json:"pass"`
	Reason *string `json:"reason"`
}

type Result struct {
	Pass    bool           `json:"pass"`
	Clauses []ClauseResult `json:"clauses"`
	Failed  []int          `json:"failed"`
}

type clause struct {
	n    int
	name string
	run  func(Input) clauseVerdict
}

type clauseVerdict struct {
	pass   bool
	reason string
}

var clauses = []clause{
	{1, "verifiable-completion", clauseVerifiableCompletion},
	{2, "maker-not-checker", clauseMakerChecker},
	{3, "safety-carve-out", clauseSafetyCarveOut},
	{4, "anti-gaming", clauseAntiGaming},
	{5, "scope-respected", clauseScope},
	{6, "bounded-terminating", clauseTerminating},
	{7, "floor-gated-action", clauseFloorGated},
}

func Enforce(input Input) Result {
	results := make([]ClauseResult, 0, len(clauses))
	var failed []int
	for _, c := range clauses {
		v := safeRun(c.run, input)
		var reason *string
		if !v.pass {
			reason = &v.reason
			failed = append(failed, c.n)
		}
		results = append(results, ClauseResult{
			Clause: c.n,
			Name:   c.name,
			Pass:   v.pass,
			Reason: reason,
		})
	}
	return Result{Pass: len(failed) == 0, Clauses: results, Failed: failed}
}

func IsSafetyTier(input Input) bool {
	files := input.ChangedFiles
	haystack := strings.ToLower(input.Patch)
	for _, f := range files {
		haystack += "\n" + strings.ToLower(f)
	}
	for _, term := range safetyTerms {
		if strings.Contains(haystack, term) {
			return true
		}
	}
	for _, marker := range safetySinkMarkers {
		if strings.Contains(haystack, marker) {
			return true
		}
	}
	for _, file := range files {
		for _, segment := range strings.Split(strings.ToLower(file), "/") {
			for _, pattern := range safetyPathPatterns {
				if strings.Contains(segment, pattern) {
					return true
				}
			}
		}
	}
	return regexp.MustCompile(`(?i)<risk>\s*safety\s*</risk>`).MatchString(input.Spec)
}

func clauseVerifiableCompletion(input Input) clauseVerdict {
	if input.Gates == nil || !input.Gates.OK {
		return fail("gates did not run green (gates.ok !== true)")
	}
	if input.Verdict == nil {
		return fail("no independent acceptance verdict attached")
	}
	if accepted, _ := input.Verdict["accepted"].(bool); !accepted {
		return fail("attached verdict did not accept (verdict.accepted !== true)")
	}
	if isLogicTier(input) && !hasSubstantiveAttestation(input.Verdict, "adversarialReviewed", "adversarialReason") {
		return fail("this is a logic/trust diff (control-flow/arithmetic/parsing, or safety-tier) — a code-read acceptance verdict is not enough; it needs an attested INDEPENDENT adversarial check that someone tried to BREAK the behavior; attach an adversarial review to the verdict (adversarialReviewed: true + a substantive adversarialReason) and re-attach")
	}
	if reason := missingDeclaredReviews(input); reason != "" {
		return fail(reason)
	}
	return pass()
}

func clauseMakerChecker(input Input) clauseVerdict {
	if input.Verdict == nil {
		return fail("no verdict to check provenance on — attach an independent acceptance verdict carrying byEngine/bySession")
	}
	checkerEngine, okEngine := input.Verdict["byEngine"].(string)
	checkerSession, okSession := input.Verdict["bySession"].(string)
	if !okEngine || !okSession || strings.TrimSpace(checkerEngine) == "" || strings.TrimSpace(checkerSession) == "" {
		return fail("verdict lacks byEngine/bySession provenance — cannot verify independence; re-attach a verdict that records which engine+session produced it")
	}
	if strings.TrimSpace(input.MakerEngine) == "" || strings.TrimSpace(input.MakerSession) == "" {
		return fail("maker provenance unknown (makerEngine/makerSession absent) — cannot verify the checker differs from the maker; attach maker provenance (the engine+session that produced the diff)")
	}
	if !engine.IsKnown(checkerEngine) {
		return fail(fmt.Sprintf("verdict.byEngine (%s) is not a recognized engine — must be one of: %s (mint validates engine identity against its own registry, not caller-supplied strings)", checkerEngine, strings.Join(engine.Keys(), ", ")))
	}
	if !engine.IsKnown(input.MakerEngine) {
		return fail(fmt.Sprintf("makerEngine (%s) is not a recognized engine — cannot verify independence against an unknown maker engine; supply the real engine that produced the diff (%s)", input.MakerEngine, strings.Join(engine.Keys(), "|")))
	}

	if IsSafetyTier(input) {
		if !sameProvenance(checkerEngine, input.MakerEngine) {
			return pass()
		}
		return fail(fmt.Sprintf("safety-tier diff requires a verdict from a DIFFERENT engine; verdict.byEngine (%s) === makerEngine (%s) — re-run acceptance on a different engine and re-attach the verdict", checkerEngine, input.MakerEngine))
	}
	if !sameProvenance(checkerEngine, input.MakerEngine) {
		return pass()
	}
	if !sameProvenance(checkerSession, input.MakerSession) {
		if !engine.IsStrictSession(checkerSession) {
			return fail(fmt.Sprintf("verdict.bySession (%s) is not a valid session id (not an ASCII session/ref charset) — a non-ASCII/confusable session cannot establish a fresh session; attach a verdict from a DIFFERENT engine, or re-run acceptance in a real fresh session and re-attach", checkerSession))
		}
		return pass()
	}
	return fail(fmt.Sprintf("verdict came from the maker's own engine+session (%s/%s) — re-run acceptance in a fresh session (or on a different engine) and re-attach", input.MakerEngine, input.MakerSession))
}

func clauseSafetyCarveOut(input Input) clauseVerdict {
	if !IsSafetyTier(input) {
		return pass()
	}
	if hasSubstantiveAttestation(input.Verdict, "safetyReviewed", "safetyReason") {
		return pass()
	}
	return fail("this diff touches the safety carve-out (security/trust-boundary/data-loss) — it cannot reach done without an attested safety review; attach an INDEPENDENT safety review to the verdict (safetyReviewed: true + a substantive safetyReason) and re-attach")
}

func clauseAntiGaming(input Input) clauseVerdict {
	if strings.TrimSpace(input.Patch) == "" {
		return pass()
	}
	signals := antiGamingSignals(input.Patch)
	if len(signals) > 0 {
		summary := signalSummary(signals)
		if hasSubstantiveAttestation(input.Verdict, "tamperingReviewed", "tamperingReason") {
			return clauseVerdict{pass: true, reason: fmt.Sprintf("tampering flagged but acknowledged by independent review: %s [signals: %s]", strings.TrimSpace(input.Verdict["tamperingReason"].(string)), summary)}
		}
		return fail("verifier-tampering detected: " + summary)
	}
	return pass()
}

func clauseScope(input Input) clauseVerdict {
	allowed := specschema.ResolveCanModify(input.Spec)
	var offending []string
	for _, file := range input.ChangedFiles {
		ok := false
		for _, lane := range allowed {
			if scopematch.MatchesAllowed(file, lane) {
				ok = true
				break
			}
		}
		if !ok {
			offending = append(offending, file)
		}
	}
	if len(offending) > 0 {
		allowedText := strings.Join(allowed, ", ")
		if allowedText == "" {
			allowedText = "(none)"
		}
		return fail(fmt.Sprintf("out-of-lane file(s): %s — allowed: %s", strings.Join(offending, ", "), allowedText))
	}
	return pass()
}

func clauseTerminating(input Input) clauseVerdict {
	for _, state := range TerminalStates {
		if input.TerminalState == state {
			return pass()
		}
	}
	return fail(fmt.Sprintf("terminalState %q is not one of %s", input.TerminalState, strings.Join(TerminalStates, "|")))
}

func clauseFloorGated(Input) clauseVerdict {
	return clauseVerdict{pass: true, reason: "gated-by-this-kernel"}
}

func missingDeclaredReviews(input Input) string {
	var required []string
	for _, r := range input.RequiredReviews {
		r = strings.TrimSpace(r)
		if r != "" {
			required = append(required, r)
		}
	}
	if len(required) == 0 {
		return ""
	}
	if len(input.ChangedFiles) > 0 {
		docsOnly := true
		for _, file := range input.ChangedFiles {
			if !docclassify.IsDocPath(file) {
				docsOnly = false
				break
			}
		}
		if docsOnly {
			return ""
		}
	}

	var unmet []string
	var lenses []string
	for _, lens := range required {
		verdict, ok := input.Reviews[lens]
		if !ok {
			unmet = append(unmet, lens+" (not run)")
			lenses = append(lenses, lens)
		} else if verdict != "passed" {
			unmet = append(unmet, lens+" ("+verdict+")")
			lenses = append(lenses, lens)
		}
	}
	if len(unmet) == 0 {
		return ""
	}
	return "declared review(s) not satisfied: " + strings.Join(unmet, ", ") + " — the driver declared these review lenses (mint session set-reviews / spec <reviews>) and the floor requires each to have an attached PASSING verdict before done. Run them and record the result: mint review --<lens> for " + strings.Join(lenses, ",") + ", then mint exec record-review <slug> <spec-id> <lens> passed"
}

func isLogicTier(input Input) bool {
	if IsSafetyTier(input) {
		return true
	}
	if strings.TrimSpace(input.Patch) == "" {
		return false
	}
	for _, line := range walkPatch(input.Patch) {
		if line.Sign != "+" && line.Sign != "-" {
			continue
		}
		if line.File == "" {
			continue
		}
		if docclassify.IsDocPath(line.File) || isLogicScanSkip(line.File) || isConfigFile(line.File) || isTestFile(line.File) {
			continue
		}
		if isCommentLine(line.Content) {
			continue
		}
		trimmed := strings.TrimSpace(line.Content)
		if trimmed == "" || isImportOnlyLine(trimmed) {
			continue
		}
		guardLine := line.Content
		if stripped, ok := regexStrippedLine(line.Content); ok {
			guardLine = stripped
		}
		if logicSignalRE.MatchString(guardLine) && !isInStringLiteral(guardLine, logicSignalRE) {
			return true
		}
		for _, span := range interpolationSpans(line.Content) {
			if spanTripsLogic(span) {
				return true
			}
		}
	}
	return false
}

type patchLine struct {
	File    string
	PreFile string
	Sign    string
	Content string
}

func walkPatch(patch string) []patchLine {
	var out []patchLine
	var file, preFile string
	hunkOld, hunkNew := 0, 0
	hunkRE := regexp.MustCompile(`^@@ -\d+(?:,(\d+))? \+\d+(?:,(\d+))? @@`)
	for _, line := range strings.Split(patch, "\n") {
		raw := strings.TrimSuffix(line, "\r")
		inHunk := hunkOld > 0 || hunkNew > 0
		if !inHunk {
			switch {
			case strings.HasPrefix(raw, "diff --git "):
				file, preFile = "", ""
			case strings.HasPrefix(raw, "--- "):
				preFile = headerPath(raw, "---")
			case strings.HasPrefix(raw, "+++ "):
				file = headerPath(raw, "+++")
			default:
				if m := hunkRE.FindStringSubmatch(raw); m != nil {
					hunkOld = parseHunkCount(m[1])
					hunkNew = parseHunkCount(m[2])
				}
			}
			continue
		}
		if strings.HasPrefix(raw, `\`) {
			continue
		}
		sign := " "
		content := raw
		if raw != "" {
			sign = raw[:1]
			content = raw[1:]
		}
		switch sign {
		case "+":
			hunkNew--
			out = append(out, patchLine{File: file, PreFile: preFile, Sign: "+", Content: content})
		case "-":
			hunkOld--
			out = append(out, patchLine{File: file, PreFile: preFile, Sign: "-", Content: content})
		default:
			hunkOld--
			hunkNew--
			out = append(out, patchLine{File: file, PreFile: preFile, Sign: " ", Content: content})
		}
	}
	return out
}

func headerPath(raw, prefix string) string {
	rest := strings.TrimSpace(strings.TrimPrefix(raw, prefix+" "))
	rest = strings.Split(rest, "\t")[0]
	rest = strings.TrimPrefix(rest, "a/")
	rest = strings.TrimPrefix(rest, "b/")
	if rest == "/dev/null" {
		return ""
	}
	return rest
}

func parseHunkCount(s string) int {
	if s == "" {
		return 1
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	if n == 0 {
		return 1
	}
	return n
}

var logicSignalRE = regexp.MustCompile(strings.Join([]string{
	`\b(?:if|else|for|while|switch|case|return)\b`,
	`\?[^?]*:`,
	`&&|\|\|`,
	`[\w)\]"'\x60]\s*(?:===|!==|==|!=|<=|>=|<|>)\s*[\w($"'\x60+-]`,
	`[\w)\]]\s*(?:\+|-|\*|/|%)\s*[\w($]`,
}, `|`))

var (
	skipOnlyRE    = regexp.MustCompile(`(\.skip\b|\.only\b|\bxit\b|\bxtest\b|\bxdescribe\b|\bfdescribe\b|\bfit\b|\bit\.todo\b|\btest\.skip\b|\btest\.only\b|\bdescribe\.skip\b|\bdescribe\.only\b|@pytest\.mark\.skipif\b|@pytest\.mark\.skip\b|@unittest\.skip(?:If|Unless)?\b|\bt\.SkipNow\b|\bt\.Skip\(|\[\s*["'\x60](?:skip|only)["'\x60]\s*\])`)
	assertionRE   = regexp.MustCompile(`(\bexpect\s*\(|\bassert\b|\bassert\.|\.should\b|\.to\.|\btoBe\b|\btoEqual\b|\btoThrow\b|\btoHaveBeenCalled\b|\bEXPECT_|\bASSERT_|\brequire\.|\bt\.is\s*\(|\bt\.deepEqual\s*\()`)
	disableWrapRE = regexp.MustCompile(`\bif\s*\(\s*(?:false|0)\s*\)`)
	gateOffRE     = regexp.MustCompile(`("(?:lint|types|tests|coverage)"\s*:\s*false\b|\bbail\s*:\s*false\b|--no-verify\b)`)
	coverageKeyRE = regexp.MustCompile(`(?i)\b(coverage|threshold|branches|lines|statements|functions)\b`)
)

func isLogicScanSkip(file string) bool {
	base := filepath.Base(file)
	if strings.HasPrefix(file, "dist/") || strings.HasPrefix(file, "build/") {
		return true
	}
	if strings.HasSuffix(file, ".min.js") || strings.HasSuffix(file, ".min.css") || strings.HasSuffix(file, ".map") {
		return true
	}
	switch base {
	case "package-lock.json", "yarn.lock", "pnpm-lock.yaml", "bun.lockb", "Cargo.lock",
		"composer.lock", "Gemfile.lock", "go.sum", "Pipfile.lock", "poetry.lock",
		"mix.lock", "pubspec.lock", "flake.lock", "Podfile.lock":
		return true
	}
	return regexp.MustCompile(`^tsconfig[^/]*\.json$`).MatchString(base)
}

func isTestFile(file string) bool {
	return regexp.MustCompile(`(\.(test|spec)\.|_test\.|(^|/)test_|/tests?/)`).MatchString(file)
}

func isConfigFile(file string) bool {
	base := filepath.Base(file)
	if file == ".mint/config.json" || strings.HasSuffix(file, "/.mint/config.json") {
		return true
	}
	switch base {
	case "package.json", "pytest.ini", "setup.cfg", "tox.ini", "pyproject.toml":
		return true
	}
	return regexp.MustCompile(`^(jest|vitest|playwright|cypress|babel)\.config\.[cm]?[jt]s$`).MatchString(base) ||
		regexp.MustCompile(`^\.nycrc(\.json)?$`).MatchString(base)
}

func isCommentLine(content string) bool {
	t := strings.TrimSpace(content)
	return strings.HasPrefix(t, "//") ||
		strings.HasPrefix(t, "#") ||
		strings.HasPrefix(t, "*") ||
		strings.HasPrefix(t, "/*") ||
		strings.HasPrefix(t, "<!--")
}

func isImportOnlyLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "import ") ||
		strings.HasPrefix(trimmed, "import\t") ||
		regexp.MustCompile(`^export\b[^=]*\bfrom\b`).MatchString(trimmed) ||
		regexp.MustCompile(`^(?:const|let|var)\s+[\w{}\s,*]+=\s*require\(`).MatchString(trimmed) ||
		strings.HasPrefix(trimmed, "require(")
}

func interpolationSpans(content string) []string {
	var spans []string
	for i := 0; i < len(content); {
		if content[i] == '$' && i+1 < len(content) && content[i+1] == '{' {
			depth := 1
			j := i + 2
			start := j
			var quote byte
			for j < len(content) && depth > 0 {
				ch := content[j]
				if ch == '\\' {
					j += 2
					continue
				}
				if quote != 0 {
					if ch == quote {
						quote = 0
					}
					j++
					continue
				}
				if ch == '"' || ch == '\'' || ch == '`' {
					quote = ch
					j++
					continue
				}
				if ch == '{' {
					depth++
				} else if ch == '}' {
					depth--
				}
				if depth == 0 {
					break
				}
				j++
			}
			if j <= len(content) {
				spans = append(spans, content[start:j])
			}
			i = j + 1
			continue
		}
		if end, ok := scanRegexLiteral(content, i); ok {
			spans = append(spans, content[:i]+content[end+1:])
			i = end + 1
			continue
		}
		i++
	}
	return spans
}

func spanTripsLogic(span string) bool {
	if logicSignalRE.MatchString(span) && !isInStringLiteral(span, logicSignalRE) {
		return true
	}
	for _, inner := range interpolationSpans(span) {
		if inner != span && spanTripsLogic(inner) {
			return true
		}
	}
	return false
}

func regexStrippedLine(content string) (string, bool) {
	var out strings.Builder
	hadRegex := false
	for i := 0; i < len(content); {
		if end, ok := scanRegexLiteral(content, i); ok {
			hadRegex = true
			i = end + 1
			continue
		}
		out.WriteByte(content[i])
		i++
	}
	return out.String(), hadRegex
}

func scanRegexLiteral(content string, i int) (int, bool) {
	if i >= len(content) || content[i] != '/' || (i+1 < len(content) && (content[i+1] == '/' || content[i+1] == '*')) {
		return 0, false
	}
	prev := lastNonSpace(content, i-1)
	if !(prev == 0 || strings.ContainsRune("(,=:[!&|?{;+-*%<>", rune(prev)) || endsWithReturn(content, i)) {
		return 0, false
	}
	inClass := false
	for j := i + 1; j < len(content); j++ {
		ch := content[j]
		if ch == '\\' {
			j++
			continue
		}
		if ch == '[' {
			inClass = true
		} else if ch == ']' {
			inClass = false
		} else if ch == '/' && !inClass {
			return j, true
		}
	}
	return 0, false
}

func lastNonSpace(content string, i int) byte {
	for k := i; k >= 0; k-- {
		if !unicode.IsSpace(rune(content[k])) {
			return content[k]
		}
	}
	return 0
}

func endsWithReturn(content string, i int) bool {
	k := i - 1
	skipped := 0
	for k >= 0 && unicode.IsSpace(rune(content[k])) && skipped < 16 {
		k--
		skipped++
	}
	if k >= 0 && skipped == 16 && unicode.IsSpace(rune(content[k])) {
		return false
	}
	if k < 5 {
		return content[:k+1] == "return"
	}
	if content[k-5:k+1] != "return" {
		return false
	}
	before := content[k-6]
	return !(before == '_' || before >= '0' && before <= '9' || before >= 'A' && before <= 'Z' || before >= 'a' && before <= 'z')
}

func isInStringLiteral(content string, re *regexp.Regexp) bool {
	matches := re.FindAllStringIndex(content, -1)
	if len(matches) == 0 {
		return false
	}
	oldMap := stringMap(content, false)
	strictMap := stringMap(content, true)
	for _, span := range matches {
		for i := span[0]; i < span[1]; i++ {
			if i >= len(oldMap) || !oldMap[i] || !strictMap[i] {
				return false
			}
		}
	}
	return true
}

func stringMap(content string, strict bool) []bool {
	inString := make([]bool, len(content))
	var quote byte
	for i := 0; i < len(content); {
		c := content[i]
		if quote != 0 {
			if c == '\\' {
				i += 2
				continue
			}
			if c == quote {
				quote = 0
			} else {
				inString[i] = true
			}
			i++
			continue
		}
		if strict {
			if c == '/' && i+1 < len(content) && content[i+1] == '/' {
				break
			}
			if c == '/' && i+1 < len(content) && content[i+1] == '*' {
				close := strings.Index(content[i+2:], "*/")
				if close == -1 {
					break
				}
				i += close + 4
				continue
			}
			if end, ok := scanRegexLiteral(content, i); ok {
				i = end + 1
				continue
			}
		}
		if c == '"' || c == '\'' || c == '`' {
			quote = c
		}
		i++
	}
	return inString
}

func hasSubstantiveAttestation(verdict map[string]any, flagKey, reasonKey string) bool {
	if verdict == nil {
		return false
	}
	flag, ok := verdict[flagKey].(bool)
	if !ok || !flag {
		return false
	}
	reason, ok := verdict[reasonKey].(string)
	if !ok {
		return false
	}
	reason = strings.TrimSpace(reason)
	if len(reason) < 8 {
		return false
	}
	for _, r := range reason {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

type tamperSignal struct {
	Signal   string
	File     string
	Evidence string
}

func antiGamingSignals(patch string) []tamperSignal {
	var signals []tamperSignal
	type assertCount struct{ removed, added int }
	assertByFile := map[string]*assertCount{}
	var covRemoved []coverageLine
	covFile := ""

	for _, line := range walkPatch(patch) {
		added := line.Sign == "+"
		removed := line.Sign == "-"

		if line.File == "" && line.PreFile != "" && isTestFile(line.PreFile) {
			if !hasSignal(signals, "f:test-file-deleted", line.PreFile) {
				signals = append(signals, tamperSignal{"f:test-file-deleted", line.PreFile, "entire test file deleted (post-image /dev/null)"})
			}
			continue
		}
		if line.File == "" {
			continue
		}

		if added && isTestFile(line.File) && !isCommentLine(line.Content) && skipOnlyRE.MatchString(line.Content) && !isInStringLiteral(line.Content, skipOnlyRE) {
			signals = append(signals, tamperSignal{"a:skip-or-only-added", line.File, strings.TrimSpace(line.Content)})
		}
		if added && isTestFile(line.File) && !isCommentLine(line.Content) && disableWrapRE.MatchString(line.Content) && !isInStringLiteral(line.Content, disableWrapRE) {
			signals = append(signals, tamperSignal{"g:test-disabled-wrap", line.File, strings.TrimSpace(line.Content)})
		}
		if isTestFile(line.File) && !isCommentLine(line.Content) && assertionRE.MatchString(line.Content) && !isInStringLiteral(line.Content, assertionRE) {
			ac := assertByFile[line.File]
			if ac == nil {
				ac = &assertCount{}
				assertByFile[line.File] = ac
			}
			if added {
				ac.added++
			} else if removed {
				ac.removed++
			}
		}
		if added && isConfigFile(line.File) && gateOffRE.MatchString(line.Content) {
			signals = append(signals, tamperSignal{"c:gate-disabled", line.File, strings.TrimSpace(line.Content)})
		}
		if isScorerFile(line.File) && (added || removed) && !hasSignal(signals, "e:scorer-or-runner-edited", line.File) {
			signals = append(signals, tamperSignal{"e:scorer-or-runner-edited", line.File, strings.TrimSpace(line.Content)})
		}
		if isConfigFile(line.File) {
			if line.File != covFile {
				covRemoved = nil
				covFile = line.File
			}
			if removed && coverageKeyRE.MatchString(line.Content) {
				if num, ok := numberFor(line.Content); ok {
					covRemoved = append(covRemoved, coverageLine{Content: strings.TrimSpace(line.Content), Num: num, Key: keyFor(line.Content)})
				}
			} else if added && coverageKeyRE.MatchString(line.Content) {
				if addNum, ok := numberFor(line.Content); ok {
					addKey := keyFor(line.Content)
					for _, removedLine := range covRemoved {
						if removedLine.Key == addKey && addNum < removedLine.Num {
							signals = append(signals, tamperSignal{
								Signal:   "d:coverage-threshold-lowered",
								File:     line.File,
								Evidence: fmt.Sprintf("%s → %s (lowered %g → %g)", removedLine.Content, strings.TrimSpace(line.Content), removedLine.Num, addNum),
							})
							break
						}
					}
				}
			}
		}
	}
	// Per-file accounting: a net assertion deletion in ANY single test file trips the
	// signal, so a net-zero cross-file swap (delete real asserts in A, add trivial ones
	// in B) can't hide behind a repo-wide even count. One signal per offending file,
	// sorted for deterministic output.
	assertFiles := make([]string, 0, len(assertByFile))
	for file := range assertByFile {
		assertFiles = append(assertFiles, file)
	}
	sort.Strings(assertFiles)
	for _, file := range assertFiles {
		ac := assertByFile[file]
		if ac.removed > ac.added {
			signals = append(signals, tamperSignal{
				Signal:   "b:net-assertion-deletion",
				File:     file,
				Evidence: fmt.Sprintf("%d assertion line(s) removed vs %d added in this test file (net deletion)", ac.removed, ac.added),
			})
		}
	}
	return signals
}

type coverageLine struct {
	Content string
	Num     float64
	Key     string
}

func signalSummary(signals []tamperSignal) string {
	parts := make([]string, 0, len(signals))
	for _, s := range signals {
		parts = append(parts, fmt.Sprintf("%s in %s: %s", s.Signal, s.File, s.Evidence))
	}
	return strings.Join(parts, "; ")
}

func hasSignal(signals []tamperSignal, signal, file string) bool {
	for _, s := range signals {
		if s.Signal == signal && s.File == file {
			return true
		}
	}
	return false
}

func isScorerFile(file string) bool {
	base := filepath.Base(file)
	if file == ".mint/config.json" || strings.HasSuffix(file, "/.mint/config.json") {
		return true
	}
	if base == "conftest.py" || base == "setup.cfg" || base == "pytest.ini" || base == "tox.ini" {
		return true
	}
	if regexp.MustCompile(`^(jest|vitest|playwright|cypress|karma|babel)\.config\.[cm]?[jt]s$`).MatchString(base) {
		return true
	}
	if regexp.MustCompile(`^(jest|vitest)\.setup\.[cm]?[jt]s$`).MatchString(base) {
		return true
	}
	return regexp.MustCompile(`\.setup\.[cm]?[jt]s$`).MatchString(base)
}

func numberFor(line string) (float64, bool) {
	m := regexp.MustCompile(`-?\d+(?:\.\d+)?`).FindString(line)
	if m == "" {
		return 0, false
	}
	var sign, value, frac, div float64 = 1, 0, 0, 1
	seenDot := false
	for i, r := range m {
		if i == 0 && r == '-' {
			sign = -1
			continue
		}
		if r == '.' {
			seenDot = true
			continue
		}
		if r < '0' || r > '9' {
			continue
		}
		if seenDot {
			div *= 10
			frac += float64(r-'0') / div
		} else {
			value = value*10 + float64(r-'0')
		}
	}
	return sign * (value + frac), true
}

func keyFor(line string) string {
	m := regexp.MustCompile(`["']?([A-Za-z_][\w-]*)["']?\s*[:=]`).FindStringSubmatch(line)
	if m == nil {
		return ""
	}
	return strings.ToLower(m[1])
}

func safeRun(run func(Input) clauseVerdict, input Input) (out clauseVerdict) {
	defer func() {
		if r := recover(); r != nil {
			out = fail(fmt.Sprintf("clause threw on malformed facts: %v", r))
		}
	}()
	return run(input)
}

func pass() clauseVerdict {
	return clauseVerdict{pass: true}
}

func fail(reason string) clauseVerdict {
	return clauseVerdict{pass: false, reason: reason}
}

func normalizeProvenance(s string) string {
	return engine.Canonical(s)
}

func sameProvenance(a, b string) bool {
	return normalizeProvenance(a) == normalizeProvenance(b)
}

var safetyTerms = []string{
	"security",
	"trust-boundary",
	"validation",
	"accessibility",
	"data-loss",
	"auth",
	"crypto",
	"token",
	"password",
	"secret",
	"credential",
	"validate",
	"sanitize",
}

var safetySinkMarkers = []string{
	"innerhtml",
	"dangerouslysetinnerhtml",
	"outerhtml",
	"insertadjacenthtml",
	".exec(",
	"execsync",
	"spawnsync",
	"child_process",
	"spawn(",
	"eval(",
	"new function",
	"req.query",
	"req.body",
	"req.params",
	"request.query",
	"request.body",
	"document.cookie",
	"set-cookie",
	"httponly",
	"samesite",
	"access-control-allow-origin",
	"cors",
	"select ",
	"insert into",
	"delete from",
	"update ",
	"drop table",
	" where ",
	".redirect(",
	"location.href",
	"res.redirect",
	"rejectunauthorized",
	"node_tls_reject",
}

var safetyPathPatterns = []string{
	"auth",
	"oauth",
	"login",
	"password",
	"permission",
	"crypto",
	"secret",
	"token",
	"jwt",
	"security",
	"csrf",
	"cors",
	"payment",
	"billing",
	"sanitiz",
	"validat",
}

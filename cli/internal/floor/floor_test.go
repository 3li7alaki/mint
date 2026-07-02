package floor

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"mint/internal/engine"
)

func TestTerminalStates(t *testing.T) {
	want := []string{"done-verified", "budget-exhausted", "stuck-escalated", "external-stop"}
	if !reflect.DeepEqual(TerminalStates, want) {
		t.Fatalf("TerminalStates = %#v, want %#v", TerminalStates, want)
	}
}

func TestEnforceFixedSevenClauses(t *testing.T) {
	for _, input := range []Input{{}, passingInput()} {
		result := Enforce(input)
		if len(result.Clauses) != 7 {
			t.Fatalf("got %d clauses, want 7: %#v", len(result.Clauses), result.Clauses)
		}
		for i, clause := range result.Clauses {
			if clause.Clause != i+1 {
				t.Fatalf("clause order = %#v", result.Clauses)
			}
			if clause.Name == "" {
				t.Fatalf("clause %d has empty name", clause.Clause)
			}
		}
	}
}

func TestPassingBaseline(t *testing.T) {
	result := Enforce(passingInput())
	if !result.Pass || len(result.Failed) != 0 {
		t.Fatalf("Enforce(passingInput) = %#v", result)
	}
}

func TestClauseScope(t *testing.T) {
	cases := []struct {
		name    string
		spec    string
		changed []string
		want    bool
	}{
		{"exact", specWith("cli/lib/a.js", "cli/lib/b.js"), []string{"cli/lib/a.js", "cli/lib/b.js"}, true},
		{"directory prefix", specWith("cli/lib"), []string{"cli/lib/a.js", "cli/lib/nested/b.js"}, true},
		{"glob", specWith("cli/lib/*.test.js", "src/**/*.js"), []string{"cli/lib/floor-kernel.test.js", "src/deep/nested/x.js"}, true},
		{"star boundary", specWith("cli/lib/*.js"), []string{"cli/lib/nested/x.js"}, false},
		{"out of lane", specWith("cli/lib/floor-kernel.js"), []string{"cli/lib/floor-kernel.js", "cli/commands/verify.js"}, false},
		{"empty diff", specWith("cli/lib/a.js"), nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := passingInput()
			input.Spec = tc.spec
			input.ChangedFiles = tc.changed
			c5 := clauseResult(t, Enforce(input), 5)
			if c5.Pass != tc.want {
				t.Fatalf("clause 5 pass = %v, want %v, reason=%v", c5.Pass, tc.want, reason(c5))
			}
			if !tc.want && reason(c5) == "" {
				t.Fatal("failing scope clause should include a reason")
			}
		})
	}
}

func TestClauseScopeReDoSResistanceAndGlobstarSemantics(t *testing.T) {
	// ReDoS-resistance is structural (segment-walk scope matcher, no backtracking regex): a
	// pathological globstar lane is bounded by construction. The CONTRACT assertion is that
	// the non-matching pathological lane still fails scope; a backtracking regression would
	// hang and trip go test's deadline rather than slip through. No wall-clock check (flaked
	// under CPU load).
	input := passingInput()
	input.Spec = specWith(strings.Repeat("**/", 40) + "z")
	input.ChangedFiles = []string{strings.Repeat("a/", 40) + strings.Repeat("a", 60) + "Q"}
	if c5 := clauseResult(t, Enforce(input), 5); c5.Pass {
		t.Fatalf("non-matching pathological lane should still fail scope: %#v", c5)
	}

	input = passingInput()
	input.Spec = specWith("src/**/**/x.js")
	input.ChangedFiles = []string{"src/a/b/c/x.js"}
	if c5 := clauseResult(t, Enforce(input), 5); !c5.Pass {
		t.Fatalf("redundant globstar lane should preserve match semantics: %#v", c5)
	}

	input = passingInput()
	input.Spec = specWith(strings.Repeat("*", 5000))
	input.ChangedFiles = []string{"cli/lib/floor-kernel.js"}
	if c5 := clauseResult(t, Enforce(input), 5); c5.Pass {
		t.Fatalf("absurdly long scope lane should fail closed: %#v", c5)
	}
}

func TestClauseTerminating(t *testing.T) {
	for _, state := range TerminalStates {
		input := passingInput()
		input.TerminalState = state
		if c6 := clauseResult(t, Enforce(input), 6); !c6.Pass {
			t.Fatalf("terminal state %q failed: %s", state, reason(c6))
		}
	}
	for _, state := range []string{"done", "", "finished"} {
		input := passingInput()
		input.TerminalState = state
		c6 := clauseResult(t, Enforce(input), 6)
		if c6.Pass || !strings.Contains(reason(c6), "not one of") {
			t.Fatalf("terminal state %q got clause 6 %#v", state, c6)
		}
	}
}

func TestEnforceTopLevelVerdictAggregation(t *testing.T) {
	input := passingInput()
	input.Spec = specWith("cli/lib/floor-kernel.js")
	input.ChangedFiles = []string{"cli/commands/verify.js"}
	result := Enforce(input)
	if result.Pass {
		t.Fatalf("out-of-lane unit should not pass: %#v", result)
	}
	if !reflect.DeepEqual(result.Failed, []int{5}) {
		t.Fatalf("expected only clause 5 to fail, got %#v in %#v", result.Failed, result)
	}

	result = Enforce(passingInput())
	if !result.Pass || len(result.Failed) != 0 || len(result.Clauses) != 7 {
		t.Fatalf("clean non-safety unit should pass all seven clauses: %#v", result)
	}
	for _, clause := range result.Clauses {
		if !clause.Pass {
			t.Fatalf("clean non-safety clause failed: %#v", clause)
		}
	}
}

func TestEnforceSafetyMilestoneRequiresAllStackedAttestations(t *testing.T) {
	safe := passingInput()
	safe.Spec = specWith("cli/lib/auth.js", "cli/lib/floor-kernel.test.js")
	safe.ChangedFiles = []string{"cli/lib/auth.js"}
	safe.Patch = "+ const token = signIn(password);"
	safe.MakerEngine = "claude"
	safe.MakerSession = "sess-maker"
	safe.Verdict = map[string]any{
		"accepted":            true,
		"byEngine":            "codex",
		"bySession":           "sess-checker",
		"safetyReviewed":      true,
		"safetyReason":        "auth token handling reviewed for leakage",
		"adversarialReviewed": true,
		"adversarialReason":   "fuzzed the sign-in token path for bypasses",
	}
	result := Enforce(safe)
	if !result.Pass || len(result.Failed) != 0 {
		t.Fatalf("fully attested safety unit should pass: %#v", result)
	}

	delete(safe.Verdict, "safetyReviewed")
	delete(safe.Verdict, "safetyReason")
	result = Enforce(safe)
	if clauseResult(t, result, 1).Pass != true || clauseResult(t, result, 2).Pass != true {
		t.Fatalf("without safety review, clauses 1 and 2 should still pass: %#v", result)
	}
	if c3 := clauseResult(t, result, 3); c3.Pass || !strings.Contains(reason(c3), "safety carve-out") {
		t.Fatalf("without safety review, clause 3 should fail: %#v", c3)
	}
	if !reflect.DeepEqual(result.Failed, []int{3}) {
		t.Fatalf("expected only clause 3 failed without safety review, got %#v", result.Failed)
	}
}

func TestClauseVerifiableCompletion(t *testing.T) {
	input := passingInput()
	input.Gates = nil
	if c1 := clauseResult(t, Enforce(input), 1); c1.Pass || !strings.Contains(reason(c1), "gates") {
		t.Fatalf("missing gates got %#v", c1)
	}

	input = passingInput()
	input.Verdict = nil
	if c1 := clauseResult(t, Enforce(input), 1); c1.Pass || !strings.Contains(reason(c1), "verdict") {
		t.Fatalf("missing verdict got %#v", c1)
	}

	input = passingInput()
	input.Verdict["accepted"] = false
	if c1 := clauseResult(t, Enforce(input), 1); c1.Pass || !strings.Contains(reason(c1), "did not accept") {
		t.Fatalf("rejected verdict got %#v", c1)
	}

	input = passingInput()
	input.RequiredReviews = []string{"security", "quality"}
	input.Reviews = map[string]string{"security": "passed"}
	if c1 := clauseResult(t, Enforce(input), 1); c1.Pass || !strings.Contains(reason(c1), "quality (not run)") {
		t.Fatalf("missing declared review got %#v", c1)
	}

	input = passingInput()
	input.RequiredReviews = []string{"security"}
	input.Reviews = map[string]string{"security": "failed"}
	if c1 := clauseResult(t, Enforce(input), 1); c1.Pass || !strings.Contains(reason(c1), "security (failed)") {
		t.Fatalf("failed declared review got %#v", c1)
	}

	input = passingInput()
	input.RequiredReviews = []string{"security", "quality"}
	input.Reviews = map[string]string{"security": "passed", "quality": "passed"}
	if c1 := clauseResult(t, Enforce(input), 1); !c1.Pass {
		t.Fatalf("all declared reviews passed should satisfy clause 1: %#v", c1)
	}

	input = passingInput()
	input.RequiredReviews = []string{"", "  "}
	input.Reviews = map[string]string{}
	if c1 := clauseResult(t, Enforce(input), 1); !c1.Pass {
		t.Fatalf("blank declared review names should be ignored: %#v", c1)
	}

	input = passingInput()
	input.Spec = specWith("README.md", "docs/x.md")
	input.ChangedFiles = []string{"README.md", "docs/x.md"}
	input.RequiredReviews = []string{"security"}
	input.Reviews = map[string]string{}
	if c1 := clauseResult(t, Enforce(input), 1); !c1.Pass {
		t.Fatalf("docs-only declared review should be no-op: %#v", c1)
	}

	input = passingInput()
	input.Spec = specWith("README.md", "cli/lib/floor-kernel.js")
	input.ChangedFiles = []string{"README.md", "cli/lib/floor-kernel.js"}
	input.RequiredReviews = []string{"security"}
	input.Reviews = map[string]string{}
	if c1 := clauseResult(t, Enforce(input), 1); c1.Pass || !strings.Contains(reason(c1), "security (not run)") {
		t.Fatalf("mixed code/docs diff should still require declared reviews: %#v", c1)
	}
}

func TestClauseVerifiableCompletionAdversarialDefault(t *testing.T) {
	logicPatch := diffOf("cli/lib/calc.js", []string{
		"+func fee(total, rate int) int {",
		"+  if total > 0 { return total * rate }",
		"+  return 0",
		"+}",
	})
	input := passingInput()
	input.Spec = specWith("cli/lib/calc.js")
	input.ChangedFiles = []string{"cli/lib/calc.js"}
	input.Patch = logicPatch
	input.Verdict["byEngine"] = "codex"
	c1 := clauseResult(t, Enforce(input), 1)
	if c1.Pass || !strings.Contains(reason(c1), "logic/trust diff") || !strings.Contains(reason(c1), "adversarialReviewed") {
		t.Fatalf("logic diff without adversarial attestation got %#v", c1)
	}

	input.Verdict["adversarialReviewed"] = true
	input.Verdict["adversarialReason"] = "fuzzed fee branch with zero and negative totals"
	if result := Enforce(input); !result.Pass {
		t.Fatalf("attested logic diff should pass full floor: %#v", result)
	}
}

func TestLogicTierTrivialDiffsDoNotRequireAdversarialAttestation(t *testing.T) {
	cases := []struct {
		name  string
		file  string
		lines []string
	}{
		{"docs prose", "docs/guide.md", []string{"+If you return to the loop, switch here for clarity."}},
		{"config version", "package.json", []string{`+  "version": "1.2.3",`}},
		{"import only", "cli/lib/calc.js", []string{`-import { old } from './a.js';`, `+import { renamed } from './b.js';`}},
		{"test only", "cli/lib/calc.test.js", []string{`+test("fee when total > 0", () => expect(fee(10, 2)).toBe(20));`}},
		{"comment only", "cli/lib/calc.js", []string{"+// if total > 0 we return the fee; comment only", "+   "}},
		{"string constant url", "cli/lib/calc.js", []string{`+const ENDPOINT = "https://api.example.com/v1/users";`}},
		{"string constant comparison", "cli/lib/calc.js", []string{`+const HELP = "use a < b or a > b to compare";`}},
		{"lockfile url", "package-lock.json", []string{`+      "resolved": "https://registry.npmjs.org/foo/-/foo-1.0.0.tgz",`}},
		{"tsconfig string", "tsconfig.build.json", []string{`+    "comment": "requires node 18 + npm 9"`}},
		{"trivial interpolation", "cli/lib/calc.js", []string{"+" + "const p = `path: ${dir}/sub`;"}},
		{"regex body", "cli/lib/calc.js", []string{"+const slug = /[a-z]+-[0-9]+/;"}},
		{"pipfile lock", "Pipfile.lock", []string{`+ "url": "https://files.pythonhosted.org/packages/a/b/pkg-1.0.tar.gz"`}},
		{"poetry lock", "poetry.lock", []string{`+ url = "https://files.pythonhosted.org/packages/a/b/pkg-1.0.tar.gz"`}},
		{"mix lock", "mix.lock", []string{`+ {:plug, "1.14.0", "abc123", [:mix], [], "hexpm"}`}},
		{"pubspec lock", "pubspec.lock", []string{`+     url: "https://pub.dev"`}},
		{"flake lock", "flake.lock", []string{`+ "narHash": "sha256-a+b/c"`}},
		{"podfile lock", "Podfile.lock", []string{`+  - AFNetworking (4.0.1)`}},
		{"minified js", "public/app.min.js", []string{"+function x(){return a?b:c}"}},
		{"minified css", "public/app.min.css", []string{"+.x{width:calc(100% - 1px)}"}},
		{"sourcemap", "public/app.js.map", []string{`+{"mappings":"AAAA,CAAC;A > B"}`}},
		{"dist artifact", "dist/app.js", []string{"+if(a>b){return a*b}"}},
		{"build artifact", "build/app.js", []string{"+if(a>b){return a*b}"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := passingInput()
			input.Spec = specWith(tc.file)
			input.ChangedFiles = []string{tc.file}
			input.Patch = diffOf(tc.file, tc.lines)
			input.Verdict["byEngine"] = "codex"
			c1 := clauseResult(t, Enforce(input), 1)
			if !c1.Pass {
				t.Fatalf("trivial diff required adversarial attestation: %#v", c1)
			}
		})
	}
}

func TestLogicTierBuildArtifactAnchorBoundaries(t *testing.T) {
	input := passingInput()
	input.Spec = specWith("src/build/calc.js")
	input.ChangedFiles = []string{"src/build/calc.js"}
	input.Patch = diffOf("src/build/calc.js", []string{"+if (a > b) return a * b;"})
	input.Verdict["byEngine"] = "codex"
	c1 := clauseResult(t, Enforce(input), 1)
	if c1.Pass || !strings.Contains(reason(c1), "logic/trust diff") {
		t.Fatalf("hand-written src/build file should not be skipped as generated artifact: %#v", c1)
	}
}

func TestLogicTierInterpolationAndRegexDodgeClose(t *testing.T) {
	cases := []struct {
		name string
		line string
	}{
		{"ternary interpolation", "+" + "const label = `${a > b ? a : b}`;"},
		{"arithmetic interpolation", "+" + "const msg = `fee is ${total * rate} dollars`;"},
		{"brace in interpolation string", "+const v = `${ obj[\"a}b\"] > 0 ? 1 : 2 }`;"},
		{"regex quote desync", `+const re = /"/; if (a > b) return a;`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := passingInput()
			input.Spec = specWith("cli/lib/calc.js")
			input.ChangedFiles = []string{"cli/lib/calc.js"}
			input.Patch = diffOf("cli/lib/calc.js", []string{tc.line})
			input.Verdict["byEngine"] = "codex"
			c1 := clauseResult(t, Enforce(input), 1)
			if c1.Pass || !strings.Contains(reason(c1), "logic/trust diff") {
				t.Fatalf("logic dodge did not trip clause 1: %#v", c1)
			}
		})
	}
}

func TestLogicTierRegexAndInterpolationRound4Parity(t *testing.T) {
	trips := []struct {
		name string
		line string
	}{
		{"return regex then ternary", "+  return /re/.test(s) ? a : b;"},
		{"returnValue division", "+  const x = returnValue / 2;"},
		{"member return division", "+  const y = foo.return / 2;"},
		{"return slash regex still control flow", "+  return/a-b+c/;"},
		{"real comparison in interpolation", "+const a = `${ a < b }`;"},
		{"real arithmetic in interpolation", "+const b = `${ a + b }`;"},
	}
	for _, tc := range trips {
		t.Run(tc.name, func(t *testing.T) {
			input := passingInput()
			input.Spec = specWith("cli/lib/calc.js")
			input.ChangedFiles = []string{"cli/lib/calc.js"}
			input.Patch = diffOf("cli/lib/calc.js", []string{tc.line})
			input.Verdict["byEngine"] = "codex"
			c1 := clauseResult(t, Enforce(input), 1)
			if c1.Pass || !strings.Contains(reason(c1), "logic/trust diff") {
				t.Fatalf("expected logic-tier clause 1 failure for %q: %#v", tc.line, c1)
			}
		})
	}

	nonTrips := []struct {
		name string
		line string
	}{
		{"same regex body without return keyword", "+  const re = /a-b+c/;"},
		{"comparison only inside interpolation string", "+const a = `${ \"a < b\" }`;"},
		{"arithmetic only inside interpolation call string", "+const b = `${ greeting(\"2+2\") }`;"},
		{"minus only inside interpolation object string", "+const c = `${ JSON.stringify({a:\"1-2-3\"}) }`;"},
	}
	for _, tc := range nonTrips {
		t.Run(tc.name, func(t *testing.T) {
			input := passingInput()
			input.Spec = specWith("cli/lib/calc.js")
			input.ChangedFiles = []string{"cli/lib/calc.js"}
			input.Patch = diffOf("cli/lib/calc.js", []string{tc.line})
			input.Verdict["byEngine"] = "codex"
			c1 := clauseResult(t, Enforce(input), 1)
			if !c1.Pass {
				t.Fatalf("honest string/regex content required adversarial attestation for %q: %#v", tc.line, c1)
			}
		})
	}
}

func TestLogicTierRegexQuoteDesyncRound5Parity(t *testing.T) {
	trips := []string{
		`+const s = label + /"/.source; if (a > b) charge(total * rate);`,
		`+const fee = base + /"/.source; const total = qty * price;`,
		`+const r = /a/ + /"/; a > b ? p : q;`,
		`+x + /"/; if (a > b) doX();`,
		`+x * /"/; if (a > b) doX();`,
		`+x % /"/; if (a > b) doX();`,
		`+x > /"/; if (a > b) doX();`,
		`+x < /"/; if (a > b) doX();`,
		"+const q = a / b / c;",
		"+const r = total / count;",
		"+const x = arr[i] / n;",
		"+const y = sum / len;",
		"+const z = a + b;",
	}
	for _, line := range trips {
		t.Run(line, func(t *testing.T) {
			input := passingInput()
			input.Spec = specWith("cli/lib/calc.js")
			input.ChangedFiles = []string{"cli/lib/calc.js"}
			input.Patch = diffOf("cli/lib/calc.js", []string{line})
			input.Verdict["byEngine"] = "codex"
			c1 := clauseResult(t, Enforce(input), 1)
			if c1.Pass || !strings.Contains(reason(c1), "logic/trust diff") {
				t.Fatalf("expected logic-tier clause 1 failure for %q: %#v", line, c1)
			}
		})
	}

	input := passingInput()
	input.Spec = specWith("cli/lib/calc.js")
	input.ChangedFiles = []string{"cli/lib/calc.js"}
	input.Patch = diffOf("cli/lib/calc.js", []string{"+const slug = /[a-z]+-[0-9]+/;"})
	input.Verdict["byEngine"] = "codex"
	if c1 := clauseResult(t, Enforce(input), 1); !c1.Pass {
		t.Fatalf("pure regex preceded by assignment should not trap: %#v", c1)
	}
}

func TestClauseMakerChecker(t *testing.T) {
	input := passingInput()
	input.Verdict["bySession"] = input.MakerSession
	c2 := clauseResult(t, Enforce(input), 2)
	if c2.Pass || !strings.Contains(reason(c2), "maker's own engine+session") {
		t.Fatalf("self verdict got %#v", c2)
	}

	input = passingInput()
	input.Verdict["byEngine"] = "codex"
	if c2 := clauseResult(t, Enforce(input), 2); !c2.Pass {
		t.Fatalf("different engine should pass: %#v", c2)
	}

	input = passingInput()
	input.MakerEngine = "phantom"
	if c2 := clauseResult(t, Enforce(input), 2); c2.Pass || !strings.Contains(reason(c2), "not a recognized engine") {
		t.Fatalf("unknown maker engine got %#v", c2)
	}

	input = passingInput()
	input.ChangedFiles = []string{"src/auth/login.go"}
	input.Spec = specWith("src/auth/login.go")
	input.Patch = "+ validate password token"
	c2 = clauseResult(t, Enforce(input), 2)
	if c2.Pass || !strings.Contains(reason(c2), "DIFFERENT engine") {
		t.Fatalf("safety same-engine verdict got %#v", c2)
	}
	input.Verdict["byEngine"] = "codex"
	if c2 := clauseResult(t, Enforce(input), 2); !c2.Pass {
		t.Fatalf("safety different-engine verdict should pass: %#v", c2)
	}
}

func TestClauseMakerCheckerEngineIdentityNormalization(t *testing.T) {
	for _, checker := range []string{"codex", "Codex", " CODEX ", "ｃｏｄｅｘ", "codex\u200e", "co\u200bdex"} {
		input := passingInput()
		input.Verdict["byEngine"] = checker
		c2 := clauseResult(t, Enforce(input), 2)
		if !c2.Pass {
			t.Fatalf("checker engine variant %q should normalize to known cross-engine codex: %#v", checker, c2)
		}
	}
	input := passingInput()
	input.Verdict["byEngine"] = "оpus"
	c2 := clauseResult(t, Enforce(input), 2)
	if c2.Pass || !strings.Contains(reason(c2), "not a recognized engine") {
		t.Fatalf("confusable engine should fail closed: %#v", c2)
	}
}

func TestClauseMakerCheckerIdentityAllowlistAndDisguises(t *testing.T) {
	for _, checker := range engine.Keys() {
		maker := "claude"
		if checker == maker {
			maker = "codex"
		}
		input := passingInput()
		input.MakerEngine = maker
		input.Verdict["byEngine"] = checker
		c2 := clauseResult(t, Enforce(input), 2)
		if !c2.Pass {
			t.Fatalf("registered checker %q vs maker %q should pass: %#v", checker, maker, c2)
		}
	}

	for _, byEngine := range []string{"claude ", " Claude", "\tclaude\t", "CLAUDE", "clau\u200dde", "\ufeffclaude"} {
		input := passingInput()
		input.MakerEngine = "claude"
		input.MakerSession = "sess-maker"
		input.Verdict["byEngine"] = byEngine
		input.Verdict["bySession"] = "sess-maker"
		c2 := clauseResult(t, Enforce(input), 2)
		if c2.Pass || !strings.Contains(reason(c2), "maker's own engine+session") {
			t.Fatalf("disguised same engine %q should fail as self-verdict: %#v", byEngine, c2)
		}
	}

	input := passingInput()
	input.MakerEngine = "claude"
	input.MakerSession = "sess-maker"
	input.Verdict["byEngine"] = "Claude "
	input.Verdict["bySession"] = "sess-fresh"
	if c2 := clauseResult(t, Enforce(input), 2); !c2.Pass {
		t.Fatalf("same normalized engine with genuinely fresh session should pass on non-safety diff: %#v", c2)
	}

	input = passingInput()
	input.MakerEngine = "claude"
	input.MakerSession = "sess 1"
	input.Verdict["byEngine"] = "claude"
	input.Verdict["bySession"] = "sess  1"
	if c2 := clauseResult(t, Enforce(input), 2); c2.Pass || !strings.Contains(reason(c2), "maker's own engine+session") {
		t.Fatalf("collapsed whitespace session should be treated as same provenance: %#v", c2)
	}

	for _, bad := range []string{"phantom-engine", "not-in-engine-map", "оpus"} {
		input = passingInput()
		input.MakerEngine = "claude"
		input.Verdict["byEngine"] = bad
		c2 := clauseResult(t, Enforce(input), 2)
		if c2.Pass || !strings.Contains(reason(c2), "not a recognized engine") {
			t.Fatalf("unknown/confusable checker %q should fail closed: %#v", bad, c2)
		}
	}

	input = passingInput()
	input.MakerEngine = "phantom"
	input.Verdict["byEngine"] = "codex"
	if c2 := clauseResult(t, Enforce(input), 2); c2.Pass || !strings.Contains(reason(c2), "not a recognized engine") {
		t.Fatalf("unknown maker engine should fail closed: %#v", c2)
	}
}

func TestClauseMakerCheckerSessionConfusableFreshness(t *testing.T) {
	input := passingInput()
	input.MakerEngine = "codex"
	input.MakerSession = "sMaker"
	input.Verdict["byEngine"] = "codex"
	input.Verdict["bySession"] = "sMaker"
	if c2 := clauseResult(t, Enforce(input), 2); c2.Pass || !strings.Contains(reason(c2), "maker's own engine+session") {
		t.Fatalf("same engine same ASCII session should fail: %#v", c2)
	}

	input = passingInput()
	input.MakerEngine = "codex"
	input.MakerSession = "sMaker"
	input.Verdict["byEngine"] = "codex"
	input.Verdict["bySession"] = "ѕMaker"
	c2 := clauseResult(t, Enforce(input), 2)
	if c2.Pass || !strings.Contains(reason(c2), "not a valid session id") {
		t.Fatalf("confusable checker session should not establish freshness: %#v", c2)
	}

	for _, session := range []string{
		"otherSession",
		"0195e3a1b2c0-a1b2c3d4",
		"3dabc66c-862f-4c77-9c3f-f6bf050319e0",
		"sMakerl",
		"sMaker1",
		"sMakerO",
		"sMaker0",
	} {
		input = passingInput()
		input.MakerEngine = "codex"
		input.MakerSession = "sMaker"
		input.Verdict["byEngine"] = "codex"
		input.Verdict["bySession"] = session
		if c2 := clauseResult(t, Enforce(input), 2); !c2.Pass {
			t.Fatalf("strict ASCII fresh session %q should pass: %#v", session, c2)
		}
	}

	for _, session := range []string{"ѕMaker", "оtherodd", "sess fresh", "sess;rm -rf", "sess$(x)", "s#fresh", "sess\tfresh", "éfresh", "sess０１"} {
		input = passingInput()
		input.MakerEngine = "codex"
		input.MakerSession = "sMaker"
		input.Verdict["byEngine"] = "codex"
		input.Verdict["bySession"] = session
		c2 := clauseResult(t, Enforce(input), 2)
		if c2.Pass || !strings.Contains(reason(c2), "not a valid session id") {
			t.Fatalf("non-strict session %q should fail closed: %#v", session, c2)
		}
	}

	for _, session := range []string{"", "   ", "\t\n"} {
		input = passingInput()
		input.MakerEngine = "codex"
		input.MakerSession = "sMaker"
		input.Verdict["byEngine"] = "codex"
		input.Verdict["bySession"] = session
		c2 := clauseResult(t, Enforce(input), 2)
		if c2.Pass || !strings.Contains(reason(c2), "provenance") {
			t.Fatalf("empty session %q should fail as missing provenance: %#v", session, c2)
		}
	}

	input = passingInput()
	input.MakerEngine = "codex"
	input.MakerSession = "sMaker"
	input.Verdict["byEngine"] = "claude"
	input.Verdict["bySession"] = "ѕMaker"
	if c2 := clauseResult(t, Enforce(input), 2); !c2.Pass {
		t.Fatalf("different known engine should pass even with confusable session: %#v", c2)
	}
}

func TestClauseMakerCheckerSafetyAndOverrideComposition(t *testing.T) {
	for _, bySession := range []string{"otherSession", "ѕMaker"} {
		input := passingInput()
		input.ChangedFiles = []string{"cli/lib/auth/login.js"}
		input.Spec = specWith("cli/lib/auth/login.js")
		input.MakerEngine = "codex"
		input.MakerSession = "sMaker"
		input.Verdict["byEngine"] = "codex"
		input.Verdict["bySession"] = bySession
		c2 := clauseResult(t, Enforce(input), 2)
		if c2.Pass || !strings.Contains(reason(c2), "DIFFERENT engine") {
			t.Fatalf("safety same-engine session %q should fail different-engine bar: %#v", bySession, c2)
		}
	}

	input := passingInput()
	input.ChangedFiles = []string{"cli/lib/auth/login.js"}
	input.Spec = specWith("cli/lib/auth/login.js")
	input.MakerEngine = "codex"
	input.MakerSession = "sMaker"
	input.Verdict["byEngine"] = "claude"
	input.Verdict["bySession"] = "ѕMaker"
	if c2 := clauseResult(t, Enforce(input), 2); !c2.Pass {
		t.Fatalf("safety different-engine verdict should pass, sessions irrelevant: %#v", c2)
	}

	flaggedPatch := diffOf("cli/lib/x.test.js", []string{"-it('t', () => {});", "+it.skip('t', () => {});"})
	input = passingInput()
	input.Patch = flaggedPatch
	input.MakerEngine = "claude"
	input.MakerSession = "sess-maker"
	input.Verdict["byEngine"] = "claude "
	input.Verdict["bySession"] = "sess-maker"
	input.Verdict["tamperingReviewed"] = true
	input.Verdict["tamperingReason"] = "claims a review but is self attached"
	result := Enforce(input)
	if c4 := clauseResult(t, result, 4); !c4.Pass {
		t.Fatalf("tampering override should pass clause 4: %#v", c4)
	}
	if c2 := clauseResult(t, result, 2); c2.Pass || !strings.Contains(reason(c2), "maker's own engine+session") {
		t.Fatalf("tampering override must not bypass clause 2: %#v", c2)
	}
}

func TestClauseSafetyCarveOut(t *testing.T) {
	input := passingInput()
	input.ChangedFiles = []string{"src/auth/login.go"}
	input.Spec = specWith("src/auth/login.go")
	input.Patch = "+ validate password token"
	input.Verdict["byEngine"] = "codex"
	c3 := clauseResult(t, Enforce(input), 3)
	if c3.Pass || !strings.Contains(reason(c3), "safety carve-out") {
		t.Fatalf("missing safety attestation got %#v", c3)
	}
	input.Verdict["safetyReviewed"] = true
	input.Verdict["safetyReason"] = "reviewed auth boundary"
	if c3 := clauseResult(t, Enforce(input), 3); !c3.Pass {
		t.Fatalf("safety attestation should pass: %#v", c3)
	}
}

func TestAttestationsFailClosedOnNonStrictFlagsAndJunkReasons(t *testing.T) {
	safetyFlags := []any{false, "true", 1, map[string]any{}, nil}
	for _, flag := range safetyFlags {
		t.Run("safety flag", func(t *testing.T) {
			input := passingInput()
			input.ChangedFiles = []string{"src/auth/login.go"}
			input.Spec = specWith("src/auth/login.go")
			input.Patch = "+ validate password token"
			input.Verdict["byEngine"] = "codex"
			input.Verdict["safetyReviewed"] = flag
			input.Verdict["safetyReason"] = "auth token flow reviewed for leakage"
			c3 := clauseResult(t, Enforce(input), 3)
			if c3.Pass {
				t.Fatalf("non-strict safetyReviewed=%#v passed clause 3", flag)
			}
		})
	}

	for _, junk := range []string{"", ".", "ab", "12345678"} {
		t.Run("adversarial reason "+junk, func(t *testing.T) {
			input := passingInput()
			input.Spec = specWith("cli/lib/calc.js")
			input.ChangedFiles = []string{"cli/lib/calc.js"}
			input.Patch = diffOf("cli/lib/calc.js", []string{"+if (a > b) return a;"})
			input.Verdict["byEngine"] = "codex"
			input.Verdict["adversarialReviewed"] = true
			input.Verdict["adversarialReason"] = junk
			c1 := clauseResult(t, Enforce(input), 1)
			if c1.Pass {
				t.Fatalf("junk adversarialReason=%q passed clause 1", junk)
			}
		})
	}

	input := passingInput()
	input.Patch = diffOf("cli/lib/calc.test.js", []string{"-it('x', () => {})", "+it.skip('x', () => {})"})
	input.Verdict["tamperingReviewed"] = "true"
	input.Verdict["tamperingReason"] = "looks like a real review"
	if c4 := clauseResult(t, Enforce(input), 4); c4.Pass {
		t.Fatalf("non-strict tamperingReviewed string passed clause 4")
	}
}

func TestSafetyTierStacksWithAdversarialDefault(t *testing.T) {
	input := passingInput()
	input.ChangedFiles = []string{"src/auth/login.go"}
	input.Spec = specWith("src/auth/login.go")
	input.Patch = "+ validate password token"
	input.Verdict["byEngine"] = "codex"
	input.Verdict["safetyReviewed"] = true
	input.Verdict["safetyReason"] = "reviewed auth boundary"
	result := Enforce(input)
	if clauseResult(t, result, 3).Pass != true {
		t.Fatalf("safety clause should pass with safety attestation: %#v", result)
	}
	c1 := clauseResult(t, result, 1)
	if c1.Pass || !strings.Contains(reason(c1), "adversarialReviewed") {
		t.Fatalf("safety-tier diff should still need adversarial attestation: %#v", c1)
	}
	input.Verdict["adversarialReviewed"] = true
	input.Verdict["adversarialReason"] = "attacked sign-in path with forged token cases"
	if result := Enforce(input); !result.Pass {
		t.Fatalf("safety diff with both attestations should pass: %#v", result)
	}
}

func TestClauseAntiGaming(t *testing.T) {
	input := passingInput()
	input.Patch = diffOf("cli/lib/calc.test.js", []string{"-it('hard case', () => {})", "+it.skip('hard case', () => {})"})
	c4 := clauseResult(t, Enforce(input), 4)
	if c4.Pass || !strings.Contains(reason(c4), "tampering") {
		t.Fatalf("tampering patch got %#v", c4)
	}
	input.Verdict["tamperingReviewed"] = true
	input.Verdict["tamperingReason"] = "reviewed intentional test harness change"
	if c4 := clauseResult(t, Enforce(input), 4); !c4.Pass {
		t.Fatalf("tampering attestation should pass: %#v", c4)
	}
}

func TestClauseAntiGamingStructuralSignals(t *testing.T) {
	cases := []struct {
		name       string
		patch      string
		wantSignal string
		wantText   string
	}{
		{
			name:       "skip added",
			patch:      diffOf("cli/lib/calc.test.js", []string{"-it('x', () => {})", "+it.skip('x', () => {})"}),
			wantSignal: "a:skip-or-only-added",
			wantText:   "it.skip",
		},
		{
			name: "net assertion deletion",
			patch: diffOf("cli/lib/calc.test.js", []string{
				"-expect(a).toBe(1)",
				"-expect(b).toBe(2)",
				"-expect(c).toBe(3)",
				"+expect(a).toBe(1)",
			}),
			wantSignal: "b:net-assertion-deletion",
			wantText:   "3 assertion line(s) removed vs 1 added in this test file",
		},
		{
			// The net-zero cross-file swap: delete N real asserts in A, add N trivial
			// ones in B. Repo-wide count is even, but per-file accounting trips on A.
			// This is the leak Lane 3 closes.
			name: "net-zero cross-file assertion swap trips on the deleting file",
			patch: diffOf("cli/lib/a.test.js", []string{
				"-expect(a).toBe(1)",
				"-expect(b).toBe(2)",
				"-expect(c).toBe(3)",
			}) + "\n" + diffOf("cli/lib/b.test.js", []string{
				"+expect(1).toBe(1)",
				"+expect(1).toBe(1)",
				"+expect(1).toBe(1)",
			}),
			wantSignal: "b:net-assertion-deletion",
			wantText:   "cli/lib/a.test.js",
		},
		{
			// Two files each net-deleting -> one signal per file, sorted order.
			// wantText checks the alphabetically-first file is named.
			name: "two files each net-deleting emit per-file signals",
			patch: diffOf("cli/lib/a.test.js", []string{
				"-expect(a).toBe(1)",
			}) + "\n" + diffOf("cli/lib/z.test.js", []string{
				"-expect(z).toBe(9)",
			}),
			wantSignal: "b:net-assertion-deletion",
			wantText:   "cli/lib/a.test.js",
		},
		{
			name:       "gate disabled",
			patch:      diffOf("package.json", []string{`-  "tests": true,`, `+  "tests": false,`}),
			wantSignal: "c:gate-disabled",
			wantText:   `"tests": false`,
		},
		{
			name:       "coverage lowered",
			patch:      diffOf(".nycrc.json", []string{`-  "branches": 90,`, `+  "branches": 80,`}),
			wantSignal: "d:coverage-threshold-lowered",
			wantText:   "lowered 90",
		},
		{
			name:       "scorer edited",
			patch:      diffOf("vitest.setup.js", []string{`-globalThis.score = realScore`, `+globalThis.score = () => true`}),
			wantSignal: "e:scorer-or-runner-edited",
			wantText:   "vitest.setup.js",
		},
		{
			name:       "whole test file deleted",
			patch:      deletedFileDiff("cli/lib/calc.test.js", []string{"-expect(a).toBe(1)"}),
			wantSignal: "f:test-file-deleted",
			wantText:   "entire test file deleted",
		},
		{
			name:       "always false wrap",
			patch:      diffOf("cli/lib/calc.test.js", []string{"-it('x', () => {})", "+if (false) { it('x', () => {}) }"}),
			wantSignal: "g:test-disabled-wrap",
			wantText:   "if (false)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := passingInput()
			input.Patch = tc.patch
			c4 := clauseResult(t, Enforce(input), 4)
			if c4.Pass || !strings.Contains(reason(c4), tc.wantSignal) || !strings.Contains(reason(c4), tc.wantText) {
				t.Fatalf("clause 4 = %#v, want signal %s and text %q", c4, tc.wantSignal, tc.wantText)
			}
		})
	}
}

func TestClauseAntiGamingFalsePositiveBoundaries(t *testing.T) {
	cases := []struct {
		name  string
		patch string
	}{
		{
			name:  "skip in non-test source",
			patch: diffOf("cli/lib/queue.js", []string{"+queue.skip(next);"}),
		},
		{
			// A genuine 1:1 move WITHIN one file is per-file net zero -> no signal.
			// (The cross-FILE move used to live here as "benign"; per-file accounting
			// now correctly trips it, so it moved to the true-positive suite.)
			name:  "in-file 1:1 assertion refactor is net zero",
			patch: diffOf("cli/lib/a.test.js", []string{"-expect(a).toBe(1)", "+expect(a).toEqual(1)"}),
		},
		{
			name:  "comment containing assertion is ignored",
			patch: diffOf("cli/lib/a.test.js", []string{"-// expect(a).toBe(1)", "+// cleaned comment"}),
		},
		{
			name:  "string fixture containing assertion is ignored",
			patch: diffOf("cli/lib/a.test.js", []string{`-const fixture = "expect(a).toBe(1)"`, `+const fixture = "expect(a).toBe(2)"`}),
		},
		{
			name:  "test name mentioning skip is ignored",
			patch: diffOf("cli/lib/a.test.js", []string{`+test("documents how .skip behaves", () => {})`}),
		},
		{
			name:  "coverage raised is not tampering",
			patch: diffOf(".nycrc.json", []string{`-  "branches": 90,`, `+  "branches": 95,`}),
		},
		{
			name: "markdown quoted tampering diff is not parsed as test header",
			patch: diffOf("docs/anti-gaming.md", []string{
				"+```diff",
				"++++ b/cli/lib/a.test.js",
				"+@@ -1,1 +1,1 @@",
				"++it.skip('works', () => {});",
				"+```",
			}),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := passingInput()
			input.Patch = tc.patch
			c4 := clauseResult(t, Enforce(input), 4)
			if !c4.Pass {
				t.Fatalf("clause 4 trapped honest work: %#v", c4)
			}
		})
	}
}

func TestClauseAntiGamingParserAndCompositionParity(t *testing.T) {
	t.Run("collects multiple signals", func(t *testing.T) {
		patch := diffOf("jest.config.js", []string{"+const hack = 1;"}) + "\n" +
			diffOf("cli/lib/x.test.js", []string{"-it('t', () => {});", "+it.skip('t', () => {});"})
		c4 := clauseResult(t, Enforce(withPatch(patch)), 4)
		if c4.Pass || !strings.Contains(reason(c4), "e:scorer-or-runner-edited") || !strings.Contains(reason(c4), "a:skip-or-only-added") {
			t.Fatalf("expected both scorer and skip signals: %#v", c4)
		}
	})

	t.Run("headers are not content", func(t *testing.T) {
		patch := "diff --git a/cli/lib/router.js b/cli/lib/router.js\n" +
			"--- a/cli/lib/router.js\n" +
			"+++ b/cli/lib/router.js\n" +
			"@@ -1,1 +1,1 @@\n" +
			" unchanged context line"
		if c4 := clauseResult(t, Enforce(withPatch(patch)), 4); !c4.Pass {
			t.Fatalf("header-only patch should not trip clause 4: %#v", c4)
		}
	})

	t.Run("CRLF patch still flags tampering", func(t *testing.T) {
		lines := []string{
			"diff --git a/x.test.js b/x.test.js",
			"--- a/x.test.js",
			"+++ b/x.test.js",
			"@@ -1,1 +1,1 @@",
			"-it(\"x\", () => {})",
			"+it.skip(\"x\", () => {})",
		}
		c4 := clauseResult(t, Enforce(withPatch(strings.Join(lines, "\r\n"))), 4)
		if c4.Pass || !strings.Contains(reason(c4), "a:skip-or-only-added") {
			t.Fatalf("CRLF tampering diff should still flag: %#v", c4)
		}
	})

	t.Run("semantic assertion weakening remains verdict work", func(t *testing.T) {
		patch := diffOf("cli/lib/x.test.js", []string{"-  expect(result).toBe(42);", "+  expect(result).toBeDefined();"})
		if c4 := clauseResult(t, Enforce(withPatch(patch)), 4); !c4.Pass {
			t.Fatalf("semantic assertion weakening should not be structurally flagged: %#v", c4)
		}
	})

	t.Run("cross-engine tampering override can make whole floor pass", func(t *testing.T) {
		input := withPatch(diffOf("cli/lib/x.test.js", []string{"-it('t', () => {});", "+it.skip('t', () => {});"}))
		input.MakerEngine = "claude"
		input.MakerSession = "sess-maker"
		input.Verdict["byEngine"] = "codex"
		input.Verdict["bySession"] = "sess-checker"
		input.Verdict["tamperingReviewed"] = true
		input.Verdict["tamperingReason"] = "legit refactor independently reviewed"
		result := Enforce(input)
		if !result.Pass || len(result.Failed) != 0 {
			t.Fatalf("cross-engine tampering override should satisfy full floor: %#v", result)
		}
	})

	t.Run("same-engine same-session tampering override leaves clause 2 red", func(t *testing.T) {
		input := withPatch(diffOf("cli/lib/x.test.js", []string{"-it('t', () => {});", "+it.skip('t', () => {});"}))
		input.MakerEngine = "claude"
		input.MakerSession = "sess-maker"
		input.Verdict["byEngine"] = "claude"
		input.Verdict["bySession"] = "sess-maker"
		input.Verdict["tamperingReviewed"] = true
		input.Verdict["tamperingReason"] = "claims a review but self attached"
		result := Enforce(input)
		if c4 := clauseResult(t, result, 4); !c4.Pass {
			t.Fatalf("clause 4 should honor local override: %#v", c4)
		}
		if c2 := clauseResult(t, result, 2); c2.Pass || !strings.Contains(reason(c2), "maker's own engine+session") {
			t.Fatalf("clause 2 should still reject self-attached override: %#v", c2)
		}
		if !reflect.DeepEqual(result.Failed, []int{2}) {
			t.Fatalf("expected only clause 2 failed, got %#v", result.Failed)
		}
	})
}

func TestClauseAntiGamingSkipAssertionAndWrapEdgeParity(t *testing.T) {
	t.Run("skip and if false mentions in comments names and strings do not flag", func(t *testing.T) {
		lines := []string{
			"+  // do not use it.skip in this suite",
			"+test('documents how .skip behaves', () => {",
			`+  const m = "pass .only the id";`,
			"+test('handles if (false) in the input', () => {",
			"+  // we removed the old if (false) wrap",
		}
		for _, line := range lines {
			patch := diffOf("x.test.js", []string{" suite(() => {", line})
			if c4 := clauseResult(t, Enforce(withPatch(patch)), 4); !c4.Pass {
				t.Fatalf("honest skip/wrap mention flagged for %q: %#v", line, c4)
			}
		}
	})

	t.Run("real skip and if false code still flag", func(t *testing.T) {
		cases := []struct {
			name string
			line string
			want string
		}{
			{"skip", `+it.skip("x", () => {})`, "a:skip-or-only-added"},
			{"wrap false", "+  if (false) {", "g:test-disabled-wrap"},
			{"wrap zero", "+  if ( 0 ) {", "g:test-disabled-wrap"},
		}
		for _, tc := range cases {
			patch := diffOf("x.test.js", []string{" it(\"x\", () => {", tc.line})
			c4 := clauseResult(t, Enforce(withPatch(patch)), 4)
			if c4.Pass || !strings.Contains(reason(c4), tc.want) {
				t.Fatalf("%s did not flag %s: %#v", tc.name, tc.want, c4)
			}
		}
	})

	t.Run("assertion token inside removed string fixture is not deletion", func(t *testing.T) {
		for _, removed := range []string{`-  parse("assert x == 1");`, `-  const f = "expect(x).toBe(1)";`} {
			patch := diffOf("lint.test.js", []string{" test(\"parses\", () => {", removed, " });"})
			if c4 := clauseResult(t, Enforce(withPatch(patch)), 4); !c4.Pass {
				t.Fatalf("string assertion fixture removal flagged for %q: %#v", removed, c4)
			}
		}
	})

	t.Run("real assertion removal still flags with string token argument", func(t *testing.T) {
		patch := diffOf("a.test.js", []string{" test(\"x\", () => {", `-  expect(parse("assert y")).toBe(1);`, " });"})
		c4 := clauseResult(t, Enforce(withPatch(patch)), 4)
		if c4.Pass || !strings.Contains(reason(c4), "b:net-assertion-deletion") {
			t.Fatalf("real assertion with string token did not flag: %#v", c4)
		}
	})

	t.Run("entire test file deletion and non-test deletion boundaries", func(t *testing.T) {
		c4 := clauseResult(t, Enforce(withPatch(deletedFileDiff("cli/lib/gone.test.js", []string{"-test('was here', () => {", "-  expect(a).toBe(1);", "-});"}))), 4)
		if c4.Pass || !strings.Contains(reason(c4), "f:test-file-deleted") {
			t.Fatalf("test file deletion should flag: %#v", c4)
		}
		c4 = clauseResult(t, Enforce(withPatch(deletedFileDiff("cli/lib/helper.js", []string{"-export function helper() {}"}))), 4)
		if !c4.Pass {
			t.Fatalf("non-test file deletion should not be test-file-deleted: %#v", c4)
		}
	})

	t.Run("broader skip syntax flags", func(t *testing.T) {
		cases := []struct {
			file string
			line string
		}{
			{"tests/test_x.py", `+@pytest.mark.skipif(sys.platform == "win32", reason="x")`},
			{"pkg/thing_test.go", "+    t.SkipNow()"},
			{"cli/lib/x.test.js", `+describe["skip"]("suite", () => {});`},
			{"cli/lib/y.test.js", `+it['only']('focused', () => {});`},
		}
		for _, tc := range cases {
			c4 := clauseResult(t, Enforce(withPatch(diffOf(tc.file, []string{" context line", tc.line, " trailing context"}))), 4)
			if c4.Pass || !strings.Contains(reason(c4), "a:skip-or-only-added") {
				t.Fatalf("skip syntax %s did not flag: %#v", tc.line, c4)
			}
		}
	})

	t.Run("commenting out an assertion flags as net deletion", func(t *testing.T) {
		patch := diffOf("cli/lib/x.test.js", []string{" test('disabled by comment', () => {", "-  expect(result).toBe(1);", "+  // expect(result).toBe(1);", " });"})
		c4 := clauseResult(t, Enforce(withPatch(patch)), 4)
		if c4.Pass || !strings.Contains(reason(c4), "b:net-assertion-deletion") {
			t.Fatalf("commented-out assertion should flag: %#v", c4)
		}
	})

	t.Run("ambiguous early return is not structurally flagged", func(t *testing.T) {
		patch := diffOf("cli/lib/x.test.js", []string{" test('early', () => {", "+  return;", "   expect(a).toBe(1);", " });"})
		if c4 := clauseResult(t, Enforce(withPatch(patch)), 4); !c4.Pass {
			t.Fatalf("early return should be verdict work, not structural tampering: %#v", c4)
		}
	})
}

func TestStringLiteralSuppressorRegexCommentDesyncParity(t *testing.T) {
	t.Run("quote regex before real skip no longer suppresses", func(t *testing.T) {
		lines := []string{
			`  const re = /"/; it.skip("x", () => {});`,
			`  const re = /["']/; describe.only("x", () => {});`,
			`  const re = /\"/; describe.only("x", () => {});`,
			`  const re = /'/; describe.only("x", () => {});`,
		}
		for _, line := range lines {
			if !isInStringLiteralOld(line, skipOnlyRE) {
				t.Fatalf("old suppressor fixture did not suppress as expected: %q", line)
			}
			if isInStringLiteral(line, skipOnlyRE) {
				t.Fatalf("hardened suppressor still hides real skip/only: %q", line)
			}
			c4 := clauseResult(t, Enforce(withPatch(diffOf("x.test.js", []string{" suite(() => {", "+" + strings.TrimSpace(line)}))), 4)
			if c4.Pass || !strings.Contains(reason(c4), "a:skip-or-only-added") {
				t.Fatalf("real skip/only after quote regex should flag: %#v", c4)
			}
		}
	})

	t.Run("regex and block comment apostrophes before real assertions no longer suppress", func(t *testing.T) {
		lines := []string{
			`x = /'/; expect(y).toBe(1)`,
			`x = /* it's fine */ y; expect(z).toBe(1)`,
		}
		for _, line := range lines {
			if !isInStringLiteralOld(line, assertionRE) {
				t.Fatalf("old suppressor fixture did not suppress as expected: %q", line)
			}
			if isInStringLiteral(line, assertionRE) {
				t.Fatalf("hardened suppressor still hides real assertion: %q", line)
			}
		}
	})

	t.Run("keywords genuinely inside strings remain suppressed", func(t *testing.T) {
		cases := []struct {
			line string
			re   *regexp.Regexp
		}{
			{`const m = "a test about .skip semantics";`, skipOnlyRE},
			{`const n = 'documents how .only works';`, skipOnlyRE},
			{`parse("assert x == 1");`, assertionRE},
		}
		for _, tc := range cases {
			if !isInStringLiteral(tc.line, tc.re) {
				t.Fatalf("genuine string keyword should remain suppressed: %q", tc.line)
			}
		}
		c4 := clauseResult(t, Enforce(withPatch(diffOf("x.test.js", []string{`+test("documents how .skip behaves", () => {`}))), 4)
		if !c4.Pass {
			t.Fatalf("test name mentioning .skip should not flag: %#v", c4)
		}
	})
}

func TestStringLiteralSuppressorMonotoneFuzz(t *testing.T) {
	tokens := []string{
		`/'/`, `/"/`, `/a/`, `/['"]/`, `/\"/`, `/[a-z]+/`,
		`'`, `"`, "`", `.skip`, `.only`, `it.skip`, `describe.only`,
		`expect(`, `assert`, `.toBe`, `//`, `/* `, ` */`, `*/`,
		`x`, `=`, `;`, ` `, `a`, `\`, `(`, `)`, `> b ? `, ` : `,
	}
	rng := uint32(0xC0FFEE)
	next := func() uint32 {
		rng = rng*1103515245 + 12345
		return rng
	}
	falseToTrue := 0
	trueToFalse := 0
	for i := 0; i < 10000; i++ {
		n := int(next()%12) + 1
		var b strings.Builder
		for j := 0; j < n; j++ {
			b.WriteString(tokens[int(next()%uint32(len(tokens)))])
		}
		line := b.String()
		for _, re := range []*regexp.Regexp{skipOnlyRE, assertionRE} {
			before := isInStringLiteralOld(line, re)
			after := isInStringLiteral(line, re)
			if !before && after {
				falseToTrue++
			}
			if before && !after {
				trueToFalse++
			}
		}
	}
	if falseToTrue != 0 {
		t.Fatalf("hardened suppressor introduced %d new false-to-true suppressions", falseToTrue)
	}
	if trueToFalse == 0 {
		t.Fatalf("fuzz did not exercise any true-to-false un-suppression cases")
	}
}

func TestClauseAntiGamingOverride(t *testing.T) {
	input := passingInput()
	input.Patch = diffOf("cli/lib/calc.test.js", []string{"-it('x', () => {})", "+it.skip('x', () => {})"})
	flagged := clauseResult(t, Enforce(input), 4)
	if flagged.Pass || !strings.Contains(reason(flagged), "a:skip-or-only-added") {
		t.Fatalf("expected flagged tampering before override: %#v", flagged)
	}
	input.Verdict["tamperingReviewed"] = true
	input.Verdict["tamperingReason"] = "obsolete test for removed feature, independently reviewed"
	overridden := clauseResult(t, Enforce(input), 4)
	if !overridden.Pass {
		t.Fatalf("expected tampering override to pass clause 4: %#v", overridden)
	}
}

func TestIsSafetyTier(t *testing.T) {
	cases := []struct {
		name  string
		input Input
	}{
		{"protected term security", Input{Patch: "+ security check changed"}},
		{"protected term trust-boundary", Input{Patch: "+ trust-boundary validation changed"}},
		{"protected term validation", Input{Patch: "+ validation rule changed"}},
		{"protected term accessibility", Input{Patch: "+ accessibility labels changed"}},
		{"protected term data-loss", Input{Patch: "+ data-loss fallback changed"}},
		{"auth synonym", Input{Patch: "+ auth token validation changed"}},
		{"concrete credential markers", Input{Patch: "+ rotate secret credential token"}},
		{"risky path auth", Input{ChangedFiles: []string{"src/auth/login.go"}}},
		{"risky path jwt", Input{ChangedFiles: []string{"src/jwtVerify.go"}}},
		{"risky path payment", Input{ChangedFiles: []string{"internal/payment/charge.go"}}},
		{"spec declared", Input{Spec: "<task><risk> safety </risk></task>"}},
		{"xss sink", Input{Patch: "+ element.innerHTML = req.body.name"}},
		{"command sink", Input{Patch: "+ child_process.execSync(cmd)"}},
		{"sql sink", Input{Patch: "+ SELECT * FROM accounts WHERE id = x"}},
		{"redirect sink", Input{Patch: "+ res.redirect(req.query.next)"}},
		{"tls disabled", Input{Patch: "+ rejectUnauthorized: false"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !IsSafetyTier(tc.input) {
				t.Fatalf("IsSafetyTier(%#v) = false, want true", tc.input)
			}
		})
	}
	nonSafety := []struct {
		name  string
		input Input
	}{
		{"plain session path", Input{ChangedFiles: []string{"src/session/state.go"}, Patch: "+ return a + b"}},
		{"weak data term", Input{Patch: "+ data model changed"}},
		{"weak user term", Input{Patch: "+ user display name changed"}},
		{"weak key term", Input{Patch: "+ map key changed"}},
		{"space separated trust boundary", Input{Patch: "+ trust boundary text changed"}},
		{"broad api path excluded", Input{ChangedFiles: []string{"src/api/users.go"}}},
	}
	for _, tc := range nonSafety {
		t.Run("non-safety "+tc.name, func(t *testing.T) {
			if IsSafetyTier(tc.input) {
				t.Fatalf("IsSafetyTier(%#v) = true, want false", tc.input)
			}
		})
	}
}

func TestIsSafetyTierResidualSecurityMatrix(t *testing.T) {
	residuals := []struct {
		label string
		path  string
		patch string
	}{
		{"JWT alg none", "lib/auth/jwt.js", `+ const decoded = decode(t, { algorithms: ["none"] });`},
		{"SSRF", "src/crypto.js", "+ const r = await fetch(input.url);"},
		{"prototype merge", "lib/auth/merge.js", "+ for (const k in src) dst[k] = src[k];"},
		{"ReDoS regex", "src/validate/email.js", "+ const ok = /(a+)+$/.test(s);"},
		{"authz bypass", "lib/authz/guard.js", "+ if (ctx.role || true) return allow();"},
		{"weak hash", "src/crypto/hash.js", "+ const h = md5(input);"},
		{"hardcoded key", "lib/secret/keys.js", `+ const k = "9f8e7d6c5b4a";`},
		{"path traversal", "lib/permission/files.js", "+ const p = base + req.name;"},
		{"mass assignment", "lib/payment/account.js", "+ Object.assign(acct, body);"},
		{"csrf off", "lib/csrf/mw.js", "+ const guard = (r, x, next) => next();"},
		{"weak random id", "lib/crypto/id.js", "+ const id = Math.random().toString(36);"},
		{"PII logging", "src/billing/log.js", "+ logger.info(ssn, dob, pan);"},
	}
	for _, tc := range residuals {
		t.Run(tc.label+" content alone", func(t *testing.T) {
			if IsSafetyTier(Input{ChangedFiles: []string{"lib/innocuous.js"}, Patch: tc.patch}) {
				t.Fatalf("content-only residual safety diff should not be classified without path/spec signal")
			}
		})
		t.Run(tc.label+" risky path", func(t *testing.T) {
			if !IsSafetyTier(Input{ChangedFiles: []string{tc.path}, Patch: tc.patch}) {
				t.Fatalf("risky path %q did not classify as safety", tc.path)
			}
			input := passingInput()
			input.ChangedFiles = []string{tc.path}
			input.Spec = specWith(tc.path)
			input.Patch = tc.patch
			input.Verdict["byEngine"] = "claude"
			input.Verdict["bySession"] = "sess-fresh"
			result := Enforce(input)
			if c2 := clauseResult(t, result, 2); c2.Pass || !strings.Contains(reason(c2), "DIFFERENT engine") {
				t.Fatalf("risky safety path should require different engine, got %#v", c2)
			}
			if c3 := clauseResult(t, result, 3); c3.Pass || !strings.Contains(reason(c3), "safety carve-out") {
				t.Fatalf("risky safety path should require safety review, got %#v", c3)
			}
		})
		t.Run(tc.label+" spec declared", func(t *testing.T) {
			spec := `<task><risk>safety</risk><scope><can-modify><path>lib/innocuous.js</path></can-modify></scope></task>`
			if !IsSafetyTier(Input{ChangedFiles: []string{"lib/innocuous.js"}, Patch: tc.patch, Spec: spec}) {
				t.Fatalf("spec-declared safety did not classify as safety")
			}
		})
	}
}

func TestIsSafetyTierDangerousSinkMarkers(t *testing.T) {
	sinks := []struct {
		label string
		file  string
		patch string
	}{
		{"SQL injection", "cli/lib/db.js", `+ const q = "SELECT id FROM accounts WHERE id=" + uid;`},
		{"XSS innerHTML sink", "cli/lib/view.js", "+ el.innerHTML = name;"},
		{"command injection execSync", "cli/lib/git.js", `+ execSync("git checkout " + branch);`},
		{"new Function eval", "cli/lib/calc.js", `+ const f = new Function("return " + expr);`},
		{"insecure cookie flag", "cli/lib/session.js", `+ res.cookie("sid", id, { httpOnly: false });`},
		{"wide CORS", "cli/lib/server.js", `+ res.setHeader("Access-Control-Allow-Origin", "*");`},
		{"untrusted req.query source", "cli/lib/handler.js", "+ const id = req.query.id;"},
		{"open redirect", "cli/lib/nav.js", "+ res.redirect(target);"},
		{"TLS verification disabled", "cli/lib/client.js", "+ const opts = { rejectUnauthorized: false };"},
		{"child_process spawn", "cli/lib/proc.js", `+ const cp = require("child_process");`},
	}
	for _, tc := range sinks {
		t.Run(tc.label, func(t *testing.T) {
			if !IsSafetyTier(Input{ChangedFiles: []string{tc.file}, Patch: tc.patch}) {
				t.Fatalf("sink case did not classify safety: %#v", tc)
			}
			input := passingInput()
			input.ChangedFiles = []string{tc.file}
			input.Spec = specWith(tc.file)
			input.Patch = tc.patch
			input.MakerEngine = "claude"
			input.MakerSession = "sess-maker"
			input.Verdict["byEngine"] = "claude"
			input.Verdict["bySession"] = "sess-fresh"
			c2 := clauseResult(t, Enforce(input), 2)
			if c2.Pass || !strings.Contains(reason(c2), "DIFFERENT engine") {
				t.Fatalf("same-engine safety sink should fail different-engine bar: %#v", c2)
			}
		})
	}

	plain := Input{
		ChangedFiles: []string{"cli/lib/format.js"},
		Patch:        "+ export function pad(s, n) { return s.padEnd(n); }",
	}
	if IsSafetyTier(plain) {
		t.Fatalf("plain util refactor classified as safety: %#v", plain)
	}
	tooBroad := Input{
		ChangedFiles: []string{"cli/lib/timer.js"},
		Patch:        "+ setTimeout(fn, 100);\n+ const s = a + \"-\" + b;\n+ import x from \"../util.js\";",
	}
	if IsSafetyTier(tooBroad) {
		t.Fatalf("deferred broad markers classified as safety: %#v", tooBroad)
	}
}

func TestIsSafetyTierRiskDeclarationBoundaries(t *testing.T) {
	for _, decl := range []string{
		"<risk>safety</risk>",
		"<RISK>SAFETY</RISK>",
		"<risk> safety </risk>",
		"<Risk>\n  safety\n</Risk>",
	} {
		t.Run(decl, func(t *testing.T) {
			if !IsSafetyTier(Input{ChangedFiles: []string{"lib/helper.js"}, Patch: "+ x;", Spec: "<task>" + decl + "</task>"}) {
				t.Fatalf("risk declaration %q did not classify as safety", decl)
			}
		})
	}
	nonSafety := []Input{
		{ChangedFiles: []string{"lib/helper.js"}, Patch: "+ x;", Spec: "<task><risk>standard</risk></task>"},
		{ChangedFiles: []string{"lib/helper.js"}, Patch: "+ x;", Spec: "<task><notes>safety matters</notes></task>"},
		{ChangedFiles: []string{"cli/lib/session.js"}, Patch: "+ const id = hex();"},
		{ChangedFiles: []string{"db/sessionPool.js"}, Patch: "+ x;"},
	}
	for _, input := range nonSafety {
		if IsSafetyTier(input) {
			t.Fatalf("non-safety input classified as safety: %#v", input)
		}
	}
	if !IsSafetyTier(Input{ChangedFiles: []string{"lib/auth/session.js"}, Patch: "+ x;"}) {
		t.Fatal("auth session path should classify as safety through auth segment")
	}
	if !IsSafetyTier(Input{ChangedFiles: []string{"lib/session.js"}, Patch: "+ const token = signJwt(user);"}) {
		t.Fatal("session file with token content should classify as safety through content")
	}
}

func passingInput() Input {
	return Input{
		Spec:         specWith("cli/lib/floor-kernel.js", "cli/lib/floor-kernel.test.js"),
		ChangedFiles: []string{"cli/lib/floor-kernel.js"},
		Gates:        &GateResult{OK: true},
		Verdict: map[string]any{
			"accepted":  true,
			"byEngine":  "claude",
			"bySession": "sess-checker",
		},
		MakerEngine:   "claude",
		MakerSession:  "sess-maker",
		TerminalState: "done-verified",
	}
}

func withPatch(patch string) Input {
	input := passingInput()
	input.Patch = patch
	return input
}

func isInStringLiteralOld(content string, re *regexp.Regexp) bool {
	matches := re.FindAllStringIndex(content, -1)
	if len(matches) == 0 {
		return false
	}
	inString := make([]bool, len(content))
	var quote byte
	for i := 0; i < len(content); i++ {
		c := content[i]
		if quote != 0 {
			if c == '\\' {
				i++
				continue
			}
			if c == quote {
				quote = 0
			} else {
				inString[i] = true
			}
			continue
		}
		if c == '"' || c == '\'' || c == '`' {
			quote = c
		}
	}
	for _, span := range matches {
		for i := span[0]; i < span[1]; i++ {
			if i >= len(inString) || !inString[i] {
				return false
			}
		}
	}
	return true
}

func specWith(paths ...string) string {
	var b strings.Builder
	b.WriteString("<task><scope><can-modify>\n")
	for _, path := range paths {
		b.WriteString("<path>")
		b.WriteString(path)
		b.WriteString("</path>\n")
	}
	b.WriteString("</can-modify></scope></task>")
	return b.String()
}

func diffOf(file string, lines []string) string {
	added, removed := 0, 0
	for _, line := range lines {
		if strings.HasPrefix(line, "+") {
			added++
		}
		if strings.HasPrefix(line, "-") {
			removed++
		}
	}
	return "diff --git a/" + file + " b/" + file + "\n" +
		"--- a/" + file + "\n" +
		"+++ b/" + file + "\n" +
		"@@ -1," + strconv.Itoa(removed+1) + " +1," + strconv.Itoa(added+1) + " @@\n" +
		strings.Join(lines, "\n")
}

func deletedFileDiff(file string, lines []string) string {
	removed := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "-") {
			removed++
		}
	}
	return "diff --git a/" + file + " b/" + file + "\n" +
		"--- a/" + file + "\n" +
		"+++ /dev/null\n" +
		"@@ -1," + strconv.Itoa(removed+1) + " +0,0 @@\n" +
		strings.Join(lines, "\n")
}

func clauseResult(t *testing.T, result Result, n int) ClauseResult {
	t.Helper()
	for _, clause := range result.Clauses {
		if clause.Clause == n {
			return clause
		}
	}
	t.Fatalf("missing clause %d in %#v", n, result.Clauses)
	return ClauseResult{}
}

func reason(clause ClauseResult) string {
	if clause.Reason == nil {
		return ""
	}
	return *clause.Reason
}

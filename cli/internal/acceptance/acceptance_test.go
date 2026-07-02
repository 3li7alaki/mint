package acceptance

import "testing"

func TestParseEARSPatterns(t *testing.T) {
	body := accept(`- THE system SHALL boot
- WHEN x happens, THE system SHALL react
- WHILE busy, THE system SHALL queue
- WHERE flag set, THE system SHALL branch
- IF y, THEN THE system SHALL recover
- GIVEN a, WHEN b, THEN c`)
	result := Parse(body)
	if len(result.Errors) != 0 || len(result.Warnings) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", result)
	}
	want := []string{"ubiquitous", "event", "state", "optional", "unwanted", "gwt"}
	if len(result.Criteria) != len(want) {
		t.Fatalf("criteria = %#v", result.Criteria)
	}
	for i, pattern := range want {
		if result.Criteria[i].Pattern != pattern {
			t.Fatalf("criterion %d pattern = %q, want %q", i, result.Criteria[i].Pattern, pattern)
		}
	}
}

func TestIsGateRestatement(t *testing.T) {
	for _, line := range []string{
		"lint ✅ types ✅ tests ✅",
		"tests pass",
		"coverage ≥ 80%",
		"lint and types and tests",
		"tests must pass",
		"make sure lint is clean",
		"typecheck succeeds",
	} {
		if !IsGateRestatement(line) {
			t.Fatalf("IsGateRestatement(%q) = false, want true", line)
		}
	}
	for _, line := range []string{
		"Cover all types",
		"Add tests",
		"WHEN the suite runs, THE kernel SHALL keep 587 tests green",
		"THE system SHALL lint user input",
		"No regressions in existing functionality",
		"",
	} {
		if IsGateRestatement(line) {
			t.Fatalf("IsGateRestatement(%q) = true, want false", line)
		}
	}
}

func TestParseGateRestatementAndProse(t *testing.T) {
	result := Parse(accept("- THE parser SHALL run\n- lint ✅ types ✅ tests ✅"))
	if len(result.Criteria) != 1 || result.Criteria[0].Pattern != "ubiquitous" {
		t.Fatalf("criteria = %#v", result.Criteria)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("errors = %#v", result.Errors)
	}

	onlyGate := Parse(accept("- tests pass"))
	if len(onlyGate.Criteria) != 0 || !has(onlyGate.Errors, "acceptance has no criteria") {
		t.Fatalf("only gate result = %#v", onlyGate)
	}

	prose := Parse(accept("- No regressions in existing functionality"))
	if len(prose.Errors) != 0 || len(prose.Warnings) != 1 || prose.Criteria[0].Pattern != "prose" {
		t.Fatalf("prose result = %#v", prose)
	}
}

func TestParseLineExtraction(t *testing.T) {
	result := Parse(accept("- THE a SHALL one\n\n1. THE b SHALL two\n2) THE c SHALL three"))
	if len(result.Criteria) != 3 {
		t.Fatalf("criteria = %#v", result.Criteria)
	}
	if *result.Criteria[0].System != "a" || *result.Criteria[1].System != "b" || *result.Criteria[2].System != "c" {
		t.Fatalf("systems = %#v", result.Criteria)
	}
}

func accept(body string) string {
	return "<task><acceptance>\n" + body + "\n</acceptance></task>"
}

func has(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

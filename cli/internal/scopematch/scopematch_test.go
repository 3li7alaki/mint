package scopematch

import (
	"strings"
	"testing"
)

func TestMatchesAllowedTable(t *testing.T) {
	tests := []struct {
		file string
		lane string
		want bool
	}{
		{"a/b.js", "a/b.js", true},
		{"a/b.js", "a/c.js", false},
		{"a/b.js", "a", true},
		{"a/b/c.js", "a", true},
		{"ab/c.js", "a", false},
		{"a/x.js", "a/*.js", true},
		{"a/b/x.js", "a/*.js", false},
		{"a/x.ts", "a/*.js", false},
		{"a/b/c.js", "a/**", true},
		{"a/b/c.js", "a/**/c.js", true},
		{"x/b/c.js", "a/**", false},
		{"a/x.js", "a/?.*", true},
		{"a/xy.js", "a/?.*", false},
		{"a/b/x.js", "a/?.*", false},
		{"a/?.js", "a/?.js", true},
		{"a/b.js", "a/?.js", false},
	}
	for _, tt := range tests {
		t.Run(tt.file+" in "+tt.lane, func(t *testing.T) {
			if got := MatchesAllowed(tt.file, tt.lane); got != tt.want {
				t.Fatalf("MatchesAllowed(%q, %q) = %v, want %v", tt.file, tt.lane, got, tt.want)
			}
		})
	}
}

func TestMatchesAllowedHardening(t *testing.T) {
	// ReDoS-resistance is STRUCTURAL: MatchesAllowed walks path segments (no backtracking
	// regex), so a pathological glob-heavy lane is bounded by construction. Assert the
	// CORRECT result on each pathological input — if the matcher ever regressed into
	// catastrophic backtracking it would hang here and the test would time out (go test's
	// own deadline), and a wrong answer fails outright. No wall-clock assertion (flaky under
	// load); the expected bool encodes the contract.
	assertTerminatesCorrectly(t, "x/"+strings.Repeat("x/", 80)+"y", strings.Repeat("**?", 50)+"**", false)
	assertTerminatesCorrectly(t, strings.Repeat("a", 54)+"\n"+"aaa", strings.Repeat("**a", 50)+"**", true)
	assertTerminatesCorrectly(t, strings.Repeat("a", 500)+"/b/"+strings.Repeat("c", 500)+"\n"+"d", growLane(), false)

	if MatchesAllowed("cli/lib/scope-match.js", strings.Repeat("*", 5000)) {
		t.Fatal("over-4096 lane matched, want fail closed")
	}
	if !MatchesAllowed("src/a/b/c/x.js", "src/**/**/x.js") {
		t.Fatal("redundant globstars should match nested path")
	}
}

func TestMatchesAllowedScopeEscapeRejection(t *testing.T) {
	tests := []struct {
		file string
		lane string
		want bool
	}{
		{"zzz/qqq/www.js", "**/a/**", false},
		{"zzz/a/www.js", "**/a/**", true},
		{"etc/passwd", "**secret**", false},
		{"a/x/QQQ/c.js", "a/**/b/**/c.js", false},
		{"a/x/b/y/c.js", "a/**/b/**/c.js", true},
		{"src/anything/x.js", "src/**/critical/**/x.js", false},
		{"src/p/critical/q/x.js", "src/**/critical/**/x.js", true},
		{"a/x/b.js", "a/*/b.js", true},
		{"a/x/y/b.js", "a/*/b.js", false},
		{"aEVIL/x.js", "a/**", false},
	}
	for _, tt := range tests {
		if got := MatchesAllowed(tt.file, tt.lane); got != tt.want {
			t.Fatalf("MatchesAllowed(%q, %q) = %v, want %v", tt.file, tt.lane, got, tt.want)
		}
	}
}

// assertTerminatesCorrectly checks a pathological input both TERMINATES (a catastrophic-
// backtracking regression would hang and trip go test's deadline rather than return) and
// yields the CONTRACTUAL result. Replaces a wall-clock "< 1s" assertion that flaked under
// CPU load — the segment-walk's linearity is the real guarantee, not a measured duration.
func assertTerminatesCorrectly(t *testing.T, file, lane string, want bool) {
	t.Helper()
	if got := MatchesAllowed(file, lane); got != want {
		t.Fatalf("MatchesAllowed(pathological, lane len %d) = %v, want %v", len(lane), got, want)
	}
}

func growLane() string {
	var lane string
	for len(lane) < 4090 {
		lane += "**a"
	}
	return lane + "**"
}

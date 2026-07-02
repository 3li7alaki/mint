package reviewcmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestReviewListShowsCanonicalLenses(t *testing.T) {
	var out bytes.Buffer
	code, err := Run(nil, Flags{List: true}, &out, &bytes.Buffer{})
	if err != nil || code != 0 {
		t.Fatalf("Run list code=%d err=%v", code, err)
	}
	text := out.String()
	for _, name := range []string{"security", "quality", "performance", "conventions", "business", "test", "adversarial"} {
		if !strings.Contains(text, name) {
			t.Fatalf("list missing %s:\n%s", name, text)
		}
	}
	if !strings.Contains(text, "review lens the driver selects") {
		t.Fatalf("list missing summary:\n%s", text)
	}
}

func TestReviewLensByFlagAndPositional(t *testing.T) {
	var out bytes.Buffer
	code, err := Run(nil, Flags{Lens: "security"}, &out, &bytes.Buffer{})
	if err != nil || code != 0 || !strings.Contains(out.String(), "name: security") || !strings.Contains(out.String(), "security lens") {
		t.Fatalf("security code=%d err=%v out=%q", code, err, out.String())
	}
	out.Reset()
	code, err = Run([]string{"quality"}, Flags{}, &out, &bytes.Buffer{})
	if err != nil || code != 0 || !strings.Contains(out.String(), "name: quality") {
		t.Fatalf("quality code=%d err=%v out=%q", code, err, out.String())
	}
}

func TestReviewFocus(t *testing.T) {
	var out bytes.Buffer
	code, err := Run(nil, Flags{Focus: "retry-path error handling"}, &out, &bytes.Buffer{})
	if err != nil || code != 0 || !strings.Contains(out.String(), "ad-hoc review lens") || !strings.Contains(out.String(), "retry-path error handling") {
		t.Fatalf("focus code=%d err=%v out=%q", code, err, out.String())
	}
}

func TestReviewUnknownLensFails(t *testing.T) {
	var stderr bytes.Buffer
	code, err := Run(nil, Flags{Lens: "nonexistent"}, &bytes.Buffer{}, &stderr)
	if err != nil || code != 1 || !strings.Contains(stderr.String(), `No lens named "nonexistent"`) {
		t.Fatalf("unknown code=%d err=%v stderr=%q", code, err, stderr.String())
	}
}

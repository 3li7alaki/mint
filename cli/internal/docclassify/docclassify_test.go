package docclassify

import "testing"

func TestIsDocPath(t *testing.T) {
	tests := map[string]bool{
		"README.md":          true,
		"docs/guide/page.md": true,
		".mint/session.json": true,
		"image.png":          true,
		"src/main.go":        false,
		"cli/lib/a.js":       false,
	}
	for path, want := range tests {
		if got := IsDocPath(path); got != want {
			t.Fatalf("IsDocPath(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestIsDocsOnly(t *testing.T) {
	if IsDocsOnly(nil) {
		t.Fatal("empty change list should not be docs-only")
	}
	if !IsDocsOnly([]string{"README.md", "docs/a.md", ".mint/tasks/x.xml"}) {
		t.Fatal("docs/assets change should be docs-only")
	}
	if IsDocsOnly([]string{"README.md", "cmd/mint/main.go"}) {
		t.Fatal("mixed code/doc change should not be docs-only")
	}
}

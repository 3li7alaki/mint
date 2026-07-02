package notelist

import (
	"strings"
	"testing"
	"time"
)

func at() time.Time { return time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC) }

func TestAppendAccumulates(t *testing.T) {
	root := t.TempDir()
	if _, err := Append(root, "loader-bug", "first finding", nil, at()); err != nil {
		t.Fatal(err)
	}
	n, err := Append(root, "loader-bug", "second finding", nil, at())
	if err != nil {
		t.Fatal(err)
	}
	if n.Entries != 2 {
		t.Fatalf("entries = %d, want 2", n.Entries)
	}
	body, err := Body(root, "loader-bug")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "first finding") || !strings.Contains(body, "second finding") {
		t.Fatalf("body did not accumulate both entries:\n%s", body)
	}
	// One row per topic (not per entry).
	notes, _ := Read(root)
	if len(notes) != 1 {
		t.Fatalf("rows = %d, want 1 (topic-keyed)", len(notes))
	}
}

func TestTopicNormalization(t *testing.T) {
	root := t.TempDir()
	// Spaces → dashes, case-folded; "Loader Bug" and "loader-bug" are the same topic.
	Append(root, "Loader Bug", "x", nil, at())
	Append(root, "loader-bug", "y", nil, at())
	notes, _ := Read(root)
	if len(notes) != 1 || notes[0].Topic != "loader-bug" {
		t.Fatalf("normalization failed: %#v", notes)
	}
}

func TestInvalidTopicRejected(t *testing.T) {
	root := t.TempDir()
	// Path-traversal attempt must be rejected (topic becomes a filename).
	if _, err := Append(root, "../etc/passwd", "x", nil, at()); err == nil {
		t.Fatal("expected invalid-topic error for traversal")
	}
}

func TestNothingToRecordRejected(t *testing.T) {
	root := t.TempDir()
	if _, err := Append(root, "topic", "   ", nil, at()); err == nil {
		t.Fatal("expected error when no text and no files")
	}
}

func TestFilesMergeAcrossAppends(t *testing.T) {
	root := t.TempDir()
	Append(root, "bug", "a", []string{"x.go"}, at())
	n, _ := Append(root, "bug", "b", []string{"x.go", "y.go"}, at())
	if len(n.Files) != 2 {
		t.Fatalf("files not unioned: %#v", n.Files)
	}
}

func TestFilesOnlyEntry(t *testing.T) {
	root := t.TempDir()
	// A note with only file refs (no text) is valid and records the topic without bumping
	// the entry count (no body section written).
	n, err := Append(root, "scope", "", []string{"floor.go"}, at())
	if err != nil {
		t.Fatal(err)
	}
	if n.Entries != 0 || len(n.Files) != 1 {
		t.Fatalf("files-only note = %#v", n)
	}
}

func TestSummaryEmptyAndPopulated(t *testing.T) {
	root := t.TempDir()
	if Summary(root) != "" {
		t.Fatal("empty notes → empty summary")
	}
	Append(root, "topic-a", "x", nil, at())
	if !strings.Contains(Summary(root), "topic-a") {
		t.Fatalf("summary = %q", Summary(root))
	}
}

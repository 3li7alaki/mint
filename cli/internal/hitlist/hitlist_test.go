package hitlist

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fixedNow() time.Time { return time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC) }

func TestAddListLifecycle(t *testing.T) {
	root := t.TempDir()
	now := fixedNow()

	a, err := Add(root, "fix Y", "now", AddOpts{Session: "sess1"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "h1" || a.Status != "open" || a.Priority != "now" {
		t.Fatalf("add = %#v", a)
	}
	if _, err := Add(root, "refactor Z", "next", AddOpts{Session: "sess1"}, now); err != nil {
		t.Fatal(err)
	}

	// done h1
	if _, err := Done(root, "h1", now); err != nil {
		t.Fatal(err)
	}
	open, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 1 || open[0].ID != "h2" {
		t.Fatalf("open after done = %#v", open)
	}

	// done state persisted with doneAt
	all, _ := Read(root)
	if all[0].Status != "done" || all[0].DoneAt == "" {
		t.Fatalf("h1 not marked done: %#v", all[0])
	}
}

func TestPriorityOrdering(t *testing.T) {
	root := t.TempDir()
	now := fixedNow()
	// Add out of priority order; Open must return now → next → later.
	Add(root, "c", "later", AddOpts{Session: "s"}, now)
	Add(root, "a", "now", AddOpts{Session: "s"}, now)
	Add(root, "b", "next", AddOpts{Session: "s"}, now)
	open, _ := Open(root)
	got := []string{open[0].Text, open[1].Text, open[2].Text}
	want := []string{"a", "b", "c"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestDefaultAndInvalidPriority(t *testing.T) {
	root := t.TempDir()
	now := fixedNow()
	it, err := Add(root, "no prio", "", AddOpts{Session: "s"}, now)
	if err != nil || it.Priority != "next" {
		t.Fatalf("default priority = %q err=%v", it.Priority, err)
	}
	if _, err := Add(root, "bad", "urgent", AddOpts{Session: "s"}, now); err == nil {
		t.Fatal("expected error on invalid priority")
	}
}

func TestEmptyTextRejected(t *testing.T) {
	root := t.TempDir()
	if _, err := Add(root, "   ", "now", AddOpts{Session: "s"}, fixedNow()); err == nil {
		t.Fatal("expected error on empty text")
	}
}

func TestDoneUnknownID(t *testing.T) {
	root := t.TempDir()
	if _, err := Done(root, "h99", fixedNow()); err == nil {
		t.Fatal("expected error on unknown id")
	}
}

func TestPruneOldDone(t *testing.T) {
	root := t.TempDir()
	old := fixedNow()
	Add(root, "ancient", "now", AddOpts{Session: "s"}, old)
	Done(root, "h1", old)
	// A later Add prunes done items older than the retention window.
	later := old.Add(doneRetention + time.Hour)
	Add(root, "fresh", "now", AddOpts{Session: "s"}, later)
	all, _ := Read(root)
	for _, it := range all {
		if it.Text == "ancient" {
			t.Fatalf("old done hit was not pruned: %#v", all)
		}
	}
	if len(all) != 1 || all[0].Text != "fresh" {
		t.Fatalf("after prune = %#v", all)
	}
}

func TestSummary(t *testing.T) {
	root := t.TempDir()
	if Summary(root) != "" {
		t.Fatal("empty hitlist should summarize to empty string")
	}
	Add(root, "fix Y", "now", AddOpts{Session: "s"}, fixedNow())
	if got := Summary(root); got == "" || got[:4] != "📌" {
		t.Fatalf("summary = %q", got)
	}
}

func TestBodySpillsToFileAndReadsBack(t *testing.T) {
	root := t.TempDir()
	now := fixedNow()
	bigBody := "# Loader bug analysis\n\nTried X, didn't work.\nZ looks suspicious because…\n"
	it, err := Add(root, "loader bug", "now", AddOpts{
		Body:  bigBody,
		Files: []string{"cli/loader.go", "cli/merge.go"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	// Big content does NOT live in the row — it spills to a file, row carries bodyPath.
	if it.BodyPath == "" {
		t.Fatal("expected a bodyPath for a hit with a body")
	}
	if len(it.Files) != 2 {
		t.Fatalf("files = %#v", it.Files)
	}
	got, err := Body(root, it)
	if err != nil || got != bigBody {
		t.Fatalf("body round-trip: got=%q err=%v", got, err)
	}
	// The body is a real file on disk under .mint/hits/.
	if _, err := os.Stat(filepath.Join(root, it.BodyPath)); err != nil {
		t.Fatalf("body file missing: %v", err)
	}
}

func TestEmptyBodyWritesNoFile(t *testing.T) {
	root := t.TempDir()
	it, err := Add(root, "small hit", "now", AddOpts{Body: "  "}, fixedNow())
	if err != nil {
		t.Fatal(err)
	}
	if it.BodyPath != "" {
		t.Fatalf("empty body should not spill a file, got bodyPath=%q", it.BodyPath)
	}
}

func TestReadMissingFileIsEmpty(t *testing.T) {
	items, err := Read(t.TempDir())
	if err != nil || items != nil {
		t.Fatalf("missing file should be empty: items=%#v err=%v", items, err)
	}
}

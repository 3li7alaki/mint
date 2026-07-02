// Package hitlist is mint's durable mid-session agenda — the "work spawns work" backlog.
// Mid-task you notice Y is broken, Z should be refactored, the user asks for W: those
// intentions normally live in chat scrollback (dies on /clear) or someone's head. A hit
// is a discrete, checkable INTENTION with a lifecycle (open → done|dropped), stored in
// .mint/hitlist.jsonl so it survives the session and resurfaces at the right moment
// (session start, handoff seed, status). This is the human's agenda — distinct from
// execution/pipeline state (tracker, already command-owned) and from free-form reasoning
// (scratchpad, a separate future `mint note`). v1: capture + lifecycle + resurface.
package hitlist

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// relPath is the per-project hitlist file. COMMITTED, not gitignored — an open backlog is
// genuinely shared state ("would a teammate cloning the repo want this?" → yes).
const relPath = ".mint/hitlist.jsonl"

// doneRetention is how long a done/dropped hit is kept before self-cleaning, so the file
// doesn't grow forever (same decay discipline as the rest of mint's state).
const doneRetention = 14 * 24 * time.Hour

// Item is one hit. The ROW is a small JSONL index entry; big content does NOT live here.
// Text is the short title/intention. A substantial body (analysis, markdown, a writeup)
// spills to its own file at .mint/hits/<id>.md and is referenced by BodyPath — so the row
// stays greppable and the body stays a real readable/editable/diffable file. Files are repo
// paths this hit is ABOUT (references), enabling file-aware resurfacing.
type Item struct {
	ID              string   `json:"id"`
	Text            string   `json:"text"`
	Status          string   `json:"status"`   // open | done | dropped
	Priority        string   `json:"priority"` // now | next | later
	Files           []string `json:"files,omitempty"`
	BodyPath        string   `json:"bodyPath,omitempty"` // repo-rel path to the .md body, if any
	RaisedInSession string   `json:"raisedInSession,omitempty"`
	RaisedAt        string   `json:"raisedAt"`
	DoneAt          string   `json:"doneAt,omitempty"`
}

// bodyDir holds the spilled markdown bodies for hits with substantial content.
const bodyDir = ".mint/hits"

func path(root string) string { return filepath.Join(root, relPath) }

// writeBody spills a hit's big content to .mint/hits/<id>.md and returns the repo-relative
// bodyPath to store on the row. Empty body → "" (no file written, row carries no bodyPath).
func writeBody(root, id, body string) (string, error) {
	if strings.TrimSpace(body) == "" {
		return "", nil
	}
	rel := filepath.Join(bodyDir, id+".md")
	abs := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		return "", err
	}
	return rel, nil
}

// Body reads the spilled markdown body for an item, or "" if it has none.
func Body(root string, it Item) (string, error) {
	if it.BodyPath == "" {
		return "", nil
	}
	b, err := os.ReadFile(filepath.Join(root, it.BodyPath))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(b), nil
}

// validPriority normalizes/validates a priority; empty defaults to "next".
func validPriority(p string) (string, error) {
	switch p {
	case "":
		return "next", nil
	case "now", "next", "later":
		return p, nil
	default:
		return "", fmt.Errorf("invalid priority %q (use now|next|later)", p)
	}
}

// Read loads all hits in file order. A missing file is an empty list (not an error); a
// malformed line is skipped so one bad row never blocks the whole agenda.
func Read(root string) ([]Item, error) {
	f, err := os.Open(path(root))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var items []Item
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var it Item
		if json.Unmarshal([]byte(line), &it) == nil && it.ID != "" {
			items = append(items, it)
		}
	}
	return items, sc.Err()
}

// write persists the full list, rewriting the file (after add/done/drop the whole list is
// re-serialized — simpler and safe at agenda scale; the list is tens of items, not millions).
func write(root string, items []Item) error {
	p := path(root)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	for _, it := range items {
		row, err := json.Marshal(it)
		if err != nil {
			return err
		}
		b.Write(row)
		b.WriteByte('\n')
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0o644); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, p); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// nextID allocates a short sequential id (h1, h2, …) above the highest existing numeric id,
// so ids stay stable and human-typeable for `mint hit done h3`.
func nextID(items []Item) string {
	max := 0
	for _, it := range items {
		var n int
		if _, err := fmt.Sscanf(it.ID, "h%d", &n); err == nil && n > max {
			max = n
		}
	}
	return fmt.Sprintf("h%d", max+1)
}

// AddOpts carries the optional body + file references for a new hit. Text+priority are the
// required minimum; Body (big markdown) and Files (repo paths it's about) are optional.
type AddOpts struct {
	Body    string
	Files   []string
	Session string
}

// Add appends a new open hit and returns it. A non-empty opts.Body spills to .mint/hits/<id>.md
// and the row records its bodyPath. now is passed in (not time.Now()) so callers stay testable.
func Add(root, text, priority string, opts AddOpts, now time.Time) (Item, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Item{}, fmt.Errorf("hit text is empty")
	}
	prio, err := validPriority(priority)
	if err != nil {
		return Item{}, err
	}
	items, err := Read(root)
	if err != nil {
		return Item{}, err
	}
	id := nextID(items)
	bodyPath, err := writeBody(root, id, opts.Body)
	if err != nil {
		return Item{}, err
	}
	it := Item{
		ID:              id,
		Text:            text,
		Status:          "open",
		Priority:        prio,
		Files:           cleanFiles(opts.Files),
		BodyPath:        bodyPath,
		RaisedInSession: opts.Session,
		RaisedAt:        now.UTC().Format(time.RFC3339),
	}
	items = append(pruneAt(root, items, now), it)
	if err := write(root, items); err != nil {
		return Item{}, err
	}
	return it, nil
}

// cleanFiles trims + drops empty file references; nil when none remain.
func cleanFiles(files []string) []string {
	var out []string
	for _, f := range files {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}

// setStatus marks a hit done|dropped by id. Returns the updated item.
func setStatus(root, id, status string, now time.Time) (Item, error) {
	items, err := Read(root)
	if err != nil {
		return Item{}, err
	}
	for i := range items {
		if items[i].ID == id {
			items[i].Status = status
			items[i].DoneAt = now.UTC().Format(time.RFC3339)
			if err := write(root, pruneAt(root, items, now)); err != nil {
				return Item{}, err
			}
			return items[i], nil
		}
	}
	return Item{}, fmt.Errorf("no hit %q (see `mint hit list`)", id)
}

// Done marks a hit completed. Drop marks it abandoned. Both retain the row (with doneAt)
// until self-clean, so a glance back shows what was handled.
func Done(root, id string, now time.Time) (Item, error) { return setStatus(root, id, "done", now) }
func Drop(root, id string, now time.Time) (Item, error) { return setStatus(root, id, "dropped", now) }

// Open returns just the open hits, ordered now → next → later (the agenda the driver acts on).
func Open(root string) ([]Item, error) {
	items, err := Read(root)
	if err != nil {
		return nil, err
	}
	rank := map[string]int{"now": 0, "next": 1, "later": 2}
	var open []Item
	for _, it := range items {
		if it.Status == "open" {
			open = append(open, it)
		}
	}
	// Stable insertion sort by priority rank (small list; keeps creation order within a tier).
	for i := 1; i < len(open); i++ {
		for j := i; j > 0 && rank[open[j].Priority] < rank[open[j-1].Priority]; j-- {
			open[j], open[j-1] = open[j-1], open[j]
		}
	}
	return open, nil
}

// prune drops done/dropped hits older than the retention window. Open hits are never pruned.
// pruneRoot, when non-empty, lets prune delete the aged-out hit's spilled body file too so
// .mint/hits/ doesn't accumulate orphans.
func pruneAt(root string, items []Item, now time.Time) []Item {
	kept := items[:0:0]
	for _, it := range items {
		if it.Status == "open" || it.DoneAt == "" {
			kept = append(kept, it)
			continue
		}
		if t, err := time.Parse(time.RFC3339, it.DoneAt); err == nil && now.Sub(t) > doneRetention {
			if root != "" && it.BodyPath != "" {
				_ = os.Remove(filepath.Join(root, it.BodyPath)) // best-effort body cleanup
			}
			continue // aged out
		}
		kept = append(kept, it)
	}
	return kept
}

// prune is the body-agnostic variant (pure, no I/O) used where the root isn't threaded.
func prune(items []Item, now time.Time) []Item { return pruneAt("", items, now) }

// Summary is the one-line resurfacing string for session start / handoff / status, e.g.
// "📌 3 open hits: [now] fix Y · [next] refactor Z". Empty string when there are no open
// hits (callers print nothing rather than a noisy "0 hits").
func Summary(root string) string {
	open, err := Open(root)
	if err != nil || len(open) == 0 {
		return ""
	}
	parts := make([]string, 0, len(open))
	for _, it := range open {
		parts = append(parts, fmt.Sprintf("[%s] %s", it.Priority, it.Text))
	}
	return fmt.Sprintf("📌 %d open hit(s): %s", len(open), strings.Join(parts, " · "))
}

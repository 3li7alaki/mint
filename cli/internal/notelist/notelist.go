// Package notelist stores append-mostly, topic-keyed reasoning that supports
// unit and floor decisions. Each topic accumulates timestamped entries in the
// current worktree's private global state.
package notelist

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"mint/internal/statehome"
)

const relPath = "notes/index.jsonl"
const bodyDir = "notes"

// topicRe constrains a topic to a safe, file-name-able slug (the topic IS the body filename),
// so a topic can never escape the notes directory via path traversal.
var topicRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

// Note is one topic's index row. The accumulating reasoning lives in BodyPath's markdown
// file; the row stays small + greppable.
type Note struct {
	Topic     string   `json:"topic"`
	Files     []string `json:"files,omitempty"`
	BodyPath  string   `json:"bodyPath"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
	Entries   int      `json:"entries"`
}

func path(root string) string { return filepath.Join(statehome.Resolve(root).Dir, relPath) }
func bodyAbs(root, topic string) string {
	return filepath.Join(statehome.Resolve(root).Dir, bodyDir, topic+".md")
}

// normalizeTopic lowercases + validates a topic. A topic with spaces is dashed; anything
// outside the safe slug charset is rejected (it becomes a filename).
func normalizeTopic(topic string) (string, error) {
	t := strings.ToLower(strings.TrimSpace(topic))
	t = strings.ReplaceAll(t, " ", "-")
	if !topicRe.MatchString(t) {
		return "", fmt.Errorf("invalid topic %q (use letters, digits, . _ -)", topic)
	}
	return t, nil
}

// Read loads all note rows in file order. Missing file → empty; malformed lines skipped.
func Read(root string) ([]Note, error) {
	f, err := os.Open(path(root))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var notes []Note
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var n Note
		if json.Unmarshal([]byte(line), &n) == nil && n.Topic != "" {
			notes = append(notes, n)
		}
	}
	return notes, sc.Err()
}

func write(root string, notes []Note) error {
	p := path(root)
	if _, err := statehome.Ensure(root); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	var b strings.Builder
	for _, n := range notes {
		row, err := json.Marshal(n)
		if err != nil {
			return err
		}
		b.Write(row)
		b.WriteByte('\n')
	}
	return statehome.Write(p, []byte(b.String()))
}

// Append adds an entry to a topic's reasoning, creating the topic on first use. The entry is
// appended as a timestamped section to notes/<topic>.md (append-mostly — earlier
// reasoning is never overwritten), and the index row's UpdatedAt/Entries advance. Empty text
// AND empty files is rejected (nothing to record). now is injected for testability.
func Append(root, topic, text string, files []string, now time.Time) (Note, error) {
	t, err := normalizeTopic(topic)
	if err != nil {
		return Note{}, err
	}
	text = strings.TrimSpace(text)
	files = cleanFiles(files)
	if text == "" && len(files) == 0 {
		return Note{}, fmt.Errorf("nothing to record (pass text and/or --file)")
	}

	notes, err := Read(root)
	if err != nil {
		return Note{}, err
	}
	stamp := now.UTC().Format(time.RFC3339)

	// Append the timestamped entry to the topic's markdown body.
	if text != "" {
		if _, err := statehome.Ensure(root); err != nil {
			return Note{}, err
		}
		if err := os.MkdirAll(filepath.Join(statehome.Resolve(root).Dir, bodyDir), 0o700); err != nil {
			return Note{}, err
		}
		section := fmt.Sprintf("## %s\n\n%s\n\n", stamp, text)
		f, err := os.OpenFile(bodyAbs(root, t), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return Note{}, err
		}
		if _, err := f.WriteString(section); err != nil {
			f.Close()
			return Note{}, err
		}
		f.Close()
	}

	// Upsert the index row.
	relBody := filepath.Join(bodyDir, t+".md")
	idx := -1
	for i := range notes {
		if notes[i].Topic == t {
			idx = i
			break
		}
	}
	if idx == -1 {
		notes = append(notes, Note{
			Topic: t, Files: files, BodyPath: relBody,
			CreatedAt: stamp, UpdatedAt: stamp, Entries: boolToInt(text != ""),
		})
		idx = len(notes) - 1
	} else {
		notes[idx].UpdatedAt = stamp
		notes[idx].BodyPath = relBody
		notes[idx].Files = mergeFiles(notes[idx].Files, files)
		if text != "" {
			notes[idx].Entries++
		}
	}
	if err := write(root, notes); err != nil {
		return Note{}, err
	}
	return notes[idx], nil
}

// Body reads a topic's accumulated reasoning, or "" if the topic/file is absent.
func Body(root, topic string) (string, error) {
	t, err := normalizeTopic(topic)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(bodyAbs(root, t))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(b), nil
}

// Summary is a one-line topic summary for status consumers, e.g.
// "🗒 2 note(s): loader-bug, scope-model". Empty when there are no notes.
func Summary(root string) string {
	notes, err := Read(root)
	if err != nil || len(notes) == 0 {
		return ""
	}
	topics := make([]string, 0, len(notes))
	for _, n := range notes {
		topics = append(topics, n.Topic)
	}
	return fmt.Sprintf("🗒 %d note(s): %s", len(notes), strings.Join(topics, ", "))
}

func cleanFiles(files []string) []string {
	var out []string
	for _, f := range files {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}

// mergeFiles unions two file-reference lists, preserving order and dropping duplicates.
func mergeFiles(existing, add []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, f := range append(append([]string{}, existing...), add...) {
		if f = strings.TrimSpace(f); f != "" && !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

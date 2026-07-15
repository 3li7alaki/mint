// Package unitstore defines mint's durable unit and attempt filesystem layout.
// It stores no project, ticket, terminal, or worktree lifecycle state.
package unitstore

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"mint/internal/atomic"
	"mint/internal/statehome"
)

const unitsDir = "units"

func UnitsDir(root string) string { return filepath.Join(statehome.Resolve(root).Dir, unitsDir) }

func Ensure(root string) error {
	loc, err := statehome.Ensure(root)
	if err != nil {
		return err
	}
	for _, name := range []string{"units", "attempts", "receipts", "notes"} {
		dir := filepath.Join(loc.Dir, name)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			return err
		}
	}
	return nil
}

func UnitDir(root, slug, specID string) string {
	if !valid(slug, specID) {
		return ""
	}
	return filepath.Join(UnitsDir(root), slug, specID)
}

func SpecsDir(root, slug string) string {
	if !atomic.IsLiteralSegment(slug) {
		return ""
	}
	return filepath.Join(UnitsDir(root), slug)
}

func SpecPath(root, slug, specID string) string {
	dir := UnitDir(root, slug, specID)
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "spec.xml")
}

func AttemptPath(root, slug, specID, attemptID string) string {
	if !valid(slug, specID) || !atomic.IsLiteralSegment(attemptID) {
		return ""
	}
	return filepath.Join(statehome.Resolve(root).Dir, "attempts", slug, specID, attemptID+".json")
}

func AttemptLockPath(root, slug, specID, attemptID string) string {
	path := AttemptPath(root, slug, specID, attemptID)
	if path == "" {
		return ""
	}
	return strings.TrimSuffix(path, ".json") + ".lock"
}

func VerdictPath(root, slug, specID, attemptID string) string {
	path := AttemptPath(root, slug, specID, attemptID)
	if path == "" {
		return ""
	}
	return strings.TrimSuffix(path, ".json") + ".verdict.json"
}

func Attempts(root, slug, specID string) []string {
	dir := UnitDir(root, slug, specID)
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(filepath.Join(statehome.Resolve(root).Dir, "attempts", slug, specID))
	if err != nil {
		return nil
	}
	var ids []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		if atomic.IsLiteralSegment(id) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func GenerateAttemptID(now time.Time) (string, error) {
	random := make([]byte, 6)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return fmt.Sprintf("a-%x-%s", now.UTC().UnixMilli(), hex.EncodeToString(random)), nil
}

func ResolveSpec(root, slug, specID string) (string, bool) {
	path := SpecPath(root, slug, specID)
	if path == "" {
		return "", false
	}
	info, err := os.Stat(path)
	return path, err == nil && info.Mode().IsRegular()
}

func List(root string) []UnitRef {
	var units []UnitRef
	slugs, err := os.ReadDir(UnitsDir(root))
	if err != nil {
		return nil
	}
	for _, slug := range slugs {
		if !slug.IsDir() || !atomic.IsLiteralSegment(slug.Name()) {
			continue
		}
		ids, _ := os.ReadDir(filepath.Join(UnitsDir(root), slug.Name()))
		for _, id := range ids {
			if id.IsDir() && atomic.IsLiteralSegment(id.Name()) {
				if _, ok := ResolveSpec(root, slug.Name(), id.Name()); ok {
					units = append(units, UnitRef{Slug: slug.Name(), SpecID: id.Name()})
				}
			}
		}
	}
	sort.Slice(units, func(i, j int) bool {
		if units[i].Slug == units[j].Slug {
			return units[i].SpecID < units[j].SpecID
		}
		return units[i].Slug < units[j].Slug
	})
	return units
}

type UnitRef struct {
	Slug   string `json:"slug"`
	SpecID string `json:"specId"`
}

func ValidSegment(value string) bool { return atomic.IsLiteralSegment(value) }

func valid(slug, specID string) bool {
	return atomic.IsLiteralSegment(slug) && atomic.IsLiteralSegment(specID)
}

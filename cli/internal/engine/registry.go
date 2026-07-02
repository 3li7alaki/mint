package engine

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

type Row struct {
	Key          string
	Binary       string
	Args         []string
	Permission   map[string][]string
	Reference    bool
	ProbeOrder   int
	StopFromExit bool
}

type manifest struct {
	Trusted []Row `json:"trusted"`
}

//go:embed engines.json
var manifestBytes []byte

var Registry = mustLoadRegistry(manifestBytes)

func mustLoadRegistry(data []byte) map[string]Row {
	registry, err := loadRegistry(data)
	if err != nil {
		panic(err)
	}
	return registry
}

func loadRegistry(data []byte) (map[string]Row, error) {
	var doc manifest
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse engine manifest: %w", err)
	}
	registry := make(map[string]Row, len(doc.Trusted))
	for _, row := range doc.Trusted {
		key := Canonical(row.Key)
		if key == "" {
			return nil, fmt.Errorf("engine manifest has empty key")
		}
		if key != row.Key {
			return nil, fmt.Errorf("engine key %q is not canonical", row.Key)
		}
		if row.Binary == "" {
			return nil, fmt.Errorf("engine %q has empty binary", row.Key)
		}
		if _, exists := registry[key]; exists {
			return nil, fmt.Errorf("engine %q is duplicated", key)
		}
		row.Key = key
		registry[key] = row
	}
	if len(registry) == 0 {
		return nil, fmt.Errorf("engine manifest has no trusted engines")
	}
	return registry, nil
}

func Keys() []string {
	return TrustedKeys()
}

func TrustedKeys() []string {
	return sortedKeys(Registry)
}

func ReferenceSlots() []string {
	rows := make([]Row, 0, len(Registry))
	for _, row := range Registry {
		if row.Reference {
			rows = append(rows, row)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].ProbeOrder == rows[j].ProbeOrder {
			return rows[i].Key < rows[j].Key
		}
		return rows[i].ProbeOrder < rows[j].ProbeOrder
	})
	slots := make([]string, 0, len(rows))
	for _, row := range rows {
		slots = append(slots, row.Key)
	}
	return slots
}

func Available(probe func(Row) bool) []Row {
	if probe == nil {
		return nil
	}
	slots := ReferenceSlots()
	available := make([]Row, 0, len(slots))
	for _, slot := range slots {
		row := Registry[slot]
		if probe(row) {
			available = append(available, row)
		}
	}
	return available
}

func sortedKeys(registry map[string]Row) []string {
	keys := make([]string, 0, len(registry))
	for key := range registry {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func IsKnown(name string) bool {
	_, ok := Registry[Canonical(name)]
	return ok
}

func Canonical(name string) string {
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'Ａ' && r <= 'Ｚ':
			return r - 'Ａ' + 'A'
		case r >= 'ａ' && r <= 'ｚ':
			return r - 'ａ' + 'a'
		case r >= '０' && r <= '９':
			return r - '０' + '0'
		case isUnicodeSpace(r):
			return ' '
		case isIdentityIgnored(r):
			return -1
		default:
			return r
		}
	}, name)
	name = strings.ToLower(name)
	return strings.Join(strings.Fields(name), " ")
}

func isUnicodeSpace(r rune) bool {
	return r == '\u00a0' ||
		r >= '\u2000' && r <= '\u200a' ||
		r == '\u202f' ||
		r == '\u205f' ||
		r == '\u3000' ||
		unicode.IsSpace(r)
}

func isIdentityIgnored(r rune) bool {
	return r <= '\u001f' ||
		r == '\u007f' ||
		r == '\u180e' ||
		r >= '\u200b' && r <= '\u200f' ||
		r >= '\u202a' && r <= '\u202e' ||
		r == '\u2060' ||
		r >= '\u2066' && r <= '\u2069' ||
		r >= '\ufe00' && r <= '\ufe0f' ||
		r == '\ufeff' ||
		r >= 0xe0000 && r <= 0xe01ef ||
		unicode.Is(unicode.Mn, r)
}

func IsStrictSession(session string) bool {
	if session == "" {
		return false
	}
	for _, r := range session {
		if r > unicode.MaxASCII {
			return false
		}
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) ||
			r == '_' || r == '-' || r == '.' || r == '@' || r == '{' ||
			r == '}' || r == '+' || r == '~' || r == '^' || r == '/') {
			return false
		}
	}
	return true
}

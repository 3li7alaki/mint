// Package receipt stores immutable, versioned completion records and checks
// whether they still describe the current source contents.
package receipt

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"mint/internal/atomic"
	"mint/internal/execstate"
	"mint/internal/floor"
	"mint/internal/snapshot"
	"mint/internal/statehome"
)

const SchemaVersion = 1

type UnitRef struct {
	Slug   string `json:"slug"`
	SpecID string `json:"specId"`
}

type Evidence struct {
	Maker           *execstate.Maker            `json:"maker,omitempty"`
	Checker         execstate.Provenance        `json:"checker"`
	Acceptance      map[string]any              `json:"acceptance"`
	RequiredReviews []string                    `json:"requiredReviews,omitempty"`
	Reviews         map[string]execstate.Review `json:"reviews,omitempty"`
}

type Record struct {
	SchemaVersion int                  `json:"schemaVersion"`
	ID            string               `json:"id"`
	Unit          UnitRef              `json:"unit"`
	AttemptID     string               `json:"attemptId"`
	Terminal      string               `json:"terminal"`
	Accepted      bool                 `json:"accepted"`
	Snapshot      snapshot.Source      `json:"snapshot"`
	Clauses       []floor.ClauseResult `json:"clauses"`
	Evidence      Evidence             `json:"evidence"`
	IssuedAt      string               `json:"issuedAt"`
}

type Validation struct {
	SchemaVersion int    `json:"schemaVersion"`
	Valid         bool   `json:"valid"`
	Current       bool   `json:"current"`
	ReceiptDigest string `json:"receiptDigest"`
	CurrentDigest string `json:"currentDigest,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

type NewOptions struct {
	Slug      string
	SpecID    string
	AttemptID string
	Terminal  string
	Snapshot  snapshot.Source
	Result    floor.Result
	Input     floor.Input
	IssuedAt  time.Time
}

func New(opts NewOptions) (Record, error) {
	if !atomic.IsLiteralSegment(opts.Slug) || !atomic.IsLiteralSegment(opts.SpecID) || !atomic.IsLiteralSegment(opts.AttemptID) {
		return Record{}, fmt.Errorf("invalid receipt unit %q/%q", opts.Slug, opts.SpecID)
	}
	if !opts.Result.Pass {
		return Record{}, fmt.Errorf("cannot issue a completion receipt for a failing floor")
	}
	id, err := generateID(opts.IssuedAt)
	if err != nil {
		return Record{}, err
	}
	checker, err := provenanceFromVerdict(opts.Input.Verdict)
	if err != nil {
		return Record{}, err
	}
	var maker *execstate.Maker
	if opts.Input.MakerExecutor != "" || opts.Input.MakerExecutionRef != "" {
		maker = &execstate.Maker{
			Executor: opts.Input.MakerExecutor, Vendor: opts.Input.MakerVendor,
			Model: opts.Input.MakerModel, Locality: opts.Input.MakerLocality,
			ExecutionRef: opts.Input.MakerExecutionRef,
		}
	}
	return Record{
		SchemaVersion: SchemaVersion,
		ID:            id,
		Unit:          UnitRef{Slug: opts.Slug, SpecID: opts.SpecID},
		AttemptID:     opts.AttemptID,
		Terminal:      opts.Terminal,
		Accepted:      true,
		Snapshot:      opts.Snapshot,
		Clauses:       append([]floor.ClauseResult(nil), opts.Result.Clauses...),
		Evidence: Evidence{
			Maker: maker, Checker: checker, Acceptance: opts.Input.Verdict,
			RequiredReviews: append([]string(nil), opts.Input.RequiredReviews...),
			Reviews:         opts.Input.Reviews,
		},
		IssuedAt: opts.IssuedAt.UTC().Format(time.RFC3339Nano),
	}, nil
}

// Store writes a receipt exactly once. The random receipt ID prevents a retry
// from overwriting historical evidence; os.O_EXCL enforces immutability.
func Store(root string, record Record) (string, error) {
	if !atomic.IsLiteralSegment(record.ID) {
		return "", fmt.Errorf("invalid receipt id %q", record.ID)
	}
	if !atomic.IsLiteralSegment(record.AttemptID) {
		return "", fmt.Errorf("invalid receipt attempt id %q", record.AttemptID)
	}
	dir := filepath.Join(statehome.Resolve(root).Dir, "receipts", record.Unit.Slug, record.Unit.SpecID)
	if _, err := statehome.Ensure(root); err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	marker := filepath.Join(dir, record.AttemptID+".completed")
	claim, err := os.OpenFile(marker, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return "", fmt.Errorf("attempt %s already has a completion receipt", record.AttemptID)
		}
		return "", err
	}
	claimed := false
	defer func() {
		_ = claim.Close()
		if !claimed {
			_ = os.Remove(marker)
		}
	}()
	if _, err := claim.WriteString(record.ID + "\n"); err != nil {
		return "", err
	}
	if err := claim.Sync(); err != nil {
		return "", err
	}
	if err := claim.Close(); err != nil {
		return "", err
	}
	path := filepath.Join(dir, record.ID+".json")
	b, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return "", err
	}
	b = append(b, '\n')
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", err
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	claimed = true
	return path, nil
}

func Read(path string) (Record, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Record{}, err
	}
	var record Record
	if err := json.Unmarshal(b, &record); err != nil {
		return Record{}, err
	}
	if record.SchemaVersion != SchemaVersion || !atomic.IsLiteralSegment(record.ID) || !atomic.IsLiteralSegment(record.AttemptID) || record.Snapshot.Digest == "" {
		return Record{}, fmt.Errorf("unsupported or malformed receipt")
	}
	return record, nil
}

func List(root, slug, specID string) []string {
	dir := filepath.Join(statehome.Resolve(root).Dir, "receipts", slug, specID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			paths = append(paths, filepath.Join(dir, entry.Name()))
		}
	}
	sort.Strings(paths)
	return paths
}

// OrphanClaims returns completion markers whose referenced receipt was never
// durably created. This can only happen when receipt creation is interrupted
// between claiming an attempt and writing its immutable JSON record.
func OrphanClaims(root string) []string {
	dir := filepath.Join(statehome.Resolve(root).Dir, "receipts")
	var paths []string
	_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".completed") {
			return nil
		}
		contents, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		id := strings.TrimSpace(string(contents))
		if !atomic.IsLiteralSegment(id) {
			paths = append(paths, path)
			return nil
		}
		if _, statErr := os.Stat(filepath.Join(filepath.Dir(path), id+".json")); os.IsNotExist(statErr) {
			paths = append(paths, path)
		}
		return nil
	})
	sort.Strings(paths)
	return paths
}

func Validate(root string, record Record) Validation {
	result := Validation{SchemaVersion: 1, Valid: true, ReceiptDigest: record.Snapshot.Digest}
	current, err := snapshot.Capture(root, record.Snapshot.Base)
	if err != nil {
		result.Valid = false
		result.Reason = err.Error()
		return result
	}
	result.CurrentDigest = current.Digest
	result.Current = current.Digest == record.Snapshot.Digest
	if !result.Current {
		result.Reason = "source snapshot changed after receipt issuance"
	}
	return result
}

func generateID(now time.Time) (string, error) {
	random := make([]byte, 6)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	stamp := now.UTC().Format("20060102T150405.000000000Z")
	stamp = strings.ReplaceAll(stamp, ".", "-")
	return stamp + "-" + hex.EncodeToString(random), nil
}

func provenanceFromVerdict(verdict map[string]any) (execstate.Provenance, error) {
	field := func(key string) string {
		value, _ := verdict[key].(string)
		return value
	}
	return execstate.ValidateProvenance(execstate.Provenance{
		Executor: field("executor"), Vendor: field("vendor"), Model: field("model"),
		Locality: field("locality"), ExecutionRef: field("executionRef"),
		ObservedBy: field("observedBy"), Attestation: field("attestation"),
	})
}

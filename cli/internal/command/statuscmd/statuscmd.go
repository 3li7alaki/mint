package statuscmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"mint/internal/execstate"
	"mint/internal/receipt"
	"mint/internal/specschema"
	"mint/internal/statehome"
	"mint/internal/unitstore"
	"mint/internal/version"
)

type Flags struct{ JSON bool }

type Report struct {
	SchemaVersion int          `json:"schemaVersion"`
	Version       string       `json:"version"`
	RepositoryID  string       `json:"repositoryId"`
	WorktreeID    string       `json:"worktreeId"`
	WorktreeRoot  string       `json:"worktreeRoot"`
	StateDir      string       `json:"stateDir"`
	Units         []UnitStatus `json:"units"`
}

type UnitStatus struct {
	Slug     string          `json:"slug"`
	SpecID   string          `json:"specId"`
	Attempts []AttemptStatus `json:"attempts"`
	Receipts []ReceiptStatus `json:"receipts"`
}

type AttemptStatus struct {
	AttemptID string   `json:"attemptId"`
	Status    string   `json:"status"`
	Missing   []string `json:"missingEvidence,omitempty"`
}

type ReceiptStatus struct {
	ID      string `json:"id"`
	Path    string `json:"path"`
	Current bool   `json:"current"`
	Valid   bool   `json:"valid"`
}

func Run(root string, flags Flags, stdout io.Writer) (int, error) {
	report := Build(root)
	if flags.JSON {
		b, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return 1, err
		}
		_, err = fmt.Fprintln(stdout, string(b))
		return 0, err
	}
	fmt.Fprintf(stdout, "mint %s\nworktree %s\nstate %s\n", report.Version, report.WorktreeRoot, report.StateDir)
	if len(report.Units) == 0 {
		fmt.Fprintln(stdout, "units: none")
		return 0, nil
	}
	for _, unit := range report.Units {
		fmt.Fprintf(stdout, "%s/%s\n", unit.Slug, unit.SpecID)
		for _, attempt := range unit.Attempts {
			fmt.Fprintf(stdout, "  attempt %s: %s", attempt.AttemptID, attempt.Status)
			if len(attempt.Missing) > 0 {
				fmt.Fprintf(stdout, " (missing: %v)", attempt.Missing)
			}
			fmt.Fprintln(stdout)
		}
		for _, item := range unit.Receipts {
			freshness := "stale"
			if item.Valid && item.Current {
				freshness = "current"
			}
			fmt.Fprintf(stdout, "  receipt %s: %s (%s)\n", item.ID, freshness, item.Path)
		}
	}
	return 0, nil
}

func Build(root string) Report {
	loc := statehome.Resolve(root)
	report := Report{SchemaVersion: 1, Version: version.Version, RepositoryID: loc.RepositoryID, WorktreeID: loc.WorktreeID, WorktreeRoot: loc.WorktreeRoot, StateDir: loc.Dir}
	report.Units = []UnitStatus{}
	for _, ref := range unitstore.List(root) {
		unit := UnitStatus{Slug: ref.Slug, SpecID: ref.SpecID, Attempts: []AttemptStatus{}, Receipts: []ReceiptStatus{}}
		specPath, _ := unitstore.ResolveSpec(root, ref.Slug, ref.SpecID)
		specBytes, _ := os.ReadFile(specPath)
		gates := specschema.ResolveSpecGates(string(specBytes))
		reviews := specschema.ResolveSpecReviews(string(specBytes))
		for _, attemptID := range unitstore.Attempts(root, ref.Slug, ref.SpecID) {
			state, ok := execstate.Read(root, ref.Slug, ref.SpecID, attemptID)
			if !ok {
				continue
			}
			missing := missingEvidence(state, gates, reviews)
			if _, err := os.Stat(unitstore.VerdictPath(root, ref.Slug, ref.SpecID, attemptID)); err != nil {
				missing = append(missing, "acceptance-verdict")
				sort.Strings(missing)
			}
			unit.Attempts = append(unit.Attempts, AttemptStatus{AttemptID: attemptID, Status: state.Status, Missing: missing})
		}
		for _, path := range receipt.List(root, ref.Slug, ref.SpecID) {
			record, err := receipt.Read(path)
			if err != nil {
				unit.Receipts = append(unit.Receipts, ReceiptStatus{ID: filepath.Base(path), Path: path})
				continue
			}
			validation := receipt.Validate(root, record)
			unit.Receipts = append(unit.Receipts, ReceiptStatus{ID: record.ID, Path: path, Current: validation.Current, Valid: validation.Valid})
		}
		report.Units = append(report.Units, unit)
	}
	return report
}

func missingEvidence(state *execstate.State, gates map[string]string, reviews []string) []string {
	var missing []string
	if state.Maker == nil {
		missing = append(missing, "maker")
	}
	if state.Gates["tier"] != "skip" {
		for label := range gates {
			if state.Gates[label] != "pass" {
				missing = append(missing, "gate:"+label)
			}
		}
		for _, lens := range reviews {
			if state.Reviews[lens].Verdict != "passed" {
				missing = append(missing, "review:"+lens)
			}
		}
	}
	sort.Strings(missing)
	return missing
}

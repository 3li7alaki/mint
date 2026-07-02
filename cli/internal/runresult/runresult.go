package runresult

import (
	"path/filepath"

	"mint/internal/atomic"
	"mint/internal/session"
)

const (
	Done        = 0
	HardFailure = 1
	Escalation  = 2
)

type Gates struct {
	Tier    string            `json:"tier,omitempty"`
	Results map[string]string `json:"results,omitempty"`
	OK      bool              `json:"ok"`
}

type Report struct {
	Status  string
	Gates   *Gates
	ScopeOK *bool
}

type IDs struct {
	SpecID    string
	SessionID string
}

type Result struct {
	SpecID       *string `json:"specId"`
	SessionID    string  `json:"sessionId"`
	Status       string  `json:"status"`
	ExitCode     int     `json:"exitCode"`
	Gates        *Gates  `json:"gates"`
	OpenBlockers int     `json:"openBlockers"`
	Resumable    bool    `json:"resumable"`
}

func ClassifyExit(report Report) int {
	if report.Status == "escalation" {
		return Escalation
	}
	if report.Status == "passed" && report.Gates != nil && report.Gates.OK && report.ScopeOK != nil && *report.ScopeOK {
		return Done
	}
	return HardFailure
}

func Build(report Report, ids IDs) Result {
	exitCode := ClassifyExit(report)
	openBlockers := 0
	if report.Status == "escalation" {
		openBlockers = 1
	} else {
		if report.Gates != nil {
			for _, result := range report.Gates.Results {
				if result == "fail" {
					openBlockers++
				}
			}
		}
		if report.ScopeOK != nil && !*report.ScopeOK {
			openBlockers++
		}
	}
	return Result{
		SpecID:       nullableString(ids.SpecID),
		SessionID:    ids.SessionID,
		Status:       report.Status,
		ExitCode:     exitCode,
		Gates:        report.Gates,
		OpenBlockers: openBlockers,
		Resumable:    exitCode != Done,
	}
}

func Write(root string, report Report, ids IDs) (Result, error) {
	result := Build(report, ids)
	path := filepath.Join(session.GetSessionsDir(root), ids.SessionID, "result.json")
	if err := atomic.WriteJSON(path, result); err != nil {
		return Result{}, err
	}
	return result, nil
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func Bool(value bool) *bool {
	return &value
}

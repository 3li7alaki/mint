package verify

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"mint/internal/docclassify"
	"mint/internal/execstate"
	"mint/internal/replay"
	"mint/internal/snapshot"
	"mint/internal/specschema"
	"mint/internal/unitstore"
)

type ChangedFile struct {
	Path   string
	Status string
}

type Result struct {
	SchemaVersion int               `json:"schemaVersion"`
	Tier          string            `json:"tier"`
	Results       map[string]string `json:"results"`
	Reasons       map[string]string `json:"reasons,omitempty"`
	OK            bool              `json:"ok"`
	Declared      bool              `json:"declared"`
	Error         string            `json:"error,omitempty"`
	Mode          string            `json:"mode,omitempty"`
	Reverted      []string          `json:"reverted,omitempty"`
	Logs          map[string]string `json:"logs,omitempty"`
	PrepareLog    string            `json:"prepareLog,omitempty"`
	PrepareFailed bool              `json:"prepareFailed,omitempty"`
	Hint          string            `json:"hint,omitempty"`
}

func Run(root, slug, specID, attemptID, specPath string) (result Result) {
	result = Result{SchemaVersion: 1, Results: map[string]string{}, Mode: "in-place"}
	state, exists := execstate.Read(root, slug, specID, attemptID)
	if !exists {
		result.Error = "attempt does not exist; run mint exec init"
		return result
	}

	var paths []string
	if state.BaselineTree != nil {
		result.Mode = "replay"
		var err error
		paths, _, _, err = snapshot.ChangedSince(root, *state.BaselineTree)
		if err != nil {
			result.Error = err.Error()
			return result
		}
	} else {
		for _, file := range ChangedFiles(root) {
			paths = append(paths, file.Path)
		}
	}
	if docclassify.IsDocsOnly(paths) {
		if _, err := execstate.RecordGate(root, slug, specID, "tier", "skip", attemptID); err != nil {
			result.Error = err.Error()
			return result
		}
		result.Tier, result.OK, result.Declared = "skip", true, true
		return result
	}

	specBytes, _ := os.ReadFile(specPath)
	spec := string(specBytes)
	gates := specschema.ResolveSpecGates(spec)
	labels := make([]string, 0, len(gates))
	for label := range gates {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	if len(labels) == 0 {
		if _, err := execstate.RecordGate(root, slug, specID, "tier", "none", attemptID); err != nil {
			result.Error = err.Error()
			return result
		}
		result.Tier = "none"
		return result
	}

	if _, err := execstate.RecordGate(root, slug, specID, "tier", "full", attemptID); err != nil {
		result.Error = err.Error()
		return result
	}
	result.Tier = "full"
	logDir := unitstore.GateLogDir(root, slug, specID, attemptID)
	if err := os.RemoveAll(logDir); err != nil {
		result.Error = err.Error()
		return result
	}
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		result.Error = err.Error()
		return result
	}
	result.Logs = map[string]string{}
	result.Reasons = map[string]string{}
	gateRoot := root
	prepare := specschema.ResolvePrepare(spec)
	var prepareErr error
	if state.BaselineTree != nil {
		worktree, err := replay.Prepare(root, *state.BaselineTree, specschema.ResolveProtected(spec))
		if err != nil {
			result.Error = err.Error()
			return result
		}
		defer func() {
			if err := worktree.Close(); err != nil {
				result.OK = false
				result.Error = err.Error()
			}
		}()
		gateRoot, result.Reverted = worktree.Dir, worktree.Reverted
		if prepare != "" {
			path := filepath.Join(logDir, "prepare.log")
			log, err := openGateLog(path)
			if err != nil {
				result.Error = err.Error()
				return result
			}
			result.PrepareLog = path
			prepareErr = runGate(gateRoot, prepare, log, "MINT_SOURCE_ROOT="+root)
			result.PrepareFailed = prepareErr != nil
			if err := log.Close(); err != nil {
				result.Error = err.Error()
				return result
			}
		}
	}
	result.OK = true
	for i, label := range labels {
		command := gates[label]
		if strings.TrimSpace(command) == "" {
			continue
		}
		var gateErr error
		if prepareErr == nil {
			name := []byte(label)
			for j, b := range name {
				if !(b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || b == '.' || b == '_' || b == '-') {
					name[j] = '-'
				}
			}
			path := filepath.Join(logDir, fmt.Sprintf("%02d-%s.log", i+1, name))
			log, err := openGateLog(path)
			if err != nil {
				result.OK = false
				result.Error = err.Error()
				return result
			}
			result.Logs[label] = path
			gateErr = runGate(gateRoot, command, log)
			if err := log.Close(); err != nil {
				result.OK = false
				result.Error = err.Error()
				return result
			}
		}
		gateResult, reason := classifyGate(gateErr)
		if prepareErr != nil {
			_, prepareReason := classifyGate(prepareErr)
			gateResult = "inconclusive"
			reason = "not run: prepare failed (" + prepareReason + ")"
		}
		if gateResult != "pass" {
			result.OK = false
			result.Reasons[label] = reason
		}
		result.Results[label] = gateResult
		if _, err := execstate.RecordGate(root, slug, specID, label, gateResult, attemptID); err != nil {
			result.OK = false
			result.Error = err.Error()
			return result
		}
	}
	if result.Mode == "replay" && prepare == "" && !result.OK {
		result.Hint = "gates ran in a replay worktree that holds only tracked files, so gitignored dependencies (node_modules, vendor, .env, generated code) are absent; declare them with <prepare>, e.g. mint spec set <spec> --prepare 'pnpm install --frozen-lockfile'"
	}
	result.Declared = true
	return result
}

func ChangedFiles(root string) []ChangedFile {
	seen := map[string]bool{}
	files := []ChangedFile{}
	for _, line := range strings.Split(gitOutput(root, "diff", "--name-only", "HEAD"), "\n") {
		p := strings.TrimSpace(line)
		if p != "" && !seen[p] {
			seen[p] = true
			files = append(files, ChangedFile{Path: p, Status: "modified"})
		}
	}
	for _, line := range strings.Split(gitOutput(root, "ls-files", "--others", "--exclude-standard"), "\n") {
		p := strings.TrimSpace(line)
		if p != "" && !seen[p] {
			seen[p] = true
			files = append(files, ChangedFile{Path: p, Status: "new"})
		}
	}
	return files
}

func ResolveGates(specPath string) map[string]string {
	if specPath != "" {
		if b, err := os.ReadFile(specPath); err == nil {
			if gates := specschema.ResolveSpecGates(string(b)); len(gates) > 0 {
				return gates
			}
		}
	}
	return map[string]string{}
}

func classifyGate(err error) (result, reason string) {
	if err == nil {
		return "pass", ""
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return "inconclusive", "could not start: " + err.Error()
	}
	switch code := exitErr.ExitCode(); code {
	case -1:
		return "inconclusive", "killed by a signal"
	case 126:
		return "inconclusive", "exit 126: command not executable"
	case 127:
		return "inconclusive", "exit 127: command not found"
	default:
		// ponytail: Exit codes above 128, including a child killed under sh (137), stay fail because tests can crash too; upgrade to a mint-owned timeout that knows when it killed the gate.
		return "fail", fmt.Sprintf("exit %d", code)
	}
}

func openGateLog(path string) (*os.File, error) {
	// Gate output can contain secrets from copied .env files, so logs stay private.
	// ponytail: No size cap; chatty gates write MBs per run. Keep only the tail if storage becomes costly.
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
}

func runGate(root, command string, log io.Writer, env ...string) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Dir = root
	cmd.Stdout = log
	cmd.Stderr = log
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	return cmd.Run()
}

func gitOutput(root string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

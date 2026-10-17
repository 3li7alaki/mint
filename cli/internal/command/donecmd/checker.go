package donecmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"mint/internal/execstate"
	"mint/internal/floor"
)

// A full review of a large diff by a high-effort model routinely takes several minutes.
const DefaultCheckerTimeout = 15 * time.Minute

const checkerOutputLimit = 1 << 20 // Bound checker output to 1 MiB.

func runChecker(root, command string, provenance execstate.Provenance, timeout time.Duration, input floor.Input) map[string]any {
	verdict := map[string]any{
		"schemaVersion": 1, "accepted": false, "source": "checker",
		"executor": provenance.Executor, "vendor": provenance.Vendor,
		"model": provenance.Model, "locality": provenance.Locality,
		"executionRef": provenance.ExecutionRef,
	}
	fail := func(reason string) map[string]any {
		verdict["checkerReason"] = reason
		return verdict
	}
	payload := struct {
		SchemaVersion int      `json:"schemaVersion"`
		Spec          string   `json:"spec"`
		ChangedFiles  []string `json:"changedFiles"`
		Patch         string   `json:"patch"`
		Gates         struct {
			OK       bool     `json:"ok"`
			Mode     string   `json:"mode"`
			Reverted []string `json:"reverted"`
		} `json:"gates"`
	}{SchemaVersion: 1, Spec: input.Spec, ChangedFiles: input.ChangedFiles, Patch: input.Patch}
	if payload.ChangedFiles == nil {
		payload.ChangedFiles = []string{}
	}
	payload.Gates.Reverted = []string{}
	if input.Gates != nil {
		payload.Gates.OK = input.Gates.OK
		payload.Gates.Mode = input.Gates.GateMode
		payload.Gates.Reverted = append(payload.Gates.Reverted, input.Gates.Reverted...)
	}
	stdin, err := json.Marshal(payload)
	if err != nil {
		return fail(fmt.Sprintf("checker input: %v", err))
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = root
	cmd.Stdin = bytes.NewReader(stdin)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if err == syscall.ESRCH {
			return os.ErrProcessDone
		}
		return err
	}
	// Bound inherited pipes if a descendant outlives the shell or escapes its process group.
	cmd.WaitDelay = time.Second
	stdout := checkerOutput{onLimit: cancel}
	var stderr checkerOutput
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err = cmd.Run()
	if err != nil && cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	if ctx.Err() == context.DeadlineExceeded {
		return fail(fmt.Sprintf("checker timed out after %s", timeout))
	}
	if stdout.exceeded {
		return fail("checker stdout exceeded 1 MiB")
	}
	if err != nil {
		reason := fmt.Sprintf("checker failed: %v", err)
		if detail := strings.TrimSpace(stderr.buffer.String()); detail != "" {
			reason += ": " + detail
		}
		return fail(reason)
	}
	first := true
	for _, line := range strings.Split(stdout.buffer.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if first {
			first = false
			decision, reason, ok := strings.Cut(line, ": ")
			reason = strings.TrimSpace(reason)
			if !ok || (decision != "ALLOW" && decision != "BLOCK") || reason == "" {
				return fail("malformed checker first line: expected ALLOW: <reason> or BLOCK: <reason>")
			}
			verdict["accepted"] = decision == "ALLOW"
			verdict["checkerReason"] = reason
			continue
		}
		if !strings.HasPrefix(line, "ATTEST ") {
			continue
		}
		name, reason, ok := strings.Cut(strings.TrimPrefix(line, "ATTEST "), ": ")
		reason = strings.TrimSpace(reason)
		if !ok || reason == "" {
			continue
		}
		switch name {
		case "adversarial", "safety", "tampering":
			verdict[name+"Reviewed"] = true
			verdict[name+"Reason"] = reason
		case "vendorDiversity":
			verdict["vendorDiversityWaived"] = true
			verdict["vendorDiversityReason"] = reason
		}
	}
	if first {
		return fail("checker produced empty output")
	}
	return verdict
}

type checkerOutput struct {
	buffer   bytes.Buffer
	exceeded bool
	onLimit  func()
}

func (b *checkerOutput) Write(p []byte) (int, error) {
	n := len(p)
	remaining := checkerOutputLimit - b.buffer.Len()
	if n > remaining {
		if !b.exceeded && b.onLimit != nil {
			b.onLimit()
		}
		b.exceeded = true
		p = p[:remaining]
	}
	_, _ = b.buffer.Write(p)
	// Drain excess output while cancellation kills the checker; stderr only truncates.
	return n, nil
}

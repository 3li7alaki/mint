//go:build !windows

package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"mint/internal/unitstore"
)

func TestVerifySignalRemovesReplayWorktree(t *testing.T) {
	bin := buildBin(t)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	for _, tc := range []struct {
		name string
		sig  syscall.Signal
		code int
	}{
		{"SIGTERM", syscall.SIGTERM, 143},
		{"SIGINT", syscall.SIGINT, 130},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("MINT_STATE_HOME", t.TempDir())
			run := func(name string, args ...string) string {
				t.Helper()
				cmd := exec.Command(name, args...)
				cmd.Dir = root
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("%s %v: %v\n%s", name, args, err, out)
				}
				return string(out)
			}
			run("git", "init", "-q")
			run("git", "-c", "user.name=Test", "-c", "user.email=t@example.com", "commit", "--allow-empty", "-qm", "base")
			run(bin, "spec", "new", "signal cleanup", "--slug", "unit", "--gate", "tests: sleep 30")
			run(bin, "exec", "init", "unit", "001", "--attempt", "a1", "--executor", "codex", "--vendor", "openai", "--model", "gpt", "--locality", "remote", "--execution-ref", "maker")
			if err := os.WriteFile(filepath.Join(root, "app.go"), []byte("package app\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			log, err := os.Create(filepath.Join(t.TempDir(), "verify.log"))
			if err != nil {
				t.Fatal(err)
			}
			defer log.Close()
			cmd := exec.CommandContext(ctx, bin, "verify", "unit", "001", "--attempt", "a1")
			cmd.Dir = root
			cmd.Stdout, cmd.Stderr = log, log
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			defer syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			wait := make(chan error, 1)
			go func() { wait <- cmd.Wait() }()
			gateLog := filepath.Join(unitstore.GateLogDir(root, "unit", "001", "a1"), "01-tests.log")
			var replayDir string
			for replayDir == "" {
				listing := run("git", "worktree", "list", "--porcelain")
				if _, err := os.Stat(gateLog); err == nil {
					for _, line := range strings.Split(listing, "\n") {
						if strings.HasPrefix(line, "worktree ") && strings.Contains(line, "mint-replay-") {
							replayDir = strings.TrimPrefix(line, "worktree ")
						}
					}
				}
				if replayDir != "" {
					break
				}
				select {
				case err := <-wait:
					t.Fatalf("verify exited before its gate: %v", err)
				case <-ctx.Done():
					t.Fatal("replay worktree did not appear before deadline")
				case <-time.After(20 * time.Millisecond):
				}
			}
			if err := cmd.Process.Signal(tc.sig); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-wait:
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) || exitErr.ExitCode() != tc.code {
					t.Fatalf("verify exit=%v, want %d", err, tc.code)
				}
			case <-ctx.Done():
				t.Fatal("verify did not exit before deadline")
			}
			if got := run("git", "worktree", "list", "--porcelain"); strings.Contains(got, "mint-replay-") || strings.Count(got, "worktree ") != 1 {
				t.Fatalf("replay worktree remains: %s", got)
			}
			if _, err := os.Stat(replayDir); !os.IsNotExist(err) {
				t.Fatalf("replay directory remains: %v", err)
			}
		})
	}
}

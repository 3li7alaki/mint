//go:build !windows

package verify

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"mint/internal/execstate"
	"mint/internal/unitstore"
)

func TestRunVerifyLockHeld(t *testing.T) {
	for _, body := range []string{strconv.Itoa(os.Getpid()), "not a pid", ""} {
		t.Run(body, func(t *testing.T) {
			root := repo(t)
			t.Setenv("MINT_STATE_HOME", t.TempDir())
			spec := unitstore.SpecPath(root, "unit", "001")
			write(t, spec, `<task><gates>tests: true</gates></task>`)
			maker := &execstate.Maker{Executor: "codex", Vendor: "openai", Model: "gpt", Locality: "remote", ExecutionRef: "maker"}
			if _, err := execstate.Init(root, "unit", "001", "a1", maker); err != nil {
				t.Fatal(err)
			}
			if _, err := execstate.RecordGate(root, "unit", "001", "tests", "fail", "a1"); err != nil {
				t.Fatal(err)
			}
			attemptPath := unitstore.AttemptPath(root, "unit", "001", "a1")
			before, err := os.ReadFile(attemptPath)
			if err != nil {
				t.Fatal(err)
			}
			logPath := filepath.Join(unitstore.GateLogDir(root, "unit", "001", "a1"), "old.log")
			write(t, logPath, "prior output")
			lockPath := unitstore.VerifyLockPath(root, "unit", "001", "a1")
			holdVerifyLock(t, lockPath, body)
			write(t, filepath.Join(root, "app.go"), "package app\nvar X = 2\n")
			result := Run(root, "unit", "001", "a1", spec)
			want := "verify already running for unit/001/a1"
			if body == strconv.Itoa(os.Getpid()) {
				want += " (pid " + body + ")"
			}
			if result.Error != want || result.OK || len(result.Results) != 0 {
				t.Fatalf("result=%#v, want %q", result, want)
			}
			after, err := os.ReadFile(attemptPath)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("attempt changed: %q err=%v", after, err)
			}
			for path, want := range map[string]string{lockPath: body, logPath: "prior output"} {
				got, err := os.ReadFile(path)
				if err != nil || string(got) != want {
					t.Fatalf("file %q=%q, want %q err=%v", path, got, want, err)
				}
			}
		})
	}
}

func TestRunVerifyLockBeforeAttemptRead(t *testing.T) {
	t.Setenv("MINT_STATE_HOME", t.TempDir())
	root := t.TempDir()
	path := unitstore.VerifyLockPath(root, "unit", "001", "a1")
	holdVerifyLock(t, path, strconv.Itoa(os.Getpid()))
	result := Run(root, "unit", "001", "a1", "")
	want := fmt.Sprintf("verify already running for unit/001/a1 (pid %d)", os.Getpid())
	if result.Error != want {
		t.Fatalf("error=%q, want %q", result.Error, want)
	}
}

func holdVerifyLock(t *testing.T, path, body string) {
	t.Helper()
	write(t, path, body)
	file, err := os.OpenFile(path, os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { file.Close() })
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyLockHelper(t *testing.T) {
	if os.Getenv("MINT_VERIFY_LOCK_HELPER") != "1" {
		return
	}
	lock, err := acquireVerifyLock(os.Getenv("MINT_VERIFY_LOCK"), "unit", "001", "a1")
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	fmt.Println("locked")
	if _, err := io.Copy(io.Discard, os.Stdin); err != nil {
		t.Fatal(err)
	}
}

func TestRunVerifyLockReleasedOnKill(t *testing.T) {
	root := repo(t)
	t.Setenv("MINT_STATE_HOME", t.TempDir())
	spec := unitstore.SpecPath(root, "unit", "001")
	write(t, spec, `<task><gates>tests: true</gates></task>`)
	maker := &execstate.Maker{Executor: "codex", Vendor: "openai", Model: "gpt", Locality: "remote", ExecutionRef: "maker"}
	if _, err := execstate.Init(root, "unit", "001", "a1", maker); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, "app.go"), "package app\nvar X = 2\n")
	path := unitstore.VerifyLockPath(root, "unit", "001", "a1")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestVerifyLockHelper$")
	cmd.Env = append(os.Environ(), "MINT_VERIFY_LOCK_HELPER=1", "MINT_VERIFY_LOCK="+path)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	if line, err := bufio.NewReader(stdout).ReadString('\n'); err != nil || line != "locked\n" {
		t.Fatalf("helper readiness=%q err=%v", line, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	result := Run(root, "unit", "001", "a1", spec)
	want := fmt.Sprintf("verify already running for unit/001/a1 (pid %d)", cmd.Process.Pid)
	if result.Error != want {
		t.Fatalf("result=%#v, want %q", result, want)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	var exitErr *exec.ExitError
	if err := cmd.Wait(); !errors.As(err, &exitErr) || exitErr.Sys().(syscall.WaitStatus).Signal() != syscall.SIGKILL {
		t.Fatalf("helper exit=%v stderr=%s, want SIGKILL", err, &stderr)
	}
	result = Run(root, "unit", "001", "a1", spec)
	if result.Error != "" || !result.OK || result.Results["tests"] != "pass" {
		t.Fatalf("result=%#v", result)
	}
	after, err := os.Stat(path)
	if err != nil || !os.SameFile(info, after) {
		t.Fatalf("lock inode changed: %v", err)
	}
	assertVerifyLockFree(t, path)
}

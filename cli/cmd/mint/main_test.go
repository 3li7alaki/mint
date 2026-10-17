package main

import (
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestMainHelp(t *testing.T) {
	bin := buildBin(t)
	for _, spelling := range []string{"-h", "--help", "help"} {
		t.Run(spelling, func(t *testing.T) {
			cmd := exec.Command(bin, spelling)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("help must exit 0: %v (stderr=%q)", err, stderr.String())
			}
			if stdout.Len() == 0 {
				t.Fatalf("help must print to stdout (stderr=%q)", stderr.String())
			}
		})
	}
}

func TestMainUnknownArgument(t *testing.T) {
	bin := buildBin(t)
	cmd := exec.Command(bin, "definitely-not-a-command")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	var exitErr *exec.ExitError
	if err == nil || !errors.As(err, &exitErr) {
		t.Fatalf("an unrecognized argument must exit non-zero: %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("an unrecognized argument must not write stdout: %q", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Fatalf("an unrecognized argument must print a usage message to stderr")
	}
}

func TestParseSpecArgsProtected(t *testing.T) {
	for _, command := range []string{"new", "set"} {
		args, flags, err := parseSpecArgs([]string{command, "unit", "--protected", "a,b"})
		if err != nil || len(args) != 2 || args[0] != command || args[1] != "unit" || flags.Protected != "a,b" {
			t.Fatalf("args=%v flags=%#v err=%v", args, flags, err)
		}
	}
	if _, _, err := parseSpecArgs([]string{"new", "unit", "--protected"}); err == nil {
		t.Fatal("missing protected value accepted")
	}
}

func TestParseSpecArgsPrepare(t *testing.T) {
	for _, command := range []string{"new", "set"} {
		args, flags, err := parseSpecArgs([]string{command, "unit", "--prepare", `cp "$MINT_SOURCE_ROOT/.env" .`})
		if err != nil || len(args) != 2 || args[0] != command || args[1] != "unit" || flags.Prepare != `cp "$MINT_SOURCE_ROOT/.env" .` {
			t.Fatalf("args=%v flags=%#v err=%v", args, flags, err)
		}
	}
	if _, _, err := parseSpecArgs([]string{"new", "unit", "--prepare"}); err == nil {
		t.Fatal("missing prepare value accepted")
	}
}

func TestParseDoneCheckerFlags(t *testing.T) {
	args, flags, err := parseDoneArgs([]string{
		"unit", "001", "--checker", "printf 'ALLOW: looks right\\n'",
		"--checker-executor", "opencode", "--checker-vendor", "zai",
		"--checker-model", "glm", "--checker-locality", "remote",
		"--checker-execution-ref", "checker", "--checker-timeout", "1s",
	})
	if err != nil || !reflect.DeepEqual(args, []string{"unit", "001"}) || flags.Checker != "printf 'ALLOW: looks right\\n'" || flags.CheckerExecutor != "opencode" || flags.CheckerVendor != "zai" || flags.CheckerModel != "glm" || flags.CheckerLocality != "remote" || flags.CheckerExecutionRef != "checker" || flags.CheckerTimeout != time.Second {
		t.Fatalf("args=%v flags=%#v err=%v", args, flags, err)
	}
	_, flags, err = parseDoneArgs([]string{"unit", "001"})
	if err != nil || flags.CheckerTimeout != 15*time.Minute {
		t.Fatalf("default timeout=%s err=%v", flags.CheckerTimeout, err)
	}
}

func TestParseDoneCheckerInvalidFlags(t *testing.T) {
	for _, flag := range []string{"--checker", "--checker-executor", "--checker-vendor", "--checker-model", "--checker-locality", "--checker-execution-ref", "--checker-timeout"} {
		if _, _, err := parseDoneArgs([]string{"unit", "001", flag}); err == nil {
			t.Fatalf("missing %s value accepted", flag)
		}
	}
	for _, duration := range []string{"invalid", "0s", "-1s"} {
		if _, _, err := parseDoneArgs([]string{"unit", "001", "--checker-timeout", duration}); err == nil {
			t.Fatalf("invalid timeout %q accepted", duration)
		}
	}
	for _, args := range [][]string{
		{"--checker", ""}, {"--checker", " "},
		{"--checker", "true", "--verdict", "verdict.json"},
		{"--checker", "true", "--verdict", ""},
	} {
		if _, _, err := parseDoneArgs(args); err == nil {
			t.Fatalf("invalid arguments accepted: %v", args)
		}
	}
}

func buildBin(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "bin")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
	return bin
}

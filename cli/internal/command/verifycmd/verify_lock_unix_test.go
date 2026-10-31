//go:build !windows

package verifycmd

import (
	"bytes"
	"fmt"
	"os"
	"syscall"
	"testing"

	"mint/internal/statehome"
	"mint/internal/unitstore"
)

func TestRunReportsVerifyLockHolder(t *testing.T) {
	t.Setenv("MINT_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := statehome.Write(unitstore.SpecPath(root, "unit", "001"), []byte("<task/>")); err != nil {
		t.Fatal(err)
	}
	path := unitstore.VerifyLockPath(root, "unit", "001", "a1")
	if err := statehome.Write(path, []byte(fmt.Sprint(os.Getpid()))); err != nil {
		t.Fatal(err)
	}
	lock, err := os.OpenFile(path, os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	code, err := Run(root, []string{"unit", "001"}, Flags{Attempt: "a1"}, &stdout)
	want := fmt.Sprintf("verify: verify already running for unit/001/a1 (pid %d)", os.Getpid())
	if code != 1 || err == nil || err.Error() != want || stdout.Len() != 0 {
		t.Fatalf("code=%d err=%v stdout=%q, want %q", code, err, &stdout, want)
	}
}

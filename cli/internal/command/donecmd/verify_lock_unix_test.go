//go:build !windows

package donecmd

import (
	"bytes"
	"fmt"
	"os"
	"syscall"
	"testing"

	"mint/internal/receipt"
	"mint/internal/statehome"
	"mint/internal/unitstore"
)

func TestDoneReportsVerifyLockHolder(t *testing.T) {
	root := checkerUnit(t, false, "true")
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
	var stdout, stderr bytes.Buffer
	code, err := Run(root, []string{"unit", "001"}, Flags{Attempt: "a1"}, &stdout, &stderr)
	want := fmt.Sprintf("verify: verify already running for unit/001/a1 (pid %d)\n", os.Getpid())
	if code != 1 || err != nil || stderr.String() != want || stdout.Len() != 0 {
		t.Fatalf("code=%d err=%v stdout=%q stderr=%q, want %q", code, err, &stdout, &stderr, want)
	}
	if paths := receipt.List(root, "unit", "001"); len(paths) != 0 {
		t.Fatalf("receipts=%v", paths)
	}
}

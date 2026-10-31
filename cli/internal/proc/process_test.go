package proc

import (
	"os"
	"os/exec"
	"runtime"
	"testing"
)

func TestAliveCurrentProcess(t *testing.T) {
	if !Alive(os.Getpid()) {
		t.Fatal("current process reported dead")
	}
}

func TestAliveWaitedChild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows conservatively treats every process as alive")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	child := exec.Command(executable, "-test.run=^$")
	if err := child.Run(); err != nil {
		t.Fatal(err)
	}
	if Alive(child.Process.Pid) {
		t.Fatalf("waited child %d reported alive", child.Process.Pid)
	}
}

//go:build !windows

package proc

import (
	"errors"
	"syscall"
)

func Alive(pid int) bool {
	return !errors.Is(syscall.Kill(pid, 0), syscall.ESRCH)
}

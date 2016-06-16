// +build !windows

package utils

import (
	"os"
	"syscall"
)

func checkProcessRunning(process *os.Process) error {
	return process.Signal(syscall.Signal(0))
}

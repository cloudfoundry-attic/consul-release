// +build !windows

package utils

import (
	"os"
	"syscall"
)

func checkProcessRunning(process *os.Process) error {
	return process.Signal(syscall.Signal(0))
	// fmt.Println("checkProcessRunning", process.Pid, res)
	// time.Sleep(30 * time.Second)
	// return res
}

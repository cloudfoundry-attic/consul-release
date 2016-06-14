// +build windows

package utils

import (
	"os"
	"syscall"
)

func checkProcessRunning(process *os.Process) error {
	handle, e := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(process.Pid))
	if e != nil {
		return os.NewSyscallError("OpenProcess", e)
	}
	defer syscall.CloseHandle(handle)
	var ec uint32
	e = syscall.GetExitCodeProcess(syscall.Handle(handle), &ec)
	if e != nil {
		return os.NewSyscallError("GetExitCodeProcess", e)
	}
	return nil
}

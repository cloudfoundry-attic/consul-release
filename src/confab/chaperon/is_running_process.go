package chaperon

import (
	"io/ioutil"
	"os"
	"strconv"
	"syscall"
)

func IsRunningProcess(pidFilePath string) bool {
	pidFileContents, err := ioutil.ReadFile(pidFilePath)
	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(string(pidFileContents))
	if err != nil {
		return false
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return false // never returns an error according to the go docs
	}

	return proc.Signal(syscall.Signal(0)) == nil
}

package utils

import "os"

func CheckProcessRunning(process *os.Process) error {
	return checkProcessRunning(process)
}

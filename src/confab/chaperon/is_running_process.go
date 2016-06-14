package chaperon

import (
	"io/ioutil"
	"os"
	"strconv"

	"github.com/cloudfoundry-incubator/consul-release/src/confab/utils"
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

	return utils.CheckProcessRunning(proc) == nil
}

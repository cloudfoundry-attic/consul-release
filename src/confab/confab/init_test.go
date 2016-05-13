package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMain(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "confab/confab")
}

var (
	pathToFakeAgent string
	pathToConfab    string
)

var _ = BeforeSuite(func() {
	var err error
	pathToFakeAgent, err = gexec.Build("github.com/cloudfoundry-incubator/consul-release/src/confab/fakes/agent")
	Expect(err).NotTo(HaveOccurred())

	pathToConfab, err = gexec.Build("github.com/cloudfoundry-incubator/consul-release/src/confab/confab")
	Expect(err).NotTo(HaveOccurred())

	cmd := exec.Command("which", "lsof")
	Expect(cmd.Run()).To(Succeed())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

type FakeAgentOutputData struct {
	Args                []string
	PID                 int
	LeaveCallCount      int
	UseKeyCallCount     int
	InstallKeyCallCount int
	StatsCallCount      int
}

func killProcessWithPIDFile(pidFilePath string) {
	pidFileContents, err := ioutil.ReadFile(pidFilePath)
	if err != nil {
		return
	}

	pid, err := strconv.Atoi(string(pidFileContents))
	Expect(err).NotTo(HaveOccurred())

	killPID(pid)
}

func killPID(pid int) {
	process, err := os.FindProcess(pid)
	Expect(err).NotTo(HaveOccurred())

	process.Signal(syscall.SIGKILL)
}

func pidIsForRunningProcess(pidFilePath string) bool {
	pid, err := getPID(pidFilePath)
	if err != nil {
		return false
	}

	running, err := isPIDRunning(pid)
	if err != nil {
		return false
	}

	return running
}

func getPID(pidFilePath string) (int, error) {
	pidFileContents, err := ioutil.ReadFile(pidFilePath)
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(string(pidFileContents))
}

func isPIDRunning(pid int) (bool, error) {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false, err
	}

	if err := process.Signal(syscall.Signal(0)); err != nil {
		return false, err
	}

	return true, nil
}

func fakeAgentOutput(configDir string) (FakeAgentOutputData, error) {
	var decodedFakeOutput FakeAgentOutputData

	fakeOutput, err := ioutil.ReadFile(filepath.Join(configDir, "fake-output.json"))
	if err != nil {
		return decodedFakeOutput, err
	}

	err = json.Unmarshal(fakeOutput, &decodedFakeOutput)
	if err != nil {
		return decodedFakeOutput, err
	}

	return decodedFakeOutput, nil
}

func killProcessAttachedToPort(port int) {
	cmdLine := fmt.Sprintf("lsof -i :%d | tail -1 | cut -d' ' -f4", port)
	cmd := exec.Command("bash", "-c", cmdLine)
	buffer := bytes.NewBuffer([]byte{})
	cmd.Stdout = buffer
	Expect(cmd.Run()).To(Succeed())

	pidStr := strings.TrimSpace(buffer.String())
	if pidStr != "" {
		pid, err := strconv.Atoi(pidStr)
		Expect(err).NotTo(HaveOccurred())
		killPID(pid)
	}
}

func writeConfigurationFile(filename string, configuration map[string]interface{}) {
	configData, err := json.Marshal(configuration)
	Expect(err).NotTo(HaveOccurred())

	err = ioutil.WriteFile(filename, configData, os.ModePerm)
	Expect(err).NotTo(HaveOccurred())
}

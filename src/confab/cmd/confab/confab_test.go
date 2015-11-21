package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const COMMAND_TIMEOUT = "15s"

var _ = Describe("confab", func() {
	var (
		tempDir         string
		consulConfigDir string
		pidFile         *os.File
	)

	BeforeEach(func() {
		var err error
		tempDir, err = ioutil.TempDir("", "testing")
		Expect(err).NotTo(HaveOccurred())

		consulConfigDir, err = ioutil.TempDir(tempDir, "fake-agent-config-dir")
		Expect(err).NotTo(HaveOccurred())

		pidFile, err = ioutil.TempFile(tempDir, "fake-pid-file")
		Expect(err).NotTo(HaveOccurred())

		options := []byte(`{"Members": ["member-1", "member-2", "member-3"]}`)
		err = ioutil.WriteFile(filepath.Join(consulConfigDir, "options.json"), options, 0600)
		Expect(err).NotTo(HaveOccurred())

	})

	AfterEach(func() {
		killProcessAttachedToPort(8400)
		killProcessAttachedToPort(8500)

		err := os.RemoveAll(tempDir)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("when managing the entire process lifecycle", func() {
		It("starts and stops the consul process as a daemon", func() {
			start := exec.Command(pathToConfab,
				"start",
				"--server=false",
				"--pid-file", pidFile.Name(),
				"--agent-path", pathToFakeAgent,
				"--consul-config-dir", consulConfigDir,
				"--expected-member", "member-1",
				"--expected-member", "member-2",
				"--expected-member", "member-3",
			)
			Eventually(start.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).Should(Succeed())

			pid, err := getPID(pidFile.Name())
			Expect(err).NotTo(HaveOccurred())
			Expect(isPIDRunning(pid)).To(BeTrue())

			stop := exec.Command(pathToConfab,
				"stop",
				"--server=false",
				"--pid-file", pidFile.Name(),
				"--agent-path", pathToFakeAgent,
				"--consul-config-dir", consulConfigDir,
				"--expected-member", "member-1",
				"--expected-member", "member-2",
				"--expected-member", "member-3",
			)
			Eventually(stop.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).Should(Succeed())

			_, err = isPIDRunning(pid)
			Expect(err).To(MatchError(ContainSubstring("process already finished")))

			Expect(fakeAgentOutput(consulConfigDir)).To(Equal(map[string]interface{}{
				"PID": float64(pid),
				"Args": []interface{}{
					"agent",
					fmt.Sprintf("-config-dir=%s", consulConfigDir),
				},
				"LeaveCallCount":      float64(1),
				"UseKeyCallCount":     float64(0),
				"InstallKeyCallCount": float64(0),
			}))
		})
	})

	Context("when starting", func() {
		AfterEach(func() {
			killProcessWithPIDFile(pidFile.Name())
		})

		Context("for a client", func() {
			It("starts a consul agent as a client", func() {
				cmd := exec.Command(pathToConfab,
					"start",
					"--server=false",
					"--pid-file", pidFile.Name(),
					"--agent-path", pathToFakeAgent,
					"--consul-config-dir", consulConfigDir,
					"--expected-member", "member-1",
					"--expected-member", "member-2",
					"--expected-member", "member-3",
				)
				Eventually(cmd.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).Should(Succeed())

				pid, err := getPID(pidFile.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(isPIDRunning(pid)).To(BeTrue())

				Expect(fakeAgentOutput(consulConfigDir)).To(Equal(map[string]interface{}{
					"PID": float64(pid),
					"Args": []interface{}{
						"agent",
						fmt.Sprintf("-config-dir=%s", consulConfigDir),
					},
					"LeaveCallCount":      float64(0),
					"UseKeyCallCount":     float64(0),
					"InstallKeyCallCount": float64(0),
				}))
			})
		})

		Context("for a server", func() {
			BeforeEach(func() {
				options := []byte(`{"Members": ["member-1", "member-2", "member-3"]}`)
				Expect(ioutil.WriteFile(filepath.Join(consulConfigDir, "options.json"), options, 0600)).To(Succeed())
			})

			AfterEach(func() {
				killProcessWithPIDFile(pidFile.Name())
			})

			It("starts a consul agent as a server", func() {
				cmd := exec.Command(pathToConfab,
					"start",
					"--server=true",
					"--pid-file", pidFile.Name(),
					"--agent-path", pathToFakeAgent,
					"--consul-config-dir", consulConfigDir,
					"--expected-member", "member-1",
					"--expected-member", "member-2",
					"--expected-member", "member-3",
					"--encryption-key", "key-1",
					"--encryption-key", "key-2",
				)
				Eventually(cmd.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).Should(Succeed())

				pid, err := getPID(pidFile.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(isPIDRunning(pid)).To(BeTrue())

				Eventually(func() (map[string]interface{}, error) {
					return fakeAgentOutput(consulConfigDir)
				}, "2s").Should(Equal(map[string]interface{}{
					"PID": float64(pid),
					"Args": []interface{}{
						"agent",
						fmt.Sprintf("-config-dir=%s", consulConfigDir),
					},
					"LeaveCallCount":      float64(0),
					"InstallKeyCallCount": float64(2),
					"UseKeyCallCount":     float64(1),
				}))
			})
		})
	})

	Context("when stopping", func() {
		BeforeEach(func() {
			options := []byte(`{"Members": ["member-1", "member-2", "member-3"]}`)
			Expect(ioutil.WriteFile(filepath.Join(consulConfigDir, "options.json"), options, 0600)).To(Succeed())
		})

		It("stops the consul agent", func() {
			cmd := exec.Command(pathToConfab,
				"start",
				"--server=true",
				"--pid-file", pidFile.Name(),
				"--agent-path", pathToFakeAgent,
				"--consul-config-dir", consulConfigDir,
				"--expected-member", "member-1",
				"--expected-member", "member-2",
				"--expected-member", "member-3",
				"--encryption-key", "key-1",
				"--encryption-key", "key-2",
			)
			Eventually(cmd.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).Should(Succeed())
			Eventually(func() error {
				conn, err := net.Dial("tcp", "localhost:8400")
				if err == nil {
					conn.Close()
				}
				return err
			}, "5s").Should(Succeed())

			cmd = exec.Command(pathToConfab,
				"stop",
				"--pid-file", pidFile.Name(),
				"--agent-path", pathToFakeAgent,
			)
			Eventually(cmd.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).Should(Succeed())

			Eventually(func() bool {
				return pidIsForRunningProcess(pidFile.Name())
			}, "5s").Should(BeFalse())

			pid, err := getPID(pidFile.Name())
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeAgentOutput(consulConfigDir)).To(Equal(map[string]interface{}{
				"PID": float64(pid),
				"Args": []interface{}{
					"agent",
					fmt.Sprintf("-config-dir=%s", consulConfigDir),
				},
				"LeaveCallCount":      float64(1),
				"InstallKeyCallCount": float64(2),
				"UseKeyCallCount":     float64(1),
			}))
		})
	})

	Context("failure cases", func() {
		Context("when no arguments are provided", func() {
			It("returns a non-zero status code and prints usage", func() {
				cmd := exec.Command(pathToConfab)
				buffer := bytes.NewBuffer([]byte{})
				cmd.Stderr = buffer
				Eventually(cmd.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).ShouldNot(Succeed())
				Expect(buffer).To(ContainSubstring("invalid number of arguments"))

				usageLines := []string{
					"usage: confab COMMAND OPTIONS",
					"COMMAND: \"start\" or \"stop\"",
					"-agent-path executable",
					"path to the on-filesystem consul executable",
					"-consul-config-dir directory",
					"path to consul configuration directory",
					"-expected-member list",
					"address list of the expected members",
					"-pid-file file",
					"path to consul PID file",
				}
				for _, line := range usageLines {
					Expect(buffer).To(ContainSubstring(line))
				}
			})
		})

		Context("when no command is provided", func() {
			It("returns a non-zero status code and prints usage", func() {
				cmd := exec.Command(pathToConfab,
					"--server=false",
					"--agent-path", pathToFakeAgent,
					"--pid-file", pidFile.Name(),
				)
				buffer := bytes.NewBuffer([]byte{})
				cmd.Stderr = buffer
				Eventually(cmd.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).ShouldNot(Succeed())
				Expect(buffer).To(ContainSubstring("invalid COMMAND \"--server=false\""))
				Expect(buffer).To(ContainSubstring("usage: confab COMMAND OPTIONS"))
			})
		})

		Context("when an invalid command is provided", func() {
			It("returns a non-zero status code and prints usage", func() {
				cmd := exec.Command(pathToConfab, "banana",
					"--agent-path", pathToFakeAgent,
					"--pid-file", pidFile.Name(),
					"--consul-config-dir", consulConfigDir)
				buffer := bytes.NewBuffer([]byte{})
				cmd.Stderr = buffer
				Eventually(cmd.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).ShouldNot(Succeed())
				Expect(buffer).To(ContainSubstring("invalid COMMAND \"banana\""))
				Expect(buffer).To(ContainSubstring("usage: confab COMMAND OPTIONS"))
			})
		})

		Context("expected-member is missing", func() {
			It("prints an error and usage", func() {
				cmd := exec.Command(pathToConfab, "start",
					"--agent-path", pathToFakeAgent,
					"--pid-file", pidFile.Name(),
					"--consul-config-dir", consulConfigDir)
				buffer := bytes.NewBuffer([]byte{})
				cmd.Stderr = buffer
				Eventually(cmd.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).ShouldNot(Succeed())
				Expect(buffer).To(ContainSubstring("at least one \"expected-member\" must be provided"))
				Expect(buffer).To(ContainSubstring("usage: confab COMMAND OPTIONS"))
			})
		})

		Context("when the agent executable does not exist", func() {
			It("prints an error and usage", func() {
				cmd := exec.Command(pathToConfab, "start",
					"--expected-member", "member-1",
					"--agent-path", "/tmp/path/that/does/not/exist",
					"--pid-file", pidFile.Name(),
					"--consul-config-dir", consulConfigDir)
				buffer := bytes.NewBuffer([]byte{})
				cmd.Stderr = buffer
				Eventually(cmd.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).ShouldNot(Succeed())
				Expect(buffer).To(ContainSubstring("\"agent-path\" \"/tmp/path/that/does/not/exist\" cannot be found"))
				Expect(buffer).To(ContainSubstring("usage: confab COMMAND OPTIONS"))
			})
		})

		Context("when the PID file option is not provided", func() {
			It("prints an error and usage", func() {
				cmd := exec.Command(pathToConfab, "start",
					"--expected-member", "member-1",
					"--agent-path", pathToFakeAgent,
					"--consul-config-dir", consulConfigDir)
				buffer := bytes.NewBuffer([]byte{})
				cmd.Stderr = buffer
				Eventually(cmd.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).ShouldNot(Succeed())
				Expect(buffer).To(ContainSubstring("\"pid-file\" cannot be empty"))
				Expect(buffer).To(ContainSubstring("usage: confab COMMAND OPTIONS"))
			})
		})

		Context("when the consul config dir is not provided", func() {
			It("prints an error and usage", func() {
				cmd := exec.Command(pathToConfab, "start",
					"--expected-member", "member-1",
					"--agent-path", pathToFakeAgent,
					"--pid-file", pidFile.Name(),
					"--consul-config-dir", "/tmp/path/that/does/not/exist")
				buffer := bytes.NewBuffer([]byte{})
				cmd.Stderr = buffer
				Eventually(cmd.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).ShouldNot(Succeed())
				Expect(buffer).To(ContainSubstring("\"consul-config-dir\" \"/tmp/path/that/does/not/exist\" could not be found"))
				Expect(buffer).To(ContainSubstring("usage: confab COMMAND OPTIONS"))
			})
		})

		Context("when the pid file contains the pid of a running process", func() {
			It("prints an error and exits status 1", func() {
				myPID := os.Getpid()
				Expect(ioutil.WriteFile(pidFile.Name(), []byte(fmt.Sprintf("%d", myPID)), 0644)).To(Succeed())

				cmd := exec.Command(pathToConfab,
					"start",
					"--server=false",
					"--pid-file", pidFile.Name(),
					"--agent-path", pathToFakeAgent,
					"--consul-config-dir", consulConfigDir,
					"--expected-member", "member-1",
					"--expected-member", "member-2",
					"--expected-member", "member-3",
				)
				buffer := bytes.NewBuffer([]byte{})
				cmd.Stderr = buffer
				Eventually(cmd.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).ShouldNot(Succeed())
				Expect(buffer).To(ContainSubstring("error booting consul agent"))
				Expect(buffer).To(ContainSubstring("already running"))
			})
		})

		Context("when the rpc connection cannot be created", func() {
			It("returns an error and exits with status 1", func() {
				options := []byte(`{ "Members": ["member-1", "member-2", "member-3"], "FailRPCServer": true }`)
				Expect(ioutil.WriteFile(filepath.Join(consulConfigDir, "options.json"), options, 0600)).To(Succeed())

				cmd := exec.Command(pathToConfab,
					"start",
					"--server=true",
					"--pid-file", pidFile.Name(),
					"--agent-path", pathToFakeAgent,
					"--consul-config-dir", consulConfigDir,
					"--expected-member", "member-1",
					"--expected-member", "member-2",
					"--expected-member", "member-3",
				)
				buffer := bytes.NewBuffer([]byte{})
				cmd.Stderr = buffer
				Eventually(cmd.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).ShouldNot(Succeed())
				Expect(buffer).To(ContainSubstring("error connecting to RPC server"))
			})
		})
	})
})

func killProcessWithPIDFile(pidFilePath string) {
	pidFileContents, err := ioutil.ReadFile(pidFilePath)
	Expect(err).NotTo(HaveOccurred())

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

func fakeAgentOutput(configDir string) (map[string]interface{}, error) {
	fakeOutput, err := ioutil.ReadFile(filepath.Join(configDir, "fake-output.json"))
	if err != nil {
		return nil, err
	}

	var decodedFakeOutput map[string]interface{}
	err = json.Unmarshal(fakeOutput, &decodedFakeOutput)
	if err != nil {
		return nil, err
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

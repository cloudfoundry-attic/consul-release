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
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const COMMAND_TIMEOUT = "15s"

var _ = Describe("confab", func() {
	var (
		tempDir         string
		consulConfigDir string
		pidFile         *os.File
		configFile      *os.File
	)

	BeforeEach(func() {
		var err error
		tempDir, err = ioutil.TempDir("", "testing")
		Expect(err).NotTo(HaveOccurred())

		consulConfigDir, err = ioutil.TempDir(tempDir, "fake-agent-config-dir")
		Expect(err).NotTo(HaveOccurred())

		pidFile, err = ioutil.TempFile(tempDir, "fake-pid-file")
		Expect(err).NotTo(HaveOccurred())

		err = os.Remove(pidFile.Name())
		Expect(err).NotTo(HaveOccurred())

		configFile, err = ioutil.TempFile(tempDir, "config-file")
		Expect(err).NotTo(HaveOccurred())

		configData, err := json.Marshal(map[string]interface{}{
			"node": map[string]interface{}{
				"name":  "my-node",
				"index": 3,
			},
			"agent": map[string]interface{}{
				"services": map[string]interface{}{
					"cloud_controller": map[string]interface{}{
						"checks": []map[string]string{{
							"name":     "do_something",
							"script":   "/var/vcap/jobs/cloudcontroller/bin/do_something",
							"interval": "5m",
						}},
					},
					"router": map[string]interface{}{
						"name": "gorouter",
					},
				},
			},
		})

		_, err = configFile.Write(configData)
		Expect(err).NotTo(HaveOccurred())

		options := []byte(`{"Members": ["member-1", "member-2", "member-3"]}`)
		err = ioutil.WriteFile(filepath.Join(consulConfigDir, "options.json"), options, 0600)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		killProcessAttachedToPort(8400)
		killProcessAttachedToPort(8500)

		err := os.Chmod(consulConfigDir, os.ModePerm)
		Expect(err).NotTo(HaveOccurred())

		err = os.RemoveAll(tempDir)
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
				"--recursor", "8.8.8.8",
				"--recursor", "10.0.2.3",
				"--config-file", configFile.Name(),
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
				"--config-file", configFile.Name(),
			)
			Eventually(stop.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).Should(Succeed())

			_, err = isPIDRunning(pid)
			Expect(err).To(MatchError(ContainSubstring("process already finished")))

			Expect(fakeAgentOutput(consulConfigDir)).To(Equal(map[string]interface{}{
				"PID": float64(pid),
				"Args": []interface{}{
					"agent",
					fmt.Sprintf("-config-dir=%s", consulConfigDir),
					"-recursor=8.8.8.8",
					"-recursor=10.0.2.3",
				},
				"LeaveCallCount":      float64(1),
				"UseKeyCallCount":     float64(0),
				"InstallKeyCallCount": float64(0),
				"StatsCallCount":      float64(0),
			}))

			serviceConfig, err := ioutil.ReadFile(filepath.Join(consulConfigDir, "service-cloud_controller.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(serviceConfig)).To(MatchJSON(`{
				"service": {
					"name": "cloud-controller",
					"check": {
						"name": "dns_health_check",
						"script": "/var/vcap/jobs/cloud_controller/bin/dns_health_check",
						"interval": "3s"
					},
					"checks": [
						{
							"name": "do_something",
							"script": "/var/vcap/jobs/cloudcontroller/bin/do_something",
							"interval": "5m"
						}
					],
					"tags": ["my-node-3"]
				}
			}`))

			serviceConfig, err = ioutil.ReadFile(filepath.Join(consulConfigDir, "service-router.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(serviceConfig)).To(MatchJSON(`{
				"service": {
					"name": "gorouter",
					"check": {
						"name": "dns_health_check",
						"script": "/var/vcap/jobs/router/bin/dns_health_check",
						"interval": "3s"
					},
					"tags": ["my-node-3"]
				}
			}`))
		})

		Context("when ssl-disabled is set to true", func() {
			It("does not set encryption keys", func() {
				start := exec.Command(pathToConfab,
					"start",
					"--server",
					"--pid-file", pidFile.Name(),
					"--agent-path", pathToFakeAgent,
					"--consul-config-dir", consulConfigDir,
					"--expected-member", "member-1",
					"--expected-member", "member-2",
					"--expected-member", "member-3",
					"--encryption-key", "key-1",
					"--encryption-key", "key-2",
					"--ssl-disabled",
					"--config-file", configFile.Name(),
				)
				Eventually(start.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).Should(Succeed())

				pid, err := getPID(pidFile.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(isPIDRunning(pid)).To(BeTrue())

				stop := exec.Command(pathToConfab,
					"stop",
					"--server",
					"--pid-file", pidFile.Name(),
					"--agent-path", pathToFakeAgent,
					"--consul-config-dir", consulConfigDir,
					"--expected-member", "member-1",
					"--expected-member", "member-2",
					"--expected-member", "member-3",
					"--encryption-key", "key-1",
					"--encryption-key", "key-2",
					"--ssl-disabled",
					"--config-file", configFile.Name(),
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
					"StatsCallCount":      float64(1),
				}))
			})
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
					"--config-file", configFile.Name(),
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
					"StatsCallCount":      float64(0),
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
					"--config-file", configFile.Name(),
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
					"StatsCallCount":      float64(1),
				}))
			})

			It("checks sync state up to the timeout", func() {
				options := []byte(`{"Members": ["member-1", "member-2", "member-3"], "FailStatsEndpoint": true}`)
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
					"--encryption-key", "key-1",
					"--encryption-key", "key-2",
					"--timeout-seconds", "3",
					"--config-file", configFile.Name(),
				)

				start := time.Now()
				Eventually(cmd.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).ShouldNot(Succeed())
				Expect(time.Now()).To(BeTemporally("~", start.Add(3*time.Second), 1*time.Second))

				output, err := fakeAgentOutput(consulConfigDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(output["StatsCallCount"]).To(BeNumerically(">", 0))
				Expect(output["StatsCallCount"]).To(BeNumerically("<", 4))
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
				"--config-file", configFile.Name(),
			)
			Eventually(cmd.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).Should(Succeed())
			Eventually(func() error {
				conn, err := net.Dial("tcp", "localhost:8400")
				if err == nil {
					conn.Close()
				}
				return err
			}, "5s").Should(Succeed())

			pid, err := getPID(pidFile.Name())
			Expect(err).NotTo(HaveOccurred())

			cmd = exec.Command(pathToConfab,
				"stop",
				"--pid-file", pidFile.Name(),
				"--config-file", configFile.Name(),
				"--agent-path", pathToFakeAgent,
			)
			Eventually(cmd.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).Should(Succeed())

			Eventually(func() bool {
				return pidIsForRunningProcess(pidFile.Name())
			}, "5s").Should(BeFalse())

			Expect(fakeAgentOutput(consulConfigDir)).To(Equal(map[string]interface{}{
				"PID": float64(pid),
				"Args": []interface{}{
					"agent",
					fmt.Sprintf("-config-dir=%s", consulConfigDir),
				},
				"LeaveCallCount":      float64(1),
				"InstallKeyCallCount": float64(2),
				"UseKeyCallCount":     float64(1),
				"StatsCallCount":      float64(1),
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
					"-config-file",
					"specifies the config file",
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
					"--config-file", configFile.Name(),
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
					"--consul-config-dir", consulConfigDir,
					"--config-file", configFile.Name())
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
					"--consul-config-dir", consulConfigDir,
					"--config-file", configFile.Name())
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
					"--consul-config-dir", consulConfigDir,
					"--config-file", configFile.Name())
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
					"--consul-config-dir", consulConfigDir,
					"--config-file", configFile.Name())
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
					"--consul-config-dir", "/tmp/path/that/does/not/exist",
					"--config-file", configFile.Name())
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
					"--config-file", configFile.Name(),
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
					"--config-file", configFile.Name(),
				)
				buffer := bytes.NewBuffer([]byte{})
				cmd.Stderr = buffer
				Eventually(cmd.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).ShouldNot(Succeed())
				Expect(buffer).To(ContainSubstring("error connecting to RPC server"))
			})
		})

		Context("when an invalid flag is provided", func() {
			It("exits and prints usage", func() {
				cmd := exec.Command(pathToConfab, "start", "--banana")
				buffer := bytes.NewBuffer([]byte{})
				cmd.Stderr = buffer

				Eventually(cmd.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).ShouldNot(Succeed())
				Expect(buffer).To(ContainSubstring("flag provided but not defined: -banana"))
				Expect(buffer).NotTo(ContainSubstring("usage: confab COMMAND OPTIONS"))
			})
		})

		Context("when the config file does not exist", func() {
			It("returns an error and exits with status 1", func() {
				cmd := exec.Command(pathToConfab,
					"start",
					"--pid-file", pidFile.Name(),
					"--agent-path", pathToFakeAgent,
					"--config-file", "/some-missing-file.json",
				)
				buffer := bytes.NewBuffer([]byte{})
				cmd.Stderr = buffer
				Eventually(cmd.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).ShouldNot(Succeed())
				Expect(buffer).To(ContainSubstring("no such file or directory"))
			})
		})

		Context("when the config file is malformed json", func() {
			It("returns an error and exits with status 1", func() {
				tmpFile, err := ioutil.TempFile(tempDir, "config")
				Expect(err).NotTo(HaveOccurred())

				_, err = tmpFile.Write([]byte(`%%%%%%%%%`))
				Expect(err).NotTo(HaveOccurred())

				cmd := exec.Command(pathToConfab,
					"start",
					"--pid-file", pidFile.Name(),
					"--agent-path", pathToFakeAgent,
					"--config-file", tmpFile.Name(),
				)
				buffer := bytes.NewBuffer([]byte{})
				cmd.Stderr = buffer
				Eventually(cmd.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).ShouldNot(Succeed())
				Expect(buffer).To(ContainSubstring("invalid character"))
			})
		})

		Context("when the consul config dir is not writeable", func() {
			It("returns an error and exits with status 1", func() {
				err := os.Chmod(consulConfigDir, 0000)
				Expect(err).NotTo(HaveOccurred())

				cmd := exec.Command(pathToConfab,
					"start",
					"--pid-file", pidFile.Name(),
					"--agent-path", pathToFakeAgent,
					"--config-file", configFile.Name(),
					"--expected-member", "member-1",
					"--consul-config-dir", consulConfigDir,
				)
				buffer := bytes.NewBuffer([]byte{})
				cmd.Stderr = buffer
				Eventually(cmd.Run, COMMAND_TIMEOUT, COMMAND_TIMEOUT).ShouldNot(Succeed())
				Expect(buffer).To(ContainSubstring("permission denied"))
			})
		})
	})
})

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

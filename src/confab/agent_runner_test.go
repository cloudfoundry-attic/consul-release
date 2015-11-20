package confab_test

import (
	"bytes"
	"confab"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type FakeAgentOutput struct {
	Args []string
	PID  int
}

func getFakeAgentOutput(runner confab.AgentRunner) FakeAgentOutput {
	bytes, err := ioutil.ReadFile(filepath.Join(runner.ConfigDir, "fake-output.json"))
	if err != nil {
		return FakeAgentOutput{}
	}
	var output FakeAgentOutput
	if err = json.Unmarshal(bytes, &output); err != nil {
		return FakeAgentOutput{}
	}
	return output
}

func getPID(runner confab.AgentRunner) (int, error) {
	pidFileContents, err := ioutil.ReadFile(runner.PIDFile)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(string(pidFileContents))
	if err != nil {
		return 0, err
	}

	return pid, nil
}

func processIsRunning(runner confab.AgentRunner) bool {
	pid, err := getPID(runner)
	Expect(err).NotTo(HaveOccurred())

	process, err := os.FindProcess(pid)
	Expect(err).NotTo(HaveOccurred())

	errorSendingSignal := process.Signal(syscall.Signal(0))

	return (errorSendingSignal == nil)
}

var _ = Describe("AgentRunner", func() {
	var (
		runner confab.AgentRunner
	)

	BeforeEach(func() {
		var err error
		configDir, err := ioutil.TempDir("", "fake-agent-config-dir")
		Expect(err).NotTo(HaveOccurred())

		pidFile, err := ioutil.TempFile("", "fake-agent-pid")
		Expect(err).NotTo(HaveOccurred())
		pidFile.Close()
		os.Remove(pidFile.Name()) // so that the pid file doesn't exist at all
		pidFileName := pidFile.Name()

		runner = confab.AgentRunner{
			Path:      pathToFakeProcess,
			ConfigDir: configDir,
			PIDFile:   pidFileName,
			// Stdout:    os.Stdout,  // uncomment this to see output from test agent
			// Stderr:    os.Stderr,
		}
	})

	AfterEach(func() {
		os.Remove(runner.PIDFile)
		os.RemoveAll(runner.ConfigDir)
	})

	Describe("stop", func() {
		It("kills the process", func() {
			By("launching the process, configured to spin")
			Expect(ioutil.WriteFile(filepath.Join(runner.ConfigDir, "options.json"), []byte(`{ "WaitForHUP": true }`), 0600)).To(Succeed())
			Expect(runner.Run()).To(Succeed())

			By("waiting for the process to start enough that it has ignored signals")
			Eventually(func() error {
				_, err := os.Stat(filepath.Join(runner.ConfigDir, "fake-output.json"))
				return err
			}).Should(Succeed())

			By("calling stop")
			Expect(runner.Stop()).To(Succeed())

			By("checking that the process no longer exists")
			Eventually(func() bool { return processIsRunning(runner) }).Should(BeFalse())
		})

		Context("when the PID file cannot be read", func() {
			It("returns an error", func() {
				Expect(runner.Run()).To(Succeed())

				runner.PIDFile = "/tmp/nope-i-do-not-exist"
				Expect(runner.Stop()).To(MatchError(ContainSubstring("no such file or directory")))
			})
		})

		Context("when the PID file contains nonsense", func() {
			It("returns an error", func() {
				Expect(runner.Run()).To(Succeed())

				Expect(ioutil.WriteFile(runner.PIDFile, []byte("nonsense"), 0644)).To(Succeed())
				Expect(runner.Stop()).To(MatchError(ContainSubstring("ParseInt")))
			})
		})

		Context("when the PID file has the wrong PID", func() {
			It("returns an error", func() {
				Expect(runner.Run()).To(Succeed())

				Expect(ioutil.WriteFile(runner.PIDFile, []byte("-10"), 0644)).To(Succeed())
				Expect(runner.Stop()).To(HaveOccurred())
			})
		})
	})

	Describe("stop & wait", func() {
		It("stops the process / waits until it exits", func() {
			By("launching the process, configured to spin")
			Expect(ioutil.WriteFile(filepath.Join(runner.ConfigDir, "options.json"), []byte(`{ "WaitForHUP": true }`), 0600)).To(Succeed())
			Expect(runner.Run()).To(Succeed())

			By("waiting for the process to get started")
			Eventually(func() error {
				_, err := os.Stat(filepath.Join(runner.ConfigDir, "fake-output.json"))
				return err
			}).Should(Succeed())

			By("checking that the process is running")
			Expect(processIsRunning(runner)).To(BeTrue())

			By("checking that Wait() blocks", func() {})
			done := make(chan struct{})
			go func() {
				if err := runner.Wait(); err != nil {
					panic(err)
				}
				done <- struct{}{}
			}()
			Consistently(done, "100ms").ShouldNot(Receive())

			By("stopping the process", func() {})
			Expect(runner.Stop()).To(Succeed())

			By("checking that Wait returns")
			Expect(runner.Wait()).To(Succeed())

			By("checking that the process no longer exists")
			Eventually(func() bool { return processIsRunning(runner) }).Should(BeFalse())
		})

		Context("when the PID file cannot be read", func() {
			It("returns an error", func() {
				Expect(runner.Run()).To(Succeed())

				runner.PIDFile = "/tmp/nope-i-do-not-exist"
				Expect(runner.Wait()).To(MatchError(ContainSubstring("no such file or directory")))
			})
		})

		Context("when the PID file contains nonsense", func() {
			It("returns an error", func() {
				Expect(runner.Run()).To(Succeed())

				Expect(ioutil.WriteFile(runner.PIDFile, []byte("nonsense"), 0644)).To(Succeed())
				Expect(runner.Wait()).To(MatchError(ContainSubstring("ParseInt")))
			})
		})
	})

	Describe("Run", func() {
		It("writes the pid of the agent process to the pid file", func() {
			Expect(runner.Run()).To(Succeed())

			Expect(runner.Wait()).To(Succeed())
			pid, err := getPID(runner)
			Expect(err).NotTo(HaveOccurred())

			outputs := getFakeAgentOutput(runner)
			Expect(pid).To(Equal(outputs.PID))
		})

		It("makes the pid file world readable", func() {
			Expect(runner.Run()).To(Succeed())

			Expect(runner.Wait()).To(Succeed())

			fileInfo, err := os.Stat(runner.PIDFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(fileInfo.Mode().Perm()).To(BeEquivalentTo(0644))
		})

		It("sets the arguments correctly", func() {
			Expect(runner.Run()).To(Succeed())
			Expect(runner.Wait()).To(Succeed())

			Expect(getFakeAgentOutput(runner).Args).To(Equal([]string{
				"agent",
				fmt.Sprintf("-config-dir=%s", runner.ConfigDir),
			}))
		})

		It("returns without waiting for the process to exit", func() {
			Expect(ioutil.WriteFile(filepath.Join(runner.ConfigDir, "options.json"), []byte(`{ "WaitForHUP": true }`), 0600)).To(Succeed())
			done := make(chan struct{})
			go func() {
				runner.Run()
				done <- struct{}{}
			}()
			Eventually(done, "1s").Should(Receive())

			Eventually(func() bool { return processIsRunning(runner) }).Should(BeTrue())

			err := runner.Stop()
			Expect(err).NotTo(HaveOccurred())
		})

		It("wires up the stdout and stderr pipes", func() {
			stdoutBytes := &bytes.Buffer{}
			stderrBytes := &bytes.Buffer{}
			runner.Stdout = stdoutBytes
			runner.Stderr = stderrBytes

			Expect(runner.Run()).To(Succeed())
			Expect(runner.Wait()).To(Succeed())

			Expect(stdoutBytes.String()).To(Equal("some standard out"))
			Expect(stderrBytes.String()).To(Equal("some standard error"))
		})

		Context("when the pid file already exists", func() {
			Context("when the pid file points at the pid of a currently running process", func() {
				It("errors without running the command", func() {
					myPID := os.Getpid()
					Expect(ioutil.WriteFile(runner.PIDFile, []byte(fmt.Sprintf("%d", myPID)), 0666)).To(Succeed())

					Expect(runner.Run()).To(MatchError("consul_agent is already running, please stop it first"))
					Expect(getFakeAgentOutput(runner).PID).To(Equal(0))
				})
			})
			Context("when the pid file points to a non-existent process", func() {
				It("overwrites the stale pid file and succeeds", func() {
					Expect(ioutil.WriteFile(runner.PIDFile, []byte("-1"), 0666)).To(Succeed())

					Expect(runner.Run()).To(Succeed())
					pidFileContents, err := ioutil.ReadFile(runner.PIDFile)
					Expect(err).NotTo(HaveOccurred())
					Expect(pidFileContents).NotTo(Equal([]byte("some-pid")))
				})
			})
		})

		Context("when writing the PID file errors", func() {
			It("returns the error", func() {
				Expect(ioutil.WriteFile(runner.PIDFile, []byte("some-pid"), 0100)).To(Succeed())

				Expect(runner.Run()).To(MatchError(ContainSubstring("error writing PID file")))
				Expect(runner.Run()).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		Context("when starting the process fails", func() {
			It("returns the error and does not create a pid file", func() {
				runner.Path = "/tmp/not-a-thing-we-can-launch"
				Expect(runner.Run()).To(MatchError(ContainSubstring("no such file or directory")))

				_, err := os.Stat(runner.PIDFile)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when the ConfigDir is missing", func() {
			It("returns an error immediately, without starting a process", func() {
				runner.ConfigDir = fmt.Sprintf("/tmp/this-directory-does-not-existi-%x", rand.Int31())
				Expect(runner.Run()).To(MatchError(ContainSubstring("Config dir does not exist")))

				_, err := os.Stat(runner.PIDFile)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})

package consul_test

import (
	"acceptance-tests/testing/consul"
	"os"
	"os/exec"
	"syscall"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Agent", func() {
	var agent *consul.Agent

	BeforeEach(func() {
		agent = consul.NewAgent(consul.AgentOptions{
			DataDir:   "/tmp/client",
			RetryJoin: []string{"127.0.0.1:8301"},
		})
	})

	AfterEach(func() {
		if agent.Process == nil {
			return
		}

		agent.Process.Kill()
	})

	Describe("Start", func() {
		It("starts a long-running agent process", func() {
			err := agent.Start()
			Expect(err).NotTo(HaveOccurred())
			Expect(agent.Process).NotTo(BeNil())

			pid := agent.Process.Pid
			Expect(pid).NotTo(Equal(0))

			p, err := os.FindProcess(pid)
			Expect(err).NotTo(HaveOccurred())

			err = p.Signal(syscall.Signal(0))
			Expect(err).NotTo(HaveOccurred())
		})

		It("uses the configuration options in the command", func() {
			agent = consul.NewAgent(consul.AgentOptions{
				DataDir:   "/tmp/some-data-dir",
				RetryJoin: []string{"127.0.0.1:5555", "192.168.0.5:1111"},
			})

			err := agent.Start()
			Expect(err).NotTo(HaveOccurred())
			Expect(agent.Args).To(ConsistOf([]string{
				"consul",
				"agent",
				"-node", "localnode",
				"-bind", "127.0.0.1",
				"-data-dir", "/tmp/some-data-dir",
				"-retry-join", "127.0.0.1:5555",
				"-retry-join", "192.168.0.5:1111",
			}))
		})

		Context("when an agent is already running", func() {
			var pid int

			AfterEach(func() {
				p, err := os.FindProcess(pid)
				Expect(err).NotTo(HaveOccurred())

				err = p.Kill()
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an error", func() {
				err := agent.Start()
				Expect(err).NotTo(HaveOccurred())

				By("guaranteeing the consul process has started", func() {
					Eventually(func() error {
						return exec.Command("consul", "members").Run()
					}, "10s", "1s").Should(Succeed())
					pid = agent.Process.Pid
				})

				Eventually(agent.Start, "10s", "1s").Should(MatchError("consul agent is already running"))
			})
		})
	})

	Describe("Stop", func() {
		Context("when the agent is running", func() {
			BeforeEach(func() {
				err := agent.Start()
				Expect(err).NotTo(HaveOccurred())
				Expect(agent.Process).NotTo(BeNil())
			})

			It("stops the agent process", func() {
				pid := agent.Process.Pid

				err := agent.Stop()
				Expect(err).NotTo(HaveOccurred())

				p, err := os.FindProcess(pid)
				Expect(err).NotTo(HaveOccurred())

				err = p.Signal(syscall.Signal(0))
				Expect(err).To(MatchError(ContainSubstring("already finished")))
			})

			Context("failure cases", func() {
				BeforeEach(func() {
					Expect(agent.Stop()).To(Succeed())
				})

				It("errors on a bogus process", func() {
					agent.Cmd.Process = &os.Process{Pid: -1}

					err := agent.Stop()
					Expect(err).To(MatchError("os: process already released"))
				})
			})
		})

		Context("when the agent is not running", func() {
			It("returns successfully", func() {
				Expect(agent.Stop()).To(Succeed())
				Expect(agent.Process).To(BeNil())
			})
		})
	})
})

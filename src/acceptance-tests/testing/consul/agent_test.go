package consul_test

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/consul"
	"github.com/pivotal-cf-experimental/destiny"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Agent", func() {
	var (
		agent     *consul.Agent
		configDir string
	)

	BeforeEach(func() {
		var err error
		configDir, err = ioutil.TempDir("", "consul-agent")
		Expect(err).NotTo(HaveOccurred())

		agent = consul.NewAgent(consul.AgentOptions{
			ConfigDir:  configDir,
			DataDir:    "/tmp/client",
			Domain:     "cf.internal",
			Key:        destiny.AgentKey,
			Cert:       destiny.AgentCert,
			CACert:     destiny.CACert,
			Encrypt:    destiny.Encrypt,
			ServerName: "consul agent",
			RetryJoin:  []string{"127.0.0.1:8301"},
		})
	})

	AfterEach(func() {
		err := os.Chmod(configDir, os.ModePerm)
		Expect(err).NotTo(HaveOccurred())

		err = os.RemoveAll(configDir)
		Expect(err).NotTo(HaveOccurred())

		consul.ResetCreateFile()

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
				ConfigDir:  configDir,
				DataDir:    "/tmp/some-data-dir",
				Domain:     "cf.internal",
				Key:        "some-key",
				Cert:       "some-cert",
				CACert:     "some-ca-cert",
				Encrypt:    "some-encrypt",
				ServerName: "some-server-name",
				RetryJoin:  []string{"127.0.0.1:5555", "192.168.0.5:1111"},
			})

			err := agent.Start()
			Expect(err).NotTo(HaveOccurred())
			Expect(agent.Args).To(ConsistOf([]string{
				"consul",
				"agent",
				"-node", "localnode",
				"-bind", "127.0.0.1",
				"-data-dir", "/tmp/some-data-dir",
				"-config-dir", configDir,
				"-retry-join", "127.0.0.1:5555",
				"-retry-join", "192.168.0.5:1111",
			}))

			configFile, err := os.Open(filepath.Join(configDir, "config.json"))
			Expect(err).NotTo(HaveOccurred())

			var config map[string]interface{}
			Expect(json.NewDecoder(configFile).Decode(&config)).To(Succeed())

			Expect(config).To(Equal(map[string]interface{}{
				"ca_file":                filepath.Join(configDir, "ca.cert"),
				"cert_file":              filepath.Join(configDir, "agent.cert"),
				"key_file":               filepath.Join(configDir, "agent.key"),
				"encrypt":                "some-encrypt",
				"server_name":            "some-server-name",
				"domain":                 "cf.internal",
				"verify_incoming":        true,
				"verify_outgoing":        true,
				"verify_server_hostname": true,
			}))

			keyFile, err := ioutil.ReadFile(filepath.Join(configDir, "agent.key"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(keyFile)).To(Equal("some-key"))

			certFile, err := ioutil.ReadFile(filepath.Join(configDir, "agent.cert"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(certFile)).To(Equal("some-cert"))

			caFile, err := ioutil.ReadFile(filepath.Join(configDir, "ca.cert"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(caFile)).To(Equal("some-ca-cert"))
		})

		It("it optionally accepts a config dir", func() {
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

		Context("failure cases", func() {
			badCreate := func(filename string) (*os.File, error) {
				file, err := os.Create(filename)
				if err != nil {
					return file, err
				}

				if err := file.Close(); err != nil {
					return file, err
				}

				return file, nil
			}

			It("errors when the config dir cannot be created", func() {
				err := os.Chmod(configDir, 0000)
				Expect(err).NotTo(HaveOccurred())
				agent = consul.NewAgent(consul.AgentOptions{
					ConfigDir: filepath.Join(configDir, "something"),
				})

				err = agent.Start()
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})

			It("errors when the config dir cannot be written to", func() {
				err := os.Chmod(configDir, 0000)
				Expect(err).NotTo(HaveOccurred())
				agent = consul.NewAgent(consul.AgentOptions{
					ConfigDir: configDir,
				})

				err = agent.Start()
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})

			It("errors when the config file cannot be written to", func() {
				agent = consul.NewAgent(consul.AgentOptions{
					ConfigDir: configDir,
				})

				consul.SetCreateFile(badCreate)

				err := agent.Start()
				Expect(err).To(MatchError(ContainSubstring("bad file descriptor")))
			})

			It("errors when the any of the cert files cannot be created", func() {
				agent = consul.NewAgent(consul.AgentOptions{
					ConfigDir: configDir,
				})

				consul.SetCreateFile(func(filename string) (*os.File, error) {
					if strings.Contains(filename, "agent.cert") {
						return nil, errors.New("permission denied")
					}
					return os.Create(filename)
				})

				err := agent.Start()
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})

			It("errors when any of the cert files cannot be written to", func() {
				agent = consul.NewAgent(consul.AgentOptions{
					ConfigDir: configDir,
				})

				consul.SetCreateFile(func(filename string) (*os.File, error) {
					if strings.Contains(filename, "agent.cert") {
						return badCreate(filename)
					}
					return os.Create(filename)
				})

				err := agent.Start()
				Expect(err).To(MatchError(ContainSubstring("bad file descriptor")))
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

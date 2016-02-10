package confab_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudfoundry-incubator/consul-release/src/confab"
	"github.com/cloudfoundry-incubator/consul-release/src/confab/config"
	"github.com/cloudfoundry-incubator/consul-release/src/confab/fakes"
	"github.com/pivotal-golang/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf-experimental/gomegamatchers"
)

var _ = Describe("Controller", func() {
	var (
		clock          *fakes.Clock
		agentRunner    *fakes.AgentRunner
		agentClient    *fakes.AgentClient
		logger         *fakes.Logger
		serviceDefiner *fakes.ServiceDefiner
		controller     confab.Controller
	)

	BeforeEach(func() {
		clock = &fakes.Clock{}
		logger = &fakes.Logger{}

		agentClient = &fakes.AgentClient{}
		agentClient.VerifyJoinedCalls.Returns.Errors = []error{nil}
		agentClient.VerifySyncedCalls.Returns.Errors = []error{nil}

		agentRunner = &fakes.AgentRunner{}
		agentRunner.RunCalls.Returns.Errors = []error{nil}

		serviceDefiner = &fakes.ServiceDefiner{}

		confabConfig := config.Default()
		confabConfig.Node = config.ConfigNode{Name: "node", Index: 0}

		controller = confab.Controller{
			AgentClient:    agentClient,
			AgentRunner:    agentRunner,
			SyncRetryDelay: 10 * time.Millisecond,
			SyncRetryClock: clock,
			EncryptKeys:    []string{"key 1", "key 2", "key 3"},
			Logger:         logger,
			ConfigDir:      "/tmp/config",
			ServiceDefiner: serviceDefiner,
			Config:         confabConfig,
		}
	})

	Describe("WriteConsulConfig", func() {
		var configDir string

		BeforeEach(func() {
			var err error
			configDir, err = ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())

			controller.Config.Path.ConsulConfigDir = configDir
		})

		It("writes a config file to the consul_config dir", func() {
			err := controller.WriteConsulConfig()
			Expect(err).NotTo(HaveOccurred())

			buf, err := ioutil.ReadFile(filepath.Join(configDir, "config.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(buf).To(MatchJSON(`{
				"server": false,
				"domain": "cf.internal",
				"datacenter": "",
				"data_dir": "/var/vcap/store/consul_agent",
				"log_level": "",
				"node_name": "node-0",
				"ports": {
					"dns": 53
				},
				"rejoin_after_leave": true,
				"retry_join": [],
				"bind_addr": "",
				"disable_remote_exec": true,
				"disable_update_check": true,
				"protocol": 0,
				"verify_outgoing": true,
				"verify_incoming": true,
				"verify_server_hostname": true,
				"ca_file": "/var/vcap/jobs/consul_agent/config/certs/ca.crt",
				"key_file": "/var/vcap/jobs/consul_agent/config/certs/agent.key",
				"cert_file": "/var/vcap/jobs/consul_agent/config/certs/agent.crt"
			}`))

			Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
				{
					Action: "controller.write-consul-config.generate-configuration",
				},
				{
					Action: "controller.write-consul-config.write-configuration",
					Data: []lager.Data{{
						"config": config.GenerateConfiguration(controller.Config),
					}},
				},
				{
					Action: "controller.write-consul-config.success",
				},
			}))
		})

		Context("failure cases", func() {
			It("returns an error when the config file can't be written to", func() {
				err := os.Chmod(configDir, 0000)
				Expect(err).NotTo(HaveOccurred())

				err = controller.WriteConsulConfig()
				Expect(err).To(MatchError(ContainSubstring("permission denied")))

				Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "controller.write-consul-config.generate-configuration",
					},
					{
						Action: "controller.write-consul-config.write-configuration",
						Data: []lager.Data{{
							"config": config.GenerateConfiguration(controller.Config),
						}},
					},
					{
						Action: "controller.write-consul-config.write-configuration.failed",
						Error:  fmt.Errorf("open %s: permission denied", filepath.Join(configDir, "config.json")),
					},
				}))
			})
		})
	})

	Describe("ConfigureClient", func() {
		It("writes the pid file", func() {
			err := controller.ConfigureClient()
			Expect(err).NotTo(HaveOccurred())

			Expect(agentRunner.WritePIDCall.CallCount).To(Equal(1))
		})

		Context("failure cases", func() {
			It("returns an error when the pid file can not be written", func() {
				agentRunner.WritePIDCall.Returns.Error = errors.New("something bad happened")

				err := controller.ConfigureClient()
				Expect(err).To(MatchError("something bad happened"))
			})
		})
	})

	Describe("WriteServiceDefinitions", func() {
		It("delegates to the service definer", func() {
			definitions := []config.ServiceDefinition{{
				Name: "banana",
			}}
			serviceDefiner.GenerateDefinitionsCall.Returns.Definitions = definitions

			Expect(controller.WriteServiceDefinitions()).To(Succeed())
			Expect(serviceDefiner.GenerateDefinitionsCall.Receives.Config).To(Equal(controller.Config))
			Expect(serviceDefiner.WriteDefinitionsCall.Receives.ConfigDir).To(Equal("/tmp/config"))
			Expect(serviceDefiner.WriteDefinitionsCall.Receives.Definitions).To(Equal(definitions))

			Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
				{
					Action: "controller.write-service-definitions.generate-definitions",
				},
				{
					Action: "controller.write-service-definitions.write",
				},
				{
					Action: "controller.write-service-definitions.success",
				},
			}))
		})

		Context("when there is an error", func() {
			It("returns the error", func() {
				serviceDefiner.WriteDefinitionsCall.Returns.Error = errors.New("write definitions error")

				err := controller.WriteServiceDefinitions()
				Expect(err).To(MatchError(errors.New("write definitions error")))

				Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "controller.write-service-definitions.generate-definitions",
					},
					{
						Action: "controller.write-service-definitions.write",
					},
					{
						Action: "controller.write-service-definitions.write.failed",
						Error:  errors.New("write definitions error"),
					},
				}))
			})
		})
	})

	Describe("BootAgent", func() {
		It("launches the consul agent and confirms that it joined the cluster", func() {
			Expect(controller.BootAgent(confab.NewTimeout(make(chan time.Time)))).To(Succeed())
			Expect(agentRunner.RunCalls.CallCount).To(Equal(1))
			Expect(agentClient.VerifyJoinedCalls.CallCount).To(Equal(1))
			Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
				{
					Action: "controller.boot-agent.run",
				},
				{
					Action: "controller.boot-agent.verify-joined",
				},
				{
					Action: "controller.boot-agent.success",
				},
			}))
		})

		Context("when starting the agent fails", func() {
			It("immediately returns an error", func() {
				agentRunner.RunCalls.Returns.Errors = []error{errors.New("some error")}

				Expect(controller.BootAgent(confab.NewTimeout(make(chan time.Time)))).To(MatchError("some error"))
				Expect(agentRunner.RunCalls.CallCount).To(Equal(1))
				Expect(agentClient.VerifyJoinedCalls.CallCount).To(Equal(0))
				Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "controller.boot-agent.run",
					},
					{
						Action: "controller.boot-agent.run.failed",
						Error:  errors.New("some error"),
					},
				}))
			})
		})

		Context("joining fails at first but later succeeds", func() {
			It("retries until it joins", func() {
				agentClient.VerifyJoinedCalls.Returns.Errors = make([]error, 10)
				for i := 0; i < 9; i++ {
					agentClient.VerifyJoinedCalls.Returns.Errors[i] = errors.New("some error")
				}

				Expect(controller.BootAgent(confab.NewTimeout(make(chan time.Time)))).To(Succeed())
				Expect(agentClient.VerifyJoinedCalls.CallCount).To(Equal(10))
				Expect(clock.SleepCall.CallCount).To(Equal(9))
				Expect(clock.SleepCall.Receives.Duration).To(Equal(10 * time.Millisecond))
				Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "controller.boot-agent.run",
					},
					{
						Action: "controller.boot-agent.verify-joined",
					},
					{
						Action: "controller.boot-agent.success",
					},
				}))
			})
		})

		Context("joining never succeeds within timeout period", func() {
			It("immediately returns an error", func() {
				agentClient.VerifyJoinedCalls.Returns.Errors = make([]error, 10)
				for i := 0; i < 9; i++ {
					agentClient.VerifyJoinedCalls.Returns.Errors[i] = errors.New("some error")
				}

				timer := make(chan time.Time)
				timeout := confab.NewTimeout(timer)
				timer <- time.Now()

				err := controller.BootAgent(timeout)

				Expect(err).To(MatchError("timeout exceeded"))
				Expect(agentClient.VerifyJoinedCalls.CallCount).To(Equal(0))
				Expect(agentClient.VerifySyncedCalls.CallCount).To(Equal(0))

				Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "controller.boot-agent.run",
					},
					{
						Action: "controller.boot-agent.verify-joined",
					},
					{
						Action: "controller.boot-agent.verify-joined.failed",
						Error:  errors.New("timeout exceeded"),
					},
				}))
			})
		})
	})

	Describe("StopAgent", func() {
		It("tells client to leave the cluster and waits for the agent to stop", func() {
			controller.StopAgent()
			Expect(agentClient.LeaveCall.CallCount).To(Equal(1))
			Expect(agentRunner.WaitCall.CallCount).To(Equal(1))
			Expect(agentRunner.CleanupCall.CallCount).To(Equal(1))
			Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
				{
					Action: "controller.stop-agent.leave",
				},
				{
					Action: "controller.stop-agent.wait",
				},
				{
					Action: "controller.stop-agent.cleanup",
				},
				{
					Action: "controller.stop-agent.success",
				},
			}))
		})

		Context("when the agent client Leave() returns an error", func() {
			BeforeEach(func() {
				agentClient.LeaveCall.Returns.Error = errors.New("leave error")
			})

			It("tells the runner to stop the agent", func() {
				controller.StopAgent()
				Expect(agentRunner.StopCall.CallCount).To(Equal(1))
				Expect(agentRunner.WaitCall.CallCount).To(Equal(1))
				Expect(agentRunner.CleanupCall.CallCount).To(Equal(1))
				Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "controller.stop-agent.leave",
					},
					{
						Action: "controller.stop-agent.leave.failed",
						Error:  errors.New("leave error"),
					},
					{
						Action: "controller.stop-agent.stop",
					},
					{
						Action: "controller.stop-agent.wait",
					},
					{
						Action: "controller.stop-agent.cleanup",
					},
					{
						Action: "controller.stop-agent.success",
					},
				}))
			})

			Context("when agent runner Stop() returns an error", func() {
				BeforeEach(func() {
					agentRunner.StopCall.Returns.Error = errors.New("stop error")
				})

				It("logs the error", func() {
					controller.StopAgent()
					Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
						{
							Action: "controller.stop-agent.leave",
						},
						{
							Action: "controller.stop-agent.leave.failed",
							Error:  errors.New("leave error"),
						},
						{
							Action: "controller.stop-agent.stop",
						},
						{
							Action: "controller.stop-agent.stop.failed",
							Error:  errors.New("stop error"),
						},
						{
							Action: "controller.stop-agent.wait",
						},
						{
							Action: "controller.stop-agent.cleanup",
						},
						{
							Action: "controller.stop-agent.success",
						},
					}))
				})
			})
		})

		Context("when agent runner Wait() returns an error", func() {
			BeforeEach(func() {
				agentRunner.WaitCall.Returns.Error = errors.New("wait error")
			})

			It("logs the error", func() {
				controller.StopAgent()
				Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "controller.stop-agent.leave",
					},
					{
						Action: "controller.stop-agent.wait",
					},
					{
						Action: "controller.stop-agent.wait.failed",
						Error:  errors.New("wait error"),
					},
					{
						Action: "controller.stop-agent.cleanup",
					},
					{
						Action: "controller.stop-agent.success",
					},
				}))
			})
		})

		Context("when agent runner Cleanup() returns an error", func() {
			BeforeEach(func() {
				agentRunner.CleanupCall.Returns.Error = errors.New("cleanup error")
			})

			It("logs the error", func() {
				controller.StopAgent()
				Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "controller.stop-agent.leave",
					},
					{
						Action: "controller.stop-agent.wait",
					},
					{
						Action: "controller.stop-agent.cleanup",
					},
					{
						Action: "controller.stop-agent.cleanup.failed",
						Error:  errors.New("cleanup error"),
					},
					{
						Action: "controller.stop-agent.success",
					},
				}))
			})
		})
	})

	Describe("ConfigureServer", func() {
		Context("when it is not the last node in the cluster", func() {
			It("does not check that it is synced", func() {
				Expect(controller.ConfigureServer(confab.NewTimeout(make(chan time.Time)))).To(Succeed())
				Expect(agentClient.VerifySyncedCalls.CallCount).To(Equal(0))
				Expect(agentRunner.WritePIDCall.CallCount).To(Equal(1))
				Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "controller.configure-server.is-last-node",
					},
					{
						Action: "controller.configure-server.set-keys",
						Data: []lager.Data{{
							"keys": []string{"key 1", "key 2", "key 3"},
						}},
					},
					{
						Action: "controller.configure-server.success",
					},
				}))
			})
		})

		Context("setting keys", func() {
			It("sets the encryption keys used by the agent", func() {
				Expect(controller.ConfigureServer(confab.NewTimeout(make(chan time.Time)))).To(Succeed())
				Expect(agentClient.SetKeysCall.Receives.Keys).To(Equal([]string{
					"key 1",
					"key 2",
					"key 3",
				}))
				Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "controller.configure-server.is-last-node",
					},
					{
						Action: "controller.configure-server.set-keys",
						Data: []lager.Data{{
							"keys": []string{"key 1", "key 2", "key 3"},
						}},
					},
					{
						Action: "controller.configure-server.success",
					},
				}))
			})

			Context("when setting keys errors", func() {
				It("returns the error", func() {
					agentClient.SetKeysCall.Returns.Error = errors.New("oh noes")
					Expect(controller.ConfigureServer(confab.NewTimeout(make(chan time.Time)))).To(MatchError("oh noes"))
					Expect(agentClient.SetKeysCall.Receives.Keys).To(Equal([]string{
						"key 1",
						"key 2",
						"key 3",
					}))
					Expect(agentRunner.WritePIDCall.CallCount).To(Equal(0))
					Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
						{
							Action: "controller.configure-server.is-last-node",
						},
						{
							Action: "controller.configure-server.set-keys",
							Data: []lager.Data{{
								"keys": []string{"key 1", "key 2", "key 3"},
							}},
						},
						{
							Action: "controller.configure-server.set-keys.failed",
							Error:  errors.New("oh noes"),
							Data: []lager.Data{{
								"keys": []string{"key 1", "key 2", "key 3"},
							}},
						},
					}))
				})
			})

			Context("when the server does not have ssl enabled", func() {
				BeforeEach(func() {
					controller.Config.Consul.RequireSSL = false
				})

				It("does not set keys", func() {
					Expect(controller.ConfigureServer(confab.NewTimeout(make(chan time.Time)))).To(Succeed())
					Expect(agentClient.SetKeysCall.Receives.Keys).To(BeNil())
					Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
						{
							Action: "controller.configure-server.is-last-node",
						},
						{
							Action: "controller.configure-server.success",
						},
					}))
				})
			})

			Context("when ssl is enabled but no keys are provided", func() {
				BeforeEach(func() {
					controller.EncryptKeys = []string{}
				})

				It("returns an error", func() {
					Expect(controller.ConfigureServer(confab.NewTimeout(make(chan time.Time)))).To(MatchError("encrypt keys cannot be empty if ssl is enabled"))
					Expect(agentClient.SetKeysCall.Receives.Keys).To(BeNil())
					Expect(agentRunner.WritePIDCall.CallCount).To(Equal(0))

					Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
						{
							Action: "controller.configure-server.is-last-node",
						},
						{
							Action: "controller.configure-server.no-encrypt-keys",
							Error:  errors.New("encrypt keys cannot be empty if ssl is enabled"),
						},
					}))
				})
			})
		})

		Context("when it is the last node in the cluster", func() {
			BeforeEach(func() {
				agentClient.IsLastNodeCall.Returns.IsLastNode = true
			})

			It("checks that it is synced", func() {
				Expect(controller.ConfigureServer(confab.NewTimeout(make(chan time.Time)))).To(Succeed())
				Expect(agentClient.VerifySyncedCalls.CallCount).To(Equal(1))
				Expect(agentRunner.WritePIDCall.CallCount).To(Equal(1))

				Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "controller.configure-server.is-last-node",
					},
					{
						Action: "controller.configure-server.verify-synced",
					},
					{
						Action: "controller.configure-server.set-keys",
						Data: []lager.Data{{
							"keys": []string{"key 1", "key 2", "key 3"},
						}},
					},
					{
						Action: "controller.configure-server.success",
					},
				}))
			})

			Context("verifying sync fails at first but later succeeds", func() {
				It("retries until it verifies sync successfully", func() {
					agentClient.VerifySyncedCalls.Returns.Errors = make([]error, 10)
					for i := 0; i < 9; i++ {
						agentClient.VerifySyncedCalls.Returns.Errors[i] = errors.New("some error")
					}

					Expect(controller.ConfigureServer(confab.NewTimeout(make(chan time.Time)))).To(Succeed())
					Expect(agentClient.VerifySyncedCalls.CallCount).To(Equal(10))
					Expect(clock.SleepCall.CallCount).To(Equal(9))
					Expect(clock.SleepCall.Receives.Duration).To(Equal(10 * time.Millisecond))
					Expect(agentRunner.WritePIDCall.CallCount).To(Equal(1))

					Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
						{
							Action: "controller.configure-server.is-last-node",
						},
						{
							Action: "controller.configure-server.verify-synced",
						},
						{
							Action: "controller.configure-server.set-keys",
							Data: []lager.Data{{
								"keys": []string{"key 1", "key 2", "key 3"},
							}},
						},
						{
							Action: "controller.configure-server.success",
						},
					}))
				})
			})

			Context("verifying synced never succeeds within the timeout period", func() {
				It("immediately returns an error", func() {
					agentClient.VerifySyncedCalls.Returns.Errors = make([]error, 10)
					for i := 0; i < 9; i++ {
						agentClient.VerifySyncedCalls.Returns.Errors[i] = errors.New("some error")
					}

					timer := make(chan time.Time)
					timeout := confab.NewTimeout(timer)
					timer <- time.Now()

					err := controller.ConfigureServer(timeout)
					Expect(err).To(MatchError("timeout exceeded"))
					Expect(agentClient.VerifySyncedCalls.CallCount).To(Equal(0))
					Expect(agentClient.SetKeysCall.Receives.Keys).To(BeNil())
					Expect(agentRunner.WritePIDCall.CallCount).To(Equal(0))

					Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
						{
							Action: "controller.configure-server.is-last-node",
						},
						{
							Action: "controller.configure-server.verify-synced",
						},
						{
							Action: "controller.configure-server.verify-synced.failed",
							Error:  errors.New("timeout exceeded"),
						},
					}))
				})
			})

			Context("error while checking if it is the last node", func() {
				It("immediately returns the error", func() {
					agentClient.IsLastNodeCall.Returns.Error = errors.New("some error")
					Expect(controller.ConfigureServer(confab.NewTimeout(make(chan time.Time)))).To(MatchError("some error"))
					Expect(agentClient.VerifySyncedCalls.CallCount).To(Equal(0))
					Expect(agentClient.SetKeysCall.Receives.Keys).To(BeNil())
					Expect(agentRunner.WritePIDCall.CallCount).To(Equal(0))
					Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
						{
							Action: "controller.configure-server.is-last-node",
						},
						{
							Action: "controller.configure-server.is-last-node.failed",
							Error:  errors.New("some error"),
						},
					}))
				})
			})
		})

		Context("when writing the PID file fails", func() {
			It("returns the error", func() {
				agentRunner.WritePIDCall.Returns.Error = errors.New("failed to write PIDFILE")

				controller.Config.Consul.RequireSSL = false
				err := controller.ConfigureServer(confab.NewTimeout(make(chan time.Time)))
				Expect(err).To(MatchError("failed to write PIDFILE"))

				Expect(agentRunner.WritePIDCall.CallCount).To(Equal(1))
				Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "controller.configure-server.is-last-node",
					},
					{
						Action: "controller.configure-server.write-pid.failed",
						Error:  errors.New("failed to write PIDFILE"),
					},
				}))
			})
		})
	})
})

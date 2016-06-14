package config_test

import (
	"github.com/cloudfoundry-incubator/consul-release/src/confab/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ConsulConfigDefiner", func() {
	Describe("GenerateConfiguration", func() {
		var consulConfig config.ConsulConfig
		var configDir string

		BeforeEach(func() {
			configDir = "/var/vcap/jobs/consul_agent/config"
			consulConfig = config.GenerateConfiguration(config.Config{}, configDir)
		})

		Describe("datacenter", func() {
			It("defaults to empty string", func() {
				Expect(consulConfig.Datacenter).To(Equal(""))
			})

			Context("when the `consul.agent.datacenter` property is set", func() {
				It("uses that value", func() {
					consulConfig = config.GenerateConfiguration(config.Config{
						Consul: config.ConfigConsul{
							Agent: config.ConfigConsulAgent{
								Datacenter: "my-datacenter",
							},
						},
					}, configDir)
					Expect(consulConfig.Datacenter).To(Equal("my-datacenter"))
				})
			})
		})

		Describe("domain", func() {
			It("it gets the domain suffix from the config", func() {
				config := config.GenerateConfiguration(config.Config{
					Consul: config.ConfigConsul{
						Agent: config.ConfigConsulAgent{
							Domain: "some-domain",
						},
					},
				}, configDir)

				Expect(config.Domain).To(Equal("some-domain"))
			})
		})

		Describe("data_dir", func() {
			It("defaults to `/var/vcap/store/consul_agent`", func() {
				Expect(consulConfig.DataDir).To(Equal("/var/vcap/store/consul_agent"))
			})
		})

		Describe("log_level", func() {
			It("defaults to empty string", func() {
				Expect(consulConfig.LogLevel).To(Equal(""))
			})

			Context("when the `consul.agent.log_level` property is set", func() {
				It("uses that value", func() {
					consulConfig = config.GenerateConfiguration(config.Config{
						Consul: config.ConfigConsul{
							Agent: config.ConfigConsulAgent{
								LogLevel: "some-log-level",
							},
						},
					}, configDir)
					Expect(consulConfig.LogLevel).To(Equal("some-log-level"))
				})
			})
		})

		Describe("node_name", func() {
			It("uses the job name and index as the value", func() {
				consulConfig = config.GenerateConfiguration(config.Config{
					Node: config.ConfigNode{
						Name:  "node_name",
						Index: 0,
					},
				}, configDir)
				Expect(consulConfig.NodeName).To(Equal("node-name-0"))
			})
		})

		Describe("server", func() {
			It("defaults to false", func() {
				Expect(consulConfig.Server).To(BeFalse())
			})

			Context("when the `consul.agent.mode property` is `server`", func() {
				It("sets the value to true", func() {
					consulConfig = config.GenerateConfiguration(config.Config{
						Consul: config.ConfigConsul{
							Agent: config.ConfigConsulAgent{
								Mode: "server",
							},
						},
					}, configDir)
					Expect(consulConfig.Server).To(BeTrue())
				})
			})

			Context("when the `consul.agent.mode` property is not `server`", func() {
				It("sets the value to false", func() {
					consulConfig = config.GenerateConfiguration(config.Config{
						Consul: config.ConfigConsul{
							Agent: config.ConfigConsulAgent{
								Mode: "banana",
							},
						},
					}, configDir)
					Expect(consulConfig.Server).To(BeFalse())
				})
			})
		})

		Describe("ports", func() {
			It("defaults to a struct containing port 53 for DNS", func() {
				Expect(consulConfig.Ports).To(Equal(config.ConsulConfigPorts{
					DNS: 53,
				}))
			})
		})

		Describe("rejoin_after_leave", func() {
			It("defaults to true", func() {
				Expect(consulConfig.RejoinAfterLeave).To(BeTrue())
			})
		})

		Describe("retry_join", func() {
			It("defaults to an empty slice of strings", func() {
				Expect(consulConfig.RetryJoin).To(Equal([]string{}))
			})

			Context("when `consul.agent.servers.lan` has a list of servers", func() {
				It("uses those values", func() {
					consulConfig = config.GenerateConfiguration(config.Config{
						Consul: config.ConfigConsul{
							Agent: config.ConfigConsulAgent{
								Servers: config.ConfigConsulAgentServers{
									LAN: []string{
										"first-server",
										"second-server",
										"third-server",
									},
								},
							},
						},
					}, configDir)
					Expect(consulConfig.RetryJoin).To(Equal([]string{
						"first-server",
						"second-server",
						"third-server",
					}))
				})
			})
		})

		Describe("retry_join_wan", func() {
			It("defaults to an empty slice of strings", func() {
				Expect(consulConfig.RetryJoinWAN).To(Equal([]string{}))
			})

			Context("when `consul.agent.servers.wan` has a list of servers", func() {
				It("uses those values", func() {
					consulConfig = config.GenerateConfiguration(config.Config{
						Consul: config.ConfigConsul{
							Agent: config.ConfigConsulAgent{
								Servers: config.ConfigConsulAgentServers{
									WAN: []string{
										"first-wan-server",
										"second-wan-server",
										"third-wan-server",
									},
								},
							},
						},
					}, configDir)
					Expect(consulConfig.RetryJoinWAN).To(Equal([]string{
						"first-wan-server",
						"second-wan-server",
						"third-wan-server",
					}))
				})
			})
		})

		Describe("bind_addr", func() {
			It("defaults to an empty string", func() {
				Expect(consulConfig.BindAddr).To(Equal(""))
			})

			Context("when `node.external_ip` is provided", func() {
				It("uses those values", func() {
					consulConfig = config.GenerateConfiguration(config.Config{
						Node: config.ConfigNode{
							ExternalIP: "0.0.0.0",
						},
					}, configDir)
					Expect(consulConfig.BindAddr).To(Equal("0.0.0.0"))
				})
			})
		})

		Describe("disable_remote_exec", func() {
			It("defaults to true", func() {
				Expect(consulConfig.DisableRemoteExec).To(BeTrue())
			})
		})

		Describe("disable_update_check", func() {
			It("defaults to true", func() {
				Expect(consulConfig.DisableUpdateCheck).To(BeTrue())
			})
		})

		Describe("protocol", func() {
			It("defaults to 0", func() {
				Expect(consulConfig.Protocol).To(Equal(0))
			})

			Context("when `consul.agent.protocol_version` is specified", func() {
				It("uses that value", func() {
					consulConfig = config.GenerateConfiguration(config.Config{
						Consul: config.ConfigConsul{
							Agent: config.ConfigConsulAgent{
								ProtocolVersion: 21,
							},
						},
					}, configDir)
					Expect(consulConfig.Protocol).To(Equal(21))
				})
			})
		})

		Describe("verify_outgoing", func() {
			It("is true", func() {
				consulConfig = config.GenerateConfiguration(config.Config{}, configDir)
				Expect(consulConfig.VerifyOutgoing).NotTo(BeNil())
				Expect(*consulConfig.VerifyOutgoing).To(BeTrue())
			})
		})

		Describe("verify_incoming", func() {
			It("is true", func() {
				consulConfig = config.GenerateConfiguration(config.Config{}, configDir)
				Expect(consulConfig.VerifyIncoming).NotTo(BeNil())
				Expect(*consulConfig.VerifyIncoming).To(BeTrue())
			})
		})

		Describe("verify_server_hostname", func() {
			It("is true", func() {
				consulConfig = config.GenerateConfiguration(config.Config{}, configDir)
				Expect(consulConfig.VerifyServerHostname).NotTo(BeNil())
				Expect(*consulConfig.VerifyServerHostname).To(BeTrue())
			})
		})

		Describe("ca_file", func() {
			It("is the location of the ca file", func() {
				consulConfig = config.GenerateConfiguration(config.Config{}, "/var/vcap/jobs/consul_agent_windows/config")
				Expect(consulConfig.CAFile).NotTo(BeNil())
				Expect(*consulConfig.CAFile).To(Equal("/var/vcap/jobs/consul_agent_windows/config/certs/ca.crt"))
			})
		})

		Describe("key_file", func() {
			Context("when `consul.agent.mode` is `server`", func() {
				It("is the location of the server.key file", func() {
					consulConfig = config.GenerateConfiguration(config.Config{
						Consul: config.ConfigConsul{
							Agent: config.ConfigConsulAgent{
								Mode: "server",
							},
						},
					}, configDir)
					Expect(consulConfig.KeyFile).NotTo(BeNil())
					Expect(*consulConfig.KeyFile).To(Equal("/var/vcap/jobs/consul_agent/config/certs/server.key"))
				})
			})

			Context("when `consul.agent.mode` is not `server`", func() {
				It("is the location of the agent.key file", func() {
					consulConfig = config.GenerateConfiguration(config.Config{}, configDir)
					Expect(consulConfig.KeyFile).NotTo(BeNil())
					Expect(*consulConfig.KeyFile).To(Equal("/var/vcap/jobs/consul_agent/config/certs/agent.key"))
				})
			})
		})

		Describe("cert_file", func() {
			Context("when `consul.agent.mode` is `server`", func() {
				It("is the location of the server.crt file", func() {
					consulConfig = config.GenerateConfiguration(config.Config{
						Consul: config.ConfigConsul{
							Agent: config.ConfigConsulAgent{
								Mode: "server",
							},
						},
					}, configDir)
					Expect(consulConfig.CertFile).NotTo(BeNil())
					Expect(*consulConfig.CertFile).To(Equal("/var/vcap/jobs/consul_agent/config/certs/server.crt"))
				})
			})

			Context("when `consul.agent.mode` is not `server`", func() {
				It("is the location of the agent.key file", func() {
					consulConfig = config.GenerateConfiguration(config.Config{}, configDir)
					Expect(consulConfig.CertFile).NotTo(BeNil())
					Expect(*consulConfig.CertFile).To(Equal("/var/vcap/jobs/consul_agent/config/certs/agent.crt"))
				})
			})
		})

		Describe("encrypt", func() {
			Context("when `consul.encrypt_keys` is empty", func() {
				It("is nil", func() {
					consulConfig = config.GenerateConfiguration(config.Config{}, configDir)
					Expect(consulConfig.Encrypt).To(BeNil())
				})
			})

			Context("when `consul.encrypt_keys` is provided with keys", func() {
				It("base 64 encodes the key if it is not already encoded", func() {
					consulConfig = config.GenerateConfiguration(
						config.Config{
							Consul: config.ConfigConsul{
								EncryptKeys: []string{"banana"},
							},
						}, configDir)
					Expect(consulConfig.Encrypt).NotTo(BeNil())
					Expect(*consulConfig.Encrypt).To(Equal("enqzXBmgKOy13WIGsmUk+g=="))
				})

				It("leaves the key alone if it is already base 64 encoded", func() {
					consulConfig = config.GenerateConfiguration(
						config.Config{
							Consul: config.ConfigConsul{
								EncryptKeys: []string{"enqzXBmgKOy13WIGsmUk+g=="},
							},
						}, configDir)
					Expect(consulConfig.Encrypt).NotTo(BeNil())
					Expect(*consulConfig.Encrypt).To(Equal("enqzXBmgKOy13WIGsmUk+g=="))
				})
			})
		})

		Describe("bootstrap_expect", func() {
			Context("when `consul.agent.mode` is not `server`", func() {
				It("is nil", func() {
					Expect(consulConfig.BootstrapExpect).To(BeNil())
				})
			})

			Context("when `consul.agent.mode` is `server`", func() {
				It("sets it to the number of servers in the cluster", func() {
					consulConfig = config.GenerateConfiguration(config.Config{
						Consul: config.ConfigConsul{
							Agent: config.ConfigConsulAgent{
								Mode: "server",
								Servers: config.ConfigConsulAgentServers{
									LAN: []string{
										"first-server",
										"second-server",
										"third-server",
									},
								},
							},
						},
					}, configDir)
					Expect(consulConfig.BootstrapExpect).NotTo(BeNil())
					Expect(*consulConfig.BootstrapExpect).To(Equal(3))
				})
			})
		})
	})
})

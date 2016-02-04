package confab_test

import (
	"confab"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Describe("DefaultConfig", func() {
		It("returns a default configuration", func() {
			config := confab.Config{
				Consul: confab.ConfigConsul{
					RequireSSL: true,
					Agent: confab.ConfigConsulAgent{
						Servers: confab.ConfigConsulAgentServers{
							LAN: []string{},
						},
					},
				},
				Path: confab.ConfigPath{
					AgentPath:       "/var/vcap/packages/consul/bin/consul",
					ConsulConfigDir: "/var/vcap/jobs/consul_agent/config",
					PIDFile:         "/var/vcap/sys/run/consul_agent/consul_agent.pid",
				},
				Confab: confab.ConfigConfab{
					TimeoutInSeconds: 55,
				},
			}
			Expect(confab.DefaultConfig()).To(Equal(config))
		})
	})

	Describe("ConfigFromJSON", func() {
		It("returns a config given JSON", func() {
			json := []byte(`{
				"node": {
					"name": "nodename",
					"index": 1234,
					"external_ip": "10.0.0.1"
				},
				"path": {
					"agent_path": "/path/to/agent",
					"consul_config_dir": "/consul/config/dir",
					"pid_file": "/path/to/pidfile"
				},
				"consul": {
					"agent": {
						"services": {
							"myservice": {
								"name" : "myservicename"	
							}
						},
						"mode": "server",
						"datacenter": "dc1",
						"log_level": "debug",
						"protocol_version": 1
					},
					"require_ssl": true,
					"encrypt_keys": ["key-1", "key-2"]
				},
				"confab": {
					"timeout_in_seconds": 30
				}
			}`)

			config, err := confab.ConfigFromJSON(json)
			Expect(err).NotTo(HaveOccurred())
			Expect(config).To(Equal(confab.Config{
				Path: confab.ConfigPath{
					AgentPath:       "/path/to/agent",
					ConsulConfigDir: "/consul/config/dir",
					PIDFile:         "/path/to/pidfile",
				},
				Node: confab.ConfigNode{
					Name:       "nodename",
					Index:      1234,
					ExternalIP: "10.0.0.1",
				},
				Consul: confab.ConfigConsul{
					Agent: confab.ConfigConsulAgent{
						Services: map[string]confab.ServiceDefinition{
							"myservice": confab.ServiceDefinition{
								Name: "myservicename",
							},
						},
						Mode:            "server",
						Datacenter:      "dc1",
						LogLevel:        "debug",
						ProtocolVersion: 1,
						Servers: confab.ConfigConsulAgentServers{
							LAN: []string{},
						},
					},
					RequireSSL:  true,
					EncryptKeys: []string{"key-1", "key-2"},
				},
				Confab: confab.ConfigConfab{
					TimeoutInSeconds: 30,
				},
			}))
		})

		It("returns a config with default values", func() {
			json := []byte(`{}`)
			config, err := confab.ConfigFromJSON(json)
			Expect(err).NotTo(HaveOccurred())
			Expect(config).To(Equal(confab.Config{
				Path: confab.ConfigPath{
					AgentPath:       "/var/vcap/packages/consul/bin/consul",
					ConsulConfigDir: "/var/vcap/jobs/consul_agent/config",
					PIDFile:         "/var/vcap/sys/run/consul_agent/consul_agent.pid",
				},
				Consul: confab.ConfigConsul{
					RequireSSL: true,
					Agent: confab.ConfigConsulAgent{
						Servers: confab.ConfigConsulAgentServers{
							LAN: []string{},
						},
					},
				},
				Confab: confab.ConfigConfab{
					TimeoutInSeconds: 55,
				},
			}))
		})

		It("returns an error on invalid json", func() {
			json := []byte(`{%%%{{}{}{{}{}{{}}}}}}}`)
			_, err := confab.ConfigFromJSON(json)
			Expect(err).To(MatchError(ContainSubstring("invalid character")))
		})
	})
})

package confab_test

import (
	"confab"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ConsulConfigDefiner", func() {
	Describe("GenerateConfiguration", func() {
		FIt("generates a consul config given a confab config", func() {
			confabConfig := confab.Config{
				Node: confab.ConfigNode{
					Name:  "my_node",
					Index: 0,
				},
				Consul: confab.ConfigConsul{
					Agent: confab.ConfigAgent{
						Server:          true,
						Datacenter:      "dc1",
						LogLevel:        "info",
						ProtocolVersion: 2,
					},
				},
			}

			consulConfig := confab.GenerateConfiguration(confabConfig)

			Expect(consulConfig).To(Equal(
				confab.ConsulConfig{
					Server:     true,
					Domain:     "cf.internal",
					Datacenter: "dc1",
					DataDir:    "/var/vcap/store/consul_agent",
					LogLevel:   "info",
					NodeName:   "my-node-0",
					Ports: confab.ConsulConfigPorts{
						DNS: 53,
					},
					RejoinAfterLeave:   true,
					DisableRemoteExec:  true,
					DisableUpdateCheck: true,
					Protocol:           2,
				},
			))
		})

		PDescribe("when the consul agent is a server", func() {
			It("assigns server.key and server.cert encryption files", func() {
			})
		})
		PDescribe("when the consul agent is a client", func() {
			It("assigns agent.key and agent.cert encryption files", func() {
			})
		})
	})

})

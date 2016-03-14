package chaperon_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cloudfoundry-incubator/consul-release/src/confab/chaperon"
	"github.com/cloudfoundry-incubator/consul-release/src/confab/config"
	"github.com/cloudfoundry-incubator/consul-release/src/confab/fakes"
	"github.com/pivotal-golang/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf-experimental/gomegamatchers"
)

var _ = Describe("ConfigWriter", func() {
	var (
		configDir string
		cfg       config.Config
		writer    chaperon.ConfigWriter
		logger    *fakes.Logger
	)

	Describe("Write", func() {

		BeforeEach(func() {
			logger = &fakes.Logger{}

			var err error
			configDir, err = ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())

			cfg = config.Default()
			cfg.Node = config.ConfigNode{Name: "node", Index: 0}
			cfg.Path.ConsulConfigDir = configDir

			writer = chaperon.NewConfigWriter(configDir, logger)
		})

		It("writes a config file to the consul_config dir", func() {
			err := writer.Write(cfg)
			Expect(err).NotTo(HaveOccurred())

			buf, err := ioutil.ReadFile(filepath.Join(configDir, "config.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(buf).To(MatchJSON(`{
				"server": false,
				"domain": "",
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
					Action: "config-writer.write.generate-configuration",
				},
				{
					Action: "config-writer.write.write-file",
					Data: []lager.Data{{
						"config": config.GenerateConfiguration(cfg),
					}},
				},
				{
					Action: "config-writer.write.success",
				},
			}))
		})

		Context("failure cases", func() {
			It("returns an error when the config file can't be written to", func() {
				err := os.Chmod(configDir, 0000)
				Expect(err).NotTo(HaveOccurred())

				err = writer.Write(cfg)
				Expect(err).To(MatchError(ContainSubstring("permission denied")))

				Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "config-writer.write.generate-configuration",
					},
					{
						Action: "config-writer.write.write-file",
						Data: []lager.Data{{
							"config": config.GenerateConfiguration(cfg),
						}},
					},
					{
						Action: "config-writer.write.write-file.failed",
						Error:  fmt.Errorf("open %s: permission denied", filepath.Join(configDir, "config.json")),
					},
				}))
			})
		})
	})
})

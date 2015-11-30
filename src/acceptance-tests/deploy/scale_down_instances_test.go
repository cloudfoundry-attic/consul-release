package deploy_test

import (
	"acceptance-tests/helpers"

	capi "github.com/hashicorp/consul/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Scaling down Instances", func() {
	var (
		consulManifest  *helpers.Manifest
		consulServerIPs []string
		runner          *helpers.AgentRunner
	)

	BeforeEach(func() {
		consulManifest = new(helpers.Manifest)
		consulServerIPs = []string{}

		bosh.GenerateAndSetDeploymentManifest(
			consulManifest,
			consulManifestGeneration,
			directorUUIDStub,
			helpers.InstanceCount3NodesStubPath,
			helpers.PersistentDiskStubPath,
			config.IAASSettingsConsulStubPath,
			helpers.PropertyOverridesStubPath,
			consulNameOverrideStub,
		)

		By("deploying", func() {
			Expect(bosh.Command("-n", "deploy")).To(gexec.Exit(0))
			Expect(len(consulManifest.Properties.Consul.Agent.Servers.Lans)).To(Equal(3))
		})

		By("starting a consul agent", func() {
			for _, elem := range consulManifest.Properties.Consul.Agent.Servers.Lans {
				consulServerIPs = append(consulServerIPs, elem)
			}

			runner = helpers.NewAgentRunner(consulServerIPs, config.BindAddress)
			runner.Start()
		})
	})

	AfterEach(func() {
		By("stopping the consul agent", func() {
			runner.Stop()
		})

		By("deleting the deployment", func() {
			bosh.Command("-n", "delete", "deployment", consulDeployment)
		})
	})

	Describe("scaling from 3 nodes to 1", func() {
		It("succesfully scales to multiple consul nodes", func() {
			consatsKey := "consats-key"
			consatsValue := []byte("consats-value")

			By("setting a persistent value", func() {
				consatsClient := runner.NewClient()
				keyValueClient := consatsClient.KV()

				pair := &capi.KVPair{Key: consatsKey, Value: consatsValue}
				_, err := keyValueClient.Put(pair, nil)
				Expect(err).ToNot(HaveOccurred())

				resultPair, _, err := keyValueClient.Get(consatsKey, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(resultPair.Value).To(Equal(consatsValue))

				bosh.GenerateAndSetDeploymentManifest(
					consulManifest,
					consulManifestGeneration,
					directorUUIDStub,
					helpers.InstanceCount1NodeStubPath,
					helpers.PersistentDiskStubPath,
					config.IAASSettingsConsulStubPath,
					helpers.PropertyOverridesStubPath,
					consulNameOverrideStub,
				)
			})

			By("deploying", func() {
				Expect(bosh.Command("-n", "deploy")).To(gexec.Exit(0))
				Expect(len(consulManifest.Properties.Consul.Agent.Servers.Lans)).To(Equal(1))
			})

			By("reading the value from consul", func() {
				runner.Stop()
				runner = helpers.NewAgentRunner(consulServerIPs, config.BindAddress)
				runner.Start()

				consatsClient := runner.NewClient()
				keyValueClient := consatsClient.KV()

				resultPair, _, err := keyValueClient.Get(consatsKey, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(resultPair).NotTo(BeNil())
				Expect(resultPair.Value).To(Equal(consatsValue))
			})
		})
	})
})

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

		By("deploying")
		Expect(bosh.Command("-n", "deploy")).To(gexec.Exit(0))
		Expect(len(consulManifest.Properties.Consul.Agent.Servers.Lans)).To(Equal(3))

		for _, elem := range consulManifest.Properties.Consul.Agent.Servers.Lans {
			consulServerIPs = append(consulServerIPs, elem)
		}

		runner = helpers.NewAgentRunner(consulServerIPs, config.BindAddress)
		runner.Start()
	})

	AfterEach(func() {
		By("delete deployment")
		runner.Stop()
		bosh.Command("-n", "delete", "deployment", consulDeployment)
	})

	Describe("scaling from 3 nodes to 1", func() {
		It("succesfully scales to multiple consul nodes", func() {
			By("setting a persistent value")
			consatsClient := runner.NewClient()

			consatsKey := "consats-key"
			consatsValue := []byte("consats-value")

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

			By("deploying")
			Expect(bosh.Command("-n", "deploy")).To(gexec.Exit(0))
			Expect(len(consulManifest.Properties.Consul.Agent.Servers.Lans)).To(Equal(1))

			By("reading the value from consul")
			resultPair, _, err = keyValueClient.Get(consatsKey, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(resultPair).NotTo(BeNil())
			Expect(resultPair.Value).To(Equal(consatsValue))
		})
	})
})

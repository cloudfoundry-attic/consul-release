package deploy_test

import (
	"acceptance-tests/helpers"
	"fmt"

	capi "github.com/hashicorp/consul/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Multiple Instance Rolling deploys", func() {
	var (
		consulManifest  *helpers.Manifest
		consulServerIPs []string
		runner          *helpers.AgentRunner
	)

	BeforeEach(func() {
		consulManifest = new(helpers.Manifest)

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

	It("Saves data after a rolling deploy", func() {
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

		// generate new stub that overwrites a property
		consulStub := fmt.Sprintf(`---
property_overrides:
  consul:
    server_cert: something different
    require_ssl: false
`)

		consulRollingDeployStub := helpers.WriteStub(consulStub)

		bosh.GenerateAndSetDeploymentManifest(
			consulManifest,
			consulManifestGeneration,
			directorUUIDStub,
			helpers.InstanceCount3NodesStubPath,
			helpers.PersistentDiskStubPath,
			config.IAASSettingsConsulStubPath,
			consulRollingDeployStub,
			consulNameOverrideStub,
		)

		By("deploying")
		Expect(bosh.Command("-n", "deploy")).To(gexec.Exit(0))

		By("reading the value from consul")
		resultPair, _, err = keyValueClient.Get(consatsKey, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(resultPair.Value).To(Equal(consatsValue))
	})
})

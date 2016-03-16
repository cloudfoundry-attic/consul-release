package deploy_test

import (
	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/consul"
	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/helpers"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Encryption key rotation", func() {
	var (
		manifest  destiny.Manifest
		kv        consul.KV
		testKey   string
		testValue string
	)

	BeforeEach(func() {
		guid, err := helpers.NewGUID()
		Expect(err).NotTo(HaveOccurred())

		testKey = "consul-key-" + guid
		testValue = "consul-value-" + guid

		manifest, kv, err = helpers.DeployConsulWithInstanceCount(3, client, config)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return client.DeploymentVMs(manifest.Name)
		}, "1m", "10s").Should(ConsistOf([]bosh.VM{
			{"running"},
			{"running"},
			{"running"},
			{"running"},
		}))
	})

	AfterEach(func() {
		if !CurrentGinkgoTestDescription().Failed {
			err := client.DeleteDeployment(manifest.Name)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	It("successfully rolls with a new encryption key", func() {
		By("adding a consul client job", func() {
			consulClient := destiny.Job{
				Name:      "consul_client",
				Instances: 1,
				Networks: []destiny.JobNetwork{
					{
						Name:      manifest.Jobs[0].Networks[0].Name,
						StaticIPs: []string{manifest.Networks[0].Subnets[0].Static[3]},
					},
				},
				PersistentDisk: 1024,
				Properties: &destiny.JobProperties{
					Consul: destiny.JobPropertiesConsul{
						Agent: destiny.JobPropertiesConsulAgent{
							Mode: "client",
						},
					},
				},
				ResourcePool: manifest.Jobs[0].ResourcePool,
				Templates:    manifest.Jobs[0].Templates,
				Update:       manifest.Jobs[0].Update,
			}

			manifest.Jobs = append(manifest.Jobs, consulClient)
		})

		By("deploying with the original key", func() {
			yaml, err := manifest.ToYAML()
			Expect(err).NotTo(HaveOccurred())

			yaml, err = client.ResolveManifestVersions(yaml)
			Expect(err).NotTo(HaveOccurred())

			err = client.Deploy(yaml)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs(manifest.Name)
			}, "1m", "10s").Should(ConsistOf([]bosh.VM{
				{"running"},
				{"running"},
				{"running"},
				{"running"},
				{"running"},
			}))
		})

		By("adding a new primary encryption key", func() {
			manifest.Properties.Consul.EncryptKeys = append([]string{"this is some encrypted key"}, manifest.Properties.Consul.EncryptKeys...)
		})

		By("deploying with the new key", func() {
			yaml, err := manifest.ToYAML()
			Expect(err).NotTo(HaveOccurred())

			yaml, err = client.ResolveManifestVersions(yaml)
			Expect(err).NotTo(HaveOccurred())

			err = client.Deploy(yaml)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs(manifest.Name)
			}, "1m", "10s").Should(ConsistOf([]bosh.VM{
				{"running"},
				{"running"},
				{"running"},
				{"running"},
				{"running"},
			}))
		})

		By("setting a persistent value", func() {
			err := kv.Set(testKey, testValue)
			Expect(err).NotTo(HaveOccurred())
		})

		By("reading the value from consul", func() {
			value, err := kv.Get(testKey)
			Expect(err).NotTo(HaveOccurred())
			Expect(value).To(Equal(testValue))
		})

		By("removing the old encryption key", func() {
			manifest.Properties.Consul.EncryptKeys = []string{manifest.Properties.Consul.EncryptKeys[0]}
		})

		By("deploying with the original key removed", func() {
			yaml, err := manifest.ToYAML()
			Expect(err).NotTo(HaveOccurred())

			yaml, err = client.ResolveManifestVersions(yaml)
			Expect(err).NotTo(HaveOccurred())

			err = client.Deploy(yaml)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs(manifest.Name)
			}, "1m", "10s").Should(ConsistOf([]bosh.VM{
				{"running"},
				{"running"},
				{"running"},
				{"running"},
				{"running"},
			}))
		})

		By("setting a persistent value", func() {
			err := kv.Set(testKey, testValue)
			Expect(err).NotTo(HaveOccurred())
		})

		By("reading the value from consul", func() {
			value, err := kv.Get(testKey)
			Expect(err).NotTo(HaveOccurred())
			Expect(value).To(Equal(testValue))
		})
	})
})

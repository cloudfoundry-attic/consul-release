package deploy_test

import (
	"acceptance-tests/testing/bosh"
	"acceptance-tests/testing/consul"
	"acceptance-tests/testing/destiny"
	"acceptance-tests/testing/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Scaling down Instances", func() {
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

		manifest, kv, err = helpers.DeployConsulWithInstanceCount(3, client)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return client.DeploymentVMs(manifest.Name)
		}, "1m", "10s").Should(ConsistOf([]bosh.VM{
			{"running"},
			{"running"},
			{"running"},
		}))
	})

	AfterEach(func() {
		err := client.DeleteDeployment(manifest.Name)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("scaling from 3 nodes to 1", func() {
		PIt("saves data after a rolling deploy", func() {
			By("setting a persistent value", func() {
				err := kv.Set(testKey, testValue)
				Expect(err).NotTo(HaveOccurred())
			})

			By("scaling from 3 nodes to 1", func() {
				manifest.Jobs[0], manifest.Properties = destiny.SetJobInstanceCount(manifest.Jobs[0], manifest.Networks[0], manifest.Properties, 1)

				members := manifest.ConsulMembers()
				Expect(members).To(HaveLen(1))

				yaml, err := manifest.ToYAML()
				Expect(err).NotTo(HaveOccurred())

				err = client.Deploy(yaml)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() ([]bosh.VM, error) {
					return client.DeploymentVMs(manifest.Name)
				}, "1m", "10s").Should(ConsistOf([]bosh.VM{
					{"running"},
				}))
			})

			By("reading the value from consul", func() {
				value, err := kv.Get(testKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(value).To(Equal(testValue))
			})
		})
	})
})

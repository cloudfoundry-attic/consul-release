package turbulence_test

import (
	"acceptance-tests/testing/bosh"
	"acceptance-tests/testing/consul"
	"acceptance-tests/testing/destiny"
	"acceptance-tests/testing/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("KillVm", func() {
	var (
		consulManifest destiny.Manifest
		kv             consul.KV

		testKey   string
		testValue string
	)

	BeforeEach(func() {
		guid, err := helpers.NewGUID()
		Expect(err).NotTo(HaveOccurred())

		testKey = "consul-key-" + guid
		testValue = "consul-value-" + guid

		consulManifest, kv, err = helpers.DeployConsulWithInstanceCount(3, client)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return client.DeploymentVMs(consulManifest.Name)
		}, "1m", "10s").Should(ConsistOf([]bosh.VM{
			{"running"},
			{"running"},
			{"running"},
		}))
	})

	AfterEach(func() {
		By("fixing the deployment", func() {
			yaml, err := consulManifest.ToYAML()
			Expect(err).NotTo(HaveOccurred())

			err = client.ScanAndFix(yaml)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs(consulManifest.Name)
			}, "1m", "10s").Should(ConsistOf([]bosh.VM{
				{"running"},
				{"running"},
				{"running"},
			}))
		})

		By("deleting the deployment", func() {
			err := client.DeleteDeployment(consulManifest.Name)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when a consul node is killed", func() {
		It("is still able to function on healthy vms", func() {
			By("setting a persistent value", func() {
				err := kv.Set(testKey, testValue)
				Expect(err).NotTo(HaveOccurred())
			})

			By("killing indices", func() {
				err := turbulenceClient.KillIndices(consulManifest.Name, "consul_z1", []int{0})
				Expect(err).ToNot(HaveOccurred())
			})

			By("reading the value from consul", func() {
				value, err := kv.Get(testKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(value).To(Equal(testValue))
			})
		})
	})
})

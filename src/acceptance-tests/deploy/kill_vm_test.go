package deploy_test

import (
	"math/rand"
	"time"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/consulclient"
	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/helpers"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	turbulenceclient "github.com/pivotal-cf-experimental/bosh-test/turbulence"
	"github.com/pivotal-cf-experimental/destiny/consul"
	"github.com/pivotal-cf-experimental/destiny/turbulence"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("KillVm", func() {
	var (
		turbulenceManifest turbulence.Manifest
		turbulenceClient   turbulenceclient.Client

		consulManifest consul.Manifest
		kv             consulclient.HTTPKV

		spammer   *helpers.Spammer
		testKey   string
		testValue string
	)

	BeforeEach(func() {
		var err error
		turbulenceManifest, err = helpers.DeployTurbulence(boshClient, config)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return boshClient.DeploymentVMs(turbulenceManifest.Name)
		}, "1m", "10s").Should(ConsistOf(helpers.GetTurbulenceVMsFromManifest(turbulenceManifest)))

		turbulenceClient = helpers.NewTurbulenceClient(turbulenceManifest)

		guid, err := helpers.NewGUID()
		Expect(err).NotTo(HaveOccurred())

		testKey = "consul-key-" + guid
		testValue = "consul-value-" + guid

		consulManifest, kv, err = helpers.DeployConsulWithInstanceCount(3, boshClient, config)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return boshClient.DeploymentVMs(consulManifest.Name)
		}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(consulManifest)))

		spammer = helpers.NewSpammer(kv, 1*time.Second, "test-consumer-0")
	})

	AfterEach(func() {
		By("fixing the deployment", func() {
			yaml, err := consulManifest.ToYAML()
			Expect(err).NotTo(HaveOccurred())

			err = boshClient.ScanAndFix(yaml)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return boshClient.DeploymentVMs(consulManifest.Name)
			}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(consulManifest)))
		})

		By("deleting the deployment", func() {
			if !CurrentGinkgoTestDescription().Failed {
				err := boshClient.DeleteDeployment(consulManifest.Name)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		By("deleting the turbulence deployment", func() {
			if !CurrentGinkgoTestDescription().Failed {
				err := boshClient.DeleteDeployment(turbulenceManifest.Name)
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})
	Context("when a consul node is killed", func() {
		It("is still able to function on healthy vms", func() {
			By("setting a persistent value", func() {
				err := kv.Set(testKey, testValue)
				Expect(err).NotTo(HaveOccurred())
			})

			By("killing indices", func() {
				spammer.Spam()

				err := turbulenceClient.KillIndices(consulManifest.Name, "consul_z1", []int{rand.Intn(3)})
				Expect(err).ToNot(HaveOccurred())

				yaml, err := consulManifest.ToYAML()
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() error {
					return boshClient.ScanAndFix(yaml)
				}, "5m", "1m").ShouldNot(HaveOccurred())

				Eventually(func() ([]bosh.VM, error) {
					return boshClient.DeploymentVMs(consulManifest.Name)
				}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(consulManifest)))

				spammer.Stop()
			})

			By("reading the value from consul", func() {
				value, err := kv.Get(testKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(value).To(Equal(testValue))

				err = spammer.Check()
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})

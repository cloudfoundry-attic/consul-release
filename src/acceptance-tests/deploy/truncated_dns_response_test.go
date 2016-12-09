package deploy_test

import (
	"fmt"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/helpers"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/consul"

	testconsumerclient "github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/testconsumer/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("given large DNS response", func() {
	var (
		consulManifest     consul.ManifestV2
		testConsumerClient testconsumerclient.Client
		err                error
	)
	BeforeEach(func() {
		consulManifest, _, err = helpers.DeployConsulWithFakeDNSServer("large-dns-response", 1, boshClient, config)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return helpers.DeploymentVMs(boshClient, consulManifest.Name)
		}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(consulManifest)))

		testConsumerClient = testconsumerclient.New(fmt.Sprintf("http://%s:6769", consulManifest.InstanceGroups[1].Networks[0].StaticIPs[0]))
	})

	AfterEach(func() {
		By("deleting consul deployment", func() {
			if !CurrentGinkgoTestDescription().Failed {
				Eventually(func() ([]string, error) {
					return lockedDeployments()
				}, "10m", "30s").ShouldNot(ContainElement(consulManifest.Name))

				err := boshClient.DeleteDeployment(consulManifest.Name)
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})

	It("does not error out", func() {
		addresses, err := testConsumerClient.DNS("large-dns-response.fake.local")
		Expect(err).NotTo(HaveOccurred())
		Expect(addresses).To(Equal([]string{
			"1.2.3.0", "1.2.3.0",
			"1.2.3.1", "1.2.3.1",
			"1.2.3.2", "1.2.3.2",
			"1.2.3.3", "1.2.3.3",
			"1.2.3.0", "1.2.3.0",
			"1.2.3.1", "1.2.3.1",
			"1.2.3.2", "1.2.3.2",
			"1.2.3.3", "1.2.3.3",
		}))
	})
})

package deploy_test

import (
	"fmt"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/helpers"
	testconsumerclient "github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/testconsumer/client"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/ops"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Scaling down to zero server instances", func() {
	var (
		manifest     string
		manifestName string

		tcClient testconsumerclient.Client
	)

	Describe("Scaling down to zero server instances", func() {
		BeforeEach(func() {
			var err error
			manifest, err = helpers.DeployConsulWithInstanceCount("scale-down-to-zero", 1, config.WindowsClients, boshClient)
			Expect(err).NotTo(HaveOccurred())

			manifestName, err = ops.ManifestName(manifest)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return helpers.DeploymentVMs(boshClient, manifestName)
			}, "5m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))

			testConsumerIPs, err := helpers.GetVMIPs(boshClient, manifestName, "testconsumer")
			Expect(err).NotTo(HaveOccurred())

			tcClient = testconsumerclient.New(fmt.Sprintf("http://%s:6769", testConsumerIPs[0]))
		})

		AfterEach(func() {
			if !CurrentGinkgoTestDescription().Failed {
				err := boshClient.DeleteDeployment(manifestName)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("disables the consul_agent on the testconsumer VM", func() {
			By("verifying that the consul_agent is running on the testconsumer", func() {
				consulIPs, err := helpers.GetVMIPs(boshClient, manifestName, "consul")
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() ([]string, error) {
					return tcClient.DNS("consul.service.cf.internal")
				}, "5m", "10s").Should(ConsistOf(consulIPs))
			})

			By("scaling down to zero servier instances", func() {
				var err error
				manifest, err = ops.ApplyOp(manifest, ops.Op{
					Type:  "replace",
					Path:  "/instance_groups/name=consul/instances",
					Value: 0,
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = boshClient.Deploy([]byte(manifest))
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() ([]bosh.VM, error) {
					return helpers.DeploymentVMs(boshClient, manifestName)
				}, "5m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))
			})

			By("verifying that the consul_agent is no longer running on the testconsumer", func() {
				Eventually(func() ([]string, error) {
					return tcClient.DNS("consul.service.cf.internal")
				}, "5m", "10s").Should(BeEmpty())
			})
		})
	})
})

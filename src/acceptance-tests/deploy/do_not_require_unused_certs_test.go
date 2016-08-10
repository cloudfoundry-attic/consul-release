package deploy_test

import (
	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/helpers"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/consul"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Do not require unused certs in server/client modes", func() {
	var (
		manifest consul.Manifest
	)

	AfterEach(func() {
		if !CurrentGinkgoTestDescription().Failed {
			err := boshClient.DeleteDeployment(manifest.Name)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	It("successfully deploys a consul server and client without unused certs", func() {
		var err error
		manifest, err = helpers.DeployConsulWithJobLevelConsulProperties(boshClient, config)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return boshClient.DeploymentVMs(manifest.Name)
		}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))
	})
})

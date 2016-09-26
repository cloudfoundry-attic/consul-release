package deploy_test

import (
	"fmt"
	"testing"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/helpers"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/turbulence"

	turbulenceclient "github.com/pivotal-cf-experimental/bosh-test/turbulence"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	config               helpers.Config
	boshClient           bosh.Client
	consulReleaseVersion string
	turbulenceManifest   turbulence.Manifest
	turbulenceClient     turbulenceclient.Client
)

func TestDeploy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "deploy")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	configPath, err := helpers.ConfigPath()
	Expect(err).NotTo(HaveOccurred())

	config, err := helpers.LoadConfig(configPath)
	Expect(err).NotTo(HaveOccurred())

	boshClient := bosh.NewClient(bosh.Config{
		URL:              fmt.Sprintf("https://%s:25555", config.BOSH.Target),
		Username:         config.BOSH.Username,
		Password:         config.BOSH.Password,
		AllowInsecureSSL: true,
	})

	turbulenceManifest, err := helpers.DeployTurbulence(boshClient, config)
	Expect(err).NotTo(HaveOccurred())

	Eventually(func() ([]bosh.VM, error) {
		return helpers.DeploymentVMs(boshClient, turbulenceManifest.Name)
	}, "1m", "10s").Should(ConsistOf(helpers.GetTurbulenceVMsFromManifest(turbulenceManifest)))

	turbulenceManifestBytes, err := turbulenceManifest.ToYAML()
	Expect(err).NotTo(HaveOccurred())

	return turbulenceManifestBytes
}, func(turbulenceManifestBytes []byte) {
	var err error
	turbulenceManifest, err = turbulence.FromYAML(turbulenceManifestBytes)
	Expect(err).NotTo(HaveOccurred())

	configPath, err := helpers.ConfigPath()
	Expect(err).NotTo(HaveOccurred())

	config, err = helpers.LoadConfig(configPath)
	Expect(err).NotTo(HaveOccurred())

	consulReleaseVersion = helpers.ConsulReleaseVersion()
	boshClient = bosh.NewClient(bosh.Config{
		URL:              fmt.Sprintf("https://%s:25555", config.BOSH.Target),
		Username:         config.BOSH.Username,
		Password:         config.BOSH.Password,
		AllowInsecureSSL: true,
	})

	turbulenceClient = helpers.NewTurbulenceClient(turbulenceManifest)
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	err := boshClient.DeleteDeployment(turbulenceManifest.Name)
	Expect(err).NotTo(HaveOccurred())
})

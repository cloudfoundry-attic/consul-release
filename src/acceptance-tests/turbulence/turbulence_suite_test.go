package turbulence_test

import (
	"acceptance-tests/testing/bosh"
	"acceptance-tests/testing/destiny"
	"acceptance-tests/testing/helpers"
	"acceptance-tests/testing/turbulence"

	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestTurbulence(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "turbulence")
}

var (
	config helpers.Config
	client bosh.Client

	turbulenceManifest destiny.Manifest
	turbulenceClient   turbulence.Client
)

var _ = BeforeSuite(func() {
	configPath, err := helpers.ConfigPath()
	Expect(err).NotTo(HaveOccurred())

	config, err = helpers.LoadConfig(configPath)
	Expect(err).NotTo(HaveOccurred())

	client = bosh.NewClient(bosh.Config{
		URL:              config.BOSHTarget,
		Username:         config.BOSHUsername,
		Password:         config.BOSHPassword,
		AllowInsecureSSL: true,
	})

	By("deploying turbulence", func() {
		uuid, err := client.DirectorUUID()
		Expect(err).NotTo(HaveOccurred())

		guid, err := helpers.NewGUID()
		Expect(err).NotTo(HaveOccurred())

		turbulenceManifest = destiny.NewTurbulence(destiny.Config{
			DirectorUUID: uuid,
			Name:         "turbulence-" + guid,
		})

		yaml, err := turbulenceManifest.ToYAML()
		Expect(err).NotTo(HaveOccurred())

		yaml, err = client.ResolveManifestVersions(yaml)
		Expect(err).NotTo(HaveOccurred())

		turbulenceManifest, err = destiny.FromYAML(yaml)
		Expect(err).NotTo(HaveOccurred())

		err = client.Deploy(yaml)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return client.DeploymentVMs(turbulenceManifest.Name)
		}, "1m", "10s").Should(ConsistOf([]bosh.VM{
			{"running"},
		}))
	})

	By("preparing turbulence client", func() {
		turbulenceUrl := fmt.Sprintf("https://turbulence:%s@%s:8080",
			turbulenceManifest.Properties.TurbulenceAPI.Password,
			turbulenceManifest.Jobs[0].Networks[0].StaticIPs[0])

		turbulenceClient = turbulence.NewClient(turbulenceUrl)
	})
})

var _ = AfterSuite(func() {
	By("deleting the turbulence deployment", func() {
		err := client.DeleteDeployment(turbulenceManifest.Name)
		Expect(err).NotTo(HaveOccurred())
	})
})

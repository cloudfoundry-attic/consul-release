package turbulence_test

import (
	"acceptance-tests/helpers"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"testing"
)

func TestTurbulence(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Turbulence Suite")
}

var (
	goPath             string
	config             helpers.Config
	bosh               *helpers.Bosh
	turbulenceManifest *helpers.Manifest

	consulRelease    = fmt.Sprintf("consul-%s", generator.RandomName())
	consulDeployment = consulRelease

	turbulenceDeployment = fmt.Sprintf("turb-%s", generator.RandomName())

	directorUUIDStub, consulNameOverrideStub, turbulenceNameOverrideStub string

	turbulenceManifestGeneration string
	consulManifestGeneration     string
)

var _ = BeforeSuite(func() {
	goPath = helpers.SetupGoPath()
	gemfilePath := helpers.SetupFastBosh()
	config = helpers.LoadConfig()
	boshOperationTimeout := helpers.GetBoshOperationTimeout(config)

	bosh = helpers.NewBosh(gemfilePath, goPath, config.BoshTarget, boshOperationTimeout)

	turbulenceManifestGeneration = filepath.Join(goPath, "scripts", "generate_turbulence_deployment_manifest")
	consulManifestGeneration = filepath.Join(goPath, "scripts", "generate-consul-deployment-manifest")

	directorUUIDStub = bosh.TargetDeployment()

	err := os.Chdir(goPath)
	Expect(err).ToNot(HaveOccurred())

	uploadBoshCpiRelease()

	createTurbulenceStub()

	turbulenceManifest = new(helpers.Manifest)
	bosh.GenerateAndSetDeploymentManifest(
		turbulenceManifest,
		turbulenceManifestGeneration,
		directorUUIDStub,
		helpers.TurbulenceInstanceCountOverridesStubPath,
		helpers.TurbulencePersistentDiskOverridesStubPath,
		config.IAASSettingsTurbulenceStubPath,
		config.TurbulencePropertiesStubPath,
		turbulenceNameOverrideStub,
	)

	By("uploading the turbulence release")
	Expect(bosh.Command("-n", "upload", "release", config.TurbulenceReleaseLocation)).To(Exit(0))

	By("deploying the turbulence release")
	Expect(bosh.Command("-n", "deploy")).To(Exit(0))

	createConsulStub()
	bosh.CreateAndUploadRelease(filepath.Join(goPath, "..", ".."), consulRelease)
})

var _ = AfterSuite(func() {
	By("delete consul release")
	bosh.Command("-n", "delete", "release", consulRelease)

	By("delete turbulence deployment")
	bosh.Command("-n", "delete", "deployment", turbulenceDeployment)

	By("deleting the cpi release")
	bosh.Command("-n", "delete", "release", config.CPIReleaseName)

	By("deleting the turbulence release")
	bosh.Command("-n", "delete", "release", config.TurbulenceReleaseName)

})

func createConsulStub() {
	By("creating the consul overrides stub")
	consulStub := fmt.Sprintf(`---
name_overrides:
  release_name: %s
  deployment_name: %s
`, consulRelease, consulDeployment)

	consulNameOverrideStub = helpers.WriteStub(consulStub)
}

func createTurbulenceStub() {
	By("creating the turbulence overrides stub")
	turbulenceStub := fmt.Sprintf(`---
name_overrides:
  deployment_name: %s
  turbulence:
    release_name: %s
  warden_cpi:
    release_name: %s
`, turbulenceDeployment, config.TurbulenceReleaseName, config.CPIReleaseName)

	turbulenceNameOverrideStub = helpers.WriteStub(turbulenceStub)
}

func uploadBoshCpiRelease() {
	By("Downloading remote release")
	if config.CPIReleaseLocation == "" {
		panic("missing required cpi release location")
	}

	if config.CPIReleaseName == "" {
		panic("missing required warden_cpi release name")
	}

	Expect(bosh.Command("-n", "upload", "release", config.CPIReleaseLocation, "--skip-if-exists")).To(Exit(0))
}

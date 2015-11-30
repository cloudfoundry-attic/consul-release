package deploy_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"acceptance-tests/helpers"

	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestDeploy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deploy Suite")
}

var (
	goPath string
	config helpers.Config

	bosh *helpers.Bosh

	consulManifestGeneration string

	directorUUIDStub string

	consulRelease          = fmt.Sprintf("consul-%s", generator.RandomName())
	consulDeployment       = consulRelease
	consulNameOverrideStub string
)

var _ = BeforeSuite(func() {
	goPath = helpers.SetupGoPath()
	gemfilePath := helpers.SetupFastBosh()
	config = helpers.LoadConfig()
	boshOperationTimeout := helpers.GetBoshOperationTimeout(config)
	bosh = helpers.NewBosh(gemfilePath, goPath, config.BoshTarget, boshOperationTimeout)

	consulManifestGeneration = filepath.Join(goPath, "scripts", "generate-consul-deployment-manifest")

	err := os.Chdir(goPath)
	Expect(err).ToNot(HaveOccurred())

	directorUUIDStub = bosh.TargetDeployment()
	createConsulStub()
	bosh.CreateAndUploadRelease(filepath.Join(goPath, "..", ".."), consulRelease)
})

var _ = AfterSuite(func() {
	if bosh == nil {
		return
	}

	By("delete release")
	bosh.Command("-n", "delete", "release", consulRelease)
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

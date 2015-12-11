package deploy_test

import (
	"acceptance-tests/testing/bosh"
	"acceptance-tests/testing/helpers"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	config helpers.Config
	client bosh.Client
)

func TestDeploy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "deploy")
}

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
})

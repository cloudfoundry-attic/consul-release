package dns_test

import (
	"fmt"
	"testing"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/helpers"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	config helpers.Config
	client bosh.Client
)

func TestDNS(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "dns")
}

var _ = BeforeSuite(func() {
	configPath, err := helpers.ConfigPath()
	Expect(err).NotTo(HaveOccurred())

	config, err = helpers.LoadConfig(configPath)
	Expect(err).NotTo(HaveOccurred())

	client = bosh.NewClient(bosh.Config{
		URL:              fmt.Sprintf("https://%s:25555", config.BOSH.Target),
		Username:         config.BOSH.Username,
		Password:         config.BOSH.Password,
		AllowInsecureSSL: true,
	})
})

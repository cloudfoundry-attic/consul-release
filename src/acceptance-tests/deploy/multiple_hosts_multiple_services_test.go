package deploy_test

import (
	"fmt"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/helpers"
	testconsumerclient "github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/testconsumer/client"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/consul"
	"github.com/pivotal-cf-experimental/destiny/core"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Multiple hosts multiple services", func() {
	var (
		manifest consul.Manifest
		tcClient testconsumerclient.Client
	)

	BeforeEach(func() {
		var err error

		manifest, _, err = helpers.DeployConsulWithInstanceCount(3, boshClient, config)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return boshClient.DeploymentVMs(manifest.Name)
		}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))

		tcClient = testconsumerclient.New(fmt.Sprintf("http://%s:6769", manifest.Jobs[1].Networks[0].StaticIPs[0]))
	})

	AfterEach(func() {
		if !CurrentGinkgoTestDescription().Failed {
			err := boshClient.DeleteDeployment(manifest.Name)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	It("discovers multiples services on multiple hosts", func() {
		By("registering services", func() {
			healthCheck := fmt.Sprintf("curl -f http://%s:6769/health_check", manifest.Jobs[1].Networks[0].StaticIPs[0])
			manifest.Jobs[0].Properties.Consul.Agent.Services = core.JobPropertiesConsulAgentServices{
				"some-service": core.JobPropertiesConsulAgentService{
					Check: &core.JobPropertiesConsulAgentServiceCheck{
						Name:     "some-service-check",
						Script:   healthCheck,
						Interval: "1m",
					},
				},
				"some-other-service": core.JobPropertiesConsulAgentService{
					Check: &core.JobPropertiesConsulAgentServiceCheck{
						Name:     "some-other-service-check",
						Script:   healthCheck,
						Interval: "1m",
					},
				},
			}
		})

		By("deploying", func() {
			yaml, err := manifest.ToYAML()
			Expect(err).NotTo(HaveOccurred())

			yaml, err = boshClient.ResolveManifestVersions(yaml)
			Expect(err).NotTo(HaveOccurred())

			_, err = boshClient.Deploy(yaml)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return boshClient.DeploymentVMs(manifest.Name)
			}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))
		})

		By("resolving service addresses", func() {
			Eventually(func() ([]string, error) {
				return tcClient.DNS("some-service.service.cf.internal")
			}, "1m", "10s").Should(ConsistOf(manifest.Jobs[0].Networks[0].StaticIPs))

			Eventually(func() ([]string, error) {
				return tcClient.DNS("consul-z1-0.some-service.service.cf.internal")
			}, "1m", "10s").Should(ConsistOf(manifest.Jobs[0].Networks[0].StaticIPs[0]))

			Eventually(func() ([]string, error) {
				return tcClient.DNS("consul-z1-1.some-service.service.cf.internal")
			}, "1m", "10s").Should(ConsistOf(manifest.Jobs[0].Networks[0].StaticIPs[1]))

			Eventually(func() ([]string, error) {
				return tcClient.DNS("consul-z1-2.some-service.service.cf.internal")
			}, "1m", "10s").Should(ConsistOf(manifest.Jobs[0].Networks[0].StaticIPs[2]))

			Eventually(func() ([]string, error) {
				return tcClient.DNS("some-other-service.service.cf.internal")
			}, "1m", "10s").Should(ConsistOf(manifest.Jobs[0].Networks[0].StaticIPs))

			Eventually(func() ([]string, error) {
				return tcClient.DNS("consul-z1-0.some-other-service.service.cf.internal")
			}, "1m", "10s").Should(ConsistOf(manifest.Jobs[0].Networks[0].StaticIPs[0]))

			Eventually(func() ([]string, error) {
				return tcClient.DNS("consul-z1-1.some-other-service.service.cf.internal")
			}, "1m", "10s").Should(ConsistOf(manifest.Jobs[0].Networks[0].StaticIPs[1]))

			Eventually(func() ([]string, error) {
				return tcClient.DNS("consul-z1-2.some-other-service.service.cf.internal")
			}, "1m", "10s").Should(ConsistOf(manifest.Jobs[0].Networks[0].StaticIPs[2]))
		})
	})
})

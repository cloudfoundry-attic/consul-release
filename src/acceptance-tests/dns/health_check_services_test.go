package dns_test

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/consul"
	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/helpers"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Health Check", func() {
	var (
		manifest       destiny.Manifest
		agent          consul.AgentStartStopper
		healthCheckURL string
	)

	BeforeEach(func() {
		var err error

		manifest, _, err = helpers.DeployConsulWithInstanceCount(1, client, config)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return client.DeploymentVMs(manifest.Name)
		}, "1m", "10s").Should(ConsistOf([]bosh.VM{
			{"running"},
			{"running"},
		}))

		agent, err = helpers.NewConsulAgent(manifest, 2)
		Expect(err).NotTo(HaveOccurred())

		agent.Start()
		healthCheckURL = fmt.Sprintf("http://%s:6769/health_check", manifest.Jobs[1].Networks[0].StaticIPs[0])
	})

	AfterEach(func() {
		if !CurrentGinkgoTestDescription().Failed {
			err := client.DeleteDeployment(manifest.Name)
			Expect(err).NotTo(HaveOccurred())
		}
		agent.Stop()
	})

	It("deregisters a service if the health check fails", func() {
		By("registering a service", func() {
			manifest.Jobs[0].Properties.Consul.Agent.Services = destiny.JobPropertiesConsulAgentServices{
				"some-service": destiny.JobPropertiesConsulAgentService{
					Name: "some-service-name",
					Check: &destiny.JobPropertiesConsulAgentServiceCheck{
						Name:     "some-service-check",
						Script:   fmt.Sprintf("curl -f %s", healthCheckURL),
						Interval: "10s",
					},
					Tags: []string{"some-service-tag"},
				},
			}
		})

		By("deploying", func() {
			yaml, err := manifest.ToYAML()
			Expect(err).NotTo(HaveOccurred())

			yaml, err = client.ResolveManifestVersions(yaml)
			Expect(err).NotTo(HaveOccurred())

			err = client.Deploy(yaml)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs(manifest.Name)
			}, "1m", "10s").Should(ConsistOf([]bosh.VM{
				{"running"},
				{"running"},
			}))
		})

		By("resolving the service address", func() {
			Eventually(func() ([]string, error) {
				return checkService("some-service-name.service.cf.internal")
			}, "1m", "10s").Should(ConsistOf(manifest.Jobs[0].Networks[0].StaticIPs))
		})

		By("causing the health check to fail", func() {
			response, err := http.Post(healthCheckURL, "", strings.NewReader("false"))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))
		})

		By("the service should be deregistered", func() {
			Eventually(func() ([]string, error) {
				return checkService("some-service-name.service.cf.internal")
			}, "1m", "10s").Should(BeEmpty())
		})

		By("causing the health check to succeed", func() {
			response, err := http.Post(healthCheckURL, "", strings.NewReader("true"))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))
		})

		By("the service should be alive", func() {
			Eventually(func() ([]string, error) {
				return checkService("some-service-name.service.cf.internal")
			}, "1m", "10s").Should(ConsistOf(manifest.Jobs[0].Networks[0].StaticIPs))
		})
	})
})

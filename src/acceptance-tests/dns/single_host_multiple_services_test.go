package dns_test

import (
	"fmt"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/consulclient"
	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/helpers"
	"github.com/miekg/dns"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Single host multiple services", func() {
	var (
		manifest destiny.Manifest
		agent    consulclient.AgentStartStopper
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
	})

	AfterEach(func() {
		if !CurrentGinkgoTestDescription().Failed {
			err := client.DeleteDeployment(manifest.Name)
			Expect(err).NotTo(HaveOccurred())
		}
		agent.Stop()
	})

	It("discovers multiples services on a single host", func() {
		By("registering services", func() {
			manifest.Jobs[0].Properties.Consul.Agent.Services = destiny.JobPropertiesConsulAgentServices{
				"some-service": destiny.JobPropertiesConsulAgentService{
					Name: "some-service-name",
					Check: &destiny.JobPropertiesConsulAgentServiceCheck{
						Name:     "some-service-check",
						Script:   fmt.Sprintf("curl http://%s:6769/health_check", manifest.Jobs[1].Networks[0].StaticIPs[0]),
						Interval: "1m",
					},
					Tags: []string{"some-service-tag"},
				},
				"some-other-service": destiny.JobPropertiesConsulAgentService{
					Name: "some-other-service-name",
					Check: &destiny.JobPropertiesConsulAgentServiceCheck{
						Name:     "some-other-service-check",
						Script:   fmt.Sprintf("curl http://%s:6769/health_check", manifest.Jobs[1].Networks[0].StaticIPs[0]),
						Interval: "1m",
					},
					Tags: []string{"some-other-service-tag"},
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

		By("resolving service addresses", func() {
			Eventually(func() ([]string, error) {
				return checkService("some-service-name.service.cf.internal")
			}, "1m", "10s").Should(ConsistOf(manifest.Jobs[0].Networks[0].StaticIPs))

			Eventually(func() ([]string, error) {
				return checkService("some-other-service-name.service.cf.internal")
			}, "1m", "10s").Should(ConsistOf(manifest.Jobs[0].Networks[0].StaticIPs))
		})
	})
})

func checkService(service string) ([]string, error) {
	c := dns.Client{}
	m := dns.Msg{}

	m.SetQuestion(service+".", dns.TypeA)

	r, _, err := c.Exchange(&m, "127.0.0.1:8600")
	if err != nil {
		return []string{}, err
	}

	var ips []string
	for _, ans := range r.Answer {
		Arecord := ans.(*dns.A)
		ips = append(ips, Arecord.A.String())
	}

	return ips, nil
}

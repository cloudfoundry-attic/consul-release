package dns_test

import (
	"fmt"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/consulclient"
	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/helpers"
	"github.com/miekg/dns"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/consul"
	"github.com/pivotal-cf-experimental/destiny/core"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Single host multiple services", func() {
	var (
		manifest consul.Manifest
		agent    consulclient.AgentStartStopper
	)

	BeforeEach(func() {
		var err error

		manifest, _, err = helpers.DeployConsulWithInstanceCount(1, client, config)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return client.DeploymentVMs(manifest.Name)
		}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))

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
			healthCheck := fmt.Sprintf("curl -f http://%s:6769/health_check", manifest.Jobs[1].Networks[0].StaticIPs[0])
			manifest.Jobs[1].Properties = &core.JobProperties{
				Consul: core.JobPropertiesConsul{
					Agent: core.JobPropertiesConsulAgent{
						Mode: "client",
						Services: core.JobPropertiesConsulAgentServices{
							"consul-test-consumer": core.JobPropertiesConsulAgentService{},
							"some-service": core.JobPropertiesConsulAgentService{
								Check: &core.JobPropertiesConsulAgentServiceCheck{
									Name:     "some-service-check",
									Script:   healthCheck,
									Interval: "1m",
								},
							},
						},
					},
				},
			}
		})

		By("deploying", func() {
			yaml, err := manifest.ToYAML()
			Expect(err).NotTo(HaveOccurred())

			yaml, err = client.ResolveManifestVersions(yaml)
			Expect(err).NotTo(HaveOccurred())

			_, err = client.Deploy(yaml)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs(manifest.Name)
			}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))
		})

		By("resolving service addresses", func() {
			Eventually(func() ([]string, error) {
				return checkService("consul-test-consumer.service.cf.internal")
			}, "1m", "10s").Should(ConsistOf(manifest.Jobs[1].Networks[0].StaticIPs))

			Eventually(func() ([]string, error) {
				return checkService("some-service.service.cf.internal")
			}, "1m", "10s").Should(ConsistOf(manifest.Jobs[1].Networks[0].StaticIPs))
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

package deploy_test

import (
	"fmt"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/helpers"
	testconsumerclient "github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/testconsumer/client"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/ops"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Multiple hosts multiple services", func() {
	var (
		manifest     string
		manifestName string

		testConsumerIP string
		tcClient       testconsumerclient.Client
	)

	BeforeEach(func() {
		var err error
		manifest, err = helpers.DeployConsulWithOpsWithInstanceCount("multiple-host-multiple-services", 3, boshClient)
		Expect(err).NotTo(HaveOccurred())

		manifestName, err = ops.ManifestName(manifest)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return helpers.DeploymentVMs(boshClient, manifestName)
		}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifestV2(manifest)))

		testConsumerIPs, err := helpers.GetVMIPs(boshClient, manifestName, "testconsumer")
		Expect(err).NotTo(HaveOccurred())

		testConsumerIP = testConsumerIPs[0]

		tcClient = testconsumerclient.New(fmt.Sprintf("http://%s:6769", testConsumerIP))
	})

	AfterEach(func() {
		if !CurrentGinkgoTestDescription().Failed {
			err := boshClient.DeleteDeployment(manifestName)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	It("discovers multiples services on multiple hosts", func() {
		By("registering services", func() {
			healthCheck := fmt.Sprintf("curl -f http://%s:6769/health_check", testConsumerIP)

			var err error
			manifest, err = ops.ApplyOp(manifest, ops.Op{
				Type: "replace",
				Path: "/instance_groups/name=consul/properties/consul/agent/services",
				Value: map[string]service{
					"some-service": service{
						Name: "some-service-name",
						Check: serviceCheck{
							Name:     "some-service-check",
							Script:   healthCheck,
							Interval: "10s",
						},
					},
					"some-other-service": service{
						Name: "some-other-service-name",
						Check: serviceCheck{
							Name:     "some-other-service-check",
							Script:   healthCheck,
							Interval: "10s",
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
		})

		By("deploying", func() {
			_, err := boshClient.Deploy([]byte(manifest))
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return helpers.DeploymentVMs(boshClient, manifestName)
			}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifestV2(manifest)))
		})

		By("resolving service addresses", func() {
			consulIPs, err := helpers.GetVMIPs(boshClient, manifestName, "consul")
			Expect(err).NotTo(HaveOccurred())

			deploymentVMs, err := boshClient.DeploymentVMs(manifestName)
			Expect(err).NotTo(HaveOccurred())

			var consulVMs []bosh.VM
			for _, vm := range deploymentVMs {
				if vm.JobName == "consul" {
					consulVMs = append(consulVMs, vm)
				}
			}

			Eventually(func() ([]string, error) {
				return tcClient.DNS("some-service-name.service.cf.internal")
			}, "2m", "10s").Should(ConsistOf(consulIPs))

			Eventually(func() ([]string, error) {
				return tcClient.DNS("consul-0.some-service-name.service.cf.internal")
			}, "2m", "10s").Should(ConsistOf(consulVMs[0].IPs))

			Eventually(func() ([]string, error) {
				return tcClient.DNS("consul-1.some-service-name.service.cf.internal")
			}, "2m", "10s").Should(ConsistOf(consulVMs[1].IPs))

			Eventually(func() ([]string, error) {
				return tcClient.DNS("consul-2.some-service-name.service.cf.internal")
			}, "2m", "10s").Should(ConsistOf(consulVMs[2].IPs))

			Eventually(func() ([]string, error) {
				return tcClient.DNS("some-other-service-name.service.cf.internal")
			}, "2m", "10s").Should(ConsistOf(consulIPs))

			Eventually(func() ([]string, error) {
				return tcClient.DNS("consul-0.some-other-service-name.service.cf.internal")
			}, "2m", "10s").Should(ConsistOf(consulVMs[0].IPs))

			Eventually(func() ([]string, error) {
				return tcClient.DNS("consul-1.some-other-service-name.service.cf.internal")
			}, "2m", "10s").Should(ConsistOf(consulVMs[1].IPs))

			Eventually(func() ([]string, error) {
				return tcClient.DNS("consul-2.some-other-service-name.service.cf.internal")
			}, "2m", "10s").Should(ConsistOf(consulVMs[2].IPs))
		})
	})
})

package deploy_test

import (
	"time"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/consul"
	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/helpers"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Scaling down instances", func() {
	var (
		manifest  destiny.Manifest
		kv        consul.HTTPKV
		testKey   string
		testValue string
		spammer   *helpers.Spammer
	)

	AfterEach(func() {
		if !CurrentGinkgoTestDescription().Failed {
			err := client.DeleteDeployment(manifest.Name)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	Describe("scaling from 3 nodes to 1", func() {
		BeforeEach(func() {
			guid, err := helpers.NewGUID()
			Expect(err).NotTo(HaveOccurred())

			testKey = "consul-key-" + guid
			testValue = "consul-value-" + guid

			manifest, kv, err = helpers.DeployConsulWithInstanceCount(3, client, config)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs(manifest.Name)
			}, "1m", "10s").Should(ConsistOf([]bosh.VM{
				{"running"},
				{"running"},
				{"running"},
				{"running"},
			}))
		})

		It("provides a functioning server after the scale down", func() {
			By("setting a persistent value to check the cluster is up", func() {
				err := kv.Set(testKey, testValue)
				Expect(err).NotTo(HaveOccurred())
			})

			By("scaling from 3 nodes to 1", func() {
				manifest.Jobs[0], manifest.Properties = helpers.SetJobInstanceCount(manifest.Jobs[0], manifest.Networks[0], manifest.Properties, 1)

				members := manifest.ConsulMembers()
				Expect(members).To(HaveLen(2))

				yaml, err := manifest.ToYAML()
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

			By("setting a persistent value to check the cluster is up after the scale down", func() {
				err := kv.Set(testKey, testValue)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("scaling from 5 nodes to 3", func() {
		BeforeEach(func() {
			guid, err := helpers.NewGUID()
			Expect(err).NotTo(HaveOccurred())

			testKey = "consul-key-" + guid
			testValue = "consul-value-" + guid

			manifest, kv, err = helpers.DeployConsulWithInstanceCount(5, client, config)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs(manifest.Name)
			}, "1m", "10s").Should(ConsistOf([]bosh.VM{
				{"running"},
				{"running"},
				{"running"},
				{"running"},
				{"running"},
				{"running"},
			}))

			spammer = helpers.NewSpammer(kv, 1*time.Second)
		})

		It("persists data throughout the scale down", func() {
			By("setting a persistent value", func() {
				err := kv.Set(testKey, testValue)
				Expect(err).NotTo(HaveOccurred())
			})

			By("scaling from 5 nodes to 3", func() {
				manifest.Jobs[0], manifest.Properties = helpers.SetJobInstanceCount(manifest.Jobs[0], manifest.Networks[0], manifest.Properties, 3)

				members := manifest.ConsulMembers()
				Expect(members).To(HaveLen(4))

				yaml, err := manifest.ToYAML()
				Expect(err).NotTo(HaveOccurred())

				spammer.Spam()

				err = client.Deploy(yaml)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() ([]bosh.VM, error) {
					return client.DeploymentVMs(manifest.Name)
				}, "1m", "10s").Should(ConsistOf([]bosh.VM{
					{"running"},
					{"running"},
					{"running"},
					{"running"},
				}))

				spammer.Stop()
			})

			By("reading the value from consul", func() {
				value, err := kv.Get(testKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(value).To(Equal(testValue))

				err = spammer.Check()
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})

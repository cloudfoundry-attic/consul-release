package deploy_test

import (
	"time"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/consulclient"
	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/helpers"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/consul"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Scaling down instances", func() {
	var (
		manifest  consul.Manifest
		kv        consulclient.HTTPKV
		testKey   string
		testValue string
		spammer   *helpers.Spammer
	)

	AfterEach(func() {
		if !CurrentGinkgoTestDescription().Failed {
			err := boshClient.DeleteDeployment(manifest.Name)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	Describe("scaling from 3 nodes to 1", func() {
		BeforeEach(func() {
			guid, err := helpers.NewGUID()
			Expect(err).NotTo(HaveOccurred())

			testKey = "consul-key-" + guid
			testValue = "consul-value-" + guid

			manifest, kv, err = helpers.DeployConsulWithInstanceCount(3, boshClient, config)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return boshClient.DeploymentVMs(manifest.Name)
			}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))
		})

		It("provides a functioning server after the scale down", func() {
			By("setting a persistent value to check the cluster is up", func() {
				err := kv.Set(testKey, testValue)
				Expect(err).NotTo(HaveOccurred())
			})

			By("scaling from 3 nodes to 1", func() {
				var err error
				manifest.Jobs[0], manifest.Properties, err = consul.SetJobInstanceCount(manifest.Jobs[0], manifest.Networks[0], manifest.Properties, 1)
				Expect(err).NotTo(HaveOccurred())

				members := manifest.ConsulMembers()
				Expect(members).To(HaveLen(4))

				yaml, err := manifest.ToYAML()
				Expect(err).NotTo(HaveOccurred())

				_, err = boshClient.Deploy(yaml)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() ([]bosh.VM, error) {
					return boshClient.DeploymentVMs(manifest.Name)
				}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))
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

			manifest, kv, err = helpers.DeployConsulWithInstanceCount(5, boshClient, config)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return boshClient.DeploymentVMs(manifest.Name)
			}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))

			spammer = helpers.NewSpammer(kv, 1*time.Second, "test-consumer-0")
		})

		It("persists data throughout the scale down", func() {
			By("setting a persistent value", func() {
				err := kv.Set(testKey, testValue)
				Expect(err).NotTo(HaveOccurred())
			})

			By("scaling from 5 nodes to 3", func() {
				var err error
				manifest.Jobs[0], manifest.Properties, err = consul.SetJobInstanceCount(manifest.Jobs[0], manifest.Networks[0], manifest.Properties, 3)
				Expect(err).NotTo(HaveOccurred())

				members := manifest.ConsulMembers()
				Expect(members).To(HaveLen(6))

				yaml, err := manifest.ToYAML()
				Expect(err).NotTo(HaveOccurred())

				spammer.Spam()

				_, err = boshClient.Deploy(yaml)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() ([]bosh.VM, error) {
					return boshClient.DeploymentVMs(manifest.Name)
				}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))

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

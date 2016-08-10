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

var _ = Describe("Scaling up Instances", func() {
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

	Describe("scaling from 1 node to 3", func() {
		BeforeEach(func() {
			guid, err := helpers.NewGUID()
			Expect(err).NotTo(HaveOccurred())

			testKey = "consul-key-" + guid
			testValue = "consul-value-" + guid

			manifest, kv, err = helpers.DeployConsulWithInstanceCount(1, boshClient, config)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return boshClient.DeploymentVMs(manifest.Name)
			}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))
		})

		It("provides a functioning cluster after the scale up", func() {
			By("setting a persistent value to check the cluster is up", func() {
				err := kv.Set(testKey, testValue)
				Expect(err).NotTo(HaveOccurred())
			})

			By("scaling from 1 nodes to 3", func() {
				var err error
				manifest.Jobs[0], manifest.Properties, err = consul.SetJobInstanceCount(manifest.Jobs[0], manifest.Networks[0], manifest.Properties, 3)
				Expect(err).NotTo(HaveOccurred())

				members := manifest.ConsulMembers()
				Expect(members).To(HaveLen(6))

				yaml, err := manifest.ToYAML()
				Expect(err).NotTo(HaveOccurred())

				_, err = boshClient.Deploy(yaml)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() ([]bosh.VM, error) {
					return boshClient.DeploymentVMs(manifest.Name)
				}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))
			})

			By("setting a persistent value to check the cluster is up after the scale up", func() {
				err := kv.Set(testKey, testValue)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("scaling from 3 nodes to 5", func() {
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

			spammer = helpers.NewSpammer(kv, 1*time.Second, "test-consumer-0")
		})

		It("persists data throughout the scale up", func() {
			By("setting a persistent value", func() {
				err := kv.Set(testKey, testValue)
				Expect(err).NotTo(HaveOccurred())
			})

			By("scaling from 3 nodes to 5", func() {
				var err error
				manifest.Jobs[0], manifest.Properties, err = consul.SetJobInstanceCount(manifest.Jobs[0], manifest.Networks[0], manifest.Properties, 5)
				Expect(err).NotTo(HaveOccurred())

				members := manifest.ConsulMembers()
				Expect(members).To(HaveLen(8))

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

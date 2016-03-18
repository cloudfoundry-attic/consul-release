package turbulence_test

import (
	"math/rand"
	"sync"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/consul"
	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/helpers"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("KillVm", func() {
	var (
		consulManifest destiny.Manifest
		kv             consul.HTTPKV

		keyVals   map[string]string
		testKey   string
		testValue string
	)

	BeforeEach(func() {
		guid, err := helpers.NewGUID()
		Expect(err).NotTo(HaveOccurred())

		testKey = "consul-key-" + guid
		testValue = "consul-value-" + guid

		consulManifest, kv, err = helpers.DeployConsulWithInstanceCount(3, client, config)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return client.DeploymentVMs(consulManifest.Name)
		}, "1m", "10s").Should(ConsistOf([]bosh.VM{
			{"running"},
			{"running"},
			{"running"},
			{"running"},
		}))
	})

	AfterEach(func() {
		By("fixing the deployment", func() {
			yaml, err := consulManifest.ToYAML()
			Expect(err).NotTo(HaveOccurred())

			err = client.ScanAndFix(yaml)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs(consulManifest.Name)
			}, "1m", "10s").Should(ConsistOf([]bosh.VM{
				{"running"},
				{"running"},
				{"running"},
				{"running"},
			}))
		})

		By("deleting the deployment", func() {
			if !CurrentGinkgoTestDescription().Failed {
				err := client.DeleteDeployment(consulManifest.Name)
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})

	Context("when a consul node is killed", func() {
		It("is still able to function on healthy vms", func() {
			By("setting a persistent value", func() {
				err := kv.Set(testKey, testValue)
				Expect(err).NotTo(HaveOccurred())
			})

			By("killing indices", func() {
				var wg sync.WaitGroup
				done := make(chan struct{})
				keyVals = make(map[string]string)

				keysChan := helpers.SpamConsul(done, &wg, kv)

				err := turbulenceClient.KillIndices(consulManifest.Name, "consul_z1", []int{rand.Intn(3)})
				Expect(err).ToNot(HaveOccurred())

				yaml, err := consulManifest.ToYAML()
				Expect(err).NotTo(HaveOccurred())

				err = client.ScanAndFix(yaml)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() ([]bosh.VM, error) {
					return client.DeploymentVMs(consulManifest.Name)
				}, "1m", "10s").Should(ConsistOf([]bosh.VM{
					{"running"},
					{"running"},
					{"running"},
					{"running"},
				}))

				close(done)

				wg.Wait()
				keyVals = <-keysChan

				if err, ok := keyVals["error"]; ok {
					Fail(err)
				}
			})

			By("reading the value from consul", func() {
				value, err := kv.Get(testKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(value).To(Equal(testValue))

				for key, value := range keyVals {
					v, err := kv.Get(key)
					Expect(err).ToNot(HaveOccurred())
					Expect(v).To(Equal(value))
				}
			})
		})
	})
})

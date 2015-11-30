package turbulence_test

import (
	"acceptance-tests/helpers"
	"acceptance-tests/turbulence/client"
	"time"

	capi "github.com/hashicorp/consul/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("KillVm", func() {
	var (
		runner         *helpers.AgentRunner
		consulManifest *helpers.Manifest
		turbulenceUrl  string

		consulClientIPs  []string
		killedConsulUrls []string
		aliveConsulUrls  []string
	)

	BeforeEach(func() {
		turbulenceUrl = "https://turbulence:" + turbulenceManifest.Properties.TurbulenceApi.Password + "@" + turbulenceManifest.Jobs[0].Networks[0].StaticIps[0] + ":8080"

		By("generating consul manifest", func() {
			consulManifest = new(helpers.Manifest)
			bosh.GenerateAndSetDeploymentManifest(
				consulManifest,
				consulManifestGeneration,
				directorUUIDStub,
				helpers.InstanceCount3NodesStubPath,
				helpers.PersistentDiskStubPath,
				config.IAASSettingsConsulStubPath,
				helpers.PropertyOverridesStubPath,
				consulNameOverrideStub,
			)
		})

		By("deploying", func() {
			Expect(bosh.Command("-n", "deploy")).To(Exit(0))
			Expect(len(consulManifest.Properties.Consul.Agent.Servers.Lans)).To(Equal(3))

		})

		By("running a consul agent locally", func() {
			for _, elem := range consulManifest.Properties.Consul.Agent.Servers.Lans {
				consulClientIPs = append(consulClientIPs, elem)
			}

			aliveConsulUrls = []string{
				"http://" + consulManifest.Jobs[1].Networks[0].StaticIps[0] + ":4001",
				"http://" + consulManifest.Jobs[1].Networks[0].StaticIps[1] + ":4001",
			}

			killedConsulUrls = []string{
				"http://" + consulManifest.Jobs[0].Networks[0].StaticIps[0] + ":4001",
			}

			runner = helpers.NewAgentRunner(consulClientIPs, config.BindAddress)
			runner.Start()
		})
	})

	AfterEach(func() {
		By("Stopping the agent runner", func() {
			runner.Stop()
		})

		By("Fixing the deployment with cloudcheck", func() {
			// It is as fast to fix the deployment with cloudcheck and subsequently
			// delete from a healthy state than to delete from an unhealthy state
			cckSession := bosh.Command("cloudcheck", "--auto")
			Eventually(cckSession, 1*time.Minute, 1*time.Second).Should(Exit(0))
		})

		By("delete deployment", func() {
			deleteSession := bosh.Command("-n", "delete", "deployment", consulDeployment)
			Eventually(deleteSession, 5*time.Minute, 5*time.Second).Should(Exit(0))
		})
	})

	Context("When a consul node is killed", func() {
		It("is still able to function on healthy vms", func() {
			consulClient := runner.NewClient()
			keyValueClient := consulClient.KV()
			consatsKey := "consats-key"
			consatsValue := []byte("consats-value")
			turbulenceOperationTimeout := helpers.GetTurbulenceOperationTimeout(config)
			turbulenceClient := client.NewClient(turbulenceUrl, turbulenceOperationTimeout)

			By("Putting key-value pair", func() {
				pair := &capi.KVPair{Key: consatsKey, Value: consatsValue}
				Eventually(func() error {
					_, err := keyValueClient.Put(pair, nil)
					return err
				}, 30*time.Second, 5*time.Second).Should(Succeed())
			})

			By("Killing indices", func() {
				err := turbulenceClient.KillIndices(consulDeployment, "consul_z1", []int{0})
				Expect(err).ToNot(HaveOccurred())
			})

			By("Checking for eventual consistency", func() {
				Eventually(func() ([]byte, error) {
					By("trying to get key")
					resultPair, _, err := keyValueClient.Get(consatsKey, nil)
					if resultPair == nil {
						return nil, err
					}
					return resultPair.Value, err
				}, 30*time.Second, 5*time.Second).Should(Equal(consatsValue))
			})
		})
	})
})

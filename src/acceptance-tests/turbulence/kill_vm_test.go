package turbulence_test

import (
	"acceptance-tests/helpers"
	"acceptance-tests/turbulence/client"

	capi "github.com/hashicorp/consul/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	"fmt"
)

var _ = Describe("KillVm", func() {
	var (
		runner          *helpers.AgentRunner
		consulManifest = new(helpers.Manifest)
		turbulenceUrl string

		consulClientIPs []string
		killedConsulUrls []string
		aliveConsulUrls  []string
	)

	BeforeEach(func() {
		turbulenceUrl = "https://turbulence:" + turbulenceManifest.Properties.TurbulenceApi.Password + "@" + turbulenceManifest.Jobs[0].Networks[0].StaticIps[0] + ":8080"

		By("generating consul manifest")
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

		By("deploying")
		Expect(bosh.Command("-n", "deploy")).To(Exit(0))
		Expect(len(consulManifest.Properties.Consul.Agent.Servers.Lans)).To(Equal(3))

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

	AfterEach(func() {
		fmt.Println("AfterEach")
		By("Fixing the release")
		bosh.Command("cck", "--auto")

		By("delete deployment")
		bosh.Command("-n", "delete", "deployment", consulDeployment)
	})

	Context("When a consul node is killed", func() {
		It("Is still able to function on healthy vms", func() {
			consulClient := runner.NewClient()

			consatsKey := "consats-key"
			consatsValue := []byte("consats-value")

			keyValueClient := consulClient.KV()

			pair := &capi.KVPair{Key: consatsKey, Value: consatsValue}
			fmt.Println("Got keyvalue")
			_, err := keyValueClient.Put(pair, nil)
			Expect(err).ToNot(HaveOccurred())

			fmt.Println("Starting consistently")
			consistent := make(chan int)
			go func() {
				defer GinkgoRecover()
				Consistently(func() ([]byte, error) {
					fmt.Println("Consistent ping")
					resultPair, _, err := keyValueClient.Get(consatsKey, nil)
					if err != nil {
						fmt.Printf("Error is %s\n", err)
					}
					fmt.Printf("Result pair is %s\n", resultPair.Value)
					return resultPair.Value, err
				}, 10, 1).Should(Equal(consatsValue))
				close(consistent)
			} ()

			turbulenceOperationTimeout := helpers.GetTurbulenceOperationTimeout(config)
			turbulenceClient := client.NewClient(turbulenceUrl, turbulenceOperationTimeout)

			fmt.Println("Killing indices")
			err = turbulenceClient.KillIndices(consulDeployment, "consul_z1", []int{0})
			Expect(err).ToNot(HaveOccurred())
			fmt.Println("Waiting test to finish")
			<- consistent
			fmt.Println("Exit test")
		})
	})
})

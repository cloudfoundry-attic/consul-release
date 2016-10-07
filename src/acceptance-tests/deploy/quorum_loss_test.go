package deploy_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/consulclient"
	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/helpers"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/consul"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = PDescribe("quorum loss", func() {
	var (
		consulManifest consul.Manifest
		kv             consulclient.HTTPKV
	)

	BeforeEach(func() {
		var err error
		consulManifest, kv, err = helpers.DeployConsulWithInstanceCount("quorum-loss", 5, boshClient, config)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return helpers.DeploymentVMs(boshClient, consulManifest.Name)
		}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(consulManifest)))
	})

	AfterEach(func() {
		By("deleting the deployment", func() {
			if !CurrentGinkgoTestDescription().Failed {
				for i := 0; i < 5; i++ {
					err := boshClient.SetVMResurrection(consulManifest.Name, "consul_z1", i, true)
					Expect(err).NotTo(HaveOccurred())
				}

				yaml, err := consulManifest.ToYAML()
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() error {
					return boshClient.ScanAndFixAll(yaml)
				}, "5m", "1m").ShouldNot(HaveOccurred())

				Eventually(func() ([]bosh.VM, error) {
					return helpers.DeploymentVMs(boshClient, consulManifest.Name)
				}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(consulManifest)))

				err = boshClient.DeleteDeployment(consulManifest.Name)
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})

	Context("when a consul node is killed", func() {
		It("is still able to function on healthy vms", func() {
			By("setting and getting a value", func() {
				guid, err := helpers.NewGUID()
				Expect(err).NotTo(HaveOccurred())
				testKey := "consul-key-" + guid
				testValue := "consul-value-" + guid

				err = kv.Set(testKey, testValue)
				Expect(err).NotTo(HaveOccurred())

				value, err := kv.Get(testKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(value).To(Equal(testValue))
			})

			By("killing indices", func() {
				for i := 0; i < 5; i++ {
					err := boshClient.SetVMResurrection(consulManifest.Name, "consul_z1", i, false)
					Expect(err).NotTo(HaveOccurred())
				}

				leader, err := jobIndexOfLeader(kv, boshClient, consulManifest.Name)
				Expect(err).ToNot(HaveOccurred())

				rand.Seed(time.Now().Unix())
				startingIndex := rand.Intn(3)
				instances := []int{startingIndex, startingIndex + 1, startingIndex + 2}

				if leader < startingIndex || leader > startingIndex+2 {
					instances[0] = leader
				}

				jobIndexToResurrect := startingIndex + 1

				err = turbulenceClient.KillIndices(consulManifest.Name, "consul_z1", instances)
				Expect(err).NotTo(HaveOccurred())

				err = boshClient.SetVMResurrection(consulManifest.Name, "consul_z1", jobIndexToResurrect, true)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() error {
					return boshClient.ScanAndFix(consulManifest.Name, "consul_z1", []int{jobIndexToResurrect})
				}, "5m", "1m").ShouldNot(HaveOccurred())

				Eventually(func() ([]bosh.VM, error) {
					return helpers.DeploymentVMs(boshClient, consulManifest.Name)
				}, "5m", "1m").Should(ContainElement(bosh.VM{JobName: "consul_z1", Index: jobIndexToResurrect, State: "running"}))
			})

			By("setting and getting a new value", func() {
				guid, err := helpers.NewGUID()
				Expect(err).NotTo(HaveOccurred())
				testKey := "consul-key-" + guid
				testValue := "consul-value-" + guid

				err = kv.Set(testKey, testValue)
				Expect(err).NotTo(HaveOccurred())

				value, err := kv.Get(testKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(value).To(Equal(testValue))
			})
		})
	})
})

func jobIndexOfLeader(kv consulclient.HTTPKV, client bosh.Client, deploymentName string) (int, error) {
	resp, err := http.Get(fmt.Sprintf("%s/v1/status/leader", kv.Address()))
	if err != nil {
		return -1, err
	}

	var leader string
	if err := json.NewDecoder(resp.Body).Decode(&leader); err != nil {
		return -1, err
	}

	vms, err := client.DeploymentVMs(deploymentName)
	if err != nil {
		return -1, err
	}

	for _, vm := range vms {
		if len(vm.IPs) > 0 {
			if vm.IPs[0] == strings.Split(leader, ":")[0] {
				return vm.Index, nil
			}
		}
	}

	return -1, errors.New("could not determine leader")
}

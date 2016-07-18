package deploy_test

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/consulclient"
	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/helpers"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/consul"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	TIMEOUT_ERROR_COUNT_THRESHOLD = 1
)

var _ = PDescribe("Migrate instance groups", func() {
	var (
		manifest consul.Manifest
		kv       consulclient.HTTPKV
		spammers []*helpers.Spammer
	)

	AfterEach(func() {
		if !CurrentGinkgoTestDescription().Failed {
			err := client.DeleteDeployment(manifest.Name)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	Describe("when migrating two instance groups from different AZs to a multi-AZ single instance group", func() {
		It("deploys successfully with minimal interruption", func() {
			By("deploying 3 node cluster across two AZs with BOSH 1.0 manifest", func() {
				var err error
				manifest, err = helpers.DeployMultiAZConsul(client, config)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() ([]bosh.VM, error) {
					return client.DeploymentVMs(manifest.Name)
				}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))

				for i, ip := range manifest.Jobs[2].Networks[0].StaticIPs {
					kv = consulclient.NewHTTPKV(fmt.Sprintf("http://%s:6769", ip))
					spammers = append(spammers, helpers.NewSpammer(kv, 1*time.Second, fmt.Sprintf("test-consumer-%d", i)))
				}
			})

			By("starting spammer", func() {
				for _, spammer := range spammers {
					spammer.Spam()
				}
			})

			By("deploying 3 node cluster across two AZs with BOSH 2.0 manifest", func() {
				err := helpers.UpdateCloudConfig(client, config)
				Expect(err).NotTo(HaveOccurred())

				manifestv2, err := helpers.DeployMultiAZConsulMigration(client, config, manifest.Name)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() ([]bosh.VM, error) {
					return client.DeploymentVMs(manifestv2.Name)
				}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifestV2(manifestv2)))
			})

			By("verifying keys are accounted for in cluster", func() {
				for _, spammer := range spammers {
					spammer.Stop()
					spammerErrs := spammer.Check()

					var errorSet helpers.ErrorSet

					switch spammerErrs.(type) {
					case helpers.ErrorSet:
						errorSet = spammerErrs.(helpers.ErrorSet)
					case nil:
						continue
					default:
						Fail(spammerErrs.Error())
					}

					timeoutErrCount := 0
					otherErrors := helpers.ErrorSet{}

					for err, occurrences := range errorSet {
						switch {
						// This happens when the testconsumer gets rolled when a connection is alive
						case strings.Contains(err, "getsockopt: operation timed out"):
							timeoutErrCount += occurrences
						default:
							otherErrors.Add(errors.New(err))
						}
					}

					Expect(otherErrors).To(HaveLen(0))
					Expect(timeoutErrCount).To(BeNumerically("<=", TIMEOUT_ERROR_COUNT_THRESHOLD))
				}
			})
		})
	})
})

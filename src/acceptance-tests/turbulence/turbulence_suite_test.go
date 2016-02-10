package turbulence_test

import (
	"time"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/helpers"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/bosh-test/turbulence"
	"github.com/pivotal-cf-experimental/destiny"

	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestTurbulence(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "turbulence")
}

var (
	config helpers.Config
	client bosh.Client

	turbulenceManifest destiny.Manifest
	turbulenceClient   turbulence.Client
)

var _ = BeforeSuite(func() {
	configPath, err := helpers.ConfigPath()
	Expect(err).NotTo(HaveOccurred())

	config, err = helpers.LoadConfig(configPath)
	Expect(err).NotTo(HaveOccurred())

	client = bosh.NewClient(bosh.Config{
		URL:              fmt.Sprintf("https://%s:25555", config.BOSH.Target),
		Username:         config.BOSH.Username,
		Password:         config.BOSH.Password,
		AllowInsecureSSL: true,
	})

	By("deploying turbulence", func() {
		info, err := client.Info()
		Expect(err).NotTo(HaveOccurred())

		guid, err := helpers.NewGUID()
		Expect(err).NotTo(HaveOccurred())

		manifestConfig := destiny.Config{
			DirectorUUID: info.UUID,
			Name:         "turbulence-consul-" + guid,
			BOSH: destiny.ConfigBOSH{
				Target:         config.BOSH.Target,
				Username:       config.BOSH.Username,
				Password:       config.BOSH.Password,
				DirectorCACert: config.BOSH.DirectorCACert,
			},
		}

		switch info.CPI {
		case "aws_cpi":
			manifestConfig.IAAS = destiny.AWS

			if config.AWS.Subnet == "" {
				Fail("aws.subnet is required for AWS IAAS deployment")
			}

			manifestConfig.IPRange = "10.0.4.0/24"
			manifestConfig.AWS = destiny.ConfigAWS{
				AccessKeyID:           config.AWS.AccessKeyID,
				SecretAccessKey:       config.AWS.SecretAccessKey,
				DefaultKeyName:        config.AWS.DefaultKeyName,
				DefaultSecurityGroups: config.AWS.DefaultSecurityGroups,
				Region:                config.AWS.Region,
				Subnet:                config.AWS.Subnet,
			}
			manifestConfig.Registry = destiny.ConfigRegistry{
				Host:     config.Registry.Host,
				Port:     config.Registry.Port,
				Username: config.Registry.Username,
				Password: config.Registry.Password,
			}
		case "warden_cpi":
			manifestConfig.IPRange = "10.244.4.0/24"
			manifestConfig.IAAS = destiny.Warden
		default:
			Fail("unknown infrastructure type")
		}

		turbulenceManifest = destiny.NewTurbulence(manifestConfig)

		yaml, err := turbulenceManifest.ToYAML()
		Expect(err).NotTo(HaveOccurred())

		yaml, err = client.ResolveManifestVersions(yaml)
		Expect(err).NotTo(HaveOccurred())

		turbulenceManifest, err = destiny.FromYAML(yaml)
		Expect(err).NotTo(HaveOccurred())

		err = client.Deploy(yaml)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return client.DeploymentVMs(turbulenceManifest.Name)
		}, "1m", "10s").Should(ConsistOf([]bosh.VM{
			{"running"},
		}))
	})

	By("preparing turbulence client", func() {
		turbulenceUrl := fmt.Sprintf("https://turbulence:%s@%s:8080",
			turbulenceManifest.Properties.TurbulenceAPI.Password,
			turbulenceManifest.Jobs[0].Networks[0].StaticIPs[0])

		turbulenceClient = turbulence.NewClient(turbulenceUrl, 5*time.Minute, 2*time.Second)
	})
})

var _ = AfterSuite(func() {
	By("deleting the turbulence deployment", func() {
		if !CurrentGinkgoTestDescription().Failed {
			err := client.DeleteDeployment(turbulenceManifest.Name)
			Expect(err).NotTo(HaveOccurred())
		}
	})
})

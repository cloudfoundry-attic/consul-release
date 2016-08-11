package turbulence_test

import (
	"time"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/helpers"
	ginkgoConfig "github.com/onsi/ginkgo/config"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	turbulenceclient "github.com/pivotal-cf-experimental/bosh-test/turbulence"
	"github.com/pivotal-cf-experimental/destiny/iaas"
	"github.com/pivotal-cf-experimental/destiny/turbulence"

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
})

func getVMsFromManifest(manifest turbulence.Manifest) []bosh.VM {
	var vms []bosh.VM

	for _, job := range manifest.Jobs {
		for i := 0; i < job.Instances; i++ {
			vms = append(vms, bosh.VM{JobName: job.Name, Index: i, State: "running"})

		}
	}

	return vms
}

func newTurbulenceClient(manifest turbulence.Manifest) turbulenceclient.Client {
	turbulenceUrl := fmt.Sprintf("https://turbulence:%s@%s:8080",
		manifest.Properties.TurbulenceAPI.Password,
		manifest.Jobs[0].Networks[0].StaticIPs[0])

	return turbulenceclient.NewClient(turbulenceUrl, 5*time.Minute, 2*time.Second)
}

func deployTurbulence() turbulence.Manifest {
	info, err := client.Info()
	Expect(err).NotTo(HaveOccurred())

	guid, err := helpers.NewGUID()
	Expect(err).NotTo(HaveOccurred())

	manifestConfig := turbulence.Config{
		DirectorUUID: info.UUID,
		Name:         "turbulence-consul-" + guid,
		BOSH: turbulence.ConfigBOSH{
			Target:         config.BOSH.Target,
			Username:       config.BOSH.Username,
			Password:       config.BOSH.Password,
			DirectorCACert: config.BOSH.DirectorCACert,
		},
	}

	var iaasConfig iaas.Config
	switch info.CPI {
	case "aws_cpi":
		awsConfig := iaas.AWSConfig{
			AccessKeyID:           config.AWS.AccessKeyID,
			SecretAccessKey:       config.AWS.SecretAccessKey,
			DefaultKeyName:        config.AWS.DefaultKeyName,
			DefaultSecurityGroups: config.AWS.DefaultSecurityGroups,
			Region:                config.AWS.Region,
			RegistryHost:          config.Registry.Host,
			RegistryPassword:      config.Registry.Password,
			RegistryPort:          config.Registry.Port,
			RegistryUsername:      config.Registry.Username,
		}

		if len(config.AWS.Subnets) > 0 {
			subnet := config.AWS.Subnets[0]

			var cidrBlock string
			cidrPool := helpers.NewCIDRPool(subnet.Range, 24, 27)
			cidrBlock, err = cidrPool.Get(ginkgoConfig.GinkgoConfig.ParallelNode - 1)
			if err != nil {
				Fail(err.Error())
			}

			awsConfig.Subnets = append(awsConfig.Subnets, iaas.AWSConfigSubnet{ID: subnet.ID, Range: cidrBlock, AZ: subnet.AZ})
			manifestConfig.IPRange = cidrBlock
		} else {
			Fail("aws.subnet is required for AWS IAAS deployment")
		}

		iaasConfig = awsConfig
	case "warden_cpi":
		var cidrBlock string
		cidrPool := helpers.NewCIDRPool("10.244.4.0", 24, 27)
		cidrBlock, err = cidrPool.Get(ginkgoConfig.GinkgoConfig.ParallelNode - 1)
		if err != nil {
			Fail(err.Error())
		}

		manifestConfig.IPRange = cidrBlock
		iaasConfig = iaas.NewWardenConfig()
	default:
		Fail("unknown infrastructure type")
	}

	turbulenceManifest, err := turbulence.NewManifest(manifestConfig, iaasConfig)
	Expect(err).NotTo(HaveOccurred())

	yaml, err := turbulenceManifest.ToYAML()
	Expect(err).NotTo(HaveOccurred())

	yaml, err = client.ResolveManifestVersions(yaml)
	Expect(err).NotTo(HaveOccurred())

	turbulenceManifest, err = turbulence.FromYAML(yaml)
	Expect(err).NotTo(HaveOccurred())

	_, err = client.Deploy(yaml)
	Expect(err).NotTo(HaveOccurred())

	Eventually(func() ([]bosh.VM, error) {
		return client.DeploymentVMs(turbulenceManifest.Name)
	}, "1m", "10s").Should(ConsistOf(getVMsFromManifest(turbulenceManifest)))

	return turbulenceManifest
}

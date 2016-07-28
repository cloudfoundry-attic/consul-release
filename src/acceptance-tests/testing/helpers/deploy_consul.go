package helpers

import (
	"errors"
	"fmt"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/consulclient"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/cloudconfig"
	"github.com/pivotal-cf-experimental/destiny/consul"
	"github.com/pivotal-cf-experimental/destiny/iaas"

	ginkgoConfig "github.com/onsi/ginkgo/config"
)

func DeployConsulWithJobLevelConsulProperties(client bosh.Client, config Config) (manifest consul.Manifest, err error) {
	guid, err := NewGUID()
	if err != nil {
		return
	}

	info, err := client.Info()
	if err != nil {
		return
	}

	manifestConfig := consul.Config{
		DirectorUUID: info.UUID,
		Name:         fmt.Sprintf("consul-%s", guid),
	}

	var iaasConfig iaas.Config
	switch info.CPI {
	case "aws_cpi":
		awsConfig := buildAWSConfig(config)
		if len(config.AWS.Subnets) > 0 {
			subnet := config.AWS.Subnets[0]

			awsConfig.Subnets = append(awsConfig.Subnets, iaas.AWSConfigSubnet{ID: subnet.ID, Range: subnet.Range, AZ: subnet.AZ})
			manifestConfig.Networks = append(manifestConfig.Networks, consul.ConfigNetwork{IPRange: subnet.Range, Nodes: 1})
		} else {
			err = errors.New("AWSSubnet is required for AWS IAAS deployment")
			return
		}

		iaasConfig = awsConfig
	case "warden_cpi":
		iaasConfig = iaas.NewWardenConfig()

		var cidrBlock string
		cidrPool := NewCIDRPool("10.244.4.0", 24, 26)
		cidrBlock, err = cidrPool.Get(ginkgoConfig.GinkgoConfig.ParallelNode - 1)
		if err != nil {
			return
		}

		manifestConfig.Networks = []consul.ConfigNetwork{
			{
				IPRange: cidrBlock,
				Nodes:   1,
			},
		}
	default:
		err = errors.New("unknown infrastructure type")
		return
	}

	manifest, err = consul.NewManifestWithJobLevelProperties(manifestConfig, iaasConfig)
	if err != nil {
		return
	}

	yaml, err := manifest.ToYAML()
	if err != nil {
		return
	}

	yaml, err = client.ResolveManifestVersions(yaml)
	if err != nil {
		return
	}

	err = consul.FromYAML(yaml, &manifest)
	if err != nil {
		return
	}

	_, err = client.Deploy(yaml)
	if err != nil {
		return
	}

	return
}

func DeployConsulWithInstanceCount(count int, client bosh.Client, config Config) (manifest consul.Manifest, kv consulclient.HTTPKV, err error) {
	guid, err := NewGUID()
	if err != nil {
		return
	}

	info, err := client.Info()
	if err != nil {
		return
	}

	manifestConfig := consul.Config{
		DirectorUUID: info.UUID,
		Name:         fmt.Sprintf("consul-%s", guid),
	}

	var iaasConfig iaas.Config
	switch info.CPI {
	case "aws_cpi":
		awsConfig := buildAWSConfig(config)
		if len(config.AWS.Subnets) > 0 {
			subnet := config.AWS.Subnets[0]

			awsConfig.Subnets = append(awsConfig.Subnets, iaas.AWSConfigSubnet{ID: subnet.ID, Range: subnet.Range, AZ: subnet.AZ})
			manifestConfig.Networks = append(manifestConfig.Networks, consul.ConfigNetwork{IPRange: subnet.Range, Nodes: count})
		} else {
			err = errors.New("AWSSubnet is required for AWS IAAS deployment")
			return
		}

		iaasConfig = awsConfig
	case "warden_cpi":
		iaasConfig = iaas.NewWardenConfig()

		var cidrBlock string
		cidrPool := NewCIDRPool("10.244.4.0", 24, 26)
		cidrBlock, err = cidrPool.Get(ginkgoConfig.GinkgoConfig.ParallelNode - 1)
		if err != nil {
			return
		}

		manifestConfig.Networks = []consul.ConfigNetwork{
			{
				IPRange: cidrBlock,
				Nodes:   count,
			},
		}
	default:
		err = errors.New("unknown infrastructure type")
		return
	}

	manifest, err = consul.NewManifest(manifestConfig, iaasConfig)
	if err != nil {
		return
	}

	yaml, err := manifest.ToYAML()
	if err != nil {
		return
	}

	yaml, err = client.ResolveManifestVersions(yaml)
	if err != nil {
		return
	}

	err = consul.FromYAML(yaml, &manifest)
	if err != nil {
		return
	}

	_, err = client.Deploy(yaml)
	if err != nil {
		return
	}

	kv = consulclient.NewHTTPKV(fmt.Sprintf("http://%s:6769", manifest.Jobs[1].Networks[0].StaticIPs[0]))
	return
}

func DeployMultiAZConsul(client bosh.Client, config Config) (manifest consul.Manifest, err error) {
	guid, err := NewGUID()
	if err != nil {
		return
	}

	info, err := client.Info()
	if err != nil {
		return
	}

	manifestConfig := consul.Config{
		DirectorUUID: info.UUID,
		Name:         fmt.Sprintf("consul-%s", guid),
	}

	var iaasConfig iaas.Config
	switch info.CPI {
	case "aws_cpi":
		awsConfig := buildAWSConfig(config)
		if len(config.AWS.Subnets) >= 2 {
			subnet := config.AWS.Subnets[0]
			awsConfig.Subnets = append(awsConfig.Subnets, iaas.AWSConfigSubnet{ID: subnet.ID, Range: subnet.Range, AZ: subnet.AZ})
			manifestConfig.Networks = append(manifestConfig.Networks, consul.ConfigNetwork{IPRange: subnet.Range, Nodes: 2})

			subnet = config.AWS.Subnets[1]
			awsConfig.Subnets = append(awsConfig.Subnets, iaas.AWSConfigSubnet{ID: subnet.ID, Range: subnet.Range, AZ: subnet.AZ})
			manifestConfig.Networks = append(manifestConfig.Networks, consul.ConfigNetwork{IPRange: subnet.Range, Nodes: 1})
		} else {
			err = errors.New("AWSSubnet is required for AWS IAAS deployment")
			return
		}

		iaasConfig = awsConfig
	case "warden_cpi":
		iaasConfig = iaas.NewWardenConfig()

		var cidrBlock string
		cidrPool := NewCIDRPool("10.244.4.0", 24, 26)
		cidrBlock, err = cidrPool.Get(ginkgoConfig.GinkgoConfig.ParallelNode - 1)
		if err != nil {
			return
		}

		var cidrBlock2 string
		cidrPool2 := NewCIDRPool("10.244.5.0", 24, 26)
		cidrBlock2, err = cidrPool2.Get(ginkgoConfig.GinkgoConfig.ParallelNode - 1)
		if err != nil {
			return
		}

		manifestConfig.Networks = []consul.ConfigNetwork{
			{IPRange: cidrBlock, Nodes: 2},
			{IPRange: cidrBlock2, Nodes: 1},
		}
	default:
		err = errors.New("unknown infrastructure type")
		return
	}

	manifest, err = consul.NewManifest(manifestConfig, iaasConfig)
	if err != nil {
		return
	}

	yaml, err := manifest.ToYAML()
	if err != nil {
		return
	}

	yaml, err = client.ResolveManifestVersions(yaml)
	if err != nil {
		return
	}

	err = consul.FromYAML(yaml, &manifest)
	if err != nil {
		return
	}

	_, err = client.Deploy(yaml)
	if err != nil {
		return
	}

	return
}

func DeployMultiAZConsulMigration(client bosh.Client, config Config, deploymentName string) (consul.ManifestV2, error) {
	info, err := client.Info()
	if err != nil {
		return consul.ManifestV2{}, err
	}

	manifestConfig := consul.ConfigV2{
		DirectorUUID: info.UUID,
		Name:         deploymentName,
	}

	var iaasConfig iaas.Config
	switch info.CPI {
	case "aws_cpi":
		awsConfig := buildAWSConfig(config)
		if len(config.AWS.Subnets) >= 2 {
			subnet := config.AWS.Subnets[0]
			awsConfig.Subnets = append(awsConfig.Subnets, iaas.AWSConfigSubnet{ID: subnet.ID, Range: subnet.Range, AZ: subnet.AZ})
			manifestConfig.AZs = append(manifestConfig.AZs, consul.ConfigAZ{Name: "z1", IPRange: subnet.Range, Nodes: 2})

			subnet = config.AWS.Subnets[1]
			awsConfig.Subnets = append(awsConfig.Subnets, iaas.AWSConfigSubnet{ID: subnet.ID, Range: subnet.Range, AZ: subnet.AZ})
			manifestConfig.AZs = append(manifestConfig.AZs, consul.ConfigAZ{Name: "z2", IPRange: subnet.Range, Nodes: 1})
		} else {
			return consul.ManifestV2{}, errors.New("AWSSubnet is required for AWS IAAS deployment")
		}

		iaasConfig = awsConfig
	case "warden_cpi":
		iaasConfig = iaas.NewWardenConfig()

		var cidrBlock string
		cidrPool := NewCIDRPool("10.244.4.0", 24, 26)
		cidrBlock, err = cidrPool.Get(ginkgoConfig.GinkgoConfig.ParallelNode - 1)
		if err != nil {
			return consul.ManifestV2{}, err
		}

		var cidrBlock2 string
		cidrPool2 := NewCIDRPool("10.244.5.0", 24, 26)
		cidrBlock2, err = cidrPool2.Get(ginkgoConfig.GinkgoConfig.ParallelNode - 1)
		if err != nil {
			return consul.ManifestV2{}, err
		}

		manifestConfig.AZs = []consul.ConfigAZ{
			{
				Name:    "z1",
				IPRange: cidrBlock,
				Nodes:   2,
			},
			{
				Name:    "z2",
				IPRange: cidrBlock2,
				Nodes:   1,
			},
		}
	default:
		return consul.ManifestV2{}, errors.New("unknown infrastructure type")
	}

	manifest := consul.NewManifestV2(manifestConfig, iaasConfig)

	manifestYAML, err := manifest.ToYAML()
	if err != nil {
		return consul.ManifestV2{}, err
	}

	_, err = client.Deploy(manifestYAML)
	if err != nil {
		return consul.ManifestV2{}, err
	}

	return manifest, nil
}

func UpdateCloudConfig(client bosh.Client, config Config) error {
	var cloudConfigOptions cloudconfig.Config

	info, err := client.Info()
	if err != nil {
		return err
	}

	switch info.CPI {
	case "aws_cpi":
		return nil
	case "warden_cpi":
		cloudConfigOptions.AZs = []cloudconfig.ConfigAZ{
			{IPRange: "10.244.4.0/24", StaticIPs: 11},
			{IPRange: "10.244.5.0/24", StaticIPs: 5},
		}
	default:
		return errors.New("unknown infrastructure type")
	}

	cloudConfig := cloudconfig.NewWardenCloudConfig(cloudConfigOptions)

	cloudConfigYAML, err := cloudConfig.ToYAML()
	if err != nil {
		return err
	}

	err = client.UpdateCloudConfig(cloudConfigYAML)
	if err != nil {
		return err
	}

	return nil
}

func buildAWSConfig(config Config) iaas.AWSConfig {
	return iaas.AWSConfig{
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
}

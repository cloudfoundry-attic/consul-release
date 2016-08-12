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

type ManifestGenerator func(consul.Config, iaas.Config) (consul.Manifest, error)

func DeployConsulWithInstanceCount(count int, client bosh.Client, config Config) (manifest consul.Manifest, kv consulclient.HTTPKV, err error) {
	return DeployConsulWithInstanceCountAndReleaseVersion(count, client, config, ConsulReleaseVersion())
}

func DeployConsulWithJobLevelConsulProperties(client bosh.Client, config Config) (manifest consul.Manifest, err error) {
	manifest, _, err = deployConsul(1, client, config, ConsulReleaseVersion(), consul.NewManifestWithJobLevelProperties)
	return
}

func DeployConsulWithInstanceCountAndReleaseVersion(count int, client bosh.Client, config Config, releaseVersion string) (manifest consul.Manifest, kv consulclient.HTTPKV, err error) {
	return deployConsul(count, client, config, releaseVersion, consul.NewManifest)
}

func deployConsul(count int, client bosh.Client, config Config, releaseVersion string, manifestGenerator ManifestGenerator) (manifest consul.Manifest, kv consulclient.HTTPKV, err error) {
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

			var cidrBlock string
			cidrPool := NewCIDRPool(subnet.Range, 24, 27)
			cidrBlock, err = cidrPool.Get(ginkgoConfig.GinkgoConfig.ParallelNode)
			if err != nil {
				return
			}

			awsConfig.Subnets = append(awsConfig.Subnets, iaas.AWSConfigSubnet{ID: subnet.ID, Range: cidrBlock, AZ: subnet.AZ})
			manifestConfig.Networks = append(manifestConfig.Networks, consul.ConfigNetwork{IPRange: cidrBlock, Nodes: count})
		} else {
			err = errors.New("AWSSubnet is required for AWS IAAS deployment")
			return
		}

		iaasConfig = awsConfig
	case "warden_cpi":
		iaasConfig = iaas.NewWardenConfig()

		var cidrBlock string
		cidrPool := NewCIDRPool("10.244.4.0", 24, 27)
		cidrBlock, err = cidrPool.Get(ginkgoConfig.GinkgoConfig.ParallelNode)
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

	manifest, err = manifestGenerator(manifestConfig, iaasConfig)
	if err != nil {
		return
	}

	for i := range manifest.Releases {
		if manifest.Releases[i].Name == "consul" {
			manifest.Releases[i].Version = releaseVersion
		}
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

	err = VerifyDeploymentRelease(client, manifestConfig.Name, releaseVersion)
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

			var cidrBlock string
			cidrPool := NewCIDRPool(subnet.Range, 24, 27)
			cidrBlock, err = cidrPool.Get(0)
			if err != nil {
				return
			}

			awsConfig.Subnets = append(awsConfig.Subnets, iaas.AWSConfigSubnet{ID: subnet.ID, Range: cidrBlock, AZ: subnet.AZ})
			manifestConfig.Networks = append(manifestConfig.Networks, consul.ConfigNetwork{IPRange: cidrBlock, Nodes: 2})

			subnet = config.AWS.Subnets[1]

			cidrPool = NewCIDRPool(subnet.Range, 24, 27)
			cidrBlock, err = cidrPool.Get(0)
			if err != nil {
				return
			}

			awsConfig.Subnets = append(awsConfig.Subnets, iaas.AWSConfigSubnet{ID: subnet.ID, Range: cidrBlock, AZ: subnet.AZ})
			manifestConfig.Networks = append(manifestConfig.Networks, consul.ConfigNetwork{IPRange: cidrBlock, Nodes: 1})
		} else {
			err = errors.New("AWSSubnet is required for AWS IAAS deployment")
			return
		}

		iaasConfig = awsConfig
	case "warden_cpi":
		iaasConfig = iaas.NewWardenConfig()

		var cidrBlock string
		cidrPool := NewCIDRPool("10.244.4.0", 24, 27)
		cidrBlock, err = cidrPool.Get(0)
		if err != nil {
			return
		}

		var cidrBlock2 string
		cidrPool2 := NewCIDRPool("10.244.5.0", 24, 27)
		cidrBlock2, err = cidrPool2.Get(0)
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

	for i := range manifest.Releases {
		if manifest.Releases[i].Name == "consul" {
			manifest.Releases[i].Version = ConsulReleaseVersion()
		}
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

	err = VerifyDeploymentRelease(client, manifestConfig.Name, ConsulReleaseVersion())
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

			var cidrBlock string
			cidrPool := NewCIDRPool(subnet.Range, 24, 27)
			cidrBlock, err = cidrPool.Get(0)
			if err != nil {
				return consul.ManifestV2{}, err
			}

			awsConfig.Subnets = append(awsConfig.Subnets, iaas.AWSConfigSubnet{ID: subnet.ID, Range: cidrBlock, AZ: subnet.AZ})
			manifestConfig.AZs = append(manifestConfig.AZs, consul.ConfigAZ{Name: "z1", IPRange: cidrBlock, Nodes: 2})

			subnet = config.AWS.Subnets[1]

			cidrPool = NewCIDRPool(subnet.Range, 24, 27)
			cidrBlock, err = cidrPool.Get(0)
			if err != nil {
				return consul.ManifestV2{}, err
			}

			awsConfig.Subnets = append(awsConfig.Subnets, iaas.AWSConfigSubnet{ID: subnet.ID, Range: cidrBlock, AZ: subnet.AZ})
			manifestConfig.AZs = append(manifestConfig.AZs, consul.ConfigAZ{Name: "z2", IPRange: cidrBlock, Nodes: 1})
		} else {
			return consul.ManifestV2{}, errors.New("AWSSubnet is required for AWS IAAS deployment")
		}

		iaasConfig = awsConfig
	case "warden_cpi":
		iaasConfig = iaas.NewWardenConfig()

		var cidrBlock string
		cidrPool := NewCIDRPool("10.244.4.0", 24, 27)
		cidrBlock, err = cidrPool.Get(0)
		if err != nil {
			return consul.ManifestV2{}, err
		}

		var cidrBlock2 string
		cidrPool2 := NewCIDRPool("10.244.5.0", 24, 27)
		cidrBlock2, err = cidrPool2.Get(0)
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

	for i := range manifest.Releases {
		if manifest.Releases[i].Name == "consul" {
			manifest.Releases[i].Version = ConsulReleaseVersion()
		}
	}

	manifestYAML, err := manifest.ToYAML()
	if err != nil {
		return consul.ManifestV2{}, err
	}

	_, err = client.Deploy(manifestYAML)
	if err != nil {
		return consul.ManifestV2{}, err
	}

	if err := VerifyDeploymentRelease(client, manifestConfig.Name, ConsulReleaseVersion()); err != nil {
		return consul.ManifestV2{}, err
	}

	return manifest, nil
}

func VerifyDeploymentRelease(client bosh.Client, deploymentName string, releaseVersion string) (err error) {
	deployments, err := client.Deployments()
	if err != nil {
		return
	}

	for _, deployment := range deployments {
		if deployment.Name == deploymentName {
			for _, release := range deployment.Releases {
				if release.Name == "consul" {
					switch {
					case len(release.Versions) > 1:
						err = errors.New("too many releases")
					case len(release.Versions) == 1 && release.Versions[0] != releaseVersion:
						err = fmt.Errorf("expected consul-release version %q but got %q", releaseVersion, release.Versions[0])
					}
				}
			}
		}
	}

	return
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
			{IPRange: "10.244.4.0/27", StaticIPs: 11},
			{IPRange: "10.244.5.0/27", StaticIPs: 5},
		}
	default:
		return errors.New("unknown infrastructure type")
	}

	cloudConfig, err := cloudconfig.NewWardenCloudConfig(cloudConfigOptions)
	if err != nil {
		return err
	}

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

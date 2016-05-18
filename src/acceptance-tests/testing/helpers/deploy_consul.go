package helpers

import (
	"errors"
	"fmt"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/consulclient"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/consul"
	"github.com/pivotal-cf-experimental/destiny/iaas"
)

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
		iaasConfig = iaas.AWSConfig{
			AccessKeyID:           config.AWS.AccessKeyID,
			SecretAccessKey:       config.AWS.SecretAccessKey,
			DefaultKeyName:        config.AWS.DefaultKeyName,
			DefaultSecurityGroups: config.AWS.DefaultSecurityGroups,
			Region:                config.AWS.Region,
			Subnet:                config.AWS.Subnet,
			RegistryHost:          config.Registry.Host,
			RegistryPassword:      config.Registry.Password,
			RegistryPort:          config.Registry.Port,
			RegistryUsername:      config.Registry.Username,
		}
		if config.AWS.Subnet != "" {
			manifestConfig.IPRange = "10.0.4.0/24"
		} else {
			err = errors.New("AWSSubnet is required for AWS IAAS deployment")
			return
		}
	case "warden_cpi":
		iaasConfig = iaas.NewWardenConfig()
		manifestConfig.IPRange = "10.244.4.0/24"
	default:
		err = errors.New("unknown infrastructure type")
		return
	}

	manifest = consul.NewManifest(manifestConfig, iaasConfig)

	manifest.Jobs[0], manifest.Properties = consul.SetJobInstanceCount(manifest.Jobs[0], manifest.Networks[0], manifest.Properties, count)

	yaml, err := manifest.ToYAML()
	if err != nil {
		return
	}

	yaml, err = client.ResolveManifestVersions(yaml)
	if err != nil {
		return
	}

	manifest, err = consul.FromYAML(yaml)
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

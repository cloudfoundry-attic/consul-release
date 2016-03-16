package helpers

import (
	"errors"
	"fmt"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/consul"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny"
)

func DeployConsulWithInstanceCount(count int, client bosh.Client, config Config) (manifest destiny.Manifest, kv consul.KV, err error) {
	guid, err := NewGUID()
	if err != nil {
		return
	}

	info, err := client.Info()
	if err != nil {
		return
	}

	manifestConfig := destiny.Config{
		DirectorUUID: info.UUID,
		Name:         fmt.Sprintf("consul-%s", guid),
	}

	switch info.CPI {
	case "aws_cpi":
		manifestConfig.IAAS = destiny.AWS
		if config.AWS.Subnet != "" {
			manifestConfig.AWS.Subnet = config.AWS.Subnet
			manifestConfig.IPRange = "10.0.4.0/24"
		} else {
			err = errors.New("AWSSubnet is required for AWS IAAS deployment")
			return
		}
	case "warden_cpi":
		manifestConfig.IPRange = "10.244.4.0/24"
		manifestConfig.IAAS = destiny.Warden
	default:
		err = errors.New("unknown infrastructure type")
		return
	}

	manifest = destiny.NewConsul(manifestConfig)

	manifest.Jobs[0], manifest.Properties = SetJobInstanceCount(manifest.Jobs[0], manifest.Networks[0], manifest.Properties, count)

	yaml, err := manifest.ToYAML()
	if err != nil {
		return
	}

	yaml, err = client.ResolveManifestVersions(yaml)
	if err != nil {
		return
	}

	manifest, err = destiny.FromYAML(yaml)
	if err != nil {
		return
	}

	err = client.Deploy(yaml)
	if err != nil {
		return
	}

	kv = consul.NewConsulClient(fmt.Sprintf("http://%s:6769", manifest.Jobs[1].Networks[0].StaticIPs[0]))
	return
}

func SetJobInstanceCount(job destiny.Job, network destiny.Network, properties destiny.Properties, count int) (destiny.Job, destiny.Properties) {
	job.Instances = count
	for i, net := range job.Networks {
		if net.Name == network.Name {
			net.StaticIPs = network.StaticIPs(count)
			properties.Consul.Agent.Servers.Lan = net.StaticIPs
		}
		job.Networks[i] = net
	}

	return job, properties
}

package helpers

import (
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/consul"
)

func GetVMsFromManifest(manifest consul.Manifest) []bosh.VM {
	var vms []bosh.VM

	for _, job := range manifest.Jobs {
		for i := 0; i < job.Instances; i++ {
			vms = append(vms, bosh.VM{JobName: job.Name, Index: i, State: "running"})

		}
	}

	return vms
}

func GetVMsFromManifestV2(manifest consul.ManifestV2) []bosh.VM {
	var vms []bosh.VM

	for _, ig := range manifest.InstanceGroups {
		for i := 0; i < ig.Instances; i++ {
			vms = append(vms, bosh.VM{JobName: ig.Name, Index: i, State: "running"})

		}
	}

	return vms
}

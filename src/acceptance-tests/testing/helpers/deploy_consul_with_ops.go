package helpers

import (
	"fmt"

	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/consulwithops"
	"github.com/pivotal-cf-experimental/destiny/ops"
)

func DeployConsulWithOpsWithInstanceCount(deploymentPrefix string, instanceCount int, boshClient bosh.Client) (string, error) {
	manifestName := fmt.Sprintf("consul-%s", deploymentPrefix)
	releaseVersion := ConsulReleaseVersion()

	info, err := boshClient.Info()
	if err != nil {
		return "", err
	}

	//TODO: AZs should be pulled from integration_config
	manifest, err := consulwithops.NewManifestV2(consulwithops.ConfigV2{
		DirectorUUID: info.UUID,
		Name:         manifestName,
		AZs:          []string{"z1", "z2"},
	})
	if err != nil {
		return "", err
	}

	manifest, err = ops.ApplyOp(manifest, ops.Op{
		Type:  "replace",
		Path:  "/releases/name=consul/version",
		Value: releaseVersion,
	})
	if err != nil {
		return "", err
	}

	manifest, err = ops.ApplyOp(manifest, ops.Op{
		Type:  "replace",
		Path:  "/instance_groups/name=consul/instances",
		Value: instanceCount,
	})
	if err != nil {
		return "", err
	}

	manifestYAML, err := boshClient.ResolveManifestVersionsV2([]byte(manifest))
	if err != nil {
		return "", err
	}

	_, err = boshClient.Deploy(manifestYAML)
	if err != nil {
		return "", err
	}

	//TODO: What is this for?
	//err = VerifyDeploymentRelease(boshClient, manifestName, releaseVersion)
	//if err != nil {
	//return "", err
	//}

	return string(manifestYAML), nil
}

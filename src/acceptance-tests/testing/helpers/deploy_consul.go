package helpers

import (
	"errors"
	"fmt"

	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/consulwithops"
	"github.com/pivotal-cf-experimental/destiny/ops"
)

func NewConsulManifestWithOpsWithInstanceCountAndReleaseVersion(deploymentPrefix string, instanceCount int, boshClient bosh.Client, releaseVersion string) (string, error) {
	manifestName := fmt.Sprintf("consul-%s", deploymentPrefix)

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

	return string(manifestYAML), nil
}

func NewConsulManifestWithOpsWithInstanceCount(deploymentPrefix string, instanceCount int, boshClient bosh.Client) (string, error) {
	return NewConsulManifestWithOpsWithInstanceCountAndReleaseVersion(deploymentPrefix, instanceCount, boshClient, ConsulReleaseVersion())
}

func DeployConsulWithOpsWithInstanceCountAndReleaseVersion(deploymentPrefix string, instanceCount int, boshClient bosh.Client, releaseVersion string) (string, error) {
	manifest, err := NewConsulManifestWithOpsWithInstanceCountAndReleaseVersion(deploymentPrefix, instanceCount, boshClient, releaseVersion)
	if err != nil {
		return "", err
	}

	_, err = boshClient.Deploy([]byte(manifest))
	if err != nil {
		return "", err
	}

	return manifest, nil
}

func DeployConsulWithOpsWithInstanceCount(deploymentPrefix string, instanceCount int, boshClient bosh.Client) (string, error) {
	return DeployConsulWithOpsWithInstanceCountAndReleaseVersion(deploymentPrefix, instanceCount, boshClient, ConsulReleaseVersion())
}

func VerifyDeploymentRelease(client bosh.Client, deploymentName string, releaseVersion string) error {
	deployments, err := client.Deployments()
	if err != nil {
		return err
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

	return err
}

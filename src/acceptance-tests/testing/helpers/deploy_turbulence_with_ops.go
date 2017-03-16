package helpers

import (
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/turbulencewithops"
)

func DeployTurbulenceWithOps(client bosh.Client) (string, error) {
	info, err := client.Info()
	if err != nil {
		return "", err
	}

	boshConfig := client.GetConfig()

	manifest, err := turbulencewithops.NewManifestV2(turbulencewithops.ConfigV2{
		DirectorUUID:     info.UUID,
		Name:             "turbulence",
		AZs:              []string{"z1"},
		DirectorHost:     boshConfig.Host,
		DirectorUsername: boshConfig.Username,
		DirectorPassword: boshConfig.Password,
	})
	if err != nil {
		return "", err
	}

	yaml, err := client.ResolveManifestVersionsV2([]byte(manifest))
	if err != nil {
		return "", err
	}

	_, err = client.Deploy(yaml)
	if err != nil {
		return "", err
	}

	return string(yaml), nil
}

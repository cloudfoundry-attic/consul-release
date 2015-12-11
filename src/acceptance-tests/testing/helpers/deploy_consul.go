package helpers

import (
	"fmt"
	"io/ioutil"

	"acceptance-tests/testing/bosh"
	"acceptance-tests/testing/consul"
	"acceptance-tests/testing/destiny"
)

func DeployConsulWithInstanceCount(count int, client bosh.Client) (manifest destiny.Manifest, kv consul.KV, err error) {
	guid, err := NewGUID()
	if err != nil {
		return
	}

	directorUUID, err := client.DirectorUUID()
	if err != nil {
		return
	}

	manifest = destiny.NewConsul(destiny.Config{
		DirectorUUID: directorUUID,
		Name:         fmt.Sprintf("consul-%s", guid),
	})

	manifest.Jobs[0], manifest.Properties = destiny.SetJobInstanceCount(manifest.Jobs[0], manifest.Networks[0], manifest.Properties, count)

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

	members := manifest.ConsulMembers()
	if len(members) != count {
		err = fmt.Errorf("expected %d consul members, found %d", count, len(members))
		return
	}

	consulMemberAddresses := []string{}
	for _, member := range members {
		consulMemberAddresses = append(consulMemberAddresses, member.Address)
	}

	dataDir, err := ioutil.TempDir("", "consul")
	if err != nil {
		return
	}

	agent := consul.NewAgent(consul.AgentOptions{
		DataDir:   dataDir,
		RetryJoin: consulMemberAddresses,
	})

	agentLocation := "http://127.0.0.1:8500"
	kv = consul.NewManagedKV(consul.ManagedKVConfig{
		Agent:   agent,
		KV:      consul.NewHTTPKV(agentLocation),
		Catalog: consul.NewHTTPCatalog(agentLocation),
	})

	return
}

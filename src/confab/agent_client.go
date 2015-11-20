package confab

import (
	"errors"

	"github.com/hashicorp/consul/api"
)

type consulAPIAgent interface {
	Members(wan bool) ([]*api.AgentMember, error)
}

type consulRPCClient interface {
	Stats() (map[string]map[string]string, error)
	ListKeys() ([]string, error)
	InstallKey(key string) error
	UseKey(key string) error
	RemoveKey(key string) error
	Leave() error
}

type AgentClient struct {
	ExpectedMembers []string
	ConsulAPIAgent  consulAPIAgent
	ConsulRPCClient consulRPCClient
}

func (c AgentClient) VerifyJoined() error {
	members, err := c.ConsulAPIAgent.Members(false)
	if err != nil {
		return err
	}

	for _, member := range members {
		for _, expectedMember := range c.ExpectedMembers {
			if member.Addr == expectedMember {
				return nil
			}
		}
	}

	return errors.New("no expected members")
}

func (c AgentClient) VerifySynced() error {
	stats, err := c.ConsulRPCClient.Stats()
	if err != nil {
		return err
	}

	if stats["raft"]["commit_index"] != stats["raft"]["last_log_index"] {
		return errors.New("log not in sync")
	}

	if stats["raft"]["commit_index"] == "0" {
		return errors.New("commit index must not be zero")
	}

	return nil
}

func (c AgentClient) IsLastNode() (bool, error) {
	members, err := c.ConsulAPIAgent.Members(false)
	if err != nil {
		return false, err
	}

	var serversCount int
	for _, member := range members {
		if member.Tags["role"] == "consul" {
			serversCount++
		}
	}

	hasAllExpectedMembers := serversCount == len(c.ExpectedMembers)

	return hasAllExpectedMembers, nil
}

func (c AgentClient) SetKeys(keys []string) error {
	if keys == nil {
		return errors.New("must provide a non-nil slice of keys")
	}

	if len(keys) == 0 {
		return errors.New("must provide a non-empty slice of keys")
	}

	existingKeys, err := c.ConsulRPCClient.ListKeys()
	if err != nil {
		return err
	}

	for _, key := range existingKeys {
		if !containsString(keys, key) {
			err := c.ConsulRPCClient.RemoveKey(key)
			if err != nil {
				return err
			}
		}
	}

	for _, key := range keys {
		err := c.ConsulRPCClient.InstallKey(key)
		if err != nil {
			return err
		}
	}

	err = c.ConsulRPCClient.UseKey(keys[0])
	if err != nil {
		return err
	}

	return nil
}

func (c AgentClient) Leave() error {
	if c.ConsulRPCClient == nil {
		return errors.New("consul rpc client is nil")
	}

	return c.ConsulRPCClient.Leave()
}

func containsString(elems []string, elem string) bool {
	for _, e := range elems {
		if elem == e {
			return true
		}
	}

	return false
}

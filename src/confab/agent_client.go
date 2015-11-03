package confab

import (
	"errors"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/command/agent"
)

const keyringToken = ""

type KeyringResponse struct {
	Keys []agent.KeyringEntry
}

type consulAPIAgent interface {
	Members(wan bool) ([]*api.AgentMember, error)
}

type consulRPCClient interface {
	Stats() (map[string]map[string]string, error)
	ListKeys(token string) (KeyringResponse, error)
	InstallKey(key, token string) (KeyringResponse, error)
	UseKey(key, token string) (KeyringResponse, error)
	RemoveKey(key, token string) (KeyringResponse, error)
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
		return errors.New("Log not in sync")
	}

	if stats["raft"]["commit_index"] == "0" {
		return errors.New("Commit index must not be zero")
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
	listKeysResponse, err := c.ConsulRPCClient.ListKeys(keyringToken)
	if err != nil {
		panic(err)
	}

	for _, keyEntry := range listKeysResponse.Keys {
		if !containsString(keys, keyEntry.Key) {
			_, err := c.ConsulRPCClient.RemoveKey(keyEntry.Key, keyringToken)
			if err != nil {
				panic(err)
			}
		}
	}

	for _, key := range keys {
		_, err := c.ConsulRPCClient.InstallKey(key, keyringToken)
		if err != nil {
			panic(err)
		}
	}

	_, err = c.ConsulRPCClient.UseKey(keys[0], keyringToken)
	if err != nil {
		panic(err)
	}

	return nil
}

func containsString(elems []string, elem string) bool {
	for _, e := range elems {
		if elem == e {
			return true
		}
	}

	return false
}

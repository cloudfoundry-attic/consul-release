package confab

import (
	"crypto/sha1"
	"encoding/base64"
	"errors"

	"golang.org/x/crypto/pbkdf2"

	"github.com/hashicorp/consul/api"
	"github.com/pivotal-golang/lager"
)

type logger interface {
	Info(action string, data ...lager.Data)
	Error(action string, err error, data ...lager.Data)
}

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
	Logger          logger
}

func (c AgentClient) VerifyJoined() error {
	c.Logger.Info("agent-client.verify-joined.members.request", lager.Data{
		"wan": false,
	})

	members, err := c.ConsulAPIAgent.Members(false)
	if err != nil {
		c.Logger.Error("agent-client.verify-joined.members.request.failed", err, lager.Data{
			"wan": false,
		})
		return err
	}

	var addresses []string
	for _, member := range members {
		addresses = append(addresses, member.Addr)
	}

	c.Logger.Info("agent-client.verify-joined.members.response", lager.Data{
		"wan":     false,
		"members": addresses,
	})

	for _, member := range members {
		for _, expectedMember := range c.ExpectedMembers {
			if member.Addr == expectedMember {
				c.Logger.Info("agent-client.verify-joined.members.joined")
				return nil
			}
		}
	}

	err = errors.New("no expected members")
	c.Logger.Error("agent-client.verify-joined.members.not-joined", err, lager.Data{
		"wan":     false,
		"members": addresses,
	})

	return err
}

func (c AgentClient) VerifySynced() error {
	c.Logger.Info("agent-client.verify-synced.stats.request")

	stats, err := c.ConsulRPCClient.Stats()
	if err != nil {
		c.Logger.Error("agent-client.verify-synced.stats.request.failed", err)
		return err
	}

	commitIndex := stats["raft"]["commit_index"]
	lastLogIndex := stats["raft"]["last_log_index"]

	c.Logger.Info("agent-client.verify-synced.stats.response", lager.Data{
		"commit_index":   commitIndex,
		"last_log_index": lastLogIndex,
	})

	if commitIndex != lastLogIndex {
		err = errors.New("log not in sync")
		c.Logger.Error("agent-client.verify-synced.not-synced", err)
		return err
	}

	if commitIndex == "0" {
		err = errors.New("commit index must not be zero")
		c.Logger.Error("agent-client.verify-synced.zero-index", err)
		return err
	}

	c.Logger.Info("agent-client.verify-synced.synced")
	return nil
}

func (c AgentClient) IsLastNode() (bool, error) {
	c.Logger.Info("agent-client.is-last-node.members.request", lager.Data{
		"wan": false,
	})

	members, err := c.ConsulAPIAgent.Members(false)
	if err != nil {
		c.Logger.Error("agent-client.is-last-node.members.request.failed", err, lager.Data{
			"wan": false,
		})
		return false, err
	}

	var addresses []string
	for _, member := range members {
		addresses = append(addresses, member.Addr)
	}

	c.Logger.Info("agent-client.is-last-node.members.response", lager.Data{
		"wan":     false,
		"members": addresses,
	})

	var serversCount int
	for _, member := range members {
		if member.Tags["role"] == "consul" {
			serversCount++
		}
	}

	hasAllExpectedMembers := serversCount == len(c.ExpectedMembers)

	c.Logger.Info("agent-client.is-last-node.result", lager.Data{
		"actual_members_count":   serversCount,
		"expected_members_count": len(c.ExpectedMembers),
		"is_last_node":           hasAllExpectedMembers,
	})

	return hasAllExpectedMembers, nil
}

func (c AgentClient) SetKeys(keys []string) error {
	if keys == nil {
		err := errors.New("must provide a non-nil slice of keys")
		c.Logger.Error("agent-client.set-keys.nil-slice", err)
		return err
	}

	if len(keys) == 0 {
		err := errors.New("must provide a non-empty slice of keys")
		c.Logger.Error("agent-client.set-keys.empty-slice", err)
		return err
	}

	c.Logger.Info("agent-client.set-keys.list-keys.request")

	var encryptedKeys []string
	for _, key := range keys {
		encryptedKey := key

		decodedKey, err := base64.StdEncoding.DecodeString(key)
		if err != nil || len(decodedKey) != 16 {
			encryptedKey = base64.StdEncoding.EncodeToString(pbkdf2.Key([]byte(key), []byte(""), 20000, 16, sha1.New))
		}

		encryptedKeys = append(encryptedKeys, encryptedKey)
	}

	existingKeys, err := c.ConsulRPCClient.ListKeys()
	if err != nil {
		c.Logger.Error("agent-client.set-keys.list-keys.request.failed", err)
		return err
	}

	c.Logger.Info("agent-client.set-keys.list-keys.response", lager.Data{
		"keys": existingKeys,
	})

	for _, key := range existingKeys {
		if !containsString(encryptedKeys, key) {
			c.Logger.Info("agent-client.set-keys.remove-key.request", lager.Data{
				"key": key,
			})
			err := c.ConsulRPCClient.RemoveKey(key)
			if err != nil {
				c.Logger.Error("agent-client.set-keys.remove-key.request.failed", err, lager.Data{
					"key": key,
				})
				return err
			}
			c.Logger.Info("agent-client.set-keys.remove-key.response", lager.Data{
				"key": key,
			})
		}
	}

	for _, key := range encryptedKeys {
		c.Logger.Info("agent-client.set-keys.install-key.request", lager.Data{
			"key": key,
		})

		err := c.ConsulRPCClient.InstallKey(key)
		if err != nil {
			c.Logger.Error("agent-client.set-keys.install-key.request.failed", err, lager.Data{
				"key": key,
			})
			return err
		}

		c.Logger.Info("agent-client.set-keys.install-key.response", lager.Data{
			"key": key,
		})
	}

	c.Logger.Info("agent-client.set-keys.use-key.request", lager.Data{
		"key": encryptedKeys[0],
	})

	err = c.ConsulRPCClient.UseKey(encryptedKeys[0])
	if err != nil {
		c.Logger.Error("agent-client.set-keys.use-key.request.failed", err, lager.Data{
			"key": encryptedKeys[0],
		})
		return err
	}

	c.Logger.Info("agent-client.set-keys.use-key.response", lager.Data{
		"key": encryptedKeys[0],
	})

	c.Logger.Info("agent-client.set-keys.success")
	return nil
}

func (c AgentClient) Leave() error {
	if c.ConsulRPCClient == nil {
		err := errors.New("consul rpc client is nil")
		c.Logger.Error("agent-client.leave.nil-rpc-client", err)
		return err
	}

	c.Logger.Info("agent-client.leave.leave.request")

	if err := c.ConsulRPCClient.Leave(); err != nil {
		c.Logger.Error("agent-client.leave.leave.request.failed", err)
		return err
	}
	c.Logger.Info("agent-client.leave.leave.response")

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

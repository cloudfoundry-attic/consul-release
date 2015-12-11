package consul

import "time"

type AgentStartStopper interface {
	Start() error
	Stop() error
}

type KVSetGetter interface {
	Set(key, value string) error
	Get(key string) (string, error)
}
type CatalogNoder interface {
	Nodes() ([]Node, error)
}

type ManagedKV struct {
	config ManagedKVConfig
}

type ManagedKVConfig struct {
	Agent                        AgentStartStopper
	KV                           KVSetGetter
	Catalog                      CatalogNoder
	VerifyJoinedMaxTries         int
	VerifyJoinedIntervalDuration time.Duration
}

func NewManagedKV(config ManagedKVConfig) ManagedKV {
	if config.VerifyJoinedIntervalDuration == time.Duration(0) {
		config.VerifyJoinedIntervalDuration = 3 * time.Second
	}

	if config.VerifyJoinedMaxTries == 0 {
		config.VerifyJoinedMaxTries = 10
	}

	return ManagedKV{config}
}

func (m ManagedKV) verifyJoined() error {
	var err error
	var nodes []Node

	for i := 0; i < m.config.VerifyJoinedMaxTries; i++ {
		nodes, err = m.config.Catalog.Nodes()
		if err != nil {
			time.Sleep(m.config.VerifyJoinedIntervalDuration)
			continue
		}

		if len(nodes) > 1 {
			return nil
		}
	}
	return err
}

func (m ManagedKV) Set(key, value string) error {
	if err := m.config.Agent.Start(); err != nil {
		return err
	}

	if err := m.verifyJoined(); err != nil {
		return err
	}

	kvErr := m.config.KV.Set(key, value)

	if err := m.config.Agent.Stop(); err != nil {
		return err
	}

	return kvErr
}

func (m ManagedKV) Get(key string) (string, error) {
	if err := m.config.Agent.Start(); err != nil {
		return "", err
	}

	if err := m.verifyJoined(); err != nil {
		return "", err
	}

	value, kvErr := m.config.KV.Get(key)

	if err := m.config.Agent.Stop(); err != nil {
		return "", err
	}

	return value, kvErr
}

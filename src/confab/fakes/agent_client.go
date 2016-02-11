package fakes

import "github.com/cloudfoundry-incubator/consul-release/src/confab/agent"

type AgentClient struct {
	VerifyJoinedCalls struct {
		CallCount int
		Returns   struct {
			Errors []error
		}
	}
	VerifySyncedCalls struct {
		CallCount int
		Returns   struct {
			Errors []error
		}
	}

	IsLastNodeCall struct {
		Returns struct {
			IsLastNode bool
			Error      error
		}
	}

	SetKeysCall struct {
		Receives struct {
			Keys []string
		}
		Returns struct {
			Error error
		}
	}

	LeaveCall struct {
		CallCount int
		Returns   struct {
			Error error
		}
	}

	SetConsulRPCClientCall struct {
		CallCount int
		Receives  struct {
			ConsulRPCClient agent.ConsulRPCClient
		}
	}
}

func (c *AgentClient) VerifyJoined() error {
	err := c.VerifyJoinedCalls.Returns.Errors[c.VerifyJoinedCalls.CallCount]
	c.VerifyJoinedCalls.CallCount++
	return err
}

func (c *AgentClient) VerifySynced() error {
	err := c.VerifySyncedCalls.Returns.Errors[c.VerifySyncedCalls.CallCount]
	c.VerifySyncedCalls.CallCount++
	return err
}

func (c *AgentClient) IsLastNode() (bool, error) {
	return c.IsLastNodeCall.Returns.IsLastNode, c.IsLastNodeCall.Returns.Error
}

func (c *AgentClient) SetKeys(keys []string) error {
	c.SetKeysCall.Receives.Keys = keys
	return c.SetKeysCall.Returns.Error
}

func (c *AgentClient) Leave() error {
	c.LeaveCall.CallCount++
	return c.LeaveCall.Returns.Error
}

func (c *AgentClient) SetConsulRPCClient(rpcClient agent.ConsulRPCClient) {
	c.SetConsulRPCClientCall.CallCount++
	c.SetConsulRPCClientCall.Receives.ConsulRPCClient = rpcClient
}

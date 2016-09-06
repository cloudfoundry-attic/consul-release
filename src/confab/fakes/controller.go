package fakes

import (
	"github.com/cloudfoundry-incubator/consul-release/src/confab"
	"github.com/hashicorp/consul/command/agent"
)

type Controller struct {
	WriteConsulConfigCall struct {
		CallCount int
		Returns   struct {
			Error error
		}
	}

	WriteServiceDefinitionsCall struct {
		CallCount int
		Returns   struct {
			Error error
		}
	}

	BootAgentCall struct {
		CallCount int
		Receives  struct {
			Timeout confab.Timeout
		}
		Returns struct {
			Error error
		}
	}

	ConfigureServerCall struct {
		CallCount int
		Receives  struct {
			Timeout   confab.Timeout
			RPCClient *agent.RPCClient
		}
		Returns struct {
			Error error
		}
	}

	ConfigureClientCall struct {
		CallCount int
		Returns   struct {
			Error error
		}
	}

	StopAgentCall struct {
		CallCount int
		Receives  struct {
			RPCClient *agent.RPCClient
		}
	}
}

func (c *Controller) WriteConsulConfig() error {
	c.WriteConsulConfigCall.CallCount++

	return c.WriteConsulConfigCall.Returns.Error
}

func (c *Controller) WriteServiceDefinitions() error {
	c.WriteServiceDefinitionsCall.CallCount++

	return c.WriteServiceDefinitionsCall.Returns.Error
}

func (c *Controller) BootAgent(timeout confab.Timeout) error {
	c.BootAgentCall.CallCount++
	c.BootAgentCall.Receives.Timeout = timeout

	return c.BootAgentCall.Returns.Error
}

func (c *Controller) ConfigureServer(timeout confab.Timeout, rpcClient *agent.RPCClient) error {
	c.ConfigureServerCall.CallCount++
	c.ConfigureServerCall.Receives.Timeout = timeout
	c.ConfigureServerCall.Receives.RPCClient = rpcClient

	return c.ConfigureServerCall.Returns.Error
}

func (c *Controller) ConfigureClient() error {
	c.ConfigureClientCall.CallCount++

	return c.ConfigureClientCall.Returns.Error
}

func (c *Controller) StopAgent(rpcClient *agent.RPCClient) {
	c.StopAgentCall.CallCount++
	c.StopAgentCall.Receives.RPCClient = rpcClient
}

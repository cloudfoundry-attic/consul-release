package chaperon

import "github.com/cloudfoundry-incubator/consul-release/src/confab"

type Client struct {
	controller   controller
	newRPCClient consulRPCClientConstructor
}

func NewClient(c controller, newRPCClient consulRPCClientConstructor) Client {
	return Client{
		controller:   c,
		newRPCClient: newRPCClient,
	}
}

func (c Client) Start(timeout confab.Timeout) error {
	if err := c.controller.WriteConsulConfig(); err != nil {
		return err
	}

	if err := c.controller.WriteServiceDefinitions(); err != nil {
		return err
	}

	if err := c.controller.BootAgent(timeout); err != nil {
		return err
	}

	if err := c.controller.ConfigureClient(); err != nil {
		return err
	}

	return nil
}

func (c Client) Stop() error {
	rpcClient, err := c.newRPCClient("localhost:8400")
	c.controller.StopAgent(rpcClient)

	return err
}

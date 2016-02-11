package chaperon

import (
	"github.com/cloudfoundry-incubator/consul-release/src/confab"
	"github.com/hashicorp/consul/command/agent"
)

type controller interface {
	WriteConsulConfig() error
	WriteServiceDefinitions() error
	BootAgent(confab.Timeout) error
	ConfigureServer(confab.Timeout, *agent.RPCClient) error
	ConfigureClient() error
	StopAgent(*agent.RPCClient)
}

type consulRPCClientConstructor func(url string) (*agent.RPCClient, error)

type Server struct {
	controller   controller
	newRPCClient consulRPCClientConstructor
}

func NewServer(c controller, newRPCClient consulRPCClientConstructor) Server {
	return Server{
		controller:   c,
		newRPCClient: newRPCClient,
	}
}

func (s Server) Start(timeout confab.Timeout) error {
	if err := s.controller.WriteConsulConfig(); err != nil {
		return err
	}

	if err := s.controller.WriteServiceDefinitions(); err != nil {
		return err
	}

	if err := s.controller.BootAgent(timeout); err != nil {
		return err
	}

	rpcClient, err := s.newRPCClient("localhost:8400")
	if err != nil {
		return err
	}

	if err := s.controller.ConfigureServer(timeout, rpcClient); err != nil {
		return err
	}

	return nil
}

func (s Server) Stop() error {
	rpcClient, err := s.newRPCClient("localhost:8400")
	s.controller.StopAgent(rpcClient)

	return err
}

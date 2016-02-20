package chaperon

import (
	"github.com/cloudfoundry-incubator/consul-release/src/confab"
	"github.com/cloudfoundry-incubator/consul-release/src/confab/config"
	"github.com/hashicorp/consul/command/agent"
)

type controller interface {
	WriteServiceDefinitions() error
	BootAgent(confab.Timeout) error
	ConfigureServer(confab.Timeout, *agent.RPCClient) error
	ConfigureClient() error
	StopAgent(*agent.RPCClient)
}

type configWriter interface {
	Write(config.Config) error
}

type consulRPCClientConstructor func(url string) (*agent.RPCClient, error)

type Server struct {
	controller   controller
	newRPCClient consulRPCClientConstructor
	configWriter configWriter
}

func NewServer(controller controller, configWriter configWriter, newRPCClient consulRPCClientConstructor) Server {
	return Server{
		controller:   controller,
		configWriter: configWriter,
		newRPCClient: newRPCClient,
	}
}

func (s Server) Start(cfg config.Config, timeout confab.Timeout) error {
	if err := s.configWriter.Write(cfg); err != nil {
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

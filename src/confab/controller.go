package confab

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/pivotal-golang/lager"
)

type agentRunner interface {
	Run() error
	Stop() error
	Wait() error
	Cleanup() error
	WritePID() error
}

type agentClient interface {
	VerifyJoined() error
	VerifySynced() error
	IsLastNode() (bool, error)
	SetKeys([]string) error
	Leave() error
}

type serviceDefiner interface {
	GenerateDefinitions(Config) []ServiceDefinition
	WriteDefinitions(string, []ServiceDefinition) error
}

type clock interface {
	Sleep(time.Duration)
}

type Controller struct {
	AgentRunner    agentRunner
	AgentClient    agentClient
	SyncRetryDelay time.Duration
	SyncRetryClock clock
	EncryptKeys    []string
	SSLDisabled    bool
	Logger         logger
	ConfigDir      string
	ServiceDefiner serviceDefiner
	Config         Config
}

func (c Controller) BootAgent(timeout Timeout) error {
	c.Logger.Info("controller.boot-agent.run")
	err := c.AgentRunner.Run()
	if err != nil {
		c.Logger.Error("controller.boot-agent.run.failed", err)
		return err
	}

	c.Logger.Info("controller.boot-agent.verify-joined")

	if err := c.callWithTimeout(timeout, c.AgentClient.VerifyJoined); err != nil {
		c.Logger.Error("controller.boot-agent.verify-joined.failed", err)
		return err
	}

	c.Logger.Info("controller.boot-agent.success")
	return nil
}

func (c Controller) callWithTimeout(timeout Timeout, f func() error) error {
	for {
		select {
		case <-timeout.Done():
			return errors.New("timeout exceeded")
		default:
			err := f()
			if err != nil {
				c.SyncRetryClock.Sleep(c.SyncRetryDelay)
				continue
			}

			return nil
		}
	}
}

func (c Controller) ConfigureServer(timeout Timeout) error {
	c.Logger.Info("controller.configure-server.is-last-node")
	lastNode, err := c.AgentClient.IsLastNode()
	if err != nil {
		c.Logger.Error("controller.configure-server.is-last-node.failed", err)
		return err
	}

	if lastNode {
		c.Logger.Info("controller.configure-server.verify-synced")
		if err := c.callWithTimeout(timeout, c.AgentClient.VerifySynced); err != nil {
			c.Logger.Error("controller.configure-server.verify-synced.failed", err)
			return err
		}
	}

	if c.Config.Consul.RequireSSL {
		if len(c.EncryptKeys) == 0 {
			err := errors.New("encrypt keys cannot be empty if ssl is enabled")
			c.Logger.Error("controller.configure-server.no-encrypt-keys", err)
			return err
		}

		c.Logger.Info("controller.configure-server.set-keys", lager.Data{
			"keys": c.EncryptKeys,
		})

		err = c.AgentClient.SetKeys(c.EncryptKeys)
		if err != nil {
			c.Logger.Error("controller.configure-server.set-keys.failed", err, lager.Data{
				"keys": c.EncryptKeys,
			})
			return err
		}
	}

	if err := c.AgentRunner.WritePID(); err != nil {
		c.Logger.Error("controller.configure-server.write-pid.failed", err)
		return err
	}

	c.Logger.Info("controller.configure-server.success")
	return nil
}

func (c Controller) ConfigureClient() error {
	return c.AgentRunner.WritePID()
}

func (c Controller) StopAgent() {
	c.Logger.Info("controller.stop-agent.leave")
	if err := c.AgentClient.Leave(); err != nil {
		c.Logger.Error("controller.stop-agent.leave.failed", err)

		c.Logger.Info("controller.stop-agent.stop")
		if err = c.AgentRunner.Stop(); err != nil {
			c.Logger.Error("controller.stop-agent.stop.failed", err)
		}
	}

	c.Logger.Info("controller.stop-agent.wait")
	if err := c.AgentRunner.Wait(); err != nil {
		c.Logger.Error("controller.stop-agent.wait.failed", err)
	}

	c.Logger.Info("controller.stop-agent.cleanup")
	if err := c.AgentRunner.Cleanup(); err != nil {
		c.Logger.Error("controller.stop-agent.cleanup.failed", err)
	}

	c.Logger.Info("controller.stop-agent.success")
}

func (c Controller) WriteServiceDefinitions() error {
	c.Logger.Info("controller.write-service-definitions.generate-definitions")
	definitions := c.ServiceDefiner.GenerateDefinitions(c.Config)

	c.Logger.Info("controller.write-service-definitions.write")
	if err := c.ServiceDefiner.WriteDefinitions(c.ConfigDir, definitions); err != nil {
		c.Logger.Error("controller.write-service-definitions.write.failed", err)
		return err
	}

	c.Logger.Info("controller.write-service-definitions.success")
	return nil
}

func (c Controller) WriteConsulConfig() error {
	consulConfig := GenerateConfiguration(c.Config)

	data, err := json.Marshal(&consulConfig)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(filepath.Join(c.Config.Path.ConsulConfigDir, "config.json"), data, os.ModePerm)
	if err != nil {
		panic(err)
	}

	return nil
}

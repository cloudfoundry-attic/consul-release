package confab

import (
	"errors"
	"time"

	"github.com/pivotal-golang/lager"
)

type agentRunner interface {
	Run() error
	Stop() error
	Wait() error
}

type agentClient interface {
	VerifyJoined() error
	VerifySynced() error
	IsLastNode() (bool, error)
	SetKeys([]string) error
	Leave() error
}

type clock interface {
	Sleep(time.Duration)
}

type Controller struct {
	AgentRunner    agentRunner
	AgentClient    agentClient
	MaxRetries     int
	SyncRetryDelay time.Duration
	SyncRetryClock clock
	EncryptKeys    []string
	SSLDisabled    bool
	Logger         logger
}

func (c Controller) BootAgent() error {
	c.Logger.Info("controller.boot-agent.run")
	err := c.AgentRunner.Run()
	if err != nil {
		c.Logger.Error("controller.boot-agent.run.failed", err)
		return err
	}

	c.Logger.Info("controller.boot-agent.verify-joined")
	for i := 1; i <= c.MaxRetries; i++ {
		err := c.AgentClient.VerifyJoined()
		if err != nil {
			if i == c.MaxRetries {
				c.Logger.Error("controller.boot-agent.verify-joined.failed", err)
				return err
			}

			c.SyncRetryClock.Sleep(c.SyncRetryDelay)
			continue
		}

		break
	}

	c.Logger.Info("controller.boot-agent.success")
	return nil
}

func (c Controller) ConfigureServer() error {
	c.Logger.Info("controller.configure-server.is-last-node")
	lastNode, err := c.AgentClient.IsLastNode()
	if err != nil {
		c.Logger.Error("controller.configure-server.is-last-node.failed", err)
		return err
	}

	if lastNode {
		c.Logger.Info("controller.configure-server.verify-synced")
		for i := 1; i <= c.MaxRetries; i++ {
			err = c.AgentClient.VerifySynced()
			if err != nil {
				if i == c.MaxRetries {
					c.Logger.Error("controller.configure-server.verify-synced.failed", err)
					return err
				}

				c.SyncRetryClock.Sleep(c.SyncRetryDelay)
				continue
			}

			break
		}
	}

	if !c.SSLDisabled {
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

	c.Logger.Info("controller.configure-server.success")
	return nil
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
	c.Logger.Info("controller.stop-agent.success")
}

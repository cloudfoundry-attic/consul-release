package confab

import (
	"errors"
	"log"
	"time"
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
	Logger         *log.Logger
}

func (c Controller) BootAgent() error {
	err := c.AgentRunner.Run()
	if err != nil {
		return err
	}

	for i := 1; i <= c.MaxRetries; i++ {
		err := c.AgentClient.VerifyJoined()
		if err != nil {
			if i == c.MaxRetries {
				return err
			}

			c.SyncRetryClock.Sleep(c.SyncRetryDelay)
			continue
		}

		break
	}

	return nil
}

func (c Controller) ConfigureServer() error {
	lastNode, err := c.AgentClient.IsLastNode()
	if err != nil {
		return err
	}

	if lastNode {
		for i := 1; i <= c.MaxRetries; i++ {
			err = c.AgentClient.VerifySynced()
			if err != nil {
				if i == c.MaxRetries {
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
			return errors.New("encrypt keys cannot be empty if ssl is enabled")
		}

		err = c.AgentClient.SetKeys(c.EncryptKeys)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c Controller) StopAgent() {
	c.Logger.Printf("%s", "STOPAGENT: calling AgentClient.Leave()")
	if err := c.AgentClient.Leave(); err != nil {
		c.Logger.Printf("%s", err)

		c.Logger.Printf("%s", "STOPAGENT: calling AgentClient.Stop()")
		if err = c.AgentRunner.Stop(); err != nil {
			c.Logger.Printf("%s", err)
		}
		c.Logger.Printf("%s", "STOPAGENT: called AgentClient.Stop()")
	}
	c.Logger.Printf("%s", "STOPAGENT: called AgentClient.Leave()")

	c.Logger.Printf("%s", "STOPAGENT: calling AgentClient.Wait()")
	if err := c.AgentRunner.Wait(); err != nil {
		c.Logger.Printf("%s", err)
	}
	c.Logger.Printf("%s", "STOPAGENT: called AgentClient.Wait()")
}

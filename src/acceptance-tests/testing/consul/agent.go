package consul

import (
	"errors"
	"os"
	"os/exec"
)

type Agent struct {
	options AgentOptions
	*exec.Cmd
}

type AgentOptions struct {
	DataDir   string
	RetryJoin []string
}

func NewAgent(options AgentOptions) *Agent {
	return &Agent{
		options: options,
		Cmd:     &exec.Cmd{},
	}
}

func (a *Agent) Start() error {
	if err := exec.Command("consul", "members").Run(); err == nil {
		return errors.New("consul agent is already running")
	}

	args := []string{
		"agent",
		"-node", "localnode",
		"-bind", "127.0.0.1",
		"-data-dir", a.options.DataDir,
	}
	for _, address := range a.options.RetryJoin {
		args = append(args, "-retry-join", address)
	}

	a.Cmd = exec.Command("consul", args...)

	return a.Cmd.Start()
}

func (a *Agent) Stop() error {
	if a.Cmd.Process == nil {
		return nil
	}

	if err := a.Cmd.Process.Signal(os.Interrupt); err != nil {
		return err
	}

	_, err := a.Cmd.Process.Wait()
	return err
}

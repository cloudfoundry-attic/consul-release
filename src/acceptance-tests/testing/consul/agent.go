package consul

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

var createFile = os.Create

type AgentStartStopper interface {
	Start() error
	Stop() error
}

type Agent struct {
	options AgentOptions
	*exec.Cmd
}

type AgentOptions struct {
	ConfigDir  string
	DataDir    string
	Domain     string
	Key        string
	Cert       string
	CACert     string
	Encrypt    string
	ServerName string
	RetryJoin  []string
}

type agentConfig struct {
	CAFile               string `json:"ca_file"`
	CertFile             string `json:"cert_file"`
	KeyFile              string `json:"key_file"`
	Encrypt              string `json:"encrypt"`
	ServerName           string `json:"server_name"`
	Domain               string `json:"domain"`
	VerifyIncoming       bool   `json:"verify_incoming"`
	VerifyOutgoing       bool   `json:"verify_outgoing"`
	VerifyServerHostname bool   `json:"verify_server_hostname"`
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

	if a.options.ConfigDir != "" {
		args = append(args, "-config-dir", a.options.ConfigDir)

		if err := os.MkdirAll(a.options.ConfigDir, os.ModePerm); err != nil {
			return err
		}

		configFile, err := createFile(filepath.Join(a.options.ConfigDir, "config.json"))
		if err != nil {
			return err
		}

		err = json.NewEncoder(configFile).Encode(agentConfig{
			CAFile:               filepath.Join(a.options.ConfigDir, "ca.cert"),
			CertFile:             filepath.Join(a.options.ConfigDir, "agent.cert"),
			KeyFile:              filepath.Join(a.options.ConfigDir, "agent.key"),
			Encrypt:              a.options.Encrypt,
			ServerName:           a.options.ServerName,
			Domain:               a.options.Domain,
			VerifyIncoming:       true,
			VerifyOutgoing:       true,
			VerifyServerHostname: true,
		})
		if err != nil {
			return err
		}

		for filename, contents := range map[string]string{
			"ca.cert":    a.options.CACert,
			"agent.key":  a.options.Key,
			"agent.cert": a.options.Cert,
		} {
			file, err := createFile(filepath.Join(a.options.ConfigDir, filename))
			if err != nil {
				return err
			}

			_, err = file.WriteString(contents)
			if err != nil {
				return err
			}
		}
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

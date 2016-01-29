package confab

import "encoding/json"

type Config struct {
	Node   ConfigNode
	Confab ConfigConfab
	Consul ConfigConsul
	Path   ConfigPath
}

type ConfigConfab struct {
	TimeoutInSeconds int `json:"timeout_in_seconds"`
}

type ConfigConsul struct {
	Agent       ConfigAgent
	RequireSSL  bool     `json:"require_ssl"`
	EncryptKeys []string `json:"encrypt_keys"`
}

type ConfigPath struct {
	AgentPath       string `json:"agent_path"`
	ConsulConfigDir string `json:"consul_config_dir"`
	PIDFile         string `json:"pid_file"`
}

type ConfigNode struct {
	Name  string
	Index int
}

type ConfigAgent struct {
	Servers         ConfigAgentServer
	Services        map[string]ServiceDefinition
	Server          bool
	Datacenter      string `json:"datacenter"`
	LogLevel        string `json:"log_level"`
	ProtocolVersion int    `json:"protocol_version"`
}

type ConfigAgentServer struct {
	LAN []string
}

func DefaultConfig() Config {
	return Config{
		Path: ConfigPath{
			AgentPath:       "/var/vcap/packages/consul/bin/consul",
			ConsulConfigDir: "/var/vcap/jobs/consul_agent/config",
			PIDFile:         "/var/vcap/sys/run/consul_agent/consul_agent.pid",
		},
		Consul: ConfigConsul{
			RequireSSL: true,
		},
		Confab: ConfigConfab{
			TimeoutInSeconds: 55,
		},
	}
}

func ConfigFromJSON(configData []byte) (Config, error) {
	config := DefaultConfig()
	if err := json.Unmarshal(configData, &config); err != nil {
		return Config{}, err
	}

	return config, nil
}

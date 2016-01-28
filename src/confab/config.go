package confab

import "encoding/json"

type Config struct {
	Node   ConfigNode
	Consul ConfigConsul
	Path   ConfigPath
}

type ConfigConsul struct {
	Agent       ConfigAgent
	RequireSSL  bool     `json:"require_ssl"`
	EncryptKeys []string `json:"encrypt_keys"`
}

type ConfigPath struct {
	AgentPath string `json:"agent_path"`
}

type ConfigNode struct {
	Name  string
	Index int
}

type ConfigAgent struct {
	Servers  ConfigAgentServer
	Services map[string]ServiceDefinition
	Server   bool
}

type ConfigAgentServer struct {
	LAN []string
}

func DefaultConfig() Config {
	return Config{
		Path: ConfigPath{
			AgentPath: "/var/vcap/packages/consul/bin/consul",
		},
		Consul: ConfigConsul{
			RequireSSL: true,
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

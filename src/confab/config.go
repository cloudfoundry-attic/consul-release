package confab

import "encoding/json"

type Config struct {
	Node        ConfigNode
	Agent       ConfigAgent
	RequireSSL  bool     `json:"require_ssl"`
	EncryptKeys []string `json:"encrypt_keys"`
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
		RequireSSL: true,
	}
}

func ConfigFromJSON(configData []byte) (Config, error) {
	config := DefaultConfig()
	if err := json.Unmarshal(configData, &config); err != nil {
		return Config{}, err
	}

	return config, nil
}

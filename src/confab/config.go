package confab

import "encoding/json"

type Config struct {
	Node       ConfigNode
	Agent      ConfigAgent
	RequireSSL bool `json:"require_ssl"`
}

type ConfigNode struct {
	Name  string
	Index int
}

type ConfigAgent struct {
	Services map[string]ServiceDefinition
	Server   bool
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

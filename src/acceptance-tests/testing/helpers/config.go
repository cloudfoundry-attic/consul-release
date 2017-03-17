package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	BOSH           ConfigBOSH `json:"bosh"`
	ParallelNodes  int        `json:"parallel_nodes"`
	WindowsClients bool       `json:"windows_clients"`
}

type ConfigBOSH struct {
	Target         string `json:"target"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	DirectorCACert string `json:"director_ca_cert"`
}

func checkAbsolutePath(configValue, jsonKey string) error {
	if !strings.HasPrefix(configValue, "/") {
		return fmt.Errorf("invalid `%s` %q - must be an absolute path", jsonKey, configValue)
	}
	return nil
}

func LoadConfig(configFilePath string) (Config, error) {
	config, err := loadConfigJsonFromPath(configFilePath)
	if err != nil {
		return Config{}, err
	}

	if config.BOSH.Target == "" {
		return Config{}, errors.New("missing `bosh.target` - e.g. 'lite' or '192.168.50.4'")
	}

	if config.BOSH.Username == "" {
		return Config{}, errors.New("missing `bosh.username` - specify username for authenticating with BOSH")
	}

	if config.BOSH.Password == "" {
		return Config{}, errors.New("missing `bosh.password` - specify password for authenticating with BOSH")
	}

	if config.ParallelNodes == 0 {
		config.ParallelNodes = 1
	}

	return config, nil
}

func loadConfigJsonFromPath(configFilePath string) (Config, error) {
	configFile, err := os.Open(configFilePath)
	if err != nil {
		return Config{}, err
	}

	var config Config
	if err := json.NewDecoder(configFile).Decode(&config); err != nil {
		return Config{}, err
	}

	return config, nil
}

func ConsulReleaseVersion() string {
	version := os.Getenv("CONSUL_RELEASE_VERSION")
	if version == "" {
		version = "latest"
	}

	return version
}

func ConfigPath() (string, error) {
	path := os.Getenv("CONSATS_CONFIG")
	if path == "" || !strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("$CONSATS_CONFIG %q does not specify an absolute path to test config file", path)
	}

	return path, nil
}

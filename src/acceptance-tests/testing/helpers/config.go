package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	BOSHTarget            string `json:"bosh_target"`
	BOSHUsername          string `json:"bosh_username"`
	BOSHPassword          string `json:"bosh_password"`
	TurbulenceReleaseName string
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

	if config.BOSHTarget == "" {
		return Config{}, errors.New("missing `bosh_target` - e.g. 'lite' or '192.168.50.4'")
	}

	if config.BOSHUsername == "" {
		return Config{}, errors.New("missing `bosh_username` - specify username for authenticating with BOSH")
	}

	if config.BOSHPassword == "" {
		return Config{}, errors.New("missing `bosh_password` - specify password for authenticating with BOSH")
	}

	config.TurbulenceReleaseName = "turbulence"

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

func ConfigPath() (string, error) {
	path := os.Getenv("CONSATS_CONFIG")
	if path == "" || !strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("$CONSATS_CONFIG %q does not specify an absolute path to test config file", path)
	}

	return path, nil
}

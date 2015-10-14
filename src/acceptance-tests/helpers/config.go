package helpers

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Config struct {
	BindAddress                    string `json:"bind_address"`
	BoshTarget                     string `json:"bosh_target"`
	IAASSettingsConsulStubPath     string `json:"iaas_settings_consul_stub_path"`
	IAASSettingsTurbulenceStubPath string `json:"iaas_settings_turbulence_stub_path"`
	TurbulencePropertiesStubPath   string `json:"turbulence_properties_stub_path"`
	CPIReleaseUrl                  string `json:"cpi_release_url"`
	CPIReleaseName                 string `json:"cpi_release_name"`
	TurbulenceReleaseUrl           string `json:"turbulence_release_url"`
	TurbulenceReleaseName          string `json:"turbulence_release_name"`
	BoshOperationTimeout           string `json:"bosh_operation_timeout"`
	TurbulenceOperationTimeout     string `json:"turbulence_operation_timeout"`
}

var loadedConfig *Config

func LoadConfig() Config {
	if loadedConfig == nil {
		loadedConfig = loadConfigJsonFromPath()
	}

	if loadedConfig.BindAddress == "" {
		panic("missing Bind Address. Specify which address consul should bind to")
	}

	if loadedConfig.BoshTarget == "" {
		panic("missing BOSH target (e.g. 'lite' or '192.168.50.4'")
	}

	if loadedConfig.IAASSettingsConsulStubPath == "" {
		panic("missing consul IaaS settings stub path")
	}

	return *loadedConfig
}

func loadConfigJsonFromPath() *Config {
	var config *Config = &Config{}

	path := configPath()

	configFile, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(config)
	if err != nil {
		panic(err)
	}

	return config
}

func configPath() string {
	path := os.Getenv("CONSATS_CONFIG")
	if path == "" {
		panic("Must set $CONSATS_CONFIG to point to an consul acceptance tests config file.")
	}

	return path
}

func GetBoshOperationTimeout(config Config) time.Duration {
	if config.BoshOperationTimeout == "" {
		return defaultBoshOperationTimeout
	}

	duration, err := time.ParseDuration(config.BoshOperationTimeout)
	if err != nil {
		panic(fmt.Sprintf("invalid duration string for BOSH operation timeout config: '%s'", config.BoshOperationTimeout))
	}

	return duration
}

func GetTurbulenceOperationTimeout(config Config) time.Duration {
	if config.TurbulenceOperationTimeout == "" {
		return defaultTurbulenceOperationTimeout
	}

	duration, err := time.ParseDuration(config.TurbulenceOperationTimeout)
	if err != nil {
		panic(fmt.Sprintf("invalid duration string for Turbulence operation timeout config: '%s'", config.TurbulenceOperationTimeout))
	}

	return duration
}

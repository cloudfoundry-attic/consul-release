package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	BindAddress                    string `json:"bind_address"`
	BoshTarget                     string `json:"bosh_target"`
	IAASSettingsConsulStubPath     string `json:"iaas_settings_consul_stub_path"`
	IAASSettingsTurbulenceStubPath string `json:"iaas_settings_turbulence_stub_path"`
	TurbulencePropertiesStubPath   string `json:"turbulence_properties_stub_path"`
	CPIReleaseLocation             string `json:"cpi_release_location"`
	CPIReleaseName                 string `json:"cpi_release_name"`
	TurbulenceReleaseLocation      string `json:"turbulence_release_location"`
	TurbulenceReleaseName          string
	BoshOperationTimeout           string `json:"bosh_operation_timeout"`
	TurbulenceOperationTimeout     string `json:"turbulence_operation_timeout"`
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

	if config.BindAddress == "" {
		return Config{}, errors.New("missing `bind_address` - specify which address consul should bind to")
	}

	if config.BoshTarget == "" {
		return Config{}, errors.New("missing `bosh_target` - e.g. 'lite' or '192.168.50.4'")
	}

	if config.IAASSettingsConsulStubPath == "" {
		return Config{}, errors.New("missing `iaas_settings_consul_stub_path` - path to consul stub file")
	}

	if err := checkAbsolutePath(config.IAASSettingsConsulStubPath, "iaas_settings_consul_stub_path"); err != nil {
		return Config{}, err
	}

	if hasTurbulenceConfig(config) {
		if err := validateTurbulenceConfig(config); err != nil {
			return Config{}, err
		}
	}

	config.TurbulenceReleaseName = "turbulence"

	return config, nil
}

func hasTurbulenceConfig(config Config) bool {
	for _, v := range []string{
		config.IAASSettingsTurbulenceStubPath,
		config.TurbulencePropertiesStubPath,
		config.TurbulenceReleaseLocation,
		config.CPIReleaseLocation,
		config.CPIReleaseName,
	} {
		if v != "" {
			return true
		}
	}

	return false
}

func validateTurbulenceConfig(config Config) error {
	if config.IAASSettingsTurbulenceStubPath == "" {
		return errors.New("missing `iaas_settings_turbulence_stub_path` - path to turbulence stub file")
	}

	if err := checkAbsolutePath(config.IAASSettingsTurbulenceStubPath, "iaas_settings_turbulence_stub_path"); err != nil {
		return err
	}

	if config.TurbulencePropertiesStubPath == "" {
		return errors.New("missing `turbulence_properties_stub_path` - path to turbulence properties stub file")
	}

	if err := checkAbsolutePath(config.TurbulencePropertiesStubPath, "turbulence_properties_stub_path"); err != nil {
		return err
	}

	if config.TurbulenceReleaseLocation == "" {
		return errors.New("missing `turbulence_release_location` - location of turbulence release")
	}

	if config.CPIReleaseLocation == "" {
		return errors.New("missing `cpi_release_location` - location of cpi release")
	}

	if config.CPIReleaseName == "" {
		return errors.New("missing `cpi_release_name` - name of cpi release")
	}

	return nil
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

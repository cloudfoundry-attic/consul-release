package fakes

import "confab"

type ServiceDefiner struct {
	GenerateDefinitionsCall struct {
		Receives struct {
			Config confab.Config
		}
		Returns struct {
			Definitions []confab.ServiceDefinition
		}
	}
	WriteDefinitionsCall struct {
		Receives struct {
			Definitions []confab.ServiceDefinition
			ConfigDir   string
		}
		Returns struct {
			Error error
		}
	}
}

func (d *ServiceDefiner) WriteDefinitions(configDir string, definitions []confab.ServiceDefinition) error {
	d.WriteDefinitionsCall.Receives.Definitions = definitions
	d.WriteDefinitionsCall.Receives.ConfigDir = configDir
	return d.WriteDefinitionsCall.Returns.Error
}

func (d *ServiceDefiner) GenerateDefinitions(config confab.Config) []confab.ServiceDefinition {
	d.GenerateDefinitionsCall.Receives.Config = config
	return d.GenerateDefinitionsCall.Returns.Definitions
}

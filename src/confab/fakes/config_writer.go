package fakes

import "github.com/cloudfoundry-incubator/consul-release/src/confab/config"

type ConfigWriter struct {
	WriteCall struct {
		Receives struct {
			Config config.Config
		}
		Returns struct {
			Error error
		}
	}
}

func (w *ConfigWriter) Write(cfg config.Config) error {
	w.WriteCall.Receives.Config = cfg

	return w.WriteCall.Returns.Error
}

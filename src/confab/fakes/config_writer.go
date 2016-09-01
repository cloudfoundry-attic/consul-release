package fakes

import "github.com/cloudfoundry-incubator/consul-release/src/confab/config"

type ConfigWriter struct {
	WriteCall struct {
		CallCount int
		Receives  struct {
			Config config.Config
		}
		Returns struct {
			Error error
		}
	}
}

func (w *ConfigWriter) Write(cfg config.Config) error {
	w.WriteCall.Receives.Config = cfg
	w.WriteCall.CallCount++

	return w.WriteCall.Returns.Error
}

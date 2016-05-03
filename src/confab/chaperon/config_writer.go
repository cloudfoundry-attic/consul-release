package chaperon

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cloudfoundry-incubator/consul-release/src/confab/config"
	"github.com/pivotal-golang/lager"
)

type ConfigWriter struct {
	dir    string
	logger logger
}

func NewConfigWriter(dir string, logger logger) ConfigWriter {
	return ConfigWriter{
		dir:    dir,
		logger: logger,
	}
}

func (w ConfigWriter) Write(cfg config.Config) error {
	w.logger.Info("config-writer.write.generate-configuration")
	consulConfig := config.GenerateConfiguration(cfg, w.dir)

	data, err := json.Marshal(&consulConfig)
	if err != nil {
		return err
	}

	w.logger.Info("config-writer.write.write-file", lager.Data{
		"config": consulConfig,
	})
	err = ioutil.WriteFile(filepath.Join(w.dir, "config.json"), data, os.ModePerm)
	if err != nil {
		w.logger.Error("config-writer.write.write-file.failed", errors.New(err.Error()))
		return err
	}

	w.logger.Info("config-writer.write.success")
	return nil
}

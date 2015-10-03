package helpers

import (
	"encoding/json"
	"os"
	"path"

	. "github.com/onsi/gomega"
)

const defaultLogLevel = "info"
const defaultProtocolVersion = 2

type configFile struct {
	BootstrapExpect    int            `json:"bootstrap_expect"`
	Datacenter         string         `json:"datacenter"`
	DataDir            string         `json:"data_dir"`
	LogLevel           string         `json:"log_level"`
	NodeName           string         `json:"node_name"`
	Server             bool           `json:"server"`
	BindAddr           string         `json:"bind_addr"`
	ProtocolVersion    int            `json:"protocol"`
	RetryJoin          []string       `json:"retry_join"`
	RejoinAfterLeave   bool           `json:"rejoin_after_leave"`
	DisableRemoteExec  bool           `json:"disable_remote_exec"`
	DisableUpdateCheck bool           `json:"disable_update_check"`
}

func newConfigFile(
dataDir string,
bindAddress string,
serverAddresses []string,
) configFile {

	return configFile{
		DataDir:            dataDir,
		LogLevel:           defaultLogLevel,
		NodeName:           "localnode",
		Server:             false,
		BindAddr:           bindAddress,
		ProtocolVersion:    defaultProtocolVersion,
		RetryJoin:          serverAddresses,
		RejoinAfterLeave:   true,
		DisableRemoteExec:  true,
		DisableUpdateCheck: true,
	}
}

func writeConfigFile(
configDir string,
dataDir string,
bindAddress string,
serverAddresses []string,
) string {
	filePath := path.Join(configDir, "config.json")
	file, err := os.Create(filePath)
	Expect(err).NotTo(HaveOccurred())

	config := newConfigFile(dataDir, bindAddress, serverAddresses)
	configJSON, err := json.Marshal(config)
	Expect(err).NotTo(HaveOccurred())

	_, err = file.Write(configJSON)
	Expect(err).NotTo(HaveOccurred())

	err = file.Close()
	Expect(err).NotTo(HaveOccurred())

	return filePath
}

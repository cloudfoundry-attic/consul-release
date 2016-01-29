package confab

import (
	"fmt"
	"strings"
)

type ConsulConfig struct {
	Server             bool              `json:"server"`
	Domain             string            `json:"domain"`
	Datacenter         string            `json:"datacenter"`
	DataDir            string            `json:"data_dir"`
	LogLevel           string            `json:"log_level"`
	NodeName           string            `json:"node_name"`
	Ports              ConsulConfigPorts `json:"ports"`
	RejoinAfterLeave   bool              `json:"rejoin_after_leave"`
	DisableRemoteExec  bool              `json:"disable_remote_exec"`
	DisableUpdateCheck bool              `json:"disable_update_check"`
	Protocol           int               `json:"protocol"`
}

type ConsulConfigPorts struct {
	DNS int `json:"dns"`
}

func GenerateConfiguration(confabConfig Config) ConsulConfig {
	return ConsulConfig{
		Server:     confabConfig.Consul.Agent.Server,
		Domain:     "cf.internal",
		Datacenter: confabConfig.Consul.Agent.Datacenter,
		DataDir:    "/var/vcap/store/consul_agent",
		LogLevel:   confabConfig.Consul.Agent.LogLevel,
		NodeName:   fmt.Sprintf("%s-%d", strings.Replace(confabConfig.Node.Name, "_", "-", -1), confabConfig.Node.Index),
		Ports: ConsulConfigPorts{
			DNS: 53,
		},
		RejoinAfterLeave:   true,
		DisableRemoteExec:  true,
		DisableUpdateCheck: true,
		Protocol:           confabConfig.Consul.Agent.ProtocolVersion,
	}
}

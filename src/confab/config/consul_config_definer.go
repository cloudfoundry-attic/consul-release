package config

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

type ConsulConfig struct {
	Server               bool              `json:"server"`
	Domain               string            `json:"domain"`
	Datacenter           string            `json:"datacenter"`
	DataDir              string            `json:"data_dir"`
	LogLevel             string            `json:"log_level"`
	NodeName             string            `json:"node_name"`
	Ports                ConsulConfigPorts `json:"ports"`
	RejoinAfterLeave     bool              `json:"rejoin_after_leave"`
	RetryJoin            []string          `json:"retry_join"`
	RetryJoinWAN         []string          `json:"retry_join_wan"`
	BindAddr             string            `json:"bind_addr"`
	DisableRemoteExec    bool              `json:"disable_remote_exec"`
	DisableUpdateCheck   bool              `json:"disable_update_check"`
	Protocol             int               `json:"protocol"`
	VerifyOutgoing       *bool             `json:"verify_outgoing,omitempty"`
	VerifyIncoming       *bool             `json:"verify_incoming,omitempty"`
	VerifyServerHostname *bool             `json:"verify_server_hostname,omitempty"`
	CAFile               *string           `json:"ca_file,omitempty"`
	KeyFile              *string           `json:"key_file,omitempty"`
	CertFile             *string           `json:"cert_file,omitempty"`
	Encrypt              *string           `json:"encrypt,omitempty"`
	BootstrapExpect      *int              `json:"bootstrap_expect,omitempty"`
}

type ConsulConfigPorts struct {
	DNS int `json:"dns"`
}

func GenerateConfiguration(config Config, configDir string) ConsulConfig {
	lan := config.Consul.Agent.Servers.LAN
	if lan == nil {
		lan = []string{}
	}

	wan := config.Consul.Agent.Servers.WAN
	if wan == nil {
		wan = []string{}
	}

	nodeName := strings.Replace(config.Node.Name, "_", "-", -1)
	nodeName = fmt.Sprintf("%s-%d", nodeName, config.Node.Index)

	isServer := config.Consul.Agent.Mode == "server"

	consulConfig := ConsulConfig{
		Server:     isServer,
		Domain:     config.Consul.Agent.Domain,
		Datacenter: config.Consul.Agent.Datacenter,
		DataDir:    "/var/vcap/store/consul_agent",
		LogLevel:   config.Consul.Agent.LogLevel,
		NodeName:   nodeName,
		Ports: ConsulConfigPorts{
			DNS: 53,
		},
		RejoinAfterLeave:   true,
		RetryJoin:          lan,
		RetryJoinWAN:       wan,
		BindAddr:           config.Node.ExternalIP,
		DisableRemoteExec:  true,
		DisableUpdateCheck: true,
		Protocol:           config.Consul.Agent.ProtocolVersion,
	}

	consulConfig.VerifyOutgoing = boolPtr(true)
	consulConfig.VerifyIncoming = boolPtr(true)
	consulConfig.VerifyServerHostname = boolPtr(true)
	certsDir := filepath.Join(configDir, "certs")
	consulConfig.CAFile = strPtr(filepath.Join(certsDir, "ca.crt"))

	if isServer {
		consulConfig.KeyFile = strPtr(filepath.Join(certsDir, "server.key"))
		consulConfig.CertFile = strPtr(filepath.Join(certsDir, "server.crt"))
	} else {
		consulConfig.KeyFile = strPtr(filepath.Join(certsDir, "agent.key"))
		consulConfig.CertFile = strPtr(filepath.Join(certsDir, "agent.crt"))
	}

	if len(config.Consul.EncryptKeys) > 0 {
		consulConfig.Encrypt = encryptKey(config.Consul.EncryptKeys[0])
	}

	if isServer {
		consulConfig.BootstrapExpect = intPtr(len(config.Consul.Agent.Servers.LAN))
	}

	return consulConfig
}

func encryptKey(key string) *string {
	decodedKey, err := base64.StdEncoding.DecodeString(key)

	if err != nil || len(decodedKey) != 16 {
		return strPtr(base64.StdEncoding.EncodeToString(pbkdf2.Key([]byte(key), []byte(""), 20000, 16, sha1.New)))
	} else {
		return strPtr(key)
	}
}

func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

func strPtr(s string) *string {
	return &s
}

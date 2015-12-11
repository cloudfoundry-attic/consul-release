package destiny

type Properties struct {
	Consul        *PropertiesConsul        `yaml:"consul,omitempty"`
	TurbulenceAPI *PropertiesTurbulenceAPI `yaml:"turbulence_api,omitempty"`
	WardenCPI     *PropertiesWardenCPI     `yaml:"warden_cpi,omitempty"`
}

type PropertiesTurbulenceAPI struct {
	Certificate string                          `yaml:"certificate"`
	CPIJobName  string                          `yaml:"cpi_job_name"`
	Director    PropertiesTurbulenceAPIDirector `yaml:"director"`
	Password    string                          `yaml:"password"`
	PrivateKey  string                          `yaml:"private_key"`
}

type PropertiesTurbulenceAPIDirector struct {
	CACert   string `yaml:"ca_cert"`
	Host     string `yaml:"host"`
	Password string `yaml:"password"`
	Username string `yaml:"username"`
}

type PropertiesWardenCPI struct {
	Agent  PropertiesWardenCPIAgent  `yaml:"agent"`
	Warden PropertiesWardenCPIWarden `yaml:"warden"`
}

type PropertiesWardenCPIAgent struct {
	Blobstore PropertiesWardenCPIAgentBlobstore `yaml:"blobstore"`
	Mbus      string                            `yaml:"mbus"`
}

type PropertiesWardenCPIAgentBlobstore struct {
	Options  PropertiesWardenCPIAgentBlobstoreOptions `yaml:"options"`
	Provider string                                   `yaml:"provider"`
}

type PropertiesWardenCPIAgentBlobstoreOptions struct {
	Endpoint string `yaml:"endpoint"`
	Password string `yaml:"password"`
	User     string `yaml:"user"`
}

type PropertiesWardenCPIWarden struct {
	ConnectAddress string `yaml:"connect_address"`
	ConnectNetwork string `yaml:"connect_network"`
}

type PropertiesConsul struct {
	Agent       PropertiesConsulAgent `yaml:"agent"`
	CACert      string                `yaml:"ca_cert"`
	AgentCert   string                `yaml:"agent_cert"`
	AgentKey    string                `yaml:"agent_key"`
	ServerCert  string                `yaml:"server_cert"`
	ServerKey   string                `yaml:"server_key"`
	EncryptKeys []string              `yaml:"encrypt_keys"`
	RequireSSL  bool                  `yaml:"require_ssl"`
}

type PropertiesConsulAgent struct {
	LogLevel string                       `yaml:"log_level,omitempty"`
	Servers  PropertiesConsulAgentServers `yaml:"servers"`
	Mode     string                       `yaml:"mode,omitempty"`
}

type PropertiesConsulAgentServers struct {
	Lan []string `yaml:"lan"`
}

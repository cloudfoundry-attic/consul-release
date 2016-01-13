package destiny

type Config struct {
	DirectorUUID string
	Name         string
	IAAS         int
	AWS          ConfigAWS
	BOSH         ConfigBOSH
	Registry     ConfigRegistry
}

type ConfigBOSH struct {
	Target         string
	Username       string
	Password       string
	DirectorCACert string
}

type ConfigAWS struct {
	AccessKeyID           string
	SecretAccessKey       string
	DefaultKeyName        string
	DefaultSecurityGroups []string
	Region                string
	Subnet                string
}

type ConfigRegistry struct {
	Host     string
	Password string
	Port     int
	Username string
}

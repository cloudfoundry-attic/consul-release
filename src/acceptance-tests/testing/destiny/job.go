package destiny

type Job struct {
	Instances      int            `yaml:"instances"`
	Lifecycle      string         `yaml:"lifecycle,omitempty"`
	Name           string         `yaml:"name"`
	Networks       []JobNetwork   `yaml:"networks"`
	ResourcePool   string         `yaml:"resource_pool"`
	Templates      []JobTemplate  `yaml:"templates"`
	PersistentDisk int            `yaml:"persistent_disk"`
	Properties     *JobProperties `yaml:"properties,omitempty"`
	Update         *JobUpdate     `yaml:"update,omitempty"`
}

type JobProperties struct {
	Consul JobPropertiesConsul `yaml:"consul"`
}

type JobPropertiesConsul struct {
	Agent JobPropertiesConsulAgent `yaml:"agent"`
}

type JobPropertiesConsulAgent struct {
	Mode string `yaml:"mode"`
}

type JobUpdate struct {
	MaxInFlight int `yaml:"max_in_flight"`
}

type JobNetwork struct {
	Name      string   `yaml:"name"`
	StaticIPs []string `yaml:"static_ips"`
}

type JobTemplate struct {
	Name    string `yaml:"name"`
	Release string `yaml:"release"`
}

func SetJobInstanceCount(job Job, network Network, properties Properties, count int) (Job, Properties) {
	job.Instances = count
	for i, net := range job.Networks {
		if net.Name == network.Name {
			net.StaticIPs = network.StaticIPs(count)
			properties.Consul.Agent.Servers.Lan = net.StaticIPs
		}
		job.Networks[i] = net
	}

	return job, properties
}

package destiny

type Network struct {
	Name    string          `yaml:"name"`
	Subnets []NetworkSubnet `yaml:"subnets"`
	Type    string          `yaml:"type"`
}

func (n Network) StaticIPs(count int) []string {
	var ips []string
	for _, subnet := range n.Subnets {
		ips = append(ips, subnet.Static...)
	}

	if len(ips) >= count {
		return ips[:count]
	}

	return []string{}
}

type NetworkSubnet struct {
	CloudProperties NetworkSubnetCloudProperties `yaml:"cloud_properties"`
	Gateway         string                       `yaml:"gateway"`
	Range           string                       `yaml:"range"`
	Reserved        []string                     `yaml:"reserved"`
	Static          []string                     `yaml:"static"`
}

type NetworkSubnetCloudProperties struct {
	Name   string `yaml:"name"`
	Subnet string `yaml:"subnet,omitempty"`
}

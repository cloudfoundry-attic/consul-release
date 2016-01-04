package destiny

func (m Manifest) ConsulMembers() []ConsulMember {
	members := []ConsulMember{}
	for _, job := range m.Jobs {
		if len(job.Networks) == 0 {
			continue
		}

		for i := 0; i < job.Instances; i++ {
			if len(job.Networks[0].StaticIPs) > i {
				members = append(members, ConsulMember{
					Address: job.Networks[0].StaticIPs[i],
				})
			}
		}
	}

	return members
}

type ConsulMember struct {
	Address string
}

func NewConsul(config Config) Manifest {
	release := Release{
		Name:    "consul",
		Version: "latest",
	}

	consulNetwork1 := Network{
		Name: "consul1",
		Subnets: []NetworkSubnet{{
			CloudProperties: NetworkSubnetCloudProperties{Name: "random"},
			Range:           "10.244.4.0/24",
			Reserved:        []string{"10.244.4.1", "10.244.4.5", "10.244.4.9", "10.244.4.13", "10.244.4.17"},
			Static:          []string{"10.244.4.2", "10.244.4.6", "10.244.4.10", "10.244.4.14", "10.244.4.18"},
		}},
		Type: "manual",
	}

	consulNetwork2 := Network{
		Name: "consul2",
		Subnets: []NetworkSubnet{{
			CloudProperties: NetworkSubnetCloudProperties{Name: "random"},
			Range:           "10.244.5.0/24",
			Reserved:        []string{"10.244.5.1", "10.244.5.5", "10.244.5.9", "10.244.5.13", "10.244.5.17"},
			Static:          []string{"10.244.5.2", "10.244.5.6", "10.244.5.10", "10.244.5.14", "10.244.5.18"},
		}},
		Type: "manual",
	}

	compilationNetwork := Network{
		Name: "compilation",
		Subnets: []NetworkSubnet{{
			CloudProperties: NetworkSubnetCloudProperties{Name: "random"},
			Range:           "10.244.6.0/24",
			Reserved:        []string{"10.244.6.1", "10.244.6.5", "10.244.6.9"},
		}},
		Type: "manual",
	}

	compilation := Compilation{
		Network:             compilationNetwork.Name,
		ReuseCompilationVMs: true,
		Workers:             3,
	}

	update := Update{
		Canaries:        1,
		CanaryWatchTime: "1000-180000",
		MaxInFlight:     50,
		Serial:          true,
		UpdateWatchTime: "1000-180000",
	}

	stemcell := ResourcePoolStemcell{
		Name:    "bosh-warden-boshlite-ubuntu-trusty-go_agent",
		Version: "latest",
	}

	z1ResourcePool := ResourcePool{
		Name:     "consul_z1",
		Network:  consulNetwork1.Name,
		Stemcell: stemcell,
	}

	z2ResourcePool := ResourcePool{
		Name:     "consul_z2",
		Network:  consulNetwork2.Name,
		Stemcell: stemcell,
	}

	z1Job := Job{
		Name:      "consul_z1",
		Instances: 1,
		Networks: []JobNetwork{{
			Name:      consulNetwork1.Name,
			StaticIPs: consulNetwork1.StaticIPs(1),
		}},
		PersistentDisk: 1024,
		Properties: &JobProperties{
			Consul: JobPropertiesConsul{
				Agent: JobPropertiesConsulAgent{
					Mode: "server",
					Services: JobPropertiesConsulAgentServices{
						"router": JobPropertiesConsulAgentService{
							Name: "gorouter",
							Check: &JobPropertiesConsulAgentServiceCheck{
								Name:     "router-check",
								Script:   "/var/vcap/jobs/router/bin/script",
								Interval: "1m",
							},
							Tags: []string{"routing"},
						},
						"cloud_controller": JobPropertiesConsulAgentService{},
					},
				},
			},
		},
		ResourcePool: z1ResourcePool.Name,
		Templates: []JobTemplate{{
			Name:    "consul_agent",
			Release: "consul",
		}},
		Update: &JobUpdate{
			MaxInFlight: 1,
		},
	}

	z2Job := Job{
		Name:      "consul_z2",
		Instances: 0,
		Networks: []JobNetwork{{
			Name: consulNetwork2.Name,
		}},
		PersistentDisk: 1024,
		Properties: &JobProperties{
			Consul: JobPropertiesConsul{
				Agent: JobPropertiesConsulAgent{
					Mode: "server",
				},
			},
		},
		ResourcePool: z2ResourcePool.Name,
		Templates: []JobTemplate{{
			Name:    "consul_agent",
			Release: "consul",
		}},
		Update: &JobUpdate{
			MaxInFlight: 1,
		},
	}

	properties := Properties{
		Consul: &PropertiesConsul{
			Agent: PropertiesConsulAgent{
				Servers: PropertiesConsulAgentServers{
					Lan: consulNetwork1.StaticIPs(1),
				},
			},
			CACert:      CACert,
			AgentCert:   AgentCert,
			AgentKey:    AgentKey,
			ServerCert:  ServerCert,
			ServerKey:   ServerKey,
			EncryptKeys: []string{EncryptKey},
			RequireSSL:  true,
		},
	}

	return Manifest{
		DirectorUUID:  config.DirectorUUID,
		Name:          config.Name,
		Releases:      []Release{release},
		Compilation:   compilation,
		Update:        update,
		ResourcePools: []ResourcePool{z1ResourcePool, z2ResourcePool},
		Jobs:          []Job{z1Job, z2Job},
		Networks:      []Network{consulNetwork1, consulNetwork2, compilationNetwork},
		Properties:    properties,
	}
}

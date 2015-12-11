package destiny

func NewTurbulence(config Config) Manifest {
	turbulenceRelease := Release{
		Name:    "turbulence",
		Version: "latest",
	}

	wardenCPIRelease := Release{
		Name:    "bosh-warden-cpi",
		Version: "latest",
	}

	turbulenceNetwork := Network{
		Name: "turbulence",
		Subnets: []NetworkSubnet{{
			CloudProperties: NetworkSubnetCloudProperties{
				Name: "random",
			},
			Range:    "10.244.7.0/24",
			Reserved: []string{"10.244.7.1"},
			Static:   []string{"10.244.7.2"},
		}},
		Type: "manual",
	}

	compilationNetwork := Network{
		Name: "compilation",
		Subnets: []NetworkSubnet{{
			CloudProperties: NetworkSubnetCloudProperties{
				Name: "random",
			},
			Range:    "10.244.8.0/24",
			Reserved: []string{"10.244.8.1", "10.244.8.5", "10.244.8.9"},
			Static:   []string{},
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
		MaxInFlight:     1,
		Serial:          true,
		UpdateWatchTime: "1000-180000",
	}

	turbulenceResourcePool := ResourcePool{
		Name:    "turbulence",
		Network: turbulenceNetwork.Name,
		Stemcell: ResourcePoolStemcell{
			Name:    "bosh-warden-boshlite-ubuntu-trusty-go_agent",
			Version: "latest",
		},
	}

	apiJob := Job{
		Instances: 1,
		Name:      "api",
		Networks: []JobNetwork{{
			Name:      turbulenceNetwork.Name,
			StaticIPs: turbulenceNetwork.StaticIPs(1),
		}},
		PersistentDisk: 1024,
		ResourcePool:   turbulenceResourcePool.Name,
		Templates: []JobTemplate{
			{
				Name:    "turbulence_api",
				Release: turbulenceRelease.Name,
			},
			{
				Name:    "warden_cpi",
				Release: wardenCPIRelease.Name,
			},
		},
	}

	properties := Properties{
		TurbulenceAPI: &PropertiesTurbulenceAPI{
			Certificate: TurbulenceAPICertificate,
			CPIJobName:  "warden_cpi",
			Director: PropertiesTurbulenceAPIDirector{
				CACert:   TurbulenceAPIDirectorCACert,
				Host:     "192.168.50.4",
				Password: "admin",
				Username: "admin",
			},
			Password:   "turbulence-password",
			PrivateKey: TurbulenceAPIPrivateKey,
		},
		WardenCPI: &PropertiesWardenCPI{
			Agent: PropertiesWardenCPIAgent{
				Blobstore: PropertiesWardenCPIAgentBlobstore{
					Options: PropertiesWardenCPIAgentBlobstoreOptions{
						Endpoint: "http://10.254.50.4:25251",
						Password: "agent-password",
						User:     "agent",
					},
					Provider: "dav",
				},
				Mbus: "nats://nats:nats-password@10.254.50.4:4222",
			},
			Warden: PropertiesWardenCPIWarden{
				ConnectAddress: "10.254.50.4:7777",
				ConnectNetwork: "tcp",
			},
		},
	}

	return Manifest{
		DirectorUUID:  config.DirectorUUID,
		Name:          config.Name,
		Releases:      []Release{turbulenceRelease, wardenCPIRelease},
		ResourcePools: []ResourcePool{turbulenceResourcePool},
		Compilation:   compilation,
		Update:        update,
		Jobs:          []Job{apiJob},
		Networks:      []Network{turbulenceNetwork, compilationNetwork},
		Properties:    properties,
	}
}

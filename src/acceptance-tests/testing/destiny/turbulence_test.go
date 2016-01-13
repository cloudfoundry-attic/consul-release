package destiny_test

import (
	"acceptance-tests/testing/destiny"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Turbulence Manifest", func() {
	Describe("NewTurbulence", func() {
		It("generates a valid Turbulence AWS manifest", func() {
			manifest := destiny.NewTurbulence(destiny.Config{
				Name:         "turbulence",
				DirectorUUID: "some-director-uuid",
				IAAS:         destiny.AWS,
				BOSH: destiny.ConfigBOSH{
					Target:         "some-bosh-target",
					Username:       "some-bosh-username",
					Password:       "some-bosh-password",
					DirectorCACert: "some-ca-cert",
				},
				AWS: destiny.ConfigAWS{
					AccessKeyID:           "some-access-key-id",
					SecretAccessKey:       "some-secret-access-key",
					DefaultKeyName:        "some-default-key-name",
					DefaultSecurityGroups: []string{"some-default-security-group1"},
					Region:                "some-region",
					Subnet:                "subnet-1234",
				},
				Registry: destiny.ConfigRegistry{
					Host:     "some-registry-host",
					Password: "some-registry-password",
					Port:     25777,
					Username: "some-registry-username",
				},
			})

			Expect(manifest).To(Equal(destiny.Manifest{
				DirectorUUID: "some-director-uuid",
				Name:         "turbulence",
				Releases: []destiny.Release{
					{
						Name:    "turbulence",
						Version: "latest",
					},
					{
						Name:    "bosh-aws-cpi",
						Version: "latest",
					},
				},
				ResourcePools: []destiny.ResourcePool{
					{
						Name:    "turbulence",
						Network: "turbulence",
						Stemcell: destiny.ResourcePoolStemcell{
							Name:    "bosh-aws-xen-hvm-ubuntu-trusty-go_agent",
							Version: "latest",
						},
						CloudProperties: destiny.ResourcePoolCloudProperties{
							InstanceType:     "m3.medium",
							AvailabilityZone: "us-east-1a",
							EphemeralDisk: &destiny.ResourcePoolCloudPropertiesEphemeralDisk{
								Size: 1024,
								Type: "gp2",
							},
						},
					},
				},
				Compilation: destiny.Compilation{
					Network:             "turbulence",
					ReuseCompilationVMs: true,
					Workers:             3,
					CloudProperties: destiny.CompilationCloudProperties{
						InstanceType:     "m3.medium",
						AvailabilityZone: "us-east-1a",
						EphemeralDisk: &destiny.CompilationCloudPropertiesEphemeralDisk{
							Size: 1024,
							Type: "gp2",
						},
					},
				},
				Update: destiny.Update{
					Canaries:        1,
					CanaryWatchTime: "1000-180000",
					MaxInFlight:     1,
					Serial:          true,
					UpdateWatchTime: "1000-180000",
				},
				Jobs: []destiny.Job{
					{
						Instances: 1,
						Name:      "api",
						Networks: []destiny.JobNetwork{
							{
								Name:      "turbulence",
								StaticIPs: []string{"10.0.4.12"},
							},
						},
						PersistentDisk: 1024,
						ResourcePool:   "turbulence",
						Templates: []destiny.JobTemplate{
							{
								Name:    "turbulence_api",
								Release: "turbulence",
							},
							{
								Name:    "aws_cpi",
								Release: "bosh-aws-cpi",
							},
						},
					},
				},
				Networks: []destiny.Network{
					{
						Name: "turbulence",
						Subnets: []destiny.NetworkSubnet{
							{
								CloudProperties: destiny.NetworkSubnetCloudProperties{
									Subnet: "subnet-1234",
								},
								Gateway: "10.0.4.1",
								Range:   "10.0.4.0/24",
								Reserved: []string{
									"10.0.4.2-10.0.4.11",
									"10.0.4.17-10.0.4.254",
								},
								Static: []string{
									"10.0.4.12",
									"10.0.4.13",
								},
							},
						},
						Type: "manual",
					},
				},
				Properties: destiny.Properties{
					TurbulenceAPI: &destiny.PropertiesTurbulenceAPI{
						Certificate: destiny.TurbulenceAPICertificate,
						CPIJobName:  "aws_cpi",
						Director: destiny.PropertiesTurbulenceAPIDirector{
							CACert:   "some-ca-cert",
							Host:     "some-bosh-target",
							Password: "some-bosh-password",
							Username: "some-bosh-username",
						},
						Password:   "turbulence-password",
						PrivateKey: destiny.TurbulenceAPIPrivateKey,
					},
					AWS: &destiny.PropertiesAWS{
						AccessKeyID:           "some-access-key-id",
						DefaultKeyName:        "some-default-key-name",
						DefaultSecurityGroups: []string{"some-default-security-group1"},
						Region:                "some-region",
						SecretAccessKey:       "some-secret-access-key",
					},
					Registry: &destiny.PropertiesRegistry{
						Host:     "some-registry-host",
						Password: "some-registry-password",
						Port:     25777,
						Username: "some-registry-username",
					},
					Blobstore: &destiny.PropertiesBlobstore{
						Address: "10.0.4.12",
						Port:    2520,
						Agent: destiny.PropertiesBlobstoreAgent{
							User:     "agent",
							Password: "agent-password",
						},
					},
					Agent: &destiny.PropertiesAgent{
						Mbus: "nats://nats:password@10.0.4.12:4222",
					},
				},
			}))
		})

		It("generates a valid Turbulence BOSH-Lite manifest", func() {
			manifest := destiny.NewTurbulence(destiny.Config{
				DirectorUUID: "some-director-uuid",
				BOSH: destiny.ConfigBOSH{
					Target:   "some-bosh-target",
					Username: "some-bosh-username",
					Password: "some-bosh-password",
				},
				Name: "turbulence",
				IAAS: destiny.Warden,
			})

			Expect(manifest).To(Equal(destiny.Manifest{
				DirectorUUID: "some-director-uuid",
				Name:         "turbulence",
				Releases: []destiny.Release{
					{
						Name:    "turbulence",
						Version: "latest",
					},
					{
						Name:    "bosh-warden-cpi",
						Version: "latest",
					},
				},
				ResourcePools: []destiny.ResourcePool{
					{
						Name:    "turbulence",
						Network: "turbulence",
						Stemcell: destiny.ResourcePoolStemcell{
							Name:    "bosh-warden-boshlite-ubuntu-trusty-go_agent",
							Version: "latest",
						},
					},
				},
				Compilation: destiny.Compilation{
					Network:             "turbulence",
					ReuseCompilationVMs: true,
					Workers:             3,
				},
				Update: destiny.Update{
					Canaries:        1,
					CanaryWatchTime: "1000-180000",
					MaxInFlight:     1,
					Serial:          true,
					UpdateWatchTime: "1000-180000",
				},
				Jobs: []destiny.Job{
					{
						Instances: 1,
						Name:      "api",
						Networks: []destiny.JobNetwork{
							{
								Name:      "turbulence",
								StaticIPs: []string{"10.244.4.12"},
							},
						},
						PersistentDisk: 1024,
						ResourcePool:   "turbulence",
						Templates: []destiny.JobTemplate{
							{
								Name:    "turbulence_api",
								Release: "turbulence",
							},
							{
								Name:    "warden_cpi",
								Release: "bosh-warden-cpi",
							},
						},
					},
				},
				Networks: []destiny.Network{
					{
						Name: "turbulence",
						Subnets: []destiny.NetworkSubnet{
							{
								CloudProperties: destiny.NetworkSubnetCloudProperties{
									Name: "random",
								},
								Gateway: "10.244.4.1",
								Range:   "10.244.4.0/24",
								Reserved: []string{
									"10.244.4.2-10.244.4.11",
									"10.244.4.17-10.244.4.254",
								},
								Static: []string{
									"10.244.4.12",
									"10.244.4.13",
								},
							},
						},
						Type: "manual",
					},
				},
				Properties: destiny.Properties{
					TurbulenceAPI: &destiny.PropertiesTurbulenceAPI{
						Certificate: destiny.TurbulenceAPICertificate,
						CPIJobName:  "warden_cpi",
						Director: destiny.PropertiesTurbulenceAPIDirector{
							CACert:   destiny.TurbulenceAPIDirectorCACert,
							Host:     "some-bosh-target",
							Password: "some-bosh-password",
							Username: "some-bosh-username",
						},
						Password:   "turbulence-password",
						PrivateKey: destiny.TurbulenceAPIPrivateKey,
					},
					WardenCPI: &destiny.PropertiesWardenCPI{
						Agent: destiny.PropertiesWardenCPIAgent{
							Blobstore: destiny.PropertiesWardenCPIAgentBlobstore{
								Options: destiny.PropertiesWardenCPIAgentBlobstoreOptions{
									Endpoint: "http://10.254.50.4:25251",
									Password: "agent-password",
									User:     "agent",
								},
								Provider: "dav",
							},
							Mbus: "nats://nats:nats-password@10.254.50.4:4222",
						},
						Warden: destiny.PropertiesWardenCPIWarden{
							ConnectAddress: "10.254.50.4:7777",
							ConnectNetwork: "tcp",
						},
					},
				},
			}))
		})
	})
})

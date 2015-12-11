package destiny_test

import (
	"acceptance-tests/testing/destiny"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Turbulence Manifest", func() {
	Describe("NewTurbulence", func() {
		It("generates a valid Turbulence BOSH-Lite manifest", func() {
			manifest := destiny.NewTurbulence(destiny.Config{
				DirectorUUID: "some-director-uuid",
				Name:         "turbulence",
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
					Network:             "compilation",
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
								StaticIPs: []string{"10.244.7.2"},
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
								Range:    "10.244.7.0/24",
								Reserved: []string{"10.244.7.1"},
								Static:   []string{"10.244.7.2"},
							},
						},
						Type: "manual",
					},
					{
						Name: "compilation",
						Subnets: []destiny.NetworkSubnet{
							{
								CloudProperties: destiny.NetworkSubnetCloudProperties{
									Name: "random",
								},
								Range:    "10.244.8.0/24",
								Reserved: []string{"10.244.8.1", "10.244.8.5", "10.244.8.9"},
								Static:   []string{},
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
							Host:     "192.168.50.4",
							Password: "admin",
							Username: "admin",
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

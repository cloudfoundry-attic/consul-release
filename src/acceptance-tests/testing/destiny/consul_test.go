package destiny_test

import (
	"acceptance-tests/testing/destiny"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Consul Manifest", func() {
	Describe("NewConsul", func() {
		It("generates a valid Consul BOSH-Lite manifest", func() {
			manifest := destiny.NewConsul(destiny.Config{
				DirectorUUID: "some-director-uuid",
				Name:         "consul-some-random-guid",
			})

			Expect(manifest).To(Equal(destiny.Manifest{
				DirectorUUID: "some-director-uuid",
				Name:         "consul-some-random-guid",
				Releases: []destiny.Release{{
					Name:    "consul",
					Version: "latest",
				}},
				Compilation: destiny.Compilation{
					Network:             "compilation",
					ReuseCompilationVMs: true,
					Workers:             3,
				},
				Update: destiny.Update{
					Canaries:        1,
					CanaryWatchTime: "1000-180000",
					MaxInFlight:     50,
					Serial:          true,
					UpdateWatchTime: "1000-180000",
				},
				ResourcePools: []destiny.ResourcePool{
					{
						Name:    "consul_z1",
						Network: "consul1",
						Stemcell: destiny.ResourcePoolStemcell{
							Name:    "bosh-warden-boshlite-ubuntu-trusty-go_agent",
							Version: "latest",
						},
					},
					{
						Name:    "consul_z2",
						Network: "consul2",
						Stemcell: destiny.ResourcePoolStemcell{
							Name:    "bosh-warden-boshlite-ubuntu-trusty-go_agent",
							Version: "latest",
						},
					},
				},
				Jobs: []destiny.Job{
					{
						Name:      "consul_z1",
						Instances: 1,
						Networks: []destiny.JobNetwork{{
							Name:      "consul1",
							StaticIPs: []string{"10.244.4.2"},
						}},
						PersistentDisk: 1024,
						Properties: &destiny.JobProperties{
							Consul: destiny.JobPropertiesConsul{
								Agent: destiny.JobPropertiesConsulAgent{
									Mode: "server",
									Services: destiny.JobPropertiesConsulAgentServices{
										"router": destiny.JobPropertiesConsulAgentService{
											Name: "gorouter",
											Check: &destiny.JobPropertiesConsulAgentServiceCheck{
												Name:     "router-check",
												Script:   "/var/vcap/jobs/router/bin/script",
												Interval: "1m",
											},
											Tags: []string{"routing"},
										},
										"cloud_controller": destiny.JobPropertiesConsulAgentService{},
									},
								},
							},
						},
						ResourcePool: "consul_z1",
						Templates: []destiny.JobTemplate{{
							Name:    "consul_agent",
							Release: "consul",
						}},
						Update: &destiny.JobUpdate{
							MaxInFlight: 1,
						},
					},
					{
						Name:      "consul_z2",
						Instances: 0,
						Networks: []destiny.JobNetwork{{
							Name: "consul2",
						}},
						PersistentDisk: 1024,
						Properties: &destiny.JobProperties{
							Consul: destiny.JobPropertiesConsul{
								Agent: destiny.JobPropertiesConsulAgent{
									Mode: "server",
								},
							},
						},
						ResourcePool: "consul_z2",
						Templates: []destiny.JobTemplate{{
							Name:    "consul_agent",
							Release: "consul",
						}},
						Update: &destiny.JobUpdate{
							MaxInFlight: 1,
						},
					},
				},
				Networks: []destiny.Network{
					{
						Name: "consul1",
						Subnets: []destiny.NetworkSubnet{
							{
								CloudProperties: destiny.NetworkSubnetCloudProperties{Name: "random"},
								Range:           "10.244.4.0/24",
								Reserved:        []string{"10.244.4.1", "10.244.4.5", "10.244.4.9", "10.244.4.13", "10.244.4.17"},
								Static:          []string{"10.244.4.2", "10.244.4.6", "10.244.4.10", "10.244.4.14", "10.244.4.18"},
							},
						},
						Type: "manual",
					},
					{
						Name: "consul2",
						Subnets: []destiny.NetworkSubnet{
							{
								CloudProperties: destiny.NetworkSubnetCloudProperties{Name: "random"},
								Range:           "10.244.5.0/24",
								Reserved:        []string{"10.244.5.1", "10.244.5.5", "10.244.5.9", "10.244.5.13", "10.244.5.17"},
								Static:          []string{"10.244.5.2", "10.244.5.6", "10.244.5.10", "10.244.5.14", "10.244.5.18"},
							},
						},
						Type: "manual",
					},
					{
						Name: "compilation",
						Subnets: []destiny.NetworkSubnet{
							{
								CloudProperties: destiny.NetworkSubnetCloudProperties{Name: "random"},
								Range:           "10.244.6.0/24",
								Reserved:        []string{"10.244.6.1", "10.244.6.5", "10.244.6.9"},
							},
						},
						Type: "manual",
					},
				},
				Properties: destiny.Properties{
					Consul: &destiny.PropertiesConsul{
						Agent: destiny.PropertiesConsulAgent{
							LogLevel: "",
							Servers: destiny.PropertiesConsulAgentServers{
								Lan: []string{"10.244.4.2"},
							},
						},
						CACert:      destiny.CACert,
						AgentCert:   destiny.AgentCert,
						AgentKey:    destiny.AgentKey,
						ServerCert:  destiny.ServerCert,
						ServerKey:   destiny.ServerKey,
						EncryptKeys: []string{destiny.EncryptKey},
						RequireSSL:  false,
					},
				},
			}))
		})
	})

	Describe("ConsulMembers", func() {
		Context("when there is a single job with a single instance", func() {
			It("returns a list of members in the cluster", func() {
				manifest := destiny.Manifest{
					Jobs: []destiny.Job{
						{
							Instances: 1,
							Networks: []destiny.JobNetwork{{
								StaticIPs: []string{"10.244.4.2"},
							}},
						},
					},
				}

				members := manifest.ConsulMembers()
				Expect(members).To(Equal([]destiny.ConsulMember{{
					Address: "10.244.4.2",
				}}))
			})
		})

		Context("when there are multiple jobs with multiple instances", func() {
			It("returns a list of members in the cluster", func() {
				manifest := destiny.Manifest{
					Jobs: []destiny.Job{
						{
							Instances: 0,
						},
						{
							Instances: 1,
							Networks: []destiny.JobNetwork{{
								StaticIPs: []string{"10.244.4.2"},
							}},
						},
						{
							Instances: 2,
							Networks: []destiny.JobNetwork{{
								StaticIPs: []string{"10.244.5.2", "10.244.5.6"},
							}},
						},
					},
				}

				members := manifest.ConsulMembers()
				Expect(members).To(Equal([]destiny.ConsulMember{
					{
						Address: "10.244.4.2",
					},
					{
						Address: "10.244.5.2",
					},
					{
						Address: "10.244.5.6",
					},
				}))
			})
		})

		Context("when the job does not have a network", func() {
			It("returns an empty list", func() {
				manifest := destiny.Manifest{
					Jobs: []destiny.Job{
						{
							Instances: 1,
							Networks:  []destiny.JobNetwork{},
						},
					},
				}

				members := manifest.ConsulMembers()
				Expect(members).To(BeEmpty())
			})
		})

		Context("when the job network does not have enough static IPs", func() {
			It("returns as much about the list as possible", func() {
				manifest := destiny.Manifest{
					Jobs: []destiny.Job{
						{
							Instances: 2,
							Networks: []destiny.JobNetwork{{
								StaticIPs: []string{"10.244.5.2"},
							}},
						},
					},
				}

				members := manifest.ConsulMembers()
				Expect(members).To(Equal([]destiny.ConsulMember{
					{
						Address: "10.244.5.2",
					},
				}))
			})
		})
	})
})

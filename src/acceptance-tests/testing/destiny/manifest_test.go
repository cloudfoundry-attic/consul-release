package destiny_test

import (
	"acceptance-tests/testing/destiny"
	"io/ioutil"

	. "acceptance-tests/testing/matchers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manifest", func() {
	Describe("ToYAML", func() {
		It("returns a YAML representation of the consul manifest", func() {
			consulManifest, err := ioutil.ReadFile("fixtures/consul_manifest.yml")
			Expect(err).NotTo(HaveOccurred())

			manifest := destiny.NewConsul(destiny.Config{
				DirectorUUID: "some-director-uuid",
				Name:         "consul",
			})

			yaml, err := manifest.ToYAML()
			Expect(err).NotTo(HaveOccurred())
			Expect(yaml).To(MatchYAML(consulManifest))
		})

		It("returns a YAML representation of the turbulence manifest", func() {
			turbulenceManifest, err := ioutil.ReadFile("fixtures/turbulence_manifest.yml")
			Expect(err).NotTo(HaveOccurred())

			manifest := destiny.NewTurbulence(destiny.Config{
				DirectorUUID: "some-director-uuid",
				Name:         "turbulence",
			})

			yaml, err := manifest.ToYAML()
			Expect(err).NotTo(HaveOccurred())
			Expect(yaml).To(MatchYAML(turbulenceManifest))
		})
	})

	Describe("FromYAML", func() {
		It("returns a Manifest matching the given YAML", func() {
			consulManifest, err := ioutil.ReadFile("fixtures/consul_manifest.yml")
			Expect(err).NotTo(HaveOccurred())

			manifest, err := destiny.FromYAML(consulManifest)
			Expect(err).NotTo(HaveOccurred())

			Expect(manifest).To(Equal(destiny.Manifest{
				DirectorUUID: "some-director-uuid",
				Name:         "consul",
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
							Name:      "consul2",
							StaticIPs: []string{},
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
								Static:          []string{},
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

		Context("failure cases", func() {
			It("should error on malformed YAML", func() {
				_, err := destiny.FromYAML([]byte("%%%%%%%%%%"))
				Expect(err).To(MatchError(ContainSubstring("yaml: ")))
			})
		})
	})
})

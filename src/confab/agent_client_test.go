package confab_test

import (
	"confab"
	"confab/fakes"
	"errors"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/command/agent"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("agent client", func() {
	// type agentClient interface {
	// 	VerifyJoined() error
	// 	VerifySynced() error
	// 	IsLastNode() (bool, error)
	// 	SetKeys([]string) error
	//  Leave() error
	// }

	Describe("VerifyJoined", func() {
		//TODO pull things out into a Before
		Context("when the set of members includes at least one that we expect", func() {
			It("succeeds", func() {
				consulAPIAgent := new(fakes.FakeconsulAPIAgent)
				consulAPIAgent.MembersReturns([]*api.AgentMember{
					&api.AgentMember{Addr: "member1"},
					&api.AgentMember{Addr: "member2"},
					&api.AgentMember{Addr: "member3"}}, nil)

				client := confab.AgentClient{
					ExpectedMembers: []string{"member1", "member2", "member3"},
					ConsulAPIAgent:  consulAPIAgent,
				}

				Expect(client.VerifyJoined()).To(Succeed())
				Expect(consulAPIAgent.MembersArgsForCall(0)).To(BeFalse())
			})
		})
		Context("when the members are all strangers", func() {
			It("returns an error", func() {
				consulAPIAgent := new(fakes.FakeconsulAPIAgent)
				consulAPIAgent.MembersReturns([]*api.AgentMember{
					&api.AgentMember{Addr: "member1"},
					&api.AgentMember{Addr: "member2"},
					&api.AgentMember{Addr: "member3"}}, nil)

				client := confab.AgentClient{
					ExpectedMembers: []string{"member4", "member5"},
					ConsulAPIAgent:  consulAPIAgent,
				}

				Expect(client.VerifyJoined()).To(MatchError("no expected members"))
				Expect(consulAPIAgent.MembersArgsForCall(0)).To(BeFalse())
			})
		})
		Context("when the members call fails", func() {
			It("returns an error", func() {
				consulAPIAgent := new(fakes.FakeconsulAPIAgent)
				consulAPIAgent.MembersReturns([]*api.AgentMember{}, errors.New("members call error"))

				client := confab.AgentClient{
					ExpectedMembers: []string{},
					ConsulAPIAgent:  consulAPIAgent,
				}

				Expect(client.VerifyJoined()).To(MatchError("members call error"))
				Expect(consulAPIAgent.MembersArgsForCall(0)).To(BeFalse())
			})
		})
	})

	Describe("VerifySynced", func() {
		var (
			expectedStats   map[string]map[string]string
			consulRPCClient *fakes.FakeconsulRPCClient
			client          confab.AgentClient
		)

		BeforeEach(func() {
			expectedStats = map[string]map[string]string{
				"raft": map[string]string{
					"commit_index":   "2",
					"last_log_index": "2",
				},
			}

			consulRPCClient = new(fakes.FakeconsulRPCClient)
			consulRPCClient.StatsReturns(expectedStats, nil)

			client = confab.AgentClient{
				ConsulRPCClient: consulRPCClient,
			}
		})

		It("verifies the sync state of the raft log", func() {
			Expect(client.VerifySynced()).To(Succeed())
			Expect(consulRPCClient.StatsCallCount()).To(Equal(1))
		})

		Context("when the last_log_index never catches up", func() {
			BeforeEach(func() {
				expectedStats = map[string]map[string]string{
					"raft": map[string]string{
						"commit_index":   "2",
						"last_log_index": "1",
					},
				}

				consulRPCClient = new(fakes.FakeconsulRPCClient)
				consulRPCClient.StatsReturns(expectedStats, nil)

				client = confab.AgentClient{
					ConsulRPCClient: consulRPCClient,
				}
			})

			It("returns an error", func() {
				//TODO return a reasonably named error
				Expect(client.VerifySynced()).To(MatchError("Log not in sync"))
				Expect(consulRPCClient.StatsCallCount()).To(Equal(1))
			})
		})

		Context("when the RPCClient returns an error", func() {
			BeforeEach(func() {
				consulRPCClient = new(fakes.FakeconsulRPCClient)
				consulRPCClient.StatsReturns(nil, errors.New("RPC error"))

				client = confab.AgentClient{
					ConsulRPCClient: consulRPCClient,
				}
			})

			It("immediately returns an error", func() {
				Expect(client.VerifySynced()).To(MatchError("RPC error"))
				Expect(consulRPCClient.StatsCallCount()).To(Equal(1))
			})
		})

		Context("when the commit index is 0", func() {
			BeforeEach(func() {
				expectedStats = map[string]map[string]string{
					"raft": map[string]string{
						"commit_index":   "0",
						"last_log_index": "0",
					},
				}

				consulRPCClient = new(fakes.FakeconsulRPCClient)
				consulRPCClient.StatsReturns(expectedStats, nil)

				client = confab.AgentClient{
					ConsulRPCClient: consulRPCClient,
				}
			})

			It("immediately returns an error", func() {
				Expect(client.VerifySynced()).To(MatchError("Commit index must not be zero"))
				Expect(consulRPCClient.StatsCallCount()).To(Equal(1))
			})
		})
	})

	Describe("IsLastNode", func() {
		var (
			consulAPIAgent *fakes.FakeconsulAPIAgent
			client         confab.AgentClient
		)

		BeforeEach(func() {
			consulAPIAgent = new(fakes.FakeconsulAPIAgent)
			consulAPIAgent.MembersReturns([]*api.AgentMember{
				&api.AgentMember{Addr: "member1", Tags: map[string]string{"role": "consul"}},
				&api.AgentMember{Addr: "member2", Tags: map[string]string{"role": "consul"}},
				&api.AgentMember{Addr: "member3", Tags: map[string]string{"role": "consul"}}}, nil)

			client = confab.AgentClient{
				ExpectedMembers: []string{"member1", "member2", "member3"},
				ConsulAPIAgent:  consulAPIAgent,
			}
		})

		It("returns true", func() {
			Expect(client.IsLastNode()).To(BeTrue())
			Expect(consulAPIAgent.MembersCallCount()).To(Equal(1))
		})

		Context("When you are not the last node", func() {
			BeforeEach(func() {
				consulAPIAgent.MembersReturns([]*api.AgentMember{
					&api.AgentMember{Addr: "member1", Tags: map[string]string{"role": "consul"}},
					&api.AgentMember{Addr: "member2", Tags: map[string]string{"role": "consul"}}}, nil)
			})

			It("returns false", func() {
				Expect(client.IsLastNode()).To(BeFalse())
				Expect(consulAPIAgent.MembersCallCount()).To(Equal(1))
			})

			Context("when there are non-server members", func() {
				BeforeEach(func() {
					consulAPIAgent.MembersReturns([]*api.AgentMember{
						&api.AgentMember{Addr: "member1", Tags: map[string]string{"role": "consul"}},
						&api.AgentMember{Addr: "member2", Tags: map[string]string{"role": "node"}},
						&api.AgentMember{Addr: "member3", Tags: map[string]string{"role": "consul"}}}, nil)
				})

				It("returns false", func() {
					Expect(client.IsLastNode()).To(BeFalse())
					Expect(consulAPIAgent.MembersCallCount()).To(Equal(1))
				})

			})
		})

		Context("When members returns an error", func() {
			BeforeEach(func() {
				consulAPIAgent.MembersReturns([]*api.AgentMember{}, errors.New("members error"))
			})

			It("returns an error", func() {
				_, err := client.IsLastNode()
				Expect(err).To(MatchError("members error"))
				Expect(consulAPIAgent.MembersCallCount()).To(Equal(1))
			})
		})
	})

	Describe("SetKeys", func() {
		var (
			consulRPCClient *fakes.FakeconsulRPCClient
			client          confab.AgentClient
		)

		BeforeEach(func() {
			consulRPCClient = new(fakes.FakeconsulRPCClient)
			consulRPCClient.InstallKeyReturns(confab.KeyringResponse{}, nil)
			consulRPCClient.UseKeyReturns(confab.KeyringResponse{}, nil)
			consulRPCClient.ListKeysReturns(confab.KeyringResponse{
				Keys: []agent.KeyringEntry{
					{Key: "key3"},
					{Key: "key4"},
				},
			}, nil)
			consulRPCClient.RemoveKeyReturns(confab.KeyringResponse{}, nil)

			client = confab.AgentClient{
				ConsulRPCClient: consulRPCClient,
			}
		})

		It("installs the given keys", func() {
			Expect(client.SetKeys([]string{"key1", "key2"})).To(Succeed())
			Expect(consulRPCClient.InstallKeyCallCount()).To(Equal(2))

			key, token := consulRPCClient.InstallKeyArgsForCall(0)
			Expect(key).To(Equal("key1"))
			Expect(token).To(Equal(""))

			key, token = consulRPCClient.InstallKeyArgsForCall(1)
			Expect(key).To(Equal("key2"))
			Expect(token).To(Equal(""))

			Expect(consulRPCClient.UseKeyCallCount()).To(Equal(1))

			key, token = consulRPCClient.UseKeyArgsForCall(0)
			Expect(key).To(Equal("key1"))
			Expect(token).To(Equal(""))
		})

		Context("when there are extra keys", func() {
			It("removes extra keys", func() {
				Expect(client.SetKeys([]string{"key1", "key2"})).To(Succeed())
				Expect(consulRPCClient.ListKeysCallCount()).To(Equal(1))
				Expect(consulRPCClient.ListKeysArgsForCall(0)).To(Equal(""))

				Expect(consulRPCClient.RemoveKeyCallCount()).To(Equal(2))

				key, token := consulRPCClient.RemoveKeyArgsForCall(0)
				Expect(key).To(Equal("key3"))
				Expect(token).To(Equal(""))

				key, token = consulRPCClient.RemoveKeyArgsForCall(1)
				Expect(key).To(Equal("key4"))
				Expect(token).To(Equal(""))
			})
		})

		Context("failure cases", func() {
			Context("when ListKeys returns an error", func() {
				It("returns the error", func() {
					consulRPCClient.ListKeysReturns(confab.KeyringResponse{}, errors.New("list keys error"))

					Expect(client.SetKeys([]string{"key1"})).To(MatchError("list keys error"))
				})
			})

			Context("when RemoveKeys returns an error", func() {
				It("returns the error", func() {
					consulRPCClient.ListKeysReturns(confab.KeyringResponse{
						Keys: []agent.KeyringEntry{
							{Key: "key3"},
							{Key: "key4"},
						},
					}, nil)
					consulRPCClient.RemoveKeyReturns(confab.KeyringResponse{}, errors.New("remove key error"))

					Expect(client.SetKeys([]string{"key1"})).To(MatchError("remove key error"))
				})
			})

			Context("when InstallKey returns an error", func() {
				It("returns the error", func() {
					consulRPCClient.InstallKeyReturns(confab.KeyringResponse{}, errors.New("install key error"))

					Expect(client.SetKeys([]string{"key1"})).To(MatchError("install key error"))
				})
			})

			Context("when UseKey returns an error", func() {
				It("returns the error", func() {
					consulRPCClient.UseKeyReturns(confab.KeyringResponse{}, errors.New("use key error"))

					Expect(client.SetKeys([]string{"key1"})).To(MatchError("use key error"))
				})
			})
		})
	})

	Describe("Leave", func() {
		var (
			consulRPCClient *fakes.FakeconsulRPCClient
			client          confab.AgentClient
		)

		BeforeEach(func() {
			consulRPCClient = new(fakes.FakeconsulRPCClient)
			client = confab.AgentClient{
				ConsulRPCClient: consulRPCClient,
			}
		})

		It("leaves the cluster", func() {
			Expect(client.Leave()).To(Succeed())
			Expect(consulRPCClient.LeaveCallCount()).To(Equal(1))
		})

		Context("when RPCClient.leave returns an error", func() {
			It("returns an error", func() {
				consulRPCClient.LeaveReturns(errors.New("leave error"))

				Expect(client.Leave()).To(MatchError("leave error"))
				Expect(consulRPCClient.LeaveCallCount()).To(Equal(1))
			})
		})
	})
})

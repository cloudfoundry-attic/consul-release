package confab_test

import (
	"confab"
	"confab/fakes"
	"errors"

	"github.com/pivotal-golang/lager"

	"github.com/hashicorp/consul/api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AgentClient", func() {
	var (
		consulAPIAgent  *fakes.FakeconsulAPIAgent
		consulRPCClient *fakes.FakeconsulRPCClient
		logger          *fakes.Logger
		client          confab.AgentClient
	)

	BeforeEach(func() {
		consulAPIAgent = &fakes.FakeconsulAPIAgent{}
		consulRPCClient = &fakes.FakeconsulRPCClient{}
		logger = &fakes.Logger{}
		client = confab.AgentClient{
			ConsulAPIAgent:  consulAPIAgent,
			ConsulRPCClient: consulRPCClient,
			Logger:          logger,
		}
	})

	Describe("VerifyJoined", func() {
		Context("when the set of members includes at least one that we expect", func() {
			It("succeeds", func() {
				client.ExpectedMembers = []string{"member1", "member2", "member3"}
				consulAPIAgent.MembersReturns([]*api.AgentMember{
					&api.AgentMember{Addr: "member1"},
					&api.AgentMember{Addr: "member2"},
					&api.AgentMember{Addr: "member3"},
				}, nil)

				Expect(client.VerifyJoined()).To(Succeed())
				Expect(consulAPIAgent.MembersArgsForCall(0)).To(BeFalse())

				Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "agent-client.verify-joined.members.request",
						Data: []lager.Data{{
							"wan": false,
						}},
					},
					{
						Action: "agent-client.verify-joined.members.response",
						Data: []lager.Data{{
							"wan":     false,
							"members": []string{"member1", "member2", "member3"},
						}},
					},
					{
						Action: "agent-client.verify-joined.members.joined",
					},
				}))
			})
		})

		Context("when the members are all strangers", func() {
			It("returns an error", func() {
				client.ExpectedMembers = []string{"member1", "member2", "member3"}
				consulAPIAgent.MembersReturns([]*api.AgentMember{
					&api.AgentMember{Addr: "member4"},
					&api.AgentMember{Addr: "member5"},
				}, nil)

				Expect(client.VerifyJoined()).To(MatchError("no expected members"))
				Expect(consulAPIAgent.MembersArgsForCall(0)).To(BeFalse())

				Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "agent-client.verify-joined.members.request",
						Data: []lager.Data{{
							"wan": false,
						}},
					},
					{
						Action: "agent-client.verify-joined.members.response",
						Data: []lager.Data{{
							"wan":     false,
							"members": []string{"member4", "member5"},
						}},
					},
					{
						Action: "agent-client.verify-joined.members.not-joined",
						Error:  errors.New("no expected members"),
						Data: []lager.Data{{
							"wan":     false,
							"members": []string{"member4", "member5"},
						}},
					},
				}))
			})
		})

		Context("when the members call fails", func() {
			It("returns an error", func() {
				consulAPIAgent.MembersReturns([]*api.AgentMember{}, errors.New("members call error"))
				client.ExpectedMembers = []string{}

				Expect(client.VerifyJoined()).To(MatchError("members call error"))
				Expect(consulAPIAgent.MembersArgsForCall(0)).To(BeFalse())

				Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "agent-client.verify-joined.members.request",
						Data: []lager.Data{{
							"wan": false,
						}},
					},
					{
						Action: "agent-client.verify-joined.members.request.failed",
						Error:  errors.New("members call error"),
						Data: []lager.Data{{
							"wan": false,
						}},
					},
				}))
			})
		})
	})

	Describe("VerifySynced", func() {
		BeforeEach(func() {
			consulRPCClient.StatsReturns(map[string]map[string]string{
				"raft": map[string]string{
					"commit_index":   "2",
					"last_log_index": "2",
				},
			}, nil)
		})

		It("verifies the sync state of the raft log", func() {
			Expect(client.VerifySynced()).To(Succeed())
			Expect(consulRPCClient.StatsCallCount()).To(Equal(1))
			Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
				{
					Action: "agent-client.verify-synced.stats.request",
				},
				{
					Action: "agent-client.verify-synced.stats.response",
					Data: []lager.Data{{
						"commit_index":   "2",
						"last_log_index": "2",
					}},
				},
				{
					Action: "agent-client.verify-synced.synced",
				},
			}))
		})

		Context("when the last_log_index never catches up", func() {
			BeforeEach(func() {
				consulRPCClient.StatsReturns(map[string]map[string]string{
					"raft": map[string]string{
						"commit_index":   "2",
						"last_log_index": "1",
					},
				}, nil)
			})

			It("returns an error", func() {
				Expect(client.VerifySynced()).To(MatchError("log not in sync"))
				Expect(consulRPCClient.StatsCallCount()).To(Equal(1))
				Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "agent-client.verify-synced.stats.request",
					},
					{
						Action: "agent-client.verify-synced.stats.response",
						Data: []lager.Data{{
							"commit_index":   "2",
							"last_log_index": "1",
						}},
					},
					{
						Action: "agent-client.verify-synced.not-synced",
						Error:  errors.New("log not in sync"),
					},
				}))
			})
		})

		Context("when the RPCClient returns an error", func() {
			BeforeEach(func() {
				consulRPCClient.StatsReturns(nil, errors.New("RPC error"))
			})

			It("immediately returns an error", func() {
				Expect(client.VerifySynced()).To(MatchError("RPC error"))
				Expect(consulRPCClient.StatsCallCount()).To(Equal(1))
				Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "agent-client.verify-synced.stats.request",
					},
					{
						Action: "agent-client.verify-synced.stats.request.failed",
						Error:  errors.New("RPC error"),
					},
				}))
			})
		})

		Context("when the commit index is 0", func() {
			BeforeEach(func() {
				consulRPCClient.StatsReturns(map[string]map[string]string{
					"raft": map[string]string{
						"commit_index":   "0",
						"last_log_index": "0",
					},
				}, nil)
			})

			It("immediately returns an error", func() {
				Expect(client.VerifySynced()).To(MatchError("commit index must not be zero"))
				Expect(consulRPCClient.StatsCallCount()).To(Equal(1))
				Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "agent-client.verify-synced.stats.request",
					},
					{
						Action: "agent-client.verify-synced.stats.response",
						Data: []lager.Data{{
							"commit_index":   "0",
							"last_log_index": "0",
						}},
					},
					{
						Action: "agent-client.verify-synced.zero-index",
						Error:  errors.New("commit index must not be zero"),
					},
				}))
			})
		})
	})

	Describe("IsLastNode", func() {
		BeforeEach(func() {
			consulAPIAgent.MembersReturns([]*api.AgentMember{
				&api.AgentMember{Addr: "member1", Tags: map[string]string{"role": "consul"}},
				&api.AgentMember{Addr: "member2", Tags: map[string]string{"role": "consul"}},
				&api.AgentMember{Addr: "member3", Tags: map[string]string{"role": "consul"}},
			}, nil)

			client.ExpectedMembers = []string{"member1", "member2", "member3"}
		})

		It("returns true", func() {
			Expect(client.IsLastNode()).To(BeTrue())
			Expect(consulAPIAgent.MembersCallCount()).To(Equal(1))
			Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
				{
					Action: "agent-client.is-last-node.members.request",
					Data: []lager.Data{{
						"wan": false,
					}},
				},
				{
					Action: "agent-client.is-last-node.members.response",
					Data: []lager.Data{{
						"wan":     false,
						"members": []string{"member1", "member2", "member3"},
					}},
				},
				{
					Action: "agent-client.is-last-node.result",
					Data: []lager.Data{{
						"actual_members_count":   3,
						"expected_members_count": 3,
						"is_last_node":           true,
					}},
				},
			}))
		})

		Context("When you are not the last node", func() {
			BeforeEach(func() {
				consulAPIAgent.MembersReturns([]*api.AgentMember{
					&api.AgentMember{Addr: "member1", Tags: map[string]string{"role": "consul"}},
					&api.AgentMember{Addr: "member2", Tags: map[string]string{"role": "consul"}},
				}, nil)
			})

			It("returns false", func() {
				Expect(client.IsLastNode()).To(BeFalse())
				Expect(consulAPIAgent.MembersCallCount()).To(Equal(1))
				Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "agent-client.is-last-node.members.request",
						Data: []lager.Data{{
							"wan": false,
						}},
					},
					{
						Action: "agent-client.is-last-node.members.response",
						Data: []lager.Data{{
							"wan":     false,
							"members": []string{"member1", "member2"},
						}},
					},
					{
						Action: "agent-client.is-last-node.result",
						Data: []lager.Data{{
							"actual_members_count":   2,
							"expected_members_count": 3,
							"is_last_node":           false,
						}},
					},
				}))

			})

			Context("when there are non-server members", func() {
				BeforeEach(func() {
					consulAPIAgent.MembersReturns([]*api.AgentMember{
						&api.AgentMember{Addr: "member1", Tags: map[string]string{"role": "consul"}},
						&api.AgentMember{Addr: "member2", Tags: map[string]string{"role": "node"}},
						&api.AgentMember{Addr: "member3", Tags: map[string]string{"role": "consul"}},
					}, nil)
				})

				It("returns false", func() {
					Expect(client.IsLastNode()).To(BeFalse())
					Expect(consulAPIAgent.MembersCallCount()).To(Equal(1))
					Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
						{
							Action: "agent-client.is-last-node.members.request",
							Data: []lager.Data{{
								"wan": false,
							}},
						},
						{
							Action: "agent-client.is-last-node.members.response",
							Data: []lager.Data{{
								"wan":     false,
								"members": []string{"member1", "member2", "member3"},
							}},
						},
						{
							Action: "agent-client.is-last-node.result",
							Data: []lager.Data{{
								"actual_members_count":   2,
								"expected_members_count": 3,
								"is_last_node":           false,
							}},
						},
					}))
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
				Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "agent-client.is-last-node.members.request",
						Data: []lager.Data{{
							"wan": false,
						}},
					},
					{
						Action: "agent-client.is-last-node.members.request.failed",
						Error:  errors.New("members error"),
						Data: []lager.Data{{
							"wan": false,
						}},
					},
				}))
			})
		})
	})

	Describe("SetKeys", func() {
		encryptedKey1 := "5v4WCjw2FyuezPYYUvo0zA=="
		encryptedKey2 := "gcC8kpXH4sUwLaxtiz2mBw=="
		encryptedKeyPercent := "OLJdB+hlOnGSUEIR7S6ekA=="

		BeforeEach(func() {
			consulRPCClient.InstallKeyReturns(nil)
			consulRPCClient.UseKeyReturns(nil)
			consulRPCClient.ListKeysReturns([]string{}, nil)
			consulRPCClient.RemoveKeyReturns(nil)
		})

		It("installs the given keys", func() {
			Expect(client.SetKeys([]string{encryptedKey1, "key2", "key%%"})).To(Succeed())
			Expect(consulRPCClient.InstallKeyCallCount()).To(Equal(3))

			key := consulRPCClient.InstallKeyArgsForCall(0)
			Expect(key).To(Equal(encryptedKey1))

			key = consulRPCClient.InstallKeyArgsForCall(1)
			Expect(key).To(Equal(encryptedKey2))

			key = consulRPCClient.InstallKeyArgsForCall(2)
			Expect(key).To(Equal(encryptedKeyPercent))

			Expect(consulRPCClient.UseKeyCallCount()).To(Equal(1))

			key = consulRPCClient.UseKeyArgsForCall(0)
			Expect(key).To(Equal(encryptedKey1))

			Expect(consulRPCClient.RemoveKeyCallCount()).To(Equal(0))

			Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
				{
					Action: "agent-client.set-keys.list-keys.request",
				},
				{
					Action: "agent-client.set-keys.list-keys.response",
					Data: []lager.Data{{
						"keys": []string{},
					}},
				},
				{
					Action: "agent-client.set-keys.install-key.request",
					Data: []lager.Data{{
						"key": encryptedKey1,
					}},
				},
				{
					Action: "agent-client.set-keys.install-key.response",
					Data: []lager.Data{{
						"key": encryptedKey1,
					}},
				},
				{
					Action: "agent-client.set-keys.install-key.request",
					Data: []lager.Data{{
						"key": encryptedKey2,
					}},
				},
				{
					Action: "agent-client.set-keys.install-key.response",
					Data: []lager.Data{{
						"key": encryptedKey2,
					}},
				},
				{
					Action: "agent-client.set-keys.install-key.request",
					Data: []lager.Data{{
						"key": encryptedKeyPercent,
					}},
				},
				{
					Action: "agent-client.set-keys.install-key.response",
					Data: []lager.Data{{
						"key": encryptedKeyPercent,
					}},
				},
				{
					Action: "agent-client.set-keys.use-key.request",
					Data: []lager.Data{{
						"key": encryptedKey1,
					}},
				},
				{
					Action: "agent-client.set-keys.use-key.response",
					Data: []lager.Data{{
						"key": encryptedKey1,
					}},
				},
				{
					Action: "agent-client.set-keys.success",
				},
			}))
		})

		Context("when there are extra keys", func() {
			It("removes extra keys", func() {
				consulRPCClient.ListKeysReturns([]string{"key3", "key4"}, nil)

				Expect(client.SetKeys([]string{"key1", "key2"})).To(Succeed())
				Expect(consulRPCClient.ListKeysCallCount()).To(Equal(1))

				Expect(consulRPCClient.RemoveKeyCallCount()).To(Equal(2))

				key := consulRPCClient.RemoveKeyArgsForCall(0)
				Expect(key).To(Equal("key3"))

				key = consulRPCClient.RemoveKeyArgsForCall(1)
				Expect(key).To(Equal("key4"))

				Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "agent-client.set-keys.list-keys.request",
					},
					{
						Action: "agent-client.set-keys.list-keys.response",
						Data: []lager.Data{{
							"keys": []string{"key3", "key4"},
						}},
					},
					{
						Action: "agent-client.set-keys.remove-key.request",
						Data: []lager.Data{{
							"key": "key3",
						}},
					},
					{
						Action: "agent-client.set-keys.remove-key.response",
						Data: []lager.Data{{
							"key": "key3",
						}},
					},
					{
						Action: "agent-client.set-keys.remove-key.request",
						Data: []lager.Data{{
							"key": "key4",
						}},
					},
					{
						Action: "agent-client.set-keys.remove-key.response",
						Data: []lager.Data{{
							"key": "key4",
						}},
					},
					{
						Action: "agent-client.set-keys.install-key.request",
						Data: []lager.Data{{
							"key": encryptedKey1,
						}},
					},
					{
						Action: "agent-client.set-keys.install-key.response",
						Data: []lager.Data{{
							"key": encryptedKey1,
						}},
					},
					{
						Action: "agent-client.set-keys.install-key.request",
						Data: []lager.Data{{
							"key": encryptedKey2,
						}},
					},
					{
						Action: "agent-client.set-keys.install-key.response",
						Data: []lager.Data{{
							"key": encryptedKey2,
						}},
					},
					{
						Action: "agent-client.set-keys.use-key.request",
						Data: []lager.Data{{
							"key": encryptedKey1,
						}},
					},
					{
						Action: "agent-client.set-keys.use-key.response",
						Data: []lager.Data{{
							"key": encryptedKey1,
						}},
					},
					{
						Action: "agent-client.set-keys.success",
					},
				}))
			})
		})

		Context("failure cases", func() {
			Context("when provided with a nil slice", func() {
				It("returns a reasonably named error", func() {
					Expect(client.SetKeys(nil)).To(MatchError("must provide a non-nil slice of keys"))
					Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
						{
							Action: "agent-client.set-keys.nil-slice",
							Error:  errors.New("must provide a non-nil slice of keys"),
						},
					}))
				})
			})

			Context("when provided with an empty slice", func() {
				It("returns a reasonably named error", func() {
					Expect(client.SetKeys([]string{})).To(MatchError("must provide a non-empty slice of keys"))
					Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
						{
							Action: "agent-client.set-keys.empty-slice",
							Error:  errors.New("must provide a non-empty slice of keys"),
						},
					}))
				})
			})

			Context("when ListKeys returns an error", func() {
				It("returns the error", func() {
					consulRPCClient.ListKeysReturns([]string{}, errors.New("list keys error"))

					Expect(client.SetKeys([]string{"key1"})).To(MatchError("list keys error"))
					Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
						{
							Action: "agent-client.set-keys.list-keys.request",
						},
						{
							Action: "agent-client.set-keys.list-keys.request.failed",
							Error:  errors.New("list keys error"),
						},
					}))
				})
			})

			Context("when RemoveKeys returns an error", func() {
				It("returns the error", func() {
					consulRPCClient.ListKeysReturns([]string{"key2"}, nil)
					consulRPCClient.RemoveKeyReturns(errors.New("remove key error"))

					Expect(client.SetKeys([]string{"key1"})).To(MatchError("remove key error"))
					Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
						{
							Action: "agent-client.set-keys.list-keys.request",
						},
						{
							Action: "agent-client.set-keys.list-keys.response",
							Data: []lager.Data{{
								"keys": []string{"key2"},
							}},
						},
						{
							Action: "agent-client.set-keys.remove-key.request",
							Data: []lager.Data{{
								"key": "key2",
							}},
						},
						{
							Action: "agent-client.set-keys.remove-key.request.failed",
							Error:  errors.New("remove key error"),
							Data: []lager.Data{{
								"key": "key2",
							}},
						},
					}))
				})
			})

			Context("when InstallKey returns an error", func() {
				It("returns the error", func() {
					consulRPCClient.InstallKeyReturns(errors.New("install key error"))

					Expect(client.SetKeys([]string{"key1"})).To(MatchError("install key error"))
					Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
						{
							Action: "agent-client.set-keys.list-keys.request",
						},
						{
							Action: "agent-client.set-keys.list-keys.response",
							Data: []lager.Data{{
								"keys": []string{},
							}},
						},
						{
							Action: "agent-client.set-keys.install-key.request",
							Data: []lager.Data{{
								"key": encryptedKey1,
							}},
						},
						{
							Action: "agent-client.set-keys.install-key.request.failed",
							Error:  errors.New("install key error"),
							Data: []lager.Data{{
								"key": encryptedKey1,
							}},
						},
					}))
				})
			})

			Context("when UseKey returns an error", func() {
				It("returns the error", func() {
					consulRPCClient.UseKeyReturns(errors.New("use key error"))

					Expect(client.SetKeys([]string{"key1"})).To(MatchError("use key error"))
					Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
						{
							Action: "agent-client.set-keys.list-keys.request",
						},
						{
							Action: "agent-client.set-keys.list-keys.response",
							Data: []lager.Data{{
								"keys": []string{},
							}},
						},
						{
							Action: "agent-client.set-keys.install-key.request",
							Data: []lager.Data{{
								"key": encryptedKey1,
							}},
						},
						{
							Action: "agent-client.set-keys.install-key.response",
							Data: []lager.Data{{
								"key": encryptedKey1,
							}},
						},
						{
							Action: "agent-client.set-keys.use-key.request",
							Data: []lager.Data{{
								"key": encryptedKey1,
							}},
						},
						{
							Action: "agent-client.set-keys.use-key.request.failed",
							Error:  errors.New("use key error"),
							Data: []lager.Data{{
								"key": encryptedKey1,
							}},
						},
					}))
				})
			})
		})
	})

	Describe("Leave", func() {
		It("leaves the cluster", func() {
			Expect(client.Leave()).To(Succeed())
			Expect(consulRPCClient.LeaveCallCount()).To(Equal(1))
			Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
				{
					Action: "agent-client.leave.leave.request",
				},
				{
					Action: "agent-client.leave.leave.response",
				},
			}))
		})

		Context("when RPCClient.leave returns an error", func() {
			It("returns an error", func() {
				consulRPCClient.LeaveReturns(errors.New("leave error"))

				Expect(client.Leave()).To(MatchError("leave error"))
				Expect(consulRPCClient.LeaveCallCount()).To(Equal(1))
				Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "agent-client.leave.leave.request",
					},
					{
						Action: "agent-client.leave.leave.request.failed",
						Error:  errors.New("leave error"),
					},
				}))
			})
		})

		Context("when the RCPClient has never been set", func() {
			It("returns an error", func() {
				client.ConsulRPCClient = nil

				Expect(client.Leave()).To(MatchError("consul rpc client is nil"))
				Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "agent-client.leave.nil-rpc-client",
						Error:  errors.New("consul rpc client is nil"),
					},
				}))
			})
		})
	})
})

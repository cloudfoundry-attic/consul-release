package confab_test

import (
	"confab"
	"confab/fakes"
	"errors"

	"github.com/hashicorp/consul/api"
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
				Expect(client.VerifySynced()).To(MatchError("some error"))
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
	})
})

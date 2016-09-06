package chaperon_test

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"time"

	"code.cloudfoundry.org/lager"

	"github.com/cloudfoundry-incubator/consul-release/src/confab/agent"
	"github.com/cloudfoundry-incubator/consul-release/src/confab/chaperon"
	"github.com/cloudfoundry-incubator/consul-release/src/confab/config"
	"github.com/cloudfoundry-incubator/consul-release/src/confab/fakes"
	"github.com/cloudfoundry-incubator/consul-release/src/confab/utils"
	"github.com/hashicorp/consul/api"
	consulagent "github.com/hashicorp/consul/command/agent"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf-experimental/gomegamatchers"
)

var _ = Describe("Checker", func() {
	Describe("StartInBootstrap", func() {
		var (
			logger               *fakes.Logger
			controller           *fakes.Controller
			configWriter         *fakes.ConfigWriter
			agentRunner          *fakes.AgentRunner
			agentClient          *fakes.AgentClient
			clock                *fakes.Clock
			rpcClient            *consulagent.RPCClient
			randomUUIDGenerator  chaperon.RandomUUIDGenerator
			bootstrapInput       chaperon.BootstrapInput
			fakeAgentServer      *httptest.Server
			fakeAgentHandlerStub func(w http.ResponseWriter, r *http.Request)
			rpcEndpoint          string
		)

		BeforeEach(func() {
			logger = &fakes.Logger{}
			controller = &fakes.Controller{}
			configWriter = &fakes.ConfigWriter{}
			agentRunner = &fakes.AgentRunner{}
			agentClient = &fakes.AgentClient{}
			clock = &fakes.Clock{}

			randomUUIDGenerator = func(io.Reader) (string, error) {
				return "some-random-guid", nil
			}
			fakeAgentHandlerStub = func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/v1/agent/self" {
					w.WriteHeader(http.StatusOK)
					return
				}

				w.WriteHeader(http.StatusTeapot)
			}

			fakeAgentServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fakeAgentHandlerStub(w, r)
			}))

			rpcClient = &consulagent.RPCClient{}
			rpcClientConstructor := func(url string) (*consulagent.RPCClient, error) {
				rpcEndpoint = url
				return rpcClient, nil
			}

			agentClient.MembersCall.Returns.Members = []*api.AgentMember{
				{
					Tags: map[string]string{
						"some-key": "some-value",
					},
				},
			}

			config := config.Config{
				Consul: config.ConfigConsul{
					Agent: config.ConfigConsulAgent{
						NodeName: "consul-z1/0",
					},
				},
			}

			bootstrapInput = chaperon.BootstrapInput{
				AgentURL:           fakeAgentServer.URL,
				Logger:             logger,
				Controller:         controller,
				AgentRunner:        agentRunner,
				ConfigWriter:       configWriter,
				Config:             config,
				GenerateRandomUUID: randomUUIDGenerator,
				AgentClient:        agentClient,
				NewRPCClient:       rpcClientConstructor,
				Retrier:            utils.NewRetrier(clock, 10*time.Millisecond),
				Timeout:            utils.NewTimeout(time.After(10 * time.Millisecond)),
			}
		})

		Context("when there is no leader in the cluster", func() {
			It("returns true", func() {
				statusLeaderCallCount := 0
				fakeAgentHandlerStub = func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/v1/status/leader":
						statusLeaderCallCount++
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`""`))
						return
					}

					w.WriteHeader(http.StatusOK)
				}

				startInBootstrap, err := chaperon.StartInBootstrap(bootstrapInput)
				Expect(err).NotTo(HaveOccurred())
				Expect(startInBootstrap).To(BeTrue())

				Expect(configWriter.WriteCall.CallCount).To(Equal(1))
				Expect(configWriter.WriteCall.Receives.Config.Consul.Agent.Mode).To(Equal("client"))
				Expect(configWriter.WriteCall.Receives.Config.Consul.Agent.NodeName).To(Equal("client-some-random-guid"))

				Expect(agentRunner.RunCalls.CallCount).To(Equal(1))

				Expect(agentClient.SelfCall.CallCount).To(Equal(1))

				Expect(agentClient.JoinMembersCall.CallCount).To(Equal(1))

				Expect(agentClient.MembersCall.CallCount).To(Equal(1))
				Expect(agentClient.MembersCall.Receives.WAN).To(BeFalse())

				Expect(statusLeaderCallCount).To(Equal(1))

				Expect(controller.StopAgentCall.CallCount).To(Equal(1))
				Expect(controller.StopAgentCall.Receives.RPCClient).To(Equal(rpcClient))
				Expect(rpcEndpoint).To(Equal("localhost:8400"))

				Expect(logger.Messages()).To(ContainSequence([]fakes.LoggerMessage{
					{
						Action: "chaperon-checker.start-in-bootstrap.generate-random-uuid",
					},
					{
						Action: "chaperon-checker.start-in-bootstrap.config-writer.write",
					},
					{
						Action: "chaperon-checker.start-in-bootstrap.agent-runner.run",
					},
					{
						Action: "chaperon-checker.start-in-bootstrap.waiting-for-agent",
					},
					{
						Action: "chaperon-checker.start-in-bootstrap.agent-client.join-members",
					},
					{
						Action: "chaperon-checker.start-in-bootstrap.agent-client.members",
					},
					{
						Action: "chaperon-checker.start-in-bootstrap.http.get",
						Data: []lager.Data{
							{
								"route": fmt.Sprintf("%s%s", fakeAgentServer.URL, "/v1/status/leader"),
							},
						},
					},
					{
						Action: "chaperon-checker.start-in-bootstrap.json-decoder.decode",
					},
					{
						Action: "chaperon-checker.start-in-bootstrap.bootstrap-true",
					},
					{
						Action: "chaperon-checker.start-in-bootstrap.controller.stop-agent",
					},
				}))
			})
		})

		Context("when the agent does not start right away", func() {
			BeforeEach(func() {
				fakeAgentHandlerStub = func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/v1/status/leader":
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`""`))
						return
					}

					w.WriteHeader(http.StatusOK)
				}

				agentClient.SelfCall.Returns.Error = nil
			})

			It("succeeds when the agent starts within the check timeout", func() {
				agentClient.SelfCall.Returns.Errors = make([]error, 10)
				for i := 0; i < 9; i++ {
					agentClient.SelfCall.Returns.Errors[i] = errors.New("some error occurred")
				}

				_, err := chaperon.StartInBootstrap(bootstrapInput)
				Expect(err).NotTo(HaveOccurred())

				Expect(agentClient.SelfCall.CallCount).To(Equal(10))
				Expect(clock.SleepCall.CallCount).To(Equal(9))
				Expect(clock.SleepCall.Receives.Duration).To(Equal(10 * time.Millisecond))

				Expect(logger.Messages()).To(ContainElement(fakes.LoggerMessage{
					Action: "chaperon-checker.start-in-bootstrap.waiting-for-agent",
				}))
			})

			It("returns an error when the agent does not start", func() {
				agentClient.SelfCall.Returns.Error = errors.New("some error occurred")
				_, err := chaperon.StartInBootstrap(bootstrapInput)

				Expect(agentClient.SelfCall.CallCount).ToNot(Equal(0))
				Expect(err).To(MatchError(`timeout exceeded: "some error occurred"`))

				Expect(logger.Messages()).To(ContainElement(fakes.LoggerMessage{
					Action: "chaperon-checker.start-in-bootstrap.waiting-for-agent.failed",
					Error:  errors.New(`timeout exceeded: "some error occurred"`),
				}))
			})
		})

		Context("when there are no members to join", func() {
			It("returns true", func() {
				agentClient.JoinMembersCall.Returns.Error = agent.NoMembersToJoinError
				startInBootstrap, err := chaperon.StartInBootstrap(bootstrapInput)
				Expect(err).NotTo(HaveOccurred())
				Expect(startInBootstrap).To(BeTrue())
				Expect(logger.Messages()).To(ContainElement(fakes.LoggerMessage{
					Action: "chaperon-checker.start-in-bootstrap.agent-client.join-members.no-members-to-join",
				}))
			})
		})

		Context("when there is a bootstrapped node in the cluster", func() {
			It("returns false", func() {
				agentClient.MembersCall.Returns.Members = []*api.AgentMember{
					{
						Name: "some-member",
						Tags: map[string]string{
							"bootstrap": "1",
						},
					},
				}

				startInBootstrap, err := chaperon.StartInBootstrap(bootstrapInput)
				Expect(err).NotTo(HaveOccurred())
				Expect(startInBootstrap).To(BeFalse())

				Expect(controller.StopAgentCall.CallCount).To(Equal(1))
				Expect(controller.StopAgentCall.Receives.RPCClient).To(Equal(rpcClient))
				Expect(logger.Messages()).To(ContainElement(fakes.LoggerMessage{
					Action: "chaperon-checker.start-in-bootstrap.bootstrap-node-exists",
					Data: []lager.Data{
						{
							"bootstrap-node": "some-member",
						},
					},
				}))
			})
		})

		Context("when there is a current leader", func() {
			It("returns false", func() {
				statusLeaderCallCount := 0
				fakeAgentHandlerStub = func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/v1/status/leader":
						statusLeaderCallCount++
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`"some-leader"`))
						return
					}

					w.WriteHeader(http.StatusOK)
				}

				startInBootstrap, err := chaperon.StartInBootstrap(bootstrapInput)
				Expect(err).NotTo(HaveOccurred())
				Expect(startInBootstrap).To(BeFalse())
				Expect(logger.Messages()).To(ContainElement(fakes.LoggerMessage{
					Action: "chaperon-checker.start-in-bootstrap.leader-exists",
					Data: []lager.Data{
						{
							"leader": "some-leader",
						},
					},
				}))
			})
		})

		Context("failure cases", func() {
			It("returns an error when the rpc client cannot be created", func() {
				bootstrapInput.NewRPCClient = func(url string) (*consulagent.RPCClient, error) {
					return nil, errors.New("error creating rpc client")
				}

				_, err := chaperon.StartInBootstrap(bootstrapInput)
				Expect(err).To(MatchError("error creating rpc client"))
				Expect(logger.Messages()).To(ContainElement(fakes.LoggerMessage{
					Action: "chaperon-checker.start-in-bootstrap.creating-rpc-client.failed",
					Error:  errors.New("error creating rpc client"),
				}))
			})

			Context("when the config writer cannot write the config", func() {
				It("returns an error", func() {
					configWriter.WriteCall.Returns.Error = errors.New("failed to write config")
					_, err := chaperon.StartInBootstrap(bootstrapInput)
					Expect(err).To(MatchError("failed to write config"))
					Expect(logger.Messages()).To(ContainElement(fakes.LoggerMessage{
						Action: "chaperon-checker.start-in-bootstrap.config-writer.write.failed",
						Error:  fmt.Errorf("failed to write config"),
					}))
				})
			})

			Context("when the client agent is unable to run", func() {
				It("returns an error", func() {
					agentRunner.RunCalls.Returns.Errors = []error{errors.New("failed to run the client")}
					_, err := chaperon.StartInBootstrap(bootstrapInput)
					Expect(err).To(MatchError("failed to run the client"))
					Expect(logger.Messages()).To(ContainElement(fakes.LoggerMessage{
						Action: "chaperon-checker.start-in-bootstrap.agent-runner.run.failed",
						Error:  fmt.Errorf("failed to run the client"),
					}))
				})
			})

			Context("when the client agent join members fails", func() {
				It("returns an error", func() {
					agentClient.JoinMembersCall.Returns.Error = errors.New("failed to join members")
					_, err := chaperon.StartInBootstrap(bootstrapInput)
					Expect(err).To(MatchError("failed to join members"))
					Expect(logger.Messages()).To(ContainElement(fakes.LoggerMessage{
						Action: "chaperon-checker.start-in-bootstrap.agent-client.join-members.failed",
						Error:  fmt.Errorf("failed to join members"),
					}))
				})
			})

			Context("when the call to list members fails", func() {
				It("returns an error", func() {
					agentClient.MembersCall.Returns.Error = errors.New("failed to list members")
					_, err := chaperon.StartInBootstrap(bootstrapInput)
					Expect(err).To(MatchError("failed to list members"))
					Expect(logger.Messages()).To(ContainElement(fakes.LoggerMessage{
						Action: "chaperon-checker.start-in-bootstrap.agent-client.members.failed",
						Error:  fmt.Errorf("failed to list members"),
					}))
				})
			})

			Context("when the request to the status leader endpoint fails", func() {
				It("returns an error", func() {
					fakeAgentHandlerStub = func(w http.ResponseWriter, r *http.Request) {
						switch r.URL.Path {
						case "/v1/status/leader":
							w.WriteHeader(0)
							return
						}

						w.WriteHeader(http.StatusOK)
					}
					_, err := chaperon.StartInBootstrap(bootstrapInput)
					Expect(err).To(MatchError(ContainSubstring("malformed HTTP status code")))
					Expect(logger.Messages()).To(ContainSequence([]fakes.LoggerMessage{
						{
							Action: "chaperon-checker.start-in-bootstrap.http.get",
							Data: []lager.Data{
								{
									"route": fmt.Sprintf("%s%s", fakeAgentServer.URL, "/v1/status/leader"),
								},
							},
						},
						{
							Action: "chaperon-checker.start-in-bootstrap.http.get.failed",
							Error:  err,
						},
					}))
				})
			})

			Context("when the status leader endpoint responds with a non-200 status code", func() {
				AfterEach(func() {
					chaperon.ResetReadAll()
				})
				Context("when agent errors for no known consul servers", func() {
					It("returns true", func() {
						fakeAgentHandlerStub = func(w http.ResponseWriter, r *http.Request) {
							switch r.URL.Path {
							case "/v1/status/leader":
								w.WriteHeader(500)
								w.Write([]byte("No known Consul servers"))
								return
							}

							w.WriteHeader(http.StatusOK)
						}
						bootstrapMode, err := chaperon.StartInBootstrap(bootstrapInput)
						Expect(err).NotTo(HaveOccurred())
						Expect(bootstrapMode).To(BeTrue())
					})
				})

				Context("leader check fails for any other reason", func() {
					It("returns an error", func() {
						fakeAgentHandlerStub = func(w http.ResponseWriter, r *http.Request) {
							switch r.URL.Path {
							case "/v1/status/leader":
								w.WriteHeader(500)
								w.Write([]byte("no server available"))
								return
							}

							w.WriteHeader(http.StatusOK)
						}
						_, err := chaperon.StartInBootstrap(bootstrapInput)
						Expect(err).To(MatchError(ContainSubstring(`Leader check returned 500 status with response "no server available"`)))
						Expect(logger.Messages()).To(ContainSequence([]fakes.LoggerMessage{
							{
								Action: "chaperon-checker.start-in-bootstrap.http.get",
								Data: []lager.Data{
									{
										"route": fmt.Sprintf("%s%s", fakeAgentServer.URL, "/v1/status/leader"),
									},
								},
							},
							{
								Action: "chaperon-checker.start-in-bootstrap.http.get.invalid-response",
								Error:  err,
							},
						}))
					})

					It("returns an error when a body cannot be read", func() {
						chaperon.SetReadAll(func(_ io.Reader) ([]byte, error) {
							return nil, errors.New("invalid body")
						})

						fakeAgentHandlerStub = func(w http.ResponseWriter, r *http.Request) {
							switch r.URL.Path {
							case "/v1/status/leader":
								w.WriteHeader(424)
								return
							}

							w.WriteHeader(http.StatusOK)
						}
						_, err := chaperon.StartInBootstrap(bootstrapInput)

						Expect(err).To(MatchError(ContainSubstring(`Leader check returned 424 status: body could not be read "invalid body"`)))

						Expect(logger.Messages()).To(ContainSequence([]fakes.LoggerMessage{
							{
								Action: "chaperon-checker.start-in-bootstrap.http.get",
								Data: []lager.Data{
									{
										"route": fmt.Sprintf("%s%s", fakeAgentServer.URL, "/v1/status/leader"),
									},
								},
							},
							{
								Action: "chaperon-checker.start-in-bootstrap.http.get.invalid-response",
								Error:  err,
							},
						}))

					})
				})
			})

			Context("when the json decoder fails", func() {
				It("returns an error", func() {
					fakeAgentHandlerStub = func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`%%%%`))
					}
					_, err := chaperon.StartInBootstrap(bootstrapInput)
					Expect(err).To(MatchError(ContainSubstring("invalid character")))
					Expect(logger.Messages()).To(ContainElement(fakes.LoggerMessage{
						Action: "chaperon-checker.start-in-bootstrap.json-decoder.decode.failed",
						Error:  err,
					}))
				})
			})

			Context("when the random uuid generator fails", func() {
				It("returns an error", func() {
					bootstrapInput.GenerateRandomUUID = func(io.Reader) (string, error) {
						return "", errors.New("uuid generator failed")
					}
					_, err := chaperon.StartInBootstrap(bootstrapInput)
					Expect(err).To(MatchError("uuid generator failed"))
					Expect(logger.Messages()).To(ContainElement(fakes.LoggerMessage{
						Action: "chaperon-checker.start-in-bootstrap.generate-random-uuid.failed",
						Error:  fmt.Errorf("uuid generator failed"),
					}))
				})
			})
		})
	})
})

func openPort() (string, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return "", err
	}

	defer l.Close()
	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return "", err
	}

	return port, nil
}

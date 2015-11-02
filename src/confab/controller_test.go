package confab_test

import (
	"confab"
	"confab/fakes"
	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("controller", func() {
	var (
		agentRunner *fakes.AgentRunner
		agentClient *fakes.AgentClient
		controller  confab.Controller
	)

	Describe("StopAgent", func() {
		BeforeEach(func() {
			agentClient = &fakes.AgentClient{}
			agentClient.VerifyJoinedCalls.Returns.Errors = []error{nil}
			agentClient.VerifySyncedCalls.Returns.Errors = []error{nil}

			agentRunner = &fakes.AgentRunner{}
			agentRunner.RunCalls.Returns.Errors = []error{nil}

			controller = confab.Controller{
				AgentClient: agentClient,
				AgentRunner: agentRunner,
			}
		})

		It("tells client to leave the cluster", func() {
			Expect(controller.StopAgent()).To(Succeed())

			Expect(agentClient.LeaveCall.CallCount).To(Equal(1))
		})

		It("tells the runner to stop the agent", func() {
			Expect(controller.StopAgent()).To(Succeed())

			Expect(agentRunner.StopCall.CallCount).To(Equal(1))
		})

		Context("The client returns an error", func() {
			BeforeEach(func() {
				agentClient.LeaveCall.Returns.Error = errors.New("leave error")
			})

			It("immediately returns an error", func() {
				Expect(controller.StopAgent()).To(MatchError("leave error"))

				Expect(agentClient.LeaveCall.CallCount).To(Equal(1))
			})
		})

		Context("The runner returns an error", func() {
			BeforeEach(func() {
				agentRunner.StopCall.Returns.Error = errors.New("stop error")
			})

			It("immediately returns an error", func() {
				Expect(controller.StopAgent()).To(MatchError("stop error"))

				Expect(agentRunner.StopCall.CallCount).To(Equal(1))
			})
		})
	})

	Describe("BootServer", func() {
		BeforeEach(func() {
			agentRunner = &fakes.AgentRunner{}
			agentRunner.RunCalls.Returns.Errors = []error{nil}

			agentClient = &fakes.AgentClient{}
			agentClient.VerifyJoinedCalls.Returns.Errors = []error{nil}
			agentClient.VerifySyncedCalls.Returns.Errors = []error{nil}

			controller = confab.Controller{
				AgentRunner:    agentRunner,
				AgentClient:    agentClient,
				MaxRetries:     10,
				SyncRetryDelay: 10 * time.Millisecond,
				EncryptKeys:    []string{"key 1", "key 2", "key 3"},
			}
		})

		It("launches the consul agent", func() {
			Expect(controller.BootServer()).To(Succeed())
			Expect(agentRunner.RunCalls.CallCount).To(Equal(1))
		})

		It("checks that the agent has joined a cluster", func() {
			Expect(controller.BootServer()).To(Succeed())
			Expect(agentClient.VerifyJoinedCalls.CallCount).To(Equal(1))
		})

		Context("when it is not the last node in the cluster", func() {
			It("does not check that it is synced", func() {
				Expect(controller.BootServer()).To(Succeed())
				Expect(agentClient.VerifySyncedCalls.CallCount).To(Equal(0))
			})
		})

		It("sets the encryption keys used by the agent", func() {
			Expect(controller.BootServer()).To(Succeed())
			Expect(agentClient.SetKeysCall.Receives.Keys).To(Equal([]string{
				"key 1", "key 2", "key 3"}))
		})

		Context("when setting keys errors", func() {
			It("returns the error", func() {
				agentClient.SetKeysCall.Returns.Error = errors.New("oh noes")
				Expect(controller.BootServer()).To(MatchError("oh noes"))
				Expect(agentClient.SetKeysCall.Receives.Keys).To(Equal([]string{
					"key 1", "key 2", "key 3"}))
			})
		})

		Context("when it is the last node in the cluster", func() {
			BeforeEach(func() {
				agentClient.IsLastNodeCall.Returns.IsLastNode = true
			})
			It("checks that it is synced", func() {
				Expect(controller.BootServer()).To(Succeed())
				Expect(agentClient.VerifySyncedCalls.CallCount).To(Equal(1))
			})

			Context("verifying sync fails at first but later succeeds", func() {
				It("retries until it verifies sync successfully", func() {
					agentClient.VerifySyncedCalls.Returns.Errors = make([]error, 10)
					for i := 0; i < 9; i++ {
						agentClient.VerifySyncedCalls.Returns.Errors[i] = errors.New("some error")
					}

					Expect(controller.BootServer()).To(Succeed())
					Expect(agentClient.VerifySyncedCalls.CallCount).To(Equal(10))
				})
			})

			Context("verifying synced never succeeds within MaxRetries", func() {
				It("immediately returns an error", func() {
					agentClient.VerifySyncedCalls.Returns.Errors = make([]error, 10)
					for i := 0; i < 9; i++ {
						agentClient.VerifySyncedCalls.Returns.Errors[i] = errors.New("some error")
					}
					agentClient.VerifySyncedCalls.Returns.Errors[9] = errors.New("the final error")

					Expect(controller.BootServer()).To(MatchError("the final error"))
					Expect(agentClient.VerifySyncedCalls.CallCount).To(Equal(10))
					Expect(agentClient.SetKeysCall.Receives.Keys).To(BeNil())
				})
			})

			Context("error while checking if it is the last node", func() {
				It("immediately returns the error", func() {
					agentClient.IsLastNodeCall.Returns.Error = errors.New("some error")
					Expect(controller.BootServer()).To(MatchError("some error"))
					Expect(agentClient.VerifySyncedCalls.CallCount).To(Equal(0))
					Expect(agentClient.SetKeysCall.Receives.Keys).To(BeNil())
				})
			})
		})

		Context("when starting the agent fails", func() {
			It("immediately returns an error", func() {
				agentRunner.RunCalls.Returns.Errors = []error{errors.New("some error")}

				Expect(controller.BootServer()).To(MatchError("some error"))
				Expect(agentRunner.RunCalls.CallCount).To(Equal(1))
				Expect(agentClient.VerifyJoinedCalls.CallCount).To(Equal(0))
			})
		})

		Context("joining fails at first but later succeeds", func() {
			It("retries until it joins", func() {
				agentClient.VerifyJoinedCalls.Returns.Errors = make([]error, 10)
				for i := 0; i < 9; i++ {
					agentClient.VerifyJoinedCalls.Returns.Errors[i] = errors.New("some error")
				}

				Expect(controller.BootServer()).To(Succeed())
				Expect(agentClient.VerifyJoinedCalls.CallCount).To(Equal(10))
			})
		})

		Context("joining never succeeds within MaxRetries", func() {
			It("immediately returns an error", func() {
				agentClient.VerifyJoinedCalls.Returns.Errors = make([]error, 10)
				for i := 0; i < 9; i++ {
					agentClient.VerifyJoinedCalls.Returns.Errors[i] = errors.New("some error")
				}
				agentClient.VerifyJoinedCalls.Returns.Errors[9] = errors.New("the final error")

				Expect(controller.BootServer()).To(MatchError("the final error"))
				Expect(agentClient.VerifyJoinedCalls.CallCount).To(Equal(10))

				Expect(agentClient.VerifySyncedCalls.CallCount).To(Equal(0))
			})
		})
	})
	Describe("BootClient", func() {
		BeforeEach(func() {
			agentRunner = &fakes.AgentRunner{}
			agentRunner.RunCalls.Returns.Errors = []error{nil}

			agentClient = &fakes.AgentClient{}
			agentClient.VerifyJoinedCalls.Returns.Errors = []error{nil}
			agentClient.VerifySyncedCalls.Returns.Errors = []error{nil}

			controller = confab.Controller{
				AgentRunner:    agentRunner,
				AgentClient:    agentClient,
				MaxRetries:     10,
				SyncRetryDelay: 10 * time.Millisecond,
				EncryptKeys:    []string{"key 1", "key 2", "key 3"},
			}
		})

		It("launches the consul agent", func() {
			Expect(controller.BootClient()).To(Succeed())
			Expect(agentRunner.RunCalls.CallCount).To(Equal(1))
		})

		It("checks that the agent has joined a cluster", func() {
			Expect(controller.BootClient()).To(Succeed())
			Expect(agentClient.VerifyJoinedCalls.CallCount).To(Equal(1))
		})

		Context("when starting the agent fails", func() {
			It("immediately returns an error", func() {
				agentRunner.RunCalls.Returns.Errors = []error{errors.New("some error")}

				Expect(controller.BootClient()).To(MatchError("some error"))
				Expect(agentRunner.RunCalls.CallCount).To(Equal(1))
				Expect(agentClient.VerifyJoinedCalls.CallCount).To(Equal(0))
			})
		})

		Context("joining fails at first but later succeeds", func() {
			It("retries until it joins", func() {
				agentClient.VerifyJoinedCalls.Returns.Errors = make([]error, 10)
				for i := 0; i < 9; i++ {
					agentClient.VerifyJoinedCalls.Returns.Errors[i] = errors.New("some error")
				}

				Expect(controller.BootClient()).To(Succeed())
				Expect(agentClient.VerifyJoinedCalls.CallCount).To(Equal(10))
			})
		})

		Context("joining never succeeds within MaxRetries", func() {
			It("immediately returns an error", func() {
				agentClient.VerifyJoinedCalls.Returns.Errors = make([]error, 10)
				for i := 0; i < 9; i++ {
					agentClient.VerifyJoinedCalls.Returns.Errors[i] = errors.New("some error")
				}
				agentClient.VerifyJoinedCalls.Returns.Errors[9] = errors.New("the final error")

				Expect(controller.BootClient()).To(MatchError("the final error"))
				Expect(agentClient.VerifyJoinedCalls.CallCount).To(Equal(10))

				Expect(agentClient.VerifySyncedCalls.CallCount).To(Equal(0))
			})
		})
	})
})

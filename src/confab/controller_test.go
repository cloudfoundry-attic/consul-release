package confab_test

import (
	"bytes"
	"confab"
	"confab/fakes"
	"errors"
	"log"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("controller", func() {
	var (
		clock       *fakes.Clock
		agentRunner *fakes.AgentRunner
		agentClient *fakes.AgentClient
		controller  confab.Controller
		logBuffer   *bytes.Buffer
	)

	BeforeEach(func() {
		clock = &fakes.Clock{}

		agentClient = &fakes.AgentClient{}
		agentClient.VerifyJoinedCalls.Returns.Errors = []error{nil}
		agentClient.VerifySyncedCalls.Returns.Errors = []error{nil}

		agentRunner = &fakes.AgentRunner{}
		agentRunner.RunCalls.Returns.Errors = []error{nil}

		logBuffer = &bytes.Buffer{}

		controller = confab.Controller{
			AgentClient:    agentClient,
			AgentRunner:    agentRunner,
			MaxRetries:     10,
			SyncRetryDelay: 10 * time.Millisecond,
			SyncRetryClock: clock,
			EncryptKeys:    []string{"key 1", "key 2", "key 3"},
			Logger:         log.New(logBuffer, "[Test]", 0),
		}
	})

	Describe("BootAgent", func() {
		It("launches the consul agent", func() {
			Expect(controller.BootAgent()).To(Succeed())
			Expect(agentRunner.RunCalls.CallCount).To(Equal(1))
		})

		It("checks that the agent has joined a cluster", func() {
			Expect(controller.BootAgent()).To(Succeed())
			Expect(agentClient.VerifyJoinedCalls.CallCount).To(Equal(1))
		})

		Context("when starting the agent fails", func() {
			It("immediately returns an error", func() {
				agentRunner.RunCalls.Returns.Errors = []error{errors.New("some error")}

				Expect(controller.BootAgent()).To(MatchError("some error"))
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

				Expect(controller.BootAgent()).To(Succeed())
				Expect(agentClient.VerifyJoinedCalls.CallCount).To(Equal(10))
				Expect(clock.SleepCall.CallCount).To(Equal(9))
				Expect(clock.SleepCall.Receives.Duration).To(Equal(10 * time.Millisecond))
			})
		})

		Context("joining never succeeds within MaxRetries", func() {
			It("immediately returns an error", func() {
				agentClient.VerifyJoinedCalls.Returns.Errors = make([]error, 10)
				for i := 0; i < 9; i++ {
					agentClient.VerifyJoinedCalls.Returns.Errors[i] = errors.New("some error")
				}
				agentClient.VerifyJoinedCalls.Returns.Errors[9] = errors.New("the final error")

				Expect(controller.BootAgent()).To(MatchError("the final error"))
				Expect(agentClient.VerifyJoinedCalls.CallCount).To(Equal(10))

				Expect(agentClient.VerifySyncedCalls.CallCount).To(Equal(0))
			})
		})
	})

	Describe("StopAgent", func() {
		It("tells client to leave the cluster", func() {
			controller.StopAgent()
			Expect(agentClient.LeaveCall.CallCount).To(Equal(1))
		})
		It("waits for the agent to stop", func() {
			controller.StopAgent()
			Expect(agentRunner.WaitCall.CallCount).To(Equal(1))
		})

		Context("when the agent client Leave() returns an error", func() {
			BeforeEach(func() {
				agentClient.LeaveCall.Returns.Error = errors.New("leave error")
			})

			It("tells the runner to stop the agent", func() {
				controller.StopAgent()
				Expect(agentRunner.StopCall.CallCount).To(Equal(1))
			})

			It("logs the error", func() {
				controller.StopAgent()
				Expect(logBuffer.String()).To(ContainSubstring("leave error"))
			})

			Context("when agent runner Stop() returns an error", func() {
				BeforeEach(func() {
					agentRunner.StopCall.Returns.Error = errors.New("stop error")
				})

				It("logs the error", func() {
					controller.StopAgent()
					Expect(logBuffer.String()).To(ContainSubstring("stop error"))
				})
			})
		})

		Context("when agent runner Wait() returns an error", func() {
			BeforeEach(func() {
				agentRunner.WaitCall.Returns.Error = errors.New("wait error")
			})

			It("logs the error", func() {
				controller.StopAgent()
				Expect(logBuffer.String()).To(ContainSubstring("wait error"))
			})
		})
	})

	Describe("ConfigureServer", func() {
		It("does not launch the consul agent", func() {
			Expect(controller.ConfigureServer()).To(Succeed())
			Expect(agentRunner.RunCalls.CallCount).To(Equal(0))
		})

		It("does not check that the agent has joined a cluster", func() {
			Expect(controller.ConfigureServer()).To(Succeed())
			Expect(agentClient.VerifyJoinedCalls.CallCount).To(Equal(0))
		})

		Context("when it is not the last node in the cluster", func() {
			It("does not check that it is synced", func() {
				Expect(controller.ConfigureServer()).To(Succeed())
				Expect(agentClient.VerifySyncedCalls.CallCount).To(Equal(0))
			})
		})

		Context("setting keys", func() {
			It("sets the encryption keys used by the agent", func() {
				Expect(controller.ConfigureServer()).To(Succeed())
				Expect(agentClient.SetKeysCall.Receives.Keys).To(Equal([]string{
					"key 1",
					"key 2",
					"key 3",
				}))
			})

			Context("when setting keys errors", func() {
				It("returns the error", func() {
					agentClient.SetKeysCall.Returns.Error = errors.New("oh noes")
					Expect(controller.ConfigureServer()).To(MatchError("oh noes"))
					Expect(agentClient.SetKeysCall.Receives.Keys).To(Equal([]string{
						"key 1",
						"key 2",
						"key 3",
					}))
				})
			})

			Context("when the server does not have ssl enabled", func() {
				BeforeEach(func() {
					controller.SSLDisabled = true
				})

				It("does not set keys", func() {
					Expect(controller.ConfigureServer()).To(Succeed())
					Expect(agentClient.SetKeysCall.Receives.Keys).To(BeNil())
				})
			})

			Context("when ssl is enabled but no keys are provided", func() {
				BeforeEach(func() {
					controller.EncryptKeys = []string{}
				})

				It("returns an error", func() {
					Expect(controller.ConfigureServer()).To(MatchError("encrypt keys cannot be empty if ssl is enabled"))
					Expect(agentClient.SetKeysCall.Receives.Keys).To(BeNil())
				})
			})
		})

		Context("when it is the last node in the cluster", func() {
			BeforeEach(func() {
				agentClient.IsLastNodeCall.Returns.IsLastNode = true
			})

			It("checks that it is synced", func() {
				Expect(controller.ConfigureServer()).To(Succeed())
				Expect(agentClient.VerifySyncedCalls.CallCount).To(Equal(1))
			})

			Context("verifying sync fails at first but later succeeds", func() {
				It("retries until it verifies sync successfully", func() {
					agentClient.VerifySyncedCalls.Returns.Errors = make([]error, 10)
					for i := 0; i < 9; i++ {
						agentClient.VerifySyncedCalls.Returns.Errors[i] = errors.New("some error")
					}

					Expect(controller.ConfigureServer()).To(Succeed())
					Expect(agentClient.VerifySyncedCalls.CallCount).To(Equal(10))
					Expect(clock.SleepCall.CallCount).To(Equal(9))
					Expect(clock.SleepCall.Receives.Duration).To(Equal(10 * time.Millisecond))
				})
			})

			Context("verifying synced never succeeds within MaxRetries", func() {
				It("immediately returns an error", func() {
					agentClient.VerifySyncedCalls.Returns.Errors = make([]error, 10)
					for i := 0; i < 9; i++ {
						agentClient.VerifySyncedCalls.Returns.Errors[i] = errors.New("some error")
					}
					agentClient.VerifySyncedCalls.Returns.Errors[9] = errors.New("the final error")

					Expect(controller.ConfigureServer()).To(MatchError("the final error"))
					Expect(agentClient.VerifySyncedCalls.CallCount).To(Equal(10))
					Expect(agentClient.SetKeysCall.Receives.Keys).To(BeNil())
				})
			})

			Context("error while checking if it is the last node", func() {
				It("immediately returns the error", func() {
					agentClient.IsLastNodeCall.Returns.Error = errors.New("some error")
					Expect(controller.ConfigureServer()).To(MatchError("some error"))
					Expect(agentClient.VerifySyncedCalls.CallCount).To(Equal(0))
					Expect(agentClient.SetKeysCall.Receives.Keys).To(BeNil())
				})
			})
		})
	})
})

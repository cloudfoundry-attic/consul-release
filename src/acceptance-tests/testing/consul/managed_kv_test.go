package consul_test

import (
	"acceptance-tests/testing/consul"
	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ManagedKV", func() {
	var (
		managedKV consul.ManagedKV
		agent     *fakeAgent
		kv        *fakeKV
		catalog   *fakeCatalog
	)

	BeforeEach(func() {
		agent = &fakeAgent{}
		kv = &fakeKV{}
		catalog = &fakeCatalog{}
		catalog.NodeCall.Returns.NodesFunc = func(c int) []consul.Node { return []consul.Node{{}, {}} }
		catalog.NodeCall.Returns.ErrorFunc = func(c int) error { return nil }

		managedKV = consul.NewManagedKV(consul.ManagedKVConfig{
			Agent:                        agent,
			KV:                           kv,
			Catalog:                      catalog,
			VerifyJoinedMaxTries:         10,
			VerifyJoinedIntervalDuration: time.Nanosecond,
		})
	})

	Describe("Set", func() {
		It("starts the agent, sets the key, and stops the agent", func() {
			err := managedKV.Set("some-key", "some-value")
			Expect(err).NotTo(HaveOccurred())

			Expect(agent.StartCall.WasCalled).To(BeTrue())
			Expect(kv.SetCall.Receives.Key).To(Equal("some-key"))
			Expect(kv.SetCall.Receives.Value).To(Equal("some-value"))
			Expect(agent.StopCall.WasCalled).To(BeTrue())
			Expect(catalog.NodeCall.CallCount).To(Equal(1))
		})

		Context("when the consul agent doesn't immediately come up", func() {
			It("checks the catalog several times", func() {
				catalogError := errors.New("connection refused")
				catalog.NodeCall.Returns.NodesFunc = func(c int) []consul.Node {
					if c < 2 {
						return []consul.Node{}
					}
					return []consul.Node{{}, {}}
				}
				catalog.NodeCall.Returns.ErrorFunc = func(c int) error {
					if c < 2 {
						return catalogError
					}
					return nil
				}

				err := managedKV.Set("some-key", "some-value")
				Expect(err).NotTo(HaveOccurred())

				Expect(agent.StartCall.WasCalled).To(BeTrue())
				Expect(kv.SetCall.Receives.Key).To(Equal("some-key"))
				Expect(kv.SetCall.Receives.Value).To(Equal("some-value"))
				Expect(agent.StopCall.WasCalled).To(BeTrue())
				Expect(catalog.NodeCall.CallCount).To(Equal(3))
			})

			Context("when the consul agent never comes up", func() {
				It("bails after trying some max number of times", func() {
					catalogError := errors.New("connection refused")
					catalog.NodeCall.Returns.NodesFunc = func(c int) []consul.Node { return []consul.Node{} }
					catalog.NodeCall.Returns.ErrorFunc = func(c int) error { return catalogError }

					err := managedKV.Set("some-key", "some-value")
					Expect(err).To(MatchError("connection refused"))
					Expect(catalog.NodeCall.CallCount).To(Equal(10))
				})
			})
		})

		Context("when the agent fails to start", func() {
			It("returns the error", func() {
				agent.StartCall.Returns.Error = errors.New("agent start error")

				err := managedKV.Set("some-key", "some-value")
				Expect(err).To(MatchError("agent start error"))

				Expect(agent.StartCall.WasCalled).To(BeTrue())
				Expect(kv.SetCall.Receives.Key).To(BeEmpty())
				Expect(kv.SetCall.Receives.Value).To(BeEmpty())
				Expect(agent.StopCall.WasCalled).To(BeFalse())
				Expect(catalog.NodeCall.CallCount).To(Equal(0))
			})
		})

		Context("when the key fails to set", func() {
			It("returns the error", func() {
				kv.SetCall.Returns.Error = errors.New("kv set error")

				err := managedKV.Set("some-key", "some-value")
				Expect(err).To(MatchError("kv set error"))

				Expect(agent.StartCall.WasCalled).To(BeTrue())
				Expect(kv.SetCall.Receives.Key).To(Equal("some-key"))
				Expect(kv.SetCall.Receives.Value).To(Equal("some-value"))
				Expect(agent.StopCall.WasCalled).To(BeTrue())
			})

			Context("and the agent fails to stop", func() {
				It("returns agent stop error", func() {
					kv.SetCall.Returns.Error = errors.New("kv set error")
					agent.StopCall.Returns.Error = errors.New("agent stop error")

					err := managedKV.Set("some-key", "some-value")
					Expect(err).To(MatchError("agent stop error"))

					Expect(agent.StartCall.WasCalled).To(BeTrue())
					Expect(kv.SetCall.Receives.Key).To(Equal("some-key"))
					Expect(kv.SetCall.Receives.Value).To(Equal("some-value"))
					Expect(agent.StopCall.WasCalled).To(BeTrue())
				})
			})
		})

		Context("when the agent fails to stop", func() {
			It("returns the error", func() {
				agent.StopCall.Returns.Error = errors.New("agent stop error")

				err := managedKV.Set("some-key", "some-value")
				Expect(err).To(MatchError("agent stop error"))

				Expect(agent.StartCall.WasCalled).To(BeTrue())
				Expect(kv.SetCall.Receives.Key).To(Equal("some-key"))
				Expect(kv.SetCall.Receives.Value).To(Equal("some-value"))
				Expect(agent.StopCall.WasCalled).To(BeTrue())
			})
		})
	})

	Describe("Get", func() {
		It("starts the agent, gets the key, and stops the agent", func() {
			kv.GetCall.Returns.Value = "some-value"

			value, err := managedKV.Get("some-key")
			Expect(err).NotTo(HaveOccurred())
			Expect(value).To(Equal("some-value"))

			Expect(agent.StartCall.WasCalled).To(BeTrue())
			Expect(kv.GetCall.Receives.Key).To(Equal("some-key"))
			Expect(agent.StopCall.WasCalled).To(BeTrue())
			Expect(catalog.NodeCall.CallCount).To(Equal(1))
		})

		Context("when the consul agent doesn't immediately come up", func() {
			It("checks the catalog several times", func() {
				catalogError := errors.New("connection refused")
				catalog.NodeCall.Returns.NodesFunc = func(c int) []consul.Node {
					if c < 2 {
						return []consul.Node{}
					}
					return []consul.Node{{}, {}}
				}
				catalog.NodeCall.Returns.ErrorFunc = func(c int) error {
					if c < 2 {
						return catalogError
					}
					return nil
				}

				kv.GetCall.Returns.Value = "some-value"

				value, err := managedKV.Get("some-key")
				Expect(err).NotTo(HaveOccurred())
				Expect(value).To(Equal("some-value"))

				Expect(agent.StartCall.WasCalled).To(BeTrue())
				Expect(kv.GetCall.Receives.Key).To(Equal("some-key"))
				Expect(agent.StopCall.WasCalled).To(BeTrue())
				Expect(catalog.NodeCall.CallCount).To(Equal(3))
			})

			Context("when the consul agent never comes up", func() {
				It("bails after trying 10 times", func() {
					catalogError := errors.New("connection refused")
					catalog.NodeCall.Returns.NodesFunc = func(c int) []consul.Node { return []consul.Node{} }
					catalog.NodeCall.Returns.ErrorFunc = func(c int) error { return catalogError }

					_, err := managedKV.Get("some-key")
					Expect(err).To(MatchError("connection refused"))
					Expect(catalog.NodeCall.CallCount).To(Equal(10))
				})
			})
		})

		Context("when the agent fails to start", func() {
			It("returns the error", func() {
				agent.StartCall.Returns.Error = errors.New("agent start error")

				_, err := managedKV.Get("some-key")
				Expect(err).To(MatchError("agent start error"))

				Expect(agent.StartCall.WasCalled).To(BeTrue())
				Expect(kv.GetCall.Receives.Key).To(BeEmpty())
				Expect(agent.StopCall.WasCalled).To(BeFalse())
			})
		})

		Context("when the get fails", func() {
			It("returns the error", func() {
				kv.GetCall.Returns.Error = errors.New("kv get error")

				_, err := managedKV.Get("some-key")
				Expect(err).To(MatchError("kv get error"))

				Expect(agent.StartCall.WasCalled).To(BeTrue())
				Expect(kv.GetCall.Receives.Key).To(Equal("some-key"))
				Expect(agent.StopCall.WasCalled).To(BeTrue())
			})

			Context("and the agent fails to stop", func() {
				It("returns agent stop error", func() {
					kv.GetCall.Returns.Error = errors.New("kv get error")
					agent.StopCall.Returns.Error = errors.New("agent stop error")

					_, err := managedKV.Get("some-key")
					Expect(err).To(MatchError("agent stop error"))

					Expect(agent.StartCall.WasCalled).To(BeTrue())
					Expect(kv.GetCall.Receives.Key).To(Equal("some-key"))
					Expect(agent.StopCall.WasCalled).To(BeTrue())
				})
			})
		})

		Context("when the agent fails to stop", func() {
			It("returns the error", func() {
				agent.StopCall.Returns.Error = errors.New("agent stop error")

				_, err := managedKV.Get("some-key")
				Expect(err).To(MatchError("agent stop error"))

				Expect(agent.StartCall.WasCalled).To(BeTrue())
				Expect(kv.GetCall.Receives.Key).To(Equal("some-key"))
				Expect(agent.StopCall.WasCalled).To(BeTrue())
			})
		})
	})
})

type fakeAgent struct {
	StartCall struct {
		WasCalled bool
		Returns   struct {
			Error error
		}
	}

	StopCall struct {
		WasCalled bool
		Returns   struct {
			Error error
		}
	}
}

func (a *fakeAgent) Start() error {
	a.StartCall.WasCalled = true

	return a.StartCall.Returns.Error
}

func (a *fakeAgent) Stop() error {
	a.StopCall.WasCalled = true

	return a.StopCall.Returns.Error
}

type fakeKV struct {
	SetCall struct {
		Receives struct {
			Key   string
			Value string
		}
		Returns struct {
			Error error
		}
	}

	GetCall struct {
		Receives struct {
			Key string
		}
		Returns struct {
			Value string
			Error error
		}
	}
}

func (kv *fakeKV) Set(key, value string) error {
	kv.SetCall.Receives.Key = key
	kv.SetCall.Receives.Value = value

	return kv.SetCall.Returns.Error
}

func (kv *fakeKV) Get(key string) (string, error) {
	kv.GetCall.Receives.Key = key

	return kv.GetCall.Returns.Value, kv.GetCall.Returns.Error
}

type fakeCatalog struct {
	NodeCall struct {
		CallCount int
		Returns   struct {
			NodesFunc func(int) []consul.Node
			ErrorFunc func(int) error
		}
	}
}

func (c *fakeCatalog) Nodes() ([]consul.Node, error) {
	callCount := c.NodeCall.CallCount
	c.NodeCall.CallCount++

	return c.NodeCall.Returns.NodesFunc(callCount), c.NodeCall.Returns.ErrorFunc(callCount)
}

package confab_test

import (
	"errors"
	"time"

	"github.com/cloudfoundry-incubator/consul-release/src/confab"
	"github.com/cloudfoundry/cf-release/src/confab/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CallWithTimeout", func() {

	Context("given a function", func() {
		var (
			clock           confab.Clock
			callWithTimeout confab.CallWithTimeout
			retryDelay      time.Duration
			timeout         confab.Timeout
		)
		BeforeEach(func() {
			clock = &fakes.Clock{}
			retryDelay = 1 * time.Millisecond
			timeout = confab.NewTimeout(make(chan time.Time))
			callWithTimeout = confab.NewCallWithTimeout(clock, retryDelay)
		})

		It("retries till the function is succesful within given timeout", func() {
			callCount := 0
			errorProneFunction := func() error {
				callCount++
				if callCount < 10 {
					return errors.New("some error occured")
				}
				return nil
			}

			err := callWithTimeout.Try(timeout, errorProneFunction)
			Expect(err).NotTo(HaveOccurred())
			Expect(callCount).To(Equal(10))
		})

		Context("failure cases", func() {
			It("returns an error if the function doesn't succeed", func() {
				timer := make(chan time.Time)
				timeout := confab.NewTimeout(timer)
				timer <- time.Now()

				callCount := 0
				errorProneFunction := func() error {
					callCount++
					return errors.New("some error occured")
				}
				err := callWithTimeout.Try(timeout, errorProneFunction)
				Expect(err).To(MatchError("timeout exceeded"))
				Expect(callCount).To(Equal(0))
			})
		})
	})
})

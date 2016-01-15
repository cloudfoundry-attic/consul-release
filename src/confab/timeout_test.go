package confab_test

import (
	"time"

	"confab"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Timeout", func() {
	Describe("Done", func() {
		It("closes the channel when the timer finishes", func() {
			timer := make(chan time.Time)
			timeout := confab.NewTimeout(timer)

			Expect(timeout.Done()).NotTo(BeClosed())

			timer <- time.Now()

			Eventually(timeout.Done).Should(BeClosed())
		})
	})
})

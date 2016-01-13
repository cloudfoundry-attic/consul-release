package bosh_test

import (
	"acceptance-tests/testing/bosh"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Stemcell", func() {
	Context("Latest", func() {
		It("should return the latest stemcell available", func() {
			stemcell := bosh.NewStemcell()
			stemcell.Versions = []string{
				"2127",
				"3147",
				"389",
				"3126",
			}

			Expect(stemcell.Latest()).To(Equal("3147"))
		})

		PIt("should handle no installed stemcells", func() {})
	})
})

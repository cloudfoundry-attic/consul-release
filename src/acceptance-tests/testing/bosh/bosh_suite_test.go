package bosh_test

import (
	"acceptance-tests/testing/bosh"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestBosh(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "testing/bosh")
}

var _ = AfterEach(func() {
	bosh.ResetBodyReader()
})

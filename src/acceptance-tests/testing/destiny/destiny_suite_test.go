package destiny_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestDestiny(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "testing/destiny")
}

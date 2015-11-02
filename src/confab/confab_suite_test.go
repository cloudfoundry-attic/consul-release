package confab_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestConfab(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Confab Suite")
}

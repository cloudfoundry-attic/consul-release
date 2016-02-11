package confab_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestConfab(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "confab")
}

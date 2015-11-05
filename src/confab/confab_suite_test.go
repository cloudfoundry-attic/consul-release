package confab_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestConfab(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Confab Suite")
}

var (
	pathToFakeAgent string
)

var _ = BeforeSuite(func() {
	var err error
	pathToFakeAgent, err = gexec.Build("confab/fakes/agent")
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

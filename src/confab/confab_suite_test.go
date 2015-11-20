package confab_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestConfab(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "confab")
}

var (
	pathToFakeProcess string
)

var _ = BeforeSuite(func() {
	var err error
	pathToFakeProcess, err = gexec.Build("confab/fakes/process")
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

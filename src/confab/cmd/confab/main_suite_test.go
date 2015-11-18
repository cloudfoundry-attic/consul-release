package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestMain(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "confab/cmd/confab")
}

var (
	pathToFakeAgent string
	pathToConfab    string
)

var _ = BeforeSuite(func() {
	var err error
	pathToFakeAgent, err = gexec.Build("confab/fakes/agent")
	Expect(err).NotTo(HaveOccurred())

	pathToConfab, err = gexec.Build("confab/cmd/confab")
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

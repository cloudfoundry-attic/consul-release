package chaperon_test

import (
	"io/ioutil"
	"os"
	"strconv"

	"github.com/cloudfoundry-incubator/consul-release/src/confab/chaperon"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("IsRunningProcess", func() {
	var (
		pidFile *os.File
	)

	BeforeEach(func() {
		var err error

		pidFile, err = ioutil.TempFile("", "")
		Expect(err).NotTo(HaveOccurred())
	})

	DescribeTable("when the pidfile exists",
		func(pid string, isRunning bool) {
			err := ioutil.WriteFile(pidFile.Name(), []byte(pid), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			processIsRunning := chaperon.IsRunningProcess(pidFile.Name())
			Expect(processIsRunning).To(Equal(isRunning))
		},
		Entry("returns false if the process is not running", "-1", false),
		Entry("returns true if the process is running", strconv.Itoa(os.Getpid()), true),
		Entry("returns false if the pidfile contains garbage", "something-bad", false),
	)

	It("returns false if the pidfile does not exist", func() {
		processIsRunning := chaperon.IsRunningProcess("/nonexistent/pidfile")
		Expect(processIsRunning).To(BeFalse())
	})
})

package utils_test

import (
	"os/exec"
	"runtime"

	"github.com/cloudfoundry-incubator/consul-release/src/confab/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("a process that is about to die", func() {
	It("is running, and eventually stops", func() {
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("ping", "-t", "127.0.0.1")
		} else {
			cmd = exec.Command("ping", "127.0.0.1")
		}
		Expect(cmd.Start()).To(Succeed())
		defer cmd.Process.Kill()
		Expect(utils.CheckProcessRunning(cmd.Process)).To(Succeed())
		cmd.Process.Kill()
		cmd.Wait()
		Expect(utils.CheckProcessRunning(cmd.Process)).ToNot(Succeed())
	})
})

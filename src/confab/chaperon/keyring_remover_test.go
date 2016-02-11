package chaperon_test

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cloudfoundry-incubator/consul-release/src/confab/chaperon"
	"github.com/cloudfoundry-incubator/consul-release/src/confab/fakes"
	"github.com/pivotal-golang/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf-experimental/gomegamatchers"
)

var _ = Describe("KeyringRemover", func() {
	Describe("Execute", func() {
		var (
			dataDir string
			keyring *os.File
			logger  *fakes.Logger
			remover chaperon.KeyringRemover
		)

		BeforeEach(func() {
			var err error
			dataDir, err = ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())

			keyring, err = ioutil.TempFile(dataDir, "keyring")
			Expect(err).NotTo(HaveOccurred())

			logger = &fakes.Logger{}

			remover = chaperon.NewKeyringRemover(keyring.Name(), logger)
		})

		It("removes the keyring file", func() {
			err := remover.Execute()
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(keyring.Name())
			Expect(err).To(MatchError(ContainSubstring("no such file")))

			Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
				{
					Action: "keyring-remover.execute",
					Data: []lager.Data{{
						"keyring": keyring.Name(),
					}},
				},
				{
					Action: "keyring-remover.execute.success",
					Data: []lager.Data{{
						"keyring": keyring.Name(),
					}},
				},
			}))
		})

		Context("when the file does not exist", func() {
			It("does not error", func() {
				err := os.Remove(keyring.Name())
				Expect(err).NotTo(HaveOccurred())

				err = remover.Execute()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("failure cases", func() {
			Context("when the file cannot be removed", func() {
				It("returns an error", func() {
					err := os.Chmod(dataDir, 0000)
					Expect(err).NotTo(HaveOccurred())

					err = remover.Execute()
					Expect(err).To(MatchError(ContainSubstring("permission denied")))

					Expect(logger.Messages).To(ContainSequence([]fakes.LoggerMessage{
						{
							Action: "keyring-remover.execute",
							Data: []lager.Data{{
								"keyring": keyring.Name(),
							}},
						},
						{
							Action: "keyring-remover.execute.failed",
							Error:  fmt.Errorf("remove %s: permission denied", keyring.Name()),
							Data: []lager.Data{{
								"keyring": keyring.Name(),
							}},
						},
					}))
				})
			})
		})
	})
})

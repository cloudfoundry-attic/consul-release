package helpers_test

import (
	"acceptance-tests/helpers"
	"errors"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func writeConfigJSON(json string) (string, error) {
	tempFile, err := ioutil.TempFile("", "config")
	if err != nil {
		return "", err
	}

	err = ioutil.WriteFile(tempFile.Name(), []byte(json), os.ModePerm)
	if err != nil {
		return "", err
	}

	return tempFile.Name(), nil
}

var _ = Describe("configuration", func() {
	Describe("LoadConfig", func() {
		Context("with a valid config options", func() {
			var configFilePath string

			BeforeEach(func() {
				var err error
				configFilePath, err = writeConfigJSON(`{
					"bind_address": "some-bind-address",
					"bosh_target": "some-bosh-target",
					"iaas_settings_consul_stub_path": "/some-consul-stub-path",
					"iaas_settings_turbulence_stub_path": "/some-turbulence-stub-path",
					"turbulence_properties_stub_path": "/some-turbulence-properties-stub-path",
					"cpi_release_location": "some-cpi-release-location",
					"cpi_release_name": "some-cpi-release-name",
					"turbulence_release_location": "some-turbulence-release-location",
					"bosh_operation_timeout": "some-bosh-operation-timeout",
					"turbulence_operation_timeout": "some-turbulence-operation-timeout"
				}`)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				err := os.Remove(configFilePath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("loads the config from the given path", func() {
				config, err := helpers.LoadConfig(configFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(config).To(Equal(helpers.Config{
					BindAddress:                    "some-bind-address",
					BoshTarget:                     "some-bosh-target",
					IAASSettingsConsulStubPath:     "/some-consul-stub-path",
					IAASSettingsTurbulenceStubPath: "/some-turbulence-stub-path",
					TurbulencePropertiesStubPath:   "/some-turbulence-properties-stub-path",
					CPIReleaseLocation:             "some-cpi-release-location",
					CPIReleaseName:                 "some-cpi-release-name",
					TurbulenceReleaseLocation:      "some-turbulence-release-location",
					TurbulenceReleaseName:          "turbulence",
					BoshOperationTimeout:           "some-bosh-operation-timeout",
					TurbulenceOperationTimeout:     "some-turbulence-operation-timeout",
				}))
			})
		})

		Context("with an invalid config json file location", func() {
			It("should return an error if the file does not exist", func() {
				_, err := helpers.LoadConfig("someblahblahfile")
				Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
			})
		})

		Context("when config file contains invalid JSON", func() {
			var configFilePath string

			BeforeEach(func() {
				var err error
				configFilePath, err = writeConfigJSON("%%%")
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				err := os.Remove(configFilePath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return an error", func() {
				_, err := helpers.LoadConfig(configFilePath)
				Expect(err).To(MatchError(ContainSubstring("invalid character '%'")))
			})
		})

		Context("when the bind_address is missing", func() {
			var configFilePath string

			BeforeEach(func() {
				var err error
				configFilePath, err = writeConfigJSON("{}")
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				err := os.Remove(configFilePath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return an error", func() {
				_, err := helpers.LoadConfig(configFilePath)
				Expect(err).To(MatchError(errors.New("missing `bind_address` - specify which address consul should bind to")))
			})
		})

		Context("when the bosh_target is missing", func() {
			var configFilePath string

			BeforeEach(func() {
				var err error
				configFilePath, err = writeConfigJSON(`{
					"bind_address": "some-bind-address"
				}`)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				err := os.Remove(configFilePath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return an error", func() {
				_, err := helpers.LoadConfig(configFilePath)
				Expect(err).To(MatchError(errors.New("missing `bosh_target` - e.g. 'lite' or '192.168.50.4'")))
			})
		})

		Context("when the iaas_settings_consul_stub_path is missing", func() {
			var configFilePath string

			BeforeEach(func() {
				var err error
				configFilePath, err = writeConfigJSON(`{
					"bind_address": "some-bind-address",
					"bosh_target": "some-bosh-target"
				}`)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				err := os.Remove(configFilePath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return an error", func() {
				_, err := helpers.LoadConfig(configFilePath)
				Expect(err).To(MatchError(errors.New("missing `iaas_settings_consul_stub_path` - path to consul stub file")))
			})
		})

		Context("when the iaas_settings_consul_stub_path is not absolute", func() {
			var configFilePath string

			BeforeEach(func() {
				var err error
				configFilePath, err = writeConfigJSON(`{
					"bind_address": "some-bind-address",
					"bosh_target": "some-bosh-target",
					"iaas_settings_consul_stub_path": "some-consul-stub-path"
				}`)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				err := os.Remove(configFilePath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return an error", func() {
				_, err := helpers.LoadConfig(configFilePath)
				Expect(err).To(MatchError(errors.New("invalid `iaas_settings_consul_stub_path` \"some-consul-stub-path\" - must be an absolute path")))
			})
		})

		Context("when turbulence config is not provided", func() {
			var configFilePath string

			BeforeEach(func() {
				var err error
				configFilePath, err = writeConfigJSON(`{
					"bind_address": "some-bind-address",
					"bosh_target": "some-bosh-target",
					"iaas_settings_consul_stub_path": "/some-consul-stub-path"
				}`)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				err := os.Remove(configFilePath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns a valid config", func() {
				config, err := helpers.LoadConfig(configFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(config).To(Equal(helpers.Config{
					BindAddress:                "some-bind-address",
					BoshTarget:                 "some-bosh-target",
					IAASSettingsConsulStubPath: "/some-consul-stub-path",
					TurbulenceReleaseName:      "turbulence",
				}))
			})
		})

		Context("when some turbulence config is provided", func() {
			Context("when iaas_settings_turbulence_stub_path is missing", func() {
				var configFilePath string

				BeforeEach(func() {
					var err error
					configFilePath, err = writeConfigJSON(`{
						"bind_address": "some-bind-address",
						"bosh_target": "some-bosh-target",
						"iaas_settings_consul_stub_path": "/some-consul-stub-path",
						"turbulence_properties_stub_path": "/some-turbulence-properties-stub-path",
						"cpi_release_location": "some-cpi-release-location",
						"cpi_release_name": "some-cpi-release-name",
						"turbulence_release_location": "some-turbulence-release-location"
					}`)
					Expect(err).NotTo(HaveOccurred())
				})

				AfterEach(func() {
					err := os.Remove(configFilePath)
					Expect(err).NotTo(HaveOccurred())
				})

				It("returns an error", func() {
					_, err := helpers.LoadConfig(configFilePath)
					Expect(err).To(MatchError(errors.New("missing `iaas_settings_turbulence_stub_path` - path to turbulence stub file")))
				})
			})

			Context("when iaas_settings_turbulence_stub_path is not absolute", func() {
				var configFilePath string

				BeforeEach(func() {
					var err error
					configFilePath, err = writeConfigJSON(`{
						"bind_address": "some-bind-address",
						"bosh_target": "some-bosh-target",
						"iaas_settings_consul_stub_path": "/some-consul-stub-path",
						"iaas_settings_turbulence_stub_path": "some-turbulence-stub-path",
						"turbulence_properties_stub_path": "/some-turbulence-properties-stub-path",
						"cpi_release_location": "some-cpi-release-location",
						"cpi_release_name": "some-cpi-release-name",
						"turbulence_release_location": "some-turbulence-release-location"
					}`)
					Expect(err).NotTo(HaveOccurred())
				})

				AfterEach(func() {
					err := os.Remove(configFilePath)
					Expect(err).NotTo(HaveOccurred())
				})

				It("returns an error", func() {
					_, err := helpers.LoadConfig(configFilePath)
					Expect(err).To(MatchError(errors.New("invalid `iaas_settings_turbulence_stub_path` \"some-turbulence-stub-path\" - must be an absolute path")))
				})
			})

			Context("when turbulence_properties_stub_path is missing", func() {
				var configFilePath string

				BeforeEach(func() {
					var err error
					configFilePath, err = writeConfigJSON(`{
						"bind_address": "some-bind-address",
						"bosh_target": "some-bosh-target",
						"iaas_settings_consul_stub_path": "/some-consul-stub-path",
						"iaas_settings_turbulence_stub_path": "/some-turbulence-stub-path",
						"cpi_release_location": "some-cpi-release-location",
						"cpi_release_name": "some-cpi-release-name",
						"turbulence_release_location": "some-turbulence-release-location"
					}`)
					Expect(err).NotTo(HaveOccurred())
				})

				AfterEach(func() {
					err := os.Remove(configFilePath)
					Expect(err).NotTo(HaveOccurred())
				})

				It("returns an error", func() {
					_, err := helpers.LoadConfig(configFilePath)
					Expect(err).To(MatchError(errors.New("missing `turbulence_properties_stub_path` - path to turbulence properties stub file")))
				})
			})

			Context("when turbulence_properties_stub_path is not absolute", func() {
				var configFilePath string

				BeforeEach(func() {
					var err error
					configFilePath, err = writeConfigJSON(`{
						"bind_address": "some-bind-address",
						"bosh_target": "some-bosh-target",
						"iaas_settings_consul_stub_path": "/some-consul-stub-path",
						"turbulence_properties_stub_path": "some-turbulence-properties-stub-path",
						"iaas_settings_turbulence_stub_path": "/some-turbulence-stub-path",
						"cpi_release_location": "some-cpi-release-location",
						"cpi_release_name": "some-cpi-release-name",
						"turbulence_release_location": "some-turbulence-release-location"
					}`)
					Expect(err).NotTo(HaveOccurred())
				})

				AfterEach(func() {
					err := os.Remove(configFilePath)
					Expect(err).NotTo(HaveOccurred())
				})

				It("returns an error", func() {
					_, err := helpers.LoadConfig(configFilePath)
					Expect(err).To(MatchError(errors.New("invalid `turbulence_properties_stub_path` \"some-turbulence-properties-stub-path\" - must be an absolute path")))
				})
			})

			Context("when cpi_release_location is missing", func() {
				var configFilePath string

				BeforeEach(func() {
					var err error
					configFilePath, err = writeConfigJSON(`{
						"bind_address": "some-bind-address",
						"bosh_target": "some-bosh-target",
						"iaas_settings_consul_stub_path": "/some-consul-stub-path",
						"iaas_settings_turbulence_stub_path": "/some-turbulence-stub-path",
						"turbulence_properties_stub_path": "/some-turbulence-properties-stub-path",
						"cpi_release_name": "some-cpi-release-name",
						"turbulence_release_location": "some-turbulence-release-location"
					}`)
					Expect(err).NotTo(HaveOccurred())
				})

				AfterEach(func() {
					err := os.Remove(configFilePath)
					Expect(err).NotTo(HaveOccurred())
				})

				It("returns an error", func() {
					_, err := helpers.LoadConfig(configFilePath)
					Expect(err).To(MatchError(errors.New("missing `cpi_release_location` - location of cpi release")))
				})
			})

			Context("when cpi_release_name is missing", func() {
				var configFilePath string

				BeforeEach(func() {
					var err error
					configFilePath, err = writeConfigJSON(`{
						"bind_address": "some-bind-address",
						"bosh_target": "some-bosh-target",
						"iaas_settings_consul_stub_path": "/some-consul-stub-path",
						"iaas_settings_turbulence_stub_path": "/some-turbulence-stub-path",
						"turbulence_properties_stub_path": "/some-turbulence-properties-stub-path",
						"cpi_release_location": "some-cpi-release-location",
						"turbulence_release_location": "some-turbulence-release-location"
					}`)
					Expect(err).NotTo(HaveOccurred())
				})

				AfterEach(func() {
					err := os.Remove(configFilePath)
					Expect(err).NotTo(HaveOccurred())
				})

				It("returns an error", func() {
					_, err := helpers.LoadConfig(configFilePath)
					Expect(err).To(MatchError(errors.New("missing `cpi_release_name` - name of cpi release")))
				})
			})

			Context("when turbulence_release_location is missing", func() {
				var configFilePath string

				BeforeEach(func() {
					var err error
					configFilePath, err = writeConfigJSON(`{
						"bind_address": "some-bind-address",
						"bosh_target": "some-bosh-target",
						"iaas_settings_consul_stub_path": "/some-consul-stub-path",
						"iaas_settings_turbulence_stub_path": "/some-turbulence-stub-path",
						"turbulence_properties_stub_path": "/some-turbulence-properties-stub-path",
						"cpi_release_location": "some-cpi-release-location",
						"cpi_release_name": "some-cpi-release-name"
					}`)
					Expect(err).NotTo(HaveOccurred())
				})

				AfterEach(func() {
					err := os.Remove(configFilePath)
					Expect(err).NotTo(HaveOccurred())
				})

				It("returns an error", func() {
					_, err := helpers.LoadConfig(configFilePath)
					Expect(err).To(MatchError(errors.New("missing `turbulence_release_location` - location of turbulence release")))
				})
			})
		})
	})

	Describe("ConfigPath", func() {
		var configPath string

		BeforeEach(func() {
			configPath = os.Getenv("CONSATS_CONFIG")
		})

		AfterEach(func() {
			os.Setenv("CONSATS_CONFIG", configPath)
		})

		Context("when a valid path is set", func() {
			It("returns the path", func() {
				os.Setenv("CONSATS_CONFIG", "/tmp/some-config.json")
				path, err := helpers.ConfigPath()
				Expect(err).NotTo(HaveOccurred())
				Expect(path).To(Equal("/tmp/some-config.json"))
			})
		})

		Context("when path is not set", func() {
			It("returns an error", func() {
				os.Setenv("CONSATS_CONFIG", "")
				_, err := helpers.ConfigPath()
				Expect(err).To(MatchError(`$CONSATS_CONFIG "" does not specify an absolute path to test config file`))
			})
		})

		Context("when the path is not absolute", func() {
			It("returns an error", func() {
				os.Setenv("CONSATS_CONFIG", "some/path.json")
				_, err := helpers.ConfigPath()
				Expect(err).To(MatchError(`$CONSATS_CONFIG "some/path.json" does not specify an absolute path to test config file`))
			})
		})
	})
})

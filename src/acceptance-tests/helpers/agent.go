package helpers

import (
	"io/ioutil"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	. "github.com/onsi/gomega"
)

type AgentRunner struct {
	consulProcess ifrit.Process
	running       bool
	dataDir       string
	configDir     string
	serverIps     []string
	bindAddress   string

	mutex *sync.RWMutex
}

const defaultDataDirPrefix = "consul_data"
const defaultConfigDirPrefix = "consul_config"

func NewAgentRunner(serverIps []string, bindAddress string) *AgentRunner {
	Expect(len(serverIps)).NotTo(Equal(0))

	return &AgentRunner{
		serverIps:   serverIps,
		bindAddress: bindAddress,
		mutex:       &sync.RWMutex{},
	}
}

func (runner *AgentRunner) Start() {
	runner.mutex.Lock()
	defer runner.mutex.Unlock()

	if runner.running {
		return
	}

	tmpDir, err := ioutil.TempDir("", defaultDataDirPrefix)
	Expect(err).NotTo(HaveOccurred())
	runner.dataDir = tmpDir

	tmpDir, err = ioutil.TempDir("", defaultConfigDirPrefix)
	Expect(err).NotTo(HaveOccurred())
	runner.configDir = tmpDir

	os.MkdirAll(runner.dataDir, 0700)

	configFilePath := writeConfigFile(
		runner.configDir,
		runner.dataDir,
		runner.bindAddress,
		runner.serverIps,
	)

	timeout := 1 * time.Minute
	process := ginkgomon.Invoke(ginkgomon.New(ginkgomon.Config{
		Name:              "consul_agent",
		AnsiColorCode:     "35m",
		StartCheck:        "agent: Join completed.",
		StartCheckTimeout: timeout,
		Command: exec.Command(
			"consul",
			"agent",
			"--config-file", configFilePath,
		),
	}))
	runner.consulProcess = process

	ready := process.Ready()
	Eventually(ready, timeout, 100*time.Millisecond).Should(BeClosed(), "Expected consul to be up and running")

	runner.running = true
}

func (runner *AgentRunner) Stop() {
	runner.mutex.Lock()
	defer runner.mutex.Unlock()

	if !runner.running {
		return
	}

	ginkgomon.Interrupt(runner.consulProcess, 5*time.Second)

	os.RemoveAll(runner.dataDir)
	os.RemoveAll(runner.configDir)
	runner.consulProcess = nil
	runner.running = false
}

func (runner *AgentRunner) NewClient() *api.Client {
	client, err := api.NewClient(api.DefaultConfig())
	Expect(err).NotTo(HaveOccurred())
	return client
}

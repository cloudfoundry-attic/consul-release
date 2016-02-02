package main

import (
	"confab"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/pivotal-golang/lager"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/command/agent"
	"github.com/pivotal-golang/clock"
)

type stringSlice []string

func (ss *stringSlice) String() string {
	return fmt.Sprintf("%s", *ss)
}

func (ss *stringSlice) Set(value string) error {
	*ss = append(*ss, value)

	return nil
}

var (
	recursors  stringSlice
	configFile string

	stdout = log.New(os.Stdout, "", 0)
	stderr = log.New(os.Stderr, "", 0)
)

func main() {
	var controller confab.Controller

	flagSet := flag.NewFlagSet("flags", flag.ContinueOnError)
	flagSet.Var(&recursors, "recursor", "specifies the address of an upstream DNS `server`, may be specified multiple times")
	flagSet.StringVar(&configFile, "config-file", "", "specifies the config `file`")

	if len(os.Args) < 2 {
		printUsageAndExit("invalid number of arguments", flagSet)
	}

	if err := flagSet.Parse(os.Args[2:]); err != nil {
		os.Exit(1)
	}

	configFileContents, err := ioutil.ReadFile(configFile)
	if err != nil {
		stderr.Printf("error reading configuration file: %s", err)
		os.Exit(1)
	}

	config, err := confab.ConfigFromJSON(configFileContents)
	if err != nil {
		stderr.Printf("error reading configuration file: %s", err)
		os.Exit(1)
	}

	path, err := exec.LookPath(config.Path.AgentPath)
	if err != nil {
		printUsageAndExit(fmt.Sprintf("\"agent_path\" %q cannot be found", config.Path.AgentPath), flagSet)
	}

	if len(config.Path.PIDFile) == 0 {
		printUsageAndExit("\"pid_file\" cannot be empty", flagSet)
	}

	logger := lager.NewLogger("confab")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))

	agentRunner := &confab.AgentRunner{
		Path:      path,
		PIDFile:   config.Path.PIDFile,
		ConfigDir: config.Path.ConsulConfigDir,
		Recursors: recursors,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
		Logger:    logger,
	}

	consulAPIClient, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		panic(err) // not tested, NewClient never errors
	}

	agentClient := &confab.AgentClient{
		ExpectedMembers: config.Consul.Agent.Servers.LAN,
		ConsulAPIAgent:  consulAPIClient.Agent(),
		ConsulRPCClient: nil,
		Logger:          logger,
	}

	controller = confab.Controller{
		AgentRunner:    agentRunner,
		AgentClient:    agentClient,
		SyncRetryDelay: 1 * time.Second,
		SyncRetryClock: clock.NewClock(),
		EncryptKeys:    config.Consul.EncryptKeys,
		Logger:         logger,
		ServiceDefiner: confab.ServiceDefiner{logger},
		ConfigDir:      config.Path.ConsulConfigDir,
		Config:         config,
	}

	switch os.Args[1] {
	case "start":
		start(flagSet, path, controller, agentClient)
	case "stop":
		stop(path, controller, agentClient)
	default:
		printUsageAndExit(fmt.Sprintf("invalid COMMAND %q", os.Args[1]), flagSet)
	}
}

func start(flagSet *flag.FlagSet, path string, controller confab.Controller, agentClient *confab.AgentClient) {
	timeout := confab.NewTimeout(time.After(time.Duration(controller.Config.Confab.TimeoutInSeconds) * time.Second))

	_, err := os.Stat(controller.Config.Path.ConsulConfigDir)
	if err != nil {
		printUsageAndExit(fmt.Sprintf("\"consul_config_dir\" %q could not be found",
			controller.Config.Path.ConsulConfigDir), flagSet)
	}

	if len(agentClient.ExpectedMembers) == 0 {
		printUsageAndExit("at least one \"expected-member\" must be provided", flagSet)
	}

	err = controller.WriteServiceDefinitions()
	if err != nil {
		stderr.Printf("error writing service definitions: %s", err)
		os.Exit(1)
	}

	err = controller.BootAgent(timeout)
	if err != nil {
		stderr.Printf("error booting consul agent: %s", err)
		exit(controller, 1)
	}

	if controller.Config.Consul.Agent.Server {
		configureServer(controller, agentClient, timeout)
	} else {
		configureClient(controller)
	}
}

func configureServer(controller confab.Controller, agentClient *confab.AgentClient, timeout confab.Timeout) {
	rpcClient, err := agent.NewRPCClient("localhost:8400")

	if err != nil {
		stderr.Printf("error connecting to RPC server: %s", err)
		exit(controller, 1)
	}

	agentClient.ConsulRPCClient = &confab.RPCClient{*rpcClient}
	err = controller.ConfigureServer(timeout)
	if err != nil {
		stderr.Printf("error configuring server: %s", err)
		exit(controller, 1)
	}
}

func configureClient(controller confab.Controller) {
	if err := controller.ConfigureClient(); err != nil {
		stderr.Printf("error configuring client: %s", err)
		exit(controller, 1)
	}
}

func stop(path string, controller confab.Controller, agentClient *confab.AgentClient) {
	rpcClient, err := agent.NewRPCClient("localhost:8400")
	if err != nil {
		stderr.Printf("error connecting to RPC server: %s", err)
		exit(controller, 1)
	}

	agentClient.ConsulRPCClient = &confab.RPCClient{*rpcClient}
	stderr.Printf("stopping agent")
	controller.StopAgent()
	stderr.Printf("stopped agent")
}

func printUsageAndExit(message string, flagSet *flag.FlagSet) {
	stderr.Printf("%s\n\n", message)
	stderr.Println("usage: confab COMMAND OPTIONS\n")
	stderr.Println("COMMAND: \"start\" or \"stop\"")
	stderr.Println("\nOPTIONS:")
	flagSet.PrintDefaults()
	stderr.Println()
	os.Exit(1)
}

func validCommand(command string) bool {
	for _, c := range []string{"start", "stop"} {
		if command == c {
			return true
		}
	}

	return false
}

func exit(controller confab.Controller, code int) {
	controller.StopAgent()
	os.Exit(code)
}

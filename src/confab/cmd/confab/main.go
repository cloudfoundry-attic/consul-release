package main

import (
	"confab"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

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
	isServer        bool
	agentPath       string
	consulConfigDir string
	pidFile         string
	expectedMembers stringSlice
	encryptionKeys  stringSlice

	stdout = log.New(os.Stdout, "", 0)
	stderr = log.New(os.Stderr, "", 0)
)

func main() {
	var controller confab.Controller

	flagSet := flag.NewFlagSet("flags", flag.ContinueOnError)
	flagSet.BoolVar(&isServer, "server", false, "whether to start the agent in server mode")
	flagSet.StringVar(&agentPath, "agent-path", "", "path to the on-filesystem consul `executable`")
	flagSet.StringVar(&consulConfigDir, "consul-config-dir", "", "path to consul configuration `directory`")
	flagSet.StringVar(&pidFile, "pid-file", "", "path to consul PID `file`")
	flagSet.Var(&expectedMembers, "expected-member", "address `list` of the expected members")
	flagSet.Var(&encryptionKeys, "encryption-key", "`key` used to encrypt consul traffic")

	if len(os.Args) < 2 {
		printUsageAndExit("invalid number of arguments", flagSet, controller)
	}

	flagSet.Parse(os.Args[2:])

	path, err := exec.LookPath(agentPath)
	if err != nil {
		printUsageAndExit(fmt.Sprintf("\"agent-path\" %q cannot be found", agentPath), flagSet, controller)
	}

	if len(pidFile) == 0 {
		printUsageAndExit("\"pid-file\" cannot be empty", flagSet, controller)
	}

	agentRunner := &confab.AgentRunner{
		Path:      path,
		PIDFile:   pidFile,
		ConfigDir: consulConfigDir,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
	}

	consulAPIClient, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		panic(err) // not tested, NewClient never errors
	}

	agentClient := &confab.AgentClient{
		ExpectedMembers: expectedMembers,
		ConsulAPIAgent:  consulAPIClient.Agent(),
		ConsulRPCClient: nil,
	}

	controller = confab.Controller{
		AgentRunner:    agentRunner,
		AgentClient:    agentClient,
		MaxRetries:     10,
		SyncRetryDelay: 1 * time.Second,
		SyncRetryClock: clock.NewClock(),
		EncryptKeys:    encryptionKeys,
		SSLDisabled:    false,
		Logger:         stdout,
	}

	switch os.Args[1] {
	case "start":
		start(flagSet, path, controller, agentClient)
	case "stop":
		stop(path, controller, agentClient)
	default:
		printUsageAndExit(fmt.Sprintf("invalid COMMAND %q", os.Args[1]), flagSet, controller)
	}
}

func start(flagSet *flag.FlagSet, path string, controller confab.Controller, agentClient *confab.AgentClient) {
	_, err := os.Stat(consulConfigDir)
	if err != nil {
		printUsageAndExit(fmt.Sprintf("\"consul-config-dir\" %q could not be found", consulConfigDir), flagSet, controller)
	}

	if len(expectedMembers) == 0 {
		printUsageAndExit("at least one \"expected-member\" must be provided", flagSet, controller)
	}

	err = controller.BootAgent()
	if err != nil {
		stderr.Printf("error booting consul agent: %s", err)

		// DO NOT call StopAgent:
		//   - the agent may aleady be running
		//   - the pidfile may contain the PID of another running process
		os.Exit(1)
	}

	if isServer {
		configureServer(controller, agentClient)
	}
}

func configureServer(controller confab.Controller, agentClient *confab.AgentClient) {
	rpcClient, err := agent.NewRPCClient("localhost:8400")
	if err != nil {
		stderr.Printf("error connecting to RPC server: %s", err)
		exit(controller, 1)
	}

	agentClient.ConsulRPCClient = &confab.RPCClient{*rpcClient}
	err = controller.ConfigureServer()
	if err != nil {
		stderr.Printf("error configuring server: %s", err)
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

func printUsageAndExit(message string, flagSet *flag.FlagSet, controller confab.Controller) {
	stderr.Printf("%s\n\n", message)
	stderr.Println("usage: confab COMMAND OPTIONS\n")
	stderr.Println("COMMAND: \"start\" or \"stop\"")
	stderr.Println("\nOPTIONS:")
	flagSet.PrintDefaults()
	stderr.Println()
	exit(controller, 1)
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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/command/agent"
)

type outputData struct {
	Args []string
	PID  int
}

func main() {
	// store information about this fake process into JSON
	signal.Ignore()

	var data outputData
	data.PID = os.Getpid()
	data.Args = os.Args[1:]

	// validate command line arguments
	// expect them to look like
	//   fake-thing agent -config-dir=/some/path/to/some/dir
	if len(data.Args) == 0 {
		log.Fatal("expecting command as first argment")
	}
	var configDir string
	flagSet := flag.NewFlagSet("", flag.ExitOnError)
	flagSet.StringVar(&configDir, "config-dir", "", "config directory")
	flagSet.Parse(data.Args[1:])
	if configDir == "" {
		log.Fatal("missing required config-dir flag")
	}

	writeOutput(configDir, data)
	defer writeOutput(configDir, data)

	// read input options provided to us by the test
	var inputOptions struct {
		Slow          bool
		WaitForHUP    bool
		StayAlive     bool
		RunClient     bool
		RunServer     bool
		Members       []string
		FailRPCServer bool
	}
	if optionsBytes, err := ioutil.ReadFile(filepath.Join(configDir, "options.json")); err == nil {
		json.Unmarshal(optionsBytes, &inputOptions)
	}

	fmt.Fprintf(os.Stdout, "some standard out")
	fmt.Fprintf(os.Stderr, "some standard error")

	if inputOptions.Slow {
		time.Sleep(10 * time.Second)
	}

	if inputOptions.WaitForHUP {
		for {
			time.Sleep(time.Second)
		}
	}

	if inputOptions.RunClient {
		fmt.Println("running client")
		ClientListener{
			Addr:    "127.0.0.1:8500",
			Members: inputOptions.Members,
		}.Serve()
	}

	tcpAddr := ""
	if !inputOptions.FailRPCServer {
		tcpAddr = "127.0.0.1:8400"
	}

	if inputOptions.RunServer {
		fmt.Println("running server")
		ServerListener{
			HTTPAddr:  "127.0.0.1:8500",
			TCPAddr:   tcpAddr,
			Members:   inputOptions.Members,
			StayAlive: inputOptions.StayAlive,
		}.Serve()
	}
}

func writeOutput(configDir string, data outputData) {
	outputBytes, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	// save information JSON to the config dir
	err = ioutil.WriteFile(filepath.Join(configDir, "fake-output.json"), outputBytes, 0600)
	if err != nil {
		panic(err)
	}
}

type ClientListener struct {
	Addr    string
	Members []string
}

func (cl ClientListener) Serve() {
	listener, err := net.Listen("tcp", cl.Addr)
	if err != nil {
		panic(err)
	}

	triggerClose := make(chan struct{})

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		var members []api.AgentMember
		for _, member := range cl.Members {
			members = append(members, api.AgentMember{
				Addr: member,
			})
		}
		json.NewEncoder(w).Encode(members)
		triggerClose <- struct{}{}
	})

	server := &http.Server{
		Addr:    cl.Addr,
		Handler: mux,
	}

	go server.Serve(listener)

	<-triggerClose
	time.Sleep(1 * time.Second)
	listener.Close()
}

type ServerListener struct {
	HTTPAddr  string
	TCPAddr   string
	Members   []string
	StayAlive bool
}

func (sl ServerListener) Serve() {
	httpListener, err := net.Listen("tcp", sl.HTTPAddr)
	if err != nil {
		panic(err)
	}

	triggerClose := make(chan struct{})

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		var members []api.AgentMember
		for _, member := range sl.Members {
			members = append(members, api.AgentMember{
				Addr: member,
			})
		}
		json.NewEncoder(w).Encode(members)
		if !sl.StayAlive {
			triggerClose <- struct{}{}
		}
	})

	server := &http.Server{
		Addr:    sl.HTTPAddr,
		Handler: mux,
	}

	go server.Serve(httpListener)

	var mockAgent *FakeAgentBackend
	var agentRPCServer *agent.AgentRPC
	if sl.TCPAddr != "" {

		tcpAddr, err := net.ResolveTCPAddr("tcp", sl.TCPAddr)
		if err != nil {
			panic(err)
		}

		tcpListener, err := net.ListenTCP("tcp", tcpAddr)
		if err != nil {
			panic(err)
		}

		mockAgent = new(FakeAgentBackend)
		agentRPCServer = agent.NewAgentRPC(mockAgent, tcpListener,
			os.Stderr, agent.NewLogWriter(42))

		go func() {
			var useKeyCalled bool
			for {
				if mockAgent.UseKeyCallCount() > 0 && !useKeyCalled {
					fmt.Println("UseKey called")
					useKeyCalled = true
				}

				time.Sleep(1 * time.Second)
			}
		}()

		go func() {
			var leaveCalled bool
			for {
				if mockAgent.LeaveCallCount() > 0 && !leaveCalled {
					fmt.Println("Leave called")
					leaveCalled = true
					triggerClose <- struct{}{}
					triggerClose <- struct{}{}
				}

				time.Sleep(1 * time.Second)
			}
		}()
	}

	<-triggerClose
	<-triggerClose
	time.Sleep(1 * time.Second)
	httpListener.Close()
	if agentRPCServer != nil {
		agentRPCServer.Shutdown()
	}
}

package main

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/command/agent"
	"github.com/hashicorp/consul/logger"
)

type Server struct {
	HTTPAddr     string
	HTTPListener net.Listener

	TCPAddr     string
	TCPListener *net.TCPListener

	OutputWriter *OutputWriter

	Members           []string
	DidLeave          bool
	FailStatsEndpoint bool
}

func (s *Server) Serve() error {
	var err error
	s.HTTPListener, err = net.Listen("tcp", s.HTTPAddr)
	if err != nil {
		return err
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", s.TCPAddr)
	if err != nil {
		return err
	}

	s.TCPListener, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err
	}

	go s.ServeTCP()
	go s.ServeHTTP()

	return nil
}

func (s *Server) ServeHTTP() {
	var (
		useKeyCallCount     int
		installKeyCallCount int
		leaveCallCount      int
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/agent/members", func(w http.ResponseWriter, req *http.Request) {
		var members []api.AgentMember
		for _, member := range s.Members {
			members = append(members, api.AgentMember{
				Addr: member,
				Tags: map[string]string{
					"role": "consul",
				},
			})
		}
		json.NewEncoder(w).Encode(members)
	})
	mux.HandleFunc("/v1/agent/self", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/v1/agent/join/", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/v1/agent/leave", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		leaveCallCount++
		s.OutputWriter.LeaveCalled()
		s.DidLeave = true
	})
	mux.HandleFunc("/v1/status/leader", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`""`)) //s.Members[0]
	})
	mux.HandleFunc("/v1/operator/keyring", func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
		case "POST":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
			installKeyCallCount++
			s.OutputWriter.InstallKeyCalled()
		case "PUT":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
			useKeyCallCount++
			s.OutputWriter.UseKeyCalled()
		case "DELETE":
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	server := &http.Server{
		Addr:    s.HTTPAddr,
		Handler: mux,
	}

	server.Serve(s.HTTPListener)
}

func (s *Server) ServeTCP() {
	mockAgent := new(FakeAgentBackend)
	if s.FailStatsEndpoint {
		mockAgent.StatsReturns(map[string]map[string]string{
			"raft": {
				"commit_index":   "5",
				"last_log_index": "2",
			},
		})
	}

	agentRPCServer := agent.NewAgentRPC(mockAgent, s.TCPListener, os.Stderr, logger.NewLogWriter(42))

	var (
		statsCallCount int
	)

	for {
		switch {
		case mockAgent.LeaveCallCount() > 0:
			agentRPCServer.Shutdown()
		case mockAgent.StatsCallCount() > statsCallCount:
			statsCallCount++
			s.OutputWriter.StatsCalled()
		}

		time.Sleep(10 * time.Millisecond)
	}
}

func (s Server) Exit() error {
	err := s.HTTPListener.Close()
	if err != nil {
		return err
	}

	err = s.TCPListener.Close()
	if err != nil {
		return err
	}

	return nil
}

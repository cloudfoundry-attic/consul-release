package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/testconsumer/server"
)

func main() {
	port, consulURL := parseCommandLineFlags()

	mux := http.NewServeMux()
	consulClient := server.NewConsulClient(consulURL)
	getKVHandler := server.NewGetKVHandler(consulClient)
	setKVHandler := server.NewSetKVHandler(consulClient)

	mux.HandleFunc("/v1/kv/", func(responseWriter http.ResponseWriter, request *http.Request) {
		fmt.Printf("%+v\n", request)
		switch request.Method {
		case "GET":
			getKVHandler.ServeHTTP(responseWriter, request)
		case "PUT":
			setKVHandler.ServeHTTP(responseWriter, request)
		default:
			http.NotFound(responseWriter, request)
		}
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", port), mux))
}

func parseCommandLineFlags() (string, string) {
	var port string
	var consulURL string

	flag.StringVar(&port, "port", "", "port to use for test consumer server")
	flag.StringVar(&consulURL, "consul-url", "", "url of local consul agent")
	flag.Parse()

	return port, consulURL
}

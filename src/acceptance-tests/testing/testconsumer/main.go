package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/testconsumer/buffered"
	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/testconsumer/handlers"
)

func main() {
	port, consulURL := parseCommandLineFlags()
	proxyURL, err := url.Parse(consulURL)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	logBuffer := bytes.NewBuffer([]byte{})
	healthCheckHandler := handlers.NewHealthCheckHandler()

	proxy := httputil.NewSingleHostReverseProxy(proxyURL)
	director := proxy.Director
	proxy.Director = func(request *http.Request) {
		director(request)
		request.URL.Path = strings.TrimPrefix(request.URL.Path, "/consul")
		request.Host = request.URL.Host
	}
	proxy.ErrorLog = log.New(logBuffer, "", 0)

	mux.HandleFunc("/consul/", func(w http.ResponseWriter, req *http.Request) {
		bufferedRW := buffered.NewResponseWriter(w, logBuffer)
		proxy.ServeHTTP(bufferedRW, req)
		bufferedRW.Copy()
	})

	mux.HandleFunc("/health_check", func(w http.ResponseWriter, req *http.Request) {
		healthCheckHandler.ServeHTTP(w, req)
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

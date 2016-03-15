package main_test

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"time"

	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Setting/Getting keys", func() {
	var (
		session *gexec.Session
		port    string
	)

	BeforeEach(func() {
		consulServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.URL.Path == "/v1/kv/some-key" {
				if req.Method == "GET" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("some-value"))
				}

				if req.Method == "PUT" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("true"))
				}
			}
		}))

		var err error
		port, err = openPort()
		Expect(err).NotTo(HaveOccurred())
		command := exec.Command(pathToConsumer, "--port", port, "--consul-url", consulServer.URL)

		session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		waitForServerToStart(port)
	})

	AfterEach(func() {
		session.Terminate().Wait()
	})

	It("can set/get a key", func() {
		status, responseBody, err := makeRequest("PUT", fmt.Sprintf("http://localhost:%s/v1/kv/some-key", port), "some-value")
		Expect(err).NotTo(HaveOccurred())
		Expect(status).To(Equal(http.StatusOK))

		status, responseBody, err = makeRequest("GET", fmt.Sprintf("http://localhost:%s/v1/kv/some-key", port), "")
		Expect(err).NotTo(HaveOccurred())
		Expect(responseBody).To(Equal("some-value"))
	})

	It("returns 404 for endpoints that are not supported", func() {
		status, _, err := makeRequest("OPTIONS", fmt.Sprintf("http://localhost:%s/v1/kv/some-key", port), "")
		Expect(err).NotTo(HaveOccurred())
		Expect(status).To(Equal(http.StatusNotFound))

		status, _, err = makeRequest("GET", fmt.Sprintf("http://localhost:%s/some/missing/path", port), "")
		Expect(err).NotTo(HaveOccurred())
		Expect(status).To(Equal(http.StatusNotFound))
	})
})

func makeRequest(method string, url string, body string) (int, string, error) {
	request, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return 0, "", err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return 0, "", err
	}

	defer response.Body.Close()
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return 0, "", err
	}

	return response.StatusCode, string(responseBody), nil
}

func openPort() (string, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return "", err
	}

	defer l.Close()
	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return "", err
	}

	return port, nil
}

func waitForServerToStart(port string) {
	timer := time.After(0 * time.Second)
	timeout := time.After(10 * time.Second)
	for {
		select {
		case <-timeout:
			panic("Failed to boot!")
		case <-timer:
			_, err := http.Get("http://localhost:" + port + "/v1/kv/banana")
			if err == nil {
				return
			}

			timer = time.After(1 * time.Second)
		}
	}
}

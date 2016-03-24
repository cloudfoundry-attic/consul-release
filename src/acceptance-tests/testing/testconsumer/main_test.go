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

var _ = Describe("Proxying consul requests", func() {
	var (
		session *gexec.Session
		port    string
	)

	BeforeEach(func() {
		var err error
		port, err = openPort()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		session.Terminate().Wait()
	})

	Context("main", func() {
		It("returns 1 when the consul url is malformed", func() {
			command := exec.Command(pathToConsumer, "--port", port, "--consul-url", "%%%%%%%%%%")

			var err error
			session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err.Contents()).To(ContainSubstring("invalid URL escape"))
		})
	})

	Context("health_check", func() {
		BeforeEach(func() {
			consulServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(http.StatusTeapot)
			}))

			command := exec.Command(pathToConsumer, "--port", port, "--consul-url", consulServer.URL)

			var err error
			session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			waitForServerToStart(port)
		})

		It("returns a 200 when the health check is alive", func() {
			status, _, err := makeRequest("GET", fmt.Sprintf("http://localhost:%s/health_check", port), "")
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(http.StatusOK))
		})

		It("returns a 503 when the health_check has been marked dead", func() {
			status, _, err := makeRequest("POST", fmt.Sprintf("http://localhost:%s/health_check", port), "false")
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(http.StatusOK))

			status, _, err = makeRequest("GET", fmt.Sprintf("http://localhost:%s/health_check", port), "")
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(http.StatusServiceUnavailable))
		})

	})

	Context("with a functioning consul", func() {
		BeforeEach(func() {
			consulServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Path == "/v1/kv/some-key" {
					if req.Method == "GET" {
						w.Write([]byte("some-value"))
						return
					}

					if req.Method == "PUT" {
						w.Write([]byte("true"))
						return
					}
				}

				w.WriteHeader(http.StatusTeapot)
			}))

			command := exec.Command(pathToConsumer, "--port", port, "--consul-url", consulServer.URL)

			var err error
			session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			waitForServerToStart(port)
		})

		It("can set/get a key", func() {
			status, responseBody, err := makeRequest("PUT", fmt.Sprintf("http://localhost:%s/consul/v1/kv/some-key", port), "some-value")
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(http.StatusOK))
			Expect(responseBody).To(Equal("true"))

			status, responseBody, err = makeRequest("GET", fmt.Sprintf("http://localhost:%s/consul/v1/kv/some-key?raw", port), "")
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(http.StatusOK))
			Expect(responseBody).To(Equal("some-value"))
		})

		It("returns 418 for endpoints that are not supported", func() {
			status, _, err := makeRequest("OPTIONS", fmt.Sprintf("http://localhost:%s/consul/v1/kv/some-key", port), "")
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(http.StatusTeapot))

			status, _, err = makeRequest("GET", fmt.Sprintf("http://localhost:%s/consul/some/missing/path", port), "")
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(http.StatusTeapot))
		})
	})

	Context("proxy errors", func() {
		It("returns the underlying proxy error message", func() {
			var err error
			command := exec.Command(pathToConsumer, "--port", port, "--consul-url", "http://localhost:999999")

			session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			waitForServerToStart(port)

			status, responseBody, err := makeRequest("GET", fmt.Sprintf("http://localhost:%s/consul/v1/kv/some-key?raw", port), "")
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(http.StatusInternalServerError))
			Expect(responseBody).To(ContainSubstring("http: proxy error: dial tcp: invalid port 999999"))
		})
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
			_, err := http.Get("http://localhost:" + port + "/consul/v1/kv/banana")
			if err == nil {
				return
			}

			timer = time.After(1 * time.Second)
		}
	}
}

package server_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/testconsumer/server"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	handler      func(http.ResponseWriter, *http.Request)
	consulClient server.ConsulClient
)

var _ = Describe("ConsulClient", func() {
	BeforeEach(func() {
		consulServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			handler(responseWriter, request)
		}))

		consulClient = server.NewConsulClient(consulServer.URL)
	})

	Describe("Set", func() {
		Context("when the consul agent successfully stores the key", func() {
			BeforeEach(func() {
				respondToPut("/v1/kv/some-key", "some-value", http.StatusOK, "true")
			})

			It("does not return an error", func() {
				err := consulClient.Set("some-key", "some-value")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the consul agent fails to store the key", func() {
			BeforeEach(func() {
				respondToPut("/v1/kv/unwritable-key", "unwritable-value", http.StatusOK, "false")
			})

			It("returns an error", func() {
				err := consulClient.Set("unwritable-key", "unwritable-value")
				Expect(err).To(MatchError("failed to store key"))
			})
		})

		Describe("failure cases", func() {
			It("returns an error when the URL cannot be parsed", func() {
				consulClient = server.NewConsulClient("%%%%%")
				err := consulClient.Set("some-key", "some-value")
				Expect(err).To(MatchError(ContainSubstring("invalid URL escape")))
			})

			It("returns an error when the URL scheme is invalid", func() {
				consulClient = server.NewConsulClient("invalid-url")
				err := consulClient.Set("some-key", "some-value")
				Expect(err).To(MatchError(ContainSubstring("unsupported protocol scheme")))
			})

			It("returns an error when consul returns an invalid response", func() {
				respondToPut("/v1/kv/bad-key", "bad-value", http.StatusOK, "banana")
				err := consulClient.Set("bad-key", "bad-value")
				Expect(err).To(MatchError("invalid consul response"))
			})
		})
	})

	Describe("Get", func() {
		Context("when the key is present", func() {
			BeforeEach(func() {
				respondToGet("/v1/kv/some-key", "raw", http.StatusOK, "some-value")
			})

			It("returns the value", func() {
				value, err := consulClient.Get("some-key")
				Expect(err).NotTo(HaveOccurred())
				Expect(value).To(Equal("some-value"))
			})
		})

		Context("when the key is missing", func() {
			BeforeEach(func() {
				respondToGet("/v1/kv/missing-key", "raw", http.StatusNotFound, "missing-value")
			})

			It("returns an error", func() {
				_, err := consulClient.Get("missing-key")
				Expect(err).To(Equal(server.ConsulNotFoundError))
			})
		})

		Describe("failure cases", func() {
			It("returns an error when the request to consul fails", func() {
				consulClient := server.NewConsulClient("http://invalid-url.com")
				_, err := consulClient.Get("some-key")
				Expect(err).To(MatchError(ContainSubstring("no such host")))
			})
		})
	})
})

func respondToGet(path string, queryParameter string, status int, responseBody string) {
	handler = func(responseWriter http.ResponseWriter, request *http.Request) {
		if request.Method == "GET" && request.URL.Path == path && request.URL.RawQuery == queryParameter {
			responseWriter.WriteHeader(status)
			responseWriter.Write([]byte(responseBody))
		}
	}
}

func respondToPut(path string, requestBody string, status int, responseBody string) {
	handler = func(responseWriter http.ResponseWriter, request *http.Request) {
		body, err := ioutil.ReadAll(request.Body)
		Expect(err).NotTo(HaveOccurred())

		if request.Method == "PUT" && request.URL.Path == path && string(body) == requestBody {
			responseWriter.WriteHeader(status)
			responseWriter.Write([]byte(responseBody))
		}
	}
}

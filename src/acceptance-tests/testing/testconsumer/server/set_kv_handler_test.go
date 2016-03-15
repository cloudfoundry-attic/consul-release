package server_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/testconsumer/server"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SetKVHandler", func() {
	var (
		consulClient *FakeConsulClient
		recorder     *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		consulClient = &FakeConsulClient{}
		recorder = httptest.NewRecorder()
	})

	Context("when the update succeeds", func() {
		It("proxies the request to the consul agent, returning a 200", func() {
			request, err := http.NewRequest("PUT", "/v1/kv/some-key", strings.NewReader("some-value"))
			Expect(err).NotTo(HaveOccurred())

			handler := server.NewSetKVHandler(consulClient)
			handler.ServeHTTP(recorder, request)

			Expect(recorder.Code).To(Equal(http.StatusOK))
			Expect(consulClient.SetCall.Receives.Key).To(Equal("some-key"))
			Expect(consulClient.SetCall.Receives.Value).To(Equal("some-value"))
		})
	})

	Context("when the update fails", func() {
		It("proxies the request to the consul agent, returning a 500", func() {
			consulClient.SetCall.Returns.Error = errors.New("failed to set key")

			request, err := http.NewRequest("PUT", "/v1/kv/unwritable-key", strings.NewReader("unwritable-value"))
			Expect(err).NotTo(HaveOccurred())

			handler := server.NewSetKVHandler(consulClient)
			handler.ServeHTTP(recorder, request)

			Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
			Expect(consulClient.SetCall.Receives.Key).To(Equal("unwritable-key"))
			Expect(consulClient.SetCall.Receives.Value).To(Equal("unwritable-value"))
		})
	})

	Describe("failure cases", func() {
		It("returns a 500 when the body cannot be read", func() {
			request, err := http.NewRequest("PUT", "/v1/kv/some-key", badReader{})
			Expect(err).NotTo(HaveOccurred())

			handler := server.NewSetKVHandler(consulClient)
			handler.ServeHTTP(recorder, request)

			Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
		})
	})
})

type badReader struct{}

func (badReader) Read([]byte) (int, error) {
	return 0, errors.New("failed to read")
}

package server_test

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/cloudfoundry-incubator/consul-release/src/acceptance-tests/testing/testconsumer/server"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetKVHandler", func() {
	var (
		consulClient *FakeConsulClient
		recorder     *httptest.ResponseRecorder
		request      *http.Request
	)

	BeforeEach(func() {
		var err error
		request, err = http.NewRequest("GET", "/v1/kv/some-key", nil)
		Expect(err).NotTo(HaveOccurred())
		recorder = httptest.NewRecorder()
		consulClient = &FakeConsulClient{}
	})

	Context("when the key exists", func() {
		It("returns the value for that key", func() {
			consulClient.GetCall.Returns.Value = "some-value"

			handler := server.NewGetKVHandler(consulClient)
			handler.ServeHTTP(recorder, request)

			Expect(recorder.Code).To(Equal(http.StatusOK))
			Expect(recorder.Body.String()).To(Equal("some-value"))
			Expect(consulClient.GetCall.Receives.Key).To(Equal("some-key"))
		})
	})

	Context("when the key does not exist", func() {
		It("returns a 404", func() {
			consulClient.GetCall.Returns.Error = server.ConsulNotFoundError

			request, err := http.NewRequest("GET", "/v1/kv/missing-key", nil)
			Expect(err).NotTo(HaveOccurred())

			handler := server.NewGetKVHandler(consulClient)
			handler.ServeHTTP(recorder, request)

			Expect(recorder.Code).To(Equal(http.StatusNotFound))
			Expect(consulClient.GetCall.Receives.Key).To(Equal("missing-key"))
		})
	})

	Describe("failure cases", func() {
		It("returns a 500 when the GET request to consul fails", func() {
			consulClient.GetCall.Returns.Error = errors.New("failed to get key")

			handler := server.NewGetKVHandler(consulClient)
			handler.ServeHTTP(recorder, request)

			Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
		})

		It("returns a 500 when the writer cannot be written to", func() {
			consulClient.GetCall.Returns.Value = "some-value"

			badRecorder := &BadRecorder{}
			handler := server.NewGetKVHandler(consulClient)
			handler.ServeHTTP(badRecorder, request)

			Expect(badRecorder.Code).To(Equal(http.StatusInternalServerError))
		})
	})
})

type BadRecorder struct {
	Code int
}

func (r BadRecorder) Header() http.Header {
	return http.Header{}
}

func (r *BadRecorder) WriteHeader(code int) {
	r.Code = code
}

func (r BadRecorder) Write([]byte) (int, error) {
	return 0, errors.New("failed to write")
}

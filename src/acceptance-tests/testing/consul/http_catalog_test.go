package consul_test

import (
	"net/http"
	"net/http/httptest"

	"acceptance-tests/testing/consul"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTTPCatalog", func() {
	Context("Nodes", func() {
		It("retrieves the list of known nodes", func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.URL.Path).To(Equal("/v1/catalog/nodes"))
				Expect(r.Method).To(Equal("GET"))

				w.Write([]byte(`[
					{"Node": "some-node-name", "Address": "10.0.0.1"},
					{"Node": "some-other-node-name", "Address": "10.0.0.2"}
				]`))
			}))

			catalog := consul.NewHTTPCatalog(server.URL)

			nodes, err := catalog.Nodes()
			Expect(err).NotTo(HaveOccurred())
			Expect(nodes).To(ConsistOf([]consul.Node{
				{Node: "some-node-name", Address: "10.0.0.1"},
				{Node: "some-other-node-name", Address: "10.0.0.2"},
			}))
		})

		Context("failure cases", func() {
			It("errors on a non 200 http status code", func() {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					Expect(r.URL.Path).To(Equal("/v1/catalog/nodes"))
					w.WriteHeader(http.StatusNotFound)
				}))

				catalog := consul.NewHTTPCatalog(server.URL)

				_, err := catalog.Nodes()
				Expect(err).To(MatchError("consul http error: 404 Not Found"))
			})

			It("errors on malformed JSON", func() {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					Expect(r.URL.Path).To(Equal("/v1/catalog/nodes"))
					Expect(r.Method).To(Equal("GET"))

					w.Write([]byte(`%%%%%%%%`))
				}))

				catalog := consul.NewHTTPCatalog(server.URL)

				_, err := catalog.Nodes()
				Expect(err).To(MatchError(ContainSubstring("invalid character")))
			})

			It("errors on an unsupported protocol", func() {
				catalog := consul.NewHTTPCatalog("banana://example.com")

				_, err := catalog.Nodes()
				Expect(err).To(MatchError(ContainSubstring("unsupported protocol")))
			})
		})
	})
})

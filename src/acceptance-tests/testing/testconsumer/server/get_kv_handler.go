package server

import (
	"net/http"
	"strings"
)

type consulKeyGetter interface {
	Get(key string) (value string, err error)
}

type GetKVHandler struct {
	consulClient consulKeyGetter
}

func NewGetKVHandler(consulClient consulKeyGetter) GetKVHandler {
	return GetKVHandler{
		consulClient: consulClient,
	}
}

func (g GetKVHandler) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	key := strings.TrimPrefix(request.URL.Path, "/v1/kv/")
	value, err := g.consulClient.Get(key)
	if err != nil {
		if err == ConsulNotFoundError {
			responseWriter.WriteHeader(http.StatusNotFound)
			return
		}

		responseWriter.WriteHeader(http.StatusInternalServerError)
		return
	}

	responseWriter.WriteHeader(http.StatusOK)
	_, err = responseWriter.Write([]byte(value))
	if err != nil {
		responseWriter.WriteHeader(http.StatusInternalServerError)
		return
	}
}

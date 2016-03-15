package server

import (
	"io/ioutil"
	"net/http"
	"strings"
)

type consulKeySetter interface {
	Set(key, value string) error
}

type SetKVHandler struct {
	consulClient consulKeySetter
}

func NewSetKVHandler(consulClient consulKeySetter) SetKVHandler {
	return SetKVHandler{
		consulClient: consulClient,
	}
}

func (h SetKVHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	key := strings.TrimPrefix(req.URL.Path, "/v1/kv/")
	value, err := ioutil.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = h.consulClient.Set(key, string(value))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

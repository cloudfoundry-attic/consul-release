package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
)

type DNSHandler struct {
	pathToCheckARecord string
}

func NewDNSHandler(pathToCheckARecord string) DNSHandler {
	return DNSHandler{
		pathToCheckARecord: pathToCheckARecord,
	}
}

func (d DNSHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	serviceName := request.URL.Query().Get("service")

	if serviceName == "" {
		response.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(response, "service is a required parameter")
		return
	}

	var addresses []string
	command := exec.Command(d.pathToCheckARecord, serviceName)

	stdout, err := command.Output()
	switch err.(type) {
	case nil:
		addresses = strings.Split(strings.TrimSpace(string(stdout)), "\n")
	case *exec.ExitError:
		addresses = []string{}
	default:
		response.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(response, err.Error())
		return
	}

	buf, err := json.Marshal(addresses)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(response, err.Error())
		return
	}

	fmt.Fprint(response, string(buf))
}

package consul

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Node struct {
	Node    string
	Address string
}

type HTTPCatalog struct {
	ConsulAddress string
}

func NewHTTPCatalog(consulAddress string) HTTPCatalog {
	return HTTPCatalog{
		ConsulAddress: consulAddress,
	}
}

func (c HTTPCatalog) Nodes() ([]Node, error) {
	nodes := []Node{}

	response, err := http.Get(fmt.Sprintf("%s/v1/catalog/nodes", c.ConsulAddress))
	if err != nil {
		return nodes, err
	}

	if response.StatusCode != http.StatusOK {
		return nodes, fmt.Errorf("consul http error: %d %s", response.StatusCode, http.StatusText(response.StatusCode))
	}

	err = json.NewDecoder(response.Body).Decode(&nodes)
	if err != nil {
		return nodes, err
	}

	return nodes, nil
}

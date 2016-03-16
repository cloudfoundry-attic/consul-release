package consul

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type ConsulClient struct {
	ConsulAddress string
}

func NewConsulClient(consulAddress string) ConsulClient {
	return ConsulClient{
		ConsulAddress: consulAddress,
	}
}

func (c ConsulClient) Set(key, value string) error {
	request, err := http.NewRequest("PUT", fmt.Sprintf("%s/v1/kv/%s", c.ConsulAddress, key), strings.NewReader(value))
	if err != nil {
		return err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return errors.New("failed to save to kv store")
	}

	return nil
}

func (c ConsulClient) Get(key string) (string, error) {
	response, err := http.Get(fmt.Sprintf("%s/v1/kv/%s", c.ConsulAddress, key))
	if err != nil {
		return "", err
	}

	if response.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("key %q not found", key)
	}

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("consul http error: %d %s", response.StatusCode, http.StatusText(response.StatusCode))
	}

	body, err := bodyReader(response.Body)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	return string(body), nil
}

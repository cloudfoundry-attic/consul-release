package server

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

var ConsulNotFoundError = errors.New("key not found")

type ConsulClient struct {
	URL string
}

func NewConsulClient(consulURL string) ConsulClient {
	return ConsulClient{consulURL}
}

func (c ConsulClient) Set(key, value string) error {
	request, err := http.NewRequest("PUT", fmt.Sprintf("%s/v1/kv/%s", c.URL, key), strings.NewReader(value))
	if err != nil {
		return err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}

	defer response.Body.Close()
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	switch string(responseBody) {
	case "true":
		return nil
	case "false":
		return errors.New("failed to store key")
	default:
		return errors.New("invalid consul response")
	}
}

func (c ConsulClient) Get(key string) (string, error) {
	response, err := http.Get(fmt.Sprintf("%s/v1/kv/%s?raw", c.URL, key))
	if err != nil {
		return "", err
	}

	if response.StatusCode == http.StatusNotFound {
		return "", ConsulNotFoundError
	}

	defer response.Body.Close()
	value, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(value), nil
}

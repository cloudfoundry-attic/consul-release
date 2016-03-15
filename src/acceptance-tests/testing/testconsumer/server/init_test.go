package server_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "server")
}

type FakeConsulClient struct {
	SetCall struct {
		Receives struct {
			Key   string
			Value string
		}
		Returns struct {
			Error error
		}
	}

	GetCall struct {
		Receives struct {
			Key string
		}
		Returns struct {
			Value string
			Error error
		}
	}
}

func (f *FakeConsulClient) Set(key, value string) error {
	f.SetCall.Receives.Key = key
	f.SetCall.Receives.Value = value

	return f.SetCall.Returns.Error
}

func (f *FakeConsulClient) Get(key string) (string, error) {
	f.GetCall.Receives.Key = key

	return f.GetCall.Returns.Value, f.GetCall.Returns.Error
}

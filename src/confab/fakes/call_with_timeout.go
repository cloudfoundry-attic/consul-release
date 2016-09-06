package fakes

import "github.com/cloudfoundry/cf-release/src/confab"

type CallWithTimeout struct {
	TryCall struct {
		CallCount int
		Returns   struct {
			Error error
		}
		Recieves struct {
			Timeout confab.Timeout
			F       func() error
		}
	}
}

func (c CallWithTimeout) Try(timeout confab.Timeout, f func() error) error {
	c.TryCall.CallCount++
	c.TryCall.Recieves.Timeout = timeout
	c.TryCall.Recieves.F = f
	return c.TryCall.Returns.Error
}

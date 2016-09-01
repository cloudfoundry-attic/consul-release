package confab

import (
	"errors"
	"time"
)

type Clock interface {
	Sleep(time.Duration)
}

type CallWithTimeout struct {
	Clock      Clock
	RetryDelay time.Duration
}

func NewCallWithTimeout(clock Clock, retryDelay time.Duration) CallWithTimeout {
	return CallWithTimeout{
		Clock:      clock,
		RetryDelay: retryDelay,
	}
}

func (c CallWithTimeout) Try(timeout Timeout, f func() error) error {
	for {
		select {
		case <-timeout.Done():
			return errors.New("timeout exceeded")
		default:
			err := f()
			if err != nil {
				c.Clock.Sleep(c.RetryDelay)
				continue
			}
			return nil
		}
	}
}

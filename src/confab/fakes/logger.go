package fakes

import (
	"sync"

	"github.com/pivotal-golang/lager"
)

type LoggerMessage struct {
	Action string
	Error  error
	Data   []lager.Data
}

type Logger struct {
	sync.Mutex
	Messages []LoggerMessage
}

func (l *Logger) Info(action string, data ...lager.Data) {
	l.Lock()
	defer l.Unlock()

	l.Messages = append(l.Messages, LoggerMessage{
		Action: action,
		Data:   data,
	})
}

func (l *Logger) Error(action string, err error, data ...lager.Data) {
	l.Lock()
	defer l.Unlock()

	l.Messages = append(l.Messages, LoggerMessage{
		Action: action,
		Error:  err,
		Data:   data,
	})
}

package chaperon

import (
	"io"
	"io/ioutil"
	"time"
)

func SetAgentCheckInterval(interval time.Duration) {
	agentCheckInterval = interval
}

func ResetAgentCheckInterval() {
	agentCheckInterval = time.Second
}

func SetReadAll(readAll func(reader io.Reader) ([]byte, error)) {
	ioutilReadAll = readAll
}

func ResetReadAll() {
	ioutilReadAll = ioutil.ReadAll
}

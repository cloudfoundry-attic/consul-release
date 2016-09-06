package chaperon

import (
	"io"
	"io/ioutil"
)

func SetReadAll(readAll func(reader io.Reader) ([]byte, error)) {
	ioutilReadAll = readAll
}

func ResetReadAll() {
	ioutilReadAll = ioutil.ReadAll
}

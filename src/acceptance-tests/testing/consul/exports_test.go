package consul

import (
	"io"
	"io/ioutil"
	"os"
)

func SetBodyReader(r func(io.Reader) ([]byte, error)) {
	bodyReader = r
}

func ResetBodyReader() {
	bodyReader = ioutil.ReadAll
}

func SetCreateFile(f func(string) (*os.File, error)) {
	createFile = f
}

func ResetCreateFile() {
	createFile = os.Create
}

package chaperon

import (
	"io/ioutil"
	"os"
)

func SetTempDir(tempDirFunc func(string, string) (string, error)) {
	tempDir = tempDirFunc
}

func ResetTempDir() {
	tempDir = ioutil.TempDir
}

func SetRemoveAll(removeAllFunc func(string) error) {
	removeAll = removeAllFunc
}

func ResetRemoveAll() {
	removeAll = os.RemoveAll
}

package confab

import "os"

func SetCreateFile(f func(string) (*os.File, error)) {
	createFile = f
}

func ResetCreateFile() {
	createFile = os.Create
}

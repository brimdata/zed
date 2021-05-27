// +build !windows

package fs

import (
	"os"
)

func OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

func Open(name string) (*os.File, error) {
	return OpenFile(name, os.O_RDONLY, 0)
}

func Create(name string) (*os.File, error) {
	return OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

func ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

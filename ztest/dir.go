package ztest

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

type Dir struct {
	path string
}

func NewDir(name, parent string) (*Dir, error) {
	path, err := ioutil.TempDir(parent, name)
	if err != nil {
		return nil, err
	}
	return &Dir{
		path: path,
	}, nil
}

func (d *Dir) RemoveAll() {
	os.RemoveAll(d.path)
}

func (d *Dir) Path() string {
	return d.path
}

func (d *Dir) Join(name string) string {
	return filepath.Join(d.path, name)
}

func (d *Dir) Write(name string, data []byte) error {
	return ioutil.WriteFile(d.Join(name), data, 0644)
}

func (d *Dir) Read(name string) ([]byte, error) {
	return ioutil.ReadFile(d.Join(name))
}

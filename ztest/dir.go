package ztest

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

// syntactic sugar for directory manipulations of a temp dir
type Dir string

func NewDir(name string) (*Dir, error) {
	path, err := ioutil.TempDir("", name)
	if err != nil {
		return nil, err
	}
	d := Dir(path)
	return &d, nil
}

func (d Dir) RemoveAll() {
	os.RemoveAll(string(d))
}

func (d Dir) Path() string {
	return string(d)
}

func (d Dir) Join(name string) string {
	return filepath.Join(string(d), name)
}

func (d Dir) Write(name string, data []byte) error {
	return ioutil.WriteFile(d.Join(name), data, 0644)
}

func (d Dir) Symlink(oldname, newname string) error {
	return os.Symlink(oldname, d.Join(newname))
}

func (d Dir) Read(name string) ([]byte, error) {
	return ioutil.ReadFile(d.Join(name))
}

package iosource

import (
	"io"
	"net/url"
	"os"

	"github.com/brimsec/zq/pkg/fs"
)

var DefaultFileSource = &FileSource{Perm: 0666}

type FileSource struct {
	Perm os.FileMode
}

func (f *FileSource) NewReader(path string) (io.ReadCloser, error) {
	return fs.Open(filePath(path))
}

func (f *FileSource) NewWriter(path string) (io.WriteCloser, error) {
	path = filePath(path)
	return fs.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, f.Perm)
}

func filePath(path string) string {
	if u, _ := url.Parse(path); u != nil {
		return u.Path
	}
	return path
}

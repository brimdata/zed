package iosource

import (
	"io"
	"net/url"
	"os"
)

var DefaultFileSource = &FileSource{Perm: 0666}

type FileSource struct {
	Perm os.FileMode
}

func (f *FileSource) NewReader(path string) (io.ReadCloser, error) {
	return os.Open(filePath(path))
}

func (f *FileSource) NewWriter(path string) (io.WriteCloser, error) {
	path = filePath(path)
	return os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, f.Perm)
}

func filePath(path string) string {
	u, _ := url.Parse(path)
	if u != nil {
		return u.Path
	}
	return path
}

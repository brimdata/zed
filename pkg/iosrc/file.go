package iosrc

import (
	"io"
	"io/ioutil"
	"os"

	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/zqe"
)

var DefaultFileSource = &FileSource{Perm: 0666}
var _ DirMaker = DefaultFileSource

type FileSource struct {
	Perm os.FileMode
}

func (f *FileSource) NewReader(uri URI) (Reader, error) {
	r, err := fs.Open(uri.Filepath())
	return r, wrapfileError(uri, err)
}

func (s *FileSource) NewWriter(uri URI) (io.WriteCloser, error) {
	w, err := fs.OpenFile(uri.Filepath(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, s.Perm)
	return w, wrapfileError(uri, err)
}

func (s *FileSource) ReadFile(uri URI) ([]byte, error) {
	d, err := ioutil.ReadFile(uri.Filepath())
	return d, wrapfileError(uri, err)
}

func (s *FileSource) WriteFile(d []byte, uri URI) error {
	err := ioutil.WriteFile(uri.Filepath(), d, s.Perm)
	return wrapfileError(uri, err)
}

func (s *FileSource) MkdirAll(uri URI, perm os.FileMode) error {
	return wrapfileError(uri, os.MkdirAll(uri.Filepath(), perm))
}

func (s *FileSource) Remove(uri URI) error {
	return wrapfileError(uri, os.Remove(uri.Filepath()))
}

func (s *FileSource) RemoveAll(uri URI) error {
	return os.RemoveAll(uri.Filepath())
}

func (s *FileSource) Stat(uri URI) (Info, error) {
	info, err := os.Stat(uri.Filepath())
	if err != nil {
		return nil, wrapfileError(uri, err)
	}
	return info, nil
}

func (s *FileSource) Exists(uri URI) (bool, error) {
	_, err := os.Stat(uri.Filepath())
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, wrapfileError(uri, err)
	}
	return true, nil
}

func (s *FileSource) NewReplacer(uri URI) (io.WriteCloser, error) {
	return fs.NewFileReplacer(uri.Filepath(), s.Perm)
}

func wrapfileError(uri URI, err error) error {
	if os.IsNotExist(err) {
		return zqe.E(zqe.NotFound, uri.String())
	}
	return err
}

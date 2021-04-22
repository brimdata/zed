package iosrc

import (
	"context"
	"io"
	"io/ioutil"
	"os"

	"github.com/brimdata/zed/pkg/fs"
	"github.com/brimdata/zed/zqe"
)

var DefaultFileSource = &FileSource{Perm: 0666}
var _ DirMaker = DefaultFileSource
var _ ReplacerAble = DefaultFileSource

type FileSource struct {
	Perm os.FileMode
}

func (f *FileSource) NewReader(_ context.Context, uri URI) (Reader, error) {
	r, err := fs.Open(uri.Filepath())
	return r, wrapfileError(uri, err)
}

func (s *FileSource) NewWriter(_ context.Context, uri URI) (io.WriteCloser, error) {
	w, err := fs.OpenFile(uri.Filepath(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, s.Perm)
	return w, wrapfileError(uri, err)
}

func (s *FileSource) ReadFile(_ context.Context, uri URI) ([]byte, error) {
	d, err := ioutil.ReadFile(uri.Filepath())
	return d, wrapfileError(uri, err)
}

func (s *FileSource) WriteFile(_ context.Context, d []byte, uri URI) error {
	err := ioutil.WriteFile(uri.Filepath(), d, s.Perm)
	return wrapfileError(uri, err)
}

func (s *FileSource) WriteFileIfNotExists(_ context.Context, b []byte, uri URI) error {
	f, err := os.OpenFile(uri.Filepath(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC|os.O_EXCL, s.Perm)
	if err != nil {
		return wrapfileError(uri, err)
	}
	_, err = f.Write(b)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	return wrapfileError(uri, err)
}

func (s *FileSource) MkdirAll(uri URI, perm os.FileMode) error {
	return wrapfileError(uri, os.MkdirAll(uri.Filepath(), perm))
}

func (s *FileSource) Remove(_ context.Context, uri URI) error {
	return wrapfileError(uri, os.Remove(uri.Filepath()))
}

func (s *FileSource) RemoveAll(_ context.Context, uri URI) error {
	return os.RemoveAll(uri.Filepath())
}

func (s *FileSource) Stat(_ context.Context, uri URI) (Info, error) {
	info, err := os.Stat(uri.Filepath())
	if err != nil {
		return nil, wrapfileError(uri, err)
	}
	return info, nil
}

func (s *FileSource) Exists(_ context.Context, uri URI) (bool, error) {
	_, err := os.Stat(uri.Filepath())
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, wrapfileError(uri, err)
	}
	return true, nil
}

func (s *FileSource) NewReplacer(_ context.Context, uri URI) (Replacer, error) {
	return fs.NewFileReplacer(uri.Filepath(), s.Perm)
}

func (s *FileSource) ReadDir(_ context.Context, uri URI) ([]Info, error) {
	entries, err := ioutil.ReadDir(uri.Filepath())
	if err != nil {
		return nil, wrapfileError(uri, err)
	}
	infos := make([]Info, len(entries))
	for i, e := range entries {
		infos[i] = e
	}
	return infos, nil
}

func wrapfileError(uri URI, err error) error {
	if os.IsNotExist(err) {
		return zqe.E(zqe.NotFound, uri.String())
	}
	return err
}

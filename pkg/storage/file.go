package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/brimdata/zed/pkg/fs"
	"github.com/brimdata/zed/zqe"
)

type FileSystem struct {
	perm   os.FileMode
	exists map[string]struct{}
}

var _ Engine = (*FileSystem)(nil)

func NewFileSystem() *FileSystem {
	return &FileSystem{
		perm:   0666,
		exists: make(map[string]struct{}),
	}
}

func (f *FileSystem) Get(ctx context.Context, u *URI) (Reader, error) {
	r, err := fs.Open(u.Filepath())
	return &fileSizer{r, u}, wrapfileError(u, err)
}

func (f *FileSystem) Put(_ context.Context, u *URI) (io.WriteCloser, error) {
	path := u.Filepath()
	if err := f.checkPath(path); err != nil {
		return nil, wrapfileError(u, err)
	}
	w, err := fs.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, f.perm)
	return w, wrapfileError(u, err)
}

func (f *FileSystem) PutIfNotExists(_ context.Context, u *URI, b []byte) error {
	path := u.Filepath()
	if err := f.checkPath(path); err != nil {
		return wrapfileError(u, err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|os.O_EXCL, f.perm)
	if err != nil {
		return wrapfileError(u, err)
	}
	_, err = io.Copy(file, bytes.NewReader(b))
	if err != nil {
		file.Close()
		f.Delete(nil, u)
		return err
	}
	return file.Close()
}

func (f *FileSystem) Delete(_ context.Context, u *URI) error {
	return wrapfileError(u, os.Remove(u.Filepath()))
}

func (f *FileSystem) DeleteByPrefix(_ context.Context, u *URI) error {
	return os.RemoveAll(u.Filepath())
}

func (f *FileSystem) Size(_ context.Context, u *URI) (int64, error) {
	info, err := os.Stat(u.Filepath())
	if err != nil {
		return 0, wrapfileError(u, err)
	}
	return info.Size(), nil
}

func (f *FileSystem) Exists(_ context.Context, u *URI) (bool, error) {
	_, err := os.Stat(u.Filepath())
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, wrapfileError(u, err)
	}
	return true, nil
}

func (f *FileSystem) List(ctx context.Context, u *URI) ([]Info, error) {
	entries, err := os.ReadDir(u.Filepath())
	if err != nil {
		return nil, wrapfileError(u, err)
	}
	infos := make([]Info, len(entries))
	for i, e := range entries {
		info, err := e.Info()
		if err != nil {
			return nil, err
		}
		infos[i] = Info{
			Name: e.Name(),
			Size: info.Size(),
		}
	}
	return infos, nil
}

func (f *FileSystem) checkPath(path string) error {
	dir := filepath.Dir(path)
	if dir == "." {
		return nil
	}
	if _, ok := f.exists[dir]; ok {
		return nil
	}
	err := os.MkdirAll(dir, 0755)
	if os.IsExist(err) {
		err = nil
	}
	if err == nil {
		f.exists[dir] = struct{}{}
	}
	return err
}

func wrapfileError(uri *URI, err error) error {
	if os.IsNotExist(err) {
		return zqe.E(zqe.NotFound, uri.String())
	}
	return err
}

type fileSizer struct {
	*os.File
	uri *URI
}

var _ Sizer = (*fileSizer)(nil)

func (f *fileSizer) Size() (int64, error) {
	info, err := os.Stat(f.uri.Filepath())
	if err != nil {
		return 0, wrapfileError(f.uri, err)
	}
	return info.Size(), nil
}

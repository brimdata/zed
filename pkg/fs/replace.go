package fs

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// NewFileReplacer returns an io.WriteCloser to a temporary file; on close,
// the temp file will be atomically renamed to the given filename.
func NewFileReplacer(filename string, perm os.FileMode) (io.WriteCloser, error) {
	filename, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}
	f, err := ioutil.TempFile(filepath.Dir(filename), ".tmp-"+filepath.Base(filename))
	if err != nil {
		return nil, err
	}
	return &replacer{
		f:        f,
		filename: filename,
		perm:     perm,
	}, nil
}

type replacer struct {
	f        *os.File
	writeErr error
	filename string
	perm     os.FileMode
}

func (r *replacer) Write(b []byte) (int, error) {
	n, err := r.f.Write(b)
	if err != nil {
		r.writeErr = err
	}
	return n, err
}

func (r *replacer) Close() (err error) {
	defer func() {
		if err != nil || r.writeErr != nil {
			os.Remove(r.f.Name())
		}
	}()
	if err := r.f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(r.f.Name(), r.perm); err != nil {
		return err
	}
	if r.writeErr == nil {
		return os.Rename(r.f.Name(), r.filename)
	}
	return nil
}

func ReplaceFile(name string, perm os.FileMode, fn func(w io.Writer) error) (err error) {
	wc, err := NewFileReplacer(name, perm)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := wc.Close()
		if err == nil {
			err = closeErr
		}
	}()
	return fn(wc)
}

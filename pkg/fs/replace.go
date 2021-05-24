package fs

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

type Replacer struct {
	f        *os.File
	err      error
	filename string
	perm     os.FileMode
}

// NewFileReplacer returns a Replacer, an io.WriteCloser that can be used
// to atomically update the content of a file. Either Close or Abort must be
// called on the Replacer; Close will rename the temp file to the given
// filename, while Abort will leave the original file unmodified.
func NewFileReplacer(filename string, perm os.FileMode) (*Replacer, error) {
	filename, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}
	f, err := os.CreateTemp(filepath.Dir(filename), ".tmp-"+filepath.Base(filename))
	if err != nil {
		return nil, err
	}
	return &Replacer{
		f:        f,
		filename: filename,
		perm:     perm,
	}, nil
}

func (r *Replacer) Write(b []byte) (int, error) {
	n, err := r.f.Write(b)
	if err != nil {
		r.err = err
	}
	return n, err
}

func (r *Replacer) Abort() {
	r.err = errors.New("replacer aborted")
	_ = r.close()
}

func (r *Replacer) Close() error {
	return r.close()
}

func (r *Replacer) close() (err error) {
	defer func() {
		if err != nil || r.err != nil {
			os.Remove(r.f.Name())
		}
	}()
	if err := r.f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(r.f.Name(), r.perm); err != nil {
		return err
	}
	if r.err == nil {
		return os.Rename(r.f.Name(), r.filename)
	}
	return r.err
}

func ReplaceFile(name string, perm os.FileMode, fn func(w io.Writer) error) error {
	r, err := NewFileReplacer(name, perm)
	if err != nil {
		return err
	}
	if err := fn(r); err != nil {
		r.Abort()
		return err
	}
	return r.Close()
}

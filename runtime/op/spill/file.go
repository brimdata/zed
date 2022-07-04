package spill

import (
	"bufio"
	"os"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/bufwriter"
	"github.com/brimdata/zed/pkg/fs"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
)

// File provides a means to write a sequence of zng records to temporary
// storage then read them back.  This is used for processing large batches of
// data that do not fit in memory and/or cannot be shuffled to a peer worker,
// but can be processed in multiple passes.  File implements zio.Reader and
// zio.Writer.
type File struct {
	*zngio.Reader
	*zngio.Writer
	file *os.File
}

// NewFile returns a File.  Records should be written to File via the zio.Writer
// interface, followed by a call to the Rewind method, followed by reading
// records via the zio.Reader interface.
func NewFile(f *os.File) *File {
	return &File{
		Writer: zngio.NewWriterWithOpts(bufwriter.New(zio.NopCloser(f)), zngio.WriterOpts{}),
		file:   f,
	}
}

func NewTempFile() (*File, error) {
	f, err := TempFile()
	if err != nil {
		return nil, err
	}
	return NewFile(f), nil
}

func NewFileWithPath(path string) (*File, error) {
	f, err := fs.Create(path)
	if err != nil {
		return nil, err
	}
	return NewFile(f), nil
}

func (f *File) Rewind(zctx *zed.Context) error {
	// Close the writer to flush any pending output but since we
	// wrapped the file in a zio.NopCloser, the file will stay open.
	if err := f.Writer.Close(); err != nil {
		return err
	}
	f.Writer = nil
	if _, err := f.file.Seek(0, 0); err != nil {
		return err
	}
	if f.Reader != nil {
		f.Reader.Close()
	}
	f.Reader = zngio.NewReader(zctx, bufio.NewReader(f.file))
	return nil
}

// CloseAndRemove closes and removes the underlying file.
func (r *File) CloseAndRemove() error {
	if r.Reader != nil {
		r.Reader.Close()
	}
	err := r.file.Close()
	if rmErr := os.Remove(r.file.Name()); err == nil {
		err = rmErr
	}
	return err
}

func (f *File) Size() (int64, error) {
	info, err := f.file.Stat()
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

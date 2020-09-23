package spill

import (
	"bufio"
	"os"

	"github.com/brimsec/zq/pkg/bufwriter"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng/resolver"
)

// File provides a means to write a sequence of zng records to tempoarary
// storage then read them back.  This is used for processing large batches of
// data that do not fit in memory and/or cannot be shuffled to a peer worker,
// but can be processed in multiple passes.  File implements zbuf.Reader and
// zbuf.Writer.
type File struct {
	*zngio.Reader
	*zngio.Writer
	file *os.File
}

// NewFile returns a File.  Records should be written to File via the zbuf.Writer
// interface, followed by a call to the Rewind method, followed by reading
// records via the zbuf.Reader interface.
func NewFile(f *os.File) (*File, error) {
	return &File{
		Writer: zngio.NewWriter(bufwriter.New(zio.NopCloser(f)), zngio.WriterOpts{}),
		file:   f,
	}, nil
}

func NewTempFile() (*File, error) {
	f, err := TempFile()
	if err != nil {
		return nil, err
	}
	return &File{
		Writer: zngio.NewWriter(bufwriter.New(zio.NopCloser(f)), zngio.WriterOpts{}),
		file:   f,
	}, nil
}

func NewFileWithPath(path string, zctx *resolver.Context) (*File, error) {
	f, err := fs.Create(path)
	if err != nil {
		return nil, err
	}
	return NewFile(f)
}

func (f *File) Rewind(zctx *resolver.Context) error {
	// Close the writer to flush any pending output but since we
	// wrapped the file in a zio.NopCloser, the file will stay open.
	if err := f.Writer.Close(); err != nil {
		return err
	}
	f.Writer = nil
	if _, err := f.file.Seek(0, 0); err != nil {
		f.closeAndRemove()
		return err
	}
	f.Reader = zngio.NewReader(bufio.NewReader(f.file), zctx)
	return nil
}

// closeAndRemove closes and removes the underlying file.
// XXX errors are ignored.
func (r *File) closeAndRemove() {
	r.file.Close()
	os.Remove(r.file.Name())
}

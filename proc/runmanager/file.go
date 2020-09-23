package runmanager

import (
	"bufio"
	"os"

	"github.com/brimsec/zq/pkg/bufwriter"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng/resolver"
)

// Run provides a means to write a sequence of zng records to tempoarary
// storage then read them back.  This is used for processing large batches of
// data that do not fit in memory and/or cannot be shuffled to a peer worker,
// but can be processed in multiple passes.  File implements zbuf.Reader and
// zbuf.Writer.
type Run struct {
	*zngio.Reader
	*zngio.Writer
	file *os.File
}

// NewRun returns a File.  Records should be written to File via the zbuf.Writer
// interface, followed by a call to the Rewind method, followed by reading
// records via the zbuf.Reader interface.
func NewRun(f *os.File) (*Run, error) {
	return &Run{
		Writer: zngio.NewWriter(bufwriter.New(zio.NopCloser(f)), zngio.WriterOpts{}),
		file:   f,
	}, nil
}

func NewTempRun() (*Run, error) {
	f, err := TempFile()
	if err != nil {
		return nil, err
	}
	return &Run{
		Writer: zngio.NewWriter(bufwriter.New(zio.NopCloser(f)), zngio.WriterOpts{}),
		file:   f,
	}, nil
}

func NewRunWithPath(path string, zctx *resolver.Context) (*Run, error) {
	f, err := fs.Create(path)
	if err != nil {
		return nil, err
	}
	return NewRun(f)
}

func (f *Run) Rewind(zctx *resolver.Context) error {
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
func (r *Run) closeAndRemove() {
	r.file.Close()
	os.Remove(r.file.Name())
}

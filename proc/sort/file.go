package sort

import (
	"bufio"
	"io"
	"os"

	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

// runFile represents a run as as a readable file containing a sorted sequence
// of records.
type runFile struct {
	file       *os.File
	nextRecord *zng.Record
	zr         zbuf.Reader
}

// newRunFile writes sorted to filename and returns a runFile that reads the
// file using zctx.
func newRunFile(filename string, sorted []*zng.Record, zctx *resolver.Context) (*runFile, error) {
	f, err := fs.Create(filename)
	if err != nil {
		return nil, err
	}
	r := &runFile{file: f}
	if err := writeZng(f, sorted); err != nil {
		r.closeAndRemove()
		return nil, err
	}
	if _, err := f.Seek(0, 0); err != nil {
		r.closeAndRemove()
		return nil, err
	}
	zr := zngio.NewReader(bufio.NewReader(f), zctx)
	rec, err := zr.Read()
	if err != nil {
		r.closeAndRemove()
		return nil, err
	}
	return &runFile{
		file:       f,
		nextRecord: rec,
		zr:         zr,
	}, nil
}

// closeAndRemove closes and removes the underlying file.
func (r *runFile) closeAndRemove() {
	r.file.Close()
	os.Remove(r.file.Name())
}

// read returns the next record along with a boolean that is true at EOF.
func (r *runFile) read() (*zng.Record, bool, error) {
	rec := r.nextRecord
	if rec != nil {
		rec = rec.Keep()
	}
	var err error
	r.nextRecord, err = r.zr.Read()
	eof := r.nextRecord == nil && err == nil
	return rec, eof, err
}

// writeZng writes records to w as a zng stream.
func writeZng(w io.Writer, records []*zng.Record) error {
	bw := bufio.NewWriter(w)
	zw := zngio.NewWriter(bw, zio.WriterFlags{})
	for _, rec := range records {
		if err := zw.Write(rec); err != nil {
			return err
		}
	}
	if err := zw.Flush(); err != nil {
		return nil
	}
	return bw.Flush()
}

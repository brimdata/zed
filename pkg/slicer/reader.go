// Package slicer provides a mechanism to read a single large file as
// a sequence of smaller slices that are essentially extracted from the
// large file on the fly to make the reader appear to produce a smaller file
// comprised of the slices.
package slicer

import (
	"io"
	"os"
)

// Reader implements io.Reader reading the sliced regions provided to it from
// the underlying file thus extracting subsets of an underlying file without
// modifying or copying the file.
type Reader struct {
	slices []Slice
	slice  Slice
	file   *os.File
	eof    bool
}

func NewReader(file *os.File, slices []Slice) (*Reader, error) {
	r := &Reader{
		slices: slices,
		file:   file,
	}
	return r, r.next()
}

func (r *Reader) next() error {
	if len(r.slices) == 0 {
		r.eof = true
		return nil
	}
	r.slice = r.slices[0]
	r.slices = r.slices[1:]
	_, err := r.file.Seek(int64(r.slice.Offset), 0)
	return err
}

func (r *Reader) Read(b []byte) (int, error) {
	if r.eof {
		return 0, io.EOF
	}
	p := b
	if uint64(len(p)) > r.slice.Length {
		p = p[:r.slice.Length]
	}
	n, err := r.file.Read(p)
	if n != 0 {
		if err == io.EOF {
			err = nil
		}
		r.slice.Length -= uint64(n)
		if r.slice.Length == 0 {
			err = r.next()
		}
	}
	return n, err
}

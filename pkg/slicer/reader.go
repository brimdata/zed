// Package slicer provides a mechanism to read a single large file as
// a sequence of smaller slices that are essentially extracted from the
// large file on the fly to make the reader appear to produce a smaller file
// comprised of the slices.
package slicer

import (
	"io"
	"os"
)

// Slicer implements io.Reader reading the sliced regions provided to it from
// the underlying file thus extracting subsets of an underlying file without
// modifying or copying the file.
type Reader struct {
	slices []Slice
	file   *os.File
	reader *io.SectionReader
}

func NewReader(file *os.File, slices []Slice) *Reader {
	first := slices[0]
	return &Reader{
		slices: slices[1:],
		file:   file,
		reader: first.NewReader(file),
	}
}

func (s *Reader) Read(b []byte) (int, error) {
	for s.reader != nil {
		n, err := s.reader.Read(b)
		if n != 0 {
			if err == io.EOF {
				err = nil
			}
			return n, err
		}
		if err != io.EOF {
			return 0, err
		}
		if len(s.slices) != 0 {
			s.reader = s.slices[0].NewReader(s.file)
			s.slices = s.slices[1:]
		} else {
			s.reader = nil
		}
	}
	return 0, io.EOF
}

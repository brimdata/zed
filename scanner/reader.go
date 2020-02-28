package scanner

import (
	"errors"
	"os"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng/resolver"
)

type Reader interface {
	zbuf.Reader
	Close() error
}

type FileReader struct {
	zbuf.Reader
	file *os.File
}

func OpenFile(zctx *resolver.Context, path, ifmt string) (*FileReader, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, errors.New("is a directory")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	r := detector.GzipReader(f)
	var zr zbuf.Reader
	if ifmt == "auto" {
		zr, err = detector.NewReader(r, zctx)
	} else {
		zr, err = detector.LookupReader(ifmt, r, zctx)
	}
	if err != nil {
		return nil, err
	}
	return &FileReader{zr, f}, nil
}

func (r *FileReader) Close() error {
	return r.file.Close()
}

func (r *FileReader) String() string {
	return r.file.Name()
}

func OpenFiles(zctx *resolver.Context, paths ...string) (Reader, error) {
	var readers []zbuf.Reader
	for _, path := range paths {
		reader, err := OpenFile(zctx, path, "auto")
		if err != nil {
			return nil, err
		}
		readers = append(readers, reader)
	}
	if len(readers) == 1 {
		return readers[0].(Reader), nil
	}
	return NewCombiner(readers), nil
}

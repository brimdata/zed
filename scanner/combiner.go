package scanner

import (
	"errors"
	"fmt"
	"os"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Reader struct {
	zbuf.Reader
	file *os.File
}

func (r *Reader) Read() (*zng.Record, error) {
	rec, err := r.Reader.Read()
	if err != nil {
		r.file.Close()
		return nil, err
	}
	if rec == nil {
		r.file.Close()
	}
	return rec, nil
}

func (r *Reader) String() string {
	return r.file.Name()
}

func OpenFile(zctx *resolver.Context, path string) (*Reader, error) {
	var f *os.File
	if path == "-" {
		f = os.Stdin
	} else {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			return nil, errors.New("is a directory")
		}
		f, err = os.Open(path)
		if err != nil {
			return nil, err
		}
	}
	r := detector.GzipReader(f)
	zr, err := detector.NewReader(r, zctx)
	if err != nil {
		return nil, err
	}
	reader := &Reader{zr, f}
	return reader, nil
}

func OpenFiles(zctx *resolver.Context, paths ...string) (zbuf.Reader, error) {
	var readers []zbuf.Reader
	for _, path := range paths {
		reader, err := OpenFile(zctx, path)
		if err != nil {
			return nil, err
		}
		readers = append(readers, reader)
	}
	if len(readers) == 1 {
		return readers[0], nil
	}
	return NewCombiner(readers), nil
}

type Combiner struct {
	readers []zbuf.Reader
	hol     []*zng.Record
	done    []bool
}

func NewCombiner(readers []zbuf.Reader) *Combiner {
	return &Combiner{
		readers: readers,
		hol:     make([]*zng.Record, len(readers)),
		done:    make([]bool, len(readers)),
	}
}

func (c *Combiner) Read() (*zng.Record, error) {
	idx := -1
	for k, l := range c.readers {
		if c.done[k] {
			continue
		}
		if c.hol[k] == nil {
			tup, err := l.Read()
			if err != nil {
				return nil, fmt.Errorf("%s: %w", c.readers[k], err)
			}
			if tup == nil {
				c.done[k] = true
				continue
			}
			c.hol[k] = tup
		}
		if idx == -1 || c.hol[k].Ts < c.hol[idx].Ts {
			idx = k
		}
	}
	if idx == -1 {
		return nil, nil
	}
	tup := c.hol[idx]
	c.hol[idx] = nil
	return tup, nil
}

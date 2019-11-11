package zsio

import (
	"io"

	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
)

type Reader struct {
	io.Reader
}

func NewReader(r io.Reader, table *resolver.Table) *Reader {
	return &Reader{Reader: r}
}

func (p *Reader) Read() (*zson.Record, error) {
	// XXX notyet
	return nil, nil
}

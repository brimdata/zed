package detector

import (
	"bytes"
	"errors"
	"io"

	"github.com/mccanne/zq/pkg/zsio/ndjson"
	"github.com/mccanne/zq/pkg/zsio/raw"
	"github.com/mccanne/zq/pkg/zsio/zjson"
	zsonio "github.com/mccanne/zq/pkg/zsio/zson"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
)

var ErrUnknown = errors.New("input format not recognized")

func NewReader(r io.Reader, n int, t *resolver.Table) (zson.Reader, error) {
	peeker, err := NewPeeker(r, n)
	if err != nil {
		return nil, err
	}
	b := peeker.Peek()
	head := bytes.NewReader(b)
	if match(zsonio.NewReader(head, resolver.NewTable())) {
		return zsonio.NewReader(peeker, t), nil
	}
	head.Reset(b)
	// zjson must come before ndjson since zjson is a subset of ndjson
	if match(zjson.NewReader(head, resolver.NewTable())) {
		return zjson.NewReader(peeker, t), nil
	}
	head.Reset(b)
	// ndjson must come after zjson since zjson is a subset of ndjson
	if match(ndjson.NewReader(head, resolver.NewTable())) {
		return ndjson.NewReader(peeker, t), nil
	}
	head.Reset(b)
	if match(raw.NewReader(head, resolver.NewTable())) {
		return raw.NewReader(peeker, t), nil
	}
	return nil, ErrUnknown
}

func match(r zson.Reader) bool {
	_, err := r.Read()
	return err == nil
}

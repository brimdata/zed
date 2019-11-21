package json

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/mccanne/zq/pkg/zson"
)

// Writer implements a Formatter for json output
type Writer struct {
	io.WriteCloser
	limit int
	array []*zson.Record
}

func NewWriter(w io.WriteCloser) *Writer {
	return &Writer{WriteCloser: w, limit: 10000}
}

func (p *Writer) Write(rec *zson.Record) error {
	// td from column 0 has been stripped out
	if len(p.array) >= p.limit {
		return fmt.Errorf("too many lines")
	}
	p.array = append(p.array, rec)
	return nil
}

func (p *Writer) Close() error {
	out, err := json.MarshalIndent(p.array, "", "    ")
	if err != nil {
		return err
	}
	_, err = p.WriteCloser.Write(out)
	if err != nil {
		return err
	}
	return p.WriteCloser.Close()
}

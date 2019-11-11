package ndjson

import (
	"errors"
	"io"

	"github.com/mccanne/zq/pkg/zson"
)

// NDJSON implements a Formatter for ndjson
type NDJSON struct {
	zson.Writer
}

func NewWriter(w io.WriteCloser) *NDJSON {
	return &NDJSON{Writer: zson.Writer{w}}
}

func (p *NDJSON) Write(rec *zson.Record) error {
	return errors.New("not yet implemented")
	// XXX not yet
	// td from column 0 has been stripped out
	// out, err := formatJSON(d, t)
	// if err != nil {
	// return err
	// }
	// out = append(out, byte('\n'))
	// _, err = p.File.Write(out)
	// return err
	return nil
}

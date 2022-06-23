package csvio

import (
	"encoding/csv"
	"errors"
	"io"
	"strconv"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zson"
)

type Reader struct {
	reader    *csv.Reader
	marshaler *zson.MarshalZNGContext
	strings   bool
	valid     bool
	hdr       []string
	vals      []interface{}
}

// XXX This is a placeholder for an option that will allow one to convert
// all csv fields to strings and defer any type coercion presumably to
// Zed shapers.  Currently, this causes an import cycle because the csvio
// Writer depends on fuse.  We should refactor this so whatever logic wants
// to tack on a fuse operator happens outside of zio.  See issue #2315
//type ReaderOpts struct {
//	StringsOnly bool
//}

func NewReader(zctx *zed.Context, r io.Reader) *Reader {
	preprocess := newPreprocess(r)
	reader := csv.NewReader(preprocess)
	reader.ReuseRecord = true
	reader.TrimLeadingSpace = true
	return &Reader{
		reader:    reader,
		marshaler: zson.NewZNGMarshalerWithContext(zctx),
		//strings:   opts.StringsOnly,
	}
}

func (r *Reader) Read() (*zed.Value, error) {
	for {
		csvRec, err := r.reader.Read()
		if err != nil {
			if err == io.EOF {
				if !r.valid {
					err = errors.New("empty csv file")
				} else {
					err = nil
				}
			}
			return nil, err
		}
		if r.hdr == nil {
			r.init(csvRec)
			continue
		}
		rec, err := r.translate(csvRec)
		if err != nil {
			return nil, err
		}
		r.valid = true
		return rec, nil
	}
}

func (r *Reader) init(hdr []string) {
	r.hdr = make([]string, len(hdr))
	copy(r.hdr, hdr)
	r.vals = make([]interface{}, len(hdr))
}

func (r *Reader) translate(fields []string) (*zed.Value, error) {
	if len(fields) != len(r.vals) {
		// This error shouldn't happen as it should be caught by the
		// csv package but we check anyway.
		return nil, errors.New("length of record doesn't match heading")
	}
	vals := r.vals[:0]
	for _, field := range fields {
		if r.strings {
			vals = append(vals, field)
		} else {
			vals = append(vals, convertString(field))
		}
	}
	return r.marshaler.MarshalCustom(r.hdr, vals)
}

func convertString(s string) interface{} {
	if s == "" {
		return nil
	}
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return v
	}
	if v, err := strconv.ParseBool(s); err == nil {
		return v
	}
	return s
}

package json

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/buger/jsonparser"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
)

const (
	MaxInputSize = 50 * 1024 * 1024
)

type Reader struct {
	reader  io.Reader
	table   *resolver.Table
	records []*zson.Record
	once    sync.Once
	err     error
}

func NewReader(reader io.Reader, r *resolver.Table) *Reader {
	return &Reader{
		reader: reader,
		table:  r,
	}
}

func (r *Reader) parseRecord(b []byte) (*zson.Record, error) {
	raw, typ, err := NewRawAndType(b)
	if err != nil {
		return nil, err
	}
	desc := r.table.GetByColumns(typ.(*zeek.TypeRecord).Columns)
	return zson.NewRecord(desc, 0, raw), nil
}

func (r *Reader) parseArray(b []byte) ([]*zson.Record, error) {
	var err error
	var records []*zson.Record
	jsonparser.ArrayEach(b, func(el []byte, typ jsonparser.ValueType, offset int, elErr error) {
		if err != nil {
			return
		}
		if err = elErr; err != nil {
			return
		}
		rec, perr := r.parseRecord(el)
		if err = perr; err != nil {
			return
		}
		records = append(records, rec)
	})
	return records, err
}

func (r *Reader) readinput() error {
	buf := bytes.NewBuffer(nil)
	_, err := io.CopyN(buf, r.reader, MaxInputSize)
	if err == nil {
		return fmt.Errorf("input exceeded max of %d bytes", MaxInputSize)
	}
	if err != io.EOF {
		return err
	}
	b := bytes.TrimSpace(buf.Bytes())
	if bytes.HasPrefix(b, []byte{'{'}) {
		rec, err := r.parseRecord(b)
		if err != nil {
			return err
		}
		r.records = append(r.records, rec)
		return nil
	} else if bytes.HasPrefix(b, []byte{'['}) {
		if r.records, err = r.parseArray(b); err != nil {
			return err
		}
		return nil
	}
	return errors.New("input not a recognizable JSON object or array")
}

func (r *Reader) Read() (*zson.Record, error) {
	var err error
	r.once.Do(func() {
		err = r.readinput()
	})
	if err != nil {
		return nil, err
	}
	if len(r.records) == 0 {
		return nil, nil
	}
	rec := r.records[0]
	r.records = r.records[1:]
	return rec, nil
}

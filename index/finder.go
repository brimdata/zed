package index

import (
	"context"
	"errors"
	"fmt"

	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/tzngio"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

var ErrNotFound = errors.New("key not found")

// Finder looks up values in a microindex using its embedded index.
type Finder struct {
	*Reader
	zctx *zson.Context
	uri  iosrc.URI
}

// NewFinder returns an object that is used to lookup keys in a microindex.
// It opens the file and reads the trailer, returning errors if the file is
// corrupt, doesn't exist, or has an invalid trailer.  If the microindex exists
// but is empty, zero values are returned for any lookups. If the microindex
// does not exist, a wrapped zqe.NotFound error is returned.
func NewFinder(ctx context.Context, zctx *zson.Context, uri iosrc.URI) (*Finder, error) {
	reader, err := NewReaderFromURI(ctx, zctx, uri)
	if err != nil {
		return nil, err
	}
	return &Finder{
		Reader: reader,
		zctx:   zctx,
		uri:    uri,
	}, nil
}

type operator int

const (
	eql operator = iota
	gte
	lte
)

// lookup searches for a match of the given key compared to the
// key values in the records read from the reader.  If the op argument is eql
// then only exact matches are returned.  Otherwise, the record with the
// largest key smaller (or larger) than the key argument is returned.
func lookup(reader zbuf.Reader, compare expr.KeyCompareFn, order zbuf.Order, op operator) (*zng.Record, error) {
	if order == zbuf.OrderAsc {
		return lookupAsc(reader, compare, op)
	}
	return lookupDesc(reader, compare, op)
}

func lookupAsc(reader zbuf.Reader, fn expr.KeyCompareFn, op operator) (*zng.Record, error) {
	var prev *zng.Record
	for {
		rec, err := reader.Read()
		if rec == nil || err != nil {
			if op == eql || op == gte {
				prev = nil
			}
			return prev, err
		}
		if cmp := fn(rec); cmp >= 0 {
			if cmp == 0 {
				return rec, nil
			}
			if op == eql {
				rec = nil
			}
			if op == lte {
				return prev, nil
			}
			return rec, nil
		}
		prev = rec
	}
}

func lookupDesc(reader zbuf.Reader, fn expr.KeyCompareFn, op operator) (*zng.Record, error) {
	var prev *zng.Record
	for {
		rec, err := reader.Read()
		if rec == nil || err != nil {
			if op == eql || op == lte {
				prev = nil
			}
			return prev, err
		}
		if cmp := fn(rec); cmp <= 0 {
			if cmp == 0 {
				return rec, nil
			}
			if op == eql {
				rec = nil
			}
			if op == gte {
				return prev, nil
			}
			return rec, nil
		}
		prev = rec
	}
}

func (f *Finder) search(compare expr.KeyCompareFn) (zbuf.Reader, error) {
	if f.reader == nil {
		panic("finder hasn't been opened")
	}
	// We start with the topmost level of the microindex file and
	// find the first key that matches according to the comparison,
	// then repeat the process for that frame in the next index file
	// till we get to the base layer and return a reader positioned at
	// that offset.
	n := len(f.trailer.Sections)
	off := int64(0)
	for level := 1; level < n; level++ {
		reader, err := f.newSectionReader(level, off)
		if err != nil {
			return nil, err
		}
		op := lte
		if f.trailer.Order == zbuf.OrderDesc {
			op = gte
		}
		rec, err := lookup(reader, compare, f.trailer.Order, op)
		if err != nil {
			return nil, err
		}
		if rec == nil {
			// This key can't be in the microindex since it is
			// smaller than the smallest key present.
			return nil, ErrNotFound
		}
		off, err = rec.AccessInt(f.trailer.ChildOffsetField)
		if err != nil {
			return nil, fmt.Errorf("b-tree child field: %w", err)
		}
	}
	return f.newSectionReader(0, off)
}

func (f *Finder) Lookup(keys *zng.Record) (*zng.Record, error) {
	if f.IsEmpty() {
		return nil, nil
	}
	compare, err := expr.NewKeyCompareFn(keys)
	if err != nil {
		return nil, err
	}
	reader, err := f.search(compare)
	if err != nil {
		if err == ErrNotFound {
			// Return nil/success when exact-match lookup fails
			err = nil
		}
		return nil, err
	}
	return lookup(reader, compare, f.trailer.Order, eql)
}

func (f *Finder) LookupAll(ctx context.Context, hits chan<- *zng.Record, keys *zng.Record) error {
	if f.IsEmpty() {
		return nil
	}
	compare, err := expr.NewKeyCompareFn(keys)
	if err != nil {
		return err
	}
	reader, err := f.search(compare)
	if err != nil {
		return err
	}
	for {
		// As long as we have an exact key-match, where unset key
		// columns are "don't care", keep reading records and return
		// them via the channel.
		rec, err := lookup(reader, compare, f.trailer.Order, eql)
		if err != nil {
			return err
		}
		if rec == nil {
			return nil
		}
		select {
		case hits <- rec:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// ClosestGTE returns the closest record that is greater than or equal to the
// provided key values.
func (f *Finder) ClosestGTE(keys *zng.Record) (*zng.Record, error) {
	return f.closest(keys, gte)
}

// ClosestLTE returns the closest record that is less than or equal to the
// provided key values.
func (f *Finder) ClosestLTE(keys *zng.Record) (*zng.Record, error) {
	return f.closest(keys, lte)
}

func (f *Finder) closest(keys *zng.Record, op operator) (*zng.Record, error) {
	if f.IsEmpty() {
		return nil, nil
	}
	compare, err := expr.NewKeyCompareFn(keys)
	if err != nil {
		return nil, err
	}
	reader, err := f.search(compare)
	if err != nil {
		return nil, err
	}
	return lookup(reader, compare, f.trailer.Order, op)
}

// ParseKeys uses the key template from the microindex trailer to parse
// a slice of string values which correspnod to the DFS-order
// of the fields in the key.  The inputs may be smaller than the
// number of key fields, in which case they are "don't cares"
// in terms of key lookups.  Any don't-care fields must all be
// at the end of the key record.
func (f *Finder) ParseKeys(inputs ...string) (*zng.Record, error) {
	if f.IsEmpty() {
		return nil, nil
	}
	// XXX this should parse a ZSON literal and try to cast it
	// to the key type.  For now, we let tzngio handle this as
	// Z literal syntax is coming soon.
	rec, err := tzngio.ParseKeys(f.trailer.KeyType, inputs...)
	if err == zng.ErrIncomplete {
		err = nil
	}
	return rec, err
}

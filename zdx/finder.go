package zdx

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

var ErrNotFound = errors.New("key not found")

// Finder looks up values in a zdx using its hierarchical index files.
type Finder struct {
	path        string
	keys        *zng.TypeRecord
	builder     *zng.Builder
	offsetField string
	zctx        *resolver.Context
	files       []*Reader
}

// NewFinder returns an object that is used to lookup keys in a zdx.
func NewFinder(zctx *resolver.Context, path string) *Finder {
	return &Finder{
		path: path,
		zctx: zctx,
	}
}

func (f *Finder) Keys() *zng.TypeRecord {
	return f.keys
}

// Open prepares the underlying zdx index for lookups.  It opens the file
// and reads the header, returning errors if the file is corrrupt, doesn't
// exist, or its zdx header is invalid.  If the index exists but is empty,
// zero values are returned for any lookups.  If the index does not exist,
// os.ErrNotExist is returned.
func (f *Finder) Open() error {
	level := 0
	for {
		r, err := newReader(f.zctx, f.path, level)
		if err != nil {
			if os.IsNotExist(err) {
				break
			}
			if len(f.files) > 0 {
				// Ignore this error; the prior is more
				// interesting.
				_ = f.Close()
			}
			return err
		}
		level += 1
		f.files = append(f.files, r)
	}
	if len(f.files) == 0 {
		return os.ErrNotExist
	}
	// Read the first record as the zdx header.
	rec, err := f.files[0].Read()
	if err != nil {
		return err
	}
	if rec == nil {
		// files exists but is empty
		return fmt.Errorf("%s: cannnot read zdx header", f.path)
	}
	childField, keysType, err := ParseHeader(rec)
	if err != nil {
		return fmt.Errorf("%s: %s", f.path, err)
	}
	f.keys = keysType
	f.offsetField = childField
	return nil
}

func (f *Finder) Path() string {
	return f.path
}

func (f *Finder) Close() error {
	for _, r := range f.files {
		if err := r.Close(); err != nil {
			return err
		}
	}
	return nil
}

// lookup searches for a match of the given key compared to the
// key values in the records read from the reader.  If the boolean argument
// "exact" is true, then only exact matches are returned.  Otherwise, the
// record with the lagest key smaller than the key argument is returned.
func lookup(reader zbuf.Reader, compare expr.KeyCompareFn, exact bool) (*zng.Record, error) {
	var prev *zng.Record
	for {
		rec, err := reader.Read()
		if err != nil || rec == nil {
			if exact {
				prev = nil
			}
			return prev, err
		}
		if cmp := compare(rec); cmp >= 0 {
			if cmp == 0 {
				return rec, nil
			}
			if exact {
				return nil, nil
			}
			return prev, nil
		}
		prev = rec
	}
}

func (f *Finder) search(compare expr.KeyCompareFn) error {
	n := len(f.files)
	if n == 0 {
		panic("open should have detected this")
	}
	// We start with the topmost index file of the zdx bundle and
	// find the greatest key smaller than or equal to othe lookup key then repeat
	// the process for that frame in the next index file till we get to the
	// base file and return that offset.
	level := n - 1
	off := int64(0)
	for level > 0 {
		reader := f.files[level]
		if _, err := reader.Seek(off); err != nil {
			return err
		}
		rec, err := lookup(reader, compare, false)
		if err != nil {
			return err
		}
		if rec == nil {
			// This key can't be in the zdx since it is smaller than
			// the smallest key in the zdx's index files.
			return ErrNotFound
		}
		off, err = rec.AccessInt(f.offsetField)
		if err != nil {
			return fmt.Errorf("b-tree child field: %w", err)
		}
		level -= 1
	}
	_, err := f.files[0].Seek(off)
	return err
}

func (f *Finder) Lookup(keys *zng.Record) (*zng.Record, error) {
	compare, err := expr.NewKeyCompareFn(keys)
	if err != nil {
		return nil, err
	}
	if err := f.search(compare); err != nil {
		if err == ErrNotFound {
			// Return nil/success when exact-match lookup fails
			err = nil
		}
		return nil, err
	}
	return lookup(f.files[0], compare, true)
}

func (f *Finder) LookupAll(ctx context.Context, hits chan<- *zng.Record, keys *zng.Record) error {
	compare, err := expr.NewKeyCompareFn(keys)
	if err != nil {
		return err
	}
	if err := f.search(compare); err != nil {
		return err
	}
	for {
		// As long as we have an exact key-match, where unset key
		// columns are "don't care", keep reading records and return
		// them via the channel.
		rec, err := lookup(f.files[0], compare, true)
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

func (f *Finder) LookupClosest(keys *zng.Record) (*zng.Record, error) {
	compare, err := expr.NewKeyCompareFn(keys)
	if err != nil {
		return nil, err
	}
	if err := f.search(compare); err != nil {
		return nil, err
	}
	return lookup(f.files[0], compare, false)
}

// ParseKeys uses the key template from the zdx header to parse
// a slice of string values which correspnod to the DFS-order
// of the fields in the key.  The inputs may be smaller than the
// number of key fields, in which case they are "don't cares"
// in terms of key lookups.  Any don't-care fields must all be
// at the end of the key record.
func (f *Finder) ParseKeys(inputs []string) (*zng.Record, error) {
	if f.builder == nil {
		f.builder = zng.NewBuilder(f.keys)
	}
	rec, err := f.builder.Parse(inputs...)
	if err == zng.ErrIncomplete {
		err = nil
	}
	return rec, err
}

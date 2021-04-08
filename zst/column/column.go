// Package column implements the organization of columns on storage for a
// zst columnar storage object.
//
// A zst object is created by allocating a RecordWriter for a top-level zng row type
// (i.e., "schema") via NewRecordWriter.  The object to be written to is wrapped
// in a Spiller with a column threshold.  Output is streamed to the underlying spiller
// in a single pass.  (In the future, we may implement multiple passes to optimize
// the storage layout of column data or spread a given zst object across multiple
//
// NewRecordWriter recursively decends the record type, allocating a column Writer
// for each node in the type tree.  The top-level record body is written via a call
// to Write and all of the columns are called with their respetive values represented
// as a zcode.Bytes.  The columns buffer data in memorry until they reach their
// byte threshold or until Flush is called.
//
// After all of the zng data is written, a reassembly record may be formed for
// the RecordColumn by calling its MarshalZNG method, which builds the record
// value in place using zcode.Builder and returns the zng.TypeRecord (i.e., schema)
// of that record column.
//
// Data is read from a zst file by scanning the reassembly records then unmarshaling
// a zng.Record body into an empty Record by calling Record.UnmarshalZNG, which
// recusirvely builds an assembly structure.  An io.ReaderAt is passed to unmarshal
// so each column reader can access the underlying storage object and read its
// column data effciently in largish column chunks.
//
// Once an assembly is built, the recontructed zng row data can be read from the
// assembly by calling the Read method on the top-level Record and passing in
// a zcode.Builder to reconstruct the record body in place.  The assembly does not
// need any type information as the structure of values is entirely self describing
// in the zng data format.
package column

import (
	"io"

	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

const MaxSegmentThresh = 20 * 1024 * 1024

type Writer interface {
	// Write encodes the given value into memory.  When the column exceeds
	// a threshold, it is automatically flushed.  Flush may also be called
	// explicitly to push columns to storage and thus avoid too much row skew
	// between columns.
	Write(zcode.Bytes) error
	// Push all in-memory column data to the storage layer.
	Flush(bool) error
	// MarshalZNG is called after all data is flushed to build the reassembly
	// record for this column.
	MarshalZNG(*zson.Context, *zcode.Builder) (zng.Type, error)
}

func NewWriter(typ zng.Type, spiller *Spiller) Writer {
	switch typ := typ.(type) {
	case *zng.TypeAlias:
		return NewWriter(typ.Type, spiller)
	case *zng.TypeRecord:
		return NewRecordWriter(typ, spiller)
	case *zng.TypeArray:
		return NewArrayWriter(typ.Type, spiller)
	case *zng.TypeSet:
		// Sets encode the same way as arrays but behave
		// differently semantically, and we don't care here.
		return NewArrayWriter(typ.Type, spiller)
	case *zng.TypeUnion:
		return NewUnionWriter(typ, spiller)
	default:
		return NewPrimitiveWriter(spiller)
	}
}

type Interface interface {
	Read(*zcode.Builder) error
}

func Unmarshal(typ zng.Type, in zng.Value, r io.ReaderAt) (Interface, error) {
	switch typ := typ.(type) {
	case *zng.TypeAlias:
		return Unmarshal(typ.Type, in, r)
	case *zng.TypeRecord:
		record := &Record{}
		err := record.UnmarshalZNG(typ, in, r)
		return record, err
	case *zng.TypeArray:
		a := &Array{}
		err := a.UnmarshalZNG(typ.Type, in, r)
		return a, err
	case *zng.TypeSet:
		// Sets encode the same way as arrays but behave
		// differently semantically, and we don't care here.
		a := &Array{}
		err := a.UnmarshalZNG(typ.Type, in, r)
		return a, err
	case *zng.TypeUnion:
		u := &Union{}
		err := u.UnmarshalZNG(typ, in, r)
		return u, err
	default:
		p := &Primitive{}
		err := p.UnmarshalZNG(in, r)
		return p, err
	}
}

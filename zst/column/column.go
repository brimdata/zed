// Package column implements the organization of columns on storage for a
// ZST columnar storage object.
//
// A ZST object is created by allocating a Writer for any top-level Zed type
// via NewWriter.  The object to be written is wrapped
// in a Spiller with a column threshold.  Output is streamed to the underlying spiller
// in a single pass.  (In the future, we may implement multiple passes to optimize
// the storage layout of column data or spread a given ZST object across multiple
// files.
//
// NewWriter recursively decends into the Zed type, allocating a Writer
// for each node in the type tree.  The top-level body is written via a call
// to Write.  The columns buffer data in memory until they reach their
// byte threshold or until Flush is called.
//
// After all of the Zed data is written, a reassembly map is formed for
// each column writer by calling its EncodeMap method, which builds the
// value in place using zcode.Builder and returns the Zed type of
// the reassembly map value.
//
// Data is read from a ZST file by scanning the reassembly maps to build
// column Readers for each Zed type by calling NewReader with the map, which
// recusirvely builds an assembly structure.  An io.ReaderAt is passed to NewReader
// so each column reader can access the underlying storage object and read its
// column data effciently in largish column chunks.
//
// Once an assembly is built, the recontructed Zed row data can be read from the
// assembly by calling the Read method on the top-level Record and passing in
// a zcode.Builder to reconstruct the record body in place.  The assembly does not
// need any type information as the structure of values is entirely self describing
// in the Zed data format.
package column

import (
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
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
	// EncodeMap is called after all data is flushed to build the reassembly
	// record for this column.
	EncodeMap(*zed.Context, *zcode.Builder) (zed.Type, error)
}

func NewWriter(typ zed.Type, spiller *Spiller) Writer {
	switch typ := typ.(type) {
	case *zed.TypeNamed:
		return NewWriter(typ.Type, spiller)
	case *zed.TypeRecord:
		return NewRecordWriter(typ, spiller)
	case *zed.TypeArray:
		return NewArrayWriter(typ.Type, spiller)
	case *zed.TypeSet:
		// Sets encode the same way as arrays but behave
		// differently semantically, and we don't care here.
		return NewArrayWriter(typ.Type, spiller)
	case *zed.TypeUnion:
		return NewUnionWriter(typ, spiller)
	default:
		return NewPrimitiveWriter(spiller)
	}
}

type Reader interface {
	Read(*zcode.Builder) error
}

func NewReader(typ zed.Type, in zed.Value, r io.ReaderAt) (Reader, error) {
	switch typ := typ.(type) {
	case *zed.TypeNamed:
		return NewReader(typ.Type, in, r)
	case *zed.TypeRecord:
		return NewRecordReader(typ, in, r)
	case *zed.TypeArray:
		return NewArrayReader(typ.Type, in, r)
	case *zed.TypeSet:
		return NewArrayReader(typ.Type, in, r)
	case *zed.TypeUnion:
		return NewUnionReader(typ, in, r)
	case *zed.TypeMap:
		return nil, errors.New("type 'map' is currently unsupported by the ZST implementation")
	case *zed.TypeEnum:
		return nil, errors.New("type 'enum' is currently unsupported by the ZST implementation")
	case *zed.TypeError:
		return nil, errors.New("type 'error' is currently unsupported by the ZST implementation")
	default:
		return NewPrimitiveReader(in, r)
	}
}

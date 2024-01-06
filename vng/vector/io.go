// Package vector implements the organization of Zed data on storage as
// vectors in a VNG vector storage object.
//
// A VNG object is created by allocating a Writer for any top-level Zed type
// via NewWriter.  The object to be written is wrapped
// in a Spiller with a vector threshold.  Output is streamed to the underlying spiller
// in a single pass.
//
// NewWriter recursively descends into the Zed type, allocating a Writer
// for each node in the type tree.  The top-level body is written via a call
// to Write.  Each vector buffers its data in memory until it reaches a
// byte threshold or until Flush is called.
//
// After all of the Zed data is written, a metadata section is written consisting
// of segment maps for each vector, each obtained by calling the Metadata
// method on the vng.Writer interface.
//
// Nulls for complex types are encoded by a special Nulls object.  Each complex
// type is wrapped by a NullsWriter, which runlength encodes any alternating
// sequences of nulls and values.  If no nulls are encountered, then the Nulls
// object is omitted from the metadata.
//
// Data is read from a VNG file by scanning the metadata maps to build
// vector Readers for each Zed type by calling NewReader with the metadata, which
// recusirvely builds reassembly segments.  An io.ReaderAt is passed to NewReader
// so each vector reader can access the underlying storage object and read its
// vector data effciently in largish vector segments.
//
// Once the metadata is assembled in memory, the recontructed Zed sequence data can be
// read from the vector segments by calling the Read method on the top-level
// Reader and passing in a zcode.Builder to reconstruct the Zed value in place.
package vector

import (
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"golang.org/x/sync/errgroup"
)

type Writer interface {
	// Write encodes the given value into memory.
	Write(zcode.Bytes)
	// Encoded all in-memory vector data into its storage-ready serialized format.
	// Vectors may be encoded concurrently and errgroup.Group is used to sync
	// and return errors.
	Encode(*errgroup.Group)
	// Metadata returns the data structure conforming to the VNG specification
	// describing the layout of vectors.  This is called after all data is
	// written and encoded by the Writer with the result marshaled to build
	// the header section of the VNG object.  An offset is passed down into
	//  the traversal representing where in the data section the vector data
	// will land.  This is called in a sequential fashion (no parallelism) so
	// that the metadata can be computed and the VNG header written before the
	// vector data is written via Emit.
	Metadata(uint64) (uint64, Metadata)
	Emit(w io.Writer) error
}

func NewWriter(typ zed.Type) Writer {
	switch typ := typ.(type) {
	case *zed.TypeNamed:
		return &NamedWriter{NewWriter(typ.Type), typ.Name}
	case *zed.TypeRecord:
		return NewNullsWriter(NewRecordWriter(typ))
	case *zed.TypeArray:
		return NewNullsWriter(NewArrayWriter(typ))
	case *zed.TypeSet:
		// Sets encode the same way as arrays but behave
		// differently semantically, and we don't care here.
		return NewNullsWriter(NewSetWriter(typ))
	case *zed.TypeMap:
		return NewNullsWriter(NewMapWriter(typ))
	case *zed.TypeUnion:
		return NewNullsWriter(NewUnionWriter(typ))
	default:
		if !zed.IsPrimitiveType(typ) {
			panic(fmt.Sprintf("unsupported type in VNG file: %T", typ))
		}
		return NewNullsWriter(NewPrimitiveWriter(typ, true))
	}
}

type NamedWriter struct {
	Writer
	name string
}

func (n *NamedWriter) Metadata(off uint64) (uint64, Metadata) {
	off, meta := n.Writer.Metadata(off)
	return off, &Named{n.name, meta}
}

type Reader interface {
	Read(*zcode.Builder) error
}

func NewReader(meta Metadata, r io.ReaderAt) (Reader, error) {
	switch meta := meta.(type) {
	case nil:
		return nil, nil
	case *Nulls:
		inner, err := NewReader(meta.Values, r)
		if err != nil {
			return nil, err
		}
		return NewNullsReader(inner, meta.Runs, r), nil
	case *Named:
		return NewReader(meta.Values, r)
	case *Record:
		return NewRecordReader(meta, r)
	case *Array:
		return NewArrayReader(meta, r)
	case *Set:
		return NewArrayReader((*Array)(meta), r)
	case *Map:
		return NewMapReader(meta, r)
	case *Union:
		return NewUnionReader(meta, r)
	case *Primitive:
		if len(meta.Dict) != 0 {
			return NewDictReader(meta, r), nil
		}
		return NewPrimitiveReader(meta, r), nil
	case *Const:
		return NewConstReader(meta), nil
	default:
		return nil, fmt.Errorf("unknown VNG metadata type: %T", meta)
	}
}

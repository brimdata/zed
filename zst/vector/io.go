// Package vector implements the organization of Zed data on storage as
// vectors in a ZST vector storage object.
//
// A ZST object is created by allocating a Writer for any top-level Zed type
// via NewWriter.  The object to be written is wrapped
// in a Spiller with a vector threshold.  Output is streamed to the underlying spiller
// in a single pass.
//
// NewWriter recursively decends into the Zed type, allocating a Writer
// for each node in the type tree.  The top-level body is written via a call
// to Write.  The vectors buffer data in memory until they reach their
// byte threshold or until Flush is called.
//
// After all of the Zed data is written, a metadata section is written consisting
// of segment maps for each vector, each obtained by calling the Metadata
// method on the zst.Writer interface.
//
// Data is read from a ZST file by scanning the metadata maps to build
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
)

const MaxSegmentThresh = 20 * 1024 * 1024

type Writer interface {
	// Write encodes the given value into memory.  When the vector exceeds
	// a threshold, it is automatically flushed.  Flush may also be called
	// explicitly to push vectors to storage and thus avoid too much row skew
	// between vectors.
	Write(zcode.Bytes) error
	// Push all in-memory vector data to the storage layer.
	Flush(bool) error
	// Metadata returns the data structure conforming to the ZST specification
	// describing the layout of vectors.  This is called after all data is
	// written and flushed by the Writer with the result marshaled to build
	// the metadata section of the ZST file.
	Metadata() Metadata
}

func NewWriter(typ zed.Type, spiller *Spiller) Writer {
	switch typ := typ.(type) {
	case *zed.TypeNamed:
		return &NamedWriter{NewWriter(typ.Type, spiller), typ.Name}
	case *zed.TypeRecord:
		return NewRecordWriter(typ, spiller)
	case *zed.TypeArray:
		return NewArrayWriter(typ.Type, spiller)
	case *zed.TypeSet:
		// Sets encode the same way as arrays but behave
		// differently semantically, and we don't care here.
		return NewSetWriter(typ.Type, spiller)
	case *zed.TypeUnion:
		return NewUnionWriter(typ, spiller)
	default:
		if !zed.IsPrimitiveType(typ) {
			panic(fmt.Sprintf("unsupported type in ZST file: %T", typ))
		}
		return NewPrimitiveWriter(typ, spiller)
	}
}

type NamedWriter struct {
	Writer
	name string
}

func (n *NamedWriter) Metadata() Metadata {
	return &Named{n.name, n.Writer.Metadata()}
}

type Reader interface {
	Read(*zcode.Builder) error
}

func NewReader(meta Metadata, r io.ReaderAt) (Reader, error) {
	switch meta := meta.(type) {
	case nil:
		return nil, nil
	case *Named:
		return NewReader(meta.Values, r)
	case *Record:
		return NewRecordReader(meta, r)
	case *Array:
		return NewArrayReader(meta, r)
	case *Set:
		return NewArrayReader((*Array)(meta), r)
	case *Union:
		return NewUnionReader(meta, r)
	case *Primitive:
		return NewPrimitiveReader(meta, r), nil
	default:
		return nil, fmt.Errorf("unknown ZST metadata type: %T", meta)
	}
}

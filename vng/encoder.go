package vng

import (
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"golang.org/x/sync/errgroup"
)

type Encoder interface {
	// Write collects up values to be encoded into memory.
	Write(zcode.Bytes)
	// Encode encodes all in-memory vector data into its storage-ready serialized format.
	// Vectors may be encoded concurrently and errgroup.Group is used to sync
	// and return errors.
	Encode(*errgroup.Group)
	// Metadata returns the data structure conforming to the VNG specification
	// describing the layout of vectors.  This is called after all data is
	// written and encoded by the Encode with the result marshaled to build
	// the header section of the VNG object.  An offset is passed down into
	// the traversal representing where in the data section the vector data
	// will land.  This is called in a sequential fashion (no parallelism) so
	// that the metadata can be computed and the VNG header written before the
	// vector data is written via Emit.
	Metadata(uint64) (uint64, Metadata)
	Emit(w io.Writer) error
}

func NewEncoder(zctx *zed.Context, typ zed.Type) Encoder {
	switch typ := typ.(type) {
	case *zed.TypeNamed:
		return &NamedEncoder{NewEncoder(zctx, typ.Type), typ.Name}
	case *zed.TypeRecord:
		return NewNullsEncoder(NewRecordEncoder(zctx, typ))
	case *zed.TypeArray:
		return NewNullsEncoder(NewArrayEncoder(zctx, typ))
	case *zed.TypeSet:
		// Sets encode the same way as arrays but behave
		// differently semantically, and we don't care here.
		return NewNullsEncoder(NewSetEncoder(zctx, typ))
	case *zed.TypeMap:
		return NewNullsEncoder(NewMapEncoder(zctx, typ))
	case *zed.TypeUnion:
		return NewNullsEncoder(NewUnionEncoder(zctx, typ))
	default:
		if !zed.IsPrimitiveType(typ) {
			panic(fmt.Sprintf("unsupported type in VNG file: %T", typ))
		}
		return NewNullsEncoder(NewPrimitiveEncoder(zctx, typ, true))
	}
}

type NamedEncoder struct {
	Encoder
	name string
}

func (n *NamedEncoder) Metadata(off uint64) (uint64, Metadata) {
	off, meta := n.Encoder.Metadata(off)
	return off, &Named{n.name, meta}
}

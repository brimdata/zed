package vcache

import (
	"context"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zst"
	"github.com/segmentio/ksuid"
	"golang.org/x/sync/errgroup"
)

const MaxTypesPerObject = 2500

// Object represents the collection of vectors that are loaded into
// memory for a given data object as referenced by its ID.
// An Object structure mirrors the metadata structures used in ZST but here
// we support dynamic loading of vectors as they are needed and data and
// metadata are all cached in memory.
type Object struct {
	id     ksuid.KSUID
	uri    *storage.URI
	engine storage.Engine
	reader storage.Reader
	// We keep a local context for each object since a new type context is created
	// for each query and we need to map the ZST object context to the query
	// context.  Of course, with Zed, this is very cheap.
	local *zed.Context
	// There is one vector per Zed type and the typeIDs array provides
	// the sequence order of each vector to be accessed.  When
	// ordering doesn't matter, the vectors can be traversed directly
	// without an indirection through the typeIDs array.
	vectors []Vector
	types   []zed.Type
	typeIDs []int32
}

// NewObject creates a new in-memory Object corresponding to a ZST object
// residing in storage.  It loads the list of ZST root types (one per value
// in the file) and the ZST metadata for vector reassembly.  This provides
// the metadata needed to load vector chunks on demand only as they are
// referenced.
func NewObject(ctx context.Context, engine storage.Engine, uri *storage.URI, id ksuid.KSUID) (*Object, error) {
	// XXX currently we open a storage.Reader for every object and never close it.
	// We should either close after a timeout and reopen when needed or change the
	// storage API to have a more reasonable semantics around the Put/Get not leaving
	// a file descriptor open for every long Get.  Perhaps there should be another
	// method for intermitted random access.
	reader, err := engine.Get(ctx, uri)
	if err != nil {
		return nil, err
	}
	size, err := storage.Size(reader)
	if err != nil {
		return nil, err
	}
	zctx := zed.NewContext()
	z, err := zst.NewObject(zctx, reader, size)
	if err != nil {
		return nil, err
	}
	typeIDs, metas, err := z.FetchMetadata()
	if err != nil {
		return nil, err
	}
	if len(metas) == 0 {
		return nil, fmt.Errorf("empty ZST object: %s", uri)
	}
	if len(metas) > MaxTypesPerObject {
		return nil, fmt.Errorf("too many types is ZST object: %s", uri)
	}
	types := make([]zed.Type, 0, len(metas))
	for _, meta := range metas {
		types = append(types, meta.Type(zctx))
	}
	var group errgroup.Group
	vectors := make([]Vector, len(metas))
	for k, meta := range metas {
		which := k
		this := meta
		group.Go(func() error {
			v, err := NewVector(this, reader)
			if err != nil {
				return err
			}
			vectors[which] = v
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	return &Object{
		id:      id,
		uri:     uri,
		engine:  engine,
		reader:  reader,
		local:   zctx,
		vectors: vectors,
		types:   types,
		typeIDs: typeIDs,
	}, nil
}

func (o *Object) Close() error {
	if o.reader != nil {
		return o.reader.Close()
	}
	return nil
}

func (o *Object) NewReader() *Reader {
	return &Reader{
		object: o,
		iters:  make([]iterator, len(o.vectors)),
	}
}

func (o *Object) NewProjection(fields []string) (*Projection, error) {
	return NewProjection(o, fields)
}

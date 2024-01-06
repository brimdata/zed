package vcache

import (
	"context"
	"io"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/vector"
	"github.com/brimdata/zed/vng"
	meta "github.com/brimdata/zed/vng/vector"
	"github.com/segmentio/ksuid"
)

const MaxTypesPerObject = 2500

// Object represents the collection of vectors that are loaded into
// memory for a given data object as referenced by its ID.
// An Object structure mirrors the metadata structures used in VNG but here
// we support dynamic loading of vectors as they are needed and data and
// metadata are all cached in memory.
type Object struct {
	mu     []sync.Mutex
	id     ksuid.KSUID
	uri    *storage.URI
	engine storage.Engine
	reader io.ReaderAt
	// We keep a local context for each object since a new type context is created
	// for each query and we need to map the VNG object context to the query
	// context.  Of course, with Zed, this is very cheap.
	local *zed.Context

	//XXX this is all gonna change in a subsequent PR when we get the Variant
	// data type working across vng, vcache, and vector
	metas []meta.Metadata
	types []zed.Type
	tags  []int32

	vectors []vector.Any
}

// NewObject creates a new in-memory Object corresponding to a VNG object
// residing in storage.  The VNG header and metadata section are read and
// the metadata is deserialized so that vectors can be loaded into the cache
// on demand only as needed and retained in memory for future use.
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
	// XXX use the query's zctx so we don't have to map?,
	// or maybe use a single context across all objects in the cache?
	zctx := zed.NewContext()
	z, err := vng.NewObject(zctx, reader)
	if err != nil {
		return nil, err
	}
	types, metas, tags, err := z.MiscMeta()
	if err != nil {
		return nil, err
	}
	return &Object{
		mu:      make([]sync.Mutex, len(metas)),
		id:      id,
		uri:     uri,
		engine:  engine,
		reader:  z.DataReader(),
		local:   zctx,
		metas:   metas,
		types:   types,
		tags:    tags,
		vectors: make([]vector.Any, len(metas)),
	}, nil
}

func (o *Object) Close() error {
	if o.reader != nil {
		if closer, ok := o.reader.(io.Closer); ok {
			return closer.Close()
		}
	}
	return nil
}

func (o *Object) LocalContext() *zed.Context {
	return o.local
}

func (o *Object) Types() []zed.Type {
	return o.types
}

func (o *Object) Load(tag uint32, path field.Path) (vector.Any, error) {
	l := loader{o.local, o.reader}
	o.mu[tag].Lock()
	defer o.mu[tag].Unlock()
	return l.loadVector(&o.vectors[tag], o.types[tag], path, o.metas[tag])
}

func (o *Object) NewReader() *Reader {
	return &Reader{
		object:   o,
		builders: make([]vector.Builder, len(o.vectors)),
	}
}

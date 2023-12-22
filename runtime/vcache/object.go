package vcache

import (
	"context"
	"fmt"
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
	reader storage.Reader
	// We keep a local context for each object since a new type context is created
	// for each query and we need to map the VNG object context to the query
	// context.  Of course, with Zed, this is very cheap.
	local *zed.Context
	metas []meta.Metadata
	// There is one vector per Zed type and the typeKeys array provides
	// the sequence order of each vector to be accessed.
	vectors  []vector.Any
	typeDict []zed.Type
	typeKeys []int32
	//slots    map[int32][]int32 //XXX handle this differently?
}

// NewObject creates a new in-memory Object corresponding to a VNG object
// residing in storage.  It loads the list of VNG root types (one per value
// in the file) and the VNG metadata for vector reassembly.  A table for each
// type is also created to map the global slot number in the object to the local
// slot number in the type so that an element's local position in the vector
// (within a particular type) can be related to its slot number in the object,
// e.g., so that filtering of a local vector can be turned into the list of
// matching object slots.  The object provides the metadata needed to load vectors
// on demand only as they are referenced.  A vector is loaded by calling its Load method,
// which decodes its zcode.Bytes into its native representation.
// XXX we may want to change the VNG format to code vectors in native format.
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
	// XXX use the query's zctx so we don't have to map?,
	// or maybe use a single context across all objects in the cache?
	zctx := zed.NewContext()
	z, err := vng.NewObject(zctx, reader, size)
	if err != nil {
		return nil, err
	}
	typeKeys, metas, err := z.FetchMetadata()
	if err != nil {
		return nil, err
	}
	if len(metas) == 0 {
		return nil, fmt.Errorf("empty VNG object: %s", uri)
	}
	if len(metas) > MaxTypesPerObject {
		return nil, fmt.Errorf("too many types in VNG object: %s", uri)
	}
	typeDict := make([]zed.Type, 0, len(metas))
	for _, meta := range metas {
		typeDict = append(typeDict, meta.Type(zctx)) //XXX commanet about context locality
	}
	vectors := make([]vector.Any, len(metas))
	return &Object{
		mu:       make([]sync.Mutex, len(typeDict)),
		id:       id,
		uri:      uri,
		engine:   engine,
		reader:   reader,
		local:    zctx,
		metas:    metas,
		vectors:  vectors,
		typeDict: typeDict,
		typeKeys: typeKeys,
		//slots:    slots,
	}, nil
}

func (o *Object) Close() error {
	if o.reader != nil {
		return o.reader.Close()
	}
	return nil
}

func (o *Object) LocalContext() *zed.Context {
	return o.local
}

func (o *Object) Types() []zed.Type {
	return o.typeDict
}

func (o *Object) TypeKeys() []int32 {
	return o.typeKeys
}

func (o *Object) LookupType(typeKey uint32) zed.Type {
	return o.typeDict[typeKey]
}

func (o *Object) Len() int {
	return len(o.typeKeys)
}

// XXX fix comment
// Due to the heterogenous nature of Zed data, a given path can appear in
// multiple types and a given type can have multiple vectors XXX (due to union
// types in the hiearchy).  Load returns a Group for each type and the Group
// may contain multiple vectors.
func (o *Object) Load(typeKey uint32, path field.Path) (vector.Any, error) {
	l := loader{o.local, o.reader}
	o.mu[typeKey].Lock()
	defer o.mu[typeKey].Unlock()
	return l.loadVector(&o.vectors[typeKey], o.typeDict[typeKey], path, o.metas[typeKey])
}

func (o *Object) NewReader() *Reader {
	return &Reader{
		object:   o,
		builders: make([]vector.Builder, len(o.vectors)),
	}
}

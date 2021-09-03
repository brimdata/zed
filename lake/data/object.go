package data

import (
	"context"
	"fmt"
	"regexp"

	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zqe"
	"github.com/segmentio/ksuid"
)

const DefaultThreshold = 500 * 1024 * 1024

// A FileKind is the first part of a file name, used to differentiate files
// when they are listed from the archive's backing store.
type FileKind string

const (
	FileKindUnknown  FileKind = ""
	FileKindData     FileKind = "data"
	FileKindMetadata FileKind = "meta"
	FileKindSeek     FileKind = "seek"
)

func (k FileKind) Description() string {
	switch k {
	case FileKindData:
		return "data"
	case FileKindMetadata:
		return "metadata"
	case "FileKindSeek":
		return "seekindex"
	default:
		return "unknown"
	}
}

//XXX all file types are cacheable but seek index etc is not matched here.
//and seekindexes really do want to be cached as they are small and
// eliminate round-trips, especially when you are ready sub-ranges of
// cached data files!

var fileRegex = regexp.MustCompile(`([0-9A-Za-z]{27}-(data|meta)).zng$`)

//XXX this won't work right until we integrate segID
func FileMatch(s string) (kind FileKind, id ksuid.KSUID, ok bool) {
	match := fileRegex.FindStringSubmatch(s)
	if match == nil {
		return
	}
	k := FileKind(match[1])
	switch k {
	case FileKindData:
	case FileKindMetadata:
	default:
		return
	}
	id, err := ksuid.Parse(match[2])
	if err != nil {
		return
	}
	return k, id, true
}

type Metadata struct {
	First   zng.Value `zng:"first"`
	Last    zng.Value `zng:"last"`
	Count   uint64    `zng:"count"`
	RowSize int64     `zng:"row_size"`
}

//XXX
// A Segment is a file that holds Zed records ordered according to the
// pool's data order.
// seekIndexPath returns the path of an associated seek index for the ZNG
// version of data, which can be used to lookup a nearby seek offset
// for a desired pool-key value.
// MetadataPath returns the path of an associated ZNG file that holds
// information about the records in the chunk, including the total number,
// and the first and last (hence smallest and largest) record values of the pool key.
// XXX should First/Last be wrt pool order or be smallest and largest?

type Object struct {
	ID       ksuid.KSUID `zng:"id"`
	Metadata `zng:"meta"`
}

func (o Object) IsZero() bool {
	return o.ID == ksuid.Nil
}

func (o Object) String() string {
	return fmt.Sprintf("%s %d record%s in %d data bytes", o.ID, o.Count, plural(int(o.Count)), o.RowSize)
}

func plural(ordinal int) string {
	if ordinal == 1 {
		return ""
	}
	return "s"
}

func (o Object) StringRange() string {
	return fmt.Sprintf("%s %s %s", o.ID, o.First, o.Last)
}

func (o *Object) Equal(to *Object) bool {
	return o.ID == to.ID
}

func NewObject() Object {
	return Object{ID: ksuid.New()}
}

func (o Object) Span(order order.Which) *extent.Generic {
	if o.First.Bytes == nil || o.Last.Bytes == nil {
		//XXX
		return nil
	}
	return extent.NewGenericFromOrder(o.First, o.Last, order)
}

// ObjectPrefix returns a prefix for the various objects that comprise
// a data object so they can all be deleted with the storage engine's
// DeleteByPrefix method.
func (o Object) ObjectPrefix(path *storage.URI) *storage.URI {
	return path.AppendPath(o.ID.String())
}

func (o Object) RowObjectName() string {
	return RowObjectName(o.ID)
}

func RowObjectName(id ksuid.KSUID) string {
	return fmt.Sprintf("%s.zng", id)
}

func (o Object) RowObjectPath(path *storage.URI) *storage.URI {
	return RowObjectPath(path, o.ID)
}

func RowObjectPath(path *storage.URI, id ksuid.KSUID) *storage.URI {
	return path.AppendPath(RowObjectName(id))
}

func (o Object) SeekObjectName() string {
	return fmt.Sprintf("%s-seek.zng", o.ID)
}

func (o Object) SeekObjectPath(path *storage.URI) *storage.URI {
	return path.AppendPath(o.SeekObjectName())
}

func (o Object) Range() string {
	//XXX need to handle any key... will the String method work?
	return fmt.Sprintf("[%d-%d]", o.First, o.Last)
}

// Remove deletes the row object and its seek index.
// Any 'not found' errors are ignored.
func (o Object) Remove(ctx context.Context, engine storage.Engine, path *storage.URI) error {
	if err := engine.DeleteByPrefix(ctx, o.ObjectPrefix(path)); err != nil && !zqe.IsNotFound(err) {
		return err
	}
	return nil
}

package segment

import (
	"context"
	"fmt"
	"regexp"
	"sort"

	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zbuf"
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
	First   nano.Ts `zng:"first"`
	Last    nano.Ts `zng:"last"`
	Count   uint64  `zng:"count"`
	Size    int64   `zng:"size"`
	RowSize int64   `zng:"row_size"`
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

type Reference struct {
	ID       ksuid.KSUID `zng:"id"`
	Metadata `zng:"meta"`
}

func (r Reference) IsZero() bool {
	return r.ID == ksuid.Nil
}

func (r Reference) String() string {
	return fmt.Sprintf("%s %d records in %d data bytes (%d byte object)", r.ID, r.Count, r.Size, r.RowSize)
}

func (r Reference) StringRange() string {
	return fmt.Sprintf("%s %s %s", r.ID, r.First, r.Last)
}

func (r *Reference) Equal(to *Reference) bool {
	return r.ID == to.ID
}

func New() Reference {
	return Reference{ID: ksuid.New()}
}

//XXX should be pool-key range
func (r Reference) Span() nano.Span {
	return nano.Span{Ts: r.First, Dur: 1}.Union(nano.Span{Ts: r.Last, Dur: 1})
}

func (r Reference) RowObjectName() string {
	return fmt.Sprintf("%s.zng", r.ID)
}

func (r Reference) RowObjectPath(path iosrc.URI) iosrc.URI {
	return path.AppendPath(r.RowObjectName())
}

func (r Reference) SeekObjectName() string {
	return fmt.Sprintf("%s-seek.zng", r.ID)
}

func (r Reference) SeekObjectPath(path iosrc.URI) iosrc.URI {
	return path.AppendPath(r.SeekObjectName())
}

func (r Reference) Range() string {
	//XXX need to handle any key... will the String method work?
	return fmt.Sprintf("[%d-%d]", r.First, r.Last)
}

// Remove deletes the row object and its seek index.
// Any 'not found' errors are ignored.
func (r Reference) Remove(ctx context.Context, path iosrc.URI) error {
	uris := []iosrc.URI{
		r.RowObjectPath(path), //XXX need ZNG and ZST
		r.SeekObjectPath(path),
	}
	for _, u := range uris {
		if err := iosrc.RemoveAll(ctx, u); err != nil && !zqe.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func Less(order zbuf.Order, a, b *Reference) bool {
	if order == zbuf.OrderDesc {
		a, b = b, a
	}
	//XXX need to handle arbitrary key type
	switch {
	case a.First != b.First:
		return a.First < b.First
	case a.Last != b.Last:
		return a.Last < b.Last
	case a.Count != b.Count:
		return a.Count < b.Count
	}
	// XXX shoulf we look at segID when Id's the same?
	// this happens when all the keys are the same and this shouldn't matter
	return ksuid.Compare(a.ID, b.ID) < 0
}

func Sort(order zbuf.Order, r []*Reference) {
	sort.Slice(r, func(i, j int) bool {
		return Less(order, r[i], r[j])
	})
}

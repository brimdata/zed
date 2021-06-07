package lakecli

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"strconv"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/segmentio/ksuid"
)

type Flags interface {
	SetFlags(fs *flag.FlagSet)
	Quiet() bool
	PoolName() string
	Create(ctx context.Context) (Root, error)
	Open(ctx context.Context) (Root, error)
	OpenPool(ctx context.Context) (Pool, error)
	CreatePool(ctx context.Context, layout order.Layout, thresh int64) (Pool, error)
}

type Root interface {
	AddIndex(context.Context, []index.Index) error
	CreatePool(context.Context, string, order.Layout, int64) (Pool, error)
	OpenPool(context.Context, ksuid.KSUID) (Pool, error)
	ScanPools(context.Context, zio.Writer) error
	LookupPoolByName(context.Context, string) (*lake.PoolConfig, error)
	RemovePool(context.Context, ksuid.KSUID) error
	DeleteIndices(context.Context, []ksuid.KSUID) ([]index.Index, error)
	LookupIndices(context.Context, []ksuid.KSUID) ([]index.Index, error)
	ScanIndex(context.Context, zio.Writer, []ksuid.KSUID) error
	Query(context.Context, driver.Driver, string) (zbuf.ScannerStats, error)
}

type Pool interface {
	Config() lake.PoolConfig
	// Add adds the contents of the reader to a pool. If the commit is not nil
	// the data will be committed to the pool.
	Add(ctx context.Context, r zio.Reader, commit *api.CommitRequest) (ksuid.KSUID, error)
	Delete(ctx context.Context, tags []ksuid.KSUID, commit *api.CommitRequest) (ksuid.KSUID, error)
	Commit(ctx context.Context, id ksuid.KSUID, commit api.CommitRequest) error
	ScanLog(ctx context.Context, w zio.Writer, head, tail journal.ID) error
	ScanStaging(ctx context.Context, w zio.Writer, ids []ksuid.KSUID) error
	ScanSegments(ctx context.Context, w zio.Writer, at string, partitions bool, span extent.Span) error
	Squash(ctx context.Context, ids []ksuid.KSUID, commit api.CommitRequest) (ksuid.KSUID, error)
	Index(ctx context.Context, rules []index.Index, ids []ksuid.KSUID) (ksuid.KSUID, error)
}

type baseFlags struct {
	quiet    bool
	poolName string
}

func (b *baseFlags) SetFlags(set *flag.FlagSet) {
	set.BoolVar(&b.quiet, "q", false, "quiet mode")
	set.StringVar(&b.poolName, "p", "", "name of pool")
}

func (b *baseFlags) Quiet() bool      { return b.quiet }
func (b *baseFlags) PoolName() string { return b.poolName }

func ParseID(s string) (ksuid.KSUID, error) {
	// Check if this is a cut-and-paste from ZNG, which encodes
	// the 20-byte KSUID as a 40 character hex string with 0x prefix.
	var id ksuid.KSUID
	if len(s) == 42 && s[0:2] == "0x" {
		b, err := hex.DecodeString(s[2:])
		if err != nil {
			return ksuid.Nil, fmt.Errorf("illegal hex tag: %s", s)
		}
		id, err = ksuid.FromBytes(b)
		if err != nil {
			return ksuid.Nil, fmt.Errorf("illegal hex tag: %s", s)
		}
	} else {
		var err error
		id, err = ksuid.Parse(s)
		if err != nil {
			return ksuid.Nil, fmt.Errorf("%s: invalid commit ID", s)
		}
	}
	return id, nil
}

func ParseIDs(ss []string) ([]ksuid.KSUID, error) {
	ids := make([]ksuid.KSUID, 0, len(ss))
	for _, s := range ss {
		id, err := ParseID(s)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func parseJournalID(ctx context.Context, pool *lake.Pool, at string) (journal.ID, error) {
	if num, err := strconv.Atoi(at); err == nil {
		ok, err := pool.IsJournalID(ctx, journal.ID(num))
		if err != nil {
			return journal.Nil, err
		}
		if ok {
			return journal.ID(num), nil
		}
	}
	commitID, err := ParseID(at)
	if err != nil {
		return journal.Nil, fmt.Errorf("not a valid journal number or a commit tag: %s", at)
	}
	id, err := pool.Log().JournalIDOfCommit(ctx, 0, commitID)
	if err != nil {
		return journal.Nil, fmt.Errorf("not a valid journal number or a commit tag: %s", at)
	}
	return id, nil
}

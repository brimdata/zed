package lakeflags

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"strconv"

	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/journal"
	"github.com/segmentio/ksuid"
)

type Flags struct {
	Quiet    bool
	PoolName string
}

func (f *Flags) SetFlags(fs *flag.FlagSet) {
	fs.BoolVar(&f.Quiet, "q", false, "quiet mode")
	fs.StringVar(&f.PoolName, "p", "", "name of pool")
}

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

//XXX this needs to go away

func ParseJournalID(ctx context.Context, pool *lake.Pool, at string) (journal.ID, error) {
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

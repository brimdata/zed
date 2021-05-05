package lake

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"strconv"

	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/zio"
	"github.com/segmentio/ksuid"
)

var Cmd = &charm.Spec{
	Name:  "lake",
	Usage: "lake [options] sub-command",
	Short: "create, manage, and search Zed lakes",
	Long: `
The "zed lake" command
operates on collections of Zed data files partitioned by and organized
by a specified key and stored either on a filesystem or an S3 compatible object store.

See the zed lake README in the zed repository for more information:
https://github.com/brimdata/zed/blob/main/docs/lake/README.md
`,
	New: New,
}

type Command struct {
	*root.Command
	Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	c.Flags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	_, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) == 0 {
		return charm.NeedHelp
	}
	return charm.ErrNoRun
}

func ParseKeys(s string) (field.List, bool) {
	if s == "" {
		return nil, false
	}
	return field.DottedList(s), true
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

func ParseIDs(args []string) ([]ksuid.KSUID, error) {
	ids := make([]ksuid.KSUID, 0, len(args))
	for _, s := range args {
		id, err := ParseID(s)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

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

func CopyToOutput(ctx context.Context, flags outputflags.Flags, r zio.Reader) error {
	w, err := flags.Open(ctx)
	if err != nil {
		return err
	}
	err = zio.Copy(w, r)
	if closeErr := w.Close(); err == nil {
		err = closeErr
	}
	return err
}

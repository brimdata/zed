package lake

import (
	"encoding/hex"
	"flag"
	"fmt"
	"strings"

	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/zbuf"
	"github.com/segmentio/ksuid"
)

var Cmd = &charm.Spec{
	Name:  "lake",
	Usage: "lake [global options] command [options] [arguments...]",
	Short: "create, manage, and search zed lakes",
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
	root.Command
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return &Command{}, nil
}

func (c *Command) Run(args []string) error {
	if len(args) == 0 {
		return charm.NeedHelp
	}
	return charm.ErrNoRun
}

func ParseKeys(s string) ([]field.Static, bool) {
	if s == "" {
		return nil, false
	}
	return field.DottedList(s), true
}

func ParseOrder(s string) (zbuf.Order, error) {
	switch strings.ToLower(s) {
	case "desc":
		return zbuf.OrderDesc, nil
	case "asc":
		return zbuf.OrderAsc, nil
	}
	return zbuf.OrderDesc, fmt.Errorf("unknown order parameter: %q", s)
}

func ParseIDs(args []string) ([]ksuid.KSUID, error) {
	ids := make([]ksuid.KSUID, 0, len(args))
	for _, s := range args {
		// Check if this is a cut-and-paste from ZNG, which encodes
		// the 20-byte KSUID as a 40 character hex string with 0x prefix.
		var id ksuid.KSUID
		if len(s) == 42 && s[0:2] == "0x" {
			b, err := hex.DecodeString(s[2:])
			if err != nil {
				return nil, fmt.Errorf("illegal hex tag: %s", s)
			}
			id, err = ksuid.FromBytes(b)
			if err != nil {
				return nil, fmt.Errorf("illegal hex tag: %s", s)
			}
		} else {
			var err error
			id, err = ksuid.Parse(s)
			if err != nil {
				return nil, fmt.Errorf("%s: invalid commit ID", s)
			}
		}
		ids = append(ids, id)
	}
	return ids, nil
}

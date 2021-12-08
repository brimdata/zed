package ls

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/segmentio/ksuid"
)

var Cmd = &charm.Spec{
	Name:  "ls",
	Usage: "ls [options] [pool]",
	Short: "list pools in a lake or branches in a pool",
	Long: `
"zed lake ls" shows a listing of a data pool's data objects as IDs.
If a pool name or pool ID is given, then the pool's branches are listed
along with the ID of their commit object, which points at the tip of each branch.
`,
	New: New,
}

type Command struct {
	*root.Command
	cli.LakeFlags
	partition   bool
	at          string
	outputFlags outputflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	c.outputFlags.DefaultFormat = "lake"
	c.outputFlags.SetFlags(f)
	c.LakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	var poolName string
	switch len(args) {
	case 0:
	case 1:
		poolName = args[0]
	default:
		return errors.New("too many arguments")
	}
	ctx, cleanup, err := c.Init(&c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	local := storage.NewLocalEngine()
	lake, err := c.Open(ctx)
	if err != nil {
		return err
	}
	var query string
	if poolName == "" {
		query = "from :pools"
	} else {
		if strings.IndexByte(poolName, '\'') >= 0 {
			return errors.New("pool name may not contain quote characters")
		}
		query = fmt.Sprintf("from '%s':branches", poolName)
	}
	//XXX at should be a date/time
	var at ksuid.KSUID
	if c.at != "" {
		at, err = ksuid.Parse(c.at)
		if err != nil {
			return err
		}
		query = fmt.Sprintf("%s at %s", query, at)
	}
	zw, err := c.outputFlags.Open(ctx, local)
	if err != nil {
		return err
	}
	_, err = lake.Query(ctx, driver.NewCLI(zw), nil, query)
	if closeErr := zw.Close(); err == nil {
		err = closeErr
	}
	return err
}

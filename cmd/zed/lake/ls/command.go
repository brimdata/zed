package ls

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/lakeflags"
	"github.com/brimdata/zed/cli/outputflags"
	zedapi "github.com/brimdata/zed/cmd/zed/api"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/segmentio/ksuid"
)

var Ls = &charm.Spec{
	Name:  "ls",
	Usage: "ls [options]",
	Short: "list objects in data pool",
	Long: `
"zed lake ls" shows a listing of a data pool's segments as tags.
The the pool flag "-p" is not given, then the lake's pools are listed.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Ls)
	zedapi.Cmd.Add(Ls)
}

type Command struct {
	lake        zedlake.Command
	partition   bool
	at          string
	outputFlags outputflags.Flags
	lakeFlags   lakeflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(zedlake.Command)}
	f.StringVar(&c.at, "at", "", "commit tag for time travel")
	f.BoolVar(&c.partition, "partition", false, "display partitions as determined by scan logic")
	c.outputFlags.DefaultFormat = "lake"
	c.outputFlags.SetFlags(f)
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	if len(args) > 0 {
		return errors.New("zed lake ls: too many arguments")
	}
	ctx, cleanup, err := c.lake.Root().Init(&c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	local := storage.NewLocalEngine()
	lake, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	var query string
	if c.lakeFlags.PoolName == "" {
		query = "from [pools]"
	} else {
		query = fmt.Sprintf("from '%s'[objects]", c.lakeFlags.PoolName)
	}
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
	_, err = lake.Query(ctx, driver.NewCLI(zw), query)
	if closeErr := zw.Close(); err == nil {
		err = closeErr
	}
	return err
}

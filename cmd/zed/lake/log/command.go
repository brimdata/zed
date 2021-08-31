package log

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/brimdata/zed/cli/lakeflags"
	"github.com/brimdata/zed/cli/outputflags"
	zedapi "github.com/brimdata/zed/cmd/zed/api"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
)

var Log = &charm.Spec{
	Name:  "log",
	Usage: "log [options] [pattern]",
	Short: "show a data pool's commit log",
	Long: `
"zed lake log" outputs a data pool's commit log in the format desired.
By default, output is in the ZNG format so that it can easily be piped
to zq or other tooling for analysis.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Log)
	zedapi.Cmd.Add(Log)
}

type Command struct {
	lake        zedlake.Command
	outputFlags outputflags.Flags
	lakeFlags   lakeflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(zedlake.Command)}
	c.outputFlags.DefaultFormat = "lake"
	c.outputFlags.SetFlags(f)
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	if c.lakeFlags.PoolName == "" {
		return errors.New("pool must be specified")
	}
	ctx, cleanup, err := c.lake.Root().Init(&c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) != 0 {
		return errors.New("zed lake load: no arguments allowed")
	}
	lake, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	w, err := c.outputFlags.Open(ctx, storage.NewLocalEngine())
	if err != nil {
		return err
	}
	defer w.Close()
	poolName, branchName := c.lakeFlags.Branch()
	if strings.IndexByte(poolName, '\'') >= 0 || strings.IndexByte(branchName, '\'') >= 0 {
		return errors.New("pool name may not contain quote characters")
	}
	query := fmt.Sprintf("from '%s'/'%s'[log]", poolName, branchName)
	_, err = lake.Query(ctx, driver.NewCLI(w), query)
	if closeErr := w.Close(); err == nil {
		err = closeErr
	}
	return err
}

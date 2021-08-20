package init

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/lakeflags"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/lake/api"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
)

var Init = &charm.Spec{
	Name:  "init",
	Usage: "create and initialize a new, empty lake",
	Short: "init [ path ]",
	Long: `
"zed lake init" ...
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Init)
}

type Command struct {
	lake      *zedlake.LocalCommand
	lakeFlags lakeflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(*zedlake.LocalCommand)}
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Root().Init()
	if err != nil {
		return err
	}
	defer cleanup()
	var path string
	if len(args) == 0 {
		path = zedlake.DefaultRoot()
		if path != "" && !c.lakeFlags.Quiet {
			fmt.Printf("using environment variable %s\n", zedlake.RootEnv)
		}
	} else if len(args) == 1 {
		path = args[0]
	}
	if path == "" {
		return errors.New("single lake path argument required")
	}
	lakePath, err := storage.ParseURI(path)
	if err != nil {
		return err
	}
	if _, err := api.CreateLocalLake(ctx, lakePath); err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		fmt.Printf("lake created: %s\n", path)
	}
	return nil
}

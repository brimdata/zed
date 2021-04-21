package init

import (
	"errors"
	"flag"
	"fmt"

	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
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
	lake *zedlake.Command
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(*zedlake.Command)}
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	var path string
	if len(args) == 0 {
		path = zedlake.DefaultRoot()
		if path != "" && !c.lake.Flags.Quiet {
			fmt.Printf("using environment variable %s\n", zedlake.RootEnv)
		}
	} else if len(args) == 1 {
		path = args[0]
	}
	if path == "" {
		return errors.New("zed lake create lake: requires a single lake path argument")
	}
	c.lake.Flags.Root = path
	if _, err := c.lake.Flags.Create(ctx); err != nil {
		return err
	}
	if !c.lake.Flags.Quiet {
		name, _ := c.lake.Flags.RootPath()
		fmt.Printf("lake created: %s\n", name)
	}
	return nil
}

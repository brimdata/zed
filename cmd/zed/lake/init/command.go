package init

import (
	"context"
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
	*zedlake.Command
	quiet     bool
	lakeFlags zedlake.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*zedlake.Command)}
	f.BoolVar(&c.quiet, "q", false, "quiet mode")
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx := context.TODO()
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}
	var path string
	if len(args) == 0 {
		path = zedlake.DefaultRoot()
		if path != "" && !c.quiet {
			fmt.Printf("using environment variable %s\n", zedlake.RootEnv)
		}
	} else if len(args) == 1 {
		path = args[0]
	}
	if path == "" {
		return errors.New("zed lake create lake: requires a single lake path argument")
	}
	c.lakeFlags.Root = path
	if _, err := c.lakeFlags.Create(ctx); err != nil {
		return err
	}
	if !c.quiet {
		name, _ := c.lakeFlags.RootPath()
		fmt.Printf("lake created: %s\n", name)
	}
	return nil
}

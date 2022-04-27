package init

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/lake/api"
	"github.com/brimdata/zed/pkg/charm"
)

var Cmd = &charm.Spec{
	Name:  "init",
	Usage: "create and initialize a new, empty lake",
	Short: "init [ path ]",
	Long: `
"zed init" ...
`,
	New: New,
}

type Command struct {
	*root.Command
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	var path string
	if len(args) == 0 {
		path = c.Lake
	} else if len(args) == 1 {
		path = args[0]
	}
	if path == "" {
		return errors.New("single lake path argument required")
	}
	if api.IsLakeService(path) {
		return fmt.Errorf("init command not valid on remote lake")
	}
	if _, err := api.CreateLocalLake(ctx, path); err != nil {
		return err
	}
	if !c.Quiet {
		fmt.Printf("lake created: %s\n", path)
	}
	return nil
}

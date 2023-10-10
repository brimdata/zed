package init

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/lake/api"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
	"go.uber.org/zap"
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
	return &Command{Command: parent.(*root.Command)}, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	var u *storage.URI
	if len(args) == 0 {
		if u, err = c.LakeFlags.URI(); err != nil {
			return err
		}
	} else if len(args) == 1 {
		path := args[0]
		if path == "" {
			return errors.New("single lake path argument required")
		}
		if u, err = storage.ParseURI(path); err != nil {
			return err
		}
	}
	if api.IsLakeService(u.String()) {
		return fmt.Errorf("init command not valid on remote lake")
	}
	if _, err := api.CreateLocalLake(ctx, zap.Must(zap.NewProduction()), u.String()); err != nil {
		return err
	}
	if !c.LakeFlags.Quiet {
		fmt.Printf("lake created: %s\n", u)
	}
	return nil
}

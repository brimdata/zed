package repl

import (
	"flag"
	"fmt"
	"io"

	"github.com/brimdata/zed/api"
	apicmd "github.com/brimdata/zed/cmd/zed/api"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/repl"
	"github.com/brimdata/zed/pkg/units"
)

var Repl = &charm.Spec{
	Name:  "repl",
	Usage: "repl [flags]",
	Short: "enter read-eval-print loop",
	Long: `The repl command takes a single argument and creates a new, empty space
named as specified.`,
	New: New,
}

func init() {
	apicmd.Cmd.Add(Repl)
}

type Command struct {
	*apicmd.Command
	kind     api.StorageKind
	datapath string
	thresh   units.Bytes
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{
		Command: parent.(*apicmd.Command),
	}
	return c, nil
}

func (c *Command) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}
	// do not enter repl if space is not selected
	if _, err := c.SpaceID(); err != nil {
		return err
	}
	err := repl.Run(c)
	if err == io.EOF {
		fmt.Println("")
		err = nil
	}
	return err
}

package repl

import (
	"flag"
	"fmt"
	"io"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/cmd/zapi/cmd"
	"github.com/brimsec/zq/pkg/repl"
	"github.com/brimsec/zq/pkg/units"
	"github.com/brimsec/zq/pkg/charm"
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
	cmd.CLI.Add(Repl)
}

type Command struct {
	*cmd.Command
	kind     api.StorageKind
	datapath string
	thresh   units.Bytes
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{
		Command: parent.(*cmd.Command),
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

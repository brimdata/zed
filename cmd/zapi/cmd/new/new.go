package new

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimsec/zq/cmd/zapi/cmd"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/storage"
	"github.com/mccanne/charm"
)

var New = &charm.Spec{
	Name:  "new",
	Usage: "new [spacename]",
	Short: "create a new space",
	Long: `The new command takes a single argument and creates a new, empty space
named as specified.`,
	New: NewFn,
}

func init() {
	cmd.CLI.Add(New)
}

type NewCommand struct {
	*cmd.Command
	kind     storage.Kind
	datapath string
}

func NewFn(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &NewCommand{
		Command: parent.(*cmd.Command),
		kind:    storage.FileStore,
	}
	f.Var(&c.kind, "k", "kind of storage for this space")
	f.StringVar(&c.datapath, "d", "", "specific directory for storage data")
	return c, nil
}

func (c *NewCommand) Run(args []string) error {
	if len(args) > 1 {
		return errors.New("too many arguments")
	}
	if len(args) != 1 {
		return errors.New("must specify a space name")
	}
	client := c.Client()
	name := args[0]
	req := api.SpacePostRequest{
		Name:     name,
		DataPath: c.datapath,
		Storage: &storage.Config{
			Kind: storage.Kind(c.kind),
		},
	}
	if _, err := client.SpacePost(c.Context(), req); err != nil {
		return fmt.Errorf("couldn't create new space %s: %v", name, err)
	}
	fmt.Printf("%s: space created\n", name)
	return nil
}

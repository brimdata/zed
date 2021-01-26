package newsubspace

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/cmd/zapi/cmd"
	"github.com/mccanne/charm"
)

var NewSubspace = &charm.Spec{
	Name:  "newsubspace",
	Usage: "newsubspace -p parent_space_id -n subspace_name log1 [log2 ...]",
	Short: "create a new subspace",
	Long:  `Creates a subspace of the given logs from the parent space.`,
	New:   New,
}

func init() {
	cmd.CLI.Add(NewSubspace)
}

type Command struct {
	*cmd.Command
	parentID string
	name     string
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*cmd.Command)}
	f.StringVar(&c.parentID, "p", "", "id of parent space")
	f.StringVar(&c.name, "n", "", "name for subspace")
	return c, nil
}

func (c *Command) Run(args []string) error {
	if len(args) == 0 {
		return errors.New("must specify at least one log from parent")
	}
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}
	conn := c.Connection()
	req := api.SubspacePostRequest{
		Name: c.name,
		OpenOptions: api.ArchiveOpenOptions{
			LogFilter: args,
		},
	}
	if _, err := conn.SubspacePost(c.Context(), api.SpaceID(c.parentID), req); err != nil {
		return fmt.Errorf("couldn't create subspace %s: %w", c.name, err)
	}
	fmt.Printf("%s: subspace created\n", c.name)
	return nil
}

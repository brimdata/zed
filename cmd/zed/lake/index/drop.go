package index

import (
	"errors"
	"flag"
	"fmt"

	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
)

var Drop = &charm.Spec{
	Name:  "drop",
	Usage: "drop [-R root] [options] id... ",
	Short: "drop index rule(s) from a lake",
	New:   NewDrop,
}

type DropCommand struct {
	lake *zedlake.Command
}

func NewDrop(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &DropCommand{lake: parent.(*Command).Command}
	return c, nil
}

func (c *DropCommand) Run(args []string) error {
	ctx, cleanup, err := c.lake.Init()
	if err != nil {
		return err
	}
	defer cleanup()

	if len(args) == 0 {
		return errors.New("must specify one or more xrule tags")
	}

	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}

	ids, err := zedlake.ParseIDs(args)
	if err != nil {
		return err
	}

	root, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}

	xrules, err := root.DeleteXRules(ctx, ids)
	if err != nil {
		return err
	}

	if !c.lake.Quiet {
		for _, xrule := range xrules {
			fmt.Printf("%s dropped\n", xrule.ID)
		}
	}

	return nil
}

/* NOT YET
func (c *DropCommand) run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}

	if len(args) == 0 {
		return errors.New("no index definition specified")
	}

	lk, err := lake.OpenLake(c.root, nil)
	if err != nil {
		return err
	}

	alldefs, err := lk.ReadDefinitions(context.TODO())
	if err != nil {
		return err
	}

	defs := make([]*index.Definition, 0, len(args))
	for _, arg := range args {
		id, err := ksuid.Parse(arg)
		if err != nil {
			return err
		}
		def := alldefs.Lookup(id)
		if def == nil {
			fmt.Fprintf(os.Stderr, "defintion for id '%s' not found\n", arg)
			continue
		}
		defs = append(defs, def)
	}

	if len(defs) == 0 {
		return errors.New("no definitions deleted")
	}

	if err := lake.RemoveDefinitions(context.TODO(), lk, defs...); err != nil {
		return err
	}

	if !c.noapply {
		if !c.quiet {
			c.progress = make(chan string)
			go c.displayProgress()
		}
		if err := lake.RemoveIndices(context.TODO(), lk, c.progress, defs...); err != nil {
			return err
		}
	}
	fmt.Printf("%d index definitions removed\n", len(defs))
	return nil
}

func (c *DropCommand) displayProgress() {
	for line := range c.progress {
		fmt.Println(line)
	}
}
*/

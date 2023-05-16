package vacuum

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/plural"
)

var Cmd = &charm.Spec{
	Name:  "vacuum",
	Usage: "vacuum [options]",
	Short: "clear space by removing unreferenced objects",
	Long: `
"zed vacuum" clears up space in a pool by removing objects that are not visible 
from a pool's branch or commit.

DANGER ZONE:
Running this command will permanently delete objects referenced in 
previous commits causing missing data when time traveling to those commits.
`,
	New: New,
}

type Command struct {
	*root.Command
	force  bool
	dryrun bool
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.BoolVar(&c.force, "f", false, "do not prompt for confirmation")
	f.BoolVar(&c.dryrun, "dryrun", false, "vacuum without deleting objects")
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	at, err := c.LakeFlags.HEAD()
	if err != nil {
		return err
	}
	lk, err := c.LakeFlags.Open(ctx)
	if err != nil {
		return err
	}
	if c.dryrun {
		oids, err := lk.Vacuum(ctx, at.Pool, at.Branch, true)
		if err != nil {
			return err
		}
		// It's a bit weird to specify -q and -dryrun, but do as they say.
		if !c.LakeFlags.Quiet {
			fmt.Printf("would vacuum %d object%s\n", len(oids), plural.Slice(oids, "s"))
		}
		return nil
	}
	if err := c.confirm(at.String()); err != nil {
		return err
	}
	oids, err := lk.Vacuum(ctx, at.Pool, at.Branch, false)
	if err != nil {
		return err
	}
	if !c.LakeFlags.Quiet {
		fmt.Printf("vacuumed %d object%s\n", len(oids), plural.Slice(oids, "s"))
	}
	return nil
}

func (c *Command) confirm(name string) error {
	if c.force {
		return nil
	}
	fmt.Printf("Are you sure you want to vacuum data objects from %q? There is no going back... [y|n]\n", name)
	var input string
	if _, err := fmt.Scanln(&input); err != nil {
		return err
	}
	input = strings.ToLower(input)
	if input == "y" || input == "yes" {
		return nil
	}
	return errors.New("operation canceled")
}

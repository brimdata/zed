package branch

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/brimdata/zed/cli/lakeflags"
	zedapi "github.com/brimdata/zed/cmd/zed/api"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/compiler/parser"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/segmentio/ksuid"
)

var Branch = &charm.Spec{
	Name:  "branch",
	Usage: "branch [-at tag] -p pool[/branch] branch",
	Short: "create a new branch",
	Long: `
The lake branch command creates a new branch with the indicated name.

The -p option is required and specifies the name for the pool and
the branch.  If an existing branch name is not included in the -p option,
then "main" is assumed.

The -at option specifies a commit tag in the parent at which the branch
is formed.  If absent, the tip of the parent branch is assumed.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Branch)
	zedapi.Cmd.Add(Branch)
}

type Command struct {
	lake      zedlake.Command
	at        string
	lakeFlags lakeflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(zedlake.Command)}
	f.StringVar(&c.at, "at", "", "commit tag in parent to use as branch point")
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Root().Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) != 1 {
		return errors.New("branch name must be specified")
	}
	branch := args[0]
	poolName, parent := c.lakeFlags.Branch()
	if poolName == "" {
		return errors.New("a pool must be specified with -p")
	}
	var at ksuid.KSUID
	if c.at != "" {
		at, err = parser.ParseID(c.at)
		if err != nil {
			return err
		}
	}
	lake, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	poolID, _ := parser.ParseID(poolName)
	parentID, _ := parser.ParseID(parent)
	if poolID == ksuid.Nil || parentID == ksuid.Nil {
		poolID, parentID, err = lake.IDs(ctx, poolName, parent)
		if err != nil {
			return err
		}
	}
	id, err := lake.CreateBranch(ctx, poolID, branch, parentID, at)
	if err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		fmt.Printf("branch created: %s %s\n", branch, id)
	}
	return nil
}

func parseParent(in string) (string, ksuid.KSUID, error) {
	list := strings.Split(in, "@")
	switch len(list) {
	case 1:
		return in, ksuid.Nil, nil
	case 2:
		id, err := parser.ParseID(list[1])
		if err != nil {
			return "", ksuid.Nil, errors.New("branch address (following '@') from -in option is not a tag")
		}
		return list[0], id, nil
	}
	return "", ksuid.Nil, errors.New("-in option can have only one branch address ('@')")
}

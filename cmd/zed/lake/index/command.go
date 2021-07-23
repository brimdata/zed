package index

import (
	"flag"

	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
)

var Index = &charm.Spec{
	Name:  "index",
	Usage: "index [subcommand]",
	Short: "create and drop indices, index data",
	Long: `
The index subcommands control the creation, management, and deletion
of search indexes in a Zed lake.  Unlike traditional approaches to search
based on a consolidated inverted index pointing at documents, Zed indexes
are highly modular where each index rule creates an index object per data obejct.
Each index is incrementally built and transactionally attached to
its immutable data object.  While objects are indexed, queries may run with
or without the presence of such indexes.  The planner uses an object's indexes
to prune the object from the query when possible, making queries run faster
when an object's index indicated that it can be skipped.

The Zed lake service does not automatically apply index rules to newly
loaded data.  Instead, the creation of indexes is driven by agents
external to the service, e.g., different orchestration logic can be deployed
for different workloads allowing out-of-order data, for instance, to
"settle down" and be "rolled up" before being indexed.

Index rules are organized into name sets so a set of indexes can be easily
applied to one or more data objects with the "index apply" command.
Once an rule set has been applied to an object, any changes to the named rules
do not have an immediate effect and the named set
must be reapplied.  In this case, "index apply" creates only the needed
new index objects to reflect the change.
`, New: New,
}

func init() {
	Index.Add(Apply)
	Index.Add(Create)
	Index.Add(Drop)
	Index.Add(Ls)
	zedlake.Cmd.Add(Index)
}

type Command struct {
	*zedlake.Command
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return &Command{parent.(*zedlake.Command)}, nil
}

func (c *Command) Run(args []string) error {
	if len(args) == 0 {
		return charm.NeedHelp
	}
	return charm.ErrNoRun
}

package lake

import (
	"flag"

	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/pkg/charm"
)

var Cmd = &charm.Spec{
	Name:  "lake",
	Usage: "lake [global options] command [options] [arguments...]",
	Short: "create and search zed lakes",
	Long: `
The "zed lake" command
operates on collections of ZNG files partitioned by time and stored either
on a filesystem or an S3 compatible object store. An individual
item of data (a file or object) is called a chunk, and each chunk may have
other named ZNG files associated with it, stored "near" to the chunk. For
filesystem archives, the associated files are stored in a directory next
to the chunk file.

An example of a chunk associated file is a micro-index: a ZNG file that holds
keyed records and supports very fast lookup of keys. When the key represents
a value in the associated chunk file, micro-indexes can be used to to make
searching an archive very fast.

See the zed lake README in the zed repository for more information:
https://github.com/brimdata/zed/blob/main/docs/lake/README.md
`,
	New: New,
}

type Command struct {
	charm.Command
	cli cli.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{}
	c.cli.SetFlags(f)
	return c, nil
}

func (c *Command) Cleanup() {
	c.cli.Cleanup()
}

func (c *Command) Init(all ...cli.Initializer) error {
	return c.cli.Init(all...)
}

func (c *Command) Run(args []string) error {
	if len(args) == 0 {
		return charm.NeedHelp
	}
	return charm.ErrNoRun
}

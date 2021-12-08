package inspect

import (
	"errors"
	"flag"
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/cli/outputflags"
	devzst "github.com/brimdata/zed/cmd/zed/dev/zst"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zst"
)

var Cut = &charm.Spec{
	Name:  "cut",
	Usage: "cut [flags] -k field-expr path",
	Short: "cut a column from a zst file",
	Long: `
The cut command cuts a single column from a zst file and writes the column
to the output in the format of choice.

This command is most useful for test, debug, and demo, as more efficient
and complete "cuts" on zst files will eventually be available from zq
in the future.  For example, zq cut will optmize the query

	count() by _path

to cut the path field and run analytics directly on the result without having
to scan all of the zng row data.
`,
	New: newCommand,
}

func init() {
	devzst.Cmd.Add(Cut)
}

type Command struct {
	*devzst.Command
	outputFlags outputflags.Flags
	fieldExpr   string
}

func newCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*devzst.Command)}
	f.StringVar(&c.fieldExpr, "k", "", "dotted field expression of field to cut")
	c.outputFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init(&c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) != 1 {
		return errors.New("zst cut: must be run with a single input file")
	}
	if c.fieldExpr == "" {
		return errors.New("zst cut: must specify field to cut with -k")
	}
	fields := strings.Split(c.fieldExpr, ".")
	path := args[0]
	local := storage.NewLocalEngine()
	cutter, err := zst.NewCutterFromPath(ctx, zed.NewContext(), local, path, fields)
	if err != nil {
		return err
	}
	defer cutter.Close()
	writer, err := c.outputFlags.Open(ctx, local)
	if err != nil {
		return err
	}
	if err := zio.Copy(writer, cutter); err != nil {
		writer.Close()
		return err
	}
	return writer.Close()
}

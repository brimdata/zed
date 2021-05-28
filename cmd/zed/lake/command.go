package lake

import (
	"context"
	"flag"

	"github.com/brimdata/zed/cli/lakecli"
	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
)

var Cmd = &charm.Spec{
	Name:  "lake",
	Usage: "lake [options] sub-command",
	Short: "create, manage, and search Zed lakes",
	Long: `
The "zed lake" command
operates on collections of Zed data files partitioned by and organized
by a specified key and stored either on a filesystem or an S3 compatible object store.

See the zed lake README in the zed repository for more information:
https://github.com/brimdata/zed/blob/main/docs/lake/README.md
`,
	New: New,
}

type Command struct {
	*root.Command
	lakecli.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	c.Flags = lakecli.NewLocalFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	_, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) == 0 {
		return charm.NeedHelp
	}
	return charm.ErrNoRun
}

func ParseKeys(s string) (field.List, bool) {
	if s == "" {
		return nil, false
	}
	return field.DottedList(s), true
}

func CopyToOutput(ctx context.Context, engine storage.Engine, flags outputflags.Flags, r zio.Reader) error {
	w, err := flags.Open(ctx, engine)
	if err != nil {
		return err
	}
	err = zio.Copy(w, r)
	if closeErr := w.Close(); err == nil {
		err = closeErr
	}
	return err
}

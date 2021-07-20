package lake

import (
	"context"
	"errors"
	"flag"
	"os"

	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/lake/api"
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

type Command interface {
	Open(context.Context) (api.Interface, error)
	Root() *root.Command
}

var _ Command = (*LocalCommand)(nil)

type LocalCommand struct {
	*root.Command
	Path string
}

const RootEnv = "ZED_LAKE_ROOT"

func DefaultRoot() string {
	return os.Getenv(RootEnv)
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &LocalCommand{Command: parent.(*root.Command)}
	f.StringVar(&c.Path, "R", DefaultRoot(), "URI of path to Zed lake store")
	return c, nil
}

func (c *LocalCommand) Root() *root.Command {
	return c.Command
}

func (c *LocalCommand) Open(ctx context.Context) (api.Interface, error) {
	if c.Path == "" {
		return nil, errors.New("no lake path specied: use -R or set ZED_LAKE_ROOT")
	}
	path, err := storage.ParseURI(c.Path)
	if err != nil {
		return nil, err
	}
	return api.OpenLocalLake(ctx, path)
}

func (c *LocalCommand) Run(args []string) error {
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

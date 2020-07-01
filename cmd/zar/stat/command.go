package stat

import (
	"context"
	"errors"
	"flag"
	"os"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/cmd/zar/root"
	"github.com/brimsec/zq/emitter"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/mccanne/charm"
)

var Stat = &charm.Spec{
	Name:  "stat",
	Usage: "stat [options]",
	Short: "archive component statistics",
	Long: `
"zar stat" generates a ZNG stream with information about the chunks in
an archive.
`,
	New: New,
}

func init() {
	root.Zar.Add(Stat)
}

type Command struct {
	*root.Command
	root   string
	format string
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.root, "R", os.Getenv("ZAR_ROOT"), "root directory of zar archive to walk")
	f.StringVar(&c.format, "f", "table", "output format")
	return c, nil
}

func (c *Command) Run(args []string) (err error) {
	if len(args) > 0 {
		return errors.New("zar stat: too many arguments")
	}

	ark, err := archive.OpenArchive(c.root, nil)
	if err != nil {
		return err
	}

	wc, err := emitter.NewFile("", &zio.WriterFlags{Format: c.format})
	if err != nil {
		return err
	}
	defer func() {
		wcErr := wc.Close()
		if err == nil {
			err = wcErr
		}
	}()

	rc, err := archive.Stat(context.Background(), ark)
	if err != nil {
		return err
	}
	defer func() {
		rcErr := rc.Close()
		if err == nil {
			err = rcErr
		}
	}()

	return zbuf.Copy(wc, rc)
}

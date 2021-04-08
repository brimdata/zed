package stat

import (
	"context"
	"errors"
	"flag"
	"os"

	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/emitter"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
)

var Stat = &charm.Spec{
	Name:  "stat",
	Usage: "stat [-R root] [options]",
	Short: "archive component statistics",
	Long: `
"zar stat" generates a ZNG stream with information about the chunks in
an archive.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Stat)
}

type Command struct {
	*zedlake.Command
	root   string
	format string
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*zedlake.Command)}
	f.StringVar(&c.root, "R", os.Getenv("ZED_LAKE_ROOT"), "root location of zar archive to walk")
	f.StringVar(&c.format, "f", "table", "format for output data [zng,ndjson,table,text,zeek,zjson,tzng] (default \"table\")")
	return c, nil
}

func (c *Command) Run(args []string) (err error) {
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}
	if len(args) > 0 {
		return errors.New("zar stat: too many arguments")
	}

	lk, err := lake.OpenLake(c.root, nil)
	if err != nil {
		return err
	}

	wc, err := emitter.NewFile(context.TODO(), "", zio.WriterOpts{Format: c.format})
	if err != nil {
		return err
	}
	defer func() {
		wcErr := wc.Close()
		if err == nil {
			err = wcErr
		}
	}()

	rc, err := lake.Stat(context.Background(), zson.NewContext(), lk)
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

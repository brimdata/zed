package stat

import (
	"errors"
	"flag"

	"github.com/brimdata/zed/cli/outputflags"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
)

var Stat = &charm.Spec{
	Name:  "stat",
	Usage: "stat [options]",
	Short: "zed lake statistics",
	Long: `
"zar lake stat" generates a ZNG stream with information about the data in
a Zed lake.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Stat)
}

type Command struct {
	*zedlake.Command
	lakeFlags   zedlake.Flags
	outputFlags outputflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*zedlake.Command)}
	c.lakeFlags.SetFlags(f)
	c.outputFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) (err error) {
	// replace this with zed lake log?   which can also output zng
	// all the info this used to print is available now in the log
	return errors.New("TBD: Issue #2534")
	/*
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
	*/
}

package index

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/brimsec/zq/cmd/zar/root"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/bzngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/mccanne/charm"
)

var Chop = &charm.Spec{
	Name:  "chop",
	Usage: "chop [options] file",
	Short: "chop bzng files into pieces",
	Long: `
	TBD
`,
	New: New,
}

func init() {
	root.Zar.Add(Chop)
}

type Command struct {
	*root.Command
	size int
	dir  string
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.dir, "d", ".", "directory in which to chop files")
	f.IntVar(&c.size, "s", 500, "target size of chopped files in MB")
	return c, nil
}

func tsDir(ts nano.Ts) string {
	year, month, day := ts.Time().Date()
	return fmt.Sprintf("%d%02d%02d", year, month, day)
}

func (c *Command) Run(args []string) error {
	if len(args) != 1 {
		return errors.New("zar chop: exactly one input file must be specified (- for stdin)")
	}
	var file *os.File
	filename := args[0]
	if filename == "-" {
		file = os.Stdin
	} else {
		var err error
		file, err = os.Open(filename)
		if err != nil {
			return err
		}
		defer file.Close()
	}
	r := bzngio.NewReader(file, resolver.NewContext())
	var w *bufio.Writer
	var zw zbuf.Writer
	var out *os.File
	var n int
	thresh := c.size * 1024 * 1024
	for {
		rec, err := r.Read()
		if err != nil || rec == nil {
			if out != nil {
				if err := w.Flush(); err != nil {
					return err
				}
				if err := out.Close(); err != nil {
					return err
				}
			}
			return err
		}
		if w == nil {
			ts := rec.Ts
			dir := filepath.Join(c.dir, tsDir(ts))
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			path := filepath.Join(dir, ts.StringFloat()+".bzng")
			//XXX for now just truncate any existing file.
			// a future PR will do a split/merge.
			out, err = os.OpenFile(path, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				return err
			}
			fmt.Printf("writing %s\n", path)
			w = bufio.NewWriter(out)
			zw = bzngio.NewWriter(out, zio.Flags{})
		}
		if err := zw.Write(rec); err != nil {
			return err
		}
		n += len(rec.Raw)
		if n >= thresh {
			if err := w.Flush(); err != nil {
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
			w = nil
			n = 0
		}
	}
}

package section

import (
	"errors"
	"flag"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/cli/outputflags"
	zedindex "github.com/brimdata/zed/cmd/zed/index"
	"github.com/brimdata/zed/index"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
)

var Section = &charm.Spec{
	Name:  "section",
	Usage: "section [flags] path",
	Short: "extract a section of a zed index file",
	Long: `
The section command extracts a section from a zed index file and
writes it to the output.  The -trailer option writes
the zed index trailer to the output in addition to the section if the section
number was specified.

See the "zed index" command help for a description of a zed index file.`,
	New: newCommand,
}

func init() {
	zedindex.Cmd.Add(Section)
}

type Command struct {
	*zedindex.Command
	outputFlags outputflags.Flags
	trailer     bool
	section     int
}

func newCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*zedindex.Command)}
	f.BoolVar(&c.trailer, "trailer", false, "include the zed index trailer in the output")
	f.IntVar(&c.section, "s", -1, "include the indicated section in the output")
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
		return errors.New("zed index section: must be run with a single path argument")
	}
	path := args[0]
	local := storage.NewLocalEngine()
	reader, err := index.NewReader(zed.NewContext(), local, path)
	if err != nil {
		return err
	}
	defer reader.Close()
	writer, err := c.outputFlags.Open(ctx, local)
	if err != nil {
		return err
	}
	defer func() {
		if writer != nil {
			writer.Close()
		}
	}()
	if c.section >= 0 {
		r, err := reader.NewSectionReader(c.section)
		if err != nil {
			return err
		}
		if err := zio.Copy(writer, r); err != nil {
			return err
		}
	}
	if c.trailer {
		r, err := reader.NewTrailerReader()
		if err != nil {
			return err
		}
		if err := zio.Copy(writer, r); err != nil {
			return err
		}
	}
	err = writer.Close()
	writer = nil
	return err
}

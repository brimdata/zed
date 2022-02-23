package section

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cmd/zed/dev/dig"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
)

var Section = &charm.Spec{
	Name:  "section",
	Usage: "section [flags] number file",
	Short: "extract a section of a sectioned Zed file",
	Long: `
The section command takes an integer section number and a file argument
(which must be a sectioned Zed file having a Zed trailer),
extracts the requested section of the file (where the section must be encoded
in the ZNG format) and outputs the section in any Zed format.`,
	New: newCommand,
}

func init() {
	dig.Cmd.Add(Section)
}

type Command struct {
	*dig.Command
	outputFlags outputflags.Flags
	trailer     bool
	section     int
}

func newCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*dig.Command)}
	c.outputFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init(&c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) != 2 {
		return errors.New("two arguments required")
	}
	uri, err := storage.ParseURI(args[1])
	if err != nil {
		return err
	}
	engine := storage.NewLocalEngine()
	r, err := engine.Get(ctx, uri)
	if err != nil {
		return err
	}
	defer r.Close()
	size, err := storage.Size(r)
	if err != nil {
		return err
	}
	trailer, err := zngio.ReadTrailer(r, size)
	if err != nil {
		return err
	}
	which, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("bad section number: %w", err)
	}
	reader, err := newSectionReader(r, which, trailer.Sections)
	if err != nil {
		return err
	}
	defer reader.Close()
	writer, err := c.outputFlags.Open(ctx, engine)
	if err != nil {
		return err
	}
	if err := zio.Copy(writer, reader); err != nil {
		writer.Close()
		return err
	}
	return writer.Close()
}

func newSectionReader(r io.ReaderAt, which int, sections []int64) (*zngio.Reader, error) {
	if which >= len(sections) {
		return nil, fmt.Errorf("section %d does not exist", which)
	}
	off := int64(0)
	var k int
	for ; k < which; k++ {
		off += sections[k]
	}
	reader := io.NewSectionReader(r, off, sections[which])
	return zngio.NewReader(reader, zed.NewContext()), nil
}

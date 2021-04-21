package create

import (
	"context"
	"errors"
	"flag"

	"github.com/brimdata/zed/cli/inputflags"
	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cmd/zed/zst"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zson"
)

var Create = &charm.Spec{
	Name:  "create",
	Usage: "create [-coltresh thresh] [-skewthresh thesh] -o file files...",
	Short: "create a zst columnar object from a zng file or stream",
	Long: `
The create command generates a columnar zst object from a zng input stream,
which may be stdin or one or more zng storage objects (local files or s3 objects).
The output can be a local file or an s3 URI.

The -colthresh flag specifies the byte threshold (in MiB) at which chunks
of column data are written to disk.

The -skewthresh flag specifies a rough byte threshold (in MiB) that controls
how much column data is collectively buffered in memory before being entirely
flushed to disk.  This parameter controls the amount of buffering "skew" required
keep rows in alignment so that a reader should not have to use more than
this (approximate) memory footprint.

Unlike parquet, zst column data may be laid out any way a client so chooses
and is not constrained to the "row group" concept.  Thus, care should be
taken here to control the amount of row skew that can arise.`,
	New: newCommand,
}

func init() {
	zst.Cmd.Add(Create)
}

type Command struct {
	*zst.Command
	outputFlags outputflags.Flags
	inputFlags  inputflags.Flags
}

func MibToBytes(mib float64) int {
	return int(mib * 1024 * 1024)
}

func newCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*zst.Command)}
	c.inputFlags.SetFlags(f)
	c.outputFlags.SetFlagsWithFormat(f, "zst")
	return c, nil
}

func (c *Command) Run(args []string) error {
	_, cleanup, err := c.Init(&c.inputFlags, &c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) == 0 {
		return errors.New("must specify one or more input files")
	}
	zctx := zson.NewContext()
	readers, err := c.inputFlags.Open(zctx, args, true)
	if err != nil {
		return err
	}
	defer zbuf.CloseReaders(readers)
	reader, err := zbuf.MergeReadersByTsAsReader(context.Background(), readers, zbuf.OrderAsc)
	if err != nil {
		return err
	}
	writer, err := c.outputFlags.Open(context.TODO())
	if err != nil {
		return err
	}
	if err := zbuf.Copy(writer, reader); err != nil {
		return err
	}
	return writer.Close()
}

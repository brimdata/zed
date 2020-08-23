package create

import (
	"errors"
	"flag"

	"github.com/brimsec/zq/cli"
	"github.com/brimsec/zq/cmd/zst/root"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zst"
	"github.com/mccanne/charm"
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
	root.Zst.Add(Create)
}

type Command struct {
	*root.Command
	colThresh   float64
	skewThresh  float64
	outputFile  string
	readerFlags zio.ReaderFlags
}

func MibToBytes(mib float64) int {
	return int(mib * 1024 * 1024)
}

func newCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{
		Command: parent.(*root.Command),
	}
	f.Float64Var(&c.colThresh, "coltresh", 5, "minimum frame size (MiB) used for zst columns")
	f.Float64Var(&c.skewThresh, "skewtresh", 25, "minimum skew size (MiB) used to group zst columns")
	f.StringVar(&c.outputFile, "o", "", "name of zst output file")
	c.readerFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	defer c.Cleanup()
	if ok, err := c.Init(); !ok {
		return err
	}
	if len(args) == 0 {
		return errors.New("must specify one or more input files")
	}
	if err := c.readerFlags.Init(); err != nil {
		return err
	}
	if c.outputFile == "" {
		return errors.New("must specify an output file with -o")
	}
	zctx := resolver.NewContext()
	readers, err := cli.OpenInputs(zctx, c.readerFlags.Options(), args, true)
	if err != nil {
		return err
	}
	reader := zbuf.NewCombiner(readers, zbuf.CmpTimeForward)
	defer reader.Close()

	skewThresh := MibToBytes(c.skewThresh)
	colThresh := MibToBytes(c.colThresh)
	writer, err := zst.NewWriterFromPath(c.outputFile, skewThresh, colThresh)
	if err != nil {
		return err
	}
	if err := zbuf.Copy(writer, reader); err != nil {
		writer.Abort()
		return err
	}
	return writer.Close()
}

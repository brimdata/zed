package merge

import (
	"errors"
	"flag"

	"github.com/brimsec/zq/cmd/zdx/root"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zng"
	"github.com/mccanne/charm"
)

var Merge = &charm.Spec{
	Name:  "merge",
	Usage: "merge [ -f framesize ] -o file file1, file2, ...  ",
	Short: "merge two or moore zdx bundles into the output bundle",
	Long: `
The merge command takes two or more zdx bundles as input and
merges the input bundles into a new output bundle,
as specified by the -o argument, while preserving
the lexicographic order of the keys.  When two merged records
have the same value, the first one is preserved and the subsequent
ones are discarded.`,
	New: newMergeCommand,
}

func init() {
	root.Zdx.Add(Merge)
}

type MergeCommand struct {
	*root.Command
	oflag     string
	framesize int
}

func newMergeCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &MergeCommand{Command: parent.(*root.Command)}
	f.IntVar(&c.framesize, "f", 32*1024, "minimum frame size used in the output zdx file")
	f.StringVar(&c.oflag, "o", "", "output file name")
	return c, nil
}

// The combine function depends on the underlying data type but here as
// an example, we simply take the first value of two records with the
// same key and discard the other value(s).  This works fine when the
// base layer is a key-only zdx.
func combine(a, b *zng.Record) *zng.Record {
	return a
}

func (c *MergeCommand) Run(args []string) error {
	if len(args) < 2 {
		return errors.New("must specify at least two input files")
	}
	if c.oflag == "" {
		return errors.New("must specify output file with -o")
	}
	var files []zbuf.Reader
	for _, fname := range args {
		reader, err := zdx.NewReader(fname)
		if err != nil {
			return err
		}
		files = append(files, reader)
	}
	combiner := zdx.NewCombiner(files, combine)
	defer combiner.Close()
	writer, err := zdx.NewWriter(c.oflag, c.framesize)
	if err != nil {
		return err
	}
	defer writer.Close()
	return zbuf.Copy(writer, combiner)
}

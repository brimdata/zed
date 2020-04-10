package merge

import (
	"errors"
	"flag"

	"github.com/brimsec/zq/cmd/sst/root"
	"github.com/brimsec/zq/pkg/sst"
	"github.com/mccanne/charm"
)

var Merge = &charm.Spec{
	Name:  "merge",
	Usage: "merge [ -f framesize ] -o file file1, file2, ...  ",
	Short: "merge two or sst files into the output file",
	Long: `
The merge command takes two or more sst files as input and presumes the
values are roaring bitmaps.  It merges the input files into
a new file, as specified by the -o argument, while preserving
the lexicographic order of the keys and concatenating the values.`,
	New: newMergeCommand,
}

func init() {
	root.Sst.Add(Merge)
}

type MergeCommand struct {
	*root.Command
	oflag     string
	framesize int
}

func newMergeCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &MergeCommand{Command: parent.(*root.Command)}
	f.IntVar(&c.framesize, "f", 32*1024, "minimum frame size used in the output sst file")
	f.StringVar(&c.oflag, "o", "", "output file name")
	return c, nil
}

// The combine function depends on the underlying data type but here as
// an example, we simply contenate the values.
func combine(a, b []byte) []byte {
	out := make([]byte, 0, len(a)+len(b))
	out = append(out, a...)
	return append(out, b...)
}

func (c *MergeCommand) Run(args []string) error {
	if len(args) < 2 {
		return errors.New("must specify at least two input files")
	}
	if c.oflag == "" {
		return errors.New("must specify output file with -o")
	}
	var files []sst.Stream
	for _, fname := range args {
		files = append(files, sst.NewReader(fname))
	}
	combiner := sst.NewCombiner(files, combine)
	defer combiner.Close()
	writer, err := sst.NewWriter(c.oflag, c.framesize, 0)
	if err != nil {
		return err
	}
	return sst.Copy(writer, combiner)
}

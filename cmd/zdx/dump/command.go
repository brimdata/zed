package dump

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"

	"github.com/brimsec/zq/cmd/zdx/root"
	"github.com/brimsec/zq/zdx"
	"github.com/mccanne/charm"
)

var Dump = &charm.Spec{
	Name:  "dump",
	Usage: "dump [-l level] [-k key] <file>",
	Short: "dump the contents of a zdx index or file",
	Long: `
The dump command prints out the keys and offsets of a frame in an zdx file,
or the keys within a frame indicated by the key parameter as hex strings.
If level is 0, then the key/value pairs are displayed.  If level is 1 or higher,
then the index is displayed as a sequence of base keys and file offsets where
the offset is the offset in the file below the index one lower in the hierarchy
where that key is defined.`,
	New: newDumpCommand,
}

func init() {
	root.Zdx.Add(Dump)
}

type DumpCommand struct {
	*root.Command
	level int
}

func newDumpCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &DumpCommand{Command: parent.(*root.Command)}
	f.IntVar(&c.level, "l", 0, "zdx level to dump")
	return c, nil
}

func (c *DumpCommand) dumpBase(path string) error {
	reader := zdx.NewReader(path)
	if err := reader.Open(); err != nil {
		return err
	}
	defer reader.Close()
	for {
		pair, err := reader.Read()
		if err != nil {
			return err
		}
		if pair.Key == nil {
			return nil
		}
		fmt.Printf("%s:%s\n", format(pair.Key), format(pair.Value))
	}
	return nil
}

func (c *DumpCommand) dumpIndex(path string, level int) error {
	reader := zdx.NewIndexReader(path, level)
	if err := reader.Open(); err != nil {
		return err
	}
	defer reader.Close()
	for {
		key, off, err := reader.Read()
		if err != nil {
			return err
		}
		if key == nil {
			return nil
		}
		fmt.Printf("%s:%d\n", format(key), off)
	}
	return nil
}

func (c *DumpCommand) Run(args []string) error {
	if len(args) != 1 {
		return errors.New("zdx dump: must be run with a single file argument")
	}
	path := args[0]
	if c.level == 0 {
		return c.dumpBase(path)
	} else {
		return c.dumpIndex(path, c.level)
	}
	return nil
}

func format(b []byte) string {
	if len(b) == 0 {
		//XXX should just omit the value for these
		return "<empty>"
	}
	return hex.EncodeToString(b)
}

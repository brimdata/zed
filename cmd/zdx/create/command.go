package create

import (
	"errors"
	"flag"
	"os"

	"github.com/brimsec/zq/cmd/zdx/root"
	"github.com/brimsec/zq/zdx"
	"github.com/mccanne/charm"
)

var Create = &charm.Spec{
	Name:  "create",
	Usage: "create [-f framesize] [ -o file ] <key-value-file>",
	Short: "generate an zdx file from a tsv file",
	Long: `
The create command generates an zdx containing string keys and binary values.
Each line in the input file constists of key as a hex string optionally
followed by a colon and a hex string the key's value.  A nil value is represented
with no characters (i.e., string key, colon, then newline)`,
	New: newCreateCommand,
}

func init() {
	root.Zdx.Add(Create)
}

type CreateCommand struct {
	*root.Command
	framesize  int
	outputFile string
	inputFile  string
}

func newCreateCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &CreateCommand{Command: parent.(*root.Command)}
	f.IntVar(&c.framesize, "f", 32*1024, "minimum frame size used in zdx file")
	f.StringVar(&c.outputFile, "o", "zdx", "output file name")
	return c, nil
}

func (c *CreateCommand) Run(args []string) error {
	if len(args) != 1 {
		return errors.New("must specify a single input file containing keys and optional values")
	}
	path := args[0]
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()
	table := NewTable()
	if err := table.Scan(in); err != nil {
		return err
	}
	writer, err := zdx.NewWriter(c.outputFile, c.framesize, 0)
	if err != nil {
		return err
	}
	defer writer.Close()
	return zdx.Copy(writer, table)
}

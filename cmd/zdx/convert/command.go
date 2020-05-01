package convert

import (
	"errors"
	"flag"
	"strings"

	"github.com/brimsec/zq/cmd/zdx/root"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/mccanne/charm"
)

var Convert = &charm.Spec{
	Name:  "convert",
	Usage: "convert [-f framesize] [ -o file ] -k field[,field,...] file",
	Short: "generate an zdx file from one or more zng files",
	Long: `
The convert command generates a zdx index containing keys and optional values
from the input file.  The required flag -k specifies one or more zng record
field names that comprise the index search keys, in precedence order.
The keys must be pre-sorted in ascending order with
respect to the stream of zng records; otherwise the index will not work correctly.
The input records are all copied to the base layer of the output index, as is,
so any information stored alongside the keys (e.g., pre-computed aggregations).
It is an error if the key or value fields are not of uniform type.`,
	New: newCommand,
}

func init() {
	root.Zdx.Add(Convert)
}

type Command struct {
	*root.Command
	framesize   int
	outputFile  string
	keys        string
	ReaderFlags zio.ReaderFlags
}

func newCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{
		Command: parent.(*root.Command),
	}
	f.IntVar(&c.framesize, "f", 32*1024, "minimum frame size used in zdx file")
	f.StringVar(&c.outputFile, "o", "zdx", "output zdx bundle name")
	f.StringVar(&c.keys, "k", "", "comma-separated list of field names for keys")
	c.ReaderFlags.SetFlags(f)

	return c, nil
}

func (c *Command) Run(args []string) error {
	if c.keys == "" {
		return errors.New("must specify at least one key field with -k")
	}
	//XXX no reason to limit this
	if len(args) != 1 {
		return errors.New("must specify a single zng input file containing keys and optional values")
	}
	path := args[0]
	zctx := resolver.NewContext()
	cfg := detector.OpenConfig{
		Format:    c.ReaderFlags.Format,
		DashStdin: true,
		//JSONTypeConfig: c.jsonTypeConfig,
		//JSONPathRegex:  c.jsonPathRegexp,
	}
	file, err := detector.OpenFile(zctx, path, cfg)
	if err != nil {
		return err
	}
	keys := strings.Split(c.keys, ",")
	writer, err := zdx.NewWriter(zctx, c.outputFile, keys, c.framesize)
	if err != nil {
		return err
	}
	close := true
	defer func() {
		if close {
			writer.Close()
		}
	}()
	if err := zbuf.Copy(writer, zbuf.Reader(file)); err != nil {
		return err
	}
	close = false
	return writer.Close()
}

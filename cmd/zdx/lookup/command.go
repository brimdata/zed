package lookup

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"

	"github.com/brimsec/zq/cmd/zdx/root"
	"github.com/brimsec/zq/zdx"
	"github.com/mccanne/charm"
)

var Lookup = &charm.Spec{
	Name:  "lookup",
	Usage: "lookup -k <key> <file>",
	Short: "lookup a key in an zdx file and print value as hex bytes",
	Long: `
The lookup command uses the index files of an zdx hierarchy to locate the
specified key in the base zdx file and displays the value as bytes.`,
	New: newLookupCommand,
}

func init() {
	root.Zdx.Add(Lookup)
}

type LookupCommand struct {
	*root.Command
	key string
}

func newLookupCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &LookupCommand{Command: parent.(*root.Command)}
	f.StringVar(&c.key, "k", "", "key to search")
	return c, nil
}

func (c *LookupCommand) Run(args []string) error {
	if len(args) != 1 {
		return errors.New("zdx lookup: must be run with a single file argument")
	}
	path := args[0]
	if c.key == "" {
		return errors.New("must specify a key")
	}
	key, err := hex.DecodeString(c.key)
	if err != nil {
		return err
	}
	finder, err := zdx.NewFinder(path)
	if err != nil {
		return err
	}
	defer finder.Close()
	val, err := finder.Lookup(key)
	if err != nil {
		return err
	}
	if val == nil {
		fmt.Println("not found")
	} else {
		fmt.Println(hex.EncodeToString(val))
	}
	return nil
}

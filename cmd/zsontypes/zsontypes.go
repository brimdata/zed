package main

import (
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/inputflags"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/zio/ndjsonio/compat"
	"github.com/brimdata/zed/zio/tzngio"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zng/resolver"
	"github.com/brimdata/zed/zson"
)

var ZsonTypes = &charm.Spec{
	Name:  "zsontypes",
	Usage: "zsontypes <file.json>",
	Short: "tool for printing zson types for legacy types.json",
	Long: `
The Zsontypes tool prints a ZSON-formatted string for each type defined in a legacy types.json file.
It is intended to help those transitioning from the "types.json" approach to the newer Z-based approach.

`,
	New: func(parent charm.Command, flags *flag.FlagSet) (charm.Command, error) {
		return New(flags)
	},
}

func init() {
	ZsonTypes.Add(charm.Help)
}

type Command struct{}

func New(f *flag.FlagSet) (charm.Command, error) {
	return &Command{}, nil
}

func (c *Command) Run(args []string) error {
	if len(args) != 1 {
		return ZsonTypes.Exec(c, []string{"help"})
	}
	fname := args[0]
	return c.printTypes(fname)
}

func (c *Command) printTypes(fname string) error {
	tc, err := inputflags.LoadJSONConfig(fname)
	if err != nil {
		return err
	}
	zctx := resolver.NewContext()
	// (from ndjson.io.NewReader)
	// Note: we add hardwired aliases for "port" to "uint16" when reading
	// *any* json file but they are only used when the schema mapper
	// (aka typings config) references such types from a configured schema.
	// However, the schema mapper should be responsible for creating these
	// aliases according to its configuration.  See issue #1427.
	if _, err := zctx.LookupTypeAlias("zenum", zng.TypeString); err != nil {
		return err
	}
	if _, err = zctx.LookupTypeAlias("port", zng.TypeUint16); err != nil {
		return err
	}
	tp := tzngio.NewTypeParser(zctx)

	for key, columns := range tc.Descriptors {
		typeName, err := compat.DecodeType(columns)
		if err != nil {
			return fmt.Errorf("error decoding type \"%s\": %s", typeName, err)
		}
		typ, err := tp.Parse(typeName)
		if err != nil {
			return err
		}
		fmt.Printf("%s: %s\n", key, zson.FormatType(typ))
	}
	return nil
}

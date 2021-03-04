package main

import (
	"flag"
	"fmt"

	"github.com/brimsec/zq/cli/inputflags"
	"github.com/brimsec/zq/zio/ndjsonio/compat"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/mccanne/charm"
)

var ZsonTypes = &charm.Spec{
	Name:  "zsontypes",
	Usage: "zsontypes <file.json>",
	Short: "tool for printing zson types for legacy types.json",
	Long: `
The Zsontypes tool prints a ZSON-formatted for each type defined in a legacy types.json file.
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
	c := &Command{}
	return c, nil
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
	tp := tzngio.NewTypeParser(zctx)

	// (from ndjson.io.NewReader)
	// Note: we add hardwired aliases for "port" to "uint16" when reading
	// *any* json file but they are only used when the schema mapper
	// (aka typings config) references such types from a configured schema.
	// However, the schema mapper should be responsible for creating these
	// aliases according to its configuration.  See issue #1427.
	_, err = zctx.LookupTypeAlias("zenum", zng.TypeString)
	if err != nil {
		return err
	}
	_, err = zctx.LookupTypeAlias("port", zng.TypeUint16)
	if err != nil {
		return err
	}

	for key, columns := range tc.Descriptors {
		typeName, err := compat.DecodeType(columns)
		if err != nil {
			return fmt.Errorf("error decoding type \"%s\": %s", typeName, err)
		}
		typ, err := tp.Parse(typeName)
		if err != nil {
			return err
		}
		fmt.Printf("%s: %s\n", key, typ.ZSON())
	}
	return nil
}

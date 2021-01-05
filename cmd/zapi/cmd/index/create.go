package idx

import (
	"errors"
	"flag"
	"strings"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/cmd/zapi/cmd"
	"github.com/brimsec/zq/compiler"
	"github.com/mccanne/charm"
)

var Create = &charm.Spec{
	Name:  "create",
	Usage: "create [options] [-z zql] [ pattern [ pattern ...]]",
	Short: "create index on a space",
	Long: `
"zapi index create" creates index files in a zar archive using one or more indexing
rules.

A pattern is either a field name or a ":" followed by a zng type name.
For example, to index the all fields of type port and the field id.orig_h,
you would run:

	zapi index create id.orig_h :port

Each pattern results in a separate, single-key microindex file for each log file found.

For custom indexes, zql can be used instead of a pattern. This
requires specifying the key and output file name. For example:

    zapi index create -k id.orig_h -o custom -z "count() by _path, id.orig_h | sort id.orig_h"
Multiple keys may be specified with multiple -k arguments, in which case the first key is the primary search key, the second key is the secondary search key, and so forth.  For example,

zapi index create -k id.orig_h -k count -o custom -z "count() by _path, id.orig_h | sort id.orig_h,count"
`,
	New: NewCreate,
}

type CreateCmd struct {
	*cmd.Command
	root       string
	inputFile  string
	outputFile string
	keys       arrayFlag
	zql        string
}

func NewCreate(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &CreateCmd{Command: parent.(*Command).Command}
	f.Var(&c.keys, "k", "key fields (can be specified multiple times)")
	f.StringVar(&c.inputFile, "i", "", "input file relative to each zar directory ('' means archive log file in the parent of the zar directory)")
	f.StringVar(&c.outputFile, "o", "index.zng", "name of microindex output file (for custom indexes)")
	f.StringVar(&c.zql, "z", "", "zql for custom indexes")
	return c, nil
}

func (c *CreateCmd) Run(args []string) error {
	if len(args) == 0 && c.zql == "" {
		return errors.New("zapi index create: one or more indexing patterns must be specified")
	}
	id, err := c.SpaceID()
	if err != nil {
		return err
	}
	req := api.IndexPostRequest{
		Patterns:   args,
		InputFile:  c.inputFile,
		OutputFile: c.outputFile,
	}
	if c.zql != "" {
		_, err := compiler.ParseProc(c.zql)
		if err != nil {
			return err
		}
		req.ZQL = c.zql
		if err != nil {
			return err
		}
		req.Keys = c.keys
	}
	return c.Connection().IndexPost(c.Context(), id, req)
}

type arrayFlag []string

func (i *arrayFlag) String() string {
	return strings.Join(*i, ", ")
}

func (i *arrayFlag) Set(value string) error {
	*i = append(*i, value)
	return nil
}

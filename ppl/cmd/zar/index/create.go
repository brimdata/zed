package index

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/brimsec/zq/cli/procflags"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/pkg/rlimit"
	"github.com/brimsec/zq/pkg/signalctx"
	"github.com/brimsec/zq/ppl/cmd/zar/root"
	"github.com/brimsec/zq/ppl/lake"
	"github.com/brimsec/zq/ppl/lake/index"
	"github.com/mccanne/charm"
)

var Create = &charm.Spec{
	Name:  "create",
	Usage: "create [-R root] [options] [-z zql] [ pattern [ pattern ...]]",
	Short: "create index rule for archive chunk files",
	Long: `
"zar index create" creates index files in a zar archive using one or more indexing
rules.

A pattern is either a field name or a ":" followed by a zng type name.
For example, to create two indexes, one on the field id.orig_h, and one on all
fields of type uint16, you would run:

	zar index create -R /path/to/logs id.orig_h :uint16

Each pattern results in a separate microindex file for each log file found.

For custom indexes, zql can be used instead of a pattern. This
requires specifying the key and output file name. For example:

       zar index create -k id.orig_h -o custom -z "count() by _path, id.orig_h | sort id.orig_h"
`,
	New: NewCreate,
}

type CreateCommand struct {
	*root.Command
	displayer  displayer
	framesize  int
	inputFile  string
	keys       string
	outputFile string
	ensure     bool
	noapply    bool
	procFlags  procflags.Flags
	root       string
	zql        string
}

func NewCreate(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &CreateCommand{Command: parent.(*Command).Command}
	f.StringVar(&c.root, "R", os.Getenv("ZAR_ROOT"), "root location of zar archive to walk")
	f.BoolVar(&c.ensure, "ensure", false, "ensures that index rules are applied to all chunks")
	f.StringVar(&c.keys, "k", "key", "one or more comma-separated key fields")
	f.IntVar(&c.framesize, "f", 32*1024, "minimum frame size used in microindex file")
	f.StringVar(&c.inputFile, "i", "_", "input file relative to each zar directory ('_' means archive log file in the parent of the zar directory)")
	f.BoolVar(&c.noapply, "noapply", false, "create index rules but do not apply them to the archive")
	f.StringVar(&c.outputFile, "o", "index.zng", "name of microindex output file (for custom indexes)")
	f.BoolVar(&c.displayer.quiet, "q", false, "don't print progress on stdout")
	f.StringVar(&c.zql, "z", "", "zql for custom indexes")
	c.procFlags.SetFlags(f)
	return c, nil
}

func (c *CreateCommand) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(&c.procFlags); err != nil {
		return err
	}
	if len(args) == 0 && c.zql == "" && !c.ensure {
		return errors.New("unless -ensure is specified, one or more indexing patterns must be specified")
	}
	if c.root == "" {
		return errors.New("a directory must be specified with -R or ZAR_ROOT")
	}
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}

	lk, err := lake.OpenLake(c.root, nil)
	if err != nil {
		return err
	}

	ctx, cancel := signalctx.New(os.Interrupt)
	defer cancel()

	defs, err := c.addRules(ctx, lk, args)
	if err != nil {
		return err
	}

	if c.noapply {
		return nil
	}

	if c.ensure {
		defs, err = lk.ReadDefinitions(ctx)
		if err != nil {
			return err
		}
	}

	c.displayer.run()
	defer c.displayer.close()

	return lake.WriteIndices(ctx, lk, c.displayer.ch, defs...)
}

func (c *CreateCommand) addRules(ctx context.Context, lk *lake.Lake, args []string) ([]*index.Definition, error) {
	if len(args) == 0 && c.zql == "" {
		return nil, nil
	}
	var input string
	if c.inputFile != "_" {
		input = c.inputFile
	}

	var rules []index.Rule
	if c.zql != "" {
		rule, err := index.NewZqlRule(c.zql, c.outputFile, field.DottedList(c.keys))
		if err != nil {
			return nil, err
		}
		rule.Framesize = c.framesize
		rule.Input = input
		rules = append(rules, rule)
	}

	for _, pattern := range args {
		rule, err := index.NewRule(pattern)
		if err != nil {
			return nil, err
		}
		rule.Input = input
		rules = append(rules, rule)
	}

	return lake.AddRules(ctx, lk, rules)
}

type displayer struct {
	ch    chan string
	quiet bool
	wg    sync.WaitGroup
}

func (d *displayer) run() {
	d.ch = make(chan string)
	d.wg.Add(1)
	go func() {
		for line := range d.ch {
			if !d.quiet {
				fmt.Println(line)
			}
		}
		d.wg.Done()
	}()
}

func (d *displayer) close() {
	close(d.ch)
	d.wg.Wait()
}

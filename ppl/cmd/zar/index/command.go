package index

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/brimsec/zq/cli/procflags"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/pkg/rlimit"
	"github.com/brimsec/zq/pkg/signalctx"
	"github.com/brimsec/zq/ppl/archive"
	"github.com/brimsec/zq/ppl/archive/index"
	"github.com/brimsec/zq/ppl/cmd/zar/root"
	"github.com/mccanne/charm"
)

var Index = &charm.Spec{
	Name:  "index",
	Usage: "index [-R root] [options] [-z zql] [ pattern [ pattern ...]]",
	Short: "create index files for zng files",
	Long: `
"zar index" creates index files in a zar archive using one or more indexing
rules.

A pattern is either a field name or a ":" followed by a zng type name.
For example, to index the all fields of type port and the field id.orig_h,
you would run:

	zar index -R /path/to/logs id.orig_h :port

Each pattern results in a separate microindex file for each log file found.

For custom indexes, zql can be used instead of a pattern. This
requires specifying the key and output file name. For example:

       zar index -k id.orig_h -o custom -z "count() by _path, id.orig_h | sort id.orig_h"
`,
	New: New,
}

func init() {
	root.Zar.Add(Index)
}

type Command struct {
	*root.Command
	framesize  int
	inputFile  string
	keys       string
	outputFile string
	procFlags  procflags.Flags
	progress   chan string
	quiet      bool
	root       string
	zql        string
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.root, "R", os.Getenv("ZAR_ROOT"), "root location of zar archive to walk")
	f.StringVar(&c.keys, "k", "key", "one or more comma-separated key fields")
	f.IntVar(&c.framesize, "f", 32*1024, "minimum frame size used in microindex file")
	f.StringVar(&c.inputFile, "i", "_", "input file relative to each zar directory ('_' means archive log file in the parent of the zar directory)")
	f.StringVar(&c.outputFile, "o", "index.zng", "name of microindex output file (for custom indexes)")
	f.BoolVar(&c.quiet, "q", false, "don't print progress on stdout")
	f.StringVar(&c.zql, "z", "", "zql for custom indexes")
	c.procFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(&c.procFlags); err != nil {
		return err
	}
	if len(args) == 0 && c.zql == "" {
		return errors.New("zar index: one or more indexing patterns must be specified")
	}
	if c.root == "" {
		return errors.New("zar index: a directory must be specified with -R or ZAR_ROOT")
	}
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}

	ark, err := archive.OpenArchive(c.root, nil)
	if err != nil {
		return err
	}

	rules, err := c.rules(args)
	if err != nil {
		return err
	}

	ctx, cancel := signalctx.New(os.Interrupt)
	defer cancel()

	defs, err := archive.AddRules(ctx, ark, rules)
	if err != nil {
		return err
	}
	if !c.quiet {
		c.progress = make(chan string)
		go c.displayProgress()
	}

	return archive.ApplyDefinitions(ctx, ark, c.progress, defs...)
}

func (c *Command) displayProgress() {
	for line := range c.progress {
		fmt.Println(line)
	}
}

func (c *Command) rules(args []string) ([]index.Rule, error) {
	var input string
	if c.inputFile != "_" {
		input = c.inputFile
	}

	var rules []index.Rule
	if c.zql != "" {
		rule, err := index.NewZqlRule(c.zql, c.outputFile, field.DottedList(c.keys))
		if err != nil {
			return nil, fmt.Errorf("zar index add: %w", err)
		}
		rule.Framesize = c.framesize
		rule.Input = input
		rules = append(rules, rule)
	}
	for _, pattern := range args {
		rule, err := index.NewRule(pattern)
		if err != nil {
			return nil, fmt.Errorf("zar index add: %w", err)
		}
		rule.Input = input
		rules = append(rules, rule)
	}
	return rules, nil
}

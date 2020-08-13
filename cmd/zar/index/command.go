package index

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/cmd/zar/root"
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

Each pattern results a separate zdx index file for each log file found.

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
	root       string
	quiet      bool
	inputFile  string
	outputFile string
	framesize  int
	keys       string
	zql        string
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.root, "R", os.Getenv("ZAR_ROOT"), "root location of zar archive to walk")
	f.StringVar(&c.keys, "k", "key", "one or more comma-separated key fields")
	f.IntVar(&c.framesize, "f", 32*1024, "minimum frame size used in zdx file")
	f.StringVar(&c.inputFile, "i", "_", "input file relative to each zar directory ('_' means archive log file in the parent of the zar directory)")
	f.StringVar(&c.outputFile, "o", "zdx", "output index name (for custom indexes)")
	f.BoolVar(&c.quiet, "q", false, "don't print progress on stdout")
	f.StringVar(&c.zql, "z", "", "zql for custom indexes")
	return c, nil
}

func (c *Command) Run(args []string) error {
	if len(args) == 0 && c.zql == "" {
		return errors.New("zar index: one or more indexing patterns must be specified")
	}
	if c.root == "" {
		return errors.New("zar index: a directory must be specified with -R or ZAR_ROOT")
	}

	ark, err := archive.OpenArchive(c.root, nil)
	if err != nil {
		return err
	}

	var rules []archive.Rule
	keys := strings.Split(c.keys, ",")
	if c.zql != "" {
		rule, err := archive.NewZqlRule(c.zql, c.outputFile, keys, c.framesize)
		if err != nil {
			return errors.New("zar index: " + err.Error())
		}
		rules = append(rules, *rule)
	}
	for _, pattern := range args {
		rule, err := archive.NewRule(pattern)
		if err != nil {
			return errors.New("zar index: " + err.Error())
		}
		rules = append(rules, *rule)
	}
	var wg sync.WaitGroup
	var progress chan string
	if !c.quiet {
		wg.Add(1)
		progress = make(chan string)
		go func() {
			for line := range progress {
				fmt.Println(line)
			}
			wg.Done()
		}()
	}
	err = archive.IndexDirTree(ark, rules, c.inputFile, progress)
	if progress != nil {
		close(progress)
		wg.Wait()
	}
	return err
}

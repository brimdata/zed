package index

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/cmd/zar/root"
	"github.com/mccanne/charm"
)

var Index = &charm.Spec{
	Name:  "index",
	Usage: "index [-R dir] pattern [ pattern ...]",
	Short: "create index files for zng files",
	Long: `
zar index descends the directory argument starting at dir and looks
for files with zar directories.  Each such file found is indexed according
to the one or more indexing rules provided, and the resulting indexes
are written to that file's zar directory.

A pattern is either a field name or a ":" followed by a zng type name.
For example, to index the all fields of type port and the field id.orig_h,
you would run

	zar index -R /path/to/logs id.orig_h :port

Each pattern results a separate zdx index file for each log file found.

The root directory must be specified either by the ZAR_ROOT environemnt
variable or the -R option.
`,
	New: New,
}

func init() {
	root.Zar.Add(Index)
}

type Command struct {
	*root.Command
	root  string
	quiet bool
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.root, "R", os.Getenv("ZAR_ROOT"), "root directory of zar archive to walk")
	f.BoolVar(&c.quiet, "q", false, "don't print progress on stdout")
	return c, nil
}

func (c *Command) Run(args []string) error {
	if len(args) == 0 {
		return errors.New("zar index: one or more indexing patterns must be specified")
	}
	if c.root == "" {
		return errors.New("zar index: a directory must be specified with -R or ZAR_ROOT")
	}
	var rules []archive.Rule
	for _, pattern := range args {
		rule, err := archive.NewRule(pattern)
		if err != nil {
			return errors.New("zar index: " + err.Error())
		}
		rules = append(rules, rule)
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
	err := archive.IndexDirTree(c.root, rules, progress)
	if progress != nil {
		close(progress)
		wg.Wait()
	}
	return err
}

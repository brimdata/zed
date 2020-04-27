package index

import (
	"errors"
	"flag"
	"fmt"
	"strings"
	"sync"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/cmd/zar/root"
	"github.com/mccanne/charm"
)

var Find = &charm.Spec{
	Name:  "find",
	Usage: "find [-d <dir>] <pattern>",
	Short: "look through zar index files and displays matches",
	Long: `
"zar find" descends the directory given by the -d option looking for zng files
that have a corresponding zar index conforming to the indicated <pattern>.
The <pattern> argument has the form "field=value" (for field searches)
or ":type=value" (for type searches).  For example, if type "ip" has been
indexed then the IP 10.0.1.2 can be searched by saying

	zar find -d /path/to/logs :ip=10.0.1.2

Or if the field "uri" has been indexed, you might say

	zar find -d /path/to/logs uri=/x/y/z

The path of each zng file that matches the pattern is printed.
`,
	New: New,
}

func init() {
	root.Zar.Add(Find)
}

type Command struct {
	*root.Command
	dir         string
	skipMissing bool
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.dir, "d", ".", "directory to descend")
	f.BoolVar(&c.skipMissing, "Q", false, "skip errors caused by missing index files ")
	return c, nil
}

func (c *Command) Run(args []string) error {
	if len(args) != 1 {
		return errors.New("zar find: exactly one search pattern must be provided")
	}
	v := strings.Split(args[0], "=")
	if len(v) != 2 {
		return errors.New("zar find: syntax error: " + args[0])
	}
	fieldOrType := v[0]
	pattern := v[1]
	rule, err := archive.NewRule(fieldOrType)
	if err != nil {
		return errors.New("zar find: error parsing pattern: " + err.Error())
	}
	hits := make(chan string)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for hit := range hits {
			fmt.Println(hit)
		}
		wg.Done()
	}()
	err = archive.Find(c.dir, rule, pattern, hits, c.skipMissing)
	close(hits)
	wg.Wait()
	return err
}

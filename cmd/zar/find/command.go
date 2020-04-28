package find

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/cmd/zar/root"
	"github.com/brimsec/zq/emitter"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zng"
	"github.com/mccanne/charm"
)

var Find = &charm.Spec{
	Name:  "find",
	Usage: "find [-R dir] pattern",
	Short: "look through zar index files and displays matches",
	Long: `
"zar find" descends the directory given by the -R option looking for zng files
that have a corresponding zar index conforming to the indicated <pattern>.
The "pattern" argument has the form "field=value" (for field searches)
or ":type=value" (for type searches).  For example, if type "ip" has been
indexed then the IP 10.0.1.2 can be searched by saying

	zar find -R /path/to/logs :ip=10.0.1.2

Or if the field "uri" has been indexed, you might say

	zar find -R /path/to/logs uri=/x/y/z

The path of each zng file that matches the pattern is printed.

If the root directory is not specified by either the ZAR_ROOT environemnt
variable or the -R option, then the current directory is assumed.
`,
	New: New,
}

func init() {
	root.Zar.Add(Find)
}

type Command struct {
	*root.Command
	root        string
	skipMissing bool
	indexFile   string
	outputFile  string
	WriterFlags zio.WriterFlags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.root, "R", os.Getenv("ZAR_ROOT"), "root directory of zar archive to walk")
	f.BoolVar(&c.skipMissing, "Q", false, "skip errors caused by missing index files ")
	f.StringVar(&c.indexFile, "x", "", "name of zdx index in the zar dirs")
	f.StringVar(&c.outputFile, "o", "", "write data to output file")

	// Flags added for writers are -f, -T, -F, -E, -U, and -b
	c.WriterFlags.SetFlags(f)

	return c, nil
}

func (c *Command) Run(args []string) error {
	if len(args) != 1 {
		return errors.New("zar find: exactly one search pattern must be provided")
	}
	var pattern string
	var rule archive.Rule
	zngOutput := false
	if c.outputFile != "" {
		if c.indexFile == "" {
			return errors.New("zar find: must specify -x with -o")
		}
		zngOutput = true
		//XXX hack fo now.  fix later.r
		rule, _ = archive.NewStaticRule(c.indexFile)
		pattern = args[0]
	} else {
		v := strings.Split(args[0], "=")
		if len(v) != 2 {
			return errors.New("zar find: syntax error: " + args[0])
		}
		fieldOrType := v[0]
		pattern = v[1]
		var err error
		rule, err = archive.NewRule(fieldOrType)
		if err != nil {
			return errors.New("zar find: error parsing pattern: " + err.Error())
		}
	}
	//XXX allow "-" to trigger zng but changed back for emitter API
	if c.outputFile == "-" {
		c.outputFile = ""
	}
	rootDir := c.root
	if rootDir == "" {
		rootDir = "."
	}
	var err error
	var searchErr error
	var wg sync.WaitGroup
	wg.Add(1)
	if zngOutput {
		writer, err := emitter.NewFile(c.outputFile, &c.WriterFlags)
		if err != nil {
			return err
		}
		defer writer.Close()
		hits := make(chan *zng.Record)
		go func() {
			for hit := range hits {
				if err := writer.Write(hit); err != nil {
					searchErr = err
					break
				}
			}
			wg.Done()
		}()
		err = archive.FindZng(rootDir, rule, pattern, hits, c.skipMissing)
		close(hits)
	} else {
		hits := make(chan string)
		go func() {
			for hit := range hits {
				fmt.Println(hit)
			}
			wg.Done()
		}()
		err = archive.Find(rootDir, rule, pattern, hits, c.skipMissing)
		close(hits)
	}
	wg.Wait()
	if err == nil {
		err = searchErr
	}
	return err
}

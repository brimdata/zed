package find

import (
	"errors"
	"flag"
	"os"

	"github.com/brimdata/zed/cli/outputflags"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
)

var Find = &charm.Spec{
	Name:  "find",
	Usage: "find [options] pattern [pattern...]",
	Short: "look through zar index files and displays matches",
	Long: `
TBD: update this help: Issue #2532

"zar find" searches a zar archive
looking for zng files that have been indexed and performs a search on
each such index file in accordance with the specified search pattern.
Indexes are created by "zar index".

For standard indexes, "pattern" argument has the form "field=value" (for field searches)
or ":type=value" (for type searches).  For example, if type "ip" has been
indexed then the IP 10.0.1.2 can be searched by saying

	zar find :ip=10.0.1.2

Or if the field "uri" has been indexed, you might say

	zar find uri=/x/y/z

For custom indexes, the name of index is given by the -x option,
and the "pattern" argument(s) comprise one or more values that
are parseable in accordance with the zng type of the corresponding
search keys.  For example, an index with custom keys of the form

	record[a:record[x:int64,y:ip],b:string]

could be queried using syntax like this

	zar find -x custom 99 10.0.0.1 hello

The results of a search is either a list of the paths of each
zng log that matches the pattern (the default), or a zng stream of the
records of the base layer of the index file (-z)
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Find)
}

type Command struct {
	*zedlake.Command
	root          string
	skipMissing   bool
	indexFile     string
	relativePaths bool
	outputFlags   outputflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*zedlake.Command)}
	f.StringVar(&c.root, "R", os.Getenv("ZED_LAKE_ROOT"), "root location of zar archive to walk")
	f.BoolVar(&c.skipMissing, "Q", false, "skip errors caused by missing index files ")
	f.StringVar(&c.indexFile, "x", "", "name of microindex for custom index searches")
	f.BoolVar(&c.relativePaths, "relative", false, "display paths relative to root")

	c.outputFlags.SetFlags(f)

	return c, nil
}

func (c *Command) Run(args []string) error {
	return errors.New("issue #2532")
	/* NOT YET
	ctx, cleanup, err := c.Init(&c.outputFlags)
 	if err != nil {
		return err
	}
	defer cleanup()
	lk, err := lake.OpenLake(c.root, nil)
	if err != nil {
		return err
	}

	query, err := index.ParseQuery(c.indexFile, args)
	if err != nil {
		return err
	}

	var findOptions []lake.FindOption
	if c.pathField != "" {
		findOptions = append(findOptions, lake.AddPath(c.pathField, !c.relativePaths))
	}
	if c.skipMissing {
		findOptions = append(findOptions, lake.SkipMissing())
	}
	outputFile := c.outputFlags.FileName()
	if outputFile == "-" {
		outputFile = ""
	}
	writer, err := emitter.NewFile(ctx, outputFile, c.outputFlags.Options())
	if err != nil {
		return err
	}
	defer writer.Close()
	hits := make(chan *zng.Record)
	var searchErr error
	go func() {
		searchErr = lake.Find(ctx, zson.NewContext(), lk, query, hits, findOptions...)
		close(hits)
	}()
	for hit := range hits {
		var err error
		if writer != nil {
			err = writer.Write(hit)
		} else {
			var path string
			path, err = hit.AccessString(c.pathField)
			if err == nil {
				fmt.Println(path)
			}
		}
		if err != nil {
			return err
		}
	}
	return searchErr
	*/
}

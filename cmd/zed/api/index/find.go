package idx

import (
	"flag"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/cli/outputflags"
	apicmd "github.com/brimdata/zed/cmd/zed/api"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/emitter"
)

var Find = &charm.Spec{
	Name:  "find",
	Usage: "find [options] pattern [pattern...]",
	Short: "look through zar index files and displays matches",
	Long: `
"zapi index find" searches an archive-backed space
looking for zng files that have been indexed and performs a search on
each such index file in accordance with the specified search pattern.
Indexes are created by "zapi index create".

For standard indexes, "pattern" argument has the form "field=value" (for field searches)
or ":type=value" (for type searches).  For example, if type "ip" has been
indexed then the IP 10.0.1.2 can be searched by saying

	zapi index find :ip=10.0.1.2

Or if the field "uri" has been indexed, you might say

	zapi index uri=/x/y/z

For custom indexes, the name of index is given by the -x option,
and the "pattern" argument(s) comprise one or more values that
are parseable in accordance with the zng type of the corresponding
search keys.  For example, an index with custom keys of the form

	record[a:record[x:int64,y:ip],b:string]

could be queried using syntax like this

	zapi idx find -x custom 99 10.0.0.1 hello

The results of a search is either a list of the paths of each
zng log that matches the pattern (the default), or a zng stream of the
records of the base layer of the index file (-z)
`,
	New: NewFind,
}

type FindCmd struct {
	*apicmd.Command
	indexFile     string
	pathField     string
	relativePaths bool
	outputFlags   outputflags.Flags
}

const DefaultAddPathField = "_path"

func NewFind(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &FindCmd{Command: parent.(*Command).Command}
	f.StringVar(&c.indexFile, "x", "", "name of microindex for custom index searches")
	f.StringVar(&c.pathField, "l", DefaultAddPathField, "zng field name for path name of log file")
	f.BoolVar(&c.relativePaths, "relative", false, "display paths relative to root")

	c.outputFlags.SetFlags(f)

	return c, nil
}

func (c *FindCmd) Run(args []string) error {
	ctx, cleanup, err := c.Init(&c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	req := api.IndexSearchRequest{IndexName: c.indexFile, Patterns: args}
	id, err := c.SpaceID(ctx)
	if err != nil {
		return err
	}
	stream, err := c.Connection().IndexSearch(ctx, id, req, nil)
	if err != nil {
		return err
	}
	writer, err := emitter.NewFile(ctx, c.outputFlags.FileName(), c.outputFlags.Options())
	if err != nil {
		return err
	}
	defer writer.Close()
	return zio.Copy(writer, stream)
}

package ls

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/ppl/archive"
	"github.com/brimsec/zq/ppl/archive/chunk"
	"github.com/brimsec/zq/ppl/archive/index"
	"github.com/brimsec/zq/ppl/cmd/zar/root"
	"github.com/mccanne/charm"
)

var Ls = &charm.Spec{
	Name:  "ls",
	Usage: "ls [-R root] [options] [pattern]",
	Short: "list the zar directories in an archive",
	Long: `
"zar ls" walks an archive's directories and prints out
the path of each zar directory contained with those top-level directories.
`,
	New: New,
}

func init() {
	root.Zar.Add(Ls)
}

type Command struct {
	*root.Command
	ark           *archive.Archive
	root          string
	lflag         bool
	indexDesc     bool
	defs          index.DefinitionMap
	relativePaths bool
	showRanges    bool
	spanInfos     bool
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.root, "R", os.Getenv("ZAR_ROOT"), "root location of zar archive to walk")
	f.BoolVar(&c.lflag, "l", false, "long form")
	f.BoolVar(&c.relativePaths, "relative", false, "display paths relative to root")
	f.BoolVar(&c.indexDesc, "desc", false, "display index description in lieu of index file name")
	f.BoolVar(&c.showRanges, "ranges", false, "display time ranges instead of paths")
	f.BoolVar(&c.spanInfos, "spaninfos", false, "group chunks by span infos")
	return c, nil
}

func (c *Command) Run(args []string) error {
	if len(args) > 1 {
		return errors.New("zar ls: too many arguments")
	}

	var err error
	c.ark, err = archive.OpenArchive(c.root, nil)
	if err != nil {
		return err
	}

	defs, err := c.ark.ReadDefinitions(context.TODO())
	if err != nil {
		return err
	}

	c.defs = defs.Map()

	var pattern string
	if len(args) == 1 {
		pattern = args[0]
	}
	if c.spanInfos {
		return archive.SpanWalk(context.TODO(), c.ark, nano.MaxSpan, func(si archive.SpanInfo) error {
			c.printSpanInfo(si, pattern)
			return nil
		})
	}
	return archive.Walk(context.TODO(), c.ark, func(chunk chunk.Chunk) error {
		c.printChunk(0, chunk, pattern)
		return nil
	})
}

func (c *Command) printSpanInfo(si archive.SpanInfo, pattern string) {
	fmt.Println(si.Range(c.ark.DataOrder) + ":")
	for _, chunk := range si.Chunks {
		c.printChunk(1, chunk, pattern)
	}
}

func (c *Command) printChunk(indent int, chunk chunk.Chunk, pattern string) {
	str := c.chunkString(chunk)
	var children string
	if !c.lflag {
		fmt.Println(strings.Repeat("    ", indent) + str)
		return
	}
	str += "/"
	children = c.indicesString(str, chunk)
	children += c.mapsString(str, chunk, pattern)
	fmt.Print(children)
}

func (c *Command) indicesString(prefix string, chunk chunk.Chunk) string {
	indices, err := index.Indices(context.TODO(), chunk.ZarDir(), c.defs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error listing directory: %v", err)
		return ""
	}
	var b strings.Builder
	for _, i := range indices {
		var str string
		if c.indexDesc {
			str = i.Definition.String()
		} else {
			str = i.Filename()
		}
		fmt.Fprintln(&b, prefix+str)
	}
	return b.String()
}

func (c *Command) mapsString(prefix string, chunk chunk.Chunk, pattern string) string {
	files, err := index.ListFilenames(context.TODO(), chunk.ZarDir())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error listing directory: %v", err)
		return ""
	}
	var b strings.Builder
	for _, file := range files {
		if pattern == "" || pattern == file {
			fmt.Fprintln(&b, prefix+file)
		}
	}
	return b.String()
}

func (c *Command) chunkString(chunk chunk.Chunk) string {
	if c.showRanges {
		return chunk.Range()
	}
	if c.relativePaths {
		return strings.TrimSuffix(c.ark.DataPath.RelPath(chunk.ZarDir()), "/")
	}
	return chunk.ZarDir().String()
}

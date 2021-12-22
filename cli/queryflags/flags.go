package queryflags

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zson"
)

var ErrNoHEAD = errors.New("HEAD not specified: indicate with -use or run the \"use\" command")

type Flags struct {
	Verbose  bool
	Stats    bool
	Includes Includes
}

func (f *Flags) SetFlags(fs *flag.FlagSet) {
	fs.BoolVar(&f.Stats, "s", false, "display search stats on stderr")
	fs.Var(&f.Includes, "I", "source file containing Zed query text (may be used multiple times)")
}

func (f *Flags) ParseSourcesAndInputs(paths []string) ([]string, ast.Proc, error) {
	var src string
	if len(paths) != 0 && !cli.FileExists(paths[0]) && !isURLWithKnownScheme(paths[0], "http", "https", "s3") {
		if len(paths) == 1 {
			// We don't interpret the first arg as a query if there
			// are no additional args.
			return nil, nil, fmt.Errorf("no such file: %s", paths[0])
		}
		src = paths[0]
		paths = paths[1:]
	}
	query, err := compiler.ParseProc(src, f.Includes...)
	if err != nil {
		return nil, nil, err
	}
	return paths, query, nil
}

func isURLWithKnownScheme(path string, schemes ...string) bool {
	u, err := url.Parse(path)
	if err != nil {
		return false
	}
	for _, s := range schemes {
		if u.Scheme == s {
			return true
		}
	}
	return false
}

func (f *Flags) PrintStats(stats zbuf.Progress) {
	if f.Stats {
		out, err := zson.Marshal(stats)
		if err != nil {
			out = fmt.Sprintf("error marshaling stats: %s", err)
		}
		fmt.Fprintln(os.Stderr, out)
	}
}

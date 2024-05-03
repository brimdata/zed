package queryflags

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/cli/clierrors"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/data"
	"github.com/brimdata/zed/compiler/parser"
	"github.com/brimdata/zed/compiler/semantic"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zson"
	"go.uber.org/multierr"
)

type Flags struct {
	Verbose  bool
	Stats    bool
	Includes Includes
}

func (f *Flags) SetFlags(fs *flag.FlagSet) {
	fs.BoolVar(&f.Stats, "s", false, "display search stats on stderr")
	fs.Var(&f.Includes, "I", "source file containing Zed query text (may be used multiple times)")
}

func (f *Flags) ParseSourcesAndInputs(paths []string) ([]string, ast.Seq, *parser.SourceSet, bool, error) {
	var src string
	if len(paths) != 0 && !cli.FileExists(paths[0]) && !isURLWithKnownScheme(paths[0], "http", "https", "s3") {
		src = paths[0]
		paths = paths[1:]
		if len(paths) == 0 {
			// Consider a lone argument to be a query if it compiles
			// and appears to start with a from or yield operator.
			// Otherwise, consider it a file.
			query, set, err := parse(src, f.Includes...)
			if err == nil {
				if s, err := semantic.Analyze(context.Background(), query, data.NewSource(storage.NewLocalEngine(), nil), nil); err == nil {
					if semantic.HasSource(s) {
						return nil, query, set, false, nil
					}
					if semantic.StartsWithYield(s) {
						return nil, query, set, true, nil
					}
				}
			}
			return nil, nil, nil, false, singleArgError(src, set, err)
		}
	}
	query, set, err := parse(src, f.Includes...)
	if err != nil {
		return nil, nil, set, false, clierrors.Format(set, err)
	}
	return paths, query, set, false, nil
}

func parse(src string, includes ...string) (ast.Seq, *parser.SourceSet, error) {
	set, err := parser.ConcatSource(includes, src)
	if err != nil {
		return nil, nil, err
	}
	seq, err := parser.ParseZed(string(set.Contents))
	return seq, set, err
}

func isURLWithKnownScheme(path string, schemes ...string) bool {
	u, err := url.Parse(path)
	if err != nil {
		return false
	}
	return slices.Contains(schemes, u.Scheme)
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

func singleArgError(src string, set *parser.SourceSet, err error) error {
	var b strings.Builder
	b.WriteString("could not invoke zq with a single argument because:")
	if len(src) > 20 {
		src = src[:20] + "..."
	}
	fmt.Fprintf(&b, "\n - a file could not be found with the name %q", src)
	if hasASTErrors(err) {
		b.WriteString("\n - the argument could not be compiled as a valid Zed query:")
		for _, line := range strings.Split(clierrors.Format(set, err).Error(), "\n") {
			fmt.Fprintf(&b, "\n   %s", line)
		}
	} else {
		b.WriteString("\n - the argument did not parse as a valid Zed query")
	}
	return errors.New(b.String())
}

func hasASTErrors(errs error) bool {
	for _, err := range multierr.Errors(errs) {
		var aerr *ast.Error
		if errors.As(err, &aerr) {
			return true
		}
	}
	return false
}

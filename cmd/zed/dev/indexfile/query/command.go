package query

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cmd/zed/dev/indexfile"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/index"
	"github.com/brimdata/zed/lake/mock"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zson"
)

var Query = &charm.Spec{
	Name:  "query",
	Usage: "query expr index",
	Short: "query matching entries from boolean expression",
	New:   newQueryCommand,
}

func init() {
	indexfile.Cmd.Add(Query)
}

type QueryCommand struct {
	*indexfile.Command
	outputFlags outputflags.Flags
}

func newQueryCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &QueryCommand{Command: parent.(*indexfile.Command)}
	c.outputFlags.SetFlags(f)
	return c, nil
}

func (c *QueryCommand) Run(args []string) error {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	ctx, cleanup, err := c.Init(&c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) < 2 {
		return fmt.Errorf("need query and file args")
	}
	e, err := parseFilter(args[0])
	if err != nil {
		return err
	}
	uri, err := storage.ParseURI(args[1])
	if err != nil {
		return err
	}
	local := storage.NewLocalEngine()
	finder, err := index.NewFinder(ctx, zed.NewContext(), local, uri)
	if err != nil {
		return err
	}
	vals := make(chan *zed.Value)
	var filterErr error
	go func() {
		filterErr = finder.Filter(ctx, vals, e)
		close(vals)
	}()
	for val := range vals {
		fmt.Println(zson.String(val))
	}
	return filterErr
	// return nil
}

func parseFilter(src string) (dag.Expr, error) {
	p, err := compiler.ParseOp(src)
	if err != nil {
		return nil, err
	}
	runtime, err := compiler.New(op.DefaultContext(), p, mock.NewLake(), nil)
	if err != nil {
		return nil, err
	}
	seq := runtime.Entry().(*dag.Sequential)
	for _, op := range seq.Ops {
		f, ok := op.(*dag.Filter)
		if ok {
			return f.Expr, nil
		}
	}
	return nil, errors.New("no filter found in query")
}

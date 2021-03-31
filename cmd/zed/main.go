package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/cmd/zed/api"
	_ "github.com/brimdata/zed/cmd/zed/api/auth"
	_ "github.com/brimdata/zed/cmd/zed/api/get"
	_ "github.com/brimdata/zed/cmd/zed/api/index"
	_ "github.com/brimdata/zed/cmd/zed/api/info"
	_ "github.com/brimdata/zed/cmd/zed/api/intake"
	_ "github.com/brimdata/zed/cmd/zed/api/new"
	_ "github.com/brimdata/zed/cmd/zed/api/post"
	_ "github.com/brimdata/zed/cmd/zed/api/rename"
	_ "github.com/brimdata/zed/cmd/zed/api/repl"
	_ "github.com/brimdata/zed/cmd/zed/api/rm"
	_ "github.com/brimdata/zed/cmd/zed/api/version"
	"github.com/brimdata/zed/cmd/zed/ast"
	"github.com/brimdata/zed/cmd/zed/lake"
	_ "github.com/brimdata/zed/cmd/zed/lake/compact"
	_ "github.com/brimdata/zed/cmd/zed/lake/find"
	_ "github.com/brimdata/zed/cmd/zed/lake/import"
	_ "github.com/brimdata/zed/cmd/zed/lake/index"
	_ "github.com/brimdata/zed/cmd/zed/lake/ls"
	_ "github.com/brimdata/zed/cmd/zed/lake/map"
	_ "github.com/brimdata/zed/cmd/zed/lake/query"
	_ "github.com/brimdata/zed/cmd/zed/lake/rm"
	_ "github.com/brimdata/zed/cmd/zed/lake/rmdirs"
	_ "github.com/brimdata/zed/cmd/zed/lake/stat"
	"github.com/brimdata/zed/cmd/zed/query"
	"github.com/brimdata/zed/cmd/zed/zst"
	_ "github.com/brimdata/zed/cmd/zed/zst/create"
	_ "github.com/brimdata/zed/cmd/zed/zst/cut"
	_ "github.com/brimdata/zed/cmd/zed/zst/inspect"
	_ "github.com/brimdata/zed/cmd/zed/zst/read"
	"github.com/brimdata/zed/pkg/charm"

	"github.com/brimdata/zed/cmd/zed/index"
	_ "github.com/brimdata/zed/cmd/zed/index/convert"
	_ "github.com/brimdata/zed/cmd/zed/index/create"
	_ "github.com/brimdata/zed/cmd/zed/index/lookup"
	_ "github.com/brimdata/zed/cmd/zed/index/section"
	_ "github.com/brimdata/zed/cmd/zed/index/seek"
)

func main() {
	zed.Add(charm.Help)
	zed.Add(api.Cmd)
	zed.Add(ast.Cmd)
	zed.Add(query.Cmd)
	zed.Add(zst.Cmd)
	zed.Add(lake.Cmd)
	zed.Add(index.Cmd)
	if _, err := zed.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

var zed = &charm.Spec{
	Name:  "zed",
	Usage: "zed <command> [options] [arguments...]",
	Short: "run zed commands",
	Long: `
zed is a command-line tool for creating, configuring, ingesting into,
querying, and orchestrating zed data lakes.`,
	New: newCmd,
}

type root struct {
	charm.Command
	cli cli.Flags
}

func newCmd(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	r := &root{}
	r.cli.SetFlags(f)
	return r, nil
}

func (r *root) Cleanup() {
	r.cli.Cleanup()
}

func (r *root) Init(all ...cli.Initializer) error {
	return r.cli.Init(all...)
}

func (r *root) Run(args []string) error {
	if len(args) == 0 {
		return charm.NeedHelp
	}
	return charm.ErrNoRun
}

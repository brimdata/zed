package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/brimdata/zq/cli"
	"github.com/brimdata/zq/cmd/zed/api"
	_ "github.com/brimdata/zq/cmd/zed/api/auth"
	_ "github.com/brimdata/zq/cmd/zed/api/get"
	_ "github.com/brimdata/zq/cmd/zed/api/index"
	_ "github.com/brimdata/zq/cmd/zed/api/info"
	_ "github.com/brimdata/zq/cmd/zed/api/intake"
	_ "github.com/brimdata/zq/cmd/zed/api/new"
	_ "github.com/brimdata/zq/cmd/zed/api/post"
	_ "github.com/brimdata/zq/cmd/zed/api/rename"
	_ "github.com/brimdata/zq/cmd/zed/api/repl"
	_ "github.com/brimdata/zq/cmd/zed/api/rm"
	_ "github.com/brimdata/zq/cmd/zed/api/version"
	"github.com/brimdata/zq/cmd/zed/ast"
	"github.com/brimdata/zq/cmd/zed/q"
	"github.com/brimdata/zq/cmd/zed/zst"
	_ "github.com/brimdata/zq/cmd/zed/zst/create"
	_ "github.com/brimdata/zq/cmd/zed/zst/cut"
	_ "github.com/brimdata/zq/cmd/zed/zst/inspect"
	_ "github.com/brimdata/zq/cmd/zed/zst/read"
	"github.com/brimdata/zq/pkg/charm"
)

func main() {
	zed.Add(charm.Help)
	zed.Add(api.Cmd)
	zed.Add(ast.Cmd)
	zed.Add(q.Cmd)
	zed.Add(zst.Cmd)
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

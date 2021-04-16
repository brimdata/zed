package main

import (
	"fmt"
	"os"

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
	"github.com/brimdata/zed/cmd/zed/index"
	_ "github.com/brimdata/zed/cmd/zed/index/convert"
	_ "github.com/brimdata/zed/cmd/zed/index/create"
	_ "github.com/brimdata/zed/cmd/zed/index/lookup"
	_ "github.com/brimdata/zed/cmd/zed/index/section"
	_ "github.com/brimdata/zed/cmd/zed/index/seek"
	"github.com/brimdata/zed/cmd/zed/lake"
	_ "github.com/brimdata/zed/cmd/zed/lake/add"
	_ "github.com/brimdata/zed/cmd/zed/lake/commit"
	_ "github.com/brimdata/zed/cmd/zed/lake/create"
	_ "github.com/brimdata/zed/cmd/zed/lake/delete"
	_ "github.com/brimdata/zed/cmd/zed/lake/drop"
	_ "github.com/brimdata/zed/cmd/zed/lake/find"
	_ "github.com/brimdata/zed/cmd/zed/lake/index"
	_ "github.com/brimdata/zed/cmd/zed/lake/init"
	_ "github.com/brimdata/zed/cmd/zed/lake/load"
	_ "github.com/brimdata/zed/cmd/zed/lake/log"
	_ "github.com/brimdata/zed/cmd/zed/lake/ls"
	_ "github.com/brimdata/zed/cmd/zed/lake/merge"
	_ "github.com/brimdata/zed/cmd/zed/lake/query"
	_ "github.com/brimdata/zed/cmd/zed/lake/squash"
	_ "github.com/brimdata/zed/cmd/zed/lake/stat"
	_ "github.com/brimdata/zed/cmd/zed/lake/status"
	_ "github.com/brimdata/zed/cmd/zed/lake/vacate"
	"github.com/brimdata/zed/cmd/zed/query"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/cmd/zed/zst"
	_ "github.com/brimdata/zed/cmd/zed/zst/create"
	_ "github.com/brimdata/zed/cmd/zed/zst/cut"
	_ "github.com/brimdata/zed/cmd/zed/zst/inspect"
	_ "github.com/brimdata/zed/cmd/zed/zst/read"
	"github.com/brimdata/zed/pkg/charm"
)

func main() {
	zed := root.Zed
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

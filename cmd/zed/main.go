package main

import (
	"fmt"
	"os"

	"github.com/brimdata/zed/cmd/zed/api"
	"github.com/brimdata/zed/cmd/zed/compile"
	"github.com/brimdata/zed/cmd/zed/index"
	_ "github.com/brimdata/zed/cmd/zed/index/convert"
	_ "github.com/brimdata/zed/cmd/zed/index/create"
	_ "github.com/brimdata/zed/cmd/zed/index/lookup"
	_ "github.com/brimdata/zed/cmd/zed/index/section"
	"github.com/brimdata/zed/cmd/zed/lake"
	_ "github.com/brimdata/zed/cmd/zed/lake/branch"
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
	_ "github.com/brimdata/zed/cmd/zed/lake/rename"
	_ "github.com/brimdata/zed/cmd/zed/lake/revert"
	_ "github.com/brimdata/zed/cmd/zed/lake/serve"
	_ "github.com/brimdata/zed/cmd/zed/lake/vacate"
	"github.com/brimdata/zed/cmd/zed/query"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/cmd/zed/zst"
	_ "github.com/brimdata/zed/cmd/zed/zst/create"
	_ "github.com/brimdata/zed/cmd/zed/zst/cut"
	_ "github.com/brimdata/zed/cmd/zed/zst/inspect"
	_ "github.com/brimdata/zed/cmd/zed/zst/read"
)

func main() {
	zed := root.Zed
	zed.Add(api.Cmd)
	zed.Add(compile.Cmd)
	zed.Add(query.Cmd)
	zed.Add(zst.Cmd)
	zed.Add(lake.Cmd)
	zed.Add(index.Cmd)
	if err := zed.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

package main

import (
	"fmt"
	"os"

	"github.com/brimdata/zed/cmd/zed/api"
	"github.com/brimdata/zed/cmd/zed/dev"
	_ "github.com/brimdata/zed/cmd/zed/dev/compile"
	_ "github.com/brimdata/zed/cmd/zed/dev/indexfile"
	_ "github.com/brimdata/zed/cmd/zed/dev/indexfile/convert"
	_ "github.com/brimdata/zed/cmd/zed/dev/indexfile/create"
	_ "github.com/brimdata/zed/cmd/zed/dev/indexfile/lookup"
	_ "github.com/brimdata/zed/cmd/zed/dev/indexfile/section"
	_ "github.com/brimdata/zed/cmd/zed/dev/zst"
	_ "github.com/brimdata/zed/cmd/zed/dev/zst/create"
	_ "github.com/brimdata/zed/cmd/zed/dev/zst/cut"
	_ "github.com/brimdata/zed/cmd/zed/dev/zst/inspect"
	_ "github.com/brimdata/zed/cmd/zed/dev/zst/read"
	"github.com/brimdata/zed/cmd/zed/lake"
	_ "github.com/brimdata/zed/cmd/zed/lake/auth"
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
	_ "github.com/brimdata/zed/cmd/zed/lake/use"
	_ "github.com/brimdata/zed/cmd/zed/lake/vacate"
	"github.com/brimdata/zed/cmd/zed/query"
	"github.com/brimdata/zed/cmd/zed/root"
)

func main() {
	zed := root.Zed
	zed.Add(api.Cmd)
	zed.Add(dev.Cmd)
	zed.Add(query.Cmd)
	zed.Add(lake.Cmd)
	if err := zed.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

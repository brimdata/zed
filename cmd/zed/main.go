package main

import (
	"fmt"
	"os"

	"github.com/brimdata/zed/cmd/zed/auth"
	"github.com/brimdata/zed/cmd/zed/branch"
	"github.com/brimdata/zed/cmd/zed/compact"
	"github.com/brimdata/zed/cmd/zed/create"
	zeddelete "github.com/brimdata/zed/cmd/zed/delete"
	"github.com/brimdata/zed/cmd/zed/dev"
	_ "github.com/brimdata/zed/cmd/zed/dev/compile"
	_ "github.com/brimdata/zed/cmd/zed/dev/dig/frames"
	_ "github.com/brimdata/zed/cmd/zed/dev/dig/section"
	_ "github.com/brimdata/zed/cmd/zed/dev/dig/slice"
	_ "github.com/brimdata/zed/cmd/zed/dev/dig/trailer"
	_ "github.com/brimdata/zed/cmd/zed/dev/indexfile"
	_ "github.com/brimdata/zed/cmd/zed/dev/indexfile/create"
	_ "github.com/brimdata/zed/cmd/zed/dev/indexfile/lookup"
	_ "github.com/brimdata/zed/cmd/zed/dev/zst"
	_ "github.com/brimdata/zed/cmd/zed/dev/zst/create"
	_ "github.com/brimdata/zed/cmd/zed/dev/zst/cut"
	_ "github.com/brimdata/zed/cmd/zed/dev/zst/read"
	"github.com/brimdata/zed/cmd/zed/drop"
	"github.com/brimdata/zed/cmd/zed/index"
	zedinit "github.com/brimdata/zed/cmd/zed/init"
	"github.com/brimdata/zed/cmd/zed/load"
	"github.com/brimdata/zed/cmd/zed/log"
	"github.com/brimdata/zed/cmd/zed/ls"
	"github.com/brimdata/zed/cmd/zed/merge"
	"github.com/brimdata/zed/cmd/zed/query"
	"github.com/brimdata/zed/cmd/zed/rename"
	"github.com/brimdata/zed/cmd/zed/revert"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/cmd/zed/serve"
	"github.com/brimdata/zed/cmd/zed/use"
	"github.com/brimdata/zed/cmd/zed/vacate"
)

func main() {
	zed := root.Zed
	zed.Add(auth.Cmd)
	zed.Add(branch.Cmd)
	zed.Add(compact.Cmd)
	zed.Add(create.Cmd)
	zed.Add(zeddelete.Cmd)
	zed.Add(drop.Cmd)
	zed.Add(index.Cmd)
	zed.Add(zedinit.Cmd)
	zed.Add(load.Cmd)
	zed.Add(log.Cmd)
	zed.Add(ls.Cmd)
	zed.Add(merge.Cmd)
	zed.Add(query.Cmd)
	zed.Add(rename.Cmd)
	zed.Add(revert.Cmd)
	zed.Add(serve.Cmd)
	zed.Add(use.Cmd)
	zed.Add(vacate.Cmd)
	zed.Add(dev.Cmd)
	if err := root.Zed.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

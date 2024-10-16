package main

import (
	"fmt"
	"os"

	"github.com/brimdata/super/cmd/zed/auth"
	"github.com/brimdata/super/cmd/zed/branch"
	"github.com/brimdata/super/cmd/zed/compact"
	"github.com/brimdata/super/cmd/zed/create"
	zeddelete "github.com/brimdata/super/cmd/zed/delete"
	"github.com/brimdata/super/cmd/zed/dev"
	_ "github.com/brimdata/super/cmd/zed/dev/compile"
	_ "github.com/brimdata/super/cmd/zed/dev/dig/frames"
	_ "github.com/brimdata/super/cmd/zed/dev/dig/slice"
	_ "github.com/brimdata/super/cmd/zed/dev/vector/agg"
	_ "github.com/brimdata/super/cmd/zed/dev/vector/copy"
	_ "github.com/brimdata/super/cmd/zed/dev/vector/project"
	_ "github.com/brimdata/super/cmd/zed/dev/vector/query"
	_ "github.com/brimdata/super/cmd/zed/dev/vector/search"
	_ "github.com/brimdata/super/cmd/zed/dev/vng"
	"github.com/brimdata/super/cmd/zed/drop"
	zedinit "github.com/brimdata/super/cmd/zed/init"
	"github.com/brimdata/super/cmd/zed/load"
	"github.com/brimdata/super/cmd/zed/log"
	"github.com/brimdata/super/cmd/zed/ls"
	"github.com/brimdata/super/cmd/zed/manage"
	"github.com/brimdata/super/cmd/zed/merge"
	"github.com/brimdata/super/cmd/zed/query"
	"github.com/brimdata/super/cmd/zed/rename"
	"github.com/brimdata/super/cmd/zed/revert"
	"github.com/brimdata/super/cmd/zed/root"
	"github.com/brimdata/super/cmd/zed/serve"
	"github.com/brimdata/super/cmd/zed/use"
	"github.com/brimdata/super/cmd/zed/vacate"
	"github.com/brimdata/super/cmd/zed/vacuum"
	"github.com/brimdata/super/cmd/zed/vector"
)

func main() {
	zed := root.Zed
	zed.Add(auth.Cmd)
	zed.Add(branch.Cmd)
	zed.Add(compact.Cmd)
	zed.Add(create.Cmd)
	zed.Add(zeddelete.Cmd)
	zed.Add(drop.Cmd)
	zed.Add(zedinit.Cmd)
	zed.Add(load.Cmd)
	zed.Add(log.Cmd)
	zed.Add(ls.Cmd)
	zed.Add(manage.Cmd)
	zed.Add(merge.Cmd)
	zed.Add(query.Cmd)
	zed.Add(rename.Cmd)
	zed.Add(revert.Cmd)
	zed.Add(serve.Cmd)
	zed.Add(use.Cmd)
	zed.Add(vacate.Cmd)
	zed.Add(vacuum.Cmd)
	zed.Add(vector.Cmd)
	zed.Add(dev.Cmd)
	if err := root.Zed.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

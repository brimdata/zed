package main

import (
	"fmt"
	"os"

	"github.com/brimdata/zed/cmd/zed/zst"
	_ "github.com/brimdata/zed/cmd/zed/zst/create"
	_ "github.com/brimdata/zed/cmd/zed/zst/cut"
	_ "github.com/brimdata/zed/cmd/zed/zst/inspect"
	_ "github.com/brimdata/zed/cmd/zed/zst/read"
	"github.com/brimdata/zed/pkg/charm"
)

func main() {
	zst.Cmd.Add(charm.Help)
	if _, err := zst.Cmd.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

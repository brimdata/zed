package main

import (
	"fmt"
	"os"

	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/cmd/zed/zst"
	_ "github.com/brimdata/zed/cmd/zed/zst/create"
	_ "github.com/brimdata/zed/cmd/zed/zst/cut"
	_ "github.com/brimdata/zed/cmd/zed/zst/inspect"
	_ "github.com/brimdata/zed/cmd/zed/zst/read"
)

func main() {
	root.Zed.Add(zst.Cmd)
	args := append([]string{"zst"}, os.Args[1:]...)
	if err := root.Zed.ExecRoot(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

package main

import (
	"fmt"
	"os"

	"github.com/brimdata/zed/cmd/zed/dev/zst"
	_ "github.com/brimdata/zed/cmd/zed/dev/zst/create"
	_ "github.com/brimdata/zed/cmd/zed/dev/zst/cut"
	_ "github.com/brimdata/zed/cmd/zed/dev/zst/read"
	"github.com/brimdata/zed/cmd/zed/root"
)

func main() {
	root.Zed.Add(zst.Cmd)
	args := append([]string{"zst"}, os.Args[1:]...)
	if err := root.Zed.ExecRoot(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

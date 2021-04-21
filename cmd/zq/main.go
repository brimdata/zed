package main

import (
	"fmt"
	"os"

	"github.com/brimdata/zed/cmd/zed/query"
	"github.com/brimdata/zed/cmd/zed/root"
)

func main() {
	root.Zed.Add(query.Cmd)
	args := append([]string{"query"}, os.Args[1:]...)
	if err := root.Zed.ExecRoot(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

package main

import (
	"fmt"
	"os"

	"github.com/brimdata/zed/cmd/zed/ast"
	"github.com/brimdata/zed/cmd/zed/root"
)

func main() {
	root.Zed.Add(ast.Cmd)
	args := append([]string{"ast"}, os.Args[1:]...)
	if err := root.Zed.ExecRoot(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

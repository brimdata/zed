package main

import (
	"fmt"
	"os"

	"github.com/brimdata/zed/cmd/zed/ast"
	"github.com/brimdata/zed/pkg/charm"
)

func main() {
	ast.Cmd.Add(charm.Help)
	if _, err := ast.Cmd.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

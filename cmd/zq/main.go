package main

import (
	"fmt"
	"os"

	"github.com/brimdata/zed/cmd/zed/query"
	"github.com/brimdata/zed/pkg/charm"
)

func main() {
	query.Cmd.Add(charm.Help)
	if _, err := query.Cmd.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

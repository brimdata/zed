package main

import (
	"fmt"
	"os"

	"github.com/brimdata/zq/cmd/zed/q"
	"github.com/brimdata/zq/pkg/charm"
)

func main() {
	q.Cmd.Add(charm.Help)
	if _, err := q.Cmd.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

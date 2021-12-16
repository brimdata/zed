package main

import (
	"fmt"
	"os"

	"github.com/brimdata/zed/cli/zq"
)

func main() {
	if err := zq.Cmd.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

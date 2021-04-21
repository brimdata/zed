package main

import (
	"fmt"
	"os"

	_ "github.com/brimdata/zed/ppl/cmd/pgctl/migrate"
	_ "github.com/brimdata/zed/ppl/cmd/pgctl/rmtestdb"
	"github.com/brimdata/zed/ppl/cmd/pgctl/root"
	_ "github.com/brimdata/zed/ppl/cmd/pgctl/testdb"
)

func main() {
	if err := root.CLI.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

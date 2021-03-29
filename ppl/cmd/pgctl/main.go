package main

import (
	"fmt"
	"os"

	_ "github.com/brimdata/zq/ppl/cmd/pgctl/migrate"
	_ "github.com/brimdata/zq/ppl/cmd/pgctl/rmtestdb"
	"github.com/brimdata/zq/ppl/cmd/pgctl/root"
	_ "github.com/brimdata/zq/ppl/cmd/pgctl/testdb"
)

func main() {
	_, err := root.CLI.ExecRoot(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

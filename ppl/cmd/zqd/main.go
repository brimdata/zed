package main

import (
	"fmt"
	"os"

	_ "github.com/brimdata/zed/ppl/cmd/zqd/listen"
	"github.com/brimdata/zed/ppl/cmd/zqd/root"
)

func main() {
	if err := root.Zqd.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

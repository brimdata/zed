package main

import (
	"fmt"
	"os"

	"github.com/mccanne/zq/cmd"
)

func main() {
	if _, err := cmd.Zq.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

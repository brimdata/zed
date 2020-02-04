package main

import (
	"fmt"
	"os"

	_ "github.com/mccanne/zq/cmd/zqd/listen"
	root "github.com/mccanne/zq/cmd/zqd/root"
)

func main() {
	if _, err := root.Zqd.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

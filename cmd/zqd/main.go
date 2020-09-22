package main

import (
	"fmt"
	"os"

	_ "github.com/brimsec/zq/cmd/zqd/listen"
	root "github.com/brimsec/zq/cmd/zqd/root"
	_ "github.com/brimsec/zq/cmd/zqd/winexec"
)

func main() {
	if _, err := root.Zqd.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

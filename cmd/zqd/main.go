package main

import (
	"fmt"
	"os"

	_ "github.com/brimsec/zq/cmd/zqd/listen"
	root "github.com/brimsec/zq/cmd/zqd/root"
	"github.com/brimsec/zq/zqd"
)

// Version is set via the Go linker.
var version = "unknown"

func main() {
	//XXX
	zqd.Version.Zq = version
	zqd.Version.Zqd = version
	if _, err := root.Zqd.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

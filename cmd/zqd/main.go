package main

import (
	"fmt"
	"os"

	_ "github.com/mccanne/zq/cmd/zqd/listen"
	"github.com/mccanne/zq/cmd/zqd/root"
	"github.com/mccanne/zq/zqd"
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

package main

import (
	"fmt"
	"os"

	_ "github.com/brimsec/zq/cmd/zar/chop"
	_ "github.com/brimsec/zq/cmd/zar/find"
	_ "github.com/brimsec/zq/cmd/zar/index"
	_ "github.com/brimsec/zq/cmd/zar/mkdirs"
	_ "github.com/brimsec/zq/cmd/zar/rmdirs"
	"github.com/brimsec/zq/cmd/zar/root"
)

// Version is set via the Go linker.
var version = "unknown"

func main() {
	//XXX
	//root.Version = version
	if _, err := root.Zar.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

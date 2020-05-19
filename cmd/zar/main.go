package main

import (
	"fmt"
	"os"

	_ "github.com/brimsec/zq/cmd/zar/find"
	_ "github.com/brimsec/zq/cmd/zar/import"
	_ "github.com/brimsec/zq/cmd/zar/index"
	_ "github.com/brimsec/zq/cmd/zar/ls"
	_ "github.com/brimsec/zq/cmd/zar/rm"
	_ "github.com/brimsec/zq/cmd/zar/rmdirs"
	"github.com/brimsec/zq/cmd/zar/root"
	_ "github.com/brimsec/zq/cmd/zar/zdx"
	_ "github.com/brimsec/zq/cmd/zar/zq"
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

package main

import (
	"fmt"
	"os"

	"github.com/brimsec/zq/cmd/zed/zst"
	_ "github.com/brimsec/zq/cmd/zed/zst/create"
	_ "github.com/brimsec/zq/cmd/zed/zst/cut"
	_ "github.com/brimsec/zq/cmd/zed/zst/inspect"
	_ "github.com/brimsec/zq/cmd/zed/zst/read"
	"github.com/brimsec/zq/pkg/charm"
)

func main() {
	zst.Cmd.Add(charm.Help)
	_, err := zst.Cmd.ExecRoot(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

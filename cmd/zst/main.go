package main

import (
	"fmt"
	"os"

	_ "github.com/brimsec/zq/cmd/zst/create"
	_ "github.com/brimsec/zq/cmd/zst/cut"
	_ "github.com/brimsec/zq/cmd/zst/inspect"
	_ "github.com/brimsec/zq/cmd/zst/read"
	"github.com/brimsec/zq/cmd/zst/root"
)

func main() {
	_, err := root.Zst.ExecRoot(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

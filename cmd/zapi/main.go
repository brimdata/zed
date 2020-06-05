package main

import (
	"fmt"
	"os"

	"github.com/brimsec/zq/cmd/zapi/cmd"
	_ "github.com/brimsec/zq/cmd/zapi/cmd/get"
	_ "github.com/brimsec/zq/cmd/zapi/cmd/info"
	_ "github.com/brimsec/zq/cmd/zapi/cmd/new"
	_ "github.com/brimsec/zq/cmd/zapi/cmd/newsubspace"
	_ "github.com/brimsec/zq/cmd/zapi/cmd/post"
	_ "github.com/brimsec/zq/cmd/zapi/cmd/rename"
	_ "github.com/brimsec/zq/cmd/zapi/cmd/rm"
)

// These variables are populated via the Go linker.
var (
	version   = "unknown"
	zqVersion = "unknown"
)

func main() {
	cmd.Version = version
	cmd.ZqVersion = zqVersion
	_, err := cmd.CLI.ExecRoot(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

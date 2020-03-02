package main

import (
	"fmt"
	"os"

	"github.com/brimsec/zq/cmd/zqdcli/cmd"
	_ "github.com/brimsec/zq/cmd/zqdcli/cmd/get"
	_ "github.com/brimsec/zq/cmd/zqdcli/cmd/info"
	_ "github.com/brimsec/zq/cmd/zqdcli/cmd/new"
)

// These variables are populated via the Go linker.
var (
	version   = "unknown"
	zqVersion = "unknown"
)

func main() {
	cmd.Version = version
	cmd.ZqVersion = zqVersion
	_, err := cmd.Cli.ExecRoot(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

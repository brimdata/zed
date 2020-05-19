package main

import (
	"fmt"
	"os"

	"github.com/brimsec/zq/cmd/zapi/cmd"
	_ "github.com/brimsec/zq/cmd/zapi/cmd/info"
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

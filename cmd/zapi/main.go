package main

import (
	"fmt"
	"os"

	"github.com/brimsec/zq/cmd/zapi/cmd"
	_ "github.com/brimsec/zq/cmd/zapi/cmd/auth"
	_ "github.com/brimsec/zq/cmd/zapi/cmd/get"
	_ "github.com/brimsec/zq/cmd/zapi/cmd/index"
	_ "github.com/brimsec/zq/cmd/zapi/cmd/info"
	_ "github.com/brimsec/zq/cmd/zapi/cmd/intake"
	_ "github.com/brimsec/zq/cmd/zapi/cmd/new"
	_ "github.com/brimsec/zq/cmd/zapi/cmd/post"
	_ "github.com/brimsec/zq/cmd/zapi/cmd/rename"
	_ "github.com/brimsec/zq/cmd/zapi/cmd/repl"
	_ "github.com/brimsec/zq/cmd/zapi/cmd/rm"
	_ "github.com/brimsec/zq/cmd/zapi/cmd/version"
)

func main() {
	_, err := cmd.CLI.ExecRoot(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

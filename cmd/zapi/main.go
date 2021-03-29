package main

import (
	"fmt"
	"os"

	"github.com/brimsec/zq/cmd/zed/api"
	_ "github.com/brimsec/zq/cmd/zed/api/auth"
	_ "github.com/brimsec/zq/cmd/zed/api/get"
	_ "github.com/brimsec/zq/cmd/zed/api/index"
	_ "github.com/brimsec/zq/cmd/zed/api/info"
	_ "github.com/brimsec/zq/cmd/zed/api/intake"
	_ "github.com/brimsec/zq/cmd/zed/api/new"
	_ "github.com/brimsec/zq/cmd/zed/api/post"
	_ "github.com/brimsec/zq/cmd/zed/api/rename"
	_ "github.com/brimsec/zq/cmd/zed/api/repl"
	_ "github.com/brimsec/zq/cmd/zed/api/rm"
	_ "github.com/brimsec/zq/cmd/zed/api/version"
	"github.com/brimsec/zq/pkg/charm"
)

func main() {
	api.Cmd.Add(charm.Help)
	if _, err := api.Cmd.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

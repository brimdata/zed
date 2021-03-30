package main

import (
	"fmt"
	"os"

	"github.com/brimdata/zed/cmd/zed/api"
	_ "github.com/brimdata/zed/cmd/zed/api/auth"
	_ "github.com/brimdata/zed/cmd/zed/api/get"
	_ "github.com/brimdata/zed/cmd/zed/api/index"
	_ "github.com/brimdata/zed/cmd/zed/api/info"
	_ "github.com/brimdata/zed/cmd/zed/api/intake"
	_ "github.com/brimdata/zed/cmd/zed/api/new"
	_ "github.com/brimdata/zed/cmd/zed/api/post"
	_ "github.com/brimdata/zed/cmd/zed/api/rename"
	_ "github.com/brimdata/zed/cmd/zed/api/repl"
	_ "github.com/brimdata/zed/cmd/zed/api/rm"
	_ "github.com/brimdata/zed/cmd/zed/api/version"
	"github.com/brimdata/zed/pkg/charm"
)

func main() {
	api.Cmd.Add(charm.Help)
	if _, err := api.Cmd.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

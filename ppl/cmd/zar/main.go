package main

import (
	"fmt"
	"os"

	_ "github.com/brimsec/zq/ppl/cmd/zar/compact"
	_ "github.com/brimsec/zq/ppl/cmd/zar/find"
	_ "github.com/brimsec/zq/ppl/cmd/zar/import"
	_ "github.com/brimsec/zq/ppl/cmd/zar/index"
	_ "github.com/brimsec/zq/ppl/cmd/zar/ls"
	_ "github.com/brimsec/zq/ppl/cmd/zar/map"
	_ "github.com/brimsec/zq/ppl/cmd/zar/rm"
	_ "github.com/brimsec/zq/ppl/cmd/zar/rmdirs"
	"github.com/brimsec/zq/ppl/cmd/zar/root"
	_ "github.com/brimsec/zq/ppl/cmd/zar/stat"
	_ "github.com/brimsec/zq/ppl/cmd/zar/zq"
)

func main() {
	if _, err := root.Zar.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

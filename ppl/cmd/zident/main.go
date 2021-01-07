package main

import (
	"fmt"
	"os"

	"github.com/brimsec/zq/ppl/cmd/zident/root"
	_ "github.com/brimsec/zq/ppl/cmd/zident/user"
)

func main() {
	if _, err := root.Zident.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

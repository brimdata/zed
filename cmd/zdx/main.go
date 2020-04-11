package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	_ "github.com/brimsec/zq/cmd/zdx/create"
	_ "github.com/brimsec/zq/cmd/zdx/lookup"
	_ "github.com/brimsec/zq/cmd/zdx/merge"
	"github.com/brimsec/zq/cmd/zdx/root"
)

func main() {
	//XXX Seed
	rand.Seed(time.Now().UTC().UnixNano())
	_, err := root.Zdx.ExecRoot(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

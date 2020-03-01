package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	_ "github.com/brimsec/zq/cmd/sst/create"
	_ "github.com/brimsec/zq/cmd/sst/dump"
	_ "github.com/brimsec/zq/cmd/sst/lookup"
	_ "github.com/brimsec/zq/cmd/sst/merge"
	"github.com/brimsec/zq/cmd/sst/root"
)

func main() {
	//XXX Seed
	rand.Seed(time.Now().UTC().UnixNano())
	_, err := root.Sst.ExecRoot(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

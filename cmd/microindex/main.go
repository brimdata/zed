package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	_ "github.com/brimsec/zq/cmd/microindex/convert"
	_ "github.com/brimsec/zq/cmd/microindex/create"
	_ "github.com/brimsec/zq/cmd/microindex/lookup"
	"github.com/brimsec/zq/cmd/microindex/root"
	_ "github.com/brimsec/zq/cmd/microindex/section"
	_ "github.com/brimsec/zq/cmd/microindex/seek"
)

func main() {
	//XXX Seed
	rand.Seed(time.Now().UTC().UnixNano())
	_, err := root.MicroIndex.ExecRoot(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

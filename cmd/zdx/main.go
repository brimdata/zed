package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	_ "github.com/brimsec/zq/cmd/zdx/convert"
	_ "github.com/brimsec/zq/cmd/zdx/create"
	_ "github.com/brimsec/zq/cmd/zdx/lookup"
	"github.com/brimsec/zq/cmd/zdx/root"
	_ "github.com/brimsec/zq/cmd/zdx/section"
	_ "github.com/brimsec/zq/cmd/zdx/seek"
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

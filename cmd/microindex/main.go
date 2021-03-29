package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	_ "github.com/brimdata/zq/cmd/microindex/convert"
	_ "github.com/brimdata/zq/cmd/microindex/create"
	_ "github.com/brimdata/zq/cmd/microindex/lookup"
	"github.com/brimdata/zq/cmd/microindex/root"
	_ "github.com/brimdata/zq/cmd/microindex/section"
	_ "github.com/brimdata/zq/cmd/microindex/seek"
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

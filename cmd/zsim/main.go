package main

import (
	"fmt"
	"os"
)

func main() {
	if _, err := Zsim.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

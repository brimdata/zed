package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	var a net.IP
	if _, err := Zsim.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

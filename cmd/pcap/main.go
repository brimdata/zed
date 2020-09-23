package main

import (
	"fmt"
	"os"

	_ "github.com/brimsec/zq/cmd/pcap/cut"
	_ "github.com/brimsec/zq/cmd/pcap/index"
	_ "github.com/brimsec/zq/cmd/pcap/info"
	"github.com/brimsec/zq/cmd/pcap/root"
	_ "github.com/brimsec/zq/cmd/pcap/slice"
	_ "github.com/brimsec/zq/cmd/pcap/ts"
)

// Version is set via the Go linker.
var version = "unknown"

func main() {
	//XXX
	//root.Version = version
	if _, err := root.Pcap.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

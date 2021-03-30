package main

import (
	"fmt"
	"os"

	_ "github.com/brimdata/zed/cmd/pcap/cut"
	_ "github.com/brimdata/zed/cmd/pcap/index"
	_ "github.com/brimdata/zed/cmd/pcap/info"
	"github.com/brimdata/zed/cmd/pcap/root"
	_ "github.com/brimdata/zed/cmd/pcap/slice"
	_ "github.com/brimdata/zed/cmd/pcap/ts"
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

package lakeflags

import (
	"flag"
)

type Flags struct {
	Quiet    bool
	PoolName string
}

func (f *Flags) SetFlags(fs *flag.FlagSet) {
	fs.BoolVar(&f.Quiet, "q", false, "quiet mode")
	fs.StringVar(&f.PoolName, "p", "", "name of pool")
}

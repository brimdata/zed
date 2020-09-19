package procflags

import (
	"errors"
	"flag"

	"github.com/brimsec/zq/pkg/units"
	"github.com/brimsec/zq/proc/sort"
)

type Flags struct {
	sortMemMax units.Bytes
	// fuse, groupby etc memory limits coming soon
}

func (f *Flags) SetFlags(fs *flag.FlagSet) {
	f.sortMemMax = units.Bytes(sort.MemMaxBytes)
	fs.Var(&f.sortMemMax, "sortmem", "maximum memory used by sort in MiB, MB, etc")
}

func (f *Flags) Init() error {
	if f.sortMemMax <= 0 {
		return errors.New("sortmem value must be greater than zero")
	}
	sort.MemMaxBytes = int(f.sortMemMax)
	return nil
}

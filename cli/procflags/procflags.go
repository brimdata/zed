package procflags

import (
	"errors"
	"flag"

	"github.com/brimsec/zq/pkg/units"
	"github.com/brimsec/zq/proc/fuse"
	"github.com/brimsec/zq/proc/sort"
)

type Flags struct {
	// these memory limits should be based on a shared resource model
	sortMemMax units.Bytes
	fuseMemMax units.Bytes
}

func (f *Flags) SetFlags(fs *flag.FlagSet) {
	f.sortMemMax = units.Bytes(sort.MemMaxBytes)
	fs.Var(&f.sortMemMax, "sortmem", "maximum memory used by sort in MiB, MB, etc")
	f.fuseMemMax = units.Bytes(fuse.MemMaxBytes)
	fs.Var(&f.fuseMemMax, "fusemem", "maximum memory used by sort in MiB, MB, etc")
}

func (f *Flags) Init() error {
	if f.sortMemMax <= 0 {
		return errors.New("sortmem value must be greater than zero")
	}
	sort.MemMaxBytes = int(f.sortMemMax)
	if f.fuseMemMax <= 0 {
		return errors.New("fusemem value must be greater than zero")
	}
	fuse.MemMaxBytes = int(f.fuseMemMax)
	return nil
}

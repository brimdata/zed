package procflags

import (
	"errors"
	"flag"

	"github.com/brimdata/zed/cli/auto"
	"github.com/brimdata/zed/proc/fuse"
	"github.com/brimdata/zed/proc/sort"
	"github.com/pbnjay/memory"
)

// defaultMemMaxBytes returns approximately 1/8 of total system memory,
// in bytes, bounded between 128MiB and 1GiB.
func defaultMemMaxBytes() uint64 {
	tm := memory.TotalMemory()
	const gig = 1024 * 1024 * 1024
	switch {
	case tm <= 1*gig:
		return 128 * 1024 * 1024
	case tm <= 2*gig:
		return 256 * 1024 * 1024
	case tm <= 4*gig:
		return 512 * 1024 * 1024
	default:
		return 1 * gig
	}
}

type Flags struct {
	// these memory limits should be based on a shared resource model
	sortMemMax auto.Bytes
	fuseMemMax auto.Bytes
}

func (f *Flags) SetFlags(fs *flag.FlagSet) {
	def := defaultMemMaxBytes()
	f.sortMemMax = auto.NewBytes(def)
	fs.Var(&f.sortMemMax, "sortmem", "maximum memory used by sort in MiB, MB, etc")
	f.fuseMemMax = auto.NewBytes(def)
	fs.Var(&f.fuseMemMax, "fusemem", "maximum memory used by fuse in MiB, MB, etc")
}

func (f *Flags) Init() error {
	if f.sortMemMax.Bytes <= 0 {
		return errors.New("sortmem value must be greater than zero")
	}
	sort.MemMaxBytes = int(f.sortMemMax.Bytes)
	if f.fuseMemMax.Bytes <= 0 {
		return errors.New("fusemem value must be greater than zero")
	}
	fuse.MemMaxBytes = int(f.fuseMemMax.Bytes)
	return nil
}

package procflags

import (
	"errors"
	"flag"
	"fmt"

	"github.com/alecthomas/units"
	"github.com/brimsec/zq/proc/fuse"
	"github.com/brimsec/zq/proc/sort"
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

type AutoBytes struct {
	defStr string
	bytes  units.Base2Bytes
}

func (b AutoBytes) String() string {
	if b.defStr != "" {
		return b.defStr
	}
	return b.bytes.String()
}

func (b *AutoBytes) Set(s string) error {
	b.defStr = ""
	b.bytes = 0
	bytes, err := units.ParseStrictBytes(s)
	if err != nil {
		return err
	}
	b.bytes = units.Base2Bytes(bytes)
	return nil
}

func newAutoBytes(def uint64) AutoBytes {
	bytes := units.Base2Bytes(def)
	return AutoBytes{
		defStr: fmt.Sprintf("auto(%s)", bytes),
		bytes:  bytes,
	}
}

type Flags struct {
	// these memory limits should be based on a shared resource model
	sortMemMax AutoBytes
	fuseMemMax AutoBytes
}

func (f *Flags) SetFlags(fs *flag.FlagSet) {
	def := defaultMemMaxBytes()
	f.sortMemMax = newAutoBytes(def)
	fs.Var(&f.sortMemMax, "sortmem", "maximum memory used by sort in MiB, MB, etc")
	f.fuseMemMax = newAutoBytes(def)
	fs.Var(&f.fuseMemMax, "fusemem", "maximum memory used by fuse in MiB, MB, etc")
}

func (f *Flags) Init() error {
	if f.sortMemMax.bytes <= 0 {
		return errors.New("sortmem value must be greater than zero")
	}
	sort.MemMaxBytes = int(f.sortMemMax.bytes)
	if f.fuseMemMax.bytes <= 0 {
		return errors.New("fusemem value must be greater than zero")
	}
	fuse.MemMaxBytes = int(f.fuseMemMax.bytes)
	return nil
}

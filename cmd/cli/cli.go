package cli

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/brimsec/zq/emitter"
	"github.com/brimsec/zq/proc/sort"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zio/options"
	"github.com/brimsec/zq/zng/resolver"
	"golang.org/x/crypto/ssh/terminal"
)

type Flags struct {
	sortMemMaxMiB float64
	cpuprofile    string
	memprofile    string
}

func (f *Flags) SetFlags(fs *flag.FlagSet) {
	fs.Float64Var(&f.sortMemMaxMiB, "sortmem", float64(sort.MemMaxBytes)/(1024*1024), "maximum memory used by sort, in MiB")
	fs.StringVar(&f.cpuprofile, "cpuprofile", "", "write cpu profile to `file`")
	fs.StringVar(&f.memprofile, "memprofile", "", "write memory profile to `file`")
}

func (f *Flags) Init() error {
	if f.cpuprofile != "" {
		runCPUProfile(f.cpuprofile)
	}
	if f.sortMemMaxMiB <= 0 {
		return errors.New("sortmem value must be greater than zero")
	}
	sort.MemMaxBytes = int(f.sortMemMaxMiB * 1024 * 1024)
	return nil
}

func (f *Flags) Cleanup() {
	if f.cpuprofile != "" {
		pprof.StopCPUProfile()
	}
	if f.memprofile != "" {
		runMemProfile(f.memprofile)
	}
}

type OutputFlags struct {
	dir          string
	outputFile   string
	forceBinary  bool
	textShortcut bool
}

func (f *OutputFlags) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&f.dir, "d", "", "directory for output data files")
	fs.StringVar(&f.outputFile, "o", "", "write data to output file")
	fs.BoolVar(&f.textShortcut, "t", false, "use format tzng independent of -f option")
	fs.BoolVar(&f.forceBinary, "B", false, "allow binary zng be sent to a terminal output")
}

func (f *OutputFlags) Init(opts *options.Writer) error {
	if f.textShortcut {
		if opts.Format != "zng" {
			return errors.New("cannot use -t with -f")
		}
		opts.Format = "tzng"
	}
	if f.outputFile == "" && opts.Format == "zng" && IsTerminal(os.Stdout) && !f.forceBinary {
		return errors.New("writing binary zng data to terminal; override with -B or use -t for text.")
	}
	return nil
}

func (o *OutputFlags) Open(opts options.Writer) (zbuf.WriteCloser, error) {
	if o.dir != "" {
		d, err := emitter.NewDir(o.dir, o.outputFile, os.Stderr, opts)
		if err != nil {
			return nil, err
		}
		return d, nil
	}
	w, err := emitter.NewFile(o.outputFile, opts)
	if err != nil {
		return nil, err
	}
	return w, nil
}

func OpenInputs(zctx *resolver.Context, opts options.Reader, paths []string, stopOnErr bool) ([]zbuf.Reader, error) {
	var readers []zbuf.Reader
	for _, path := range paths {
		if path == "-" {
			path = detector.StdinPath
		}
		file, err := detector.OpenFile(zctx, path, opts)
		if err != nil {
			err = fmt.Errorf("%s: %w", path, err)
			if stopOnErr {
				return nil, err
			}
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		readers = append(readers, file)
	}
	return readers, nil
}

func FileExists(path string) bool {
	if path == "-" {
		return true
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func IsTerminal(f *os.File) bool {
	return terminal.IsTerminal(int(f.Fd()))
}

func runCPUProfile(path string) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
}

func runMemProfile(path string) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	runtime.GC()
	pprof.Lookup("allocs").WriteTo(f, 0)
	f.Close()
}

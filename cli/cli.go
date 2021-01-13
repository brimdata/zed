package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
)

// Version is set via the Go linker.  See Makefile.
var Version = "unknown"

type Flags struct {
	showVersion    bool
	cpuprofile     string
	memprofile     string
	cpuProfileFile *os.File
}

func (f *Flags) SetFlags(fs *flag.FlagSet) {
	fs.BoolVar(&f.showVersion, "version", false, "print version and exit")
	fs.StringVar(&f.cpuprofile, "cpuprofile", "", "write cpu profile to given file name")
	fs.StringVar(&f.memprofile, "memprofile", "", "write memory profile to given file name")
}

type Initializer interface {
	Init() error
}

func (f *Flags) Init(all ...Initializer) error {
	if f.showVersion {
		fmt.Printf("Version: %s\n", Version)
		os.Exit(0)
	}
	var err error
	for _, flags := range all {
		if initErr := flags.Init(); err == nil {
			err = initErr
		}
	}
	if err != nil {
		return err
	}
	if f.cpuprofile != "" {
		f.runCPUProfile(f.cpuprofile)
	}
	return nil
}

func (f *Flags) Cleanup() {
	if f.cpuProfileFile != nil {
		pprof.StopCPUProfile()
		f.cpuProfileFile.Close()
	}
	if f.memprofile != "" {
		runMemProfile(f.memprofile)
	}
}

func (f *Flags) runCPUProfile(path string) {
	file, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	f.cpuProfileFile = file
	pprof.StartCPUProfile(file)
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

// FuncValue is an adaptor allowing a function to act as a flag.Value.
// When we support go 1.16, usages of this can be replaced with flag.Func.
type FuncValue func(string) error

func (f FuncValue) Set(s string) error { return f(s) }
func (f FuncValue) String() string     { return "" }

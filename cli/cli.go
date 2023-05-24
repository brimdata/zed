package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
)

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

// Init is equivalent to InitWithSignals with SIGINT, SIGPIPE, and SIGTERM.
func (f *Flags) Init(all ...Initializer) (context.Context, context.CancelFunc, error) {
	return f.InitWithSignals(all, syscall.SIGINT, syscall.SIGPIPE, syscall.SIGTERM)
}

// InitWithSignals handles the flags defined in SetFlags, calls the Init method
// for each element of all, and returns a context canceled when any signal in
// signals is raised.
func (f *Flags) InitWithSignals(all []Initializer, signals ...os.Signal) (context.Context, context.CancelFunc, error) {
	if f.showVersion {
		fmt.Printf("Version: %s\n", Version())
		os.Exit(0)
	}
	var err error
	for _, flags := range all {
		if initErr := flags.Init(); err == nil {
			err = initErr
		}
	}
	if err != nil {
		return nil, nil, err
	}
	if f.cpuprofile != "" {
		f.runCPUProfile(f.cpuprofile)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), signals...)
	cleanup := func() {
		cancel()
		f.cleanup()
	}
	return &interruptedContext{ctx}, cleanup, nil
}

type interruptedContext struct{ context.Context }

func (i *interruptedContext) Err() error {
	err := i.Context.Err()
	if errors.Is(err, context.Canceled) {
		return errors.New("interrupted")
	}
	return err
}

func (f *Flags) cleanup() {
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

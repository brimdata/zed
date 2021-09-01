package outputflags

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/exec"

	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/pkg/terminal"
	"github.com/brimdata/zed/pkg/terminal/color"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/anyio"
	"github.com/brimdata/zed/zio/emitter"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zio/zstio"
)

type Flags struct {
	anyio.WriterOpts
	DefaultFormat string
	dir           string
	outputFile    string
	forceBinary   bool
	zsonShortcut  bool
	zsonPretty    bool
	color         bool
}

func (f *Flags) Options() anyio.WriterOpts {
	return f.WriterOpts
}

func (f *Flags) setFlags(fs *flag.FlagSet) {
	// zio stuff
	fs.BoolVar(&f.UTF8, "U", false, "display zeek strings as UTF-8")
	fs.BoolVar(&f.Text.ShowTypes, "T", false, "display field types in text output")
	fs.BoolVar(&f.Text.ShowFields, "F", false, "display field names in text output")
	fs.BoolVar(&f.color, "color", true, "enable/disable color formatting for -Z and lake text output")
	fs.IntVar(&f.Zng.LZ4BlockSize, "znglz4blocksize", zngio.DefaultLZ4BlockSize,
		"LZ4 block size in bytes for ZNG compression (nonpositive to disable)")
	fs.IntVar(&f.ZSON.Pretty, "pretty", 4,
		"tab size to pretty print zson output (0 for newline-delimited zson")
	f.Zst.ColumnThresh = zstio.DefaultColumnThresh
	fs.Var(&f.Zst.ColumnThresh, "coltresh", "minimum frame size (MiB) used for zst columns")
	f.Zst.SkewThresh = zstio.DefaultSkewThresh
	fs.Var(&f.Zst.SkewThresh, "skewtresh", "minimum skew size (MiB) used to group zst columns")

	// emitter stuff
	fs.StringVar(&f.dir, "d", "", "directory for output data files")
	fs.StringVar(&f.outputFile, "o", "", "write data to output file")

}

func (f *Flags) SetFlags(fs *flag.FlagSet) {
	f.SetFormatFlags(fs)
	f.setFlags(fs)
}

func (f *Flags) SetFlagsWithFormat(fs *flag.FlagSet, format string) {
	f.setFlags(fs)
	f.Format = format
}

func (f *Flags) SetFormatFlags(fs *flag.FlagSet) {
	if f.DefaultFormat == "" {
		f.DefaultFormat = "zng"
	}
	fs.StringVar(&f.Format, "f", f.DefaultFormat, "format for output data [zng,zst,json,ndjson,parquet,table,text,csv,lake,zeek,zjson,zson,tzng]")
	fs.BoolVar(&f.zsonShortcut, "z", false, "use line-oriented zson output independent of -f option")
	fs.BoolVar(&f.zsonPretty, "Z", false, "use formatted zson output independent of -f option")
	fs.BoolVar(&f.forceBinary, "B", false, "allow binary zng be sent to a terminal output")
}

func (f *Flags) Init() error {
	if f.zsonShortcut || f.zsonPretty {
		if f.Format != f.DefaultFormat {
			return errors.New("cannot use -z or -Z with -f")
		}
		f.Format = "zson"
		if !f.zsonPretty {
			f.ZSON.Pretty = 0
		}
	}
	if f.outputFile == "-" {
		f.outputFile = ""
	}
	if f.outputFile == "" && f.Format == "zng" && terminal.IsTerminalFile(os.Stdout) && !f.forceBinary {
		f.Format = "zson"
		f.ZSON.Pretty = 0
	}
	return nil
}

func (f *Flags) FileName() string {
	return f.outputFile
}

var child *exec.Cmd

func WaitForChild() {
	if child != nil {
		child.Wait()
	}
}

func (f *Flags) Open(ctx context.Context, engine storage.Engine) (zio.WriteCloser, error) {
	if f.dir != "" {
		d, err := emitter.NewDir(ctx, engine, f.dir, f.outputFile, os.Stderr, f.WriterOpts)
		if err != nil {
			return nil, err
		}
		return d, nil
	}
	if f.outputFile == "" && f.color && terminal.IsTerminalFile(os.Stdout) {
		color.Enabled = true
		if pager := os.Getenv("ZED_PAGER"); pager != "" && (f.zsonPretty || f.Format == "lake") {
			cmd := exec.Command("/usr/bin/less", "-R")
			if w, err := cmd.StdinPipe(); err == nil {
				cmd.Stdout = os.Stdout
				if err := cmd.Start(); err == nil {
					child = cmd
					return anyio.NewWriter(w, f.WriterOpts)
				}
			}
		}
	}
	w, err := emitter.NewFileFromPath(ctx, engine, f.outputFile, f.WriterOpts)
	if err != nil {
		return nil, err
	}
	return w, nil
}

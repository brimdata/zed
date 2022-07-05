package outputflags

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"

	"github.com/brimdata/zed/cli/auto"
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
	split         string
	splitSize     auto.Bytes
	outputFile    string
	forceBinary   bool
	jsonShortcut  bool
	zsonShortcut  bool
	zsonPretty    bool
	zsonPersist   string
	color         bool
}

func (f *Flags) Options() anyio.WriterOpts {
	return f.WriterOpts
}

func (f *Flags) setFlags(fs *flag.FlagSet) {
	// zio stuff
	fs.BoolVar(&f.color, "color", true, "enable/disable color formatting for -Z and lake text output")
	fs.BoolVar(&f.ZNG.Compress, "zngcompress", true, "compress ZNG frames")
	fs.IntVar(&f.ZNG.FrameThresh, "zngframethresh", zngio.DefaultFrameThresh,
		"minimum ZNG frame size in uncompressed bytes")
	fs.IntVar(&f.ZSON.Pretty, "pretty", 4,
		"tab size to pretty print ZSON output (0 for newline-delimited ZSON")
	fs.StringVar(&f.zsonPersist, "persist", "",
		"regular expression to persist type definitions across the stream")
	f.Zst.ColumnThresh = zstio.DefaultColumnThresh
	fs.Var(&f.Zst.ColumnThresh, "coltresh", "minimum frame size (MiB) used for zst columns")
	f.Zst.SkewThresh = zstio.DefaultSkewThresh
	fs.Var(&f.Zst.SkewThresh, "skewtresh", "minimum skew size (MiB) used to group zst columns")

	// emitter stuff
	fs.StringVar(&f.split, "split", "",
		"split output into one file per data type in this directory (but see -splitsize)")
	fs.Var(&f.splitSize, "splitsize",
		"if >0 and -split is set, split into files at least this big rather than by data type")
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
	fs.StringVar(&f.Format, "f", f.DefaultFormat, "format for output data [zng,zst,json,parquet,table,text,csv,lake,zeek,zjson,zson]")
	fs.BoolVar(&f.jsonShortcut, "j", false, "use line-oriented JSON output independent of -f option")
	fs.BoolVar(&f.zsonShortcut, "z", false, "use line-oriented ZSON output independent of -f option")
	fs.BoolVar(&f.zsonPretty, "Z", false, "use formatted ZSON output independent of -f option")
	fs.BoolVar(&f.forceBinary, "B", false, "allow binary zng be sent to a terminal output")
}

func (f *Flags) Init() error {
	if f.zsonPersist != "" {
		re, err := regexp.Compile(f.zsonPersist)
		if err != nil {
			return err
		}
		f.ZSON.Persist = re
	}
	if f.jsonShortcut {
		if f.Format != f.DefaultFormat || f.zsonShortcut || f.zsonPretty {
			return errors.New("cannot use -j with -f, -z, or -Z")
		}
		f.Format = "json"
	} else if f.zsonShortcut || f.zsonPretty {
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
	if f.outputFile == "" && f.split == "" && f.Format == "zng" && !f.forceBinary &&
		terminal.IsTerminalFile(os.Stdout) {
		f.Format = "zson"
		f.ZSON.Pretty = 0
	}
	return nil
}

func (f *Flags) FileName() string {
	return f.outputFile
}

func (f *Flags) Open(ctx context.Context, engine storage.Engine) (zio.WriteCloser, error) {
	if f.split != "" {
		dir, err := storage.ParseURI(f.split)
		if err != nil {
			return nil, fmt.Errorf("-split option: %w", err)
		}
		if size := f.splitSize.Bytes; size > 0 {
			return emitter.NewSizeSplitter(ctx, engine, dir, f.outputFile, f.WriterOpts, int64(size))
		}
		d, err := emitter.NewSplit(ctx, engine, dir, f.outputFile, f.WriterOpts)
		if err != nil {
			return nil, err
		}
		return d, nil
	}
	if f.outputFile == "" && f.color && terminal.IsTerminalFile(os.Stdout) {
		color.Enabled = true
	}
	w, err := emitter.NewFileFromPath(ctx, engine, f.outputFile, f.WriterOpts)
	if err != nil {
		return nil, err
	}
	return w, nil
}

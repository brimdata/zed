package inputflags

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"sync/atomic"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/cli/auto"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/anyio"
	"github.com/brimdata/zed/zio/zngio"
)

type Flags struct {
	anyio.ReaderOpts
	ReadMax  auto.Bytes
	ReadSize auto.Bytes
}

func (f *Flags) Options() anyio.ReaderOpts {
	return f.ReaderOpts
}

func (f *Flags) SetFlags(fs *flag.FlagSet, validate bool) {
	fs.StringVar(&f.Format, "i", "auto", "format of input data [auto,zng,zst,json,ndjson,zeek,zjson,csv,parquet]")
	fs.BoolVar(&f.ZNG.Validate, "validate", validate, "validate the input format when reading ZNG streams")
	f.ReadMax = auto.NewBytes(zngio.MaxSize)
	fs.Var(&f.ReadMax, "readmax", "maximum memory used read buffers in MiB, MB, etc")
	f.ReadSize = auto.NewBytes(zngio.ReadSize)
	fs.Var(&f.ReadSize, "readsize", "target memory used read buffers in MiB, MB, etc")
}

// Init is called after flags have been parsed.
func (f *Flags) Init() error {
	f.ZNG.Max = int(f.ReadMax.Bytes)
	if f.ZNG.Max < 0 {
		return errors.New("max read buffer size must be greater than zero")
	}
	f.ZNG.Size = int(f.ReadSize.Bytes)
	if f.ZNG.Size < 0 {
		return errors.New("target read buffer size must be greater than zero")
	}
	return nil
}

func (f *Flags) Open(ctx context.Context, zctx *zed.Context, engine storage.Engine, paths []string, stopOnErr bool) ([]zio.Reader, error) {
	var readers []zio.Reader
	for _, path := range paths {
		if path == "-" {
			path = "stdio:stdin"
		}
		file, err := anyio.Open(ctx, zctx, engine, path, f.ReaderOpts)
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

func (f *Flags) OpenWithStats(ctx context.Context, zctx *zed.Context, engine storage.Engine, paths []string, stopOnErr bool) ([]*StatsReader, error) {
	e := &engineWrap{Engine: engine}
	readers, err := f.Open(ctx, zctx, e, paths, stopOnErr)
	if err != nil {
		return nil, err
	}
	statsers := make([]*StatsReader, len(readers))
	for i, r := range readers {
		sr := e.readers[i]
		size, err := storage.Size(sr.Reader)
		if err != nil && !errors.Is(err, storage.ErrNotSupported) {
			zio.CloseReaders(readers)
			return nil, err
		}
		statsers[i] = &StatsReader{
			Reader:     r,
			BytesTotal: size,
			counter:    sr,
		}
	}
	return statsers, nil
}

type engineWrap struct {
	storage.Engine
	readers []*byteCounter
}

func (e *engineWrap) Get(ctx context.Context, u *storage.URI) (storage.Reader, error) {
	r, err := e.Engine.Get(ctx, u)
	if err != nil {
		return nil, err
	}
	counter := &byteCounter{Reader: r}
	e.readers = append(e.readers, counter)
	return counter, nil
}

type StatsReader struct {
	zio.Reader
	BytesTotal int64
	counter    *byteCounter
}

func (r *StatsReader) BytesRead() int64 {
	return r.counter.BytesRead()
}

type byteCounter struct {
	storage.Reader
	n int64
}

func (r *byteCounter) Read(b []byte) (int, error) {
	n, err := r.Reader.Read(b)
	atomic.AddInt64(&r.n, int64(n))
	return n, err
}

func (r *byteCounter) BytesRead() int64 {
	return atomic.LoadInt64(&r.n)
}

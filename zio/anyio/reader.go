package anyio

import (
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/tzngio"
	"github.com/brimdata/zed/zio/zeekio"
	"github.com/brimdata/zed/zio/zjsonio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zqe"
	"github.com/brimdata/zed/zson"
)

type ReaderOpts struct {
	Format string
	Zng    zngio.ReaderOpts
	AwsCfg *aws.Config
}

func NewReaderWithOpts(r io.Reader, zctx *zson.Context, opts ReaderOpts) (zbuf.Reader, error) {
	recorder := NewRecorder(r)
	track := NewTrack(recorder)

	tzngErr := match(tzngio.NewReader(track, zson.NewContext()), "tzng")
	if tzngErr == nil {
		return tzngio.NewReader(recorder, zctx), nil
	}
	track.Reset()

	zr, err := zeekio.NewReader(track, zson.NewContext())
	if err != nil {
		return nil, err
	}
	zeekErr := match(zr, "zeek")
	if zeekErr == nil {
		return zeekio.NewReader(recorder, zctx)
	}
	track.Reset()

	// ZJSON must come before ZSON since ZJSON is a subset of ZSON
	zjsonErr := match(zjsonio.NewReader(track, zson.NewContext()), "zjson")
	if zjsonErr == nil {
		return zjsonio.NewReader(recorder, zctx), nil
	}
	track.Reset()

	zsonErr := match(zson.NewReader(track, zson.NewContext()), "zson")
	if zsonErr == nil {
		return zson.NewReader(recorder, zctx), nil
	}
	track.Reset()

	zngOpts := opts.Zng
	zngOpts.Validate = true
	zngErr := match(zngio.NewReaderWithOpts(track, zson.NewContext(), zngOpts), "zng")
	if zngErr == nil {
		return zngio.NewReaderWithOpts(recorder, zctx, opts.Zng), nil
	}
	track.Reset()

	// XXX This is a placeholder until we add a flag to the csv reader
	// for "strict" mode.  See issue #2316.
	//csvErr := match(csvio.NewReader(track, zson.NewContext()), "csv")
	//if csvErr == nil {
	//	return csvio.NewReader(recorder, Context), nil
	//}
	//track.Reset()

	parquetErr := errors.New("parquet: auto-detection not supported")
	zstErr := errors.New("zst: auto-detection not supported")
	return nil, joinErrs([]error{tzngErr, zeekErr, zjsonErr, zsonErr, zngErr, parquetErr, zstErr})
}

func NewReader(r io.Reader, zctx *zson.Context) (zbuf.Reader, error) {
	return NewReaderWithOpts(r, zctx, ReaderOpts{})
}

func joinErrs(errs []error) error {
	s := "format detection error"
	for _, e := range errs {
		s += "\n\t" + e.Error()
	}
	return zqe.E(s)
}
func match(r zbuf.Reader, name string) error {
	_, err := r.Read()
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

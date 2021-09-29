package anyio

import (
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio"
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

func NewReaderWithOpts(r io.Reader, zctx *zed.Context, opts ReaderOpts) (zio.Reader, error) {
	if opts.Format != "" && opts.Format != "auto" {
		return lookupReader(r, zctx, opts)
	}
	recorder := NewRecorder(r)
	track := NewTrack(recorder)

	tzngErr := match(tzngio.NewReader(track, zed.NewContext()), "tzng")
	if tzngErr == nil {
		return tzngio.NewReader(recorder, zctx), nil
	}
	track.Reset()

	zr, err := zeekio.NewReader(track, zed.NewContext())
	if err != nil {
		return nil, err
	}
	zeekErr := match(zr, "zeek")
	if zeekErr == nil {
		return zeekio.NewReader(recorder, zctx)
	}
	track.Reset()

	// ZJSON must come before ZSON since ZJSON is a subset of ZSON
	zjsonErr := match(zjsonio.NewReader(track, zed.NewContext()), "zjson")
	if zjsonErr == nil {
		return zjsonio.NewReader(recorder, zctx), nil
	}
	track.Reset()

	zsonErr := match(zson.NewReader(track, zed.NewContext()), "zson")
	if zsonErr == nil {
		return zson.NewReader(recorder, zctx), nil
	}
	track.Reset()

	// For the matching reader, force validation to true so we are extra
	// careful about auto-matching ZNG.  Then, once matched, relaxed
	// validation to the user setting in the actual reader returned.
	zngOpts := opts.Zng
	zngOpts.Validate = true
	zngErr := match(zngio.NewReaderWithOpts(track, zed.NewContext(), zngOpts), "zng")
	if zngErr == nil {
		return zngio.NewReaderWithOpts(recorder, zctx, opts.Zng), nil
	}
	track.Reset()

	// XXX This is a placeholder until we add a flag to the csv reader
	// for "strict" mode.  See issue #2316.
	//csvErr := match(csvio.NewReader(track, zed.NewContext()), "csv")
	//if csvErr == nil {
	//	return csvio.NewReader(recorder, Context), nil
	//}
	//track.Reset()

	csvErr := errors.New("csv: auto-detection not supported")
	jsonErr := errors.New("json: auto-detection not supported")
	parquetErr := errors.New("parquet: auto-detection not supported")
	zstErr := errors.New("zst: auto-detection not supported")
	return nil, joinErrs([]error{tzngErr, zeekErr, zjsonErr, zsonErr, zngErr, csvErr, jsonErr, parquetErr, zstErr})
}

func NewReader(r io.Reader, zctx *zed.Context) (zio.Reader, error) {
	return NewReaderWithOpts(r, zctx, ReaderOpts{})
}

func joinErrs(errs []error) error {
	s := "format detection error"
	for _, e := range errs {
		s += "\n\t" + e.Error()
	}
	return zqe.E(s)
}
func match(r zio.Reader, name string) error {
	_, err := r.Read()
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

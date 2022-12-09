package anyio

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/csvio"
	"github.com/brimdata/zed/zio/jsonio"
	"github.com/brimdata/zed/zio/zeekio"
	"github.com/brimdata/zed/zio/zjsonio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zio/zsonio"
)

type ReaderOpts struct {
	Format string
	ZNG    zngio.ReaderOpts
	AwsCfg *aws.Config
}

func NewReader(zctx *zed.Context, r io.Reader) (zio.ReadCloser, error) {
	return NewReaderWithOpts(zctx, r, ReaderOpts{})
}

func NewReaderWithOpts(zctx *zed.Context, r io.Reader, opts ReaderOpts) (zio.ReadCloser, error) {
	if opts.Format != "" && opts.Format != "auto" {
		return lookupReader(zctx, r, opts)
	}
	recorder := NewRecorder(r)
	track := NewTrack(recorder)

	zeekErr := match(zeekio.NewReader(zed.NewContext(), track), "zeek", 1)
	if zeekErr == nil {
		return zio.NopReadCloser(zeekio.NewReader(zctx, recorder)), nil
	}
	track.Reset()

	// ZJSON must come before JSON and ZSON since it is a subset of both.
	zjsonErr := match(zjsonio.NewReader(zed.NewContext(), track), "zjson", 1)
	if zjsonErr == nil {
		return zio.NopReadCloser(zjsonio.NewReader(zctx, recorder)), nil
	}
	track.Reset()

	// JSON comes before ZSON because the JSON reader is faster than the
	// ZSON reader.  The number of values wanted is greater than one for the
	// sake of tests.
	jsonErr := match(jsonio.NewReader(zed.NewContext(), track), "json", 10)
	if jsonErr == nil {
		return zio.NopReadCloser(jsonio.NewReader(zctx, recorder)), nil
	}
	track.Reset()

	zsonErr := match(zsonio.NewReader(zed.NewContext(), track), "zson", 1)
	if zsonErr == nil {
		return zio.NopReadCloser(zsonio.NewReader(zctx, recorder)), nil
	}
	track.Reset()

	// For the matching reader, force validation to true so we are extra
	// careful about auto-matching ZNG.  Then, once matched, relaxed
	// validation to the user setting in the actual reader returned.
	zngOpts := opts.ZNG
	zngOpts.Validate = true
	zngReader := zngio.NewReaderWithOpts(zed.NewContext(), track, zngOpts)
	zngErr := match(zngReader, "zng", 1)
	// Close zngReader to ensure that it does not continue to call track.Read.
	zngReader.Close()
	if zngErr == nil {
		return zngio.NewReaderWithOpts(zctx, recorder, opts.ZNG), nil
	}
	track.Reset()

	var csvErr error
	if s, err := bufio.NewReader(track).ReadString('\n'); err != nil {
		csvErr = fmt.Errorf("csv: line 1: %w", err)
	} else if !strings.Contains(s, ",") {
		csvErr = errors.New("csv: line 1: no comma found")
	} else {
		track.Reset()
		csvErr = match(csvio.NewReader(zed.NewContext(), track), "csv", 1)
		if csvErr == nil {
			return zio.NopReadCloser(csvio.NewReader(zctx, recorder)), nil
		}
	}
	track.Reset()

	parquetErr := errors.New("parquet: auto-detection not supported")
	zstErr := errors.New("zst: auto-detection not supported")
	lineErr := errors.New("line: auto-detection not supported")
	return nil, joinErrs([]error{zeekErr, zjsonErr, zsonErr, zngErr, csvErr, jsonErr, parquetErr, zstErr, lineErr})
}

func joinErrs(errs []error) error {
	s := "format detection error"
	for _, e := range errs {
		s += "\n\t" + e.Error()
	}
	return errors.New(s)
}

func match(r zio.Reader, name string, want int) error {
	for i := 0; i < want; i++ {
		val, err := r.Read()
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if val == nil {
			return nil
		}
	}
	return nil
}

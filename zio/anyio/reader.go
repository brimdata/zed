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

func NewReaderWithOpts(r io.Reader, zctx *zed.Context, opts ReaderOpts) (zio.Reader, error) {
	if opts.Format != "" && opts.Format != "auto" {
		return lookupReader(r, zctx, opts)
	}
	recorder := NewRecorder(r)
	track := NewTrack(recorder)

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

	zsonErr := match(zsonio.NewReader(track, zed.NewContext()), "zson")
	if zsonErr == nil {
		return zsonio.NewReader(recorder, zctx), nil
	}
	track.Reset()

	// JSON comes after ZSON because we want the ZSON reader to handle
	// top-level JSON objects and the JSON reader to handle top-level
	// JSON arrays.
	jsonErr := match(jsonio.NewReader(track, zed.NewContext()), "json")
	if jsonErr == nil {
		return jsonio.NewReader(recorder, zctx), nil
	}
	track.Reset()

	// For the matching reader, force validation to true so we are extra
	// careful about auto-matching ZNG.  Then, once matched, relaxed
	// validation to the user setting in the actual reader returned.
	zngOpts := opts.ZNG
	zngOpts.Validate = true
	zngErr := match(zngio.NewReaderWithOpts(track, zed.NewContext(), zngOpts), "zng")
	if zngErr == nil {
		return zngio.NewReaderWithOpts(recorder, zctx, opts.ZNG), nil
	}
	track.Reset()

	var csvErr error
	if s, err := bufio.NewReader(track).ReadString('\n'); err != nil {
		csvErr = fmt.Errorf("csv: line 1: %w", err)
	} else if !strings.Contains(s, ",") {
		csvErr = errors.New("csv: line 1: no comma found")
	} else {
		track.Reset()
		csvErr = match(csvio.NewReader(track, zed.NewContext()), "csv")
		if csvErr == nil {
			return csvio.NewReader(recorder, zctx), nil
		}
	}
	track.Reset()

	parquetErr := errors.New("parquet: auto-detection not supported")
	zstErr := errors.New("zst: auto-detection not supported")
	return nil, joinErrs([]error{zeekErr, zjsonErr, zsonErr, zngErr, csvErr, jsonErr, parquetErr, zstErr})
}

func NewReader(r io.Reader, zctx *zed.Context) (zio.Reader, error) {
	return NewReaderWithOpts(r, zctx, ReaderOpts{})
}

func joinErrs(errs []error) error {
	s := "format detection error"
	for _, e := range errs {
		s += "\n\t" + e.Error()
	}
	return errors.New(s)
}
func match(r zio.Reader, name string) error {
	_, err := r.Read()
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

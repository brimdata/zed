// Package search provides an implementation for launching zq searches and performing
// analytics on zng files stored in the server's root directory.
package search

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqe"
)

// This mtu is pretty small but it keeps the JSON object size below 64kb or so
// so the recevier can do reasonable, interactive streaming updates.
const DefaultMTU = 100

const StatsInterval = time.Millisecond * 500

const (
	MimeTypeNDJSON = "application/x-ndjson"
	MimeTypeZNG    = "application/x-zng"
)

type SearchStore interface {
	Open(ctx context.Context, zctx *resolver.Context, span nano.Span) (zbuf.ReadCloser, error)
}

type SearchOp struct {
	mux *driver.MuxOutput
	io.Closer
}

func NewSearchOp(ctx context.Context, s SearchStore, req api.SearchRequest) (*SearchOp, error) {
	if req.Span.Ts < 0 {
		return nil, errors.New("time span must have non-negative timestamp")
	}
	if req.Span.Dur < 0 {
		return nil, errors.New("time span must have non-negative duration")
	}
	// XXX zqd only supports backwards searches, remove once this has been
	// fixed.
	if req.Dir == 1 {
		return nil, zqe.E(zqe.Invalid, "forward searches not yet supported")
	}
	if req.Dir != -1 {
		return nil, zqe.E(zqe.Invalid, "time direction must be 1 or -1")
	}
	query, err := UnpackQuery(req)
	if err != nil {
		return nil, err
	}

	zctx := resolver.NewContext()
	zngReader, err := s.Open(ctx, zctx, query.Span)
	if err != nil {
		return nil, err
	}
	mapper := zbuf.NewMapper(zngReader, zctx)
	mux, err := launch(ctx, query, mapper, zctx)
	if err != nil {
		zngReader.Close()
		return nil, err
	}
	return &SearchOp{mux, zngReader}, nil
}

func (s *SearchOp) Run(output Output) error {
	d := &searchdriver{
		output:    output,
		startTime: nano.Now(),
	}
	d.start(0)
	statsTicker := time.NewTicker(StatsInterval)
	defer statsTicker.Stop()
	if err := driver.Run(s.mux, d, statsTicker.C); err != nil {
		d.abort(0, err)
		return err
	}
	if err := d.Stats(s.mux.Stats()); err != nil {
		d.abort(0, err)
		return err
	}
	return d.end(0)
}

// A Query is the internal representation of search query describing a source
// of tuples, a "search" applied to the tuples producing a set of matched
// tuples, and a proc to the process the tuples
type Query struct {
	Space api.SpaceID
	Dir   int
	Span  nano.Span
	Proc  ast.Proc
}

// UnpackQuery transforms a api.SearchRequest into a Query.
func UnpackQuery(req api.SearchRequest) (*Query, error) {
	proc, err := ast.UnpackProc(nil, req.Proc)
	if err != nil {
		return nil, err
	}
	return &Query{
		Space: req.Space,
		Dir:   req.Dir,
		Span:  req.Span,
		Proc:  proc,
	}, nil
}

// searchdriver implements driver.Driver.
type searchdriver struct {
	output    Output
	startTime nano.Ts
}

func (d *searchdriver) start(id int64) error {
	return d.output.SendControl(&api.TaskStart{"TaskStart", id})
}

func (d *searchdriver) end(id int64) error {
	return d.output.End(&api.TaskEnd{"TaskEnd", id, nil})
}

func (d *searchdriver) abort(id int64, err error) error {
	verr := &api.Error{Type: "INTERNAL", Message: err.Error()}
	return d.output.SendControl(&api.TaskEnd{"TaskEnd", id, verr})
}

func (d *searchdriver) Warn(warning string) error {
	v := api.SearchWarning{
		Type:    "SearchWarning",
		Warning: warning,
	}
	return d.output.SendControl(v)
}

func (d *searchdriver) Write(cid int, batch zbuf.Batch) error {
	return d.output.SendBatch(cid, batch)
}

func (d *searchdriver) Stats(stats api.ScannerStats) error {
	v := api.SearchStats{
		Type:         "SearchStats",
		StartTime:    d.startTime,
		UpdateTime:   nano.Now(),
		ScannerStats: stats,
	}
	return d.output.SendControl(v)
}

func (d *searchdriver) ChannelEnd(cid int) error {
	v := &api.SearchEnd{
		Type:      "SearchEnd",
		ChannelID: cid,
		Reason:    "eof",
	}
	return d.output.SendControl(v)
}

func launch(ctx context.Context, query *Query, reader zbuf.Reader, zctx *resolver.Context) (*driver.MuxOutput, error) {
	span := query.Span
	if span == (nano.Span{}) {
		span = nano.MaxSpan
	}
	// Records in a zqd filestore are sorted by descending ts (in zqd/storage/filestore.(*Storage).write).
	return driver.Compile(ctx, query.Proc, zctx, reader, driver.Config{
		ReaderSortKey:     "ts",
		ReaderSortReverse: true,
		Span:              span,
	})
}

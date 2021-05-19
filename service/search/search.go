// Package search provides an implementation for launching zq searches and performing
// analytics on zng files stored in the server's root directory.
package search

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/zed"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zqe"
	"github.com/brimdata/zed/zson"
	"go.uber.org/zap"
)

// This mtu is pretty small but it keeps the JSON object size below 64kb or so
// so the recevier can do reasonable, interactive streaming updates.
const DefaultMTU = 100

const StatsInterval = time.Millisecond * 500

const (
	MimeTypeCSV    = "text/csv"
	MimeTypeJSON   = "application/json"
	MimeTypeNDJSON = "application/x-ndjson"
	MimeTypeZJSON  = "application/x-zjson"
	MimeTypeZNG    = "application/x-zng"
)

type SearchOp struct {
	logger *zap.Logger
	query  *Query
}

func NewSearchOp(req api.SearchRequest, logger *zap.Logger) (*SearchOp, error) {
	if req.Span.Ts < 0 {
		return nil, zqe.ErrInvalid("time span must have non-negative timestamp")
	}
	if req.Span.Dur < 0 {
		return nil, zqe.ErrInvalid("time span must have non-negative duration")
	}
	// XXX zqd only supports backwards searches, remove once this has been
	// fixed.
	if req.Dir == 1 {
		return nil, zqe.ErrInvalid("forward searches not yet supported")
	}
	if req.Dir != -1 {
		return nil, zqe.ErrInvalid("time direction must be 1 or -1")
	}
	query, err := UnpackQuery(req)
	if err != nil {
		return nil, err
	}
	return &SearchOp{
		logger: logger,
		query:  query,
	}, nil
}

func (s *SearchOp) Run(ctx context.Context, adaptor proc.DataAdaptor, pool *lake.Pool, output Output, parallelism int) (err error) {
	d := &searchdriver{
		output:    output,
		startTime: nano.Now(),
	}
	d.start(0)
	defer func() {
		if err != nil {
			d.abort(0, err)
			return
		}
		d.end(0)
	}()
	// This big gunky thing is here to support the current Search API.
	// We will bring up a run API soon that supports either Zed text
	// as a string or a DAG.  The AST will not appear in the API after
	// that point.
	seq := &ast.Sequential{Kind: "Sequential"}
	if s.query.Proc != nil {
		var ok bool
		seq, ok = s.query.Proc.(*ast.Sequential)
		if !ok {
			return zqe.ErrInvalid(fmt.Sprintf("deprecated search API: Zed program does not begin with ast.Sequential: %T", s.query.Proc))
		}
	}
	var scanOrder string
	if s.query.Dir > 0 {
		scanOrder = "asc"
	} else if s.query.Dir < 0 {
		scanOrder = "desc"
	}
	var scanRange *ast.Range
	if s.query.Span.Dur != 0 {
		scanRange = &ast.Range{
			Kind: "Range",
			Lower: &zed.Primitive{
				Kind: "Primitive",
				Type: "time",
				Text: s.query.Span.Ts.Time().Format(time.RFC3339Nano),
			},
			Upper: &zed.Primitive{
				Kind: "Primitive",
				Type: "time",
				Text: s.query.Span.End().Time().Format(time.RFC3339Nano),
			},
		}
	}
	trunk := ast.Trunk{
		Kind: "Trunk",
		Source: &ast.Pool{
			Kind:      "Pool",
			Name:      pool.Name,
			Range:     scanRange,
			ScanOrder: scanOrder,
		},
	}
	seq.Prepend(&ast.From{
		Kind:   "From",
		Trunks: []ast.Trunk{trunk},
	})
	statsTicker := time.NewTicker(StatsInterval)
	defer statsTicker.Stop()
	zctx := zson.NewContext()
	return driver.RunWithLakeAndStats(ctx, d, seq, zctx, adaptor, statsTicker.C, s.logger, parallelism)
}

// A Query is the internal representation of search query describing a source
// of tuples, a "search" applied to the tuples producing a set of matched
// tuples, and a proc to the process the tuples
type Query struct {
	Dir       int
	JournalID uint64
	Span      nano.Span
	Proc      ast.Proc
}

// UnpackQuery transforms a api.SearchRequest into a Query.
func UnpackQuery(req api.SearchRequest) (*Query, error) {
	proc, err := ast.UnpackJSONAsProc(req.Proc)
	if err != nil {
		return nil, err
	}
	return &Query{
		JournalID: req.JournalID,
		Dir:       req.Dir,
		Span:      req.Span,
		Proc:      proc,
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
	if errors.Is(err, journal.ErrEmpty) {
		// XXX (nibs) - A search on an empty space should return an error. This
		// check should be in the driver though.
		return d.end(id)
	}
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

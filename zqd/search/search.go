// Package search provides an implementation for launching zq searches and performing
// analytics on bzng files stored in the server's root directory.
package search

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/brimsec/zq/ast"
	zdriver "github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/space"
	"github.com/brimsec/zq/zql"
	"go.uber.org/zap"
)

// This mtu is pretty small but it keeps the JSON object size below 64kb or so
// so the recevier can do reasonable, interactive streaming updates.
const DefaultMTU = 100

func Search(ctx context.Context, s *space.Space, req api.SearchRequest, out Output) error {
	// XXX These validation checks should result in 400 level status codes and
	// thus shouldn't occur here.
	if req.Span.Ts < 0 {
		return errors.New("time span must have non-negative timestamp")
	}
	if req.Span.Dur < 0 {
		return errors.New("time span must have non-negative duration")
	}
	// XXX allow either direction even through we do forward only right now
	if req.Dir != 1 && req.Dir != -1 {
		return errors.New("time direction must be 1 or -1")
	}
	query, err := UnpackQuery(req)
	if err != nil {
		return err
	}
	var f io.ReadCloser
	f, err = s.OpenFile(space.AllBzngFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		f = ioutil.NopCloser(strings.NewReader(""))
	}
	defer f.Close()
	zngReader, err := detector.LookupReader("bzng", f, resolver.NewContext())
	if err != nil {
		return err
	}
	zctx := resolver.NewContext()
	mapper := scanner.NewMapper(zngReader, zctx)
	mux, err := launch(ctx, query, mapper, zctx)
	if err != nil {
		return err
	}
	return run(mux, out)
}

func Copy(ctx context.Context, w []zbuf.Writer, r zbuf.Reader, prog string) error {
	p, err := zql.ParseProc(prog)
	if err != nil {
		return err
	}
	mux, err := zdriver.Compile(ctx, p, r, false, nano.MaxSpan, zap.NewNop())
	if err != nil {
		return err
	}
	d := zdriver.New(w...)
	return d.Run(mux)
}

type Output interface {
	SendBatch(int, zbuf.Batch) error
	SendControl(interface{}) error
	End(interface{}) error
}

// A Query is the internal representation of search query describing a source
// of tuples, a "search" applied to the tuples producing a set of matched
// tuples, and a proc to the process the tuples
type Query struct {
	Space string
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

type driver struct {
	output    Output
	startTime nano.Ts
}

func (d *driver) start(id int64) error {
	return d.output.SendControl(&api.TaskStart{"TaskStart", id})
}

func (d *driver) end(id int64) error {
	return d.output.End(&api.TaskEnd{"TaskEnd", id, nil})
}

func (d *driver) abort(id int64, err error) error {
	verr := &api.Error{Type: "INTERNAL", Message: err.Error()}
	return d.output.SendControl(&api.TaskEnd{"TaskEnd", id, verr})
}

// send a stats update every 500 ms XXX
const statsInterval = time.Millisecond * 500

func (d *driver) sendWarning(warning string) error {
	v := api.SearchWarnings{
		Type:     "SearchWarnings",
		Warnings: []string{warning},
	}
	return d.output.SendControl(v)
}

func (d *driver) sendStats(stats api.ScannerStats) error {
	v := api.SearchStats{
		Type:         "SearchStats",
		StartTime:    d.startTime,
		UpdateTime:   nano.Now(),
		ScannerStats: stats,
	}
	return d.output.SendControl(v)
}

func (d *driver) searchEnd(cid int, stats api.ScannerStats) error {
	err := d.sendStats(stats)
	if err != nil {
		return err
	}
	v := &api.SearchEnd{
		Type:      "SearchEnd",
		ChannelID: cid,
		Reason:    "eof",
	}
	return d.output.SendControl(v)
}

func run(out *proc.MuxOutput, output Output) error {
	//XXX scanner needs to track stats, for now send zeroes
	var stats api.ScannerStats
	d := &driver{
		output:    output,
		startTime: nano.Now(),
	}
	d.start(0)
	ticker := time.NewTicker(statsInterval)
	defer ticker.Stop()
	for !out.Complete() {
		chunk := out.Pull(ticker.C)
		if chunk.Err != nil {
			if chunk.Err == proc.ErrTimeout {
				/* not yet
				err := d.sendStats(out.Stats())
				if err != nil {
					return d.abort(0, err)
				}
				*/
				continue
			}
			if chunk.Err == context.Canceled {
				out.Drain()
				return d.abort(0, errors.New("search job killed"))
			}
			return d.abort(0, chunk.Err)
		}
		if chunk.Warning != "" {
			err := d.sendWarning(chunk.Warning)
			if err != nil {
				return d.abort(0, err)
			}
		}
		if chunk.Batch == nil {
			// a search is done on a channel.  we send stats and
			// a done message for each channel that finishes
			err := d.searchEnd(chunk.ID, stats)
			if err != nil {
				return d.abort(0, err)
			}
		} else {
			err := d.output.SendBatch(chunk.ID, chunk.Batch)
			if err != nil {
				return d.abort(0, err)
			}
		}
	}
	return d.end(0)
}

func launch(ctx context.Context, query *Query, reader zbuf.Reader, zctx *resolver.Context) (*proc.MuxOutput, error) {
	span := query.Span
	if span == (nano.Span{}) {
		span = nano.MaxSpan
	}
	reverse := query.Dir < 0
	return zdriver.Compile(context.Background(), query.Proc, reader, reverse, span, zap.NewNop())
}

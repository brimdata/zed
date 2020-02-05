// Package search provides an implementation for launching zq searches and performing
// analytics on bzng files stored in the server's root directory.
package search

import (
	"context"
	"errors"
	"time"

	"github.com/mccanne/zq/ast"
	"github.com/mccanne/zq/filter"
	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/proc"
	"github.com/mccanne/zq/scanner"
	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zng/resolver"
	"github.com/mccanne/zq/zqd/api"
	"go.uber.org/zap"
)

type Output interface {
	SendBatch(int, zbuf.Batch) error
	SendControl(interface{}) error
	End(interface{}) error
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

func run(out *proc.MuxOutput, output Output, span nano.Span) error {
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
			if !span.Overlaps(chunk.Batch.Span()) {
				continue
			}
			err := d.output.SendBatch(chunk.ID, chunk.Batch)
			if err != nil {
				return d.abort(0, err)
			}
		}
	}
	return d.end(0)
}

// from zq main - move to shared place
func liftFilter(p ast.Proc) (*ast.FilterProc, ast.Proc) {
	if fp, ok := p.(*ast.FilterProc); ok {
		pass := &ast.PassProc{
			Node: ast.Node{"PassProc"},
		}
		return fp, pass
	}
	seq, ok := p.(*ast.SequentialProc)
	if ok && len(seq.Procs) > 0 {
		if fp, ok := seq.Procs[0].(*ast.FilterProc); ok {
			rest := &ast.SequentialProc{
				Node:  ast.Node{"SequentialProc"},
				Procs: seq.Procs[1:],
			}
			return fp, rest
		}
	}
	return nil, nil
}

// from zq main - move to shared place
func compile(ctx *proc.Context, program ast.Proc, reader zbuf.Reader) (*proc.MuxOutput, error) {
	// Try to move the filter into the scanner so we can throw
	// out unmatched records without copying their contents in the
	// case of readers (like zio raw.Reader) that create volatile
	// records that are kepted by the scanner only if matched.
	// For other readers, it certainly doesn't hurt to do this.
	var f filter.Filter
	filterProc, rest := liftFilter(program)
	if filterProc != nil {
		var err error
		f, err = filter.Compile(filterProc.Filter)
		if err != nil {
			return nil, err
		}
		program = rest
	}
	input := scanner.NewScanner(reader, f)
	leaves, err := proc.CompileProc(nil, program, ctx, input)
	if err != nil {
		return nil, err
	}
	return proc.NewMuxOutput(ctx, leaves), nil
}

func launch(ctx context.Context, query *Query, reader zbuf.Reader, zctx *resolver.Context) (*proc.MuxOutput, error) {
	span := query.Span
	if span == (nano.Span{}) {
		span = nano.MaxSpan
	}
	procCtx := &proc.Context{
		Context:     ctx,
		TypeContext: zctx,
		Logger:      zap.NewNop(),
		Reverse:     query.Dir < 0,
		Warnings:    make(chan string, 5),
	}
	return compile(procCtx, query.Proc, reader)
}

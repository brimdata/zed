package zngio

import (
	"context"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/pkg/peeker"
	"github.com/brimdata/zed/zbuf"
)

type scannerSync struct {
	ctx      context.Context
	cancel   context.CancelFunc
	progress zbuf.Progress
	worker   *worker
	parser   parser
	err      error
	eof      bool
}

func newScannerSync(ctx context.Context, zctx *zed.Context, r io.Reader, filter zbuf.Filter, opts ReaderOpts) (*scannerSync, error) {
	ctx, cancel := context.WithCancel(ctx)
	s := &scannerSync{
		ctx:    ctx,
		cancel: cancel,
		parser: parser{
			peeker: peeker.NewReader(r, opts.Size, opts.Max),
			types:  NewDecoder(zctx),
		},
	}
	var bf *expr.BufferFilter
	var f expr.Evaluator
	if filter != nil {
		var err error
		bf, err = filter.AsBufferFilter()
		if err != nil {
			return nil, err
		}
		f, err = filter.AsEvaluator()
		if err != nil {
			return nil, err
		}
	}
	s.worker = newWorker(ctx, &s.progress, bf, f, expr.NewContext(), opts.Validate)
	return s, nil
}

func (s *scannerSync) Pull(done bool) (zbuf.Batch, error) {
	if done {
		s.eof = true
		return nil, nil
	}
	if s.err != nil || s.eof {
		return nil, s.err
	}
again:
	frame, err := s.parser.read()
	if err != nil {
		if err == io.EOF {
			err = nil
		}
		return nil, err
	}
	if frame.zbuf != nil {
		if err := frame.decompress(); err != nil {
			return nil, err
		}
		frame.zbuf.free()
	}
	b, err := s.worker.scanBatch(frame.ubuf, s.parser.types.local)
	if b == nil && err == nil {
		goto again
	}
	return b, err
}

func (s *scannerSync) Progress() zbuf.Progress {
	return s.progress.Copy()
}

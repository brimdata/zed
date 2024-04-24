package zngio

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/peeker"
	"github.com/brimdata/zed/runtime/sam/expr"
	"github.com/brimdata/zed/runtime/sam/op"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zcode"
)

type scanner struct {
	ctx        context.Context
	cancel     context.CancelFunc
	parser     parser
	progress   zbuf.Progress
	validate   bool
	once       sync.Once
	workers    []*worker
	workerCh   chan *worker
	resultChCh chan chan op.Result
	err        error
	eof        bool
}

func newScanner(ctx context.Context, zctx *zed.Context, r io.Reader, filter zbuf.Filter, opts ReaderOpts) (zbuf.Scanner, error) {
	ctx, cancel := context.WithCancel(ctx)
	s := &scanner{
		ctx:    ctx,
		cancel: cancel,
		parser: parser{
			peeker:  peeker.NewReader(r, opts.Size, opts.Max),
			types:   NewDecoder(zctx),
			maxSize: opts.Max,
		},
		validate:   opts.Validate,
		workerCh:   make(chan *worker, opts.Threads+1),
		resultChCh: make(chan chan op.Result, opts.Threads+1),
	}
	for i := 0; i < opts.Threads; i++ {
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
		s.workers = append(s.workers, newWorker(ctx, &s.progress, bf, f, s.validate))
	}
	return s, nil
}

func (s *scanner) Pull(done bool) (zbuf.Batch, error) {
	s.once.Do(s.start)
	if done {
		s.cancel()
		for range s.resultChCh {
			// Wait for the s.parser goroutine to exit so we know it
			// won't continue reading from the underlying io.Reader.
		}
		s.eof = true
		return nil, nil
	}
	if s.err != nil || s.eof {
		return nil, s.err
	}
	for {
		select {
		case ch := <-s.resultChCh:
			result, ok := <-ch
			if !ok {
				continue
			}
			if result.Batch == nil || result.Err != nil {
				if _, ok := result.Err.(*zbuf.Control); !ok {
					s.eof = true
					s.err = result.Err
					s.cancel()
				}
			}
			return result.Batch, result.Err
		case <-s.ctx.Done():
			return nil, s.ctx.Err()
		}
	}
}

func (s *scanner) start() {
	for _, w := range s.workers {
		go w.run(s.ctx, s.workerCh)
	}
	go func() {
		defer close(s.resultChCh)
		// This is the input goroutine that reads message blocks
		// from the input.  Types and control messages are decoded
		// in this thread and data blocks are distributed to the workers
		// with the property that all types for a given data block will
		// exist in the type context before the worker is given the buffer
		// to (optionally) uncompress, filter, and decode when matched.
		// When we hit end-of-stream, a new type context and mapper are
		// created for the new data batches.  Since all data is mapped to
		// the shared context and each worker maps its values independently,
		// the decode pipeline continues to operate concurrenlty without
		// any problem even when faced with changing type contexts.
		for {
			frame, err := s.parser.read()
			if err != nil {
				if _, ok := err.(*zbuf.Control); ok {
					s.sendControl(err)
					continue
				}
				if err == io.EOF {
					err = nil
				}
				s.sendControl(err)
				return
			}
			// Grab a free worker and give it this values message frame to work on
			// along with the present local type context and mapper.
			// We queue up the worker's resultCh so batches are delivered in order.
			select {
			case worker := <-s.workerCh:
				w := work{
					local:    s.parser.types.local,
					frame:    frame,
					resultCh: make(chan op.Result, 1),
				}
				select {
				case s.resultChCh <- w.resultCh:
					select {
					case worker.workCh <- w:
					case <-s.ctx.Done():
						close(w.resultCh)
						return
					}
				case <-s.ctx.Done():
					return
				}
			case <-s.ctx.Done():
				return
			}
		}
	}()
}

// sendControl provides a means for the input thread to send control
// messages and error/EOF in order with the worker threads.
func (s *scanner) sendControl(err error) bool {
	ch := make(chan op.Result, 1)
	ch <- op.Result{Err: err}
	select {
	case s.resultChCh <- ch:
		return true
	case <-s.ctx.Done():
		return false
	}
}

func (s *scanner) Progress() zbuf.Progress {
	return s.progress.Copy()
}

// worker is used by both the non-threaded synchronous scanner as well as
// the threaded scanner.  As long as run() is not called, scanBatch() can
// be safely used without any channel involvement.
type worker struct {
	ctx          context.Context
	zctx         *zed.Context
	progress     *zbuf.Progress
	workCh       chan work
	bufferFilter *expr.BufferFilter
	filter       expr.Evaluator
	ectx         expr.Context
	validate     bool

	mapperLookupCache zed.MapperLookupCache
}

type work struct {
	// Workers need the local context's mapper to map deserialized type IDs
	// into shared-context types and bufferfilter needs its local zctx to
	// interpret serialized type IDs in the raw value message block.
	local    localctx
	frame    frame
	resultCh chan op.Result
}

func newWorker(ctx context.Context, p *zbuf.Progress, bf *expr.BufferFilter, f expr.Evaluator, validate bool) *worker {
	return &worker{
		ctx:          ctx,
		progress:     p,
		workCh:       make(chan work),
		bufferFilter: bf,
		filter:       f,
		ectx:         expr.NewContext(zed.NewArena()),
		validate:     validate,
	}
}

func (w *worker) run(ctx context.Context, workerCh chan<- *worker) {
	for {
		// Tell the scanner we're ready for work.
		select {
		case workerCh <- w:
		case <-w.ctx.Done():
			return
		}
		// Grab the work the scanner gave us.  The scanner will arrange
		// to pull the result off our resultCh and preserve order.
		select {
		case work := <-w.workCh:
			// If the buffer is compressed, decompress it.
			// If not, it wasn't compressed in the original data
			// stream and we handle both cases the same from
			// here on out  The important bit is we are doing
			// the decompress and the boyer-moore short-circuit
			// scan on a processor cache-friendly buffer and
			// throwing it all out asap if it is not needed.
			if work.frame.zbuf != nil {
				if err := work.frame.decompress(); err != nil {
					work.resultCh <- op.Result{Err: err}
					continue
				}
				work.frame.zbuf.free()
			}
			// Either the frame was compressed or it was uncompressed.
			// In either case,the uncompressed data is now in work.blk.
			// We hand ownership of ubuf over to scanBatch.  the zbuf
			// has been freed above so no need to free work.blk.
			// If the batch survives, the work.blk.ubuf will go with it
			// and will get freed when the batch's Unref count hits 0.
			batch, err := w.scanBatch(work.frame.ubuf, work.local)
			if batch != nil || err != nil {
				work.resultCh <- op.Result{Batch: batch, Err: err}
			}
			close(work.resultCh)
		case <-w.ctx.Done():
			return
		}
	}
}

var valsPool = sync.Pool{New: func() any { return make([]zed.Value, 256) }}

func (w *worker) scanBatch(buf *buffer, local localctx) (zbuf.Batch, error) {
	// If w.bufferFilter evaluates to false, we know buf cannot contain
	// records matching w.filter.
	if w.bufferFilter != nil && !w.bufferFilter.Eval(local.zctx, buf.Bytes()) {
		atomic.AddInt64(&w.progress.BytesRead, int64(buf.length()))
		buf.free()
		return nil, nil
	}
	// Otherwise, build a batch by reading all records in the buffer.

	// XXX PR question:
	// we could include the count of records in the values message header...
	// might make allocation work out better; at some point we can have
	// pools of buffers based on size?

	w.mapperLookupCache.Reset(local.mapper)
	arena := zed.NewArenaWithBytes(buf.Bytes(), buf.Free)
	defer arena.Unref()
	vals := valsPool.Get().([]zed.Value)[:0]
	var progress zbuf.Progress
	w.ectx.Arena().Reset()
	for buf.length() > 0 {
		val, length, err := w.decodeVal(arena, buf)
		if err != nil {
			return nil, errBadFormat
		}
		if w.wantValue(val, length, &progress) {
			vals = append(vals, val)
		}
	}
	w.progress.Add(progress)
	if len(vals) == 0 {
		return nil, nil
	}
	return zbuf.NewBatchWithVarsAndFree(arena, vals, nil, func() { valsPool.Put(vals) }), nil
}

func (w *worker) decodeVal(arena *zed.Arena, buf *buffer) (zed.Value, int64, error) {
	id, err := readUvarintAsInt(buf)
	if err != nil {
		return zed.Null, 0, err
	}
	n, err := zcode.ReadTag(buf)
	if err != nil {
		return zed.Null, 0, errBadFormat
	}
	typ := w.mapperLookupCache.Lookup(id)
	if typ == nil {
		return zed.Null, 0, fmt.Errorf("zngio: type ID %d not in context", id)
	}
	if n < 0 {
		return arena.New(typ, nil), 0, nil
	}
	off := buf.off
	if n > 0 {
		if _, err := buf.read(n); err != nil && err != io.EOF {
			if err == peeker.ErrBufferOverflow {
				return zed.Null, 0, fmt.Errorf("large value of %d bytes exceeds maximum read buffer", n)
			}
			return zed.Null, 0, errBadFormat
		}
	}
	val := arena.NewFromOffsetAndLength(typ, off, n)
	if w.validate {
		if err := val.Validate(); err != nil {
			return zed.Null, 0, err
		}
	}
	return val, int64(n), nil
}

func (w *worker) wantValue(val zed.Value, length int64, progress *zbuf.Progress) bool {
	progress.BytesRead += length
	progress.RecordsRead++
	if f := w.filter; f != nil {
		// It's tempting to call w.bufferFilter.Eval on val.Bytes here,
		// but that might call FieldNameFinder.Find, which could explode
		// or return false negatives because it expects a sequence of
		// (type ID, tag, ZNG value) but val.Bytes is just a ZNG value.
		if !f.Eval(w.ectx, val).Ptr().AsBool() {
			return false
		}
	}
	progress.BytesMatched += length
	progress.RecordsMatched++
	return true
}

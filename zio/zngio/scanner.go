package zngio

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/pkg/peeker"
	"github.com/brimdata/zed/proc"
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
	resultChCh chan chan proc.Result
	err        error
	eof        bool
}

func newScanner(ctx context.Context, zctx *zed.Context, r io.Reader, filter zbuf.Filter, opts ReaderOpts) (zbuf.Scanner, error) {
	ctx, cancel := context.WithCancel(ctx)
	s := &scanner{
		ctx:    ctx,
		cancel: cancel,
		parser: parser{
			peeker: peeker.NewReader(r, opts.Size, opts.Max),
			types:  NewDecoder(zctx),
		},
		validate:   opts.Validate,
		workerCh:   make(chan *worker),
		resultChCh: make(chan chan proc.Result, opts.Threads+1),
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
		s.workers = append(s.workers, newWorker(ctx, &s.progress, bf, f, expr.NewContext(), s.validate))
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
				if err, ok := result.Err.(*zbuf.Control); !ok {
					s.eof = true
					s.err = err
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
					resultCh: make(chan proc.Result),
				}
				select {
				case s.resultChCh <- w.resultCh:
					select {
					case worker.workCh <- w:
					case <-s.ctx.Done():
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
	ch := make(chan proc.Result, 1)
	ch <- proc.Result{Err: err}
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
	progress     *zbuf.Progress
	workCh       chan work
	bufferFilter *expr.BufferFilter
	filter       expr.Evaluator
	ectx         expr.Context
	validate     bool
}

type work struct {
	// Workers need the local context's mapper to map deserialized type IDs
	// into shared-context types and bufferfilter needs its local zctx to
	// interpret serialized type IDs in the raw value message block.
	local    localctx
	frame    frame
	resultCh chan proc.Result
}

func newWorker(ctx context.Context, p *zbuf.Progress, bf *expr.BufferFilter, f expr.Evaluator, ectx expr.Context, validate bool) *worker {
	return &worker{
		ctx:          ctx,
		progress:     p,
		workCh:       make(chan work),
		bufferFilter: bf,
		filter:       f,
		ectx:         ectx, //XXX
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
					work.resultCh <- proc.Result{Err: err}
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
				work.resultCh <- proc.Result{Batch: batch, Err: err}
			}
			close(work.resultCh)
		case <-w.ctx.Done():
			return
		}
	}
}

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

	batch := newBatch(buf)
	var progress zbuf.Progress
	// We extend the batch one past its end and decode into the next
	// potential slot and only advance the batch when we decide we want to
	// keep the value.  Since we overshoot by one slot on every pass,
	// we delete the overshoot with batch.unextend() on exit from the loop.
	// I think this is what I drew on the Lawton basement whiteboard
	// in 2018 but my previous attempts implementing that picture were
	// horrible.  This attempts isn't so bad.
	valRef := batch.extend()
	for buf.length() > 0 {
		if err := decodeVal(buf, local.mapper, w.validate, valRef); err != nil {
			buf.free()
			return nil, err
		}
		if w.wantValue(valRef, &progress) {
			valRef = batch.extend()
		}
	}
	batch.unextend()
	w.progress.Add(progress)
	if len(batch.Values()) == 0 {
		batch.Unref()
		return nil, nil
	}
	return batch, nil
}

func decodeVal(r reader, m *zed.Mapper, validate bool, valRef *zed.Value) error {
	id, err := readUvarintAsInt(r)
	if err != nil {
		return err
	}
	n, err := zcode.ReadTag(r)
	if err != nil {
		return zed.ErrBadFormat
	}
	var b []byte
	if n == 0 {
		b = []byte{}
	} else if n > 0 {
		b, err = r.read(n)
		if err != nil && err != io.EOF {
			if err == peeker.ErrBufferOverflow {
				return fmt.Errorf("zngio: large value of %d bytes exceeds maximum read buffer", n)
			}
			return zed.ErrBadFormat
		}
	}
	typ := m.Lookup(id)
	if typ == nil {
		return fmt.Errorf("zngio: type ID %d not in context", id)
	}
	valRef.Type = typ
	valRef.Bytes = b
	if validate {
		if err := Validate(valRef); err != nil {
			return err
		}
	}
	return nil
}

func (w *worker) wantValue(val *zed.Value, progress *zbuf.Progress) bool {
	progress.BytesRead += int64(len(val.Bytes))
	progress.RecordsRead++
	// It's tempting to call w.bufferFilter.Eval on rec.Bytes here, but that
	// might call FieldNameFinder.Find, which could explode or return false
	// negatives because it expects a buffer of ZNG value messages, and
	// rec.Bytes is just a ZNG value.  (A ZNG value message is a header
	// indicating a type ID followed by a value of that type.)
	if w.filter == nil || check(w.ectx, val, w.filter) {
		progress.BytesMatched += int64(len(val.Bytes))
		progress.RecordsMatched++
		return true
	}
	return false
}

func check(ectx expr.Context, this *zed.Value, filter expr.Evaluator) bool {
	val := filter.Eval(ectx, this)
	return val.Type == zed.TypeBool && zed.IsTrue(val.Bytes)
}

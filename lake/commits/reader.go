package commits

import (
	"context"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

type LogReader struct {
	ctx       context.Context
	marshaler *zson.MarshalZNGContext
	store     *Store
	cursor    ksuid.KSUID
	stop      ksuid.KSUID
}

var _ zio.Reader = (*LogReader)(nil)

func newLogReader(ctx context.Context, zctx *zed.Context, store *Store, leaf, stop ksuid.KSUID) *LogReader {
	m := zson.NewZNGMarshalerWithContext(zctx)
	m.Decorate(zson.StyleSimple)
	return &LogReader{
		ctx:       ctx,
		marshaler: m,
		store:     store,
		cursor:    leaf,
		stop:      stop,
	}
}

func (r *LogReader) Read() (*zed.Value, error) {
	if r.cursor == ksuid.Nil {
		return nil, nil
	}
	_, commitObject, err := r.store.GetBytes(r.ctx, r.cursor)
	if err != nil {
		return nil, err
	}
	next := commitObject.Parent
	if next == r.stop {
		next = ksuid.Nil
	}
	r.cursor = next
	return r.marshaler.MarshalRecord(commitObject)
}

type rawReader struct {
	ctx    context.Context
	store  *Store
	cursor ksuid.KSUID
	stop   ksuid.KSUID
	spill  []byte
}

func newRawReader(ctx context.Context, store *Store, leaf, stop ksuid.KSUID) io.Reader {
	return &rawReader{
		ctx:    ctx,
		store:  store,
		cursor: leaf,
		stop:   stop,
	}
}

func (r *rawReader) Read(b []byte) (int, error) {
	return r.next(b)
	// var cc int
	// for cc < len(b) {
	// n, err := r.next(b[cc:])
	// cc += n
	// if err != nil {
	// return cc, err
	// }
	// }
	// return cc, nil
}

func (r *rawReader) next(b []byte) (int, error) {
	if len(r.spill) > 0 {
		s := r.spill
		r.spill = r.spill[:0]
		return r.copyTo(b, s), nil
	}
	if r.cursor == ksuid.Nil {
		return 0, io.EOF
	}
	bytes, commitObject, err := r.store.GetBytes(r.ctx, r.cursor)
	if err != nil {
		return 0, err
	}
	next := commitObject.Parent
	if next == r.stop {
		next = ksuid.Nil
	}
	r.cursor = next
	return r.copyTo(b, bytes), nil
}

func (r *rawReader) copyTo(dst, src []byte) int {
	n := copy(dst, src)
	if n < len(src) {
		r.spill = append(r.spill[:0], src[n:]...)
	}
	return n
}

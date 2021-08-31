package commits

import (
	"context"
	"io"

	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

type Reader struct {
	ctx    context.Context
	store  *Store
	cursor ksuid.KSUID
	stop   ksuid.KSUID
	buffer []byte
}

func newReader(ctx context.Context, store *Store, leaf, stop ksuid.KSUID) *Reader {
	return &Reader{
		ctx:    ctx,
		store:  store,
		cursor: leaf,
		stop:   stop,
	}
}

func (r *Reader) Read(b []byte) (int, error) {
	if len(r.buffer) == 0 {
		if err := r.fill(); err != nil {
			return 0, err
		}
	}
	n := copy(b, r.buffer)
	r.buffer = r.buffer[n:]
	return n, nil
}

func (r *Reader) fill() error {
	if r.cursor == ksuid.Nil {
		return io.EOF
	}
	b, commit, err := r.store.GetBytes(r.ctx, r.cursor)
	if err != nil {
		return err
	}
	next := commit.Parent
	if next == r.stop {
		next = ksuid.Nil
	}
	r.cursor = next
	r.buffer = b
	return err
}

type LogReader struct {
	ctx       context.Context
	marshaler *zson.MarshalZNGContext
	store     *Store
	cursor    ksuid.KSUID
	stop      ksuid.KSUID
}

var _ zio.Reader = (*LogReader)(nil)

func newLogReader(ctx context.Context, zctx *zson.Context, store *Store, leaf, stop ksuid.KSUID) *LogReader {
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

func (r *LogReader) Read() (*zng.Record, error) {
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

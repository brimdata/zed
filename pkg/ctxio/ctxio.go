// Package ctxio provides functionality similar to package io but with the
// ability to abort long running operations by passing through a
// context.Context.
package ctxio

import (
	"context"
	"io"
)

type writer struct {
	io.Writer
	ctx context.Context
}

func NewWriter(ctx context.Context, w io.Writer) io.Writer {
	return &writer{w, ctx}
}

func (w *writer) Write(p []byte) (n int, err error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	return w.Writer.Write(p)
}

type reader struct {
	io.Reader
	ctx context.Context
}

func NewReader(ctx context.Context, r io.Reader) io.Reader {
	return &reader{r, ctx}
}

func (r *reader) Read(p []byte) (n int, err error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.Reader.Read(p)
}

func Copy(ctx context.Context, dst io.Writer, src io.Reader) (n int64, err error) {
	if _, ok := src.(io.WriterTo); ok {
		dst = NewWriter(ctx, dst)
	} else {
		src = NewReader(ctx, src)
	}
	return io.Copy(dst, src)
}

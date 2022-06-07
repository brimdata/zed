package index

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	_ "net/http/pprof"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
)

var ErrNotFound = errors.New("key not found")

// Finder looks up values in a microindex using its embedded index.
type Finder struct {
	*Reader
	keyer *Keyer
	zctx  *zed.Context
	uri   *storage.URI
}

type KeyValue struct {
	Key   field.Path
	Value zed.Value
}

// NewFinder returns an object that is used to lookup keys in a microindex.
// It opens the file and reads the trailer, returning errors if the file is
// corrupt, doesn't exist, or has an invalid trailer.  If the microindex exists
// but is empty, zero values are returned for any lookups. If the microindex
// does not exist, a wrapped zqe.NotFound error is returned.
func NewFinder(ctx context.Context, zctx *zed.Context, engine storage.Engine, uri *storage.URI) (*Finder, error) {
	reader, err := NewReaderFromURI(ctx, zctx, engine, uri)
	if err != nil {
		return nil, err
	}
	return &Finder{
		Reader: reader,
		zctx:   zctx,
		uri:    uri,
	}, nil
}

type frame struct {
	offset int64
	last   *zed.Value
}

func (f *Finder) searchSection(reader zio.ReadCloser, eval *expr.SpanFilter) ([]frame, error) {
	defer reader.Close()
	var frames []frame
	p := zio.NewPeeker(reader)
	for {
		val, err := p.Read()
		if val == nil || err != nil {
			return frames, err
		}
		val = val.Copy()
		next, err := p.Peek()
		if next == nil || err != nil {
			return frames, err
		}
		lower := val.DerefPath(f.meta.Keys[0])
		upper := next.DerefPath(f.meta.Keys[0])
		if eval.Eval(lower, upper) {
			continue
		}
		child := val.Deref(f.meta.ChildOffsetField)
		if child == nil {
			return nil, fmt.Errorf("B-tree child field is missing")
		}
		start := child.AsInt()
		if n := len(frames); n > 0 && bytes.Compare(frames[n-1].last.Bytes, lower.Bytes) == 0 {
			frames[n-1].last = upper.Copy()
			continue
		}
		frames = append(frames, frame{start, upper.Copy()})
	}
}

func (f *Finder) search(eval *expr.SpanFilter) (zio.ReadCloser, error) {
	n := len(f.sections)
	if n == 1 {
		return f.newSectionReader(0, 0), nil
	}
	var frames []frame
	var reader zio.ReadCloser = f.newSectionReader(1, 0)
	for level := 1; level < n; level++ {
		if level > 1 {
			reader = f.newFramesReader(level, frames)
		}
		var err error
		frames, err = f.searchSection(reader, eval)
		if err != nil {
			return nil, err
		}
		fmt.Println("section frames", len(frames))
		if len(frames) == 0 {
			return nil, ErrNotFound
		}
	}
	return f.newFramesReader(0, frames), nil
}

func (f *Finder) Lookup(ctx context.Context, kvs ...KeyValue) (*zed.Value, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	hits := make(chan *zed.Value)
	var err error
	go func() {
		err = f.LookupAll(ctx, hits, kvs)
		close(hits)
	}()
	select {
	case val := <-hits:
		return val, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (f *Finder) LookupAll(ctx context.Context, hits chan<- *zed.Value, kvs []KeyValue) error {
	// XXX Enable primary, tertiary key lookups.
	val, err := zson.FormatValue(&kvs[0].Value)
	if err != nil {
		return err
	}
	e := &dag.BinaryExpr{
		Op:  "==",
		LHS: &dag.This{Path: kvs[0].Key},
		RHS: &dag.Literal{Value: val},
	}
	return f.Filter(ctx, hits, e)
}

func lookup(reader zio.Reader, valFilter expr.Evaluator) (*zed.Value, error) {
	var ectx alloc
	for {
		val, err := reader.Read()
		if val == nil || err != nil {
			return nil, err
		}
		result := valFilter.Eval(&ectx, val)
		if result.IsMissing() {
			continue
		}
		if result.Type != zed.TypeBool {
			panic("result from value filter not a bool: " + zson.String(val))
		}
		if !zed.DecodeBool(result.Bytes) {
			continue
		}
		return val.Copy(), nil
	}
}

func (f *Finder) Filter(ctx context.Context, hits chan<- *zed.Value, e dag.Expr) error {
	spanFilter, valueFilter, err := compileFilter(e, f.meta.Keys[0], f.meta.Order)
	if err != nil {
		return err
	}
	reader, err := f.search(spanFilter)
	if err != nil {
		return err
	}
	defer reader.Close()
	for ctx.Err() == nil {
		val, err := lookup(reader, valueFilter)
		if val == nil || err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case hits <- val:
		}
	}
	return ctx.Err()
}

// ParseKeys uses the key template from the microindex trailer to parse
// a slice of string values which correspnod to the DFS-order
// of the fields in the key.  The inputs may be smaller than the
// number of key fields, in which case they are "don't cares"
// in terms of key lookups.  Any don't-care fields must all be
// at the end of the key record.
func (f *Finder) ParseKeys(inputs ...string) ([]KeyValue, error) {
	if f.IsEmpty() {
		return nil, nil
	}
	keys := f.meta.Keys
	if len(inputs) > len(keys) {
		return nil, fmt.Errorf("too many keys: expected at most %d but got %d", len(keys), len(inputs))
	}
	kvs := make([]KeyValue, 0, len(inputs))
	for k := range inputs {
		if k < len(inputs) {
			s := inputs[k]
			zv, err := zson.ParseValue(f.zctx, s)
			if err != nil {
				return nil, fmt.Errorf("could not parse %q: %w", s, err)
			}
			kvs = append(kvs, KeyValue{Key: keys[k], Value: *zv})
		}
	}
	return kvs, nil
}

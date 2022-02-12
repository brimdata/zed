package index

import (
	"context"
	"fmt"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/index"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime/expr/extent"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

type expr func(context.Context, *Filter, ksuid.KSUID, []Rule) <-chan result

type result struct {
	err  error
	span extent.Span
}

func compileExpr(node dag.Expr) (expr, error) {
	e, ok := node.(*dag.BinaryExpr)
	if !ok {
		return nil, fmt.Errorf("index filter: unknown type: %T", node)
	}
	switch e.Op {
	case "or", "and":
		lhs, err := compileExpr(e.LHS)
		if err != nil {
			return nil, err
		}
		rhs, err := compileExpr(e.RHS)
		if err != nil {
			return nil, err
		}
		return logicalExpr(lhs, rhs, e.Op), nil
	case "=", "<", "<=", ">", ">=":
		literal, ok := e.RHS.(*dag.Literal)
		if !ok {
			return nil, fmt.Errorf("index comparator: RHS is not a literal: %T", e.RHS)
		}
		zv, err := zson.ParseValue(zed.NewContext(), literal.Value)
		if err != nil {
			return nil, err
		}
		this, ok := e.LHS.(*dag.This)
		if !ok {
			return nil, fmt.Errorf("index comparator: LHS is not a field path: %T", e.LHS)
		}
		kv := index.KeyValue{Key: this.Path, Value: *zv}
		return compareExpr(kv, e.Op)
	default:
		return nil, fmt.Errorf("unsupported binary expression op: %s", e.Op)
	}
}

func logicalExpr(lhs, rhs expr, op string) expr {
	return func(ctx context.Context, f *Filter, oid ksuid.KSUID, rules []Rule) <-chan result {
		lch := lhs(ctx, f, oid, rules)
		rch := rhs(ctx, f, oid, rules)
		if lch == nil || rch == nil {
			return notNil(lch, rch)
		}
		c := make(chan result, 1)
		go func() {
			if op == "or" {
				c <- orExpr(merge(lch, rch))
			} else {
				c <- andExpr(merge(lch, rch))
			}
			close(c)
		}()
		return c
	}
}

func orExpr(c <-chan result) result {
	var res result
	for r := range c {
		if r.span != nil {
			res = appendResult(res, r)
		}
		if r.err != nil {
			return r
		}
	}
	return res
}

func andExpr(c <-chan result) result {
	var res result
	for r := range c {
		if r.span == nil || r.err != nil {
			return r
		}
		res = appendResult(res, r)
	}
	return res
}

func appendResult(a, b result) result {
	if a.span == nil {
		a.span = b.span
	} else {
		a.span.Extend(b.span.First())
		a.span.Extend(b.span.Last())
	}
	return a
}

func compareExpr(kv index.KeyValue, op string) (expr, error) {
	return func(ctx context.Context, f *Filter, oid ksuid.KSUID, rules []Rule) <-chan result {
		kv, rule := matchFieldRule(rules, kv)
		if rule == nil {
			return nil
		}
		// The output of ch may not be read so make this a buffered channel so
		// this goroutine does not block indefinitely.
		ch := make(chan result, 1)
		go func() {
			var r result
			if r.err = f.sem.Acquire(ctx, 1); r.err == nil {
				r.span, r.err = f.find(ctx, oid, rule.RuleID(), kv, op)
			}
			f.sem.Release(1)
			ch <- r
			close(ch)
		}()
		return ch
	}, nil
}

func matchFieldRule(rules []Rule, in index.KeyValue) (index.KeyValue, Rule) {
	for _, rule := range rules {
		// XXX support indexes with multiple keys #3162
		// and other rule types.
		if fr, ok := rule.(*FieldRule); ok && in.Key.Equal(fr.Fields[0]) {
			return index.KeyValue{
				Key:   append(field.New("key"), in.Key...),
				Value: in.Value,
			}, rule
		}
	}
	return in, nil
}

// merge is taken from https://go.dev/blog/pipelines
func merge(cs ...<-chan result) <-chan result {
	var wg sync.WaitGroup
	// The returned channel may not be read fully so make channels the size
	// of expected results so our goroutines do not block forever.
	out := make(chan result, len(cs))
	output := func(c <-chan result) {
		for r := range c {
			out <- r
		}
		wg.Done()
	}
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func notNil(cs ...<-chan result) <-chan result {
	for _, c := range cs {
		if c != nil {
			return c
		}
	}
	return nil
}

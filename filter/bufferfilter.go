package filter

import (
	"fmt"
	"unicode"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/pkg/byteconv"
	"github.com/brimsec/zq/zng"
)

const (
	opAnd = iota
	opOr
	opStringFinder
)

// BufferFilter is a filter for byte slices containing ZNG values.
type BufferFilter struct {
	op           int
	left         *BufferFilter
	right        *BufferFilter
	stringFinder *stringFinder
}

// NewBufferFilter tries to return a BufferFilter for e such that the
// BufferFilter's Eval method returns true for any byte slice containing the ZNG
// encoding of a record matching e. (It may also return true for some byte
// slices that do not match.) NewBufferFilter returns a nil pointer and nil
// error if it cannot construct a useful filter.
func NewBufferFilter(e ast.BooleanExpr) (*BufferFilter, error) {
	switch e := e.(type) {
	case *ast.CompareAny:
		if e.Comparator != "=" && e.Comparator != "in" {
			return nil, nil
		}
		return newBufferFilterForLiteral(e.Value)
	case *ast.CompareField:
		if e.Comparator != "=" && e.Comparator != "in" {
			return nil, nil
		}
		if fc, ok := e.Field.(*ast.FieldCall); ok && fc.Fn == "Len" {
			return nil, nil
		}
		return newBufferFilterForLiteral(e.Value)
	case *ast.LogicalAnd:
		left, err := NewBufferFilter(e.Left)
		if err != nil {
			return nil, err
		}
		right, err := NewBufferFilter(e.Right)
		if err != nil {
			return nil, err
		}
		if left == nil {
			return right, nil
		}
		if right == nil {
			return left, nil
		}
		return &BufferFilter{op: opAnd, left: left, right: right}, nil
	case *ast.LogicalOr:
		left, err := NewBufferFilter(e.Left)
		if err != nil {
			return nil, err
		}
		right, err := NewBufferFilter(e.Right)
		if left == nil || right == nil || err != nil {
			return nil, err
		}
		return &BufferFilter{op: opOr, left: left, right: right}, nil
	case *ast.LogicalNot, *ast.MatchAll:
		return nil, nil
	case *ast.Search:
		if e.Value.Type == "net" || e.Value.Type == "regexp" {
			return nil, nil
		}
		if e.Value.Type == "string" {
			// TODO Match the behavior of searchRecordString.
			return nil, nil
		}
		for _, r := range e.Text {
			// TODO Make stringFinder insensitive to case. Until
			// then, bail if e.Text contains a letter.
			if unicode.IsLetter(r) {
				return nil, nil
			}
		}
		left := &BufferFilter{
			op:           opStringFinder,
			stringFinder: makeStringFinder(e.Text),
		}
		right, err := newBufferFilterForLiteral(e.Value)
		if right == nil || err != nil {
			return nil, err
		}
		return &BufferFilter{op: opOr, left: left, right: right}, nil
	default:
		panic(fmt.Sprintf("BufferFilter: unknown type %T", e))
	}
}

func newBufferFilterForLiteral(l ast.Literal) (*BufferFilter, error) {
	switch l.Type {
	case "bool", "byte", "int16", "uint16", "int32", "uint32", "int64", "uint64", "float64", "time", "duration":
		// These are all comparable, so they can require up to three
		// patterns: float, varint, and uvarint.
		return nil, nil
	case "null":
		return nil, nil
	case "regexp":
		// Could try to extract a pattern (e.g., "efg" from "(ab|cd)(efg)+[hi]").
		return nil, nil
	case "string":
		// Match the behavior of zng.ParseLiteral.
		l.Type = "bstring"
	}
	v, err := zng.Parse(l)
	if err != nil {
		return nil, err
	}
	// We're looking for a complete ZNG value, so we can lengthen the
	// pattern by calling Encode to add a tag.
	pattern := string(v.Encode(nil))
	if len(pattern) < 2 {
		// Very short patterns are unprofitable.
		return nil, nil
	}
	return &BufferFilter{
		op:           opStringFinder,
		stringFinder: makeStringFinder(pattern),
	}, nil
}

// Eval returns true if buf matches the receiver and false otherwise.
func (b *BufferFilter) Eval(buf []byte) bool {
	switch b.op {
	case opAnd:
		return b.left.Eval(buf) && b.right.Eval(buf)
	case opOr:
		return b.left.Eval(buf) || b.right.Eval(buf)
	case opStringFinder:
		return b.stringFinder.next(byteconv.UnsafeString(buf)) > -1
	default:
		panic(fmt.Sprintf("BufferFilter: unknown op %d", b.op))
	}
}

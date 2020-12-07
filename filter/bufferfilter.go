package filter

import (
	"fmt"
	"unicode/utf8"

	"github.com/brimsec/zq/pkg/byteconv"
	"github.com/brimsec/zq/zng/resolver"
)

const (
	opAnd = iota
	opOr
	opFieldNameFinder
	opStringCaseFinder
	opStringFinder
)

// BufferFilter is a filter for byte slices containing ZNG values.
type BufferFilter struct {
	op    int
	left  *BufferFilter
	right *BufferFilter
	fnf   *fieldNameFinder
	scf   *stringCaseFinder
	sf    *stringFinder
}

func NewAndBufferFilter(left, right *BufferFilter) *BufferFilter {
	return &BufferFilter{op: opAnd, left: left, right: right}
}

func NewOrBufferFilter(left, right *BufferFilter) *BufferFilter {
	return &BufferFilter{op: opOr, left: left, right: right}
}

func NewFieldNameFinder(pattern string) *BufferFilter {
	return &BufferFilter{
		op:  opFieldNameFinder,
		fnf: newFieldNameFinder(pattern),
	}
}

func NewBufferFilterForString(pattern string) *BufferFilter {
	if len(pattern) < 2 {
		// Very short patterns are unprofitable.
		return nil
	}
	return &BufferFilter{op: opStringFinder, sf: makeStringFinder(pattern)}
}

func NewBufferFilterForStringCase(pattern string) *BufferFilter {
	if len(pattern) < 2 {
		// Very short patterns are unprofitable.
		return nil
	}
	for _, r := range pattern {
		if r >= utf8.RuneSelf {
			// stringCaseFinder is sensitive to case for letters
			// with multibyte UTF-8 encodings.
			return nil
		}
	}
	return &BufferFilter{op: opStringCaseFinder, scf: makeStringCaseFinder(pattern)}
}

// Eval returns true if buf matches the receiver and false otherwise.
func (b *BufferFilter) Eval(zctx *resolver.Context, buf []byte) bool {
	switch b.op {
	case opAnd:
		return b.left.Eval(zctx, buf) && b.right.Eval(zctx, buf)
	case opOr:
		return b.left.Eval(zctx, buf) || b.right.Eval(zctx, buf)
	case opFieldNameFinder:
		return b.fnf.find(zctx, buf)
	case opStringCaseFinder:
		return b.scf.next(byteconv.UnsafeString(buf)) > -1
	case opStringFinder:
		return b.sf.next(byteconv.UnsafeString(buf)) > -1
	default:
		panic(fmt.Sprintf("BufferFilter: unknown op %d", b.op))
	}
}

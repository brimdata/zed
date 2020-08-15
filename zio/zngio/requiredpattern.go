package zngio

import (
	"fmt"
	"strconv"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/pkg/byteconv"
	"github.com/brimsec/zq/zng"
)

const (
	opAnd = iota
	opNot
	opOr
	opStringFinder
	opTrue
)

type requiredPatternFinder struct {
	op           int
	left         *requiredPatternFinder
	right        *requiredPatternFinder
	stringFinder *stringFinder
}

// newRequiredPatternFinder returns a requiredPatternFinder for e. A required
// pattern for e is a byte sequence that must be present in any buffer that
// contains the ZNG encoding of a record matching e.
func newRequiredPatterFinder(e ast.BooleanExpr) (*requiredPatternFinder, error) {
	switch e := e.(type) {
	case *ast.CompareAny:
		return patternFinderForCompare(e.Comparator, e.Value)
	case *ast.CompareField:
		return patternFinderForCompare(e.Comparator, e.Value)
	case *ast.LogicalAnd:
		left, err := newRequiredPatterFinder(e.Left)
		if err != nil {
			return nil, err
		}
		right, err := newRequiredPatterFinder(e.Right)
		if err != nil {
			return nil, err
		}
		if left == nil {
			return right, nil
		}
		if right == nil {
			return left, nil
		}
		return &requiredPatternFinder{op: opAnd, left: left, right: right}, nil
	case *ast.LogicalOr:
		left, err := newRequiredPatterFinder(e.Left)
		if err != nil {
			return nil, err
		}
		right, err := newRequiredPatterFinder(e.Right)
		if err != nil {
			return nil, err
		}
		if left == nil || right == nil {
			return nil, nil
		}
		return &requiredPatternFinder{op: opOr, left: left, right: right}, nil
	case *ast.LogicalNot:
		expr, err := newRequiredPatterFinder(e.Expr)
		if expr == nil || err != nil {
			return nil, err
		}
		return &requiredPatternFinder{op: opNot, left: expr}, nil
	case *ast.MatchAll:
		return &requiredPatternFinder{op: opTrue}, nil
	case *ast.Search:
		if e.Value.Type == "net" || e.Value.Type == "regexp" {
			// Match behavior of filter.compileSearch.
			return nil, nil
		}
		if e.Value.Type == "string" {
			// filter.searchRecordString matches field names and values.
			// xxx
			return nil, nil
		}
		// Match behavior of filter.searchRecordOther.
		or := &ast.LogicalOr{
			Left: &ast.CompareField{
				Comparator: "=",
				Value: ast.Literal{
					Type:  "string",
					Value: e.Text,
				},
			},
			Right: &ast.CompareField{
				Comparator: "=",
				Value:      e.Value,
			},
		}
		return newRequiredPatterFinder(or)
	default:
		panic(fmt.Sprintf("unknown type %T", e))
	}
}

// requiredPatternForCompare tries to determine a required pattern for an
// ast.CompareAny or ast.CompareField with comparator and value. If it cannot,
// it returns the empty string.
func patternFinderForCompare(comparator string, value ast.Literal) (*requiredPatternFinder, error) {
	if comparator != "=" && comparator != "in" {
		return nil, nil
	}
	if value.Type == "regexp" {
		return nil, nil
	}
	if value.Type == "string" {
		// This matches the behavior of zng.ParseLiteral.
		value.Type = "bstring"
	}
	v, err := zng.Parse(value)
	if err != nil {
		return nil, err
	}
	// We're looking for a complete ZNG value, so we can lengthen the
	// pattern by calling Encode to add a tag.
	pattern := string(v.Encode(nil))
	if len(pattern) <= 1 {
		// Not profitable if pattern is short.
		return nil, err
	}
	return &requiredPatternFinder{op: opStringFinder, stringFinder: makeStringFinder(pattern)}, nil
}

func (r *requiredPatternFinder) find(b []byte) bool {
	switch r.op {
	case opAnd:
		return r.left.find(b) && r.right.find(b)
	case opNot:
		return !r.left.find(b)
	case opOr:
		return r.left.find(b) || r.right.find(b)
	case opStringFinder:
		return r.stringFinder.next(byteconv.UnsafeString(b)) > -1
	case opTrue:
		return true
	default:
		panic("unknown op " + strconv.Itoa(r.op))
	}
}

package filter

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"

	"github.com/mccanne/zq/ast"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zval"
)

type Filter func(*zson.Record) bool

func LogicalAnd(left, right Filter) Filter {
	return func(p *zson.Record) bool { return left(p) && right(p) }
}

func LogicalOr(left, right Filter) Filter {
	return func(p *zson.Record) bool { return left(p) || right(p) }
}

func LogicalNot(expr Filter) Filter {
	return func(p *zson.Record) bool { return !expr(p) }
}

func SearchString(s string) Filter {
	pattern := []byte(s)
	return func(p *zson.Record) bool {
		// Go implements a very efficient string search algorithm so we
		// use it here first to rule out misses on a substring match.
		if !bytes.Contains(p.Raw, pattern) {
			return false
		}
		// If we have a hit, double check field by field in case the
		// framing bytes give us a false positive.
		// XXX we should refactor these iterators to make this tighter.
		it := p.ZvalIter()
		for _, c := range p.Type.Columns {
			val, _, err := it.Next()
			if err != nil {
				return false
			}
			switch c.Type.(type) {
			case *zeek.TypeSet, *zeek.TypeVector:
				for it2 := zval.Iter(val); !it2.Done(); {
					val2, _, err := it2.Next()
					if err != nil {
						return false
					}
					if bytes.Contains(val2, pattern) {
						return true
					}
				}
			default:
				if bytes.Contains(val, pattern) {
					return true
				}
			}
		}
		return false
	}
}

type ValResolver func(*zson.Record) (zeek.Type, []byte)

// fieldop, arrayIndex, and fieldRead are helpers used internally
// by CompileFieldExpr() below.
type fieldop interface {
	apply(zeek.Type, []byte) (zeek.Type, []byte)
}

type arrayIndex struct {
	idx int64
}
func (ai *arrayIndex) apply(typ zeek.Type, val []byte) (zeek.Type, []byte) {
	elType, elVal, err := zeek.VectorIndex(typ, val, ai.idx)
	if err != nil {
		return nil, nil
	}
	return elType, elVal
}

type fieldRead struct {
	field string
}
func (fr *fieldRead) apply(typ zeek.Type, val []byte) (zeek.Type, []byte) {
	recType, ok := typ.(*zeek.TypeRecord)
	if !ok {
		// field reference on non-record type
		return nil, nil
	}

	// XXX searching the list of columns for every record is
	// expensive, but we can receive records with different
	// types so caching this isn't straightforward.
	for n, col := range(recType.Columns) {
		if col.Name == fr.field {
			var v []byte
			it := zval.Iter(val)
			for i := 0; i <= n; i++ {
				if it.Done() {
					return nil, nil
				}
				var err error
				v, _, err = it.Next()
				if err != nil {
					return nil, nil
				}
			}
			return col.Type, v
		}
	}
	// record doesn't have the named field
	return nil, nil
}

// CompileFieldExpr() takes a FieldExpr AST (which represents either a
// simple field reference like "fieldname" or something more complex
// like "fieldname[0].subfield.subsubfield[3]") and compiles it into a
// ValResolver -- a function that takes a zson.Record and extracts the
// value to which the FieldExpr refers.  If the FieldExpr cannot be
// compiled, this function returns an error.  If the resolver is given
// a record for which the given expression cannot be evaluated (e.g.,
// if the record doesn't have a requested field or an array index is
// out of bounds), the resolver returns (nil, nil).
func CompileFieldExpr(node ast.FieldExpr) (ValResolver, error) {
	var ops []fieldop = make([]fieldop, 0)
	var field string

	// First collapse the AST to a simple array of fieldop
	// objects so the actual resolver can run reasonably efficiently.
outer:
	for {
		switch op := node.(type) {
		case *ast.FieldRead:
			field = op.Field
			break outer
		case *ast.FieldCall:
			switch op.Fn {
			// Len handled separately
			case "Index":
				idx, err := strconv.ParseInt(op.Param, 10, 64)
				if err != nil {
					return nil, err
				}
				ops = append([]fieldop{&arrayIndex{idx}}, ops...)
				node = op.Field
			case "RecordFieldRead":
				ops = append([]fieldop{&fieldRead{op.Param}}, ops...)
				node = op.Field
			default:
				return nil, fmt.Errorf("unknown FieldCall: %s", op.Fn)
			}
		default:
			return nil, errors.New("filter AST unknown field op")
		}
	}

	// Here's the actual resolver: grab the top-level field and then
	// apply any additional operations.
	return func(r *zson.Record) (zeek.Type, []byte) {
		col, ok := r.Descriptor.LUT[field]
		if !ok {
			// original field doesn't exist
			return nil, nil
		}
		typ := r.TypeOfColumn(col)
		val := r.Slice(col)
		for _, op := range(ops) {
			typ, val = op.apply(typ, val)
			if typ == nil {
				return nil, nil
			}
		}
		return typ, val
	}, nil
}

func combine(res ValResolver, pred zeek.Predicate) Filter {
	return func(r *zson.Record) bool {
		typ, val := res(r)
		if val == nil {
			// field (or sub-field) doesn't exist in this record
			return false

		}
		return pred(typ, val)
	}
}

func CompileFieldCompare(node ast.CompareField, val zeek.Value) (Filter, error) {
	// Treat len(field) specially since we're looking at a computed
	// value rather than a field from a record.
	if op, ok := node.Field.(*ast.FieldCall); ok && op.Fn == "Len" {
		i, ok := val.(*zeek.Int)
		if !ok {
			return nil, errors.New("cannot compare len() with non-integer")
		}
		comparison, err := i.NativeComparison(node.Comparator)
		if err != nil {
			return nil, err
		}
		checklen := func(typ zeek.Type, val []byte) bool {
			len, err := zeek.ContainerLength(typ, val)
			if err != nil {
				return false
			}
			return comparison(int64(len))
		}
		resolver, err := CompileFieldExpr(op.Field)
		if err != nil {
			return nil, err
		}
		return combine(resolver, checklen), nil
	}

	comparison, err := val.Comparison(node.Comparator)
	if err != nil {
		return nil, err
	}
	resolver, err := CompileFieldExpr(node.Field)
	if err != nil {
		return nil, err
	}
	return combine(resolver, comparison), nil
}

func EvalAny(eval zeek.Predicate) Filter {
	return func(p *zson.Record) bool {
		it := p.ZvalIter()
		for _, c := range p.Type.Columns {
			val, _, err := it.Next()
			if err != nil {
				return false
			}
			if eval(c.Type, val) {
				return true
			}
		}
		return false
	}
}

func Compile(node ast.BooleanExpr) (Filter, error) {
	switch v := node.(type) {
	case *ast.LogicalNot:
		expr, err := Compile(v.Expr)
		if err != nil {
			return nil, err
		}
		return LogicalNot(expr), nil

	case *ast.LogicalAnd:
		left, err := Compile(v.Left)
		if err != nil {
			return nil, err
		}
		right, err := Compile(v.Right)
		if err != nil {
			return nil, err
		}
		return LogicalAnd(left, right), nil

	case *ast.LogicalOr:
		left, err := Compile(v.Left)
		if err != nil {
			return nil, err
		}
		right, err := Compile(v.Right)
		if err != nil {
			return nil, err
		}
		return LogicalOr(left, right), nil

	case *ast.BooleanLiteral:
		return func(p *zson.Record) bool { return v.Value }, nil

	case *ast.SearchString:
		val := v.Value
		if val.Type != "string" {
			return nil, errors.New("SearchString value must be of type string")
		}
		return SearchString(val.Value), nil

	case *ast.CompareField:
		z, err := zeek.Parse(v.Value)
		if err != nil {
			return nil, err
		}

		if v.Comparator == "in" {
			resolver, err := CompileFieldExpr(v.Field)
			if err != nil {
				return nil, err
			}
			eql, _ := z.Comparison("eql")
			comparison := zeek.Contains(eql)
			return combine(resolver, comparison), nil
		}

		return CompileFieldCompare(*v, z)

	case *ast.CompareAny:
		z, err := zeek.Parse(v.Value)
		if err != nil {
			return nil, err
		}

		if v.Comparator == "in" {
			eql, _ := z.Comparison("eql")
			comparison := zeek.Contains(eql)
			return EvalAny(comparison), nil
		}

		comparison, err := z.Comparison(v.Comparator)
		if err != nil {
			return nil, err
		}
		return EvalAny(comparison), nil

	default:
		return nil, fmt.Errorf("Filter AST unknown type: %v", v)
	}
}

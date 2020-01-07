package expr

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/mccanne/zq/ast"
	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zng"
)

// A FieldExprResolver is a compiled FieldExpr (where FieldExpr is the
// abstract type representing various zql ast nodes).  This can be an
// expression as simple as "fieldname" or something more complex such as
// "len(vec[2].fieldname.subfieldname)".  A FieldExpr is compiled into a
// function that takes a zbuf.Record as input, evaluates the given
// expression against that record, and returns the resulting typed value.
// If the expression can't be resolved (i.e., because some field
// reference refers to a non-existent field, a vector index is out of
// bounds, etc.), the resolver returns (nil, nil)
type FieldExprResolver func(*zbuf.Record) zng.TypedEncoding

// fieldop, arrayIndex, and fieldRead are helpers used internally
// by CompileFieldExpr() below.
type fieldop interface {
	apply(zng.TypedEncoding) zng.TypedEncoding
}

type arrayIndex struct {
	idx int64
}

func (ai *arrayIndex) apply(e zng.TypedEncoding) zng.TypedEncoding {
	el, err := e.VectorIndex(ai.idx)
	if err != nil {
		if err == zng.ErrIndex {
			typ := zng.InnerType(e.Type)
			return zng.TypedEncoding{typ, nil}
		}
		return zng.TypedEncoding{}
	}
	return el
}

type fieldRead struct {
	field string
}

func (fr *fieldRead) apply(e zng.TypedEncoding) zng.TypedEncoding {
	recType, ok := e.Type.(*zng.TypeRecord)
	if !ok {
		// field reference on non-record type
		return zng.TypedEncoding{}
	}

	// XXX searching the list of columns for every record is
	// expensive, but we can receive records with different
	// types so caching this isn't straightforward.
	for n, col := range recType.Columns {
		if col.Name == fr.field {
			var v []byte
			it := e.Iter()
			for i := 0; i <= n; i++ {
				if it.Done() {
					return zng.TypedEncoding{}
				}
				var err error
				v, _, err = it.Next()
				if err != nil {
					return zng.TypedEncoding{}
				}
			}
			return zng.TypedEncoding{col.Type, v}
		}
	}
	// record doesn't have the named field
	return zng.TypedEncoding{}
}

// CompileFieldExpr() takes a FieldExpr AST (which represents either a
// simple field reference like "fieldname" or something more complex
// like "fieldname[0].subfield.subsubfield[3]") and compiles it into a
// ValResolver -- a function that takes a zbuf.Record and extracts the
// value to which the FieldExpr refers.  If the FieldExpr cannot be
// compiled, this function returns an error.  If the resolver is given
// a record for which the given expression cannot be evaluated (e.g.,
// if the record doesn't have a requested field or an array index is
// out of bounds), the resolver returns (nil, nil).
func CompileFieldExpr(node ast.FieldExpr) (FieldExprResolver, error) {
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
	return func(r *zbuf.Record) zng.TypedEncoding {
		col, ok := r.Descriptor.LUT[field]
		if !ok {
			// original field doesn't exist
			return zng.TypedEncoding{}
		}
		e := r.TypedSlice(col)
		for _, op := range ops {
			e = op.apply(e)
			if e.Type == nil {
				break
			}
		}
		return e
	}, nil
}

func CompileFieldExprArray(nodes []ast.FieldExpr) ([]FieldExprResolver, error) {
	var resolvers []FieldExprResolver
	if nodes != nil {
		resolvers = make([]FieldExprResolver, 0, len(nodes))
		for _, exp := range nodes {
			res, err := CompileFieldExpr(exp)
			if err != nil {
				return nil, err
			}
			resolvers = append(resolvers, res)
		}
	}
	return resolvers, nil
}

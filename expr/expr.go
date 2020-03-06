package expr

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/zng"
)

type ExpressionEvaluator func(*zng.Record) (zng.Value, error)

var ErrIncompatibleTypes = errors.New("incompatible types")

// CompileExpr tries to compile the given Expression into a function
// that evalutes the expression against a provided Record.  Returns an
// error if compilation fails for any reason.
//
// XXX this implementation isn't particularly efficient with respect
// to either memory use or runtime performance.  If we needed to optimize
// this it should be simple to compile expressions to a simple stack-based
// byte code that could evaluate expressions efficiently.
func CompileExpr(node ast.Expression) (ExpressionEvaluator, error) {
	switch n := node.(type) {
	case *ast.Literal:
		v, err := zng.Parse(*n)
		if err != nil {
			return nil, err
		}
		return func(*zng.Record) (zng.Value, error) { return v, nil }, nil

	case *ast.FieldRead:
		fn, err := CompileFieldExpr(n)
		if err != nil {
			return nil, err
		}
		return func(r *zng.Record) (zng.Value, error) { return fn(r), nil }, nil

	case *ast.BinaryExpression:
		switch n.Operator {
		case "AND", "OR", "=", "!=":
			return nil, errors.New("not implemented")
		case "<", "<=", ">", ">=":
			return nil, errors.New("not implemented")
		case "+":
			return CompileAddition(*n)
		case "-":
			return nil, errors.New("not implemented")
		case "*":
			return nil, errors.New("not implemented")
		case "/":
			return nil, errors.New("not implemented")
		default:
			return nil, fmt.Errorf("invalid binary operator %s", n.Operator)
		}

	default:
		return nil, fmt.Errorf("invalid expression type %T", node)
	}
}

func intWidth(t zng.Type) uint {
	switch t.ID() {
	case zng.IdByte:
		return 1
	case zng.IdInt16, zng.IdUint16:
		return 2
	case zng.IdInt32, zng.IdUint32:
		return 4
	case zng.IdInt64, zng.IdUint64:
		return 8
	default:
		return 0
	}
}

// CompileAddition compiles an expression of the form "expr1 + expr2".
// The complexity here all has to do with handling inputs of varying
// types.
func CompileAddition(node ast.BinaryExpression) (ExpressionEvaluator, error) {
	lhsFunc, err := CompileExpr(node.LHS)
	if err != nil {
		return nil, err
	}
	rhsFunc, err := CompileExpr(node.RHS)
	if err != nil {
		return nil, err
	}
	return func(rec *zng.Record) (zng.Value, error) {
		lhs, err := lhsFunc(rec)
		if err != nil {
			return zng.Value{}, err
		}
		rhs, err := rhsFunc(rec)
		if err != nil {
			return zng.Value{}, err
		}

		switch lhs.Type.ID() {
		case zng.IdUint16, zng.IdUint32, zng.IdUint64:
			width := intWidth(lhs.Type)
			var v uint64

			switch rhs.Type.ID() {
			case zng.IdByte:
				var b byte
				b, err = zng.DecodeByte(rhs.Bytes)
				if err != nil {
					return zng.Value{}, err
				}
				v = uint64(b)

			case zng.IdUint16, zng.IdUint32, zng.IdUint64:
				v, err = zng.DecodeUint(rhs.Bytes)
				if err != nil {
					return zng.Value{}, err
				}
				rhsWidth := intWidth(rhs.Type)
				if rhsWidth > width {
					width = rhsWidth
				}

			default:
				return zng.Value{}, ErrIncompatibleTypes
			}

			var v2 uint64
			v2, err = zng.DecodeUint(lhs.Bytes)
			if err != nil {
				return zng.Value{}, err
			}
			v += v2

			switch width {
			case 2:
				out := uint16(v)
				return zng.Value{zng.TypeUint16, zng.EncodeUint(uint64(out))}, nil
			case 4:
				out := uint32(v)
				return zng.Value{zng.TypeUint32, zng.EncodeUint(uint64(out))}, nil
			case 8:
				return zng.Value{zng.TypeUint64, zng.EncodeUint(v)}, nil

			default:
				panic("internal error in CompileAddition")
			}

		case zng.IdInt16, zng.IdInt32, zng.IdInt64:
			width := intWidth(lhs.Type)
			var v int64

			switch rhs.Type.ID() {
			case zng.IdByte:
				var b byte
				b, err = zng.DecodeByte(rhs.Bytes)
				if err != nil {
					return zng.Value{}, err
				}
				v = int64(b)

			case zng.IdInt16, zng.IdInt32, zng.IdInt64:
				v, err = zng.DecodeInt(rhs.Bytes)
				if err != nil {
					return zng.Value{}, err
				}
				rhsWidth := intWidth(rhs.Type)
				if rhsWidth > width {
					width = rhsWidth
				}

			default:
				return zng.Value{}, ErrIncompatibleTypes
			}

			var v2 int64
			v2, err = zng.DecodeInt(lhs.Bytes)
			if err != nil {
				return zng.Value{}, err
			}
			v += v2

			switch width {
			case 2:
				out := int16(v)
				return zng.Value{zng.TypeInt16, zng.EncodeInt(int64(out))}, nil
			case 4:
				out := int32(v)
				return zng.Value{zng.TypeInt32, zng.EncodeInt(int64(out))}, nil
			case 8:
				return zng.Value{zng.TypeInt64, zng.EncodeInt(v)}, nil
			default:
				panic("internal error in CompileAddition")
			}

			/*
				XXX need to complete these
				case zng.IdByte:
				case zng.IdFloat:
				case zng.IdString, zng.IdBstring:
			*/

		default:
			return zng.Value{}, ErrIncompatibleTypes
		}
	}, nil
}

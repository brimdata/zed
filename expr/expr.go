package expr

import (
	"errors"
	"fmt"
	"math"
	"net"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zngnative"
)

type ExpressionEvaluator func(*zng.Record) (zng.Value, error)

var ErrNoSuchField = errors.New("field is not present")
var ErrIncompatibleTypes = errors.New("incompatible types")
var ErrIndexOutOfBounds = errors.New("array index out of bounds")
var ErrNoSuchFunction = errors.New("no such function")

type NativeEvaluator func(*zng.Record) (zngnative.Value, error)

// CompileExpr tries to compile the given Expression into a function
// that evalutes the expression against a provided Record.  Returns an
// error if compilation fails for any reason.
//
// This is currently not particularly optimized -- it creates a bunch
// of closures and every evaluation involves some allocations.
// Eventually, we could optimize this by compiling a particular
// Expression for a particular TypeRecord into a series of byte codes
// that could be implemented by a simple stack-based evaluator much
// more efficiently.  ZNG unions are a challenge for this approach, but
// we could fail back to the "slow path" implemented here if an
// expression ever touches a union.
func CompileExpr(node ast.Expression) (ExpressionEvaluator, error) {
	ne, err := compileNative(node)
	if err != nil {
		return nil, err
	}

	return func(rec *zng.Record) (zng.Value, error) {
		nv, err := ne(rec)
		if err != nil {
			return zng.Value{}, err
		}

		return nv.ToZngValue()
	}, nil
}

func compileNative(node ast.Expression) (NativeEvaluator, error) {
	switch n := node.(type) {
	case *ast.Literal:
		v, err := zng.Parse(*n)
		if err != nil {
			return nil, err
		}
		nv, err := zngnative.ToNativeValue(v)
		if err != nil {
			return nil, err
		}
		return func(*zng.Record) (zngnative.Value, error) { return nv, nil }, nil

	case *ast.FieldRead:
		fn, err := CompileFieldExpr(n)
		if err != nil {
			return nil, err
		}
		return func(r *zng.Record) (zngnative.Value, error) {
			v := fn(r)
			if v.Type == nil {
				return zngnative.Value{}, ErrNoSuchField
			}
			nv, err := zngnative.ToNativeValue(v)
			if err != nil {
				return zngnative.Value{}, fmt.Errorf("%s: %w", n.Field, err)
			}
			return nv, nil
		}, nil

	case *ast.UnaryExpression:
		return compileUnary(*n)

	case *ast.BinaryExpression:
		lhsFunc, err := compileNative(n.LHS)
		if err != nil {
			return nil, err
		}
		rhsFunc, err := compileNative(n.RHS)
		if err != nil {
			return nil, err
		}
		switch n.Operator {
		case "AND", "OR":
			return compileLogical(lhsFunc, rhsFunc, n.Operator)
		case "=", "!=":
			return compileCompareEquality(lhsFunc, rhsFunc, n.Operator)
		case "<", "<=", ">", ">=":
			return compileCompareRelative(lhsFunc, rhsFunc, n.Operator)
		case "+", "-", "*", "/":
			return compileArithmetic(lhsFunc, rhsFunc, n.Operator)
		case "[":
			return compileArrayIndex(lhsFunc, rhsFunc, n.Operator)
		case ".":
			return compileFieldReference(lhsFunc, rhsFunc, n.Operator)
		default:
			return nil, fmt.Errorf("invalid binary operator %s", n.Operator)
		}

	case *ast.ConditionalExpression:
		return compileConditional(*n)

	case *ast.FunctionCall:
		return compileFunctionCall(*n)

	default:
		return nil, fmt.Errorf("invalid expression type %T", node)
	}
}

func compileUnary(node ast.UnaryExpression) (NativeEvaluator, error) {
	if node.Operator != "!" {
		return nil, fmt.Errorf("unknown unary operator %s\n", node.Operator)
	}
	fn, err := compileNative(node.Operand)
	if err != nil {
		return nil, err
	}
	return func(rec *zng.Record) (zngnative.Value, error) {
		val, err := fn(rec)
		if err != nil {
			return zngnative.Value{}, err
		}
		if val.Type.ID() != zng.IdBool {
			return zngnative.Value{}, ErrIncompatibleTypes
		}
		return zngnative.Value{zng.TypeBool, !(val.Value.(bool))}, nil
	}, nil
}

func compileLogical(lhsFunc, rhsFunc NativeEvaluator, operator string) (NativeEvaluator, error) {
	return func(rec *zng.Record) (zngnative.Value, error) {
		lhs, err := lhsFunc(rec)
		if err != nil {
			return zngnative.Value{}, err
		}
		if lhs.Type.ID() != zng.IdBool {
			return zngnative.Value{}, ErrIncompatibleTypes
		}

		lv := lhs.Value.(bool)
		switch operator {
		case "AND":
			if !lv {
				return lhs, nil
			}
		case "OR":
			if lv {
				return lhs, nil
			}
		default:
			panic("bad operator")
		}

		rhs, err := rhsFunc(rec)
		if err != nil {
			return zngnative.Value{}, err
		}
		if rhs.Type.ID() != zng.IdBool {
			return zngnative.Value{}, ErrIncompatibleTypes
		}

		return zngnative.Value{zng.TypeBool, rhs.Value.(bool)}, nil
	}, nil
}

func floatToInt64(f float64) (int64, bool) {
	i := int64(f)
	if float64(i) == f {
		return i, true
	}
	return 0, false
}

func floatToUint64(f float64) (uint64, bool) {
	u := uint64(f)
	if float64(u) == f {
		return u, true
	}
	return 0, false
}

func compileCompareEquality(lhsFunc, rhsFunc NativeEvaluator, operator string) (NativeEvaluator, error) {
	return func(rec *zng.Record) (zngnative.Value, error) {
		lhs, err := lhsFunc(rec)
		if err != nil {
			return zngnative.Value{}, err
		}
		rhs, err := rhsFunc(rec)
		if err != nil {
			return zngnative.Value{}, err
		}

		var equal bool
		switch lhs.Type.ID() {
		case zng.IdBool:
			if rhs.Type.ID() != zng.IdBool {
				return zngnative.Value{}, ErrIncompatibleTypes
			}
			equal = lhs.Value.(bool) == rhs.Value.(bool)

		case zng.IdInt16, zng.IdInt32, zng.IdInt64, zng.IdTime, zng.IdDuration:
			lv := lhs.Value.(int64)

			switch rhs.Type.ID() {
			case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64, zng.IdPort:
				if (lhs.Type.ID() == zng.IdTime || lhs.Type.ID() == zng.IdDuration) && rhs.Type.ID() == zng.IdPort {
					return zngnative.Value{}, ErrIncompatibleTypes
				}

				// Comparing a signed to an unsigned value.
				// Need to be careful not to find false
				// equality for two values with the same
				// bitwise representation...
				if lv < 0 {
					equal = false
				} else {
					equal = lv == int64(rhs.Value.(uint64))
				}
			case zng.IdInt16, zng.IdInt32, zng.IdInt64, zng.IdTime, zng.IdDuration:
				if (lhs.Type.ID() == zng.IdTime && rhs.Type.ID() == zng.IdDuration) || (lhs.Type.ID() == zng.IdDuration && rhs.Type.ID() == zng.IdTime) {
					return zngnative.Value{}, ErrIncompatibleTypes
				}

				// Simple comparison of two signed values
				equal = lv == rhs.Value.(int64)
			case zng.IdFloat64:
				rv, ok := floatToInt64(rhs.Value.(float64))
				if ok {
					equal = lv == int64(rv)
				} else {
					equal = false
				}
			default:
				return zngnative.Value{}, ErrIncompatibleTypes
			}

		case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64, zng.IdPort:
			lv := lhs.Value.(uint64)
			switch rhs.Type.ID() {
			case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64, zng.IdPort:
				// Simple comparison of two unsigned values
				equal = lv == rhs.Value.(uint64)
			case zng.IdInt16, zng.IdInt32, zng.IdInt64, zng.IdTime, zng.IdDuration:
				if lhs.Type.ID() == zng.IdPort && (rhs.Type.ID() == zng.IdTime || rhs.Type.ID() == zng.IdDuration) {
					return zngnative.Value{}, ErrIncompatibleTypes
				}
				// Comparing a signed to an unsigned value.
				// Need to be careful not to find false
				// equality for two values with the same
				// bitwise representation...
				rsigned := rhs.Value.(int64)
				if rsigned < 0 {
					equal = false
				} else {
					equal = lv == uint64(rsigned)
				}
			case zng.IdFloat64:
				rv, ok := floatToUint64(rhs.Value.(float64))
				if ok {
					equal = lv == uint64(rv)
				} else {
					equal = false
				}
			default:
				return zngnative.Value{}, ErrIncompatibleTypes
			}

		case zng.IdFloat64:
			lv := lhs.Value.(float64)
			switch rhs.Type.ID() {
			case zng.IdInt16, zng.IdInt32, zng.IdInt64, zng.IdTime, zng.IdDuration:
				var rv int64
				if rhs.Type.ID() == zng.IdTime {
					rv = int64(rhs.Value.(int64))
				} else {
					rv = rhs.Value.(int64)
				}
				equal = lv == float64(rv)
			case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64, zng.IdPort:
				equal = lv == float64(rhs.Value.(uint64))
			case zng.IdFloat64:
				equal = lv == rhs.Value.(float64)
			default:
				return zngnative.Value{}, ErrIncompatibleTypes
			}

		case zng.IdString, zng.IdBstring:
			if rhs.Type.ID() != zng.IdString && rhs.Type.ID() != zng.IdBstring {
				return zngnative.Value{}, ErrIncompatibleTypes
			}
			equal = lhs.Value.(string) == rhs.Value.(string)

		case zng.IdIP:
			if rhs.Type.ID() != zng.IdIP {
				return zngnative.Value{}, ErrIncompatibleTypes
			}
			equal = lhs.Value.(net.IP).Equal(rhs.Value.(net.IP))

		case zng.IdNet:
			if rhs.Type.ID() != zng.IdNet {
				return zngnative.Value{}, ErrIncompatibleTypes
			}
			// is there any other way to compare nets?
			equal = lhs.Value.(*net.IPNet).String() == rhs.Value.(*net.IPNet).String()

		default:
			return zngnative.Value{}, ErrIncompatibleTypes
		}

		switch operator {
		case "=":
			return zngnative.Value{zng.TypeBool, equal}, nil
		case "!=":
			return zngnative.Value{zng.TypeBool, !equal}, nil
		default:
			panic("bad operator")
		}
	}, nil
}

func compileCompareRelative(lhsFunc, rhsFunc NativeEvaluator, operator string) (NativeEvaluator, error) {
	return func(rec *zng.Record) (zngnative.Value, error) {
		lhs, err := lhsFunc(rec)
		if err != nil {
			return zngnative.Value{}, err
		}
		rhs, err := rhsFunc(rec)
		if err != nil {
			return zngnative.Value{}, err
		}

		// holds
		//   <0 if lhs < rhs
		//    0 if lhs == rhs
		//   >0 if lhs > rhs
		var result int
		switch lhs.Type.ID() {
		case zng.IdInt16, zng.IdInt32, zng.IdInt64, zng.IdTime, zng.IdDuration:
			lv := lhs.Value.(int64)
			var rv int64

			switch rhs.Type.ID() {
			case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64, zng.IdPort:
				if (lhs.Type.ID() == zng.IdTime || lhs.Type.ID() == zng.IdDuration) && rhs.Type.ID() == zng.IdPort {
					return zngnative.Value{}, ErrIncompatibleTypes
				}

				// signed/unsigned comparison
				runsigned := rhs.Value.(uint64)
				if lv < 0 {
					result = -1
					break
				} else if runsigned > math.MaxInt32 {
					result = 1
					break
				}
				rv = int64(runsigned)

			case zng.IdInt16, zng.IdInt32, zng.IdInt64, zng.IdTime, zng.IdDuration:
				if (lhs.Type.ID() == zng.IdTime && rhs.Type.ID() == zng.IdDuration) || (lhs.Type.ID() == zng.IdDuration && rhs.Type.ID() == zng.IdTime) {
					return zngnative.Value{}, ErrIncompatibleTypes
				}
				rv = rhs.Value.(int64)
			case zng.IdFloat64:
				lf := float64(lv)
				rf := rhs.Value.(float64)
				if lf < rf {
					result = -1
				} else if lf == rf {
					result = 0
				} else {
					result = 1
				}

			default:
				return zngnative.Value{}, ErrIncompatibleTypes
			}
			if lv < rv {
				result = -1
			} else if lv == rv {
				result = 0
			} else {
				result = 1
			}

		case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64, zng.IdPort:
			lv := lhs.Value.(uint64)
			var rv uint64
			switch rhs.Type.ID() {
			case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64, zng.IdPort:
				rv = rhs.Value.(uint64)

			case zng.IdInt16, zng.IdInt32, zng.IdInt64, zng.IdTime, zng.IdDuration:
				if lhs.Type.ID() == zng.IdPort && (rhs.Type.ID() == zng.IdTime || rhs.Type.ID() == zng.IdDuration) {
					return zngnative.Value{}, ErrIncompatibleTypes
				}
				rsigned := int64(rhs.Value.(int64))
				if rsigned < 0 {
					result = 1
					break
				} else if lv > math.MaxInt32 {
					result = -1
					break
				}
				rv = uint64(rsigned)
			case zng.IdFloat64:
				lf := float64(lv)
				rf := rhs.Value.(float64)
				if lf < rf {
					result = -1
				} else if lf == rf {
					result = 0
				} else {
					result = 1
				}

			default:
				return zngnative.Value{}, ErrIncompatibleTypes
			}
			if lv < rv {
				result = -1
			} else if lv == rv {
				result = 0
			} else {
				result = 1
			}

		case zng.IdFloat64:
			lv := lhs.Value.(float64)
			var rv float64
			switch rhs.Type.ID() {
			case zng.IdInt16, zng.IdInt32, zng.IdInt64:
				// XXX this can be lossy?
				rv = float64(rhs.Value.(int64))
			case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
				// XXX this can be lossy?
				rv = float64(rhs.Value.(uint64))
			case zng.IdFloat64:
				rv = rhs.Value.(float64)
			default:
				return zngnative.Value{}, ErrIncompatibleTypes
			}
			if lv < rv {
				result = -1
			} else if lv == rv {
				result = 0
			} else {
				result = 1
			}

		case zng.IdString, zng.IdBstring:
			if rhs.Type.ID() != zng.IdString && rhs.Type.ID() != zng.IdBstring {
				return zngnative.Value{}, ErrIncompatibleTypes
			}
			lv := lhs.Value.(string)
			rv := rhs.Value.(string)
			if lv < rv {
				result = -1
			} else if lv == rv {
				result = 0
			} else {
				result = 1
			}
		default:
			return zngnative.Value{}, ErrIncompatibleTypes
		}

		switch operator {
		case "<":
			return zngnative.Value{zng.TypeBool, result < 0}, nil
		case "<=":
			return zngnative.Value{zng.TypeBool, result <= 0}, nil
		case ">":
			return zngnative.Value{zng.TypeBool, result > 0}, nil
		case ">=":
			return zngnative.Value{zng.TypeBool, result >= 0}, nil
		default:
			panic("bad operator")
		}
	}, nil
}

// compileArithmetic compiles an expression of the form "expr1 op expr2"
// for the arithmetic operators +, -, *, /
func compileArithmetic(lhsFunc, rhsFunc NativeEvaluator, operator string) (NativeEvaluator, error) {
	return func(rec *zng.Record) (zngnative.Value, error) {
		lhs, err := lhsFunc(rec)
		if err != nil {
			return zngnative.Value{}, err
		}
		rhs, err := rhsFunc(rec)
		if err != nil {
			return zngnative.Value{}, err
		}

		switch lhs.Type.ID() {
		case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
			v := lhs.Value.(uint64)

			switch rhs.Type.ID() {
			case zng.IdInt16, zng.IdInt32, zng.IdInt64:
				if v > math.MaxInt64 {
					return zngnative.Value{}, ErrIncompatibleTypes
				}
				var r int64
				switch operator {
				case "+":
					r = int64(v) + rhs.Value.(int64)
				case "-":
					r = int64(v) - rhs.Value.(int64)
				case "*":
					r = int64(v) * rhs.Value.(int64)
				case "/":
					r = int64(v) / rhs.Value.(int64)
				default:
					panic("bad operator")
				}
				return zngnative.Value{zng.TypeInt64, r}, nil

			case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
				v2 := rhs.Value.(uint64)
				switch operator {
				case "+":
					v += v2
				case "-":
					v -= v2
				case "*":
					v *= v2
				case "/":
					v /= v2
				default:
					panic("bad operator")
				}
				return zngnative.Value{zng.TypeUint64, v}, nil

			case zng.IdFloat64:
				var r float64
				v2 := rhs.Value.(float64)
				switch operator {
				case "+":
					r = float64(v) + v2
				case "-":
					r = float64(v) - v2
				case "*":
					r = float64(v) * v2
				case "/":
					r = float64(v) / v2
				default:
					panic("bad operator")
				}
				return zngnative.Value{zng.TypeFloat64, r}, nil

			default:
				return zngnative.Value{}, ErrIncompatibleTypes
			}

		case zng.IdInt16, zng.IdInt32, zng.IdInt64:
			v := lhs.Value.(int64)

			switch rhs.Type.ID() {
			case zng.IdInt16, zng.IdInt32, zng.IdInt64:
				v2 := rhs.Value.(int64)
				switch operator {
				case "+":
					v += v2
				case "-":
					v -= v2
				case "*":
					v *= v2
				case "/":
					v /= v2
				default:
					panic("bad operator")
				}
				return zngnative.Value{zng.TypeInt64, v}, nil

			case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
				ru := rhs.Value.(uint64)
				if ru > math.MaxInt64 {
					return zngnative.Value{}, ErrIncompatibleTypes
				}
				switch operator {
				case "+":
					v += int64(ru)
				case "-":
					v -= int64(ru)
				case "*":
					v *= int64(ru)
				case "/":
					v /= int64(ru)
				default:
					panic("bad operator")
				}
				return zngnative.Value{zng.TypeInt64, v}, nil

			case zng.IdFloat64:
				var r float64
				v2 := rhs.Value.(float64)
				switch operator {
				case "+":
					r = float64(v) + v2
				case "-":
					r = float64(v) - v2
				case "*":
					r = float64(v) * v2
				case "/":
					r = float64(v) / v2
				default:
					panic("bad operator")
				}
				return zngnative.Value{zng.TypeFloat64, r}, nil

			default:
				return zngnative.Value{}, ErrIncompatibleTypes
			}

		case zng.IdFloat64:
			v := lhs.Value.(float64)
			var v2 float64

			switch rhs.Type.ID() {
			case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
				v2 = float64(rhs.Value.(uint64))

			case zng.IdInt16, zng.IdInt32, zng.IdInt64:
				v2 = float64(rhs.Value.(int64))

			case zng.IdFloat64:
				v2 = rhs.Value.(float64)

			default:
				return zngnative.Value{}, ErrIncompatibleTypes
			}

			switch operator {
			case "+":
				v += v2
			case "-":
				v -= v2
			case "*":
				v *= v2
			case "/":
				v /= v2
			default:
				panic("bad operator")
			}
			return zngnative.Value{zng.TypeFloat64, v}, nil

		case zng.IdString, zng.IdBstring:
			if operator != "+" {
				return zngnative.Value{}, ErrIncompatibleTypes
			}
			var t zng.Type
			t = zng.TypeBstring
			if lhs.Type.ID() == zng.IdString || rhs.Type.ID() == zng.IdString {
				t = zng.TypeString
			}
			return zngnative.Value{t, lhs.Value.(string) + rhs.Value.(string)}, nil

		case zng.IdTime:
			if rhs.Type.ID() != zng.IdDuration || (operator != "+" && operator != "-") {
				return zngnative.Value{}, ErrIncompatibleTypes
			}
			return zngnative.Value{zng.TypeTime, lhs.Value.(nano.Ts).Add(rhs.Value.(int64))}, nil

		default:
			return zngnative.Value{}, ErrIncompatibleTypes
		}
	}, nil
}

func getNthFromContainer(container zcode.Bytes, idx uint) (zcode.Bytes, error) {
	iter := zcode.Iter(container)
	var i uint = 0
	for ; !iter.Done(); i++ {
		zv, _, err := iter.Next()
		if err != nil {
			return nil, err
		}
		if i == idx {
			return zv, nil
		}
	}
	return nil, ErrIndexOutOfBounds
}

func compileArrayIndex(lhsFunc, rhsFunc NativeEvaluator, operator string) (NativeEvaluator, error) {
	return func(rec *zng.Record) (zngnative.Value, error) {
		lhs, err := lhsFunc(rec)
		if err != nil {
			return zngnative.Value{}, err
		}

		aType, ok := lhs.Type.(*zng.TypeArray)
		if !ok {
			return zngnative.Value{}, ErrIncompatibleTypes
		}

		rhs, err := rhsFunc(rec)
		if err != nil {
			return zngnative.Value{}, err
		}

		var idx uint
		switch rhs.Type.ID() {
		case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
			idx = uint(rhs.Value.(uint64))
		case zng.IdInt16, zng.IdInt32, zng.IdInt64:
			i := rhs.Value.(int64)
			if i < 0 {
				return zngnative.Value{}, ErrIndexOutOfBounds
			}
			idx = uint(i)
		default:
			return zngnative.Value{}, ErrIncompatibleTypes
		}

		zv, err := getNthFromContainer(lhs.Value.(zcode.Bytes), idx)
		if err != nil {
			return zngnative.Value{}, err
		}
		return zngnative.ToNativeValue(zng.Value{aType.Type, zv})
	}, nil
}

func compileFieldReference(lhsFunc, rhsFunc NativeEvaluator, operator string) (NativeEvaluator, error) {
	return func(rec *zng.Record) (zngnative.Value, error) {
		lhs, err := lhsFunc(rec)
		if err != nil {
			return zngnative.Value{}, err
		}

		var rType *zng.TypeRecord
		var ok bool
		if rType, ok = lhs.Type.(*zng.TypeRecord); !ok {
			return zngnative.Value{}, ErrIncompatibleTypes
		}

		rhs, err := rhsFunc(rec)
		if err != nil {
			return zngnative.Value{}, err
		}

		if rhs.Type.ID() != zng.IdString && rhs.Type.ID() != zng.IdBstring {
			return zngnative.Value{}, ErrIncompatibleTypes
		}

		idx, ok := rType.ColumnOfField(rhs.Value.(string))
		if !ok {
			return zngnative.Value{}, ErrNoSuchField
		}

		zv, err := getNthFromContainer(lhs.Value.(zcode.Bytes), uint(idx))
		if err != nil {
			return zngnative.Value{}, err
		}
		return zngnative.ToNativeValue(zng.Value{rType.Columns[idx].Type, zv})
	}, nil
}

func compileFunctionCall(node ast.FunctionCall) (NativeEvaluator, error) {
	fn, ok := allFns[node.Function]
	if !ok {
		return nil, fmt.Errorf("%s: %w", node.Function, ErrNoSuchFunction)
	}

	nargs := len(node.Args)
	if fn.minArgs >= 0 && nargs < fn.minArgs {
		return nil, fmt.Errorf("%s: %w", node.Function, ErrTooFewArgs)
	}
	if fn.maxArgs >= 0 && nargs > fn.maxArgs {
		return nil, fmt.Errorf("%s: %w", node.Function, ErrTooManyArgs)
	}

	exprs := make([]NativeEvaluator, nargs)
	for i, expr := range node.Args {
		eval, err := compileNative(expr)
		if err != nil {
			return nil, err
		}
		exprs[i] = eval
	}

	return func(r *zng.Record) (zngnative.Value, error) {
		args := make([]zngnative.Value, 0, nargs)
		for _, a := range exprs {
			val, err := a(r)
			if err != nil {
				return zngnative.Value{}, err
			}
			args = append(args, val)
		}

		return fn.impl(args)
	}, nil
}

func compileConditional(node ast.ConditionalExpression) (NativeEvaluator, error) {
	conditionFunc, err := compileNative(node.Condition)
	if err != nil {
		return nil, err
	}
	thenFunc, err := compileNative(node.Then)
	if err != nil {
		return nil, err
	}
	elseFunc, err := compileNative(node.Else)
	if err != nil {
		return nil, err
	}

	return func(r *zng.Record) (zngnative.Value, error) {
		condition, err := conditionFunc(r)
		if err != nil {
			return zngnative.Value{}, err
		}
		if condition.Type.ID() != zng.IdBool {
			return zngnative.Value{}, ErrIncompatibleTypes
		}
		if condition.Value.(bool) {
			return thenFunc(r)
		} else {
			return elseFunc(r)
		}
	}, nil
}

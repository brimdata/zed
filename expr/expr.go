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
)

type ExpressionEvaluator func(*zng.Record) (zng.Value, error)

var ErrNoSuchField = errors.New("field is not present")
var ErrIncompatibleTypes = errors.New("incompatible types")
var ErrIndexOutOfBounds = errors.New("array index out of bounds")

type NativeValue struct {
	typ   zng.Type
	value interface{}
}

type NativeEvaluator func(*zng.Record) (NativeValue, error)

func toNativeValue(zv zng.Value) (NativeValue, error) {
	switch zv.Type.ID() {
	case zng.IdBool:
		b, err := zng.DecodeBool(zv.Bytes)
		if err != nil {
			return NativeValue{}, err
		}
		return NativeValue{zng.TypeBool, b}, nil

	case zng.IdByte:
		b, err := zng.DecodeByte(zv.Bytes)
		if err != nil {
			return NativeValue{}, err
		}
		return NativeValue{zng.TypeByte, uint64(b)}, nil

	case zng.IdInt16, zng.IdInt32, zng.IdInt64:
		v, err := zng.DecodeInt(zv.Bytes)
		if err != nil {
			return NativeValue{}, err
		}
		return NativeValue{zv.Type, v}, nil

	case zng.IdUint16, zng.IdUint32, zng.IdUint64:
		v, err := zng.DecodeUint(zv.Bytes)
		if err != nil {
			return NativeValue{}, err
		}
		return NativeValue{zv.Type, v}, nil

	case zng.IdFloat64:
		v, err := zng.DecodeFloat64(zv.Bytes)
		if err != nil {
			return NativeValue{}, err
		}
		return NativeValue{zv.Type, v}, nil

	case zng.IdString:
		s, err := zng.DecodeString(zv.Bytes)
		if err != nil {
			return NativeValue{}, err
		}
		return NativeValue{zv.Type, s}, nil

	case zng.IdBstring:
		s, err := zng.DecodeBstring(zv.Bytes)
		if err != nil {
			return NativeValue{}, err
		}
		return NativeValue{zv.Type, s}, nil

	case zng.IdIP:
		a, err := zng.DecodeIP(zv.Bytes)
		if err != nil {
			return NativeValue{}, err
		}
		return NativeValue{zv.Type, a}, nil

	case zng.IdPort:
		p, err := zng.DecodePort(zv.Bytes)
		if err != nil {
			return NativeValue{}, err
		}
		return NativeValue{zv.Type, uint64(p)}, nil

	case zng.IdNet:
		n, err := zng.DecodeNet(zv.Bytes)
		if err != nil {
			return NativeValue{}, err
		}
		return NativeValue{zv.Type, n}, nil

	case zng.IdTime:
		t, err := zng.DecodeTime(zv.Bytes)
		if err != nil {
			return NativeValue{}, nil
		}
		return NativeValue{zv.Type, int64(t)}, nil

	case zng.IdDuration:
		d, err := zng.DecodeDuration(zv.Bytes)
		if err != nil {
			return NativeValue{}, nil
		}
		return NativeValue{zv.Type, d}, nil
	}

	// Keep arrays, sets, and records in their zval encoded form.
	// The purpose of NativeValue is to avoid encoding temporary
	// values but since we can't construct these types in expressions,
	// this just lets us lazily decode them.
	switch zv.Type.(type) {
	case *zng.TypeArray, *zng.TypeSet, *zng.TypeRecord:
		return NativeValue{zv.Type, zv.Bytes}, nil
	}

	return NativeValue{}, fmt.Errorf("unknown type %d", zv.Type.ID())
}

func (v *NativeValue) toZngValue() (zng.Value, error) {
	switch v.typ.ID() {
	case zng.IdBool:
		b := v.value.(bool)
		return zng.Value{zng.TypeBool, zng.EncodeBool(b)}, nil

	case zng.IdByte:
		b := v.value.(uint64)
		return zng.Value{zng.TypeByte, zng.EncodeByte(byte(b))}, nil

	case zng.IdInt16:
		i := v.value.(int64)
		return zng.Value{zng.TypeInt16, zng.EncodeInt(i)}, nil

	case zng.IdInt32:
		i := v.value.(int64)
		return zng.Value{zng.TypeInt32, zng.EncodeInt(i)}, nil

	case zng.IdInt64:
		i := v.value.(int64)
		return zng.Value{zng.TypeInt64, zng.EncodeInt(i)}, nil

	case zng.IdUint16:
		i := v.value.(uint64)
		return zng.Value{zng.TypeUint16, zng.EncodeUint(i)}, nil

	case zng.IdUint32:
		i := v.value.(uint64)
		return zng.Value{zng.TypeUint32, zng.EncodeUint(i)}, nil

	case zng.IdUint64:
		i := v.value.(uint64)
		return zng.Value{zng.TypeUint64, zng.EncodeUint(i)}, nil

	case zng.IdFloat64:
		f := v.value.(float64)
		return zng.Value{zng.TypeFloat64, zng.EncodeFloat64(f)}, nil

	case zng.IdString:
		s := v.value.(string)
		return zng.Value{zng.TypeString, zng.EncodeString(s)}, nil

	case zng.IdBstring:
		s := v.value.(string)
		return zng.Value{zng.TypeBstring, zng.EncodeBstring(s)}, nil

	case zng.IdIP:
		i := v.value.(net.IP)
		return zng.Value{zng.TypeIP, zng.EncodeIP(i)}, nil

	case zng.IdPort:
		p := v.value.(uint64)
		return zng.Value{zng.TypePort, zng.EncodePort(uint32(p))}, nil

	case zng.IdNet:
		n := v.value.(*net.IPNet)
		return zng.Value{zng.TypeNet, zng.EncodeNet(n)}, nil

	case zng.IdTime:
		t := nano.Ts(v.value.(int64))
		return zng.Value{zng.TypeTime, zng.EncodeTime(t)}, nil

	case zng.IdDuration:
		d := v.value.(int64)
		return zng.Value{zng.TypeDuration, zng.EncodeDuration(d)}, nil

	default:
		return zng.Value{}, errors.New("unknown type")
	}
}

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

		return nv.toZngValue()
	}, nil
}

func compileNative(node ast.Expression) (NativeEvaluator, error) {
	switch n := node.(type) {
	case *ast.Literal:
		v, err := zng.Parse(*n)
		if err != nil {
			return nil, err
		}
		nv, err := toNativeValue(v)
		if err != nil {
			return nil, err
		}
		return func(*zng.Record) (NativeValue, error) { return nv, nil }, nil

	case *ast.FieldRead:
		fn, err := CompileFieldExpr(n)
		if err != nil {
			return nil, err
		}
		return func(r *zng.Record) (NativeValue, error) {
			v := fn(r)
			if v.Type == nil {
				return NativeValue{}, ErrNoSuchField
			}
			return toNativeValue(v)
		}, nil

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

	default:
		return nil, fmt.Errorf("invalid expression type %T", node)
	}
}

func compileLogical(lhsFunc, rhsFunc NativeEvaluator, operator string) (NativeEvaluator, error) {
	return func(rec *zng.Record) (NativeValue, error) {
		lhs, err := lhsFunc(rec)
		if err != nil {
			return NativeValue{}, err
		}
		if lhs.typ.ID() != zng.IdBool {
			return NativeValue{}, ErrIncompatibleTypes
		}

		lv := lhs.value.(bool)
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
			return NativeValue{}, err
		}
		if rhs.typ.ID() != zng.IdBool {
			return NativeValue{}, ErrIncompatibleTypes
		}

		return NativeValue{zng.TypeBool, rhs.value.(bool)}, nil
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
	return func(rec *zng.Record) (NativeValue, error) {
		lhs, err := lhsFunc(rec)
		if err != nil {
			return NativeValue{}, err
		}
		rhs, err := rhsFunc(rec)
		if err != nil {
			return NativeValue{}, err
		}

		var equal bool
		switch lhs.typ.ID() {
		case zng.IdBool:
			if rhs.typ.ID() != zng.IdBool {
				return NativeValue{}, ErrIncompatibleTypes
			}
			equal = lhs.value.(bool) == rhs.value.(bool)

		case zng.IdInt16, zng.IdInt32, zng.IdInt64, zng.IdTime, zng.IdDuration:
			lv := lhs.value.(int64)

			switch rhs.typ.ID() {
			case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64, zng.IdPort:
				if (lhs.typ.ID() == zng.IdTime || lhs.typ.ID() == zng.IdDuration) && rhs.typ.ID() == zng.IdPort {
					return NativeValue{}, ErrIncompatibleTypes
				}

				// Comparing a signed to an unsigned value.
				// Need to be careful not to find false
				// equality for two values with the same
				// bitwise representation...
				if lv < 0 {
					equal = false
				} else {
					equal = lv == int64(rhs.value.(uint64))
				}
			case zng.IdInt16, zng.IdInt32, zng.IdInt64, zng.IdTime, zng.IdDuration:
				if (lhs.typ.ID() == zng.IdTime && rhs.typ.ID() == zng.IdDuration) || (lhs.typ.ID() == zng.IdDuration && rhs.typ.ID() == zng.IdTime) {
					return NativeValue{}, ErrIncompatibleTypes
				}

				// Simple comparison of two signed values
				equal = lv == rhs.value.(int64)
			case zng.IdFloat64:
				rv, ok := floatToInt64(rhs.value.(float64))
				if ok {
					equal = lv == int64(rv)
				} else {
					equal = false
				}
			default:
				return NativeValue{}, ErrIncompatibleTypes
			}

		case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64, zng.IdPort:
			lv := lhs.value.(uint64)
			switch rhs.typ.ID() {
			case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64, zng.IdPort:
				// Simple comparison of two unsigned values
				equal = lv == rhs.value.(uint64)
			case zng.IdInt16, zng.IdInt32, zng.IdInt64, zng.IdTime, zng.IdDuration:
				if lhs.typ.ID() == zng.IdPort && (rhs.typ.ID() == zng.IdTime || rhs.typ.ID() == zng.IdDuration) {
					return NativeValue{}, ErrIncompatibleTypes
				}
				// Comparing a signed to an unsigned value.
				// Need to be careful not to find false
				// equality for two values with the same
				// bitwise representation...
				rsigned := rhs.value.(int64)
				if rsigned < 0 {
					equal = false
				} else {
					equal = lv == uint64(rsigned)
				}
			case zng.IdFloat64:
				rv, ok := floatToUint64(rhs.value.(float64))
				if ok {
					equal = lv == uint64(rv)
				} else {
					equal = false
				}
			default:
				return NativeValue{}, ErrIncompatibleTypes
			}

		case zng.IdFloat64:
			lv := lhs.value.(float64)
			switch rhs.typ.ID() {
			case zng.IdInt16, zng.IdInt32, zng.IdInt64, zng.IdTime, zng.IdDuration:
				var rv int64
				if rhs.typ.ID() == zng.IdTime {
					rv = int64(rhs.value.(int64))
				} else {
					rv = rhs.value.(int64)
				}
				equal = lv == float64(rv)
			case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64, zng.IdPort:
				equal = lv == float64(rhs.value.(uint64))
			case zng.IdFloat64:
				equal = lv == rhs.value.(float64)
			default:
				return NativeValue{}, ErrIncompatibleTypes
			}

		case zng.IdString, zng.IdBstring:
			if rhs.typ.ID() != zng.IdString && rhs.typ.ID() != zng.IdBstring {
				return NativeValue{}, ErrIncompatibleTypes
			}
			equal = lhs.value.(string) == rhs.value.(string)

		case zng.IdIP:
			if rhs.typ.ID() != zng.IdIP {
				return NativeValue{}, ErrIncompatibleTypes
			}
			equal = lhs.value.(net.IP).Equal(rhs.value.(net.IP))

		case zng.IdNet:
			if rhs.typ.ID() != zng.IdNet {
				return NativeValue{}, ErrIncompatibleTypes
			}
			// is there any other way to compare nets?
			equal = lhs.value.(*net.IPNet).String() == rhs.value.(*net.IPNet).String()

		default:
			return NativeValue{}, ErrIncompatibleTypes
		}

		switch operator {
		case "=":
			return NativeValue{zng.TypeBool, equal}, nil
		case "!=":
			return NativeValue{zng.TypeBool, !equal}, nil
		default:
			panic("bad operator")
		}
	}, nil
}

func compileCompareRelative(lhsFunc, rhsFunc NativeEvaluator, operator string) (NativeEvaluator, error) {
	return func(rec *zng.Record) (NativeValue, error) {
		lhs, err := lhsFunc(rec)
		if err != nil {
			return NativeValue{}, err
		}
		rhs, err := rhsFunc(rec)
		if err != nil {
			return NativeValue{}, err
		}

		// holds
		//   <0 if lhs < rhs
		//    0 if lhs == rhs
		//   >0 if lhs > rhs
		var result int
		switch lhs.typ.ID() {
		case zng.IdInt16, zng.IdInt32, zng.IdInt64, zng.IdTime, zng.IdDuration:
			lv := lhs.value.(int64)
			var rv int64

			switch rhs.typ.ID() {
			case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64, zng.IdPort:
				if (lhs.typ.ID() == zng.IdTime || lhs.typ.ID() == zng.IdDuration) && rhs.typ.ID() == zng.IdPort {
					return NativeValue{}, ErrIncompatibleTypes
				}

				// signed/unsigned comparison
				runsigned := rhs.value.(uint64)
				if lv < 0 {
					result = -1
					break
				} else if runsigned > math.MaxInt32 {
					result = 1
					break
				}
				rv = int64(runsigned)

			case zng.IdInt16, zng.IdInt32, zng.IdInt64, zng.IdTime, zng.IdDuration:
				if (lhs.typ.ID() == zng.IdTime && rhs.typ.ID() == zng.IdDuration) || (lhs.typ.ID() == zng.IdDuration && rhs.typ.ID() == zng.IdTime) {
					return NativeValue{}, ErrIncompatibleTypes
				}
				rv = rhs.value.(int64)
			case zng.IdFloat64:
				lf := float64(lv)
				rf := rhs.value.(float64)
				if lf < rf {
					result = -1
				} else if lf == rf {
					result = 0
				} else {
					result = 1
				}
				break

			default:
				return NativeValue{}, ErrIncompatibleTypes
			}
			if lv < rv {
				result = -1
			} else if lv == rv {
				result = 0
			} else {
				result = 1
			}

		case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64, zng.IdPort:
			lv := lhs.value.(uint64)
			var rv uint64
			switch rhs.typ.ID() {
			case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64, zng.IdPort:
				rv = rhs.value.(uint64)

			case zng.IdInt16, zng.IdInt32, zng.IdInt64, zng.IdTime, zng.IdDuration:
				if lhs.typ.ID() == zng.IdPort && (rhs.typ.ID() == zng.IdTime || rhs.typ.ID() == zng.IdDuration) {
					return NativeValue{}, ErrIncompatibleTypes
				}
				rsigned := int64(rhs.value.(int64))
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
				rf := rhs.value.(float64)
				if lf < rf {
					result = -1
				} else if lf == rf {
					result = 0
				} else {
					result = 1
				}
				break

			default:
				return NativeValue{}, ErrIncompatibleTypes
			}
			if lv < rv {
				result = -1
			} else if lv == rv {
				result = 0
			} else {
				result = 1
			}

		case zng.IdFloat64:
			lv := lhs.value.(float64)
			var rv float64
			switch rhs.typ.ID() {
			case zng.IdInt16, zng.IdInt32, zng.IdInt64:
				// XXX this can be lossy?
				rv = float64(rhs.value.(int64))
			case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
				// XXX this can be lossy?
				rv = float64(rhs.value.(uint64))
			case zng.IdFloat64:
				rv = rhs.value.(float64)
			default:
				return NativeValue{}, ErrIncompatibleTypes
			}
			if lv < rv {
				result = -1
			} else if lv == rv {
				result = 0
			} else {
				result = 1
			}

		case zng.IdString, zng.IdBstring:
			if rhs.typ.ID() != zng.IdString && rhs.typ.ID() != zng.IdBstring {
				return NativeValue{}, ErrIncompatibleTypes
			}
			lv := lhs.value.(string)
			rv := rhs.value.(string)
			if lv < rv {
				result = -1
			} else if lv == rv {
				result = 0
			} else {
				result = 1
			}
		default:
			return NativeValue{}, ErrIncompatibleTypes
		}

		switch operator {
		case "<":
			return NativeValue{zng.TypeBool, result < 0}, nil
		case "<=":
			return NativeValue{zng.TypeBool, result <= 0}, nil
		case ">":
			return NativeValue{zng.TypeBool, result > 0}, nil
		case ">=":
			return NativeValue{zng.TypeBool, result >= 0}, nil
		default:
			panic("bad operator")
		}
	}, nil
}

// compileArithmetic compiles an expression of the form "expr1 op expr2"
// for the arithmetic operators +, -, *, /
func compileArithmetic(lhsFunc, rhsFunc NativeEvaluator, operator string) (NativeEvaluator, error) {
	return func(rec *zng.Record) (NativeValue, error) {
		lhs, err := lhsFunc(rec)
		if err != nil {
			return NativeValue{}, err
		}
		rhs, err := rhsFunc(rec)
		if err != nil {
			return NativeValue{}, err
		}

		switch lhs.typ.ID() {
		case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
			v := lhs.value.(uint64)

			switch rhs.typ.ID() {
			case zng.IdInt16, zng.IdInt32, zng.IdInt64:
				if v > math.MaxInt64 {
					return NativeValue{}, ErrIncompatibleTypes
				}
				var r int64
				switch operator {
				case "+":
					r = int64(v) + rhs.value.(int64)
				case "-":
					r = int64(v) - rhs.value.(int64)
				case "*":
					r = int64(v) * rhs.value.(int64)
				case "/":
					r = int64(v) / rhs.value.(int64)
				default:
					panic("bad operator")
				}
				return NativeValue{zng.TypeInt64, r}, nil

			case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
				v2 := rhs.value.(uint64)
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
				return NativeValue{zng.TypeUint64, v}, nil

			case zng.IdFloat64:
				var r float64
				v2 := rhs.value.(float64)
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
				return NativeValue{zng.TypeFloat64, r}, nil

			default:
				return NativeValue{}, ErrIncompatibleTypes
			}

		case zng.IdInt16, zng.IdInt32, zng.IdInt64:
			v := lhs.value.(int64)

			switch rhs.typ.ID() {
			case zng.IdInt16, zng.IdInt32, zng.IdInt64:
				v2 := rhs.value.(int64)
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
				return NativeValue{zng.TypeInt64, v}, nil

			case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
				ru := rhs.value.(uint64)
				if ru > math.MaxInt64 {
					return NativeValue{}, ErrIncompatibleTypes
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
				return NativeValue{zng.TypeInt64, v}, nil

			case zng.IdFloat64:
				var r float64
				v2 := rhs.value.(float64)
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
				return NativeValue{zng.TypeFloat64, r}, nil

			default:
				return NativeValue{}, ErrIncompatibleTypes
			}

		case zng.IdFloat64:
			v := lhs.value.(float64)
			var v2 float64

			switch rhs.typ.ID() {
			case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
				v2 = float64(rhs.value.(uint64))

			case zng.IdInt16, zng.IdInt32, zng.IdInt64:
				v2 = float64(rhs.value.(int64))

			case zng.IdFloat64:
				v2 = rhs.value.(float64)

			default:
				return NativeValue{}, ErrIncompatibleTypes
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
			return NativeValue{zng.TypeFloat64, v}, nil

		case zng.IdString, zng.IdBstring:
			if operator != "+" {
				return NativeValue{}, ErrIncompatibleTypes
			}
			var t zng.Type
			t = zng.TypeBstring
			if lhs.typ.ID() == zng.IdString || rhs.typ.ID() == zng.IdString {
				t = zng.TypeString
			}
			return NativeValue{t, lhs.value.(string) + rhs.value.(string)}, nil

		case zng.IdTime:
			if rhs.typ.ID() != zng.IdDuration || (operator != "+" && operator != "-") {
				return NativeValue{}, ErrIncompatibleTypes
			}
			return NativeValue{zng.TypeTime, lhs.value.(nano.Ts).Add(rhs.value.(int64))}, nil

		default:
			return NativeValue{}, ErrIncompatibleTypes
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
	return func(rec *zng.Record) (NativeValue, error) {
		lhs, err := lhsFunc(rec)
		if err != nil {
			return NativeValue{}, err
		}

		var aType *zng.TypeArray
		var ok bool
		if aType, ok = lhs.typ.(*zng.TypeArray); !ok {
			return NativeValue{}, ErrIncompatibleTypes
		}

		rhs, err := rhsFunc(rec)
		if err != nil {
			return NativeValue{}, err
		}

		var idx uint
		switch rhs.typ.ID() {
		case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
			idx = uint(rhs.value.(uint64))
		case zng.IdInt16, zng.IdInt32, zng.IdInt64:
			i := rhs.value.(int64)
			if i < 0 {
				return NativeValue{}, ErrIndexOutOfBounds
			}
			idx = uint(i)
		default:
			return NativeValue{}, ErrIncompatibleTypes
		}

		zv, err := getNthFromContainer(lhs.value.(zcode.Bytes), idx)
		if err != nil {
			return NativeValue{}, err
		}
		return toNativeValue(zng.Value{aType.Type, zv})
	}, nil
}

func compileFieldReference(lhsFunc, rhsFunc NativeEvaluator, operator string) (NativeEvaluator, error) {
	return func(rec *zng.Record) (NativeValue, error) {
		lhs, err := lhsFunc(rec)
		if err != nil {
			return NativeValue{}, err
		}

		var rType *zng.TypeRecord
		var ok bool
		if rType, ok = lhs.typ.(*zng.TypeRecord); !ok {
			return NativeValue{}, ErrIncompatibleTypes
		}

		rhs, err := rhsFunc(rec)
		if err != nil {
			return NativeValue{}, err
		}

		if rhs.typ.ID() != zng.IdString && rhs.typ.ID() != zng.IdBstring {
			return NativeValue{}, ErrIncompatibleTypes
		}

		idx, ok := rType.ColumnOfField(rhs.value.(string))
		if !ok {
			return NativeValue{}, ErrNoSuchField
		}

		zv, err := getNthFromContainer(lhs.value.(zcode.Bytes), uint(idx))
		if err != nil {
			return NativeValue{}, err
		}
		return toNativeValue(zng.Value{rType.Columns[idx].Type, zv})
	}, nil
}

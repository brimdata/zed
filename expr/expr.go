package expr

import (
	"errors"
	"fmt"
	"net"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zng"
)

type ExpressionEvaluator func(*zng.Record) (zng.Value, error)

var ErrNoSuchField = errors.New("field is not present")
var ErrIncompatibleTypes = errors.New("incompatible types")

type NativeValue struct {
	typ   int
	value interface{}
}

type NativeEvaluator func(*zng.Record) (NativeValue, error)

// CompileExpr tries to compile the given Expression into a function
// that evalutes the expression against a provided Record.  Returns an
// error if compilation fails for any reason.
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

func toNativeValue(zv zng.Value) (NativeValue, error) {
	switch zv.Type.ID() {
	case zng.IdBool:
		b, err := zng.DecodeBool(zv.Bytes)
		if err != nil {
			return NativeValue{}, err
		}
		return NativeValue{zng.IdBool, b}, nil

	case zng.IdByte:
		b, err := zng.DecodeByte(zv.Bytes)
		if err != nil {
			return NativeValue{}, err
		}
		return NativeValue{zng.IdByte, uint64(b)}, nil

	case zng.IdInt16, zng.IdInt32, zng.IdInt64:
		v, err := zng.DecodeInt(zv.Bytes)
		if err != nil {
			return NativeValue{}, err
		}
		return NativeValue{zv.Type.ID(), v}, nil

	case zng.IdUint16, zng.IdUint32, zng.IdUint64:
		v, err := zng.DecodeUint(zv.Bytes)
		if err != nil {
			return NativeValue{}, err
		}
		return NativeValue{zv.Type.ID(), v}, nil

	case zng.IdFloat64:
		v, err := zng.DecodeFloat64(zv.Bytes)
		if err != nil {
			return NativeValue{}, err
		}
		return NativeValue{zv.Type.ID(), v}, nil

	case zng.IdString:
		s, err := zng.DecodeString(zv.Bytes)
		if err != nil {
			return NativeValue{}, err
		}
		return NativeValue{zv.Type.ID(), s}, nil

	case zng.IdBstring:
		s, err := zng.DecodeBstring(zv.Bytes)
		if err != nil {
			return NativeValue{}, err
		}
		return NativeValue{zv.Type.ID(), s}, nil

	case zng.IdIP:
		a, err := zng.DecodeIP(zv.Bytes)
		if err != nil {
			return NativeValue{}, err
		}
		return NativeValue{zv.Type.ID(), a}, nil

	case zng.IdPort:
		p, err := zng.DecodePort(zv.Bytes)
		if err != nil {
			return NativeValue{}, err
		}
		return NativeValue{zv.Type.ID(), uint64(p)}, nil

	case zng.IdNet:
		n, err := zng.DecodeNet(zv.Bytes)
		if err != nil {
			return NativeValue{}, err
		}
		return NativeValue{zv.Type.ID(), n}, nil

	case zng.IdTime:
		t, err := zng.DecodeTime(zv.Bytes)
		if err != nil {
			return NativeValue{}, nil
		}
		return NativeValue{zv.Type.ID(), t}, nil

	case zng.IdDuration:
		d, err := zng.DecodeDuration(zv.Bytes)
		if err != nil {
			return NativeValue{}, nil
		}
		return NativeValue{zv.Type.ID(), d}, nil

	default:
		return NativeValue{}, fmt.Errorf("unknown type %d", zv.Type.ID())
	}

}

func (v *NativeValue) toZngValue() (zng.Value, error) {
	switch v.typ {
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
		return zng.Value{zng.TypeInt16, zng.EncodeUint(i)}, nil

	case zng.IdUint32:
		i := v.value.(uint64)
		return zng.Value{zng.TypeInt32, zng.EncodeUint(i)}, nil

	case zng.IdUint64:
		i := v.value.(uint64)
		return zng.Value{zng.TypeInt64, zng.EncodeUint(i)}, nil

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
		t := v.value.(nano.Ts)
		return zng.Value{zng.TypeTime, zng.EncodeTime(t)}, nil

	case zng.IdDuration:
		d := v.value.(int64)
		return zng.Value{zng.TypeDuration, zng.EncodeDuration(d)}, nil

	default:
		return zng.Value{}, errors.New("XXX")
	}
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
		switch n.Operator {
		case "AND", "OR":
			return compileLogical(*n)
		case "=", "!=":
			return compileCompareEquality(*n)
		case "<", "<=", ">", ">=":
			return compileCompareRelative(*n)
		case "+", "-", "*", "/":
			return compileArithmetic(*n)
		default:
			return nil, fmt.Errorf("invalid binary operator %s", n.Operator)
		}

	default:
		return nil, fmt.Errorf("invalid expression type %T", node)
	}
}

func compileLogical(node ast.BinaryExpression) (NativeEvaluator, error) {
	lhsFunc, err := compileNative(node.LHS)
	if err != nil {
		return nil, err
	}
	rhsFunc, err := compileNative(node.RHS)
	if err != nil {
		return nil, err
	}
	return func(rec *zng.Record) (NativeValue, error) {
		lhs, err := lhsFunc(rec)
		if err != nil {
			return NativeValue{}, err
		}
		if lhs.typ != zng.IdBool {
			return NativeValue{}, ErrIncompatibleTypes
		}

		lv := lhs.value.(bool)
		switch node.Operator {
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
		if rhs.typ != zng.IdBool {
			return NativeValue{}, ErrIncompatibleTypes
		}

		return NativeValue{zng.IdBool, rhs.value.(bool)}, nil
	}, nil
}

func compileCompareEquality(node ast.BinaryExpression) (NativeEvaluator, error) {
	lhsFunc, err := compileNative(node.LHS)
	if err != nil {
		return nil, err
	}
	rhsFunc, err := compileNative(node.RHS)
	if err != nil {
		return nil, err
	}
	return func(rec *zng.Record) (NativeValue, error) {
		lhs, err := lhsFunc(rec)
		if err != nil {
			return NativeValue{}, err
		}
		rhs, err := rhsFunc(rec)
		if err != nil {
			return NativeValue{}, err
		}

		// XXX comparisons between int/float/port/duration

		var equal bool
		switch lhs.typ {
		case zng.IdBool:
			if rhs.typ != zng.IdBool {
				return NativeValue{}, ErrIncompatibleTypes
			}
			equal = lhs.value.(bool) == rhs.value.(bool)

		case zng.IdInt16, zng.IdInt32, zng.IdInt64:
			var rv int64
			switch rhs.typ {
			case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
				rv = int64(rhs.value.(uint64))
			case zng.IdInt16, zng.IdInt32, zng.IdInt64:
				rv = rhs.value.(int64)
			default:
				return NativeValue{}, ErrIncompatibleTypes
			}
			equal = lhs.value.(int64) == rv

		case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64, zng.IdPort:
			var rv uint64
			switch rhs.typ {
			case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
				rv = rhs.value.(uint64)
			case zng.IdInt16, zng.IdInt32, zng.IdInt64, zng.IdPort:
				rv = uint64(rhs.value.(int64))
			default:
				return NativeValue{}, ErrIncompatibleTypes
			}
			equal = lhs.value.(uint64) == rv

		case zng.IdFloat64:
			if rhs.typ != zng.IdFloat64 {
				return NativeValue{}, ErrIncompatibleTypes
			}
			equal = lhs.value.(float64) == rhs.value.(float64)

		case zng.IdString, zng.IdBstring:
			if rhs.typ != zng.IdString && rhs.typ != zng.IdBstring {
				return NativeValue{}, ErrIncompatibleTypes
			}
			equal = lhs.value.(string) == rhs.value.(string)

		case zng.IdIP:
			if rhs.typ != zng.IdIP {
				return NativeValue{}, ErrIncompatibleTypes
			}
			equal = lhs.value.(net.IP).Equal(rhs.value.(net.IP))

		case zng.IdNet:
			if rhs.typ != zng.IdNet {
				return NativeValue{}, ErrIncompatibleTypes
			}
			// is there any other way to compare nets?
			equal = lhs.value.(*net.IPNet).String() == rhs.value.(*net.IPNet).String()

		case zng.IdTime:
			if rhs.typ != zng.IdTime {
				return NativeValue{}, ErrIncompatibleTypes
			}
			equal = lhs.value.(nano.Ts) == rhs.value.(nano.Ts)

		case zng.IdDuration:
			if rhs.typ != zng.IdDuration {
				return NativeValue{}, ErrIncompatibleTypes
			}
			equal = lhs.value.(int64) == rhs.value.(int64)

		default:
			return NativeValue{}, ErrIncompatibleTypes
		}

		switch node.Operator {
		case "=":
			return NativeValue{zng.IdBool, equal}, nil
		case "!=":
			return NativeValue{zng.IdBool, !equal}, nil
		default:
			panic("bad operator")
		}
	}, nil
}

func compileCompareRelative(node ast.BinaryExpression) (NativeEvaluator, error) {
	lhsFunc, err := compileNative(node.LHS)
	if err != nil {
		return nil, err
	}
	rhsFunc, err := compileNative(node.RHS)
	if err != nil {
		return nil, err
	}
	return func(rec *zng.Record) (NativeValue, error) {
		lhs, err := lhsFunc(rec)
		if err != nil {
			return NativeValue{}, err
		}
		rhs, err := rhsFunc(rec)
		if err != nil {
			return NativeValue{}, err
		}

		// XXX comparisons between int/float/port/duration

		// holds
		//   <0 if lhs < rhs
		//    0 if lhs == rhs
		//   >0 if lhs > rhs
		var result int
		switch lhs.typ {
		case zng.IdInt16, zng.IdInt32, zng.IdInt64:
			var rv int64
			switch rhs.typ {
			case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
				rv = int64(rhs.value.(uint64))
			case zng.IdInt16, zng.IdInt32, zng.IdInt64:
				rv = rhs.value.(int64)
			default:
				return NativeValue{}, ErrIncompatibleTypes
			}
			lv := lhs.value.(int64)
			if lv < rv {
				result = -1
			} else if lv == rv {
				result = 0
			} else {
				result = 1
			}

		case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64, zng.IdPort:
			var rv uint64
			switch rhs.typ {
			case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
				rv = rhs.value.(uint64)
			case zng.IdInt16, zng.IdInt32, zng.IdInt64:
				rv = uint64(rhs.value.(int64))
			default:
				return NativeValue{}, ErrIncompatibleTypes
			}
			lv := lhs.value.(uint64)
			if lv < rv {
				result = -1
			} else if lv == rv {
				result = 0
			} else {
				result = 1
			}

		case zng.IdFloat64:
			if rhs.typ != zng.IdFloat64 {
				return NativeValue{}, ErrIncompatibleTypes
			}
			lv := lhs.value.(float64)
			rv := rhs.value.(float64)
			if lv < rv {
				result = -1
			} else if lv == rv {
				result = 0
			} else {
				result = 1
			}

		case zng.IdString, zng.IdBstring:
			if rhs.typ != zng.IdString && rhs.typ != zng.IdBstring {
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

		case zng.IdTime:
			if rhs.typ != zng.IdTime {
				return NativeValue{}, ErrIncompatibleTypes
			}
			lv := lhs.value.(nano.Ts)
			rv := rhs.value.(nano.Ts)
			if lv < rv {
				result = -1
			} else if lv == rv {
				result = 0
			} else {
				result = 1
			}

		case zng.IdDuration:
			if rhs.typ != zng.IdDuration {
				return NativeValue{}, ErrIncompatibleTypes
			}
			lv := lhs.value.(int64)
			rv := rhs.value.(int64)
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

		switch node.Operator {
		case "<":
			return NativeValue{zng.IdBool, result < 0}, nil
		case "<=":
			return NativeValue{zng.IdBool, result <= 0}, nil
		case ">":
			return NativeValue{zng.IdBool, result > 0}, nil
		case ">=":
			return NativeValue{zng.IdBool, result >= 0}, nil
		default:
			panic("bad operator")
		}
	}, nil
}

// compileArithmetic compiles an expression of the form "expr1 op expr2"
// for the arithmetic operators +, -, *, /
func compileArithmetic(node ast.BinaryExpression) (NativeEvaluator, error) {
	lhsFunc, err := compileNative(node.LHS)
	if err != nil {
		return nil, err
	}
	rhsFunc, err := compileNative(node.RHS)
	if err != nil {
		return nil, err
	}
	return func(rec *zng.Record) (NativeValue, error) {
		lhs, err := lhsFunc(rec)
		if err != nil {
			return NativeValue{}, err
		}
		rhs, err := rhsFunc(rec)
		if err != nil {
			return NativeValue{}, err
		}

		switch lhs.typ {
		case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
			v := lhs.value.(uint64)

			switch rhs.typ {
			case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
				v2 := rhs.value.(uint64)
				switch node.Operator {
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
				return NativeValue{zng.IdUint64, v}, nil

			case zng.IdFloat64:
				var r float64
				v2 := rhs.value.(float64)
				switch node.Operator {
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
				return NativeValue{zng.IdFloat64, r}, nil

			default:
				return NativeValue{}, ErrIncompatibleTypes
			}

		case zng.IdInt16, zng.IdInt32, zng.IdInt64:
			v := lhs.value.(int64)

			switch rhs.typ {
			case zng.IdInt16, zng.IdInt32, zng.IdInt64:
				v2 := rhs.value.(int64)
				switch node.Operator {
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
				return NativeValue{zng.IdInt64, v}, nil

			case zng.IdFloat64:
				var r float64
				v2 := rhs.value.(float64)
				switch node.Operator {
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
				return NativeValue{zng.IdFloat64, r}, nil

			default:
				return NativeValue{}, ErrIncompatibleTypes
			}

		case zng.IdFloat64:
			v := lhs.value.(float64)
			var v2 float64

			switch rhs.typ {
			case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
				v2 = float64(rhs.value.(uint64))

			case zng.IdInt16, zng.IdInt32, zng.IdInt64:
				v2 = float64(rhs.value.(int64))

			case zng.IdFloat64:
				v2 = rhs.value.(float64)

			default:
				return NativeValue{}, ErrIncompatibleTypes
			}

			switch node.Operator {
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
			return NativeValue{zng.IdFloat64, v}, nil

		case zng.IdString, zng.IdBstring:
			if node.Operator != "+" {
				return NativeValue{}, ErrIncompatibleTypes
			}
			t := zng.IdBstring
			if lhs.typ == zng.IdString || rhs.typ == zng.IdString {
				t = zng.IdString
			}
			return NativeValue{t, lhs.value.(string) + rhs.value.(string)}, nil

		case zng.IdTime:
			if rhs.typ != zng.IdDuration || (node.Operator != "+" && node.Operator != "-") {
				return NativeValue{}, ErrIncompatibleTypes
			}
			return NativeValue{zng.IdTime, lhs.value.(nano.Ts).Add(rhs.value.(int64))}, nil

		default:
			return NativeValue{}, ErrIncompatibleTypes
		}
	}, nil
}

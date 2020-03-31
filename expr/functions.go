package expr

import (
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zngnative"
)

type Function func([]zngnative.Value) (zngnative.Value, error)

var ErrTooFewArgs = errors.New("too few arguments")
var ErrTooManyArgs = errors.New("too many arguments")
var ErrBadArgument = errors.New("bad argument")

var allFns = map[string]struct {
	minArgs int
	maxArgs int
	impl    Function
}{
	"len": {1, 1, lenFn},

	"Math.abs":   {1, 1, mathAbs},
	"Math.ceil":  {1, 1, mathCeil},
	"Math.floor": {1, 1, mathFloor},
	"Math.log":   {1, 1, mathLog},
	"Math.max":   {1, -1, mathMax},
	"Math.min":   {1, -1, mathMin},
	"Math.mod":   {2, 2, mathMod},
	"Math.round": {1, 1, mathRound},
	"Math.pow":   {2, 2, mathPow},
	"Math.sqrt":  {1, 1, mathSqrt},

	"String.byteLen":     {1, 1, stringByteLen},
	"String.formatFloat": {1, 1, stringFormatFloat},
	"String.formatInt":   {1, 1, stringFormatInt},
	"String.formatIp":    {1, 1, stringFormatIp},
	"String.parseFloat":  {1, 1, stringParseFloat},
	"String.parseInt":    {1, 1, stringParseInt},
	"String.parseIp":     {1, 1, stringParseIp},
	"String.replace":     {3, 3, stringReplace},
	"String.runeLen":     {1, 1, stringRuneLen},
	"String.toLower":     {1, 1, stringToLower},
	"String.toUpper":     {1, 1, stringToUpper},
	"String.trim":        {1, 1, stringTrim},
}

func err(fn string, err error) (zngnative.Value, error) {
	return zngnative.Value{}, fmt.Errorf("%s: %w", fn, err)
}

func lenFn(args []zngnative.Value) (zngnative.Value, error) {
	switch zng.AliasedType(args[0].Type).(type) {
	case *zng.TypeArray, *zng.TypeSet:
		v, err := args[0].ToZngValue()
		if err != nil {
			return zngnative.Value{}, err
		}
		l, err := v.ContainerLength()
		if err != nil {
			return zngnative.Value{}, err
		}
		return zngnative.Value{zng.TypeInt64, int64(l)}, nil
	default:
		return err("len", ErrBadArgument)
	}
}

func mathAbs(args []zngnative.Value) (zngnative.Value, error) {
	switch args[0].Type.ID() {
	case zng.IdInt16, zng.IdInt32, zng.IdInt64:
		x := args[0].Value.(int64)
		if x < 0 {
			x = -x
		}
		return zngnative.Value{zng.TypeInt64, x}, nil

	case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
		return args[0], nil

	case zng.IdFloat64:
		return zngnative.Value{zng.TypeFloat64, math.Abs(args[0].Value.(float64))}, nil

	default:
		return err("Math.abs", ErrBadArgument)
	}
}

func mathCeil(args []zngnative.Value) (zngnative.Value, error) {
	switch args[0].Type.ID() {
	case zng.IdInt16, zng.IdInt32, zng.IdInt64, zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
		return args[0], nil

	case zng.IdFloat64:
		return zngnative.Value{zng.TypeFloat64, math.Ceil(args[0].Value.(float64))}, nil

	default:
		return err("Math.Ceil", ErrBadArgument)
	}
}

func mathFloor(args []zngnative.Value) (zngnative.Value, error) {
	switch args[0].Type.ID() {
	case zng.IdInt16, zng.IdInt32, zng.IdInt64, zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
		return args[0], nil

	case zng.IdFloat64:
		return zngnative.Value{zng.TypeFloat64, math.Floor(args[0].Value.(float64))}, nil

	default:
		return err("Math.Floor", ErrBadArgument)
	}
}

func mathLog(args []zngnative.Value) (zngnative.Value, error) {
	x, ok := zngnative.CoerceNativeToFloat64(args[0])
	if !ok {
		return err("Math.log", ErrBadArgument)
	}
	if x <= 0 {
		return err("Math.log", ErrBadArgument)
	}
	return zngnative.Value{zng.TypeFloat64, math.Log(x)}, nil
}

func mathMax(args []zngnative.Value) (zngnative.Value, error) {
	switch args[0].Type.ID() {
	case zng.IdInt16, zng.IdInt32, zng.IdInt64:
		ret := args[0].Value.(int64)
		for _, val := range args[1:] {
			v, ok := zngnative.CoerceNativeToInt(val)
			if ok && v > ret {
				ret = v
			}
		}
		return zngnative.Value{zng.TypeInt64, ret}, nil

	case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
		ret := args[0].Value.(uint64)
		for _, val := range args[1:] {
			v, ok := zngnative.CoerceNativeToUint(val)
			if ok && v > ret {
				ret = v
			}
		}
		return zngnative.Value{zng.TypeUint64, ret}, nil

	case zng.IdFloat64:
		ret := args[0].Value.(float64)
		for _, val := range args[1:] {
			v, ok := zngnative.CoerceNativeToFloat64(val)
			if ok && v > ret {
				ret = v
			}
		}
		return zngnative.Value{zng.TypeFloat64, ret}, nil

	default:
		return err("Math.max", ErrBadArgument)
	}
}

func mathMin(args []zngnative.Value) (zngnative.Value, error) {
	switch args[0].Type.ID() {
	case zng.IdInt16, zng.IdInt32, zng.IdInt64:
		ret := args[0].Value.(int64)
		for _, val := range args[1:] {
			v, ok := zngnative.CoerceNativeToInt(val)
			if ok && v < ret {
				ret = v
			}
		}
		return zngnative.Value{zng.TypeInt64, ret}, nil

	case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
		ret := args[0].Value.(uint64)
		for _, val := range args[1:] {
			v, ok := zngnative.CoerceNativeToUint(val)
			if ok && v < ret {
				ret = v
			}
		}
		return zngnative.Value{zng.TypeUint64, ret}, nil

	case zng.IdFloat64:
		ret := args[0].Value.(float64)
		for _, val := range args[1:] {
			v, ok := zngnative.CoerceNativeToFloat64(val)
			if ok && v < ret {
				ret = v
			}
		}
		return zngnative.Value{zng.TypeFloat64, ret}, nil

	default:
		return err("Math.min", ErrBadArgument)
	}
}

func mathMod(args []zngnative.Value) (zngnative.Value, error) {
	y, ok := zngnative.CoerceNativeToUint(args[1])
	if !ok {
		return err("Math.mod", ErrBadArgument)
	}

	switch args[0].Type.ID() {
	case zng.IdInt16, zng.IdInt32, zng.IdInt64:
		x := args[0].Value.(int64)
		return zngnative.Value{zng.TypeInt64, x % int64(y)}, nil

	case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
		x := args[0].Value.(uint64)
		return zngnative.Value{zng.TypeUint64, x % y}, nil

	default:
		return err("Math.mod", ErrBadArgument)
	}
}

func mathRound(args []zngnative.Value) (zngnative.Value, error) {
	switch args[0].Type.ID() {
	case zng.IdInt16, zng.IdInt32, zng.IdInt64, zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
		return args[0], nil

	case zng.IdFloat64:
		return zngnative.Value{zng.TypeFloat64, math.Round(args[0].Value.(float64))}, nil

	default:
		return err("Math.round", ErrBadArgument)
	}
}

func mathPow(args []zngnative.Value) (zngnative.Value, error) {
	x, ok := zngnative.CoerceNativeToFloat64(args[0])
	if !ok {
		return err("Math.pow", ErrBadArgument)
	}
	y, ok := zngnative.CoerceNativeToFloat64(args[1])
	if !ok {
		return err("Math.pow", ErrBadArgument)
	}
	r := math.Pow(x, y)
	if math.IsNaN(r) {
		return err("Math.pow", ErrBadArgument)
	}
	return zngnative.Value{zng.TypeFloat64, r}, nil
}

func mathSqrt(args []zngnative.Value) (zngnative.Value, error) {
	var x float64
	switch args[0].Type.ID() {
	case zng.IdFloat64:
		x = args[0].Value.(float64)
	case zng.IdInt16, zng.IdInt32, zng.IdInt64:
		x = float64(args[0].Value.(int64))
	case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
		x = float64(args[0].Value.(uint64))
	default:
		return err("Math.sqrt", ErrBadArgument)
	}

	r := math.Sqrt(x)
	if math.IsNaN(r) {
		// For now we can't represent non-numeric values in a float64,
		// we will revisit this but it has implications for file
		// formats, zql, etc.
		return err("Math.sqrt", ErrBadArgument)
	}

	return zngnative.Value{zng.TypeFloat64, r}, nil
}

func stringByteLen(args []zngnative.Value) (zngnative.Value, error) {
	switch args[0].Type.ID() {
	case zng.IdString, zng.IdBstring:
		v := len(args[0].Value.(string))
		return zngnative.Value{zng.TypeInt64, int64(v)}, nil
	default:
		return err("Strings.byteLen", ErrBadArgument)
	}

}

func stringFormatFloat(args []zngnative.Value) (zngnative.Value, error) {
	if args[0].Type.ID() != zng.IdFloat64 {
		return err("string.floatToString", ErrBadArgument)
	}
	s := strconv.FormatFloat(args[0].Value.(float64), 'g', -1, 64)
	return zngnative.Value{zng.TypeString, s}, nil
}

func stringFormatInt(args []zngnative.Value) (zngnative.Value, error) {
	var s string
	switch args[0].Type.ID() {
	case zng.IdInt16, zng.IdInt32, zng.IdInt64:
		s = strconv.FormatInt(args[0].Value.(int64), 10)
	case zng.IdByte, zng.IdUint16, zng.IdUint32, zng.IdUint64:
		s = strconv.FormatUint(args[0].Value.(uint64), 10)
	default:
		return err("string.intToString", ErrBadArgument)
	}
	return zngnative.Value{zng.TypeString, s}, nil
}

func stringFormatIp(args []zngnative.Value) (zngnative.Value, error) {
	if args[0].Type.ID() != zng.IdIP {
		return err("string.ipToString", ErrBadArgument)
	}
	s := args[0].Value.(net.IP).String()
	return zngnative.Value{zng.TypeString, s}, nil
}

func stringParseInt(args []zngnative.Value) (zngnative.Value, error) {
	switch args[0].Type.ID() {
	case zng.IdString, zng.IdBstring:
		i, perr := strconv.ParseInt(args[0].Value.(string), 10, 64)
		if perr != nil {
			// Get rid of the strconv wrapping gunk to get the
			// actual error message
			e := perr.(*strconv.NumError)
			return err("String.parseInt", e.Err)
		}
		return zngnative.Value{zng.TypeInt64, i}, nil
	default:
		return err("String.parseInt", ErrBadArgument)
	}
}

func stringParseFloat(args []zngnative.Value) (zngnative.Value, error) {
	switch args[0].Type.ID() {
	case zng.IdString, zng.IdBstring:
		f, perr := strconv.ParseFloat(args[0].Value.(string), 64)
		if perr != nil {
			// Get rid of the strconv wrapping gunk to get the
			// actual error message
			e := perr.(*strconv.NumError)
			return err("String.parseFloat", e.Err)
		}
		return zngnative.Value{zng.TypeFloat64, f}, nil
	default:
		return err("String.parseFloat", ErrBadArgument)
	}
}

func stringParseIp(args []zngnative.Value) (zngnative.Value, error) {
	switch args[0].Type.ID() {
	case zng.IdString, zng.IdBstring:
		a := net.ParseIP(args[0].Value.(string))
		if a == nil {
			return err("String.parseIp", ErrBadArgument)
		}
		return zngnative.Value{zng.TypeIP, a}, nil
	default:
		return err("String.parseIp", ErrBadArgument)
	}
}

func isString(v zngnative.Value) bool {
	i := v.Type.ID()
	return i == zng.IdString || i == zng.IdBstring
}

func stringReplace(args []zngnative.Value) (zngnative.Value, error) {
	if !isString(args[0]) || !isString(args[1]) || !isString(args[2]) {
		return err("String.replace", ErrBadArgument)
	}
	s := strings.ReplaceAll(args[0].Value.(string), args[1].Value.(string), args[2].Value.(string))
	return zngnative.Value{zng.TypeString, s}, nil
}

func stringRuneLen(args []zngnative.Value) (zngnative.Value, error) {
	switch args[0].Type.ID() {
	case zng.IdString, zng.IdBstring:
		v := utf8.RuneCountInString(args[0].Value.(string))
		return zngnative.Value{zng.TypeInt64, int64(v)}, nil
	default:
		return err("Strings.byteLen", ErrBadArgument)
	}

}
func stringToLower(args []zngnative.Value) (zngnative.Value, error) {
	if !isString(args[0]) {
		return err("String.toLower", ErrBadArgument)
	}
	s := strings.ToLower(args[0].Value.(string))
	return zngnative.Value{zng.TypeString, s}, nil
}

func stringToUpper(args []zngnative.Value) (zngnative.Value, error) {
	if !isString(args[0]) {
		return err("String.toUpper", ErrBadArgument)
	}
	s := strings.ToUpper(args[0].Value.(string))
	return zngnative.Value{zng.TypeString, s}, nil
}

func stringTrim(args []zngnative.Value) (zngnative.Value, error) {
	if !isString(args[0]) {
		return err("String.trim", ErrBadArgument)
	}
	s := strings.TrimSpace(args[0].Value.(string))
	return zngnative.Value{zng.TypeString, s}, nil
}

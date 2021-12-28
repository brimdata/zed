package expr

import (
	"bytes"
	"fmt"
	"math"
	"net"
	"regexp"
	"regexp/syntax"

	//XXX this shouldn't be reaching into the AST but we'll leave it for
	// now until we factor-in the flow-based package
	"github.com/brimdata/zed"
	astzed "github.com/brimdata/zed/compiler/ast/zed"
	"github.com/brimdata/zed/pkg/byteconv"
	"github.com/brimdata/zed/zson"
)

//XXX TBD:
// - change these comparisons to work in the zcode.Bytes domain
// - add timer, interval comparisons when we add time, interval literals to the language
// - add count comparisons when we add count literals to the language
// - add set/array/record comparisons when we add container literals to the language

// Predicate is a function that takes a Value and returns a boolean result
// based on the typed value.
type Boolean func(*zed.Value) bool

var compareBool = map[string]func(bool, bool) bool{
	"=":  func(a, b bool) bool { return a == b },
	"!=": func(a, b bool) bool { return a != b },
	">":  func(a, b bool) bool { return a && !b },
	">=": func(a, b bool) bool { return a || !b },
	"<":  func(a, b bool) bool { return !a && b },
	"<=": func(a, b bool) bool { return !a || b },
}

// CompareBool returns a Predicate that compares zed.Values to a boolean literal
// that must be a boolean or coercible to an integer.  In the later case, the integer
// is converted to a boolean.
func CompareBool(op string, pattern bool) (Boolean, error) {
	compare, ok := compareBool[op]
	if !ok {
		return nil, fmt.Errorf("unknown bool comparator: %s", op)
	}
	return func(val *zed.Value) bool {
		if val.Type.ID() != zed.IDBool {
			return false
		}
		b, err := zed.DecodeBool(val.Bytes)
		if err != nil {
			return false
		}
		return compare(b, pattern)
	}, nil
}

var compareInt = map[string]func(int64, int64) bool{
	"=":  func(a, b int64) bool { return a == b },
	"!=": func(a, b int64) bool { return a != b },
	">":  func(a, b int64) bool { return a > b },
	">=": func(a, b int64) bool { return a >= b },
	"<":  func(a, b int64) bool { return a < b },
	"<=": func(a, b int64) bool { return a <= b }}

var compareFloat = map[string]func(float64, float64) bool{
	"=":  func(a, b float64) bool { return a == b },
	"!=": func(a, b float64) bool { return a != b },
	">":  func(a, b float64) bool { return a > b },
	">=": func(a, b float64) bool { return a >= b },
	"<":  func(a, b float64) bool { return a < b },
	"<=": func(a, b float64) bool { return a <= b }}

// Return a predicate for comparing this value to one more typed
// byte slices by calling the predicate function with a Value.
// Operand is one of "=", "!=", "<", "<=", ">", ">=".
func CompareInt64(op string, pattern int64) (Boolean, error) {
	CompareInt, ok1 := compareInt[op]
	CompareFloat, ok2 := compareFloat[op]
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("unknown int comparator: %s", op)
	}
	// many different Zed data types can be compared with integers
	return func(val *zed.Value) bool {
		zv := val.Bytes
		switch val.Type.ID() {
		case zed.IDInt8, zed.IDInt16, zed.IDInt32, zed.IDInt64:
			v, err := zed.DecodeInt(zv)
			if err == nil {
				return CompareInt(v, pattern)
			}
		case zed.IDUint8, zed.IDUint16, zed.IDUint32, zed.IDUint64:
			v, err := zed.DecodeUint(zv)
			if err == nil && v <= math.MaxInt64 {
				return CompareInt(int64(v), pattern)
			}
		case zed.IDFloat32, zed.IDFloat64:
			v, err := zed.DecodeFloat(zv)
			if err == nil {
				return CompareFloat(v, float64(pattern))
			}
		case zed.IDTime:
			ts, err := zed.DecodeTime(zv)
			if err == nil {
				return CompareInt(int64(ts), pattern*1e9)
			}
		case zed.IDDuration:
			v, err := zed.DecodeInt(zv)
			if err == nil {
				return CompareInt(int64(v), pattern*1e9)
			}
		}
		return false
	}, nil
}

func CompareTime(op string, pattern int64) (Boolean, error) {
	CompareInt, ok1 := compareInt[op]
	CompareFloat, ok2 := compareFloat[op]
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("unknown int comparator: %s", op)
	}
	// many different Zed data types can be compared with integers
	return func(val *zed.Value) bool {
		zv := val.Bytes
		switch val.Type.ID() {
		case zed.IDInt8, zed.IDInt16, zed.IDInt32, zed.IDInt64:
			v, err := zed.DecodeInt(zv)
			if err == nil {
				return CompareInt(v, pattern)
			}
		case zed.IDUint8, zed.IDUint16, zed.IDUint32, zed.IDUint64:
			v, err := zed.DecodeUint(zv)
			if err == nil && v <= math.MaxInt64 {
				return CompareInt(int64(v), pattern)
			}
		case zed.IDFloat32, zed.IDFloat64:
			v, err := zed.DecodeFloat(zv)
			if err == nil {
				return CompareFloat(v, float64(pattern))
			}
		case zed.IDTime:
			ts, err := zed.DecodeTime(zv)
			if err == nil {
				return CompareInt(int64(ts), pattern)
			}
		case zed.IDDuration:
			v, err := zed.DecodeInt(zv)
			if err == nil {
				return CompareInt(int64(v), pattern)
			}
		}
		return false
	}, nil
}

//XXX should just do equality and we should compare in the encoded domain
// and not make copies and have separate cases for len 4 and len 16
var compareAddr = map[string]func(net.IP, net.IP) bool{
	"=":  func(a, b net.IP) bool { return a.Equal(b) },
	"!=": func(a, b net.IP) bool { return !a.Equal(b) },
	">":  func(a, b net.IP) bool { return bytes.Compare(a, b) > 0 },
	">=": func(a, b net.IP) bool { return bytes.Compare(a, b) >= 0 },
	"<":  func(a, b net.IP) bool { return bytes.Compare(a, b) < 0 },
	"<=": func(a, b net.IP) bool { return bytes.Compare(a, b) <= 0 },
}

// Comparison returns a Predicate that compares typed byte slices that must
// be TypeAddr with the value's address using a comparison based on op.
// Only equality operands are allowed.
func CompareIP(op string, pattern net.IP) (Boolean, error) {
	compare, ok := compareAddr[op]
	if !ok {
		return nil, fmt.Errorf("unknown addr comparator: %s", op)
	}
	return func(val *zed.Value) bool {
		if val.Type.ID() != zed.IDIP {
			return false
		}
		ip, err := zed.DecodeIP(val.Bytes)
		if err != nil {
			return false
		}
		return compare(ip, pattern)
	}, nil
}

// CompareFloat64 returns a Predicate that compares typed byte slices that must
// be coercible to an double with the value's double value using a comparison
// based on op.  Int, count, port, and double types can
// all be converted to the integer value.  XXX there are some overflow issues here.
func CompareFloat64(op string, pattern float64) (Boolean, error) {
	compare, ok := compareFloat[op]
	if !ok {
		return nil, fmt.Errorf("unknown double comparator: %s", op)
	}
	return func(val *zed.Value) bool {
		zv := val.Bytes
		switch val.Type.ID() {
		// We allow comparison of float constant with integer-y
		// fields and just use typeDouble to parse since it will do
		// the right thing for integers.  XXX do we want to allow
		// integers that cause float64 overflow?  user can always
		// use an integer constant instead of a float constant to
		// compare with the integer-y field.
		case zed.IDFloat32, zed.IDFloat64:
			v, err := zed.DecodeFloat(zv)
			if err == nil {
				return compare(v, pattern)
			}
		case zed.IDInt8, zed.IDInt16, zed.IDInt32, zed.IDInt64:
			v, err := zed.DecodeInt(zv)
			if err == nil {
				return compare(float64(v), pattern)
			}
		case zed.IDUint8, zed.IDUint16, zed.IDUint32, zed.IDUint64:
			v, err := zed.DecodeUint(zv)
			if err == nil {
				return compare(float64(v), pattern)
			}
		case zed.IDTime:
			ts, err := zed.DecodeTime(zv)
			if err == nil {
				return compare(float64(ts)/1e9, pattern)
			}
		case zed.IDDuration:
			v, err := zed.DecodeDuration(zv)
			if err == nil {
				return compare(float64(v)/1e9, pattern)
			}
		}
		return false
	}, nil
}

var compareString = map[string]func(string, string) bool{
	"=":  func(a, b string) bool { return a == b },
	"!=": func(a, b string) bool { return a != b },
	">":  func(a, b string) bool { return a > b },
	">=": func(a, b string) bool { return a >= b },
	"<":  func(a, b string) bool { return a < b },
	"<=": func(a, b string) bool { return a <= b },
}

func CompareBstring(op string, pattern []byte) (Boolean, error) {
	compare, ok := compareString[op]
	if !ok {
		return nil, fmt.Errorf("unknown string comparator: %s", op)
	}
	s := string(pattern)
	return func(val *zed.Value) bool {
		switch val.Type.ID() {
		case zed.IDBstring, zed.IDString:
			return compare(byteconv.UnsafeString(val.Bytes), s)
		}
		return false
	}, nil
}

var compareBytes = map[string]func([]byte, []byte) bool{
	"=":  func(a, b []byte) bool { return bytes.Equal(a, b) },
	"!=": func(a, b []byte) bool { return !bytes.Equal(a, b) },
	">":  func(a, b []byte) bool { return bytes.Compare(a, b) > 0 },
	">=": func(a, b []byte) bool { return bytes.Compare(a, b) >= 0 },
	"<":  func(a, b []byte) bool { return bytes.Compare(a, b) < 0 },
	"<=": func(a, b []byte) bool { return bytes.Compare(a, b) <= 0 },
}

func CompareBytes(op string, pattern []byte) (Boolean, error) {
	compare, ok := compareBytes[op]
	if !ok {
		return nil, fmt.Errorf("unknown bytes comparator: %s", op)
	}
	return func(val *zed.Value) bool {
		switch val.Type.ID() {
		case zed.IDBytes:
			return compare(val.Bytes, pattern)
		}
		return false
	}, nil
}

func CompileRegexp(pattern string) (*regexp.Regexp, error) {
	re, err := regexp.Compile(string(zed.UnescapeBstring([]byte(pattern))))
	if err != nil {
		if syntaxErr, ok := err.(*syntax.Error); ok {
			syntaxErr.Expr = pattern
		}
		return nil, err
	}
	return re, err
}

// NewRegexpBoolean returns a Booelan that compares values that must
// be a stringy the given regexp.
func NewRegexpBoolean(re *regexp.Regexp) Boolean {
	return func(val *zed.Value) bool {
		if zed.IsStringy(val.Type.ID()) {
			return re.Match(val.Bytes)
		}
		return false
	}
}

func CompareNull(op string) (Boolean, error) {
	switch op {
	case "=":
		return func(val *zed.Value) bool {
			return val.IsNull()
		}, nil
	case "!=":
		return func(val *zed.Value) bool {
			return !val.IsNull()
		}, nil
	default:
		return nil, fmt.Errorf("unknown null comparator: %s", op)
	}
}

// a better way to do this would be to compare IP's and mask's but
// go doesn't provide an easy way to compare masks so we do this
// hacky thing and compare strings
var compareSubnet = map[string]func(*net.IPNet, *net.IPNet) bool{
	"=":  func(a, b *net.IPNet) bool { return bytes.Equal(a.IP, b.IP) },
	"!=": func(a, b *net.IPNet) bool { return bytes.Equal(a.IP, b.IP) },
	"<":  func(a, b *net.IPNet) bool { return bytes.Compare(a.IP, b.IP) < 0 },
	"<=": func(a, b *net.IPNet) bool { return bytes.Compare(a.IP, b.IP) <= 0 },
	">":  func(a, b *net.IPNet) bool { return bytes.Compare(a.IP, b.IP) > 0 },
	">=": func(a, b *net.IPNet) bool { return bytes.Compare(a.IP, b.IP) >= 0 },
}

var matchSubnet = map[string]func(net.IP, *net.IPNet) bool{
	"=": func(a net.IP, b *net.IPNet) bool {
		ok := b.IP.Equal(a.Mask(b.Mask))
		return ok
	},
	"!=": func(a net.IP, b *net.IPNet) bool {
		return !b.IP.Equal(a.Mask(b.Mask))
	},
	"<": func(a net.IP, b *net.IPNet) bool {
		net := a.Mask(b.Mask)
		return bytes.Compare(net, b.IP) < 0
	},
	"<=": func(a net.IP, b *net.IPNet) bool {
		net := a.Mask(b.Mask)
		return bytes.Compare(net, b.IP) <= 0
	},
	">": func(a net.IP, b *net.IPNet) bool {
		net := a.Mask(b.Mask)
		return bytes.Compare(net, b.IP) > 0
	},
	">=": func(a net.IP, b *net.IPNet) bool {
		net := a.Mask(b.Mask)
		return bytes.Compare(net, b.IP) >= 0
	},
}

// Comparison returns a Predicate that compares typed byte slices that must
// be an addr or a subnet to the value's subnet value using a comparison
// based on op.  Onluy equalty and inequality are permitted.  If the typed
// byte slice is a subnet, then the comparison is based on strict equality.
// If the typed byte slice is an addr, then the comparison is performed by
// doing a CIDR match on the address with the subnet.
func CompareSubnet(op string, pattern *net.IPNet) (Boolean, error) {
	compare, ok1 := compareSubnet[op]
	match, ok2 := matchSubnet[op]
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("unknown subnet comparator: %s", op)
	}
	return func(val *zed.Value) bool {
		switch val.Type.ID() {
		case zed.IDIP:
			ip, err := zed.DecodeIP(val.Bytes)
			if err == nil {
				return match(ip, pattern)
			}
		case zed.IDNet:
			subnet, err := zed.DecodeNet(val.Bytes)
			if err == nil {
				return compare(subnet, pattern)
			}
		}
		return false
	}, nil
}

// Given a predicate for comparing individual elements, produce a new
// predicate that implements the "in" comparison.  The new predicate looks
// at the type of the value being compared, if it is a set or array,
// the original predicate is applied to each element.  The new precicate
// returns true iff the predicate matched an element from the collection.
func Contains(compare Boolean) Boolean {
	return func(val *zed.Value) bool {
		var el zed.Value
		el.Type = zed.InnerType(val.Type)
		if el.Type == nil {
			return false
		}
		for it := val.Iter(); !it.Done(); {
			el.Bytes, _ = it.Next()
			if compare(&el) {
				return true
			}
		}
		return false
	}
}

// Comparison returns a Predicate for comparing this value to other values.
// The op argument is one of "=", "!=", "<", "<=", ">", ">=".
// See the comments of the various type implementations
// of this method as some types limit the operand to equality and
// the various types handle coercion in different ways.
func Comparison(op string, primitive astzed.Primitive) (Boolean, error) {
	// String literals inside Zed are parsed as zng bstrings
	// (since bstrings can represent a wider range of values,
	// specifically arrays of bytes that do not correspond to
	// UTF-8 encoded strings).
	if primitive.Type == "string" {
		primitive = astzed.Primitive{Kind: "Primitive", Type: "bstring", Text: primitive.Text}
	}
	zv, err := zson.ParsePrimitive(primitive.Type, primitive.Text)
	if err != nil {
		return nil, err
	}
	switch zv.Type.(type) {
	case *zed.TypeOfNull:
		return CompareNull(op)
	case *zed.TypeOfIP:
		v, err := zed.DecodeIP(zv.Bytes)
		if err != nil {
			return nil, err
		}
		return CompareIP(op, v)
	case *zed.TypeOfNet:
		v, err := zed.DecodeNet(zv.Bytes)
		if err != nil {
			return nil, err
		}
		return CompareSubnet(op, v)
	case *zed.TypeOfBool:
		v, err := zed.DecodeBool(zv.Bytes)
		if err != nil {
			return nil, err
		}
		return CompareBool(op, v)
	case *zed.TypeOfFloat64:
		v, err := zed.DecodeFloat64(zv.Bytes)
		if err != nil {
			return nil, err
		}
		return CompareFloat64(op, v)
	case *zed.TypeOfString, *zed.TypeOfBstring, *zed.TypeOfType, *zed.TypeOfError:
		return CompareBstring(op, zv.Bytes)
	case *zed.TypeOfBytes:
		return CompareBytes(op, zv.Bytes)
	case *zed.TypeOfInt64:
		v, err := zed.DecodeInt(zv.Bytes)
		if err != nil {
			return nil, err
		}
		return CompareInt64(op, v)
	case *zed.TypeOfTime, *zed.TypeOfDuration:
		v, err := zed.DecodeInt(zv.Bytes)
		if err != nil {
			return nil, err
		}
		return CompareTime(op, v)
	default:
		return nil, fmt.Errorf("literal comparison of type %q unsupported", zv.Type)
	}
}

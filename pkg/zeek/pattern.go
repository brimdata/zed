package zeek

import (
	"fmt"
	"regexp"

	"github.com/mccanne/zq/pkg/zval"
)

type TypeOfPattern struct{}

var comparePattern = map[string]func(*regexp.Regexp, []byte) bool{
	"eql":  func(re *regexp.Regexp, val []byte) bool { return re.Match(val) },
	"neql": func(re *regexp.Regexp, val []byte) bool { return !re.Match(val) },
}

func (t *TypeOfPattern) String() string {
	return "pattern"
}

func (t *TypeOfPattern) Parse(value []byte) (*regexp.Regexp, error) {
	if value == nil {
		return nil, ErrUnset
	}
	return regexp.Compile(ustring(value))
}

func (t *TypeOfPattern) Format(value []byte) (interface{}, error) {
	return t.Parse(value)
}

func (t *TypeOfPattern) New(value []byte) (Value, error) {
	re, err := regexp.Compile(string(value))
	if err != nil {
		return nil, err
	}
	return &Pattern{Native: re}, nil
}

//XXX need to check if zeek regexp and go regexp are the same, though it
// doesn't really matter because I don't think they appear in log files but
// are rather used in zeek scripts
type Pattern struct {
	Native *regexp.Regexp
}

func (p *Pattern) String() string {
	return p.Native.String()
}

func (p *Pattern) Encode(dst zval.Encoding) zval.Encoding {
	v := []byte(p.String())
	return zval.AppendValue(dst, v)
}

func (p *Pattern) Type() Type {
	return TypePattern
}

// Comparison returns a Predicate that compares typed byte slices that must
// be a string or enum with the value's regular expression using a regex
// match comparison based on equality or inequality based on op.
func (p *Pattern) Comparison(op string) (Predicate, error) {
	compare, ok := comparePattern[op]
	if !ok {
		return nil, fmt.Errorf("unknown pattern comparator: %s", op)
	}
	re := p.Native
	return func(e TypedEncoding) bool {
		switch e.Type.(type) {
		case *TypeOfString, *TypeOfEnum:
			return compare(re, e.Body)
		}
		return false
	}, nil
}

func (p *Pattern) Coerce(typ Type) Value {
	_, ok := typ.(*TypeOfPattern)
	if ok {
		return p
	}
	return nil
}

func (p *Pattern) Elements() ([]Value, bool) { return nil, false }

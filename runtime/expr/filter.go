package expr

import (
	"bytes"
	"errors"
	"fmt"
	"net/netip"
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/byteconv"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

type searchByPred struct {
	pred  Boolean
	expr  Evaluator
	types map[zed.Type]bool
}

func SearchByPredicate(pred Boolean, e Evaluator) Evaluator {
	return &searchByPred{
		pred:  pred,
		expr:  e,
		types: make(map[zed.Type]bool),
	}
}

func (s *searchByPred) Eval(ectx Context, val *zed.Value) *zed.Value {
	if s.expr != nil {
		val = s.expr.Eval(ectx, val)
		if val.IsError() {
			return zed.False
		}
	}
	if errMatch == val.Walk(func(typ zed.Type, body zcode.Bytes) error {
		if s.searchType(typ) || s.pred(zed.NewValue(typ, body)) {
			return errMatch
		}
		return nil
	}) {
		return zed.True
	}
	return zed.False
}

func (s *searchByPred) searchType(typ zed.Type) bool {
	if match, ok := s.types[typ]; ok {
		return match
	}
	var match bool
	recType := zed.TypeRecordOf(typ)
	if recType != nil {
		var nameIter FieldNameIter
		nameIter.Init(recType)
		for !nameIter.Done() {
			if s.pred(zed.NewValue(zed.TypeString, nameIter.Next())) {
				match = true
				break
			}
		}
	}
	s.types[typ] = match
	s.types[recType] = match
	return match
}

// stringSearch is like strings.Contains() but with case-insensitive
// comparison.
func stringSearch(a, b string) bool {
	alen := len(a)
	blen := len(b)

	if blen > alen {
		return false
	}

	end := alen - blen + 1
	i := 0
	for i < end {
		if strings.EqualFold(a[i:i+blen], b) {
			return true
		}
		i++
	}
	return false
}

var errMatch = errors.New("match")

type search struct {
	text    string
	compare Boolean
	expr    Evaluator
}

// NewSearch creates a filter that searches Zed records for the
// given value, which must be of a type other than string.  The filter
// matches a record that contains this value either as the value of any
// field or inside any set or array.  It also matches a record if the string
// representaton of the search value appears inside inside any string-valued
// field (or inside any element of a set or array of strings).
func NewSearch(searchtext string, searchval *zed.Value, expr Evaluator) (Evaluator, error) {
	if zed.TypeUnder(searchval.Type) == zed.TypeNet {
		n := zed.DecodeNet(searchval.Bytes)
		a, ok := netip.AddrFromSlice(n.IP)
		if !ok {
			return nil, fmt.Errorf("xxx %s", zson.MustFormatValue(searchval))
		}
		ones, _ := n.Mask.Size()
		return &searchCIDR{
			net:   netip.PrefixFrom(a, ones),
			bytes: searchval.Bytes,
		}, nil
	}
	typedCompare, err := Comparison("==", searchval)
	if err != nil {
		return nil, err
	}
	return &search{searchtext, typedCompare, expr}, nil
}

func (s *search) Eval(ectx Context, val *zed.Value) *zed.Value {
	if s.expr != nil {
		val = s.expr.Eval(ectx, val)
		if val.IsError() {
			return zed.False
		}
	}
	if errMatch == val.Walk(func(typ zed.Type, body zcode.Bytes) error {
		if typ.ID() == zed.IDString {
			if stringSearch(byteconv.UnsafeString(body), s.text) {
				return errMatch
			}
		} else if s.compare(zed.NewValue(typ, body)) {
			return errMatch
		}
		return nil
	}) {
		return zed.True
	}
	return zed.False
}

type searchCIDR struct {
	net   netip.Prefix
	bytes zcode.Bytes
}

func (s *searchCIDR) Eval(_ Context, val *zed.Value) *zed.Value {
	if errMatch == val.Walk(func(typ zed.Type, body zcode.Bytes) error {
		switch typ.ID() {
		case zed.IDNet:
			if bytes.Compare(body, s.bytes) == 0 {
				return errMatch
			}
		case zed.IDIP:
			if s.net.Contains(zed.DecodeIP(body)) {
				return errMatch
			}
		}
		return nil
	}) {
		return zed.True
	}
	return zed.False
}

type searchString struct {
	term    string
	compare Boolean
	expr    Evaluator
	types   map[zed.Type]bool
}

// NewSearchString is like NewSeach but handles the special case of matching
// field names in addition to string values.
func NewSearchString(term string, expr Evaluator) Evaluator {
	return &searchString{
		term:  term,
		expr:  expr,
		types: make(map[zed.Type]bool),
	}
}

func (s *searchString) searchType(typ zed.Type) bool {
	if match, ok := s.types[typ]; ok {
		return match
	}
	var match bool
	recType := zed.TypeRecordOf(typ)
	if recType != nil {
		var nameIter FieldNameIter
		nameIter.Init(recType)
		for !nameIter.Done() {
			if stringSearch(byteconv.UnsafeString(nameIter.Next()), s.term) {
				match = true
				break
			}
		}
	}
	s.types[typ] = match
	s.types[recType] = match
	return match
}

func (s *searchString) Eval(ectx Context, val *zed.Value) *zed.Value {
	if s.expr != nil {
		val = s.expr.Eval(ectx, val)
		if val.IsError() {
			return zed.False
		}
	}
	// Memoize the result of a search across the names in the
	// record columns for each unique record type.
	if s.searchType(val.Type) {
		return zed.True
	}
	if errMatch == val.Walk(func(typ zed.Type, body zcode.Bytes) error {
		if s.searchType(typ) {
			return errMatch
		}
		if typ.ID() == zed.IDString &&
			stringSearch(byteconv.UnsafeString(body), s.term) {
			return errMatch
		}
		return nil
	}) {
		return zed.True
	}
	return zed.False
}

type filter struct {
	expr Evaluator
	pred Boolean
}

func NewFilter(expr Evaluator, pred Boolean) Evaluator {
	return &filter{expr, pred}
}

func (f *filter) Eval(ectx Context, this *zed.Value) *zed.Value {
	val := f.expr.Eval(ectx, this)
	if val.IsError() {
		return val
	}
	if f.pred(val) {
		return zed.True
	}
	return zed.False
}

type filterApplier struct {
	zctx *zed.Context
	expr Evaluator
}

func NewFilterApplier(zctx *zed.Context, e Evaluator) Applier {
	return &filterApplier{zctx, e}
}

func (f *filterApplier) Eval(ectx Context, this *zed.Value) *zed.Value {
	val, ok := EvalBool(f.zctx, ectx, this, f.expr)
	if ok {
		if zed.IsTrue(val.Bytes) {
			return this
		}
		return f.zctx.Missing()
	}
	return val
}

func (*filterApplier) String() string { return "filter" }

func (*filterApplier) Warning() string { return "" }

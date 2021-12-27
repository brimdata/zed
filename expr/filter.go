package expr

import (
	"errors"
	"strings"

	"github.com/brimdata/zed"
	astzed "github.com/brimdata/zed/compiler/ast/zed"
	"github.com/brimdata/zed/pkg/byteconv"
	"github.com/brimdata/zed/pkg/stringsearch"
	"github.com/brimdata/zed/zcode"
)

type searchByPred struct {
	pred  Boolean
	types map[zed.Type]bool
}

func SearchByPredicate(pred Boolean) Evaluator {
	return &searchByPred{
		pred:  pred,
		types: make(map[zed.Type]bool),
	}
}

func (s *searchByPred) Eval(_ Context, this *zed.Value) *zed.Value {
	if errMatch == this.Walk(func(typ zed.Type, body zcode.Bytes) error {
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
		var nameIter stringsearch.FieldNameIter
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
}

// NewSearch creates a filter that searches Zed records for the
// given value, which must be of a type other than string.  The filter
// matches a record that contains this value either as the value of any
// field or inside any set or array.  It also matches a record if the string
// representaton of the search value appears inside inside any string-valued
// field (or inside any element of a set or array of strings).
func NewSearch(searchtext string, searchval astzed.Primitive) (Evaluator, error) {
	typedCompare, err := Comparison("=", searchval)
	if err != nil {
		return nil, err
	}
	return &search{searchtext, typedCompare}, nil
}

func (s *search) Eval(_ Context, this *zed.Value) *zed.Value {
	if errMatch == this.Walk(func(typ zed.Type, body zcode.Bytes) error {
		if zed.IsStringy(typ.ID()) {
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

type searchString struct {
	term    string
	compare Boolean
	types   map[zed.Type]bool
}

// NewSearchString is like NewSeach but handles the special case of matching
// field names in addition to string values.
func NewSearchString(term string) Evaluator {
	return &searchString{
		term:  term,
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
		var nameIter stringsearch.FieldNameIter
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

func (s *searchString) Eval(_ Context, val *zed.Value) *zed.Value {
	// Memoize the result of a search across the names in the
	// record columns for each unique record type.
	if s.searchType(val.Type) {
		return zed.True
	}
	if errMatch == val.Walk(func(typ zed.Type, body zcode.Bytes) error {
		if s.searchType(typ) {
			return errMatch
		}
		if zed.IsStringy(typ.ID()) &&
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
	expr Evaluator
}

func NewFilterApplier(e Evaluator) Applier {
	return &filterApplier{e}
}

func (f *filterApplier) Eval(ectx Context, this *zed.Value) *zed.Value {
	val, ok := EvalBool(ectx, this, f.expr)
	if ok {
		if zed.IsTrue(val.Bytes) {
			return this
		}
		return zed.Missing
	}
	return val
}

func (*filterApplier) String() string { return "filter" }

func (*filterApplier) Warning() string { return "" }

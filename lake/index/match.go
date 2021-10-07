package index

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/zson"
)

type match struct {
	rule       Rule
	lookupKeys []*zed.Record
}

type matcher func([]Rule) []match

type predicate struct {
	field field.Path
	value zed.Value
}

func compilePredicates(in []dag.IndexPredicate) (matcher, error) {
	predicates := make([]predicate, 0, len(in))
	for _, d := range in {
		zv, err := zson.ParsePrimitive(d.Value.Type, d.Value.Text)
		if err != nil {
			return nil, err
		}
		predicates = append(predicates, predicate{d.Key.Name, zv})
	}
	zctx := zed.NewContext()
	return func(rules []Rule) []match {
		var matches []match
		for _, r := range rules {
			if values := matchRule(zctx, r, predicates); len(values) > 0 {
				// XXX Maybe store RuleID in lookup table so we don't have to
				// loop through all the predicates for rules we've already
				// matched.
				matches = append(matches, match{r, values})
			}
		}
		return matches
	}, nil
}

func matchRule(zctx *zed.Context, rule Rule, predicates []predicate) []*zed.Record {
	var recs []*zed.Record
	for _, p := range predicates {
		// XXX support indexes with multiple keys #3162
		// and other rule types.
		if f, ok := rule.(*FieldRule); ok && p.field.Equal(f.Fields[0]) {
			rec, err := newLookupKey(zctx, rule, []zed.Value{p.value})
			if err != nil {
				panic(err)
			}
			recs = append(recs, rec)
		}
	}
	return recs
}

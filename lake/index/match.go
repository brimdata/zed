package index

import (
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/index"
	"github.com/brimdata/zed/zson"
)

type match struct {
	rule       Rule
	lookupKeys []index.KeyValue
}

type matcher func([]Rule) []match

func compilePredicates(in []dag.IndexPredicate) (matcher, error) {
	kvs := make([]index.KeyValue, 0, len(in))
	for _, d := range in {
		zv, err := zson.ParsePrimitive(d.Value.Type, d.Value.Text)
		if err != nil {
			return nil, err
		}
		kvs = append(kvs, index.KeyValue{Key: d.Key.Name, Value: zv})
	}
	return func(rules []Rule) []match {
		var matches []match
		for _, r := range rules {
			if values := matchRule(r, kvs); len(values) > 0 {
				// XXX Maybe store RuleID in lookup table so we don't have to
				// loop through all the predicates for rules we've already
				// matched.
				matches = append(matches, match{r, values})
			}
		}
		return matches
	}, nil
}

func matchRule(rule Rule, in []index.KeyValue) []index.KeyValue {
	var kvs []index.KeyValue
	for _, kv := range in {
		// XXX support indexes with multiple keys #3162
		// and other rule types.
		if f, ok := rule.(*FieldRule); ok && kv.Key.Equal(f.Fields[0]) {
			kvs = append(kvs, kv)
		}
	}
	return kvs
}

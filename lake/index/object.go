package index

import (
	"fmt"

	"github.com/brimdata/zed/pkg/storage"
	"github.com/segmentio/ksuid"
)

type Object struct {
	Rule Rule        `zed:"rule"`
	ID   ksuid.KSUID `zed:"id"`
}

func (o Object) String() string {
	//XXX data object looks like this:
	//	return fmt.Sprintf("%s %d record%s in %d data bytes", o.ID, o.Count, plural(int(o.Count)), o.RowSize)
	return ObjectName(o.Rule.RuleID(), o.ID)
}

func ObjectName(ruleID, id ksuid.KSUID) string {
	return fmt.Sprintf("%s/%s", ruleID, id)
}

func (o Object) Path(path *storage.URI) *storage.URI {
	return ObjectPath(path, o.Rule.RuleID(), o.ID)
}

func ObjectPath(path *storage.URI, ruleID, id ksuid.KSUID) *storage.URI {
	return path.AppendPath(ruleID.String(), id.String()+".zng")
}

type Map map[ksuid.KSUID]ObjectRules

func (m Map) Exists(object *Object) bool {
	return m.Lookup(object.Rule.RuleID(), object.ID) != nil
}

func (m Map) Lookup(ruleID, id ksuid.KSUID) *Object {
	if rules, ok := m[id]; ok {
		if object, ok := rules[ruleID]; ok {
			return object
		}
	}
	return nil
}

func (m Map) Insert(object *Object) {
	rules, ok := m[object.ID]
	if !ok {
		rules = make(map[ksuid.KSUID]*Object)
		m[object.ID] = rules
	}
	rules[object.Rule.RuleID()] = object
}

func (m Map) Delete(ruleID, id ksuid.KSUID) {
	if rules, ok := m[id]; ok {
		delete(rules, ruleID)
	}
}

func (m Map) All() []*Object {
	var objects []*Object
	for _, rules := range m {
		for _, object := range rules {
			objects = append(objects, object)
		}
	}
	return objects
}

func (m Map) Copy() Map {
	out := make(Map)
	for oid, rules := range m {
		out[oid] = make(ObjectRules)
		for rid, value := range rules {
			out[oid][rid] = value
		}
	}
	return out
}

type ObjectRules map[ksuid.KSUID]*Object

func (o ObjectRules) Rules() []Rule {
	rules := make([]Rule, 0, len(o))
	for _, object := range o {
		rules = append(rules, object.Rule)
	}
	return rules
}

func (o ObjectRules) Missing(rules []Rule) []Rule {
	var missing []Rule
	for _, r := range rules {
		if _, ok := o[r.RuleID()]; !ok {
			missing = append(missing, r)
		}
	}
	return missing
}

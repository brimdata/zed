package index

import (
	"fmt"

	"github.com/brimdata/zed/pkg/storage"
	"github.com/segmentio/ksuid"
)

type Object struct {
	Rule Rule        `zng:"rule"`
	ID   ksuid.KSUID `zng:"id"`
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
	return path.AppendPath(o.Rule.RuleID().String())
	//	return xObjectPath(path, o.Rule, o.ID)
}

type Map map[ksuid.KSUID]ObjectRules

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
		delete(rules, id)
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

type ObjectRules map[ksuid.KSUID]*Object

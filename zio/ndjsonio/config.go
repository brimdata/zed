package ndjsonio

import (
	"fmt"
)

// A Rule contains one or more matches and the name of a descriptor
// key (in the companion Descriptors map).
type Rule struct {
	Name       string `json:"name"`
	Value      string `json:"value"`
	Descriptor string `json:"descriptor"`
}

// A TypeConfig contains a map of Descriptors, keyed by name, and a
// list of rules defining which records should be mapped into which
// descriptor.
type TypeConfig struct {
	Descriptors map[string][]interface{} `json:"descriptors"`
	Rules       []Rule                   `json:"rules"`
}

func hasField(name string, columns []interface{}) bool {
	for _, colmap := range columns {
		col := colmap.(map[string]interface{})
		if col["name"] == name {
			return true
		}
	}
	return false
}

// Validate validates a typing config.
func (conf TypeConfig) Validate() error {
	for _, rule := range conf.Rules {
		d, ok := conf.Descriptors[rule.Descriptor]
		if !ok {
			return fmt.Errorf("rule %s=%s uses descriptor %s that does not exist", rule.Name, rule.Value, rule.Descriptor)
		}
		if !hasField(rule.Name, d) {
			return fmt.Errorf("rule %s refers to field %s that is not present in descriptor", rule.Descriptor, rule.Name)
		}

	}
	for name, desc := range conf.Descriptors {
		for i, d := range desc {
			col, ok := d.(map[string]interface{})
			if !ok {
				return fmt.Errorf("descriptor %s has invalid structure in element %d", name, i)
			}
			if col["name"] == "ts" && col["type"] != "time" {
				return fmt.Errorf("descriptor %s has field ts with wrong type %s", name, col["type"])
			}
		}
		col := desc[0].(map[string]interface{})
		if col["name"] != "_path" {
			return fmt.Errorf("descriptor %s does not have _path as first column", name)
		}
	}
	return nil
}

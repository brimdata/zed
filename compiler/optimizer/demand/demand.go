package demand

type Demand interface {
	isDemand()
}

func (demand all) isDemand()  {}
func (demand keys) isDemand() {}

type all struct{}
type keys map[string]Demand // No empty values.

func IsValid(demand Demand) bool {
	switch demand := demand.(type) {
	case nil:
		return false
	case all:
		return true
	case keys:
		for _, v := range demand {
			if !IsValid(v) || IsNone(v) {
				return false
			}
		}
		return true
	default:
		panic("Unreachable")
	}
}

func None() Demand {
	return keys{}
}

func All() Demand {
	return all{}
}

func IsNone(demand Demand) bool {
	switch demand := demand.(type) {
	case all:
		return false
	case keys:
		return len(demand) == 0
	default:
		panic("Unreachable")
	}
}

func IsAll(demand Demand) bool {
	_, ok := demand.(all)
	return ok
}

func Key(key string, value Demand) Demand {
	if IsNone(value) {
		return value
	}
	return keys{key: value}
}

func Union(a Demand, b Demand) Demand {
	if _, ok := a.(all); ok {
		return a
	}
	if _, ok := b.(all); ok {
		return b
	}

	{
		a, b := a.(keys), b.(keys)

		demand := make(keys, len(a)+len(b))
		for k, v := range a {
			demand[k] = v
		}
		for k, v := range b {
			if v2, ok := a[k]; ok {
				demand[k] = Union(v, v2)
			} else {
				demand[k] = v
			}
		}
		return demand
	}
}

func GetKey(demand Demand, key string) Demand {
	switch demand := demand.(type) {
	case all:
		return demand
	case keys:
		if value, ok := demand[key]; ok {
			return value
		}
		return None()
	default:
		panic("Unreachable")
	}
}

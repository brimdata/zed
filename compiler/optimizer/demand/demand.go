package demand

type Demand interface {
	isDemand()
}

func (demand All) isDemand()  {}
func (demand Keys) isDemand() {}

type All struct{}
type Keys map[string]Demand // No empty values.

func None() Demand {
	return Keys(make(map[string]Demand, 0))
}

func IsValid(demand Demand) bool {
	switch demand := demand.(type) {
	case All:
		return true
	case Keys:
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

func IsNone(demand Demand) bool {
	switch demand := demand.(type) {
	case All:
		return false
	case Keys:
		return len(demand) == 0
	default:
		panic("Unreachable")
	}
}

func Key(key string, value Demand) Demand {
	if IsNone(value) {
		return value
	}
	demand := Keys(make(map[string]Demand, 1))
	demand[key] = value
	return demand
}

func Union(a Demand, b Demand) Demand {
	if _, ok := a.(All); ok {
		return a
	}
	if _, ok := b.(All); ok {
		return b
	}

	{
		a, b := a.(Keys), b.(Keys)

		demand := Keys(make(map[string]Demand, len(a)+len(b)))
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
	case All:
		return demand
	case Keys:
		if value, ok := demand[key]; ok {
			return value
		}
		return None()
	default:
		panic("Unreachable")
	}
}

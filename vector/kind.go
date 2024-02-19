package vector

type Kind int

const (
	KindInvalid = 0
	KindInt     = 1
	KindUint    = 2
	KindFloat   = 3
	KindString  = 4
	KindBytes   = 5
	KindType    = 6
	KindFlat    = 0 << 3
	KindDict    = 1 << 3
	KindView    = 2 << 3
	KindConst   = 3 << 3
	KindWidth   = 5
)

//XXX store bytes as Go string?

func KindOf(v Any) Kind {
	switch v := v.(type) {
	case *Int:
		return KindInt
	case *Uint:
		return KindUint
	case *Float:
		return KindFloat
	case *Bytes:
		return KindBytes
	case *String:
		return KindString
	case *View:
		inner := KindOf(v.Any)
		if inner == KindInvalid {
			return KindInvalid
		}
		return KindView | inner
	case *Dict:
		inner := KindOf(v.Any)
		if inner == KindInvalid {
			return KindInvalid
		}
		return KindDict | inner
	default:
		return KindInvalid
	}
}

func KindBinary(lhs, rhs Any) Kind {
	left := KindOf(lhs)
	if left == KindInvalid {
		return KindInvalid
	}
	right := KindOf(rhs)
	if right == KindInvalid {
		return KindInvalid
	}
	return left<<KindWidth | right
}

func KindLeft(kind Kind) Kind {
	return kind >> KindWidth
}

func KindRight(kind Kind) Kind {
	return kind & ((1 << KindWidth) - 1)
}

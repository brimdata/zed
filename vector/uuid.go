package vector

import (
	"uuid"

	"github.com/superdb/super"
	"github.com/superdb/super/scode"
)

type UUID struct {
	Values []uuid.UUID
}

func NewUUID(values []uuid.UUID) *UUID {
	return &UUID{values}
}

func (*UUID) Kind() Kind {
	return KindUUID
}

func (*UUID) Type() super.Type {
	return super.TypeUUID
}

func (u *UUID) Len() uint32 {
	return uint32(len(u.Values))
}

func (u *UUID) Serialize(b *scode.Builder, slot uint32) {
	b.Append(super.EncodeUUID(u.Values[slot]))
}

// UUIDValue returns the UUID at slot of val, which must be of kind KindUUID.
func UUIDValue(val Any, slot uint32) uuid.UUID {
	switch val := val.(type) {
	case *UUID:
		return val.Values[slot]
	case *Const:
		return UUIDValue(val.Any, 0)
	case *Dict:
		slot = uint32(val.Index[slot])
		return val.Any.(*UUID).Values[slot]
	case *View:
		slot = val.Index[slot]
		return UUIDValue(val.Any, slot)
	}
	panic(val)
}

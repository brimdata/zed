package function

import (
	"uuid"

	"github.com/superdb/super/vector"
)

// UUID implements uuid(), which returns a new UUID (version 7) for each
// input value.  Conversions between uuid, string, and bytes are done with cast.
type UUID struct{}

func (*UUID) needsInput() {}

func (*UUID) Call(args ...vector.Any) vector.Any {
	vals := make([]uuid.UUID, args[0].Len())
	for k := range vals {
		vals[k] = uuid.NewV7()
	}
	return vector.NewUUID(vals)
}

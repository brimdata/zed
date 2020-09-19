package units

// XXX this should be unified with cmd/zapi/format

import (
	"github.com/alecthomas/units"
)

// Bytes implements flag.Value
type Bytes units.Base2Bytes

func String(bytes Bytes) string {
	return units.Base2Bytes(bytes).String()
}

func (b Bytes) String() string {
	return String(b)
}

func (b *Bytes) Set(s string) error {
	bytes, err := units.ParseStrictBytes(s)
	if err != nil {
		return err
	}
	*b = Bytes(bytes)
	return nil
}

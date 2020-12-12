package auto

import (
	"fmt"

	"github.com/alecthomas/units"
)

type Bytes struct {
	defStr string
	Bytes  units.Base2Bytes
}

func (b Bytes) String() string {
	if b.defStr != "" {
		return b.defStr
	}
	return b.Bytes.String()
}

func (b *Bytes) Set(s string) error {
	b.defStr = ""
	b.Bytes = 0
	bytes, err := units.ParseStrictBytes(s)
	if err != nil {
		return err
	}
	b.Bytes = units.Base2Bytes(bytes)
	return nil
}

func NewBytes(def uint64) Bytes {
	bytes := units.Base2Bytes(def)
	return Bytes{
		defStr: fmt.Sprintf("auto(%s)", bytes),
		Bytes:  bytes,
	}
}

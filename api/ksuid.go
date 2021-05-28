package api

import (
	"encoding/hex"
	"fmt"

	"github.com/segmentio/ksuid"
)

type KSUID ksuid.KSUID

func (k *KSUID) UnmarshalText(text []byte) error {
	id, err := ParseKSUID(string(text))
	*k = KSUID(id)
	return err
}

func (k KSUID) MarshalText() ([]byte, error) {
	return ksuid.KSUID(k).MarshalText()
}

func ParseKSUID(s string) (ksuid.KSUID, error) {
	// Check if this is a cut-and-paste from ZNG, which encodes
	// the 20-byte KSUID as a 40 character hex string with 0x prefix.
	var id ksuid.KSUID
	var err error
	if len(s) == 42 && s[0:2] == "0x" {
		var b []byte
		b, err := hex.DecodeString(s[2:])
		if err == nil {
			id, err = ksuid.FromBytes(b)
		}
	} else {
		id, err = ksuid.Parse(s)
	}
	if err != nil {
		return ksuid.Nil, fmt.Errorf("invalid commit ID: %s", s)
	}
	return id, nil
}

func ParseKSUIDs(in []string) (ids []ksuid.KSUID, err error) {
	ids = make([]ksuid.KSUID, len(in))
	for i, s := range in {
		if ids[i], err = ParseKSUID(s); err != nil {
			return nil, err
		}
	}
	return ids, nil
}

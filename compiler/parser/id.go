package parser

import (
	"encoding/hex"
	"fmt"

	"github.com/segmentio/ksuid"
)

func ParseID(s string) (ksuid.KSUID, error) {
	// Check if this is a cut-and-paste from ZNG, which encodes
	// the 20-byte KSUID as a 40 character hex string with 0x prefix.
	var id ksuid.KSUID
	if len(s) == 42 && s[0:2] == "0x" {
		b, err := hex.DecodeString(s[2:])
		if err != nil {
			return ksuid.Nil, fmt.Errorf("illegal hex tag: %s", s)
		}
		id, err = ksuid.FromBytes(b)
		if err != nil {
			return ksuid.Nil, fmt.Errorf("illegal hex tag: %s", s)
		}
	} else {
		var err error
		id, err = ksuid.Parse(s)
		if err != nil {
			return ksuid.Nil, fmt.Errorf("%s: invalid commit ID", s)
		}
	}
	return id, nil
}

func ParseIDs(ss []string) ([]ksuid.KSUID, error) {
	ids := make([]ksuid.KSUID, 0, len(ss))
	for _, s := range ss {
		id, err := ParseID(s)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

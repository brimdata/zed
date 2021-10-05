package api

import (
	"github.com/brimdata/zed/lakeparse"
	"github.com/segmentio/ksuid"
)

type KSUID ksuid.KSUID

func (k *KSUID) UnmarshalText(text []byte) error {
	id, err := lakeparse.ParseID(string(text))
	*k = KSUID(id)
	return err
}

func (k KSUID) MarshalText() ([]byte, error) {
	return ksuid.KSUID(k).MarshalText()
}

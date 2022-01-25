package zngio

import (
	"bytes"
	"errors"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zqe"
)

// Validate checks that val.Bytes is structurally consistent
// with val.Type.  It does not check that the actual leaf
// values when parsed are type compatible with the leaf types.
func Validate(val *zed.Value) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = zqe.RecoverError(r)
		}
	}()
	return val.Walk(func(typ zed.Type, body zcode.Bytes) error {
		if typset, ok := typ.(*zed.TypeSet); ok {
			if err := checkSet(typset, body); err != nil {
				return err
			}
			return zed.SkipContainer
		}
		if typ, ok := typ.(*zed.TypeEnum); ok {
			if err := checkEnum(typ, body); err != nil {
				return err
			}
			return zed.SkipContainer
		}
		return nil
	})
}

func checkSet(typ *zed.TypeSet, body zcode.Bytes) error {
	if body == nil {
		return nil
	}
	it := body.Iter()
	var prev zcode.Bytes
	for !it.Done() {
		tagAndBody := it.NextTagAndBody()
		if prev != nil {
			switch bytes.Compare(prev, tagAndBody) {
			case 0:
				return errors.New("invalid ZNG: duplicate set element")
			case 1:
				return errors.New("invalid ZNG: set elements not sorted")
			}
		}
		prev = tagAndBody
	}
	return nil
}

func checkEnum(typ *zed.TypeEnum, body zcode.Bytes) error {
	if body == nil {
		return nil
	}
	if selector := zed.DecodeUint(body); int(selector) >= len(typ.Symbols) {
		return errors.New("enum selector out of range")
	}
	return nil
}

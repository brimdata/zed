package zson

import (
	"github.com/brimsec/zq/zng"
)

func Implied(typ zng.Type) bool {
	switch typ.(type) {
	case *zng.TypeOfInt64, *zng.TypeOfTime, *zng.TypeOfFloat64, *zng.TypeOfBool, *zng.TypeOfBytes, *zng.TypeOfString, *zng.TypeOfIP, *zng.TypeOfNet, *zng.TypeOfType:
		return true
	}
	return false
}

func SelfDescribing(typ zng.Type) bool {
	if Implied(typ) {
		return true
	}
	switch typ := typ.(type) {
	case *zng.TypeRecord, *zng.TypeArray, *zng.TypeSet, *zng.TypeMap:
		return true
	case *zng.TypeAlias:
		return SelfDescribing(typ.Type)
	}
	return false
}

package reducer

//XXX in new model, need to do a semantic check on the reducers since they
// are compiled at runtime and you don't want to run a long time then catch
// the error that could have been caught earlier

import (
	"errors"

	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zq"
)

var (
	ErrUnsupportedType = errors.New("unsupported type")
)

type Interface interface {
	Consume(*zq.Record)
	Result() zeek.Value
}

// Result returns the Interface's result or a zeek.Unset value if r is nil.
func Result(r Interface) zeek.Value {
	if r == nil {
		return &zeek.Unset{}
	}
	return r.Result()
}

type Stats struct {
	TypeMismatch  int64
	FieldNotFound int64
}

type Reducer struct {
	Stats
}

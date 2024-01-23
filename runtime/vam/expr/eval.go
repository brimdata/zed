package expr

import "github.com/brimdata/zed/vector"

type Evaluator interface {
	Eval(vector.Any) (vector.Any, *vector.Error)
}

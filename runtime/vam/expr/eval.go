package expr

import "github.com/brimdata/zed/vector"

type Evaluator interface {
	Eval(vector.Any) (val vector.Any, err vector.Any)
}

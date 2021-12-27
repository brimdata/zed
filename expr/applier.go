package expr

import (
	"fmt"
)

type Applier interface {
	Evaluator
	fmt.Stringer
	Warning() string
}

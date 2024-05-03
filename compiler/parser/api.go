package parser

import (
	"errors"

	"github.com/brimdata/zed/compiler/ast"
	"go.uber.org/multierr"
)

var ParsingError = errors.New("error parsing Zed")

// ParseZed calls ConcatSource followed by Parse.  If Parse fails, it calls
// ImproveError.
func ParseZed(src string) (ast.Seq, error) {
	if src == "" {
		src = "*"
	}
	p, err := Parse("", []byte(src))
	if err != nil {
		return nil, convertErrors(err)
	}
	return sliceOf[ast.Op](p), nil
}

func convertErrors(err error) error {
	errs, ok := err.(errList)
	if !ok {
		return err
	}
	for i, err := range errs {
		if pe, ok := err.(*parserError); ok {
			errs[i] = ast.NewError(ParsingError, pe.pos.offset, -1)
		}
	}
	return multierr.Combine(errs...)
}

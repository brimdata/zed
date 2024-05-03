package parser

import (
	"fmt"

	"github.com/brimdata/zed/compiler/ast"
	"go.uber.org/multierr"
)

// ParseZed calls ConcatSource followed by Parse.  If Parse fails, it calls
// ImproveError.
func ParseZed(filenames []string, src string) (ast.Seq, *SourceSet, error) {
	s, err := ConcatSource(filenames, src)
	if err != nil {
		return nil, nil, err
	}
	p, err := Parse("", s.Contents)
	if err != nil {
		return nil, s, convertErrors(err)
	}
	return sliceOf[ast.Op](p), s, nil
}

type ParseError int

var _ PositionalError = (ParseError)(0)

func (p ParseError) Error() string   { return fmt.Sprintf("%d: %s", p, p.Message()) }
func (p ParseError) Message() string { return "error parsing Zed" }
func (p ParseError) Pos() int        { return int(p) }
func (p ParseError) End() int        { return -1 }

func convertErrors(err error) error {
	errs, ok := err.(errList)
	if !ok {
		return err
	}
	for i, err := range errs {
		if pe, ok := err.(*parserError); ok {
			errs[i] = ParseError(pe.pos.offset)
		}
	}
	return multierr.Combine(errs...)
}

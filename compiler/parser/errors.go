package parser

import (
	"fmt"
	"strings"

	"github.com/brimdata/zed/compiler/ast"
	"go.uber.org/multierr"
)

type PositionalError interface {
	error
	ast.Node
	Message() string
}

// LocalizeErrors returns a list of localized errors if the error can be
// localized, else returns the original error.
func (s *SourceSet) LocalizeError(errs error) error {
	var list LocalizedErrors
	for _, err := range multierr.Errors(errs) {
		if perr, ok := err.(PositionalError); ok {
			list = append(list, newLocalizedError(s, perr))
		} else {
			return errs
		}
	}
	if len(list) > 0 {
		return list
	}
	return nil
}

// LocalizedError is a parse error with nice formatting.  It includes the source code
// line containing the error.
type LocalizedError struct {
	Kind     string   `json:"kind" unpack:""`
	Filename string   `json:"filename"`
	Line     string   `json:"line"` // contains no newlines
	Open     Position `json:"open"`
	Close    Position `json:"close"`
	Msg      string   `json:"error"`
}

var _ PositionalError = (*LocalizedError)(nil)

func newLocalizedError(s *SourceSet, perr PositionalError) *LocalizedError {
	startPos := perr.Pos()
	src := s.SourceOf(startPos)
	filename, start := src.Position(startPos)
	_, end := src.Position(perr.End())
	return &LocalizedError{
		Kind:     "LocalizedError",
		Filename: filename,
		Open:     start,
		Close:    end,
		Line:     src.LineOfPos(s, startPos),
		Msg:      perr.Message(),
	}
}

func (e *LocalizedError) Message() string { return e.Msg }
func (e *LocalizedError) Pos() int        { return e.Open.Pos }
func (e *LocalizedError) End() int        { return e.Close.Pos }

func (e *LocalizedError) Error() string {
	var b strings.Builder
	b.WriteString(e.Msg)
	b.WriteString(" (")
	if e.Filename != "" {
		fmt.Fprintf(&b, "%s: ", e.Filename)
	}
	if e.Open.Line >= 1 {
		fmt.Fprintf(&b, "line %d, ", e.Open.Line)
	}
	fmt.Fprintf(&b, "column %d):\n", e.Open.Column)
	b.WriteString(e.errorContext())
	return b.String()
}

func (e *LocalizedError) errorContext() string {
	var b strings.Builder
	b.WriteString(e.Line + "\n")
	if e.Close.IsValid() {
		e.spanError(&b)
	} else {
		col := e.Open.Column - 1
		for k := 0; k < col; k++ {
			if k >= col-4 && k != col-1 {
				b.WriteByte('=')
			} else {
				b.WriteByte(' ')
			}
		}
		b.WriteString("^ ===")
	}
	return b.String()
}

func (e *LocalizedError) spanError(b *strings.Builder) {
	col := e.Open.Column - 1
	b.WriteString(strings.Repeat(" ", col))
	end := len(e.Line) - col
	if e.Open.Line == e.Close.Line {
		end = e.Close.Column - col
	}
	b.WriteString(strings.Repeat("~", end))
}

type LocalizedErrors []*LocalizedError

func (e LocalizedErrors) Error() string {
	var b strings.Builder
	for i, err := range e {
		if i != 0 {
			b.WriteByte('\n')
		}
		b.WriteString(err.Error())
	}
	return b.String()
}

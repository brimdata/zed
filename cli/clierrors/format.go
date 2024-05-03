package clierrors

import (
	"errors"
	"fmt"
	"strings"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/parser"
	"go.uber.org/multierr"
)

func Format(set *parser.SourceSet, err error) error {
	if err == nil {
		return err
	}
	var errs []error
	for _, err := range multierr.Errors(err) {
		if asterr, ok := err.(*ast.Error); ok {
			err = formatASTError(set, asterr)
		}
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func formatASTError(set *parser.SourceSet, err *ast.Error) error {
	src := set.SourceOf(err.Pos)
	start := src.Position(err.Pos)
	end := src.Position(err.End)
	var b strings.Builder
	fmt.Fprintf(&b, "%s (", err)
	if src.Filename != "" {
		fmt.Fprintf(&b, "%s: ", src.Filename)
	}
	line := src.LineOfPos(set.Contents, err.Pos)
	fmt.Fprintf(&b, "line %d, column %d):\n%s\n", start.Line, start.Column, line)
	if end.IsValid() {
		formatSpanError(&b, line, start, end)
	} else {
		formatPointError(&b, start)
	}
	return errors.New(b.String())
}

func formatSpanError(b *strings.Builder, line string, start, end parser.Position) {
	col := start.Column - 1
	b.WriteString(strings.Repeat(" ", col))
	n := len(line) - col
	if start.Line == end.Line {
		n = end.Column - col
	}
	b.WriteString(strings.Repeat("~", n))
}

func formatPointError(b *strings.Builder, start parser.Position) {
	col := start.Column - 1
	for k := 0; k < col; k++ {
		if k >= col-4 && k != col-1 {
			b.WriteByte('=')
		} else {
			b.WriteByte(' ')
		}
	}
	b.WriteString("^ ===")
}

package parser

import (
	"fmt"
	"os"
	"strings"
)

// ParseZed calls ConcatSource followed by Parse.  If Parse fails, it calls
// ImproveError.
func ParseZed(filenames []string, src string) (interface{}, error) {
	src, srcInfo, err := ConcatSource(filenames, src)
	if err != nil {
		return nil, err
	}
	p, err := Parse("", []byte(src))
	if err != nil {
		return nil, ImproveError(err, src, srcInfo)
	}
	return p, nil
}

// SourceInfo holds source file offsets.
type SourceInfo struct {
	filename string
	start    int
	end      int
}

// ConcatSource concatenates the source files in filenames followed by src,
// returning the result and a corresponding slice of SourceInfos.
func ConcatSource(filenames []string, src string) (string, []SourceInfo, error) {
	var b strings.Builder
	var sis []SourceInfo
	for _, f := range filenames {
		bb, err := os.ReadFile(f)
		if err != nil {
			return "", nil, err
		}
		start := b.Len()
		b.Write(bb)
		sis = append(sis, SourceInfo{f, start, b.Len()})
		b.WriteByte('\n')
	}
	start := b.Len()
	b.WriteString(src)
	sis = append(sis, SourceInfo{"", start, b.Len()})
	if b.Len() == 0 {
		return "*", nil, nil
	}
	return b.String(), sis, nil
}

// ImproveError tries to improve an error from Parse.  err is the error.  src is
// the source code for which Parse return err.  If src came from ConcatSource,
// sis is the corresponding slice of SourceInfo; otherwise, sis is nil.
func ImproveError(err error, src string, sis []SourceInfo) error {
	el, ok := err.(errList)
	if !ok || len(el) != 1 {
		return err
	}
	pe, ok := el[0].(*parserError)
	if !ok {
		return err
	}
	return NewError(src, sis, pe.pos.offset)
}

// Error is a parse error with nice formatting.  It includes the source code
// line containing the error.
type Error struct {
	Offset int // offset into original source code

	filename string // omitted from formatting if ""
	lineNum  int    // zero-based; omitted from formatting if negative

	line   string // contains no newlines
	column int    // zero-based
}

// NewError returns an Error.  src is the source code containing the error.  If
// src came from ConcatSource, sis is the corresponding slice of SourceInfo;
// otherwise, src is nil.  offset is the offset of the error within src.
func NewError(src string, sis []SourceInfo, offset int) error {
	var filename string
	for _, si := range sis {
		if offset < si.end {
			filename = si.filename
			offset -= si.start
			src = src[si.start:si.end]
			break
		}
	}
	lineNum := -1
	if filename != "" || strings.Count(src, "\n") > 0 {
		lineNum = strings.Count(src[:offset], "\n")
	}
	column := offset
	if i := strings.LastIndexByte(src[:offset], '\n'); i != -1 {
		column -= i + 1
		src = src[i+1:]
	}
	if i := strings.IndexByte(src, '\n'); i != -1 {
		src = src[:i]
	}
	return &Error{
		Offset:   offset,
		filename: filename,
		lineNum:  lineNum,
		line:     src,
		column:   column,
	}
}

func (e *Error) Error() string {
	var b strings.Builder
	b.WriteString("error parsing Zed ")
	if e.filename != "" {
		fmt.Fprintf(&b, "in %s ", e.filename)
	}
	b.WriteString("at ")
	if e.lineNum >= 0 {
		fmt.Fprintf(&b, "line %d, ", e.lineNum+1)
	}
	fmt.Fprintf(&b, "column %d:\n%s\n", e.column+1, e.line)
	for k := 0; k < e.column; k++ {
		if k >= e.column-4 && k != e.column-1 {
			b.WriteByte('=')
		} else {
			b.WriteByte(' ')
		}
	}
	b.WriteByte('^')
	b.WriteString(" ===")
	return b.String()
}

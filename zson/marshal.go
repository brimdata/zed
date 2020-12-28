package zson

import (
	"strings"

	"github.com/brimsec/zq/zng/resolver"
)

const (
	StyleNone    = resolver.StyleNone
	StyleSimple  = resolver.StyleSimple
	StylePackage = resolver.StylePackage
	StyleFull    = resolver.StyleFull
)

func Marshal(v interface{}) (string, error) {
	return NewMarshaler().Marshal(v)
}

type MarshalContext struct {
	*resolver.MarshalContext
	formatter *Formatter
}

func NewMarshaler() *MarshalContext {
	return NewMarshalerIndent(0)
}

func NewMarshalerIndent(indent int) *MarshalContext {
	return &MarshalContext{
		MarshalContext: resolver.NewMarshaler(),
		formatter:      NewFormatter(indent),
	}
}

func NewMarshalerWithContext(zctx *resolver.Context) *MarshalContext {
	return &MarshalContext{
		MarshalContext: resolver.NewMarshalerWithContext(zctx),
	}
}

func (m *MarshalContext) Marshal(v interface{}) (string, error) {
	zv, err := m.MarshalContext.Marshal(v)
	if err != nil {
		return "", err
	}
	return m.formatter.Format(zv)
}

func (m *MarshalContext) MarshalCustom(names []string, fields []interface{}) (string, error) {
	rec, err := m.MarshalContext.MarshalCustom(names, fields)
	if err != nil {
		return "", err
	}
	return m.formatter.FormatRecord(rec)
}

type UnmarshalContext struct {
	*resolver.UnmarshalContext
	zctx     *resolver.Context
	analyzer Analyzer
	builder  *Builder
}

func NewUnmarshaler() *UnmarshalContext {
	return &UnmarshalContext{
		UnmarshalContext: resolver.NewUnmarshaler(),
		zctx:             resolver.NewContext(),
		analyzer:         NewAnalyzer(),
		builder:          NewBuilder(),
	}
}

func Unmarshal(zson string, v interface{}) error {
	return NewUnmarshaler().Unmarshal(zson, v)
}

func (u *UnmarshalContext) Unmarshal(zson string, v interface{}) error {
	parser, err := NewParser(strings.NewReader(zson))
	if err != nil {
		return err
	}
	ast, err := parser.ParseValue()
	if err != nil {
		return err
	}
	val, err := u.analyzer.ConvertValue(u.zctx, ast)
	if err != nil {
		return err
	}
	zv, err := u.builder.Build(val)
	if err != nil {
		return nil
	}
	return u.UnmarshalContext.Unmarshal(zv, v)
}

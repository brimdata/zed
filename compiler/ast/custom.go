package ast

import (
	"github.com/brimdata/zed/field"
)

// These ast types are to support the custom procs used by the indexer.
// We should have the indexer use genric components and figure out what
// custom functionality should be transferred into the Zed core.  This will
// allow us to get rid of the custom hook that originally went in for
// reasons that are no longer relevant.  Having the definitions here allows
// the semantic pass to handle them properly.  See issue #2258.

type FieldCutter struct {
	Kind  string     `json:"kind" unpack:""`
	Field field.Path `json:"field"`
	Out   field.Path `json:"out"`
}

func (*FieldCutter) ProcAST() {}

type TypeSplitter struct {
	Kind     string     `json:"kind" unpack:""`
	Key      field.Path `json:"key"`
	TypeName string     `json:"type_name"`
}

func (t *TypeSplitter) ProcAST() {}

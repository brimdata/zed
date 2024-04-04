package zed

import (
	"github.com/brimdata/zed/compiler/ast/node"
)

type Value interface {
	node.Node
	valueNode()
}

type ImpliedValue struct {
	node.Base
	Kind string `json:"kind" unpack:""`
	Of   Any    `json:"of"`
}

type DefValue struct {
	node.Base
	Kind     string `json:"kind" unpack:""`
	Of       Any    `json:"of"`
	TypeName string `json:"type_name"`
}

type CastValue struct {
	node.Base
	Kind string `json:"kind" unpack:""`
	Of   Value  `json:"of"`
	Type Type   `json:"type"`
}

func (*ImpliedValue) valueNode() {}
func (*DefValue) valueNode()     {}
func (*CastValue) valueNode()    {}

type Any interface {
	node.Node
	anyNode()
}

type (
	Primitive struct {
		node.Base
		Kind string `json:"kind" unpack:""`
		Type string `json:"type"`
		Text string `json:"text"`
	}
	Record struct {
		node.Base
		Kind   string  `json:"Kind" unpack:""`
		Fields []Field `json:"fields"`
	}
	Field struct {
		node.Base
		Name  string `json:"name"`
		Value Value  `json:"value"`
	}
	Array struct {
		node.Base
		Kind     string  `json:"kind" unpack:""`
		Elements []Value `json:"elements"`
	}
	Set struct {
		node.Base
		Kind     string  `json:"kind" unpack:""`
		Elements []Value `json:"elements"`
	}
	Enum struct {
		node.Base
		Kind string `json:"kind" unpack:""`
		Name string `json:"name"`
	}
	Map struct {
		node.Base
		Kind    string  `json:"kind" unpack:""`
		Entries []Entry `json:"entries"`
	}
	Entry struct {
		Key   Value `json:"key"`
		Value Value `json:"value"`
	}
	TypeValue struct {
		node.Base
		Kind  string `json:"kind" unpack:""`
		Value Type   `json:"value"`
	}
	Error struct {
		node.Base
		Kind  string `json:"kind" unpack:""`
		Value Value  `json:"value"`
	}
)

func (*Primitive) anyNode() {}
func (*Record) anyNode()    {}
func (*Array) anyNode()     {}
func (*Set) anyNode()       {}
func (*Enum) anyNode()      {}
func (*Map) anyNode()       {}
func (*TypeValue) anyNode() {}
func (*Error) anyNode()     {}

func (*Primitive) ExprAST() {}
func (*TypeValue) ExprAST() {}

func (*Primitive) ExprDAG() {}
func (*TypeValue) ExprDAG() {}

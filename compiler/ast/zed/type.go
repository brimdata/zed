package zed

import (
	"github.com/brimdata/zed/compiler/ast/node"
)

type Type interface {
	node.Node
	typeNode()
}

type (
	TypePrimitive struct {
		node.Base
		Kind string `json:"kind" unpack:""`
		Name string `json:"name"`
	}
	TypeRecord struct {
		node.Base
		Kind   string      `json:"kind" unpack:""`
		Fields []TypeField `json:"fields"`
	}
	TypeField struct {
		node.Base
		Name string `json:"name"`
		Type Type   `json:"type"`
	}
	TypeArray struct {
		node.Base
		Kind string `json:"kind" unpack:""`
		Type Type   `json:"type"`
	}
	TypeSet struct {
		node.Base
		Kind string `json:"kind" unpack:""`
		Type Type   `json:"type"`
	}
	TypeUnion struct {
		node.Base
		Kind  string `json:"kind" unpack:""`
		Types []Type `json:"types"`
	}
	TypeEnum struct {
		node.Base
		Kind    string   `json:"kind" unpack:""`
		Symbols []string `json:"symbols"`
	}
	TypeMap struct {
		node.Base
		Kind    string `json:"kind" unpack:""`
		KeyType Type   `json:"key_type"`
		ValType Type   `json:"val_type"`
	}
	TypeNull struct {
		node.Base
		Kind string `json:"kind" unpack:""`
	}
	TypeError struct {
		node.Base
		Kind string `json:"kind" unpack:""`
		Type Type   `json:"type"`
	}
	TypeName struct {
		node.Base
		Kind string `json:"kind" unpack:""`
		Name string `json:"name"`
	}
	TypeDef struct {
		node.Base
		Kind string `json:"kind" unpack:""`
		Name string `json:"name"`
		Type Type   `json:"type"`
	}
)

func (*TypePrimitive) typeNode() {}
func (*TypeRecord) typeNode()    {}
func (*TypeArray) typeNode()     {}
func (*TypeSet) typeNode()       {}
func (*TypeUnion) typeNode()     {}
func (*TypeEnum) typeNode()      {}
func (*TypeMap) typeNode()       {}
func (*TypeNull) typeNode()      {}
func (*TypeError) typeNode()     {}
func (*TypeName) typeNode()      {}
func (*TypeDef) typeNode()       {}

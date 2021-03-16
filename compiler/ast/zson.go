package ast

type Value interface {
	valueNode()
}

type ImpliedValue struct {
	Kind string `json:"kind" unpack:""`
	Of   Any    `json:"of"`
}

type DefValue struct {
	Kind     string `json:"kind" unpack:""`
	Of       Any    `json:"of"`
	TypeName string `json:"type_name"`
}

type CastValue struct {
	Kind string `json:"kind" unpack:""`
	Of   Value  `json:"of"`
	Type Type   `json:"type"`
}

func (*ImpliedValue) valueNode() {}
func (*DefValue) valueNode()     {}
func (*CastValue) valueNode()    {}

type Any interface {
	anyNode()
}

type (
	Primitive struct {
		Kind string `json:"kind" unpack:""`
		Type string `json:"type"`
		Text string `json:"text"`
	}
	Record struct {
		Kind   string  `json:"Kind" unpack:""`
		Fields []Field `json:"fields"`
	}
	Field struct {
		Name  string `json:"name"`
		Value Value  `json:"value"`
	}
	Array struct {
		Kind     string  `json:"kind" unpack:""`
		Elements []Value `json:"elements"`
	}
	Set struct {
		Kind     string  `json:"kind" unpack:""`
		Elements []Value `json:"elements"`
	}
	Enum struct {
		Kind string `json:"kind" unpack:""`
		Name string `json:"name"`
	}
	Map struct {
		Kind    string  `json:"kind" unpack:""`
		Entries []Entry `json:"entries"`
	}
	Entry struct {
		Key   Value `json:"key"`
		Value Value `json:"value"`
	}
	TypeValue struct {
		Kind  string `json:"kind" unpack:""`
		Value Type   `json:"value"`
	}
)

func (*Primitive) anyNode() {}
func (*Record) anyNode()    {}
func (*Array) anyNode()     {}
func (*Set) anyNode()       {}
func (*Enum) anyNode()      {}
func (*Map) anyNode()       {}
func (*TypeValue) anyNode() {}

type Type interface {
	typeNode()
}

type (
	TypePrimitive struct {
		Kind string `json:"kind" unpack:""`
		Name string `json:"name"`
	}
	TypeRecord struct {
		Kind   string      `json:"kind" unpack:""`
		Fields []TypeField `json:"fields"`
	}
	TypeField struct {
		Name string `json:"name"`
		Type Type   `json:"type"`
	}
	TypeArray struct {
		Kind string `json:"kind" unpack:""`
		Type Type   `json:"type"`
	}
	TypeSet struct {
		Kind string `json:"kind" unpack:""`
		Type Type   `json:"type"`
	}
	TypeUnion struct {
		Kind  string `json:"kind" unpack:""`
		Types []Type `json:"types"`
	}
	// Enum has just the elements and relies on the semantic checker
	// to determine a type from the decorator either within or from above.
	TypeEnum struct {
		Kind     string  `json:"kind" unpack:""`
		Elements []Field `json:"elements"`
	}
	TypeMap struct {
		Kind    string `json:"kind" unpack:""`
		KeyType Type   `json:"key_type"`
		ValType Type   `json:"val_type"`
	}
	TypeNull struct {
		Kind string `json:"kind" unpack:""`
	}
	TypeName struct {
		Kind string `json:"kind" unpack:""`
		Name string `json:"name"`
	}
	TypeDef struct {
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
func (*TypeName) typeNode()      {}
func (*TypeDef) typeNode()       {}

package zed

type Type interface {
	Node
	typeNode()
}

type Node interface {
	Pos() int
	End() int
}

type (
	TypePrimitive struct {
		Kind    string `json:"kind" unpack:""`
		Name    string `json:"name"`
		NamePos int    `json:"name_pos"`
	}
	TypeRecord struct {
		Kind   string      `json:"kind" unpack:""`
		Lbrace int         `json:"lbrace"`
		Fields []TypeField `json:"fields"`
		Rbrace int         `json:"rbrace"`
	}
	TypeField struct {
		Name string `json:"name"`
		Type Type   `json:"type"`
	}
	TypeArray struct {
		Kind   string `json:"kind" unpack:""`
		Lbrack int    `json:"lbrack"`
		Type   Type   `json:"type"`
		Rbrack int    `json:"rbrack"`
	}
	TypeSet struct {
		Kind  string `json:"kind" unpack:""`
		Lpipe int    `json:"lpipe"`
		Type  Type   `json:"type"`
		Rpipe int    `json:"rpipe"`
	}
	TypeUnion struct {
		Kind   string `json:"kind" unpack:""`
		Lparen int    `json:"lparen"`
		Types  []Type `json:"types"`
		Rparen int    `json:"rparen"`
	}
	TypeEnum struct {
		Kind    string   `json:"kind" unpack:""`
		Symbols []string `json:"symbols"`
	}
	TypeMap struct {
		Kind    string `json:"kind" unpack:""`
		Lpipe   int    `json:"lpipe"`
		KeyType Type   `json:"key_type"`
		ValType Type   `json:"val_type"`
		Rpipe   int    `json:"rpipe"`
	}
	TypeNull struct {
		Kind       string `json:"kind" unpack:""`
		KeywordPos int    `json:"pos"`
	}
	TypeError struct {
		Kind       string `json:"kind" unpack:""`
		KeywordPos int    `json:"keyword_pos"`
		Type       Type   `json:"type"`
		Rparen     int    `json:"rparen"`
	}
	TypeName struct {
		Kind    string `json:"kind" unpack:""`
		Name    string `json:"name"`
		NamePos int    `json:"name_pos"`
	}
	TypeDef struct {
		Kind    string `json:"kind" unpack:""`
		Name    string `json:"name"`
		NamePos int    `json:"name_pos"`
		Type    Type   `json:"type"`
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

func (x *TypePrimitive) Pos() int { return x.NamePos }
func (x *TypeRecord) Pos() int    { return x.Lbrace }
func (x *TypeArray) Pos() int     { return x.Lbrack }
func (x *TypeSet) Pos() int       { return x.Lpipe }
func (x *TypeUnion) Pos() int     { return x.Lparen }
func (x *TypeEnum) Pos() int      { return -1 } // TypeEnum isn't supported in Zed language
func (x *TypeMap) Pos() int       { return x.Lpipe }
func (x *TypeNull) Pos() int      { return x.KeywordPos }
func (x *TypeError) Pos() int     { return x.KeywordPos }
func (x *TypeName) Pos() int      { return x.NamePos }
func (x *TypeDef) Pos() int       { return x.NamePos }

func (x *TypePrimitive) End() int { return x.NamePos + len(x.Name) }
func (x *TypeRecord) End() int    { return x.Rbrace + 1 }
func (x *TypeArray) End() int     { return x.Rbrack + 1 }
func (x *TypeSet) End() int       { return x.Rpipe + 1 }
func (x *TypeUnion) End() int     { return x.Rparen + 1 }
func (x *TypeEnum) End() int      { return -1 } // TypeEnum isn't supported in Zed language
func (x *TypeMap) End() int       { return x.Rpipe + 1 }
func (x *TypeNull) End() int      { return x.KeywordPos + 4 }
func (x *TypeError) End() int     { return x.Rparen + 1 }
func (x *TypeName) End() int      { return x.NamePos + len(x.Name) }
func (x *TypeDef) End() int       { return x.Type.End() }

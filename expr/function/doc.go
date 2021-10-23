package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Func struct {
	Name      string
	New       func(*zed.Context) Interface
	Signature *Signature
	Desc      string
	Examples  []Example

	root bool
}

type Example struct {
	Name   string
	Input  string
	Zed    string
	Output string
}

type Funcs []*Func

func (fns Funcs) lookup(name string, nargs int) *Func {
	for _, f := range fns {
		if f.Name == name {
			return f
		}
	}
	return nil
}

func (fns Funcs) Copy() Funcs {
	out := make([]*Func, len(fns))
	copy(out, fns)
	return fns
}

type Signature struct {
	Args   []zed.Type
	Return zed.Type
}

func sig(ret zed.Type, args ...zed.Type) *Signature {
	return &Signature{Args: args, Return: ret}
}

var (
	typeStringy = &zed.TypeAlias{
		Name: "stringy",
		Type: zed.NewTypeUnion(-1, []zed.Type{zed.TypeString, zed.TypeBstring, zed.TypeError}),
	}
	typeAny = &typeOfAny{}
)

// typeOfAny is a stand in for a parameter in a function signature that does
// not care about the type of input. This should not be used as an actual type.
type typeOfAny struct{}

func (typeOfAny) Marshal(zcode.Bytes) (interface{}, error) { return nil, nil }
func (typeOfAny) ID() int                                  { return -1 }
func (typeOfAny) String() string                           { return "any" }
func (typeOfAny) Format(zv zcode.Bytes) string             { return "" }

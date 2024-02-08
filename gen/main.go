package main

import (
	"fmt"
	"strings"

	"github.com/brimdata/zed/vector"
)

var ops = []string{"==", "!=", "<", "<=", ">", ">="}
var types = []string{"Int", "Uint", "Float", "String", "Bytes"}

func main() {
	fmt.Printf("package expr\n\n")
	fmt.Printf("// this code is automatically generated\n\n")
	fmt.Printf("import (\n")
	fmt.Printf("\t\"strings\"\n")
	fmt.Printf("\t\"bytes\"\n\n")
	fmt.Printf("\t\"github.com/brimdata/zed/vector\"\n\n")
	fmt.Printf(")\n")
	var ents strings.Builder
	for _, op := range ops {
		for _, typ := range types {
			for lform := vector.Form(0); lform < 4; lform++ {
				for rform := vector.Form(0); rform < 4; rform++ {
					if lform == vector.FormConst && rform == vector.FormConst {
						// no const x const
						continue
					}
					fmt.Println(genFunc(op, typ, lform, rform))
					ents.WriteString(fmt.Sprintf("\t%d: %s,\n", vector.CompareOpCode(op, vector.KindFromString(typ), lform, rform), funcName(op, typ, lform, rform)))
				}
			}
		}
	}
	fmt.Println("var compareFuncs = map[int]func(vector.Any, vector.Any)*vector.Bool{")
	fmt.Println(ents.String())
	fmt.Println("}")
}

func genFunc(op, typ string, lhs, rhs vector.Form) string {
	var s string
	s += "func "
	s += funcName(op, typ, lhs, rhs)
	s += "(lhs, rhs vector.Any) *vector.Bool {\n"
	s += "\tn := lhs.Len()\n"
	s += "\tout := vector.NewBoolEmpty(n, nil)\n"
	s += genVarInit("l", typ, lhs)
	s += genVarInit("r", typ, rhs)
	s += formToLoop(typ, lhs, rhs, op)
	s += "\n\treturn out\n"
	s += "}\n"
	return s
}

func genVarInit(which, typ string, form vector.Form) string {
	switch form {
	case vector.FormFlat:
		return fmt.Sprintf("\t%s := %shs.(*vector.%s)\n", which, which, typ)
	case vector.FormDict:
		s := fmt.Sprintf("\t%sd := %shs.(*vector.Dict)\n", which, which)
		s += fmt.Sprintf("\t%s := %sd.Any.(*vector.%s)\n", which, which, typ)
		s += fmt.Sprintf("\t%sx := %sd.Index\n", which, which)
		return s
	case vector.FormView:
		s := fmt.Sprintf("\t%sd := %shs.(*vector.View)\n", which, which)
		s += fmt.Sprintf("\t%s := %sd.Any.(*vector.%s)\n", which, which, typ)
		s += fmt.Sprintf("\t%sx := %sd.Index\n", which, which)
		return s
	case vector.FormConst:
		return fmt.Sprintf("\t%sconst, _ := %shs.(*vector.Const).As%s()\n", which, which, typ)
	default:
		panic("genVarInit: bad form")
	}
}

const flat_flat = `
	for k := uint32(0); k < n; k++ {
		if l.Values[k] %s r.Values[k] {
			out.Set(k)
		}
	}
`

const flat_idx = `
	for k := uint32(0); k < n; k++ {
		if l.Values[k] %s r.Values[rx[k]] {
			out.Set(k)
		}
	}
`

const flat_const = `
	for k := uint32(0); k < n; k++ {
		if l.Values[k] %s rconst {
			out.Set(k)
		}
	}`

const idx_flat = `
	for k := uint32(0); k < n; k++ {
		if l.Values[lx[k]] %s r.Values[k] {
			out.Set(k)
		}
	}
`

const idx_idx = `
	for k := uint32(0); k < n; k++ {
		if l.Values[lx[k]] %s r.Values[rx[k]] {
			out.Set(k)
		}
	}
`

const idx_const = `
	for k := uint32(0); k < n; k++ {
		if l.Values[lx[k]] %s rconst {
			out.Set(k)
		}
	}
`

const const_flat = `
	for k := uint32(0); k < n; k++ {
		if lconst %s r.Values[k] {
			out.Set(k)
		}
	}
`

const const_idx = `
	for k := uint32(0); k < n; k++ {
		if lconst %s r.Values[rx[k]] {
			out.Set(k)
		}
	}
`

const flat_flat_s = `
	for k := uint32(0); k < n; k++ {
		if %s.Compare(l.Value(k), r.Value(k)) %s 0 {
			out.Set(k)
		}
	}
`

const flat_idx_s = `
	for k := uint32(0); k < n; k++ {
		if %s.Compare(l.Value(k), r.Value(uint32(rx[k]))) %s 0 {
			out.Set(k)
		}
	}
`

const flat_const_s = `
	for k := uint32(0); k < n; k++ {
		if %s.Compare(l.Value(k), rconst) %s 0 {
			out.Set(k)
		}
	}`

const idx_flat_s = `
	for k := uint32(0); k < n; k++ {
		if %s.Compare(l.Value(uint32(lx[k])), r.Value(k)) %s 0 {
			out.Set(k)
		}
	}
`

const idx_idx_s = `
	for k := uint32(0); k < n; k++ {
		if %s.Compare(l.Value(uint32(lx[k])), r.Value(uint32(rx[k]))) %s 0 {
			out.Set(k)
		}
	}
`

const idx_const_s = `
	for k := uint32(0); k < n; k++ {
		if %s.Compare(l.Value(uint32(lx[k])), rconst) %s 0 {
			out.Set(k)
		}
	}
`

const const_flat_s = `
	for k := uint32(0); k < n; k++ {
		if %s.Compare(lconst, r.Value(k)) %s 0 {
			out.Set(k)
		}
	}
`

const const_idx_s = `
	for k := uint32(0); k < n; k++ {
		if %s.Compare(lconst, r.Value(uint32(rx[k]))) %s 0 {
			out.Set(k)
		}
	}
`

func formToLoop(typ string, lform, rform vector.Form, op string) string {
	switch typ {
	case "String", "Bytes":
		return formToLoops(typ, lform, rform, op)
	default:
		return formToLoopv(lform, rform, op)
	}
}

func formToLoops(typ string, lform, rform vector.Form, op string) string {
	pkg := strings.ToLower(typ)
	if pkg == "string" {
		pkg = "strings"
	}
	switch lform {
	case vector.FormFlat:
		switch rform {
		case vector.FormFlat:
			return fmt.Sprintf(flat_flat_s, pkg, op)
		case vector.FormDict, vector.FormView:
			return fmt.Sprintf(flat_idx_s, pkg, op)
		case vector.FormConst:
			return fmt.Sprintf(flat_const_s, pkg, op)
		}
	case vector.FormDict, vector.FormView:
		switch rform {
		case vector.FormFlat:
			return fmt.Sprintf(idx_flat_s, pkg, op)
		case vector.FormDict, vector.FormView:
			return fmt.Sprintf(idx_idx_s, pkg, op)
		case vector.FormConst:
			return fmt.Sprintf(idx_const_s, pkg, op)
		}
	case vector.FormConst:
		switch rform {
		case vector.FormFlat:
			return fmt.Sprintf(const_flat_s, pkg, op)
		case vector.FormDict, vector.FormView:
			return fmt.Sprintf(const_idx_s, pkg, op)
		}
	}
	panic("formToLoop: bad logic")
}

func formToLoopv(lform, rform vector.Form, op string) string {
	switch lform {
	case vector.FormFlat:
		switch rform {
		case vector.FormFlat:
			return fmt.Sprintf(flat_flat, op)
		case vector.FormDict, vector.FormView:
			return fmt.Sprintf(flat_idx, op)
		case vector.FormConst:
			return fmt.Sprintf(flat_const, op)
		}
	case vector.FormDict, vector.FormView:
		switch rform {
		case vector.FormFlat:
			return fmt.Sprintf(idx_flat, op)
		case vector.FormDict, vector.FormView:
			return fmt.Sprintf(idx_idx, op)
		case vector.FormConst:
			return fmt.Sprintf(idx_const, op)
		}
	case vector.FormConst:
		switch rform {
		case vector.FormFlat:
			return fmt.Sprintf(const_flat, op)
		case vector.FormDict, vector.FormView:
			return fmt.Sprintf(const_idx, op)
		}
	}
	panic("formToLoop: bad logic")
}

func funcName(op, typ string, lhs, rhs vector.Form) string {
	return "cmp_" + opToAlpha(op) + "_" + typ + "_" + lhs.String() + "_" + rhs.String()
}

func opToAlpha(op string) string {
	switch op {
	case "==":
		return "EQ"
	case "!=":
		return "NE"
	case "<":
		return "LT"
	case "<=":
		return "LE"
	case ">":
		return "GT"
	case ">=":
		return "GE"
	}
	panic("bad op")
}

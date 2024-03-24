package main

import (
	"fmt"

	"github.com/brimdata/zed/vector"
)

// code is op + lhs + rhs
// lhs/rhs is type + (flat | view | dict)

var ops = []string{"==", "!=", "<", "<=", ">", ">="}
var types = []string{"Int", "Uint", "Float", "String", "Byte"}

func main() {
	fmt.Println("package expr")
	fmt.Println("")
	for _, op := range ops {
		for _, typ := range types {
			for lform := vector.Form(0); lform < 4; lform++ {
				for rform := vector.Form(0); rform < 4; rform++ {
					fmt.Println(genFunc(op, typ, lform, rform))
				}
			}
		}
	}
}

func genFunc(op, typ string, lhs, rhs vector.Form) string {
	var s string
	s += "func "
	s += funcName(op, typ, lhs, rhs)
	s += "(l, r vector.Any) *vector.Bool {\n"
	s += "}\n"
	return s
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

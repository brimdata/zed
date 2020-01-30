package ast

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mccanne/joe"
)

type Unpacker interface {
	Unpack(string, joe.JSON) (Proc, error)
}

func unpackProcs(custom Unpacker, node joe.JSON) ([]Proc, error) {
	procList := node.Get("procs")
	if procList == joe.Undefined {
		return nil, fmt.Errorf("procs field is missing")
	}
	if !procList.IsArray() {
		return nil, fmt.Errorf("procs field is not an array")
	}
	n := procList.Len()
	procs := make([]Proc, n)
	for k := 0; k < n; k++ {
		var err error
		procs[k], err = unpackProc(custom, procList.Index(k))
		if err != nil {
			return nil, err
		}
	}
	return procs, nil
}

func unpackProc(custom Unpacker, node joe.JSON) (Proc, error) {
	op, ok := node.Get("op").String()
	if !ok {
		return nil, fmt.Errorf("AST is missing op field")
	}
	if custom != nil {
		p, err := custom.Unpack(op, node)
		if err != nil {
			return nil, err
		}
		if p != nil {
			return p, nil
		}
	}

	switch op {
	case "SequentialProc":
		procs, err := unpackProcs(custom, node)
		if err != nil {
			return nil, err
		}
		return &SequentialProc{Procs: procs}, nil
	case "ParallelProc":
		procs, err := unpackProcs(custom, node)
		if err != nil {
			return nil, err
		}
		return &ParallelProc{Procs: procs}, nil
	case "SortProc":
		fields, err := unpackFieldExprArray(node.Get("fields"))
		if err != nil {
			return nil, err
		}
		return &SortProc{Fields: fields}, nil
	case "CutProc":
		fields, err := unpackFieldExprArray(node.Get("fields"))
		if err != nil {
			return nil, err
		}
		return &CutProc{Fields: fields}, nil
	case "HeadProc":
		return &HeadProc{}, nil
	case "TailProc":
		return &TailProc{}, nil
	case "FilterProc":
		filter, err := UnpackChild(node, "filter")
		if err != nil {
			return nil, err
		}
		return &FilterProc{Filter: filter}, nil
	case "UniqProc":
		return &UniqProc{}, nil
	case "ReducerProc":
		reducers, err := unpackReducers(node.Get("reducers"))
		if err != nil {
			return nil, err
		}
		return &ReducerProc{Reducers: reducers}, nil
	case "GroupByProc":
		keys, err := unpackFieldExprArray(node.Get("keys"))
		if err != nil {
			return nil, err
		}
		reducers, err := unpackReducers(node.Get("reducers"))
		if err != nil {
			return nil, err
		}
		return &GroupByProc{Keys: keys, Reducers: reducers}, nil
	case "TopProc":
		fields, err := unpackFieldExprArray(node.Get("fields"))
		if err != nil {
			return nil, err
		}
		return &TopProc{Fields: fields}, nil
	default:
		return nil, fmt.Errorf("unknown proc op: %s", op)
	}
}

func UnpackChild(node joe.JSON, field string) (BooleanExpr, error) {
	child := node.Get(field)
	if child == joe.Undefined {
		return nil, fmt.Errorf("%s field is missing", field)
	}
	return unpackBooleanExpr(child)
}

func unpackChildren(node joe.JSON) (BooleanExpr, BooleanExpr, error) {
	left, err := UnpackChild(node, "left")
	if err != nil {
		return nil, nil, err
	}
	right, err := UnpackChild(node, "right")
	if err != nil {
		return nil, nil, err
	}
	return left, right, nil
}

func unpackBooleanExpr(node joe.JSON) (BooleanExpr, error) {
	op, ok := node.Get("op").String()
	if !ok {
		return nil, fmt.Errorf("AST is missing op field")
	}
	switch op {
	case "LogicalAnd":
		left, right, err := unpackChildren(node)
		if err != nil {
			return nil, err
		}
		return &LogicalAnd{Left: left, Right: right}, nil
	case "LogicalOr":
		left, right, err := unpackChildren(node)
		if err != nil {
			return nil, err
		}
		return &LogicalOr{Left: left, Right: right}, nil
	case "LogicalNot":
		child, err := UnpackChild(node, "expr")
		if err != nil {
			return nil, err
		}
		return &LogicalNot{Expr: child}, nil
	case "BooleanLiteral":
		return &BooleanLiteral{}, nil
	case "CompareAny":
		return &CompareAny{}, nil
	case "CompareField":
		child := node.Get("field")
		if child == joe.Undefined {
			return nil, errors.New("CompareField missing field property")
		}
		field, err := unpackFieldExpr(child)
		if err != nil {
			return nil, err
		}
		return &CompareField{Field: field}, nil

	default:
		return nil, fmt.Errorf("unknown op: %s", op)
	}
}

func unpackFieldExprArray(node joe.JSON) ([]FieldExpr, error) {
	if node == joe.Undefined {
		return nil, nil
	}
	if !node.IsArray() {
		return nil, errors.New("fields property should be an array")
	}
	n := node.Len()
	fields := make([]FieldExpr, n)
	for k := 0; k < n; k++ {
		var err error
		fields[k], err = unpackFieldExpr(node.Index(k))
		if err != nil {
			return nil, err
		}
	}
	return fields, nil
}

func unpackFieldExpr(node joe.JSON) (FieldExpr, error) {
	op, ok := node.Get("op").String()
	if !ok {
		return nil, errors.New("AST is missing op field")
	}
	switch op {
	case "FieldCall":
		child := node.Get("field")
		if child == joe.Undefined {
			return nil, errors.New("FieldCall missing field property")
		}
		field, err := unpackFieldExpr(child)
		if err != nil {
			return nil, err
		}
		return &FieldCall{Field: field}, nil
	case "FieldRead":
		return &FieldRead{}, nil
	default:
		return nil, fmt.Errorf("unknown op: %s", op)
	}
}

func unpackReducers(node joe.JSON) ([]Reducer, error) {
	if node == joe.Undefined {
		return nil, nil
	}
	if !node.IsArray() {
		return nil, errors.New("reducers property should be an array")
	}
	n := node.Len()
	reducers := make([]Reducer, n)
	for k := 0; k < n; k++ {
		fld := node.Index(k).Get("field")
		if fld == joe.Undefined {
			continue
		}
		var err error
		reducers[k].Field, err = unpackFieldExpr(fld)
		if err != nil {
			return nil, err
		}
	}
	return reducers, nil
}

// UnpackProc transforms a JSON representation of a proc into an ast.Proc.
func UnpackProc(custom Unpacker, buf []byte) (Proc, error) {
	obj, err := joe.Parse(buf)
	if err != nil {
		return nil, err
	}
	proc, err := unpackProc(custom, obj)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(buf, proc); err != nil {
		return nil, err
	}
	return proc, nil
}

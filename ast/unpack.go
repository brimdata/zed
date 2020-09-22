package ast

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/brimsec/zq/pkg/joe"
	"github.com/mitchellh/mapstructure"
)

type Unpacker interface {
	Unpack(string, joe.Interface) (Proc, error)
}

func unpackProcs(custom Unpacker, node joe.Interface) ([]Proc, error) {
	procList, err := node.Get("procs")
	if err != nil {
		return nil, fmt.Errorf("procs field is missing")
	}
	a, ok := procList.(joe.Array)
	if !ok {
		return nil, fmt.Errorf("procs field is not an array")
	}
	procs := make([]Proc, 0, len(a))
	for _, item := range a {
		proc, err := unpackProc(custom, item)
		if err != nil {
			return nil, err
		}
		procs = append(procs, proc)
	}
	return procs, nil
}

func getString(node joe.Interface, field string) (string, error) {
	item, err := node.Get(field)
	if err != nil {
		return "", fmt.Errorf("AST is missing %s field: %s", node, field)
	}
	s, err := item.String()
	if err != nil {
		return "", fmt.Errorf("AST field %s of %s is not a string", field, node)
	}
	return s, nil
}

func unpackProc(custom Unpacker, node joe.Interface) (Proc, error) {
	op, err := getString(node, "op")
	if err != nil {
		return nil, err
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
		a, _ := node.Get("fields")
		fields, err := unpackFieldExprArray(a)
		if err != nil {
			return nil, err
		}
		return &SortProc{Fields: fields}, nil
	case "CutProc":
		return &CutProc{}, nil
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
	case "PutProc":
		a, _ := node.Get("clauses")
		clauses, err := unpackExpressionAssignments(a)
		if err != nil {
			return nil, err
		}
		return &PutProc{Clauses: clauses}, nil
	case "RenameProc":
		return &RenameProc{}, nil
	case "FuseProc":
		return &FuseProc{}, nil
	case "UniqProc":
		return &UniqProc{}, nil
	case "GroupByProc":
		a, _ := node.Get("keys")
		keys, err := unpackExpressionAssignments(a)
		if err != nil {
			return nil, err
		}
		a, _ = node.Get("reducers")
		reducers, err := unpackReducers(a)
		if err != nil {
			return nil, err
		}
		return &GroupByProc{Reducers: reducers, Keys: keys}, nil
	case "TopProc":
		a, _ := node.Get("fields")
		fields, err := unpackFieldExprArray(a)
		if err != nil {
			return nil, err
		}
		return &TopProc{Fields: fields}, nil
	case "PassProc":
		return &PassProc{}, nil
	default:
		return nil, fmt.Errorf("unknown proc op: %s", op)
	}
}

func unpackExpressionAssignment(node joe.Interface) (ExpressionAssignment, error) {
	exprNode, err := node.Get("expression")
	if err != nil {
		return ExpressionAssignment{}, errors.New("ExpressionAssignment missing expression")
	}
	expr, err := UnpackExpression(exprNode)
	if err != nil {
		return ExpressionAssignment{}, err
	}
	return ExpressionAssignment{Expr: expr}, nil
}

func unpackExpressionAssignments(node joe.Interface) ([]ExpressionAssignment, error) {
	if node == nil {
		return nil, nil
	}
	a, ok := node.(joe.Array)
	if !ok {
		return nil, errors.New("assignments should be an array")
	}
	keys := make([]ExpressionAssignment, 0, len(a))
	for _, item := range a {
		assi, err := unpackExpressionAssignment(item)
		if err != nil {
			return nil, err
		}
		keys = append(keys, assi)
	}
	return keys, nil
}

func UnpackExpression(node joe.Interface) (Expression, error) {
	op, err := getString(node, "op")
	if err != nil {
		return nil, err
	}
	switch op {
	case "UnaryExpr":
		operandNode, err := node.Get("operand")
		if err != nil {
			return nil, errors.New("UnaryExpression missing operand")
		}
		operand, err := UnpackExpression(operandNode)
		if err != nil {
			return nil, err
		}
		return &UnaryExpression{Operand: operand}, nil
	case "BinaryExpr":
		lhsNode, err := node.Get("lhs")
		if err != nil {
			return nil, errors.New("BinaryExpression missing lhs")
		}
		lhs, err := UnpackExpression(lhsNode)
		if err != nil {
			return nil, err
		}

		rhsNode, err := node.Get("rhs")
		if err != nil {
			return nil, errors.New("BinaryExpression missing rhs")
		}
		rhs, err := UnpackExpression(rhsNode)
		if err != nil {
			return nil, err
		}

		return &BinaryExpression{LHS: lhs, RHS: rhs}, nil
	case "ConditionalExpr":
		conditionNode, err := node.Get("condition")
		if err != nil {
			return nil, errors.New("ConditionalExpr missing condition")
		}
		condition, err := UnpackExpression(conditionNode)
		if err != nil {
			return nil, err
		}

		thenNode, err := node.Get("then")
		if err != nil {
			return nil, errors.New("ConditionalExpr missing then")
		}
		thenClause, err := UnpackExpression(thenNode)
		if err != nil {
			return nil, err
		}

		elseNode, err := node.Get("else")
		if err != nil {
			return nil, errors.New("ConditionalExpr missing else")
		}
		elseClause, err := UnpackExpression(elseNode)
		if err != nil {
			return nil, err
		}
		return &ConditionalExpression{
			Condition: condition,
			Then:      thenClause,
			Else:      elseClause,
		}, nil
	case "FunctionCall":
		argsNode, err := node.Get("args")
		if err != nil {
			return nil, errors.New("FunctionCall missing args")
		}
		a, ok := argsNode.(joe.Array)
		if !ok {
			return nil, errors.New("FunctionCall args property must be an array")
		}
		args := make([]Expression, len(a))
		for i := range args {
			var err error
			args[i], err = UnpackExpression(a[i])
			if err != nil {
				return nil, err
			}
		}
		return &FunctionCall{Args: args}, nil
	case "CastExpr":
		exprNode, err := node.Get("expr")
		if err != nil {
			return nil, errors.New("CastExpr missing expr")
		}
		expr, err := UnpackExpression(exprNode)
		if err != nil {
			return nil, err
		}
		return &CastExpression{Expr: expr}, nil
	case "Literal":
		return &Literal{}, nil
	case "FieldRead":
		return &FieldRead{}, nil
	case "FieldCall":
		child, err := node.Get("field")
		if err != nil {
			return nil, errors.New("FieldCall missing field property")
		}
		field, err := unpackFieldExpr(child)
		if err != nil {
			return nil, err
		}
		return &FieldCall{Field: field}, nil
	default:
		return nil, fmt.Errorf("unknown Expression op %s", op)
	}
}

func UnpackChild(node joe.Interface, field string) (BooleanExpr, error) {
	child, err := node.Get(field)
	if err != nil {
		return nil, fmt.Errorf("%s field is missing", field)
	}
	return unpackBooleanExpr(child)
}

func unpackChildren(node joe.Interface) (BooleanExpr, BooleanExpr, error) {
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

func unpackBooleanExpr(node joe.Interface) (BooleanExpr, error) {
	op, err := getString(node, "op")
	if err != nil {
		return nil, fmt.Errorf("AST is missing op field")
	}
	switch op {
	case "Search":
		return &Search{}, nil
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
	case "MatchAll":
		return &MatchAll{}, nil
	case "CompareAny":
		return &CompareAny{}, nil
	case "CompareField":
		child, err := node.Get("field")
		if err != nil {
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

func unpackFieldExprArray(node joe.Interface) ([]FieldExpr, error) {
	if node == nil {
		return nil, nil
	}
	a, ok := node.(joe.Array)
	if !ok {
		return nil, errors.New("fields property should be an array")
	}
	fields := make([]FieldExpr, 0, len(a))
	for _, item := range a {
		field, err := unpackFieldExpr(item)
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}
	return fields, nil
}

func unpackFieldExpr(node joe.Interface) (FieldExpr, error) {
	op, err := getString(node, "op")
	if err != nil {
		return nil, errors.New("AST is missing op field")
	}
	switch op {
	case "FieldCall":
		child, err := node.Get("field")
		if err != nil {
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

func unpackReducers(node joe.Interface) ([]Reducer, error) {
	if node == nil {
		return nil, nil
	}
	a, ok := node.(joe.Array)
	if !ok {
		return nil, errors.New("reducers property should be an array")
	}
	reducers := make([]Reducer, len(a))
	for k, item := range a {
		fld, err := item.Get("field")
		if err != nil {
			continue
		}
		reducers[k].Field, err = unpackFieldExpr(fld)
		if err != nil {
			return nil, err
		}
	}
	return reducers, nil
}

func UnpackMap(custom Unpacker, m interface{}) (Proc, error) {
	obj := joe.Convert(m)
	proc, err := unpackProc(custom, obj)
	if err != nil {
		return nil, err
	}
	c := &mapstructure.DecoderConfig{
		TagName: "json",
		Result:  proc,
		Squash:  true,
	}
	dec, err := mapstructure.NewDecoder(c)
	if err != nil {
		return nil, err
	}
	return proc, dec.Decode(m)
}

// UnpackJSON transforms a JSON representation of a proc into an ast.Proc.
func UnpackJSON(custom Unpacker, buf []byte) (Proc, error) {
	obj, err := joe.Unmarshal(buf)
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

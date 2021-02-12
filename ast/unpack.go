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
	if node == nil {
		return nil, errors.New("ast.unpackProcs: procs field is missing")
	}
	procList, err := node.Get("procs")
	if err != nil {
		return nil, fmt.Errorf("ast.unpackProcs: procs field is missing")
	}
	a, ok := procList.(joe.Array)
	if !ok {
		return nil, fmt.Errorf("ast.unpackProcs: procs field is not an array")
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
	if node == nil {
		return nil, errors.New("bad AST: missing proc field")
	}
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
		fields, err := unpackExprs(a)
		if err != nil {
			return nil, err
		}
		return &SortProc{Fields: fields}, nil
	case "CutProc":
		a, _ := node.Get("fields")
		fas, err := unpackAssignments(a)
		if err != nil {
			return nil, err
		}
		return &CutProc{Fields: fas}, nil
	case "PickProc":
		a, _ := node.Get("fields")
		fas, err := unpackAssignments(a)
		if err != nil {
			return nil, err
		}
		return &PickProc{Fields: fas}, nil
	case "DropProc":
		a, _ := node.Get("fields")
		fields, err := unpackExprs(a)
		if err != nil {
			return nil, err
		}
		return &DropProc{Fields: fields}, nil
	case "HeadProc":
		return &HeadProc{}, nil
	case "TailProc":
		return &TailProc{}, nil
	case "FilterProc":
		f, err := node.Get("filter")
		if err != nil {
			return nil, errors.New("ast filter proc: missing filter field")
		}
		filter, err := UnpackExpression(f)
		if err != nil {
			return nil, err
		}
		return &FilterProc{Filter: filter}, nil
	case "FunctionCall":
		// A FuncionCall can appear in proc context too.
		args, err := unpackArgs(node)
		if err != nil {
			return nil, err
		}
		return &FunctionCall{Args: args}, nil
	case "PutProc":
		a, _ := node.Get("clauses")
		clauses, err := unpackAssignments(a)
		if err != nil {
			return nil, err
		}
		return &PutProc{Clauses: clauses}, nil
	case "RenameProc":
		a, _ := node.Get("fields")
		fas, err := unpackAssignments(a)
		if err != nil {
			return nil, err
		}
		return &RenameProc{Fields: fas}, nil
	case "FuseProc":
		return &FuseProc{}, nil
	case "UniqProc":
		return &UniqProc{}, nil
	case "GroupByProc":
		a, _ := node.Get("keys")
		keys, err := unpackAssignments(a)
		if err != nil {
			return nil, err
		}
		a, _ = node.Get("reducers")
		reducers, err := unpackAssignments(a)
		if err != nil {
			return nil, err
		}
		return &GroupByProc{Reducers: reducers, Keys: keys}, nil
	case "TopProc":
		a, _ := node.Get("fields")
		fields, err := unpackExprs(a)
		if err != nil {
			return nil, err
		}
		return &TopProc{Fields: fields}, nil
	case "PassProc":
		return &PassProc{}, nil
	case "JoinProc":
		n, _ := node.Get("clauses")
		clauses, err := unpackAssignments(n)
		if err != nil {
			return nil, err
		}
		n, err = node.Get("left_key")
		if err != nil {
			return nil, errors.New("ast join proc: missing left_key field")
		}
		leftKey, err := UnpackExpression(n)
		if err != nil {
			return nil, err
		}
		n, err = node.Get("right_key")
		if err != nil {
			return nil, errors.New("ast join proc: missing right_key field")
		}
		rightKey, err := UnpackExpression(n)
		if err != nil {
			return nil, err
		}
		return &JoinProc{LeftKey: leftKey, RightKey: rightKey, Clauses: clauses}, nil
	case "SqlExpr":
		return unpackSQL(node)

	default:
		return nil, fmt.Errorf("ast.unpackProc: unknown proc op: %s", op)
	}
}

func unpackExprs(node joe.Interface) ([]Expression, error) {
	if node == nil {
		return nil, nil
	}
	a, ok := node.(joe.Array)
	if !ok {
		return nil, errors.New("ast.unpackExprs: fields property should be an array")
	}
	exprs := make([]Expression, 0, len(a))
	for _, item := range a {
		e, err := UnpackExpression(item)
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, e)
	}
	return exprs, nil
}

func unpackAssignment(node joe.Interface) (Assignment, error) {
	if node == nil {
		return Assignment{}, errors.New("ast.unpackAssignment: missing assignment field")
	}
	var lhs Expression
	lhsNode, err := node.Get("lhs")
	// LHS is optional as compiler will infer a field name from the LHS.
	if err == nil && lhsNode != nil {
		lhs, err = UnpackExpression(lhsNode)
		if err != nil {
			return Assignment{}, err
		}
	}
	rhsNode, err := node.Get("rhs")
	if err != nil {
		return Assignment{}, errors.New("ast.unpackAssignment: missing rhs field")
	}
	rhs, err := UnpackExpression(rhsNode)
	if err != nil {
		return Assignment{}, err
	}
	return Assignment{LHS: lhs, RHS: rhs}, nil
}

func unpackAssignments(node joe.Interface) ([]Assignment, error) {
	if node == nil {
		return nil, nil
	}
	a, ok := node.(joe.Array)
	if !ok {
		return nil, errors.New("ast.unpackAssignments: not an array")
	}
	fas := make([]Assignment, 0, len(a))
	for _, item := range a {
		fa, err := unpackAssignment(item)
		if err != nil {
			return nil, err
		}
		fas = append(fas, fa)
	}
	return fas, nil
}

func UnpackExpression(node joe.Interface) (Expression, error) {
	if node == nil {
		return nil, errors.New("ast.UnpackExpression: no expression provided")
	}
	op, err := getString(node, "op")
	if err != nil {
		return nil, err
	}
	switch op {
	case "UnaryExpr":
		return unpackUnaryExpr(node)
	case "BinaryExpr":
		return unpackBinaryExpr(node)
	case "SelectExpr":
		return unpackSelectExpr(node)
	case "Search":
		return &Search{}, nil
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
		args, err := unpackArgs(node)
		if err != nil {
			return nil, err
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
	case "Reducer":
		exprNode, _ := node.Get("expr")
		var expr Expression
		if exprNode != nil {
			expr, _ = UnpackExpression(exprNode)
		}
		whereNode, _ := node.Get("where")
		var where Expression
		if whereNode != nil {
			where, _ = UnpackExpression(whereNode)
		}
		return &Reducer{Expr: expr, Where: where}, nil
	case "Literal":
		return &Literal{}, nil
	case "Identifier":
		return &Identifier{}, nil
	case "RootRecord":
		return &RootRecord{}, nil
	case "Empty":
		return &Empty{}, nil
	case "SqlExpr":
		return unpackSQL(node)
	default:
		return nil, fmt.Errorf("ast.UnpackExpression: unknown op %s", op)
	}
}

func unpackArgs(node joe.Interface) ([]Expression, error) {
	argsNode, err := node.Get("args")
	if err != nil {
		return nil, errors.New("ast node missing function args field")
	}
	a, ok := argsNode.(joe.Array)
	if !ok {
		return nil, errors.New("function args property must be an array")
	}
	args := make([]Expression, len(a))
	for i := range args {
		var err error
		args[i], err = UnpackExpression(a[i])
		if err != nil {
			return nil, err
		}
	}
	return args, nil
}

func unpackBinaryExpr(node joe.Interface) (*BinaryExpression, error) {
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
}

func unpackSelectExpr(node joe.Interface) (*SelectExpression, error) {
	selectorsNode, err := node.Get("selectors")
	if err != nil {
		return nil, errors.New("SelectExpression missing selectors field")
	}
	selectors, err := unpackExprs(selectorsNode)
	if err != nil {
		return nil, err
	}
	return &SelectExpression{Selectors: selectors}, nil
}

func unpackUnaryExpr(node joe.Interface) (*UnaryExpression, error) {
	operandNode, err := node.Get("operand")
	if err != nil {
		return nil, errors.New("UnaryExpr missing operand")
	}
	operand, err := UnpackExpression(operandNode)
	if err != nil {
		return nil, err
	}
	return &UnaryExpression{Operand: operand}, nil
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
	if len(buf) == 0 {
		return nil, nil
	}
	obj, err := joe.Unmarshal(buf)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, nil
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

func unpackSQL(node joe.Interface) (*SqlExpression, error) {
	// The select proc assumes all fields present in the JSON though
	// they can hold null values.
	selectField, err := node.Get("select")
	if err != nil {
		return nil, fmt.Errorf("ast.unpackSQL: 'select' field is missing")
	}
	var selectAssignments []Assignment
	if selectField != nil {
		selectAssignments, err = unpackAssignments(selectField)
		if err != nil {
			return nil, fmt.Errorf("ast.unpackSQL: 'select' field has wrong format")
		}
	}
	fromField, err := node.Get("from")
	if err != nil {
		return nil, fmt.Errorf("ast.unpackSQL: 'from' field is missing")
	}
	var fromExpr Expression
	if fromField != nil {
		fromExpr, err = UnpackExpression(fromField)
		if err != nil {
			return nil, fmt.Errorf("ast.unpackSQL: 'from' field has wrong format")
		}
	}
	joinsField, err := node.Get("joins")
	if err != nil {
		return nil, fmt.Errorf("ast.unpackSQL: 'joins' field is missing")
	}
	joins, err := unpackJoins(joinsField)
	if err != nil {
		return nil, err
	}
	whereField, err := node.Get("where")
	if err != nil {
		return nil, fmt.Errorf("ast.unpackSQL: 'where' field is missing")
	}
	var whereExpr Expression
	if whereField != nil {
		whereExpr, err = UnpackExpression(whereField)
		if err != nil {
			return nil, fmt.Errorf("ast.unpackSQL: 'where' field has wrong format")
		}
	}
	groupbyField, err := node.Get("groupby")
	if err != nil {
		return nil, fmt.Errorf("ast.unpackSQL: 'groupby' field is missing")
	}
	var groupbyExprs []Expression
	if groupbyField != nil {
		groupbyExprs, err = unpackExprs(groupbyField)
		if err != nil {
			return nil, fmt.Errorf("ast.unpackSQL: 'groupby' field has wrong format")
		}
	}
	havingField, err := node.Get("having")
	if err != nil {
		return nil, fmt.Errorf("ast.unpackSQL: 'having' field is missing")
	}
	var havingExpr Expression
	if havingField != nil {
		havingExpr, err = UnpackExpression(havingField)
		if err != nil {
			return nil, fmt.Errorf("ast.unpackSQL: 'having' field has wrong format")
		}
	}
	orderField, err := node.Get("order")
	if err != nil {
		return nil, fmt.Errorf("ast.unpackSQL: 'order' field is missing")
	}
	var orderExprs []Expression
	if orderField != nil {
		orderExprs, err = unpackExprs(orderField)
		if err != nil {
			return nil, fmt.Errorf("ast.unpackSQL: 'order' field has wrong format")
		}
	}
	return &SqlExpression{
		Op:      "SqlExpr",
		Select:  selectAssignments,
		From:    fromExpr,
		Joins:   joins,
		Where:   whereExpr,
		GroupBy: groupbyExprs,
		Having:  havingExpr,
		Order:   orderExprs,
		//Ascending literal for order doens't need further unpacking
	}, nil
}

func unpackJoins(node joe.Interface) ([]JoinClause, error) {
	if node == nil {
		return nil, nil
	}
	a, ok := node.(joe.Array)
	if !ok {
		return nil, errors.New("ast.unpackJoins: 'joins' field is not an array")
	}
	joins := make([]JoinClause, 0, len(a))
	for _, item := range a {
		e, err := unpackJoin(item)
		if err != nil {
			return nil, err
		}
		joins = append(joins, *e)
	}
	return joins, nil
}

func unpackJoin(node joe.Interface) (*JoinClause, error) {
	tableField, err := node.Get("table")
	if err != nil {
		return nil, fmt.Errorf("ast.unpackJoin: 'table' field is missing")
	}
	var tableExpr Expression
	if tableField != nil {
		tableExpr, err = UnpackExpression(tableField)
		if err != nil {
			return nil, fmt.Errorf("ast.unpackJoin: 'table' field has wrong format")
		}
	}
	leftKeyField, err := node.Get("left_key")
	if err != nil {
		return nil, fmt.Errorf("ast.unpackJoin: 'left_key' field is missing")
	}
	var leftKeyExpr Expression
	if leftKeyField != nil {
		leftKeyExpr, err = UnpackExpression(leftKeyField)
		if err != nil {
			return nil, fmt.Errorf("ast.unpackJoin: 'left_key' field has wrong format")
		}
	}
	rightKeyField, err := node.Get("right_key")
	if err != nil {
		return nil, fmt.Errorf("ast.unpackJoin: 'right_key' field is missing")
	}
	var rightKeyExpr Expression
	if rightKeyField != nil {
		rightKeyExpr, err = UnpackExpression(rightKeyField)
		if err != nil {
			return nil, fmt.Errorf("ast.unpackJoin: 'right_key' field has wrong format")
		}
	}
	aliasField, err := node.Get("alias")
	if err != nil {
		return nil, fmt.Errorf("ast.unpackJoin: 'alias' field is missing")
	}
	var aliasExpr Expression
	if aliasField != nil {
		aliasExpr, err = UnpackExpression(aliasField)
		if err != nil {
			return nil, fmt.Errorf("ast.unpackJoin: 'alias' field has wrong format")
		}
	}
	return &JoinClause{
		Op:       "JoinClause",
		Table:    tableExpr,
		LeftKey:  leftKeyExpr,
		RightKey: rightKeyExpr,
		Alias:    aliasExpr,
	}, nil
}

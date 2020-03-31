package zql

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/brimsec/zq/ast"
)

// ParseProc() is an entry point for use from external go code,
// mostly just a wrapper around Parse() that casts the return value.
func ParseProc(query string) (ast.Proc, error) {
	parsed, err := Parse("", []byte(query))
	if err != nil {
		return nil, err
	}
	ret, ok := parsed.(ast.Proc)
	if !ok {
		return nil, fmt.Errorf("parser generated a %T (expected ast.Proc)", parsed)
	}
	return ret, nil
}

// Helper to get a properly-typed slice of Procs from an interface{}.
func procArray(val interface{}) []ast.Proc {
	var ret []ast.Proc
	for _, v := range val.([]interface{}) {
		ret = append(ret, v.(ast.Proc))
	}
	return ret
}

func makeSequentialProc(procsIn interface{}) ast.Proc {
	procs := procArray(procsIn)
	if len(procs) == 0 {
		return procs[0]
	}
	return &ast.SequentialProc{ast.Node{"SequentialProc"}, procs}
}

func makeParallelProc(procsIn interface{}) ast.Proc {
	procs := procArray(procsIn)
	if len(procs) == 0 {
		return procs[0]
	}
	return &ast.ParallelProc{ast.Node{"ParallelProc"}, procArray(procsIn)}
}

func makeLiteral(typ string, val interface{}) *ast.Literal {
	return &ast.Literal{ast.Node{"Literal"}, typ, val.(string)}
}

func getValueType(val interface{}) string {
	return val.(*ast.Literal).Type
}

type FieldCallPlaceholder struct {
	op    string
	param string
}

func makeFieldCall(fn, fieldIn, paramIn interface{}) interface{} {
	var param string
	if paramIn != nil {
		param = paramIn.(string)
	}
	if fieldIn != nil {
		return &ast.FieldCall{ast.Node{"FieldCall"}, fn.(string), fieldIn.(ast.FieldExpr), param}
	}
	return &FieldCallPlaceholder{fn.(string), param}
}

func chainFieldCalls(base, derefs interface{}) ast.FieldExpr {
	var ret ast.FieldExpr
	ret = &ast.FieldRead{ast.Node{"FieldRead"}, base.(string)}
	if derefs != nil {
		for _, d := range derefs.([]interface{}) {
			call := d.(*FieldCallPlaceholder)
			ret = &ast.FieldCall{ast.Node{"FieldCall"}, call.op, ret, call.param}
		}
	}
	return ret
}

func makeMatchAll() *ast.MatchAll {
	return &ast.MatchAll{ast.Node{"MatchAll"}}
}

func makeCompareField(comparatorIn, fieldIn, valueIn interface{}) *ast.CompareField {
	comparator := comparatorIn.(string)
	field := fieldIn.(ast.FieldExpr)
	value := valueIn.(*ast.Literal)
	return &ast.CompareField{ast.Node{"CompareField"}, comparator, field, *value}
}

func makeCompareAny(comparatorIn, recurseIn, valueIn interface{}) *ast.CompareAny {
	comparator := comparatorIn.(string)
	recurse := recurseIn.(bool)
	value := valueIn.(*ast.Literal)
	return &ast.CompareAny{ast.Node{"CompareAny"}, comparator, recurse, *value}
}

func makeLogicalNot(exprIn interface{}) *ast.LogicalNot {
	return &ast.LogicalNot{ast.Node{"LogicalNot"}, exprIn.(ast.BooleanExpr)}
}

func makeOrChain(firstIn, restIn interface{}) ast.BooleanExpr {
	first := firstIn.(ast.BooleanExpr)
	if restIn == nil {
		return first
	}

	result := first
	rest := restIn.([]interface{})
	for _, r := range rest {
		term := r.(ast.BooleanExpr)
		result = &ast.LogicalOr{ast.Node{"LogicalOr"}, result, term}
	}
	return result
}

func makeAndChain(firstIn, restIn interface{}) ast.BooleanExpr {
	first := firstIn.(ast.BooleanExpr)
	if restIn == nil {
		return first
	}

	result := first
	rest := restIn.([]interface{})
	for _, r := range rest {
		term := r.(ast.BooleanExpr)
		result = &ast.LogicalAnd{ast.Node{"LogicalAnd"}, result, term}
	}
	return result
}

func fieldExprArray(val interface{}) []ast.FieldExpr {
	var ret []ast.FieldExpr
	if val != nil {
		for _, f := range val.([]interface{}) {
			ret = append(ret, f.(ast.FieldExpr))
		}
	}
	return ret
}

type ProcArg struct {
	Name  string
	Value string
}

func makeArg(nameIn, valIn interface{}) *ProcArg {
	var val string
	if valIn != nil {
		val = valIn.(string)
	}
	return &ProcArg{nameIn.(string), val}
}

func makeSortProc(argsIn, fieldsIn interface{}) (*ast.SortProc, error) {
	params := make(map[string]string)
	argsArray := argsIn.([]interface{})
	for _, a := range argsArray {
		arg := *a.(*ProcArg)
		_, have := params[arg.Name]
		if have {
			return nil, fmt.Errorf("Duplicate argument -%s", arg.Name)
		}
		params[arg.Name] = arg.Value
	}

	sortdir := 1
	_, haveR := params["r"]
	if haveR {
		sortdir = -1
	}

	var limit int
	limitArg, haveLimit := params["limit"]
	if haveLimit {
		limit = parseInt(limitArg).(int)
	}
	nullsfirst := false
	nullsArg, _ := params["nulls"]
	if nullsArg == "first" {
		nullsfirst = true
	}
	fields := fieldExprArray(fieldsIn)
	return &ast.SortProc{ast.Node{"SortProc"}, limit, fields, sortdir, nullsfirst}, nil
}

func makeTopProc(fieldsIn, limitIn, flushIn interface{}) *ast.TopProc {
	fields := fieldExprArray(fieldsIn)
	var limit int
	if limitIn != nil {
		limit = limitIn.(int)
	}
	flush := flushIn != nil
	return &ast.TopProc{ast.Node{"TopProc"}, limit, fields, flush}
}

func makeCutProc(fieldsIn interface{}) *ast.CutProc {
	fields := fieldExprArray(fieldsIn)
	return &ast.CutProc{ast.Node{"CutProc"}, fields}
}

func makeHeadProc(countIn interface{}) *ast.HeadProc {
	count := countIn.(int)
	return &ast.HeadProc{ast.Node{"HeadProc"}, count}
}

func makeTailProc(countIn interface{}) *ast.TailProc {
	count := countIn.(int)
	return &ast.TailProc{ast.Node{"TailProc"}, count}
}

func makeUniqProc(cflag bool) *ast.UniqProc {
	return &ast.UniqProc{ast.Node{"UniqProc"}, cflag}
}

func makeFilterProc(expr interface{}) *ast.FilterProc {
	return &ast.FilterProc{ast.Node{"FilterProc"}, expr.(ast.BooleanExpr)}
}

func makePutProc(target, expr interface{}) *ast.PutProc {
	return &ast.PutProc{ast.Node{"PutProc"}, target.(string), expr.(ast.Expression)}
}

func makeReducer(opIn, varIn, fieldIn interface{}) *ast.Reducer {
	var field ast.FieldExpr
	if fieldIn != nil {
		field = fieldIn.(ast.FieldExpr)
	}
	return &ast.Reducer{ast.Node{opIn.(string)}, varIn.(string), field}
}

func overrideReducerVar(reducerIn, varIn interface{}) *ast.Reducer {
	reducer := reducerIn.(*ast.Reducer)
	reducer.Var = varIn.(string)
	return reducer
}

func makeDuration(seconds interface{}) *ast.Duration {
	return &ast.Duration{seconds.(int)}
}

func reducersArray(reducersIn interface{}) []ast.Reducer {
	arr := reducersIn.([]interface{})
	ret := make([]ast.Reducer, len(arr))
	for i, r := range arr {
		ret[i] = *(r.(*ast.Reducer))
	}
	return ret
}

func makeReducerProc(reducers interface{}) *ast.ReducerProc {
	return &ast.ReducerProc{
		Node:     ast.Node{"ReducerProc"},
		Reducers: reducersArray(reducers),
	}
}

func makeGroupByProc(durationIn, limitIn, keysIn, reducersIn interface{}) *ast.GroupByProc {
	var duration ast.Duration
	if durationIn != nil {
		duration = *(durationIn.(*ast.Duration))
	}

	var limit int
	if limitIn != nil {
		limit = limitIn.(int)
	}

	keys := fieldExprArray(keysIn)
	reducers := reducersArray(reducersIn)

	return &ast.GroupByProc{
		Node:     ast.Node{"GroupByProc"},
		Duration: duration,
		Limit:    limit,
		Keys:     keys,
		Reducers: reducers,
	}
}

// Help for grammar rules that return the matched characters without
// converting to string.  We should (eventually) clean up these
// grammar rules so this isn't needed.
func getString(s interface{}) string {
	if r, ok := s.(string); ok {
		return r
	}
	a := s.([]uint8)
	return string(a)
}

func makeUnaryExpr(opIn, operandIn interface{}) ast.Expression {
	return &ast.UnaryExpression{
		ast.Node{"UnaryExpr"},
		opIn.(string),
		operandIn.(ast.Expression),
	}
}

func makeBinaryExprChain(firstIn, restIn interface{}) ast.Expression {
	first := firstIn.(ast.Expression)
	if restIn == nil {
		return first
	}

	result := first
	rest := restIn.([]interface{})
	for _, r := range rest {
		params := r.([]interface{})
		if len(params) < 4 {
			panic("expected array with at least 4 items")
		}
		op := getString(params[1])
		term := params[3].(ast.Expression)
		result = &ast.BinaryExpression{ast.Node{"BinaryExpr"}, op, result, term}
	}
	return result
}

func makeConditionalExpr(condition, thenClause, elseClause interface{}) ast.Expression {
	return &ast.ConditionalExpression{
		ast.Node{"ConditionalExpr"},
		condition.(ast.Expression),
		thenClause.(ast.Expression),
		elseClause.(ast.Expression),
	}
}

func makeFunctionCall(fn, argsIn interface{}) ast.Expression {
	argArray := argsIn.([]interface{})
	args := make([]ast.Expression, len(argArray))
	for i, a := range argArray {
		args[i] = a.(ast.Expression)
	}
	return &ast.FunctionCall{
		ast.Node{"FunctionCall"},
		fn.(string),
		args,
	}
}

func joinChars(in interface{}) string {
	str := bytes.Buffer{}
	for _, i := range in.([]interface{}) {
		// handle joining bytes or strings
		if s, ok := i.([]byte); ok {
			str.Write(s)
		} else {
			str.WriteString(i.(string))
		}
	}
	return str.String()
}

func toLowerCase(in interface{}) interface{} {
	return strings.ToLower(in.(string))
}

func parseInt(v interface{}) interface{} {
	num := v.(string)
	i, err := strconv.Atoi(num)
	if err != nil {
		return nil
	}

	return i
}

func parseFloat(v interface{}) interface{} {
	num := v.(string)
	if f, err := strconv.ParseFloat(num, 10); err != nil {
		return f
	}

	return nil
}

func OR(a, b interface{}) interface{} {
	if a != nil {
		return a
	}

	return b
}

func makeUnicodeChar(chars interface{}) string {
	var r rune
	for _, char := range chars.([]interface{}) {
		if char != nil {
			var v byte
			ch := char.([]byte)[0]
			switch {
			case ch >= '0' && ch <= '9':
				v = ch - '0'
			case ch >= 'a' && ch <= 'f':
				v = ch - 'a' + 10
			case ch >= 'A' && ch <= 'F':
				v = ch - 'A' + 10
			}
			r = (16 * r) + rune(v)
		}
	}

	return string(r)
}

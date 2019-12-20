package helpers

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/mccanne/zq/ast"
)

// Helper to get a properly-typed slice of Procs from an interface{}.
func ProcArray(val interface{}) []ast.Proc {
	var ret []ast.Proc
	for _, v := range val.([]interface{}) {
		ret = append(ret, v.(ast.Proc))
	}
	return ret
}

func MakeSequentialProc(procsIn interface{}) ast.Proc {
	procs := ProcArray(procsIn)
	if len(procs) == 0 {
		return procs[0]
	}
	return &ast.SequentialProc{ast.Node{"SequentialProc"}, procs}
}

func MakeParallelProc(procsIn interface{}) ast.Proc {
	procs := ProcArray(procsIn)
	if len(procs) == 0 {
		return procs[0]
	}
	return &ast.ParallelProc{ast.Node{"ParallelProc"}, ProcArray(procsIn)}
}

func MakeTypedValue(typ string, val interface{}) *ast.TypedValue {
	return &ast.TypedValue{typ, val.(string)}
}

func GetValueType(val interface{}) string {
	return val.(*ast.TypedValue).Type
}

type FieldCallPlaceholder struct {
	op    string
	param string
}

func MakeFieldCall(fn, fieldIn, paramIn interface{}) interface{} {
	var param string
	if paramIn != nil {
		param = paramIn.(string)
	}
	if fieldIn != nil {
		return &ast.FieldCall{ast.Node{"FieldCall"}, fn.(string), fieldIn.(ast.FieldExpr), param}
	}
	return &FieldCallPlaceholder{fn.(string), param}
}

func ChainFieldCalls(base, derefs interface{}) ast.FieldExpr {
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

func MakeBooleanLiteral(val bool) *ast.BooleanLiteral {
	return &ast.BooleanLiteral{ast.Node{"BooleanLiteral"}, val}
}

func MakeCompareField(comparatorIn, fieldIn, valueIn interface{}) *ast.CompareField {
	comparator := comparatorIn.(string)
	field := fieldIn.(ast.FieldExpr)
	value := valueIn.(*ast.TypedValue)
	return &ast.CompareField{ast.Node{"CompareField"}, comparator, field, *value}
}

func MakeCompareAny(comparatorIn, recurseIn, valueIn interface{}) *ast.CompareAny {
	comparator := comparatorIn.(string)
	recurse := recurseIn.(bool)
	value := valueIn.(*ast.TypedValue)
	return &ast.CompareAny{ast.Node{"CompareAny"}, comparator, recurse, *value}
}

func MakeLogicalNot(exprIn interface{}) *ast.LogicalNot {
	return &ast.LogicalNot{ast.Node{"LogicalNot"}, exprIn.(ast.BooleanExpr)}
}

func MakeOrChain(firstIn, restIn interface{}) ast.BooleanExpr {
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

func MakeAndChain(firstIn, restIn interface{}) ast.BooleanExpr {
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

func MakeSearchString(val interface{}) *ast.SearchString {
	return &ast.SearchString{ast.Node{"SearchString"}, *(val.(*ast.TypedValue))}
}

func ResetSearchStringType(val interface{}) {
	val.(*ast.SearchString).Value.Type = "string"
}

// Helper to get a properly-typed slice of strings from an interface{}
func StringArray(val interface{}) []string {
	var ret []string
	if val != nil {
		for _, v := range val.([]interface{}) {
			ret = append(ret, v.(string))
		}
	}
	return ret
}

func FieldExprArray(val interface{}) []ast.FieldExpr {
	var ret []ast.FieldExpr
	if val != nil {
		for _, f := range val.([]interface{}) {
			ret = append(ret, f.(ast.FieldExpr))
		}
	}
	return ret
}

func MakeSortProc(fieldsIn, dirIn, limitIn interface{}) *ast.SortProc {
	fields := FieldExprArray(fieldsIn)
	sortdir := dirIn.(int)
	var limit int
	if limitIn != nil {
		limit = limitIn.(int)
	}
	return &ast.SortProc{ast.Node{"SortProc"}, limit, fields, sortdir}
}

func MakeTopProc(fieldsIn, limitIn, flushIn interface{}) *ast.TopProc {
	fields := FieldExprArray(fieldsIn)
	var limit int
	if limitIn != nil {
		limit = limitIn.(int)
	}
	flush := flushIn != nil
	return &ast.TopProc{ast.Node{"TopProc"}, limit, fields, flush}
}

func MakeCutProc(fieldsIn interface{}) *ast.CutProc {
	fields := FieldExprArray(fieldsIn)
	return &ast.CutProc{ast.Node{"CutProc"}, fields}
}

func MakeHeadProc(countIn interface{}) *ast.HeadProc {
	count := countIn.(int)
	return &ast.HeadProc{ast.Node{"HeadProc"}, count}
}

func MakeTailProc(countIn interface{}) *ast.TailProc {
	count := countIn.(int)
	return &ast.TailProc{ast.Node{"TailProc"}, count}
}

func MakeUniqProc(cflag bool) *ast.UniqProc {
	return &ast.UniqProc{ast.Node{"UniqProc"}, cflag}
}

func MakeFilterProc(expr interface{}) *ast.FilterProc {
	return &ast.FilterProc{ast.Node{"FilterProc"}, expr.(ast.BooleanExpr)}
}

func MakeReducer(opIn, varIn, fieldIn interface{}) *ast.Reducer {
	var field string
	if fieldIn != nil {
		field = fieldIn.(string)
	}
	return &ast.Reducer{ast.Node{opIn.(string)}, varIn.(string), field}
}

func OverrideReducerVar(reducerIn, varIn interface{}) *ast.Reducer {
	reducer := reducerIn.(*ast.Reducer)
	reducer.Var = varIn.(string)
	return reducer
}

func MakeDuration(seconds interface{}) *ast.Duration {
	return &ast.Duration{seconds.(int)}
}

func ReducersArray(reducersIn interface{}) []ast.Reducer {
	arr := reducersIn.([]interface{})
	ret := make([]ast.Reducer, len(arr))
	for i, r := range arr {
		ret[i] = *(r.(*ast.Reducer))
	}
	return ret
}

func MakeReducerProc(reducers interface{}) *ast.ReducerProc {
	return &ast.ReducerProc{
		Node:     ast.Node{"ReducerProc"},
		Reducers: ReducersArray(reducers),
	}
}

func MakeGroupByProc(durationIn, limitIn, keysIn, reducersIn interface{}) *ast.GroupByProc {
	var duration ast.Duration
	if durationIn != nil {
		duration = *(durationIn.(*ast.Duration))
	}

	var limit int
	if limitIn != nil {
		limit = limitIn.(int)
	}

	keys := FieldExprArray(keysIn)
	reducers := ReducersArray(reducersIn)

	return &ast.GroupByProc{
		Node:     ast.Node{"GroupByProc"},
		Duration: duration,
		Limit:    limit,
		Keys:     keys,
		Reducers: reducers,
	}
}

func JoinChars(in interface{}) string {
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

func ToLowerCase(in interface{}) interface{} {
	return strings.ToLower(in.(string))
}

func ParseInt(v interface{}) interface{} {
	num := v.(string)
	i, err := strconv.Atoi(num)
	if err != nil {
		return nil
	}

	return i
}

func ParseFloat(v interface{}) interface{} {
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

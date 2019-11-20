package zql

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mccanne/zq/ast"
	"github.com/mccanne/zq/reglob"
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

func makeTypedValue(typ string, val interface{}) *ast.TypedValue {
	return &ast.TypedValue{typ, val.(string)}
}

func getValueType(val interface{}) string {
	return val.(*ast.TypedValue).Type
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
		return &ast.FieldCall{ast.Node{"FieldCall"}, fn.(string), *fieldIn.(*ast.FieldExpr), param}
	}
	return &FieldCallPlaceholder{fn.(string), param}
}

func chainFieldCalls(base, derefs interface{}) *ast.FieldExpr {
	var ret ast.FieldExpr
	ret = &ast.FieldRead{ast.Node{"FieldRead"}, base.(string)}
	if derefs != nil {
		for _, d := range derefs.([]interface{}) {
			call := d.(*FieldCallPlaceholder)
			ret = &ast.FieldCall{ast.Node{"FieldCall"}, call.op, ret, call.param}
		}
	}
	return &ret
}

func makeBooleanLiteral(val bool) *ast.BooleanLiteral {
	return &ast.BooleanLiteral{ast.Node{"BooleanLiteral"}, val}
}

func makeCompareField(comparatorIn, fieldIn, valueIn interface{}) *ast.CompareField {
	comparator := comparatorIn.(string)
	field := fieldIn.(*ast.FieldExpr)
	value := valueIn.(*ast.TypedValue)
	return &ast.CompareField{ast.Node{"CompareField"}, comparator, *field, *value}
}

func makeCompareAny(comparatorIn, valueIn interface{}) *ast.CompareAny {
	comparator := comparatorIn.(string)
	value := valueIn.(*ast.TypedValue)
	return &ast.CompareAny{ast.Node{"CompareAny"}, comparator, *value}
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

func makeSearchString(val interface{}) *ast.SearchString {
	return &ast.SearchString{ast.Node{"SearchString"}, *(val.(*ast.TypedValue))}
}

func resetSearchStringType(val interface{}) {
	val.(*ast.SearchString).Value.Type = "string"
}

// Helper to get a properly-typed slice of strings from an interface{}
func stringArray(val interface{}) []string {
	var ret []string
	if val != nil {
		for _, v := range val.([]interface{}) {
			ret = append(ret, v.(string))
		}
	}
	return ret
}

func makeSortProc(fieldsIn, dirIn, limitIn interface{}) *ast.SortProc {
	fields := stringArray(fieldsIn)
	sortdir := dirIn.(int)
	var limit int
	if limitIn != nil {
		limit = limitIn.(int)
	}
	return &ast.SortProc{ast.Node{"SortProc"}, limit, fields, sortdir}
}

func makeTopProc(fieldsIn, limitIn, flushIn interface{}) *ast.TopProc {
	fields := stringArray(fieldsIn)
	var limit int
	if limitIn != nil {
		limit = limitIn.(int)
	}
	flush := flushIn != nil
	return &ast.TopProc{ast.Node{"TopProc"}, limit, fields, flush}
}

func makeCutProc(fieldsIn interface{}) *ast.CutProc {
	fields := stringArray(fieldsIn)
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

func makeReducer(opIn, varIn, fieldIn interface{}) *ast.Reducer {
	var field string
	if fieldIn != nil {
		field = fieldIn.(string)
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

	keys := stringArray(keysIn)
	reducers := reducersArray(reducersIn)

	return &ast.GroupByProc{
		Node:     ast.Node{"GroupByProc"},
		Duration: duration,
		Limit:    limit,
		Keys:     keys,
		Reducers: reducers,
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

var g = &grammar{
	rules: []*rule{
		{
			name: "start",
			pos:  position{line: 303, col: 1, offset: 8698},
			expr: &actionExpr{
				pos: position{line: 303, col: 9, offset: 8706},
				run: (*parser).callonstart1,
				expr: &seqExpr{
					pos: position{line: 303, col: 9, offset: 8706},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 303, col: 9, offset: 8706},
							expr: &ruleRefExpr{
								pos:  position{line: 303, col: 9, offset: 8706},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 303, col: 12, offset: 8709},
							label: "ast",
							expr: &ruleRefExpr{
								pos:  position{line: 303, col: 16, offset: 8713},
								name: "boomCommand",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 303, col: 28, offset: 8725},
							expr: &ruleRefExpr{
								pos:  position{line: 303, col: 28, offset: 8725},
								name: "_",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 303, col: 31, offset: 8728},
							name: "EOF",
						},
					},
				},
			},
		},
		{
			name: "boomCommand",
			pos:  position{line: 305, col: 1, offset: 8753},
			expr: &choiceExpr{
				pos: position{line: 306, col: 5, offset: 8769},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 306, col: 5, offset: 8769},
						run: (*parser).callonboomCommand2,
						expr: &labeledExpr{
							pos:   position{line: 306, col: 5, offset: 8769},
							label: "procs",
							expr: &ruleRefExpr{
								pos:  position{line: 306, col: 11, offset: 8775},
								name: "procChain",
							},
						},
					},
					&actionExpr{
						pos: position{line: 310, col: 5, offset: 8948},
						run: (*parser).callonboomCommand5,
						expr: &seqExpr{
							pos: position{line: 310, col: 5, offset: 8948},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 310, col: 5, offset: 8948},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 310, col: 7, offset: 8950},
										name: "search",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 310, col: 14, offset: 8957},
									expr: &ruleRefExpr{
										pos:  position{line: 310, col: 14, offset: 8957},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 310, col: 17, offset: 8960},
									label: "rest",
									expr: &zeroOrMoreExpr{
										pos: position{line: 310, col: 22, offset: 8965},
										expr: &ruleRefExpr{
											pos:  position{line: 310, col: 22, offset: 8965},
											name: "chainedProc",
										},
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 317, col: 5, offset: 9175},
						run: (*parser).callonboomCommand14,
						expr: &labeledExpr{
							pos:   position{line: 317, col: 5, offset: 9175},
							label: "s",
							expr: &ruleRefExpr{
								pos:  position{line: 317, col: 7, offset: 9177},
								name: "search",
							},
						},
					},
				},
			},
		},
		{
			name: "procChain",
			pos:  position{line: 321, col: 1, offset: 9248},
			expr: &actionExpr{
				pos: position{line: 322, col: 5, offset: 9262},
				run: (*parser).callonprocChain1,
				expr: &seqExpr{
					pos: position{line: 322, col: 5, offset: 9262},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 322, col: 5, offset: 9262},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 322, col: 11, offset: 9268},
								name: "proc",
							},
						},
						&labeledExpr{
							pos:   position{line: 322, col: 16, offset: 9273},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 322, col: 21, offset: 9278},
								expr: &ruleRefExpr{
									pos:  position{line: 322, col: 21, offset: 9278},
									name: "chainedProc",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "chainedProc",
			pos:  position{line: 330, col: 1, offset: 9464},
			expr: &actionExpr{
				pos: position{line: 330, col: 15, offset: 9478},
				run: (*parser).callonchainedProc1,
				expr: &seqExpr{
					pos: position{line: 330, col: 15, offset: 9478},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 330, col: 15, offset: 9478},
							expr: &ruleRefExpr{
								pos:  position{line: 330, col: 15, offset: 9478},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 330, col: 18, offset: 9481},
							val:        "|",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 330, col: 22, offset: 9485},
							expr: &ruleRefExpr{
								pos:  position{line: 330, col: 22, offset: 9485},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 330, col: 25, offset: 9488},
							label: "p",
							expr: &ruleRefExpr{
								pos:  position{line: 330, col: 27, offset: 9490},
								name: "proc",
							},
						},
					},
				},
			},
		},
		{
			name: "search",
			pos:  position{line: 332, col: 1, offset: 9514},
			expr: &actionExpr{
				pos: position{line: 333, col: 5, offset: 9525},
				run: (*parser).callonsearch1,
				expr: &labeledExpr{
					pos:   position{line: 333, col: 5, offset: 9525},
					label: "expr",
					expr: &ruleRefExpr{
						pos:  position{line: 333, col: 10, offset: 9530},
						name: "searchExpr",
					},
				},
			},
		},
		{
			name: "searchExpr",
			pos:  position{line: 337, col: 1, offset: 9589},
			expr: &actionExpr{
				pos: position{line: 338, col: 5, offset: 9604},
				run: (*parser).callonsearchExpr1,
				expr: &seqExpr{
					pos: position{line: 338, col: 5, offset: 9604},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 338, col: 5, offset: 9604},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 338, col: 11, offset: 9610},
								name: "searchTerm",
							},
						},
						&labeledExpr{
							pos:   position{line: 338, col: 22, offset: 9621},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 338, col: 27, offset: 9626},
								expr: &ruleRefExpr{
									pos:  position{line: 338, col: 27, offset: 9626},
									name: "oredSearchTerm",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "oredSearchTerm",
			pos:  position{line: 342, col: 1, offset: 9694},
			expr: &actionExpr{
				pos: position{line: 342, col: 18, offset: 9711},
				run: (*parser).callonoredSearchTerm1,
				expr: &seqExpr{
					pos: position{line: 342, col: 18, offset: 9711},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 342, col: 18, offset: 9711},
							name: "_",
						},
						&ruleRefExpr{
							pos:  position{line: 342, col: 20, offset: 9713},
							name: "orToken",
						},
						&ruleRefExpr{
							pos:  position{line: 342, col: 28, offset: 9721},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 342, col: 30, offset: 9723},
							label: "t",
							expr: &ruleRefExpr{
								pos:  position{line: 342, col: 32, offset: 9725},
								name: "searchTerm",
							},
						},
					},
				},
			},
		},
		{
			name: "searchTerm",
			pos:  position{line: 344, col: 1, offset: 9755},
			expr: &actionExpr{
				pos: position{line: 345, col: 5, offset: 9770},
				run: (*parser).callonsearchTerm1,
				expr: &seqExpr{
					pos: position{line: 345, col: 5, offset: 9770},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 345, col: 5, offset: 9770},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 345, col: 11, offset: 9776},
								name: "searchFactor",
							},
						},
						&labeledExpr{
							pos:   position{line: 345, col: 24, offset: 9789},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 345, col: 29, offset: 9794},
								expr: &ruleRefExpr{
									pos:  position{line: 345, col: 29, offset: 9794},
									name: "andedSearchTerm",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "andedSearchTerm",
			pos:  position{line: 349, col: 1, offset: 9864},
			expr: &actionExpr{
				pos: position{line: 349, col: 19, offset: 9882},
				run: (*parser).callonandedSearchTerm1,
				expr: &seqExpr{
					pos: position{line: 349, col: 19, offset: 9882},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 349, col: 19, offset: 9882},
							name: "_",
						},
						&zeroOrOneExpr{
							pos: position{line: 349, col: 21, offset: 9884},
							expr: &seqExpr{
								pos: position{line: 349, col: 22, offset: 9885},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 349, col: 22, offset: 9885},
										name: "andToken",
									},
									&ruleRefExpr{
										pos:  position{line: 349, col: 31, offset: 9894},
										name: "_",
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 349, col: 35, offset: 9898},
							label: "f",
							expr: &ruleRefExpr{
								pos:  position{line: 349, col: 37, offset: 9900},
								name: "searchFactor",
							},
						},
					},
				},
			},
		},
		{
			name: "searchFactor",
			pos:  position{line: 351, col: 1, offset: 9932},
			expr: &choiceExpr{
				pos: position{line: 352, col: 5, offset: 9949},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 352, col: 5, offset: 9949},
						run: (*parser).callonsearchFactor2,
						expr: &seqExpr{
							pos: position{line: 352, col: 5, offset: 9949},
							exprs: []interface{}{
								&choiceExpr{
									pos: position{line: 352, col: 6, offset: 9950},
									alternatives: []interface{}{
										&seqExpr{
											pos: position{line: 352, col: 6, offset: 9950},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 352, col: 6, offset: 9950},
													name: "notToken",
												},
												&ruleRefExpr{
													pos:  position{line: 352, col: 15, offset: 9959},
													name: "_",
												},
											},
										},
										&seqExpr{
											pos: position{line: 352, col: 19, offset: 9963},
											exprs: []interface{}{
												&litMatcher{
													pos:        position{line: 352, col: 19, offset: 9963},
													val:        "!",
													ignoreCase: false,
												},
												&zeroOrOneExpr{
													pos: position{line: 352, col: 23, offset: 9967},
													expr: &ruleRefExpr{
														pos:  position{line: 352, col: 23, offset: 9967},
														name: "_",
													},
												},
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 352, col: 27, offset: 9971},
									label: "e",
									expr: &ruleRefExpr{
										pos:  position{line: 352, col: 29, offset: 9973},
										name: "searchExpr",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 355, col: 5, offset: 10032},
						run: (*parser).callonsearchFactor14,
						expr: &seqExpr{
							pos: position{line: 355, col: 5, offset: 10032},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 355, col: 5, offset: 10032},
									expr: &litMatcher{
										pos:        position{line: 355, col: 7, offset: 10034},
										val:        "-",
										ignoreCase: false,
									},
								},
								&labeledExpr{
									pos:   position{line: 355, col: 12, offset: 10039},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 355, col: 14, offset: 10041},
										name: "searchPred",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 356, col: 5, offset: 10074},
						run: (*parser).callonsearchFactor20,
						expr: &seqExpr{
							pos: position{line: 356, col: 5, offset: 10074},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 356, col: 5, offset: 10074},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 356, col: 9, offset: 10078},
									expr: &ruleRefExpr{
										pos:  position{line: 356, col: 9, offset: 10078},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 356, col: 12, offset: 10081},
									label: "expr",
									expr: &ruleRefExpr{
										pos:  position{line: 356, col: 17, offset: 10086},
										name: "searchExpr",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 356, col: 28, offset: 10097},
									expr: &ruleRefExpr{
										pos:  position{line: 356, col: 28, offset: 10097},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 356, col: 31, offset: 10100},
									val:        ")",
									ignoreCase: false,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "searchPred",
			pos:  position{line: 358, col: 1, offset: 10126},
			expr: &choiceExpr{
				pos: position{line: 359, col: 5, offset: 10141},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 359, col: 5, offset: 10141},
						run: (*parser).callonsearchPred2,
						expr: &seqExpr{
							pos: position{line: 359, col: 5, offset: 10141},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 359, col: 5, offset: 10141},
									val:        "*",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 359, col: 9, offset: 10145},
									expr: &ruleRefExpr{
										pos:  position{line: 359, col: 9, offset: 10145},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 359, col: 12, offset: 10148},
									label: "fieldComparator",
									expr: &ruleRefExpr{
										pos:  position{line: 359, col: 28, offset: 10164},
										name: "equalityToken",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 359, col: 42, offset: 10178},
									expr: &ruleRefExpr{
										pos:  position{line: 359, col: 42, offset: 10178},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 359, col: 45, offset: 10181},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 359, col: 47, offset: 10183},
										name: "searchValue",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 362, col: 5, offset: 10260},
						run: (*parser).callonsearchPred13,
						expr: &litMatcher{
							pos:        position{line: 362, col: 5, offset: 10260},
							val:        "*",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 365, col: 5, offset: 10319},
						run: (*parser).callonsearchPred15,
						expr: &seqExpr{
							pos: position{line: 365, col: 5, offset: 10319},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 365, col: 5, offset: 10319},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 365, col: 7, offset: 10321},
										name: "fieldExpr",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 365, col: 17, offset: 10331},
									expr: &ruleRefExpr{
										pos:  position{line: 365, col: 17, offset: 10331},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 365, col: 20, offset: 10334},
									label: "fieldComparator",
									expr: &ruleRefExpr{
										pos:  position{line: 365, col: 36, offset: 10350},
										name: "equalityToken",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 365, col: 50, offset: 10364},
									expr: &ruleRefExpr{
										pos:  position{line: 365, col: 50, offset: 10364},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 365, col: 53, offset: 10367},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 365, col: 55, offset: 10369},
										name: "searchValue",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 368, col: 5, offset: 10451},
						run: (*parser).callonsearchPred27,
						expr: &seqExpr{
							pos: position{line: 368, col: 5, offset: 10451},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 368, col: 5, offset: 10451},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 368, col: 7, offset: 10453},
										name: "searchValue",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 368, col: 19, offset: 10465},
									expr: &ruleRefExpr{
										pos:  position{line: 368, col: 19, offset: 10465},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 368, col: 22, offset: 10468},
									name: "inToken",
								},
								&zeroOrOneExpr{
									pos: position{line: 368, col: 30, offset: 10476},
									expr: &ruleRefExpr{
										pos:  position{line: 368, col: 30, offset: 10476},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 368, col: 33, offset: 10479},
									val:        "*",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 371, col: 5, offset: 10537},
						run: (*parser).callonsearchPred37,
						expr: &seqExpr{
							pos: position{line: 371, col: 5, offset: 10537},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 371, col: 5, offset: 10537},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 371, col: 7, offset: 10539},
										name: "searchValue",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 371, col: 19, offset: 10551},
									expr: &ruleRefExpr{
										pos:  position{line: 371, col: 19, offset: 10551},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 371, col: 22, offset: 10554},
									name: "inToken",
								},
								&zeroOrOneExpr{
									pos: position{line: 371, col: 30, offset: 10562},
									expr: &ruleRefExpr{
										pos:  position{line: 371, col: 30, offset: 10562},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 371, col: 33, offset: 10565},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 371, col: 35, offset: 10567},
										name: "fieldReference",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 374, col: 5, offset: 10641},
						run: (*parser).callonsearchPred48,
						expr: &labeledExpr{
							pos:   position{line: 374, col: 5, offset: 10641},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 374, col: 7, offset: 10643},
								name: "searchValue",
							},
						},
					},
				},
			},
		},
		{
			name: "searchValue",
			pos:  position{line: 383, col: 1, offset: 10937},
			expr: &choiceExpr{
				pos: position{line: 384, col: 5, offset: 10953},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 384, col: 5, offset: 10953},
						run: (*parser).callonsearchValue2,
						expr: &labeledExpr{
							pos:   position{line: 384, col: 5, offset: 10953},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 384, col: 7, offset: 10955},
								name: "quotedString",
							},
						},
					},
					&actionExpr{
						pos: position{line: 387, col: 5, offset: 11026},
						run: (*parser).callonsearchValue5,
						expr: &labeledExpr{
							pos:   position{line: 387, col: 5, offset: 11026},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 387, col: 7, offset: 11028},
								name: "reString",
							},
						},
					},
					&actionExpr{
						pos: position{line: 390, col: 5, offset: 11095},
						run: (*parser).callonsearchValue8,
						expr: &labeledExpr{
							pos:   position{line: 390, col: 5, offset: 11095},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 390, col: 7, offset: 11097},
								name: "port",
							},
						},
					},
					&actionExpr{
						pos: position{line: 393, col: 5, offset: 11156},
						run: (*parser).callonsearchValue11,
						expr: &labeledExpr{
							pos:   position{line: 393, col: 5, offset: 11156},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 393, col: 7, offset: 11158},
								name: "ip6subnet",
							},
						},
					},
					&actionExpr{
						pos: position{line: 396, col: 5, offset: 11226},
						run: (*parser).callonsearchValue14,
						expr: &labeledExpr{
							pos:   position{line: 396, col: 5, offset: 11226},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 396, col: 7, offset: 11228},
								name: "ip6addr",
							},
						},
					},
					&actionExpr{
						pos: position{line: 399, col: 5, offset: 11292},
						run: (*parser).callonsearchValue17,
						expr: &labeledExpr{
							pos:   position{line: 399, col: 5, offset: 11292},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 399, col: 7, offset: 11294},
								name: "subnet",
							},
						},
					},
					&actionExpr{
						pos: position{line: 402, col: 5, offset: 11359},
						run: (*parser).callonsearchValue20,
						expr: &labeledExpr{
							pos:   position{line: 402, col: 5, offset: 11359},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 402, col: 7, offset: 11361},
								name: "addr",
							},
						},
					},
					&actionExpr{
						pos: position{line: 405, col: 5, offset: 11422},
						run: (*parser).callonsearchValue23,
						expr: &labeledExpr{
							pos:   position{line: 405, col: 5, offset: 11422},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 405, col: 7, offset: 11424},
								name: "sdouble",
							},
						},
					},
					&actionExpr{
						pos: position{line: 408, col: 5, offset: 11490},
						run: (*parser).callonsearchValue26,
						expr: &seqExpr{
							pos: position{line: 408, col: 5, offset: 11490},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 408, col: 5, offset: 11490},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 408, col: 7, offset: 11492},
										name: "sinteger",
									},
								},
								&notExpr{
									pos: position{line: 408, col: 16, offset: 11501},
									expr: &ruleRefExpr{
										pos:  position{line: 408, col: 17, offset: 11502},
										name: "boomWord",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 411, col: 5, offset: 11566},
						run: (*parser).callonsearchValue32,
						expr: &seqExpr{
							pos: position{line: 411, col: 5, offset: 11566},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 411, col: 5, offset: 11566},
									expr: &seqExpr{
										pos: position{line: 411, col: 7, offset: 11568},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 411, col: 7, offset: 11568},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 411, col: 22, offset: 11583},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 411, col: 25, offset: 11586},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 411, col: 27, offset: 11588},
										name: "booleanLiteral",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 412, col: 5, offset: 11625},
						run: (*parser).callonsearchValue40,
						expr: &seqExpr{
							pos: position{line: 412, col: 5, offset: 11625},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 412, col: 5, offset: 11625},
									expr: &seqExpr{
										pos: position{line: 412, col: 7, offset: 11627},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 412, col: 7, offset: 11627},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 412, col: 22, offset: 11642},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 412, col: 25, offset: 11645},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 412, col: 27, offset: 11647},
										name: "unsetLiteral",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 413, col: 5, offset: 11682},
						run: (*parser).callonsearchValue48,
						expr: &seqExpr{
							pos: position{line: 413, col: 5, offset: 11682},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 413, col: 5, offset: 11682},
									expr: &seqExpr{
										pos: position{line: 413, col: 7, offset: 11684},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 413, col: 7, offset: 11684},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 413, col: 22, offset: 11699},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 413, col: 25, offset: 11702},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 413, col: 27, offset: 11704},
										name: "boomWord",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "searchKeywords",
			pos:  position{line: 421, col: 1, offset: 11928},
			expr: &choiceExpr{
				pos: position{line: 422, col: 5, offset: 11947},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 422, col: 5, offset: 11947},
						name: "andToken",
					},
					&ruleRefExpr{
						pos:  position{line: 423, col: 5, offset: 11960},
						name: "orToken",
					},
					&ruleRefExpr{
						pos:  position{line: 424, col: 5, offset: 11972},
						name: "inToken",
					},
				},
			},
		},
		{
			name: "booleanLiteral",
			pos:  position{line: 426, col: 1, offset: 11981},
			expr: &choiceExpr{
				pos: position{line: 427, col: 5, offset: 12000},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 427, col: 5, offset: 12000},
						run: (*parser).callonbooleanLiteral2,
						expr: &litMatcher{
							pos:        position{line: 427, col: 5, offset: 12000},
							val:        "true",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 428, col: 5, offset: 12068},
						run: (*parser).callonbooleanLiteral4,
						expr: &litMatcher{
							pos:        position{line: 428, col: 5, offset: 12068},
							val:        "false",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "unsetLiteral",
			pos:  position{line: 430, col: 1, offset: 12134},
			expr: &actionExpr{
				pos: position{line: 431, col: 5, offset: 12151},
				run: (*parser).callonunsetLiteral1,
				expr: &litMatcher{
					pos:        position{line: 431, col: 5, offset: 12151},
					val:        "nil",
					ignoreCase: false,
				},
			},
		},
		{
			name: "procList",
			pos:  position{line: 433, col: 1, offset: 12212},
			expr: &actionExpr{
				pos: position{line: 434, col: 5, offset: 12225},
				run: (*parser).callonprocList1,
				expr: &seqExpr{
					pos: position{line: 434, col: 5, offset: 12225},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 434, col: 5, offset: 12225},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 434, col: 11, offset: 12231},
								name: "procChain",
							},
						},
						&labeledExpr{
							pos:   position{line: 434, col: 21, offset: 12241},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 434, col: 26, offset: 12246},
								expr: &ruleRefExpr{
									pos:  position{line: 434, col: 26, offset: 12246},
									name: "parallelChain",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "parallelChain",
			pos:  position{line: 443, col: 1, offset: 12470},
			expr: &actionExpr{
				pos: position{line: 444, col: 5, offset: 12488},
				run: (*parser).callonparallelChain1,
				expr: &seqExpr{
					pos: position{line: 444, col: 5, offset: 12488},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 444, col: 5, offset: 12488},
							expr: &ruleRefExpr{
								pos:  position{line: 444, col: 5, offset: 12488},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 444, col: 8, offset: 12491},
							val:        ";",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 444, col: 12, offset: 12495},
							expr: &ruleRefExpr{
								pos:  position{line: 444, col: 12, offset: 12495},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 444, col: 15, offset: 12498},
							label: "ch",
							expr: &ruleRefExpr{
								pos:  position{line: 444, col: 18, offset: 12501},
								name: "procChain",
							},
						},
					},
				},
			},
		},
		{
			name: "proc",
			pos:  position{line: 446, col: 1, offset: 12551},
			expr: &choiceExpr{
				pos: position{line: 447, col: 5, offset: 12560},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 447, col: 5, offset: 12560},
						name: "simpleProc",
					},
					&ruleRefExpr{
						pos:  position{line: 448, col: 5, offset: 12575},
						name: "reducerProc",
					},
					&actionExpr{
						pos: position{line: 449, col: 5, offset: 12591},
						run: (*parser).callonproc4,
						expr: &seqExpr{
							pos: position{line: 449, col: 5, offset: 12591},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 449, col: 5, offset: 12591},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 449, col: 9, offset: 12595},
									expr: &ruleRefExpr{
										pos:  position{line: 449, col: 9, offset: 12595},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 449, col: 12, offset: 12598},
									label: "proc",
									expr: &ruleRefExpr{
										pos:  position{line: 449, col: 17, offset: 12603},
										name: "procList",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 449, col: 26, offset: 12612},
									expr: &ruleRefExpr{
										pos:  position{line: 449, col: 26, offset: 12612},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 449, col: 29, offset: 12615},
									val:        ")",
									ignoreCase: false,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "groupBy",
			pos:  position{line: 453, col: 1, offset: 12651},
			expr: &actionExpr{
				pos: position{line: 454, col: 5, offset: 12663},
				run: (*parser).callongroupBy1,
				expr: &seqExpr{
					pos: position{line: 454, col: 5, offset: 12663},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 454, col: 5, offset: 12663},
							val:        "by",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 454, col: 11, offset: 12669},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 454, col: 13, offset: 12671},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 454, col: 18, offset: 12676},
								name: "fieldList",
							},
						},
					},
				},
			},
		},
		{
			name: "everyDur",
			pos:  position{line: 456, col: 1, offset: 12708},
			expr: &actionExpr{
				pos: position{line: 457, col: 5, offset: 12721},
				run: (*parser).calloneveryDur1,
				expr: &seqExpr{
					pos: position{line: 457, col: 5, offset: 12721},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 457, col: 5, offset: 12721},
							val:        "every",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 457, col: 14, offset: 12730},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 457, col: 16, offset: 12732},
							label: "dur",
							expr: &ruleRefExpr{
								pos:  position{line: 457, col: 20, offset: 12736},
								name: "duration",
							},
						},
					},
				},
			},
		},
		{
			name: "equalityToken",
			pos:  position{line: 459, col: 1, offset: 12766},
			expr: &choiceExpr{
				pos: position{line: 460, col: 5, offset: 12784},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 460, col: 5, offset: 12784},
						run: (*parser).callonequalityToken2,
						expr: &litMatcher{
							pos:        position{line: 460, col: 5, offset: 12784},
							val:        "=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 461, col: 5, offset: 12814},
						run: (*parser).callonequalityToken4,
						expr: &litMatcher{
							pos:        position{line: 461, col: 5, offset: 12814},
							val:        "!=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 462, col: 5, offset: 12846},
						run: (*parser).callonequalityToken6,
						expr: &litMatcher{
							pos:        position{line: 462, col: 5, offset: 12846},
							val:        "<=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 463, col: 5, offset: 12877},
						run: (*parser).callonequalityToken8,
						expr: &litMatcher{
							pos:        position{line: 463, col: 5, offset: 12877},
							val:        ">=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 464, col: 5, offset: 12908},
						run: (*parser).callonequalityToken10,
						expr: &litMatcher{
							pos:        position{line: 464, col: 5, offset: 12908},
							val:        "<",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 465, col: 5, offset: 12937},
						run: (*parser).callonequalityToken12,
						expr: &litMatcher{
							pos:        position{line: 465, col: 5, offset: 12937},
							val:        ">",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "types",
			pos:  position{line: 467, col: 1, offset: 12963},
			expr: &choiceExpr{
				pos: position{line: 468, col: 5, offset: 12973},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 468, col: 5, offset: 12973},
						val:        "bool",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 469, col: 5, offset: 12984},
						val:        "int",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 470, col: 5, offset: 12994},
						val:        "count",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 471, col: 5, offset: 13006},
						val:        "double",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 472, col: 5, offset: 13019},
						val:        "string",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 473, col: 5, offset: 13032},
						val:        "addr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 474, col: 5, offset: 13043},
						val:        "subnet",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 475, col: 5, offset: 13056},
						val:        "port",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "dash",
			pos:  position{line: 477, col: 1, offset: 13064},
			expr: &choiceExpr{
				pos: position{line: 477, col: 8, offset: 13071},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 477, col: 8, offset: 13071},
						val:        "-",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 477, col: 14, offset: 13077},
						val:        "—",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 477, col: 25, offset: 13088},
						val:        "–",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 477, col: 36, offset: 13099},
						exprs: []interface{}{
							&litMatcher{
								pos:        position{line: 477, col: 36, offset: 13099},
								val:        "to",
								ignoreCase: false,
							},
							&andExpr{
								pos: position{line: 477, col: 40, offset: 13103},
								expr: &ruleRefExpr{
									pos:  position{line: 477, col: 42, offset: 13105},
									name: "_",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "andToken",
			pos:  position{line: 479, col: 1, offset: 13109},
			expr: &litMatcher{
				pos:        position{line: 479, col: 12, offset: 13120},
				val:        "and",
				ignoreCase: false,
			},
		},
		{
			name: "orToken",
			pos:  position{line: 480, col: 1, offset: 13126},
			expr: &litMatcher{
				pos:        position{line: 480, col: 11, offset: 13136},
				val:        "or",
				ignoreCase: false,
			},
		},
		{
			name: "inToken",
			pos:  position{line: 481, col: 1, offset: 13141},
			expr: &litMatcher{
				pos:        position{line: 481, col: 11, offset: 13151},
				val:        "in",
				ignoreCase: false,
			},
		},
		{
			name: "notToken",
			pos:  position{line: 482, col: 1, offset: 13156},
			expr: &litMatcher{
				pos:        position{line: 482, col: 12, offset: 13167},
				val:        "not",
				ignoreCase: false,
			},
		},
		{
			name: "fieldName",
			pos:  position{line: 484, col: 1, offset: 13174},
			expr: &actionExpr{
				pos: position{line: 484, col: 13, offset: 13186},
				run: (*parser).callonfieldName1,
				expr: &seqExpr{
					pos: position{line: 484, col: 13, offset: 13186},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 484, col: 13, offset: 13186},
							name: "fieldNameStart",
						},
						&zeroOrMoreExpr{
							pos: position{line: 484, col: 28, offset: 13201},
							expr: &ruleRefExpr{
								pos:  position{line: 484, col: 28, offset: 13201},
								name: "fieldNameRest",
							},
						},
					},
				},
			},
		},
		{
			name: "fieldNameStart",
			pos:  position{line: 486, col: 1, offset: 13248},
			expr: &charClassMatcher{
				pos:        position{line: 486, col: 18, offset: 13265},
				val:        "[A-Za-z_$]",
				chars:      []rune{'_', '$'},
				ranges:     []rune{'A', 'Z', 'a', 'z'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "fieldNameRest",
			pos:  position{line: 487, col: 1, offset: 13276},
			expr: &choiceExpr{
				pos: position{line: 487, col: 17, offset: 13292},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 487, col: 17, offset: 13292},
						name: "fieldNameStart",
					},
					&charClassMatcher{
						pos:        position{line: 487, col: 34, offset: 13309},
						val:        "[0-9]",
						ranges:     []rune{'0', '9'},
						ignoreCase: false,
						inverted:   false,
					},
				},
			},
		},
		{
			name: "fieldReference",
			pos:  position{line: 489, col: 1, offset: 13316},
			expr: &actionExpr{
				pos: position{line: 490, col: 4, offset: 13334},
				run: (*parser).callonfieldReference1,
				expr: &seqExpr{
					pos: position{line: 490, col: 4, offset: 13334},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 490, col: 4, offset: 13334},
							label: "base",
							expr: &ruleRefExpr{
								pos:  position{line: 490, col: 9, offset: 13339},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 490, col: 19, offset: 13349},
							label: "derefs",
							expr: &zeroOrMoreExpr{
								pos: position{line: 490, col: 26, offset: 13356},
								expr: &choiceExpr{
									pos: position{line: 491, col: 8, offset: 13365},
									alternatives: []interface{}{
										&actionExpr{
											pos: position{line: 491, col: 8, offset: 13365},
											run: (*parser).callonfieldReference8,
											expr: &seqExpr{
												pos: position{line: 491, col: 8, offset: 13365},
												exprs: []interface{}{
													&litMatcher{
														pos:        position{line: 491, col: 8, offset: 13365},
														val:        ".",
														ignoreCase: false,
													},
													&labeledExpr{
														pos:   position{line: 491, col: 12, offset: 13369},
														label: "field",
														expr: &ruleRefExpr{
															pos:  position{line: 491, col: 18, offset: 13375},
															name: "fieldName",
														},
													},
												},
											},
										},
										&actionExpr{
											pos: position{line: 492, col: 8, offset: 13456},
											run: (*parser).callonfieldReference13,
											expr: &seqExpr{
												pos: position{line: 492, col: 8, offset: 13456},
												exprs: []interface{}{
													&litMatcher{
														pos:        position{line: 492, col: 8, offset: 13456},
														val:        "[",
														ignoreCase: false,
													},
													&labeledExpr{
														pos:   position{line: 492, col: 12, offset: 13460},
														label: "index",
														expr: &ruleRefExpr{
															pos:  position{line: 492, col: 18, offset: 13466},
															name: "sinteger",
														},
													},
													&litMatcher{
														pos:        position{line: 492, col: 27, offset: 13475},
														val:        "]",
														ignoreCase: false,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "fieldExpr",
			pos:  position{line: 497, col: 1, offset: 13591},
			expr: &choiceExpr{
				pos: position{line: 498, col: 5, offset: 13605},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 498, col: 5, offset: 13605},
						run: (*parser).callonfieldExpr2,
						expr: &seqExpr{
							pos: position{line: 498, col: 5, offset: 13605},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 498, col: 5, offset: 13605},
									label: "op",
									expr: &ruleRefExpr{
										pos:  position{line: 498, col: 8, offset: 13608},
										name: "fieldOp",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 498, col: 16, offset: 13616},
									expr: &ruleRefExpr{
										pos:  position{line: 498, col: 16, offset: 13616},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 498, col: 19, offset: 13619},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 498, col: 23, offset: 13623},
									expr: &ruleRefExpr{
										pos:  position{line: 498, col: 23, offset: 13623},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 498, col: 26, offset: 13626},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 498, col: 32, offset: 13632},
										name: "fieldReference",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 498, col: 47, offset: 13647},
									expr: &ruleRefExpr{
										pos:  position{line: 498, col: 47, offset: 13647},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 498, col: 50, offset: 13650},
									val:        ")",
									ignoreCase: false,
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 501, col: 5, offset: 13714},
						name: "fieldReference",
					},
				},
			},
		},
		{
			name: "fieldOp",
			pos:  position{line: 503, col: 1, offset: 13730},
			expr: &actionExpr{
				pos: position{line: 504, col: 5, offset: 13742},
				run: (*parser).callonfieldOp1,
				expr: &litMatcher{
					pos:        position{line: 504, col: 5, offset: 13742},
					val:        "len",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldList",
			pos:  position{line: 506, col: 1, offset: 13772},
			expr: &actionExpr{
				pos: position{line: 507, col: 5, offset: 13786},
				run: (*parser).callonfieldList1,
				expr: &seqExpr{
					pos: position{line: 507, col: 5, offset: 13786},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 507, col: 5, offset: 13786},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 507, col: 11, offset: 13792},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 507, col: 21, offset: 13802},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 507, col: 26, offset: 13807},
								expr: &seqExpr{
									pos: position{line: 507, col: 27, offset: 13808},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 507, col: 27, offset: 13808},
											expr: &ruleRefExpr{
												pos:  position{line: 507, col: 27, offset: 13808},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 507, col: 30, offset: 13811},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 507, col: 34, offset: 13815},
											expr: &ruleRefExpr{
												pos:  position{line: 507, col: 34, offset: 13815},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 507, col: 37, offset: 13818},
											name: "fieldName",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "countOp",
			pos:  position{line: 517, col: 1, offset: 14013},
			expr: &actionExpr{
				pos: position{line: 518, col: 5, offset: 14025},
				run: (*parser).calloncountOp1,
				expr: &litMatcher{
					pos:        position{line: 518, col: 5, offset: 14025},
					val:        "count",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldReducerOp",
			pos:  position{line: 520, col: 1, offset: 14059},
			expr: &choiceExpr{
				pos: position{line: 521, col: 5, offset: 14078},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 521, col: 5, offset: 14078},
						run: (*parser).callonfieldReducerOp2,
						expr: &litMatcher{
							pos:        position{line: 521, col: 5, offset: 14078},
							val:        "sum",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 522, col: 5, offset: 14112},
						run: (*parser).callonfieldReducerOp4,
						expr: &litMatcher{
							pos:        position{line: 522, col: 5, offset: 14112},
							val:        "avg",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 523, col: 5, offset: 14146},
						run: (*parser).callonfieldReducerOp6,
						expr: &litMatcher{
							pos:        position{line: 523, col: 5, offset: 14146},
							val:        "stdev",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 524, col: 5, offset: 14183},
						run: (*parser).callonfieldReducerOp8,
						expr: &litMatcher{
							pos:        position{line: 524, col: 5, offset: 14183},
							val:        "sd",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 525, col: 5, offset: 14219},
						run: (*parser).callonfieldReducerOp10,
						expr: &litMatcher{
							pos:        position{line: 525, col: 5, offset: 14219},
							val:        "var",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 526, col: 5, offset: 14253},
						run: (*parser).callonfieldReducerOp12,
						expr: &litMatcher{
							pos:        position{line: 526, col: 5, offset: 14253},
							val:        "entropy",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 527, col: 5, offset: 14294},
						run: (*parser).callonfieldReducerOp14,
						expr: &litMatcher{
							pos:        position{line: 527, col: 5, offset: 14294},
							val:        "min",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 528, col: 5, offset: 14328},
						run: (*parser).callonfieldReducerOp16,
						expr: &litMatcher{
							pos:        position{line: 528, col: 5, offset: 14328},
							val:        "max",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 529, col: 5, offset: 14362},
						run: (*parser).callonfieldReducerOp18,
						expr: &litMatcher{
							pos:        position{line: 529, col: 5, offset: 14362},
							val:        "first",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 530, col: 5, offset: 14400},
						run: (*parser).callonfieldReducerOp20,
						expr: &litMatcher{
							pos:        position{line: 530, col: 5, offset: 14400},
							val:        "last",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "paddedFieldName",
			pos:  position{line: 532, col: 1, offset: 14433},
			expr: &actionExpr{
				pos: position{line: 532, col: 19, offset: 14451},
				run: (*parser).callonpaddedFieldName1,
				expr: &seqExpr{
					pos: position{line: 532, col: 19, offset: 14451},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 532, col: 19, offset: 14451},
							expr: &ruleRefExpr{
								pos:  position{line: 532, col: 19, offset: 14451},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 532, col: 22, offset: 14454},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 532, col: 28, offset: 14460},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 532, col: 38, offset: 14470},
							expr: &ruleRefExpr{
								pos:  position{line: 532, col: 38, offset: 14470},
								name: "_",
							},
						},
					},
				},
			},
		},
		{
			name: "countReducer",
			pos:  position{line: 534, col: 1, offset: 14496},
			expr: &actionExpr{
				pos: position{line: 535, col: 5, offset: 14513},
				run: (*parser).calloncountReducer1,
				expr: &seqExpr{
					pos: position{line: 535, col: 5, offset: 14513},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 535, col: 5, offset: 14513},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 535, col: 8, offset: 14516},
								name: "countOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 535, col: 16, offset: 14524},
							expr: &ruleRefExpr{
								pos:  position{line: 535, col: 16, offset: 14524},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 535, col: 19, offset: 14527},
							val:        "(",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 535, col: 23, offset: 14531},
							label: "field",
							expr: &zeroOrOneExpr{
								pos: position{line: 535, col: 29, offset: 14537},
								expr: &ruleRefExpr{
									pos:  position{line: 535, col: 29, offset: 14537},
									name: "paddedFieldName",
								},
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 535, col: 47, offset: 14555},
							expr: &ruleRefExpr{
								pos:  position{line: 535, col: 47, offset: 14555},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 535, col: 50, offset: 14558},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "fieldReducer",
			pos:  position{line: 539, col: 1, offset: 14617},
			expr: &actionExpr{
				pos: position{line: 540, col: 5, offset: 14634},
				run: (*parser).callonfieldReducer1,
				expr: &seqExpr{
					pos: position{line: 540, col: 5, offset: 14634},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 540, col: 5, offset: 14634},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 540, col: 8, offset: 14637},
								name: "fieldReducerOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 540, col: 23, offset: 14652},
							expr: &ruleRefExpr{
								pos:  position{line: 540, col: 23, offset: 14652},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 540, col: 26, offset: 14655},
							val:        "(",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 540, col: 30, offset: 14659},
							expr: &ruleRefExpr{
								pos:  position{line: 540, col: 30, offset: 14659},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 540, col: 33, offset: 14662},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 540, col: 39, offset: 14668},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 540, col: 50, offset: 14679},
							expr: &ruleRefExpr{
								pos:  position{line: 540, col: 50, offset: 14679},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 540, col: 53, offset: 14682},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reducerProc",
			pos:  position{line: 544, col: 1, offset: 14749},
			expr: &actionExpr{
				pos: position{line: 545, col: 5, offset: 14765},
				run: (*parser).callonreducerProc1,
				expr: &seqExpr{
					pos: position{line: 545, col: 5, offset: 14765},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 545, col: 5, offset: 14765},
							label: "every",
							expr: &zeroOrOneExpr{
								pos: position{line: 545, col: 11, offset: 14771},
								expr: &seqExpr{
									pos: position{line: 545, col: 12, offset: 14772},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 545, col: 12, offset: 14772},
											name: "everyDur",
										},
										&ruleRefExpr{
											pos:  position{line: 545, col: 21, offset: 14781},
											name: "_",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 545, col: 25, offset: 14785},
							label: "reducers",
							expr: &ruleRefExpr{
								pos:  position{line: 545, col: 34, offset: 14794},
								name: "reducerList",
							},
						},
						&labeledExpr{
							pos:   position{line: 545, col: 46, offset: 14806},
							label: "keys",
							expr: &zeroOrOneExpr{
								pos: position{line: 545, col: 51, offset: 14811},
								expr: &seqExpr{
									pos: position{line: 545, col: 52, offset: 14812},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 545, col: 52, offset: 14812},
											name: "_",
										},
										&ruleRefExpr{
											pos:  position{line: 545, col: 54, offset: 14814},
											name: "groupBy",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 545, col: 64, offset: 14824},
							label: "limit",
							expr: &zeroOrOneExpr{
								pos: position{line: 545, col: 70, offset: 14830},
								expr: &ruleRefExpr{
									pos:  position{line: 545, col: 70, offset: 14830},
									name: "procLimitArg",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "asClause",
			pos:  position{line: 563, col: 1, offset: 15187},
			expr: &actionExpr{
				pos: position{line: 564, col: 5, offset: 15200},
				run: (*parser).callonasClause1,
				expr: &seqExpr{
					pos: position{line: 564, col: 5, offset: 15200},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 564, col: 5, offset: 15200},
							val:        "as",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 564, col: 11, offset: 15206},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 564, col: 13, offset: 15208},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 564, col: 15, offset: 15210},
								name: "fieldName",
							},
						},
					},
				},
			},
		},
		{
			name: "reducerExpr",
			pos:  position{line: 566, col: 1, offset: 15239},
			expr: &choiceExpr{
				pos: position{line: 567, col: 5, offset: 15255},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 567, col: 5, offset: 15255},
						run: (*parser).callonreducerExpr2,
						expr: &seqExpr{
							pos: position{line: 567, col: 5, offset: 15255},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 567, col: 5, offset: 15255},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 567, col: 11, offset: 15261},
										name: "fieldName",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 567, col: 21, offset: 15271},
									expr: &ruleRefExpr{
										pos:  position{line: 567, col: 21, offset: 15271},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 567, col: 24, offset: 15274},
									val:        "=",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 567, col: 28, offset: 15278},
									expr: &ruleRefExpr{
										pos:  position{line: 567, col: 28, offset: 15278},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 567, col: 31, offset: 15281},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 567, col: 33, offset: 15283},
										name: "reducer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 570, col: 5, offset: 15346},
						run: (*parser).callonreducerExpr13,
						expr: &seqExpr{
							pos: position{line: 570, col: 5, offset: 15346},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 570, col: 5, offset: 15346},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 570, col: 7, offset: 15348},
										name: "reducer",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 570, col: 15, offset: 15356},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 570, col: 17, offset: 15358},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 570, col: 23, offset: 15364},
										name: "asClause",
									},
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 573, col: 5, offset: 15428},
						name: "reducer",
					},
				},
			},
		},
		{
			name: "reducer",
			pos:  position{line: 575, col: 1, offset: 15437},
			expr: &choiceExpr{
				pos: position{line: 576, col: 5, offset: 15449},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 576, col: 5, offset: 15449},
						name: "countReducer",
					},
					&ruleRefExpr{
						pos:  position{line: 577, col: 5, offset: 15466},
						name: "fieldReducer",
					},
				},
			},
		},
		{
			name: "reducerList",
			pos:  position{line: 579, col: 1, offset: 15480},
			expr: &actionExpr{
				pos: position{line: 580, col: 5, offset: 15496},
				run: (*parser).callonreducerList1,
				expr: &seqExpr{
					pos: position{line: 580, col: 5, offset: 15496},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 580, col: 5, offset: 15496},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 580, col: 11, offset: 15502},
								name: "reducerExpr",
							},
						},
						&labeledExpr{
							pos:   position{line: 580, col: 23, offset: 15514},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 580, col: 28, offset: 15519},
								expr: &seqExpr{
									pos: position{line: 580, col: 29, offset: 15520},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 580, col: 29, offset: 15520},
											expr: &ruleRefExpr{
												pos:  position{line: 580, col: 29, offset: 15520},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 580, col: 32, offset: 15523},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 580, col: 36, offset: 15527},
											expr: &ruleRefExpr{
												pos:  position{line: 580, col: 36, offset: 15527},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 580, col: 39, offset: 15530},
											name: "reducerExpr",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "simpleProc",
			pos:  position{line: 588, col: 1, offset: 15727},
			expr: &choiceExpr{
				pos: position{line: 589, col: 5, offset: 15742},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 589, col: 5, offset: 15742},
						name: "sort",
					},
					&ruleRefExpr{
						pos:  position{line: 590, col: 5, offset: 15751},
						name: "top",
					},
					&ruleRefExpr{
						pos:  position{line: 591, col: 5, offset: 15759},
						name: "cut",
					},
					&ruleRefExpr{
						pos:  position{line: 592, col: 5, offset: 15767},
						name: "head",
					},
					&ruleRefExpr{
						pos:  position{line: 593, col: 5, offset: 15776},
						name: "tail",
					},
					&ruleRefExpr{
						pos:  position{line: 594, col: 5, offset: 15785},
						name: "filter",
					},
					&ruleRefExpr{
						pos:  position{line: 595, col: 5, offset: 15796},
						name: "uniq",
					},
				},
			},
		},
		{
			name: "sort",
			pos:  position{line: 597, col: 1, offset: 15802},
			expr: &choiceExpr{
				pos: position{line: 598, col: 5, offset: 15811},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 598, col: 5, offset: 15811},
						run: (*parser).callonsort2,
						expr: &seqExpr{
							pos: position{line: 598, col: 5, offset: 15811},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 598, col: 5, offset: 15811},
									val:        "sort",
									ignoreCase: true,
								},
								&labeledExpr{
									pos:   position{line: 598, col: 13, offset: 15819},
									label: "rev",
									expr: &zeroOrOneExpr{
										pos: position{line: 598, col: 17, offset: 15823},
										expr: &seqExpr{
											pos: position{line: 598, col: 18, offset: 15824},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 598, col: 18, offset: 15824},
													name: "_",
												},
												&litMatcher{
													pos:        position{line: 598, col: 20, offset: 15826},
													val:        "-r",
													ignoreCase: false,
												},
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 598, col: 27, offset: 15833},
									label: "limit",
									expr: &zeroOrOneExpr{
										pos: position{line: 598, col: 33, offset: 15839},
										expr: &ruleRefExpr{
											pos:  position{line: 598, col: 33, offset: 15839},
											name: "procLimitArg",
										},
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 598, col: 48, offset: 15854},
									expr: &ruleRefExpr{
										pos:  position{line: 598, col: 48, offset: 15854},
										name: "_",
									},
								},
								&notExpr{
									pos: position{line: 598, col: 51, offset: 15857},
									expr: &litMatcher{
										pos:        position{line: 598, col: 52, offset: 15858},
										val:        "-r",
										ignoreCase: false,
									},
								},
								&labeledExpr{
									pos:   position{line: 598, col: 57, offset: 15863},
									label: "list",
									expr: &zeroOrOneExpr{
										pos: position{line: 598, col: 62, offset: 15868},
										expr: &ruleRefExpr{
											pos:  position{line: 598, col: 63, offset: 15869},
											name: "fieldList",
										},
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 603, col: 5, offset: 15995},
						run: (*parser).callonsort20,
						expr: &seqExpr{
							pos: position{line: 603, col: 5, offset: 15995},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 603, col: 5, offset: 15995},
									val:        "sort",
									ignoreCase: true,
								},
								&labeledExpr{
									pos:   position{line: 603, col: 13, offset: 16003},
									label: "limit",
									expr: &zeroOrOneExpr{
										pos: position{line: 603, col: 19, offset: 16009},
										expr: &ruleRefExpr{
											pos:  position{line: 603, col: 19, offset: 16009},
											name: "procLimitArg",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 603, col: 33, offset: 16023},
									label: "rev",
									expr: &zeroOrOneExpr{
										pos: position{line: 603, col: 37, offset: 16027},
										expr: &seqExpr{
											pos: position{line: 603, col: 38, offset: 16028},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 603, col: 38, offset: 16028},
													name: "_",
												},
												&litMatcher{
													pos:        position{line: 603, col: 40, offset: 16030},
													val:        "-r",
													ignoreCase: false,
												},
											},
										},
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 603, col: 47, offset: 16037},
									expr: &ruleRefExpr{
										pos:  position{line: 603, col: 47, offset: 16037},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 603, col: 50, offset: 16040},
									label: "list",
									expr: &zeroOrOneExpr{
										pos: position{line: 603, col: 55, offset: 16045},
										expr: &ruleRefExpr{
											pos:  position{line: 603, col: 56, offset: 16046},
											name: "fieldList",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "top",
			pos:  position{line: 609, col: 1, offset: 16169},
			expr: &actionExpr{
				pos: position{line: 610, col: 5, offset: 16177},
				run: (*parser).callontop1,
				expr: &seqExpr{
					pos: position{line: 610, col: 5, offset: 16177},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 610, col: 5, offset: 16177},
							val:        "top",
							ignoreCase: true,
						},
						&labeledExpr{
							pos:   position{line: 610, col: 12, offset: 16184},
							label: "limit",
							expr: &zeroOrOneExpr{
								pos: position{line: 610, col: 18, offset: 16190},
								expr: &ruleRefExpr{
									pos:  position{line: 610, col: 18, offset: 16190},
									name: "procLimitArg",
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 610, col: 32, offset: 16204},
							label: "flush",
							expr: &zeroOrOneExpr{
								pos: position{line: 610, col: 38, offset: 16210},
								expr: &seqExpr{
									pos: position{line: 610, col: 39, offset: 16211},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 610, col: 39, offset: 16211},
											name: "_",
										},
										&litMatcher{
											pos:        position{line: 610, col: 41, offset: 16213},
											val:        "-flush",
											ignoreCase: false,
										},
									},
								},
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 610, col: 52, offset: 16224},
							expr: &ruleRefExpr{
								pos:  position{line: 610, col: 52, offset: 16224},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 610, col: 55, offset: 16227},
							label: "list",
							expr: &zeroOrOneExpr{
								pos: position{line: 610, col: 60, offset: 16232},
								expr: &ruleRefExpr{
									pos:  position{line: 610, col: 61, offset: 16233},
									name: "fieldList",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "procLimitArg",
			pos:  position{line: 614, col: 1, offset: 16300},
			expr: &actionExpr{
				pos: position{line: 615, col: 5, offset: 16317},
				run: (*parser).callonprocLimitArg1,
				expr: &seqExpr{
					pos: position{line: 615, col: 5, offset: 16317},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 615, col: 5, offset: 16317},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 615, col: 7, offset: 16319},
							val:        "-limit",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 615, col: 16, offset: 16328},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 615, col: 18, offset: 16330},
							label: "limit",
							expr: &ruleRefExpr{
								pos:  position{line: 615, col: 24, offset: 16336},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "cut",
			pos:  position{line: 617, col: 1, offset: 16367},
			expr: &actionExpr{
				pos: position{line: 618, col: 5, offset: 16375},
				run: (*parser).calloncut1,
				expr: &seqExpr{
					pos: position{line: 618, col: 5, offset: 16375},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 618, col: 5, offset: 16375},
							val:        "cut",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 618, col: 12, offset: 16382},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 618, col: 14, offset: 16384},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 618, col: 19, offset: 16389},
								name: "fieldList",
							},
						},
					},
				},
			},
		},
		{
			name: "head",
			pos:  position{line: 619, col: 1, offset: 16433},
			expr: &choiceExpr{
				pos: position{line: 620, col: 5, offset: 16442},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 620, col: 5, offset: 16442},
						run: (*parser).callonhead2,
						expr: &seqExpr{
							pos: position{line: 620, col: 5, offset: 16442},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 620, col: 5, offset: 16442},
									val:        "head",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 620, col: 13, offset: 16450},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 620, col: 15, offset: 16452},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 620, col: 21, offset: 16458},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 621, col: 5, offset: 16506},
						run: (*parser).callonhead8,
						expr: &litMatcher{
							pos:        position{line: 621, col: 5, offset: 16506},
							val:        "head",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "tail",
			pos:  position{line: 622, col: 1, offset: 16546},
			expr: &choiceExpr{
				pos: position{line: 623, col: 5, offset: 16555},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 623, col: 5, offset: 16555},
						run: (*parser).callontail2,
						expr: &seqExpr{
							pos: position{line: 623, col: 5, offset: 16555},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 623, col: 5, offset: 16555},
									val:        "tail",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 623, col: 13, offset: 16563},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 623, col: 15, offset: 16565},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 623, col: 21, offset: 16571},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 624, col: 5, offset: 16619},
						run: (*parser).callontail8,
						expr: &litMatcher{
							pos:        position{line: 624, col: 5, offset: 16619},
							val:        "tail",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "filter",
			pos:  position{line: 626, col: 1, offset: 16660},
			expr: &actionExpr{
				pos: position{line: 627, col: 5, offset: 16671},
				run: (*parser).callonfilter1,
				expr: &seqExpr{
					pos: position{line: 627, col: 5, offset: 16671},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 627, col: 5, offset: 16671},
							val:        "filter",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 627, col: 15, offset: 16681},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 627, col: 17, offset: 16683},
							label: "expr",
							expr: &ruleRefExpr{
								pos:  position{line: 627, col: 22, offset: 16688},
								name: "searchExpr",
							},
						},
					},
				},
			},
		},
		{
			name: "uniq",
			pos:  position{line: 630, col: 1, offset: 16746},
			expr: &choiceExpr{
				pos: position{line: 631, col: 5, offset: 16755},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 631, col: 5, offset: 16755},
						run: (*parser).callonuniq2,
						expr: &seqExpr{
							pos: position{line: 631, col: 5, offset: 16755},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 631, col: 5, offset: 16755},
									val:        "uniq",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 631, col: 13, offset: 16763},
									name: "_",
								},
								&litMatcher{
									pos:        position{line: 631, col: 15, offset: 16765},
									val:        "-c",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 634, col: 5, offset: 16819},
						run: (*parser).callonuniq7,
						expr: &litMatcher{
							pos:        position{line: 634, col: 5, offset: 16819},
							val:        "uniq",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "duration",
			pos:  position{line: 638, col: 1, offset: 16874},
			expr: &choiceExpr{
				pos: position{line: 639, col: 5, offset: 16887},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 639, col: 5, offset: 16887},
						name: "seconds",
					},
					&ruleRefExpr{
						pos:  position{line: 640, col: 5, offset: 16899},
						name: "minutes",
					},
					&ruleRefExpr{
						pos:  position{line: 641, col: 5, offset: 16911},
						name: "hours",
					},
					&seqExpr{
						pos: position{line: 642, col: 5, offset: 16921},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 642, col: 5, offset: 16921},
								name: "hours",
							},
							&ruleRefExpr{
								pos:  position{line: 642, col: 11, offset: 16927},
								name: "_",
							},
							&litMatcher{
								pos:        position{line: 642, col: 13, offset: 16929},
								val:        "and",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 642, col: 19, offset: 16935},
								name: "_",
							},
							&ruleRefExpr{
								pos:  position{line: 642, col: 21, offset: 16937},
								name: "minutes",
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 643, col: 5, offset: 16949},
						name: "days",
					},
					&ruleRefExpr{
						pos:  position{line: 644, col: 5, offset: 16958},
						name: "weeks",
					},
				},
			},
		},
		{
			name: "sec_abbrev",
			pos:  position{line: 646, col: 1, offset: 16965},
			expr: &choiceExpr{
				pos: position{line: 647, col: 5, offset: 16980},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 647, col: 5, offset: 16980},
						val:        "seconds",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 648, col: 5, offset: 16994},
						val:        "second",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 649, col: 5, offset: 17007},
						val:        "secs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 650, col: 5, offset: 17018},
						val:        "sec",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 651, col: 5, offset: 17028},
						val:        "s",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "min_abbrev",
			pos:  position{line: 653, col: 1, offset: 17033},
			expr: &choiceExpr{
				pos: position{line: 654, col: 5, offset: 17048},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 654, col: 5, offset: 17048},
						val:        "minutes",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 655, col: 5, offset: 17062},
						val:        "minute",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 656, col: 5, offset: 17075},
						val:        "mins",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 657, col: 5, offset: 17086},
						val:        "min",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 658, col: 5, offset: 17096},
						val:        "m",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "hour_abbrev",
			pos:  position{line: 660, col: 1, offset: 17101},
			expr: &choiceExpr{
				pos: position{line: 661, col: 5, offset: 17117},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 661, col: 5, offset: 17117},
						val:        "hours",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 662, col: 5, offset: 17129},
						val:        "hrs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 663, col: 5, offset: 17139},
						val:        "hr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 664, col: 5, offset: 17148},
						val:        "h",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 665, col: 5, offset: 17156},
						val:        "hour",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "day_abbrev",
			pos:  position{line: 667, col: 1, offset: 17164},
			expr: &choiceExpr{
				pos: position{line: 667, col: 14, offset: 17177},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 667, col: 14, offset: 17177},
						val:        "days",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 667, col: 21, offset: 17184},
						val:        "day",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 667, col: 27, offset: 17190},
						val:        "d",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "week_abbrev",
			pos:  position{line: 668, col: 1, offset: 17194},
			expr: &choiceExpr{
				pos: position{line: 668, col: 15, offset: 17208},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 668, col: 15, offset: 17208},
						val:        "weeks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 668, col: 23, offset: 17216},
						val:        "week",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 668, col: 30, offset: 17223},
						val:        "wks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 668, col: 36, offset: 17229},
						val:        "wk",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 668, col: 41, offset: 17234},
						val:        "w",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "seconds",
			pos:  position{line: 670, col: 1, offset: 17239},
			expr: &choiceExpr{
				pos: position{line: 671, col: 5, offset: 17251},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 671, col: 5, offset: 17251},
						run: (*parser).callonseconds2,
						expr: &litMatcher{
							pos:        position{line: 671, col: 5, offset: 17251},
							val:        "second",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 672, col: 5, offset: 17296},
						run: (*parser).callonseconds4,
						expr: &seqExpr{
							pos: position{line: 672, col: 5, offset: 17296},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 672, col: 5, offset: 17296},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 672, col: 9, offset: 17300},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 672, col: 16, offset: 17307},
									expr: &ruleRefExpr{
										pos:  position{line: 672, col: 16, offset: 17307},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 672, col: 19, offset: 17310},
									name: "sec_abbrev",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "minutes",
			pos:  position{line: 674, col: 1, offset: 17356},
			expr: &choiceExpr{
				pos: position{line: 675, col: 5, offset: 17368},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 675, col: 5, offset: 17368},
						run: (*parser).callonminutes2,
						expr: &litMatcher{
							pos:        position{line: 675, col: 5, offset: 17368},
							val:        "minute",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 676, col: 5, offset: 17414},
						run: (*parser).callonminutes4,
						expr: &seqExpr{
							pos: position{line: 676, col: 5, offset: 17414},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 676, col: 5, offset: 17414},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 676, col: 9, offset: 17418},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 676, col: 16, offset: 17425},
									expr: &ruleRefExpr{
										pos:  position{line: 676, col: 16, offset: 17425},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 676, col: 19, offset: 17428},
									name: "min_abbrev",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "hours",
			pos:  position{line: 678, col: 1, offset: 17483},
			expr: &choiceExpr{
				pos: position{line: 679, col: 5, offset: 17493},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 679, col: 5, offset: 17493},
						run: (*parser).callonhours2,
						expr: &litMatcher{
							pos:        position{line: 679, col: 5, offset: 17493},
							val:        "hour",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 680, col: 5, offset: 17539},
						run: (*parser).callonhours4,
						expr: &seqExpr{
							pos: position{line: 680, col: 5, offset: 17539},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 680, col: 5, offset: 17539},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 680, col: 9, offset: 17543},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 680, col: 16, offset: 17550},
									expr: &ruleRefExpr{
										pos:  position{line: 680, col: 16, offset: 17550},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 680, col: 19, offset: 17553},
									name: "hour_abbrev",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "days",
			pos:  position{line: 682, col: 1, offset: 17611},
			expr: &choiceExpr{
				pos: position{line: 683, col: 5, offset: 17620},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 683, col: 5, offset: 17620},
						run: (*parser).callondays2,
						expr: &litMatcher{
							pos:        position{line: 683, col: 5, offset: 17620},
							val:        "day",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 684, col: 5, offset: 17668},
						run: (*parser).callondays4,
						expr: &seqExpr{
							pos: position{line: 684, col: 5, offset: 17668},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 684, col: 5, offset: 17668},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 684, col: 9, offset: 17672},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 684, col: 16, offset: 17679},
									expr: &ruleRefExpr{
										pos:  position{line: 684, col: 16, offset: 17679},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 684, col: 19, offset: 17682},
									name: "day_abbrev",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "weeks",
			pos:  position{line: 686, col: 1, offset: 17742},
			expr: &actionExpr{
				pos: position{line: 687, col: 5, offset: 17752},
				run: (*parser).callonweeks1,
				expr: &seqExpr{
					pos: position{line: 687, col: 5, offset: 17752},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 687, col: 5, offset: 17752},
							label: "num",
							expr: &ruleRefExpr{
								pos:  position{line: 687, col: 9, offset: 17756},
								name: "number",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 687, col: 16, offset: 17763},
							expr: &ruleRefExpr{
								pos:  position{line: 687, col: 16, offset: 17763},
								name: "_",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 687, col: 19, offset: 17766},
							name: "week_abbrev",
						},
					},
				},
			},
		},
		{
			name: "number",
			pos:  position{line: 689, col: 1, offset: 17829},
			expr: &ruleRefExpr{
				pos:  position{line: 689, col: 10, offset: 17838},
				name: "integer",
			},
		},
		{
			name: "addr",
			pos:  position{line: 693, col: 1, offset: 17876},
			expr: &actionExpr{
				pos: position{line: 694, col: 5, offset: 17885},
				run: (*parser).callonaddr1,
				expr: &labeledExpr{
					pos:   position{line: 694, col: 5, offset: 17885},
					label: "a",
					expr: &seqExpr{
						pos: position{line: 694, col: 8, offset: 17888},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 694, col: 8, offset: 17888},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 694, col: 16, offset: 17896},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 694, col: 20, offset: 17900},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 694, col: 28, offset: 17908},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 694, col: 32, offset: 17912},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 694, col: 40, offset: 17920},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 694, col: 44, offset: 17924},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "port",
			pos:  position{line: 696, col: 1, offset: 17965},
			expr: &actionExpr{
				pos: position{line: 697, col: 5, offset: 17974},
				run: (*parser).callonport1,
				expr: &seqExpr{
					pos: position{line: 697, col: 5, offset: 17974},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 697, col: 5, offset: 17974},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 697, col: 9, offset: 17978},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 697, col: 11, offset: 17980},
								name: "sinteger",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6addr",
			pos:  position{line: 701, col: 1, offset: 18139},
			expr: &choiceExpr{
				pos: position{line: 702, col: 5, offset: 18151},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 702, col: 5, offset: 18151},
						run: (*parser).callonip6addr2,
						expr: &seqExpr{
							pos: position{line: 702, col: 5, offset: 18151},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 702, col: 5, offset: 18151},
									label: "a",
									expr: &oneOrMoreExpr{
										pos: position{line: 702, col: 7, offset: 18153},
										expr: &ruleRefExpr{
											pos:  position{line: 702, col: 8, offset: 18154},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 702, col: 20, offset: 18166},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 702, col: 22, offset: 18168},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 705, col: 5, offset: 18232},
						run: (*parser).callonip6addr9,
						expr: &seqExpr{
							pos: position{line: 705, col: 5, offset: 18232},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 705, col: 5, offset: 18232},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 705, col: 7, offset: 18234},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 705, col: 11, offset: 18238},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 705, col: 13, offset: 18240},
										expr: &ruleRefExpr{
											pos:  position{line: 705, col: 14, offset: 18241},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 705, col: 25, offset: 18252},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 705, col: 30, offset: 18257},
									label: "d",
									expr: &zeroOrMoreExpr{
										pos: position{line: 705, col: 32, offset: 18259},
										expr: &ruleRefExpr{
											pos:  position{line: 705, col: 33, offset: 18260},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 705, col: 45, offset: 18272},
									label: "e",
									expr: &ruleRefExpr{
										pos:  position{line: 705, col: 47, offset: 18274},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 708, col: 5, offset: 18373},
						run: (*parser).callonip6addr22,
						expr: &seqExpr{
							pos: position{line: 708, col: 5, offset: 18373},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 708, col: 5, offset: 18373},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 708, col: 10, offset: 18378},
									label: "a",
									expr: &zeroOrMoreExpr{
										pos: position{line: 708, col: 12, offset: 18380},
										expr: &ruleRefExpr{
											pos:  position{line: 708, col: 13, offset: 18381},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 708, col: 25, offset: 18393},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 708, col: 27, offset: 18395},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 711, col: 5, offset: 18466},
						run: (*parser).callonip6addr30,
						expr: &seqExpr{
							pos: position{line: 711, col: 5, offset: 18466},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 711, col: 5, offset: 18466},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 711, col: 7, offset: 18468},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 711, col: 11, offset: 18472},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 711, col: 13, offset: 18474},
										expr: &ruleRefExpr{
											pos:  position{line: 711, col: 14, offset: 18475},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 711, col: 25, offset: 18486},
									val:        "::",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 714, col: 5, offset: 18554},
						run: (*parser).callonip6addr38,
						expr: &litMatcher{
							pos:        position{line: 714, col: 5, offset: 18554},
							val:        "::",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "ip6tail",
			pos:  position{line: 718, col: 1, offset: 18591},
			expr: &choiceExpr{
				pos: position{line: 719, col: 5, offset: 18603},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 719, col: 5, offset: 18603},
						name: "addr",
					},
					&ruleRefExpr{
						pos:  position{line: 720, col: 5, offset: 18612},
						name: "h16",
					},
				},
			},
		},
		{
			name: "h_append",
			pos:  position{line: 722, col: 1, offset: 18617},
			expr: &actionExpr{
				pos: position{line: 722, col: 12, offset: 18628},
				run: (*parser).callonh_append1,
				expr: &seqExpr{
					pos: position{line: 722, col: 12, offset: 18628},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 722, col: 12, offset: 18628},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 722, col: 16, offset: 18632},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 722, col: 18, offset: 18634},
								name: "h16",
							},
						},
					},
				},
			},
		},
		{
			name: "h_prepend",
			pos:  position{line: 723, col: 1, offset: 18671},
			expr: &actionExpr{
				pos: position{line: 723, col: 13, offset: 18683},
				run: (*parser).callonh_prepend1,
				expr: &seqExpr{
					pos: position{line: 723, col: 13, offset: 18683},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 723, col: 13, offset: 18683},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 723, col: 15, offset: 18685},
								name: "h16",
							},
						},
						&litMatcher{
							pos:        position{line: 723, col: 19, offset: 18689},
							val:        ":",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "sub_addr",
			pos:  position{line: 725, col: 1, offset: 18727},
			expr: &choiceExpr{
				pos: position{line: 726, col: 5, offset: 18740},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 726, col: 5, offset: 18740},
						name: "addr",
					},
					&actionExpr{
						pos: position{line: 727, col: 5, offset: 18749},
						run: (*parser).callonsub_addr3,
						expr: &labeledExpr{
							pos:   position{line: 727, col: 5, offset: 18749},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 727, col: 8, offset: 18752},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 727, col: 8, offset: 18752},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 727, col: 16, offset: 18760},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 727, col: 20, offset: 18764},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 727, col: 28, offset: 18772},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 727, col: 32, offset: 18776},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 728, col: 5, offset: 18828},
						run: (*parser).callonsub_addr11,
						expr: &labeledExpr{
							pos:   position{line: 728, col: 5, offset: 18828},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 728, col: 8, offset: 18831},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 728, col: 8, offset: 18831},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 728, col: 16, offset: 18839},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 728, col: 20, offset: 18843},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 729, col: 5, offset: 18897},
						run: (*parser).callonsub_addr17,
						expr: &labeledExpr{
							pos:   position{line: 729, col: 5, offset: 18897},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 729, col: 7, offset: 18899},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "subnet",
			pos:  position{line: 731, col: 1, offset: 18950},
			expr: &actionExpr{
				pos: position{line: 732, col: 5, offset: 18961},
				run: (*parser).callonsubnet1,
				expr: &seqExpr{
					pos: position{line: 732, col: 5, offset: 18961},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 732, col: 5, offset: 18961},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 732, col: 7, offset: 18963},
								name: "sub_addr",
							},
						},
						&litMatcher{
							pos:        position{line: 732, col: 16, offset: 18972},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 732, col: 20, offset: 18976},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 732, col: 22, offset: 18978},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6subnet",
			pos:  position{line: 736, col: 1, offset: 19054},
			expr: &actionExpr{
				pos: position{line: 737, col: 5, offset: 19068},
				run: (*parser).callonip6subnet1,
				expr: &seqExpr{
					pos: position{line: 737, col: 5, offset: 19068},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 737, col: 5, offset: 19068},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 737, col: 7, offset: 19070},
								name: "ip6addr",
							},
						},
						&litMatcher{
							pos:        position{line: 737, col: 15, offset: 19078},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 737, col: 19, offset: 19082},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 737, col: 21, offset: 19084},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "integer",
			pos:  position{line: 741, col: 1, offset: 19150},
			expr: &actionExpr{
				pos: position{line: 742, col: 5, offset: 19162},
				run: (*parser).calloninteger1,
				expr: &labeledExpr{
					pos:   position{line: 742, col: 5, offset: 19162},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 742, col: 7, offset: 19164},
						name: "sinteger",
					},
				},
			},
		},
		{
			name: "sinteger",
			pos:  position{line: 746, col: 1, offset: 19208},
			expr: &actionExpr{
				pos: position{line: 747, col: 5, offset: 19221},
				run: (*parser).callonsinteger1,
				expr: &labeledExpr{
					pos:   position{line: 747, col: 5, offset: 19221},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 747, col: 11, offset: 19227},
						expr: &charClassMatcher{
							pos:        position{line: 747, col: 11, offset: 19227},
							val:        "[0-9]",
							ranges:     []rune{'0', '9'},
							ignoreCase: false,
							inverted:   false,
						},
					},
				},
			},
		},
		{
			name: "double",
			pos:  position{line: 751, col: 1, offset: 19272},
			expr: &actionExpr{
				pos: position{line: 752, col: 5, offset: 19283},
				run: (*parser).callondouble1,
				expr: &labeledExpr{
					pos:   position{line: 752, col: 5, offset: 19283},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 752, col: 7, offset: 19285},
						name: "sdouble",
					},
				},
			},
		},
		{
			name: "sdouble",
			pos:  position{line: 756, col: 1, offset: 19332},
			expr: &choiceExpr{
				pos: position{line: 757, col: 5, offset: 19344},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 757, col: 5, offset: 19344},
						run: (*parser).callonsdouble2,
						expr: &seqExpr{
							pos: position{line: 757, col: 5, offset: 19344},
							exprs: []interface{}{
								&oneOrMoreExpr{
									pos: position{line: 757, col: 5, offset: 19344},
									expr: &ruleRefExpr{
										pos:  position{line: 757, col: 5, offset: 19344},
										name: "doubleInteger",
									},
								},
								&litMatcher{
									pos:        position{line: 757, col: 20, offset: 19359},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 757, col: 24, offset: 19363},
									expr: &ruleRefExpr{
										pos:  position{line: 757, col: 24, offset: 19363},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 757, col: 37, offset: 19376},
									expr: &ruleRefExpr{
										pos:  position{line: 757, col: 37, offset: 19376},
										name: "exponentPart",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 760, col: 5, offset: 19435},
						run: (*parser).callonsdouble11,
						expr: &seqExpr{
							pos: position{line: 760, col: 5, offset: 19435},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 760, col: 5, offset: 19435},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 760, col: 9, offset: 19439},
									expr: &ruleRefExpr{
										pos:  position{line: 760, col: 9, offset: 19439},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 760, col: 22, offset: 19452},
									expr: &ruleRefExpr{
										pos:  position{line: 760, col: 22, offset: 19452},
										name: "exponentPart",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "doubleInteger",
			pos:  position{line: 764, col: 1, offset: 19508},
			expr: &choiceExpr{
				pos: position{line: 765, col: 5, offset: 19526},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 765, col: 5, offset: 19526},
						val:        "0",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 766, col: 5, offset: 19534},
						exprs: []interface{}{
							&charClassMatcher{
								pos:        position{line: 766, col: 5, offset: 19534},
								val:        "[1-9]",
								ranges:     []rune{'1', '9'},
								ignoreCase: false,
								inverted:   false,
							},
							&zeroOrMoreExpr{
								pos: position{line: 766, col: 11, offset: 19540},
								expr: &charClassMatcher{
									pos:        position{line: 766, col: 11, offset: 19540},
									val:        "[0-9]",
									ranges:     []rune{'0', '9'},
									ignoreCase: false,
									inverted:   false,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "doubleDigit",
			pos:  position{line: 768, col: 1, offset: 19548},
			expr: &charClassMatcher{
				pos:        position{line: 768, col: 15, offset: 19562},
				val:        "[0-9]",
				ranges:     []rune{'0', '9'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "signedInteger",
			pos:  position{line: 770, col: 1, offset: 19569},
			expr: &seqExpr{
				pos: position{line: 770, col: 17, offset: 19585},
				exprs: []interface{}{
					&zeroOrOneExpr{
						pos: position{line: 770, col: 17, offset: 19585},
						expr: &charClassMatcher{
							pos:        position{line: 770, col: 17, offset: 19585},
							val:        "[+-]",
							chars:      []rune{'+', '-'},
							ignoreCase: false,
							inverted:   false,
						},
					},
					&oneOrMoreExpr{
						pos: position{line: 770, col: 23, offset: 19591},
						expr: &ruleRefExpr{
							pos:  position{line: 770, col: 23, offset: 19591},
							name: "doubleDigit",
						},
					},
				},
			},
		},
		{
			name: "exponentPart",
			pos:  position{line: 772, col: 1, offset: 19605},
			expr: &seqExpr{
				pos: position{line: 772, col: 16, offset: 19620},
				exprs: []interface{}{
					&litMatcher{
						pos:        position{line: 772, col: 16, offset: 19620},
						val:        "e",
						ignoreCase: true,
					},
					&ruleRefExpr{
						pos:  position{line: 772, col: 21, offset: 19625},
						name: "signedInteger",
					},
				},
			},
		},
		{
			name: "h16",
			pos:  position{line: 774, col: 1, offset: 19640},
			expr: &actionExpr{
				pos: position{line: 774, col: 7, offset: 19646},
				run: (*parser).callonh161,
				expr: &labeledExpr{
					pos:   position{line: 774, col: 7, offset: 19646},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 774, col: 13, offset: 19652},
						expr: &ruleRefExpr{
							pos:  position{line: 774, col: 13, offset: 19652},
							name: "hexdigit",
						},
					},
				},
			},
		},
		{
			name: "hexdigit",
			pos:  position{line: 776, col: 1, offset: 19694},
			expr: &charClassMatcher{
				pos:        position{line: 776, col: 12, offset: 19705},
				val:        "[0-9a-fA-F]",
				ranges:     []rune{'0', '9', 'a', 'f', 'A', 'F'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name:        "boomWord",
			displayName: "\"boomWord\"",
			pos:         position{line: 778, col: 1, offset: 19718},
			expr: &actionExpr{
				pos: position{line: 778, col: 23, offset: 19740},
				run: (*parser).callonboomWord1,
				expr: &labeledExpr{
					pos:   position{line: 778, col: 23, offset: 19740},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 778, col: 29, offset: 19746},
						expr: &ruleRefExpr{
							pos:  position{line: 778, col: 29, offset: 19746},
							name: "boomWordPart",
						},
					},
				},
			},
		},
		{
			name: "boomWordPart",
			pos:  position{line: 780, col: 1, offset: 19792},
			expr: &seqExpr{
				pos: position{line: 781, col: 5, offset: 19809},
				exprs: []interface{}{
					&notExpr{
						pos: position{line: 781, col: 5, offset: 19809},
						expr: &choiceExpr{
							pos: position{line: 781, col: 7, offset: 19811},
							alternatives: []interface{}{
								&charClassMatcher{
									pos:        position{line: 781, col: 7, offset: 19811},
									val:        "[\\x00-\\x1F\\x5C(),!><=\\x22|\\x27;]",
									chars:      []rune{'\\', '(', ')', ',', '!', '>', '<', '=', '"', '|', '\'', ';'},
									ranges:     []rune{'\x00', '\x1f'},
									ignoreCase: false,
									inverted:   false,
								},
								&ruleRefExpr{
									pos:  position{line: 781, col: 42, offset: 19846},
									name: "ws",
								},
							},
						},
					},
					&anyMatcher{
						line: 781, col: 46, offset: 19850,
					},
				},
			},
		},
		{
			name: "quotedString",
			pos:  position{line: 783, col: 1, offset: 19853},
			expr: &choiceExpr{
				pos: position{line: 784, col: 5, offset: 19870},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 784, col: 5, offset: 19870},
						run: (*parser).callonquotedString2,
						expr: &seqExpr{
							pos: position{line: 784, col: 5, offset: 19870},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 784, col: 5, offset: 19870},
									val:        "\"",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 784, col: 9, offset: 19874},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 784, col: 11, offset: 19876},
										expr: &ruleRefExpr{
											pos:  position{line: 784, col: 11, offset: 19876},
											name: "doubleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 784, col: 29, offset: 19894},
									val:        "\"",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 785, col: 5, offset: 19931},
						run: (*parser).callonquotedString9,
						expr: &seqExpr{
							pos: position{line: 785, col: 5, offset: 19931},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 785, col: 5, offset: 19931},
									val:        "'",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 785, col: 9, offset: 19935},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 785, col: 11, offset: 19937},
										expr: &ruleRefExpr{
											pos:  position{line: 785, col: 11, offset: 19937},
											name: "singleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 785, col: 29, offset: 19955},
									val:        "'",
									ignoreCase: false,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "doubleQuotedChar",
			pos:  position{line: 787, col: 1, offset: 19989},
			expr: &choiceExpr{
				pos: position{line: 788, col: 5, offset: 20010},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 788, col: 5, offset: 20010},
						run: (*parser).callondoubleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 788, col: 5, offset: 20010},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 788, col: 5, offset: 20010},
									expr: &choiceExpr{
										pos: position{line: 788, col: 7, offset: 20012},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 788, col: 7, offset: 20012},
												val:        "\"",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 788, col: 13, offset: 20018},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 788, col: 26, offset: 20031,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 789, col: 5, offset: 20068},
						run: (*parser).callondoubleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 789, col: 5, offset: 20068},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 789, col: 5, offset: 20068},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 789, col: 10, offset: 20073},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 789, col: 12, offset: 20075},
										name: "escapeSequence",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "singleQuotedChar",
			pos:  position{line: 791, col: 1, offset: 20109},
			expr: &choiceExpr{
				pos: position{line: 792, col: 5, offset: 20130},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 792, col: 5, offset: 20130},
						run: (*parser).callonsingleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 792, col: 5, offset: 20130},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 792, col: 5, offset: 20130},
									expr: &choiceExpr{
										pos: position{line: 792, col: 7, offset: 20132},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 792, col: 7, offset: 20132},
												val:        "'",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 792, col: 13, offset: 20138},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 792, col: 26, offset: 20151,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 793, col: 5, offset: 20188},
						run: (*parser).callonsingleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 793, col: 5, offset: 20188},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 793, col: 5, offset: 20188},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 793, col: 10, offset: 20193},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 793, col: 12, offset: 20195},
										name: "escapeSequence",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "escapeSequence",
			pos:  position{line: 795, col: 1, offset: 20229},
			expr: &choiceExpr{
				pos: position{line: 795, col: 18, offset: 20246},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 795, col: 18, offset: 20246},
						name: "singleCharEscape",
					},
					&ruleRefExpr{
						pos:  position{line: 795, col: 37, offset: 20265},
						name: "unicodeEscape",
					},
				},
			},
		},
		{
			name: "singleCharEscape",
			pos:  position{line: 797, col: 1, offset: 20280},
			expr: &choiceExpr{
				pos: position{line: 798, col: 5, offset: 20301},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 798, col: 5, offset: 20301},
						val:        "'",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 799, col: 5, offset: 20309},
						val:        "\"",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 800, col: 5, offset: 20317},
						val:        "\\",
						ignoreCase: false,
					},
					&actionExpr{
						pos: position{line: 801, col: 5, offset: 20326},
						run: (*parser).callonsingleCharEscape5,
						expr: &litMatcher{
							pos:        position{line: 801, col: 5, offset: 20326},
							val:        "b",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 802, col: 5, offset: 20355},
						run: (*parser).callonsingleCharEscape7,
						expr: &litMatcher{
							pos:        position{line: 802, col: 5, offset: 20355},
							val:        "f",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 803, col: 5, offset: 20384},
						run: (*parser).callonsingleCharEscape9,
						expr: &litMatcher{
							pos:        position{line: 803, col: 5, offset: 20384},
							val:        "n",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 804, col: 5, offset: 20413},
						run: (*parser).callonsingleCharEscape11,
						expr: &litMatcher{
							pos:        position{line: 804, col: 5, offset: 20413},
							val:        "r",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 805, col: 5, offset: 20442},
						run: (*parser).callonsingleCharEscape13,
						expr: &litMatcher{
							pos:        position{line: 805, col: 5, offset: 20442},
							val:        "t",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 806, col: 5, offset: 20471},
						run: (*parser).callonsingleCharEscape15,
						expr: &litMatcher{
							pos:        position{line: 806, col: 5, offset: 20471},
							val:        "v",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "unicodeEscape",
			pos:  position{line: 808, col: 1, offset: 20497},
			expr: &seqExpr{
				pos: position{line: 809, col: 5, offset: 20515},
				exprs: []interface{}{
					&litMatcher{
						pos:        position{line: 809, col: 5, offset: 20515},
						val:        "u",
						ignoreCase: false,
					},
					&ruleRefExpr{
						pos:  position{line: 809, col: 9, offset: 20519},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 809, col: 18, offset: 20528},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 809, col: 27, offset: 20537},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 809, col: 36, offset: 20546},
						name: "hexdigit",
					},
				},
			},
		},
		{
			name: "reString",
			pos:  position{line: 811, col: 1, offset: 20556},
			expr: &actionExpr{
				pos: position{line: 812, col: 5, offset: 20569},
				run: (*parser).callonreString1,
				expr: &seqExpr{
					pos: position{line: 812, col: 5, offset: 20569},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 812, col: 5, offset: 20569},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 812, col: 9, offset: 20573},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 812, col: 11, offset: 20575},
								name: "reBody",
							},
						},
						&litMatcher{
							pos:        position{line: 812, col: 18, offset: 20582},
							val:        "/",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reBody",
			pos:  position{line: 814, col: 1, offset: 20605},
			expr: &actionExpr{
				pos: position{line: 815, col: 5, offset: 20616},
				run: (*parser).callonreBody1,
				expr: &oneOrMoreExpr{
					pos: position{line: 815, col: 5, offset: 20616},
					expr: &choiceExpr{
						pos: position{line: 815, col: 6, offset: 20617},
						alternatives: []interface{}{
							&charClassMatcher{
								pos:        position{line: 815, col: 6, offset: 20617},
								val:        "[^/\\\\]",
								chars:      []rune{'/', '\\'},
								ignoreCase: false,
								inverted:   true,
							},
							&litMatcher{
								pos:        position{line: 815, col: 13, offset: 20624},
								val:        "\\/",
								ignoreCase: false,
							},
						},
					},
				},
			},
		},
		{
			name: "escapedChar",
			pos:  position{line: 817, col: 1, offset: 20664},
			expr: &charClassMatcher{
				pos:        position{line: 818, col: 5, offset: 20680},
				val:        "[\\x00-\\x1f\\\\]",
				chars:      []rune{'\\'},
				ranges:     []rune{'\x00', '\x1f'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "ws",
			pos:  position{line: 820, col: 1, offset: 20695},
			expr: &choiceExpr{
				pos: position{line: 821, col: 5, offset: 20702},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 821, col: 5, offset: 20702},
						val:        "\t",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 822, col: 5, offset: 20711},
						val:        "\v",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 823, col: 5, offset: 20720},
						val:        "\f",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 824, col: 5, offset: 20729},
						val:        " ",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 825, col: 5, offset: 20737},
						val:        "\u00a0",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 826, col: 5, offset: 20750},
						val:        "\ufeff",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name:        "_",
			displayName: "\"whitespace\"",
			pos:         position{line: 828, col: 1, offset: 20760},
			expr: &oneOrMoreExpr{
				pos: position{line: 828, col: 18, offset: 20777},
				expr: &ruleRefExpr{
					pos:  position{line: 828, col: 18, offset: 20777},
					name: "ws",
				},
			},
		},
		{
			name: "EOF",
			pos:  position{line: 830, col: 1, offset: 20782},
			expr: &notExpr{
				pos: position{line: 830, col: 7, offset: 20788},
				expr: &anyMatcher{
					line: 830, col: 8, offset: 20789,
				},
			},
		},
	},
}

func (c *current) onstart1(ast interface{}) (interface{}, error) {
	return ast, nil
}

func (p *parser) callonstart1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onstart1(stack["ast"])
}

func (c *current) onboomCommand2(procs interface{}) (interface{}, error) {
	filt := makeFilterProc(makeBooleanLiteral(true))
	return makeSequentialProc(append([]interface{}{filt}, (procs.([]interface{}))...)), nil

}

func (p *parser) callonboomCommand2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onboomCommand2(stack["procs"])
}

func (c *current) onboomCommand5(s, rest interface{}) (interface{}, error) {
	if len(rest.([]interface{})) == 0 {
		return s, nil
	} else {
		return makeSequentialProc(append([]interface{}{s}, (rest.([]interface{}))...)), nil
	}

}

func (p *parser) callonboomCommand5() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onboomCommand5(stack["s"], stack["rest"])
}

func (c *current) onboomCommand14(s interface{}) (interface{}, error) {
	return makeSequentialProc([]interface{}{s}), nil

}

func (p *parser) callonboomCommand14() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onboomCommand14(stack["s"])
}

func (c *current) onprocChain1(first, rest interface{}) (interface{}, error) {
	if rest != nil {
		return append([]interface{}{first}, (rest.([]interface{}))...), nil
	} else {
		return []interface{}{first}, nil
	}

}

func (p *parser) callonprocChain1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onprocChain1(stack["first"], stack["rest"])
}

func (c *current) onchainedProc1(p interface{}) (interface{}, error) {
	return p, nil
}

func (p *parser) callonchainedProc1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onchainedProc1(stack["p"])
}

func (c *current) onsearch1(expr interface{}) (interface{}, error) {
	return makeFilterProc(expr), nil

}

func (p *parser) callonsearch1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearch1(stack["expr"])
}

func (c *current) onsearchExpr1(first, rest interface{}) (interface{}, error) {
	return makeOrChain(first, rest), nil

}

func (p *parser) callonsearchExpr1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchExpr1(stack["first"], stack["rest"])
}

func (c *current) onoredSearchTerm1(t interface{}) (interface{}, error) {
	return t, nil
}

func (p *parser) callonoredSearchTerm1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onoredSearchTerm1(stack["t"])
}

func (c *current) onsearchTerm1(first, rest interface{}) (interface{}, error) {
	return makeAndChain(first, rest), nil

}

func (p *parser) callonsearchTerm1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchTerm1(stack["first"], stack["rest"])
}

func (c *current) onandedSearchTerm1(f interface{}) (interface{}, error) {
	return f, nil
}

func (p *parser) callonandedSearchTerm1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onandedSearchTerm1(stack["f"])
}

func (c *current) onsearchFactor2(e interface{}) (interface{}, error) {
	return makeLogicalNot(e), nil

}

func (p *parser) callonsearchFactor2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchFactor2(stack["e"])
}

func (c *current) onsearchFactor14(s interface{}) (interface{}, error) {
	return s, nil
}

func (p *parser) callonsearchFactor14() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchFactor14(stack["s"])
}

func (c *current) onsearchFactor20(expr interface{}) (interface{}, error) {
	return expr, nil
}

func (p *parser) callonsearchFactor20() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchFactor20(stack["expr"])
}

func (c *current) onsearchPred2(fieldComparator, v interface{}) (interface{}, error) {
	return makeCompareAny(fieldComparator, v), nil

}

func (p *parser) callonsearchPred2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchPred2(stack["fieldComparator"], stack["v"])
}

func (c *current) onsearchPred13() (interface{}, error) {
	return makeBooleanLiteral(true), nil

}

func (p *parser) callonsearchPred13() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchPred13()
}

func (c *current) onsearchPred15(f, fieldComparator, v interface{}) (interface{}, error) {
	return makeCompareField(fieldComparator, f, v), nil

}

func (p *parser) callonsearchPred15() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchPred15(stack["f"], stack["fieldComparator"], stack["v"])
}

func (c *current) onsearchPred27(v interface{}) (interface{}, error) {
	return makeCompareAny("in", v), nil

}

func (p *parser) callonsearchPred27() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchPred27(stack["v"])
}

func (c *current) onsearchPred37(v, f interface{}) (interface{}, error) {
	return makeCompareField("in", f, v), nil

}

func (p *parser) callonsearchPred37() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchPred37(stack["v"], stack["f"])
}

func (c *current) onsearchPred48(v interface{}) (interface{}, error) {
	ss := makeSearchString(v)
	if getValueType(v) == "string" {
		return ss, nil
	}
	ss = makeSearchString(makeTypedValue("string", string(c.text)))
	return makeOrChain(ss, []interface{}{makeCompareAny("eql", v), makeCompareAny("in", v)}), nil

}

func (p *parser) callonsearchPred48() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchPred48(stack["v"])
}

func (c *current) onsearchValue2(v interface{}) (interface{}, error) {
	return makeTypedValue("string", v), nil

}

func (p *parser) callonsearchValue2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchValue2(stack["v"])
}

func (c *current) onsearchValue5(v interface{}) (interface{}, error) {
	return makeTypedValue("regexp", v), nil

}

func (p *parser) callonsearchValue5() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchValue5(stack["v"])
}

func (c *current) onsearchValue8(v interface{}) (interface{}, error) {
	return makeTypedValue("port", v), nil

}

func (p *parser) callonsearchValue8() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchValue8(stack["v"])
}

func (c *current) onsearchValue11(v interface{}) (interface{}, error) {
	return makeTypedValue("subnet", v), nil

}

func (p *parser) callonsearchValue11() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchValue11(stack["v"])
}

func (c *current) onsearchValue14(v interface{}) (interface{}, error) {
	return makeTypedValue("addr", v), nil

}

func (p *parser) callonsearchValue14() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchValue14(stack["v"])
}

func (c *current) onsearchValue17(v interface{}) (interface{}, error) {
	return makeTypedValue("subnet", v), nil

}

func (p *parser) callonsearchValue17() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchValue17(stack["v"])
}

func (c *current) onsearchValue20(v interface{}) (interface{}, error) {
	return makeTypedValue("addr", v), nil

}

func (p *parser) callonsearchValue20() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchValue20(stack["v"])
}

func (c *current) onsearchValue23(v interface{}) (interface{}, error) {
	return makeTypedValue("double", v), nil

}

func (p *parser) callonsearchValue23() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchValue23(stack["v"])
}

func (c *current) onsearchValue26(v interface{}) (interface{}, error) {
	return makeTypedValue("int", v), nil

}

func (p *parser) callonsearchValue26() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchValue26(stack["v"])
}

func (c *current) onsearchValue32(v interface{}) (interface{}, error) {
	return v, nil
}

func (p *parser) callonsearchValue32() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchValue32(stack["v"])
}

func (c *current) onsearchValue40(v interface{}) (interface{}, error) {
	return v, nil
}

func (p *parser) callonsearchValue40() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchValue40(stack["v"])
}

func (c *current) onsearchValue48(v interface{}) (interface{}, error) {
	if reglob.IsGlobby(v.(string)) || v.(string) == "*" {
		re := reglob.Reglob(v.(string))
		return makeTypedValue("regexp", re), nil
	}
	return makeTypedValue("string", v), nil

}

func (p *parser) callonsearchValue48() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchValue48(stack["v"])
}

func (c *current) onbooleanLiteral2() (interface{}, error) {
	return makeTypedValue("bool", "true"), nil
}

func (p *parser) callonbooleanLiteral2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onbooleanLiteral2()
}

func (c *current) onbooleanLiteral4() (interface{}, error) {
	return makeTypedValue("bool", "false"), nil
}

func (p *parser) callonbooleanLiteral4() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onbooleanLiteral4()
}

func (c *current) onunsetLiteral1() (interface{}, error) {
	return makeTypedValue("unset", ""), nil
}

func (p *parser) callonunsetLiteral1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onunsetLiteral1()
}

func (c *current) onprocList1(first, rest interface{}) (interface{}, error) {
	fp := makeSequentialProc(first)
	if rest != nil {
		return makeParallelProc(append([]interface{}{fp}, (rest.([]interface{}))...)), nil
	} else {
		return fp, nil
	}

}

func (p *parser) callonprocList1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onprocList1(stack["first"], stack["rest"])
}

func (c *current) onparallelChain1(ch interface{}) (interface{}, error) {
	return makeSequentialProc(ch), nil
}

func (p *parser) callonparallelChain1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onparallelChain1(stack["ch"])
}

func (c *current) onproc4(proc interface{}) (interface{}, error) {
	return proc, nil

}

func (p *parser) callonproc4() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onproc4(stack["proc"])
}

func (c *current) ongroupBy1(list interface{}) (interface{}, error) {
	return list, nil
}

func (p *parser) callongroupBy1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.ongroupBy1(stack["list"])
}

func (c *current) oneveryDur1(dur interface{}) (interface{}, error) {
	return dur, nil
}

func (p *parser) calloneveryDur1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.oneveryDur1(stack["dur"])
}

func (c *current) onequalityToken2() (interface{}, error) {
	return "eql", nil
}

func (p *parser) callonequalityToken2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onequalityToken2()
}

func (c *current) onequalityToken4() (interface{}, error) {
	return "neql", nil
}

func (p *parser) callonequalityToken4() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onequalityToken4()
}

func (c *current) onequalityToken6() (interface{}, error) {
	return "lte", nil
}

func (p *parser) callonequalityToken6() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onequalityToken6()
}

func (c *current) onequalityToken8() (interface{}, error) {
	return "gte", nil
}

func (p *parser) callonequalityToken8() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onequalityToken8()
}

func (c *current) onequalityToken10() (interface{}, error) {
	return "lt", nil
}

func (p *parser) callonequalityToken10() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onequalityToken10()
}

func (c *current) onequalityToken12() (interface{}, error) {
	return "gt", nil
}

func (p *parser) callonequalityToken12() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onequalityToken12()
}

func (c *current) onfieldName1() (interface{}, error) {
	return string(c.text), nil
}

func (p *parser) callonfieldName1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldName1()
}

func (c *current) onfieldReference8(field interface{}) (interface{}, error) {
	return makeFieldCall("RecordFieldRead", nil, field), nil
}

func (p *parser) callonfieldReference8() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldReference8(stack["field"])
}

func (c *current) onfieldReference13(index interface{}) (interface{}, error) {
	return makeFieldCall("Index", nil, index), nil
}

func (p *parser) callonfieldReference13() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldReference13(stack["index"])
}

func (c *current) onfieldReference1(base, derefs interface{}) (interface{}, error) {
	return chainFieldCalls(base, derefs), nil

}

func (p *parser) callonfieldReference1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldReference1(stack["base"], stack["derefs"])
}

func (c *current) onfieldExpr2(op, field interface{}) (interface{}, error) {
	return makeFieldCall(op, field, nil), nil

}

func (p *parser) callonfieldExpr2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldExpr2(stack["op"], stack["field"])
}

func (c *current) onfieldOp1() (interface{}, error) {
	return "Len", nil
}

func (p *parser) callonfieldOp1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldOp1()
}

func (c *current) onfieldList1(first, rest interface{}) (interface{}, error) {
	result := []interface{}{first}

	for _, r := range rest.([]interface{}) {
		result = append(result, r.([]interface{})[3])
	}

	return result, nil

}

func (p *parser) callonfieldList1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldList1(stack["first"], stack["rest"])
}

func (c *current) oncountOp1() (interface{}, error) {
	return "Count", nil
}

func (p *parser) calloncountOp1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.oncountOp1()
}

func (c *current) onfieldReducerOp2() (interface{}, error) {
	return "Sum", nil
}

func (p *parser) callonfieldReducerOp2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldReducerOp2()
}

func (c *current) onfieldReducerOp4() (interface{}, error) {
	return "Avg", nil
}

func (p *parser) callonfieldReducerOp4() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldReducerOp4()
}

func (c *current) onfieldReducerOp6() (interface{}, error) {
	return "Stdev", nil
}

func (p *parser) callonfieldReducerOp6() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldReducerOp6()
}

func (c *current) onfieldReducerOp8() (interface{}, error) {
	return "Stdev", nil
}

func (p *parser) callonfieldReducerOp8() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldReducerOp8()
}

func (c *current) onfieldReducerOp10() (interface{}, error) {
	return "Var", nil
}

func (p *parser) callonfieldReducerOp10() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldReducerOp10()
}

func (c *current) onfieldReducerOp12() (interface{}, error) {
	return "Entropy", nil
}

func (p *parser) callonfieldReducerOp12() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldReducerOp12()
}

func (c *current) onfieldReducerOp14() (interface{}, error) {
	return "Min", nil
}

func (p *parser) callonfieldReducerOp14() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldReducerOp14()
}

func (c *current) onfieldReducerOp16() (interface{}, error) {
	return "Max", nil
}

func (p *parser) callonfieldReducerOp16() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldReducerOp16()
}

func (c *current) onfieldReducerOp18() (interface{}, error) {
	return "First", nil
}

func (p *parser) callonfieldReducerOp18() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldReducerOp18()
}

func (c *current) onfieldReducerOp20() (interface{}, error) {
	return "Last", nil
}

func (p *parser) callonfieldReducerOp20() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldReducerOp20()
}

func (c *current) onpaddedFieldName1(field interface{}) (interface{}, error) {
	return field, nil
}

func (p *parser) callonpaddedFieldName1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onpaddedFieldName1(stack["field"])
}

func (c *current) oncountReducer1(op, field interface{}) (interface{}, error) {
	return makeReducer(op, "count", field), nil

}

func (p *parser) calloncountReducer1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.oncountReducer1(stack["op"], stack["field"])
}

func (c *current) onfieldReducer1(op, field interface{}) (interface{}, error) {
	return makeReducer(op, toLowerCase(op), field), nil

}

func (p *parser) callonfieldReducer1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldReducer1(stack["op"], stack["field"])
}

func (c *current) onreducerProc1(every, reducers, keys, limit interface{}) (interface{}, error) {
	if OR(keys, every) != nil {
		if keys != nil {
			keys = keys.([]interface{})[1]
		} else {
			keys = []interface{}{}
		}

		if every != nil {
			every = every.([]interface{})[0]
		}

		return makeGroupByProc(every, limit, keys, reducers), nil
	}

	return makeReducerProc(reducers), nil

}

func (p *parser) callonreducerProc1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onreducerProc1(stack["every"], stack["reducers"], stack["keys"], stack["limit"])
}

func (c *current) onasClause1(v interface{}) (interface{}, error) {
	return v, nil
}

func (p *parser) callonasClause1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onasClause1(stack["v"])
}

func (c *current) onreducerExpr2(field, f interface{}) (interface{}, error) {
	return overrideReducerVar(f, field), nil

}

func (p *parser) callonreducerExpr2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onreducerExpr2(stack["field"], stack["f"])
}

func (c *current) onreducerExpr13(f, field interface{}) (interface{}, error) {
	return overrideReducerVar(f, field), nil

}

func (p *parser) callonreducerExpr13() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onreducerExpr13(stack["f"], stack["field"])
}

func (c *current) onreducerList1(first, rest interface{}) (interface{}, error) {
	result := []interface{}{first}
	for _, r := range rest.([]interface{}) {
		result = append(result, r.([]interface{})[3])
	}
	return result, nil

}

func (p *parser) callonreducerList1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onreducerList1(stack["first"], stack["rest"])
}

func (c *current) onsort2(rev, limit, list interface{}) (interface{}, error) {
	sortdir := 1
	if rev != nil {
		sortdir = -1
	}
	return makeSortProc(list, sortdir, limit), nil

}

func (p *parser) callonsort2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsort2(stack["rev"], stack["limit"], stack["list"])
}

func (c *current) onsort20(limit, rev, list interface{}) (interface{}, error) {
	sortdir := 1
	if rev != nil {
		sortdir = -1
	}
	return makeSortProc(list, sortdir, limit), nil

}

func (p *parser) callonsort20() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsort20(stack["limit"], stack["rev"], stack["list"])
}

func (c *current) ontop1(limit, flush, list interface{}) (interface{}, error) {
	return makeTopProc(list, limit, flush), nil

}

func (p *parser) callontop1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.ontop1(stack["limit"], stack["flush"], stack["list"])
}

func (c *current) onprocLimitArg1(limit interface{}) (interface{}, error) {
	return limit, nil
}

func (p *parser) callonprocLimitArg1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onprocLimitArg1(stack["limit"])
}

func (c *current) oncut1(list interface{}) (interface{}, error) {
	return makeCutProc(list), nil
}

func (p *parser) calloncut1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.oncut1(stack["list"])
}

func (c *current) onhead2(count interface{}) (interface{}, error) {
	return makeHeadProc(count), nil
}

func (p *parser) callonhead2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onhead2(stack["count"])
}

func (c *current) onhead8() (interface{}, error) {
	return makeHeadProc(1), nil
}

func (p *parser) callonhead8() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onhead8()
}

func (c *current) ontail2(count interface{}) (interface{}, error) {
	return makeTailProc(count), nil
}

func (p *parser) callontail2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.ontail2(stack["count"])
}

func (c *current) ontail8() (interface{}, error) {
	return makeTailProc(1), nil
}

func (p *parser) callontail8() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.ontail8()
}

func (c *current) onfilter1(expr interface{}) (interface{}, error) {
	return makeFilterProc(expr), nil

}

func (p *parser) callonfilter1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfilter1(stack["expr"])
}

func (c *current) onuniq2() (interface{}, error) {
	return makeUniqProc(true), nil

}

func (p *parser) callonuniq2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onuniq2()
}

func (c *current) onuniq7() (interface{}, error) {
	return makeUniqProc(false), nil

}

func (p *parser) callonuniq7() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onuniq7()
}

func (c *current) onseconds2() (interface{}, error) {
	return makeDuration(1), nil
}

func (p *parser) callonseconds2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onseconds2()
}

func (c *current) onseconds4(num interface{}) (interface{}, error) {
	return makeDuration(num), nil
}

func (p *parser) callonseconds4() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onseconds4(stack["num"])
}

func (c *current) onminutes2() (interface{}, error) {
	return makeDuration(60), nil
}

func (p *parser) callonminutes2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onminutes2()
}

func (c *current) onminutes4(num interface{}) (interface{}, error) {
	return makeDuration(num.(int) * 60), nil
}

func (p *parser) callonminutes4() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onminutes4(stack["num"])
}

func (c *current) onhours2() (interface{}, error) {
	return makeDuration(3600), nil
}

func (p *parser) callonhours2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onhours2()
}

func (c *current) onhours4(num interface{}) (interface{}, error) {
	return makeDuration(num.(int) * 3600), nil
}

func (p *parser) callonhours4() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onhours4(stack["num"])
}

func (c *current) ondays2() (interface{}, error) {
	return makeDuration(3600 * 24), nil
}

func (p *parser) callondays2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.ondays2()
}

func (c *current) ondays4(num interface{}) (interface{}, error) {
	return makeDuration(num.(int) * 3600 * 24), nil
}

func (p *parser) callondays4() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.ondays4(stack["num"])
}

func (c *current) onweeks1(num interface{}) (interface{}, error) {
	return makeDuration(num.(int) * 3600 * 24 * 7), nil
}

func (p *parser) callonweeks1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onweeks1(stack["num"])
}

func (c *current) onaddr1(a interface{}) (interface{}, error) {
	return string(c.text), nil
}

func (p *parser) callonaddr1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onaddr1(stack["a"])
}

func (c *current) onport1(v interface{}) (interface{}, error) {
	return v, nil
}

func (p *parser) callonport1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onport1(stack["v"])
}

func (c *current) onip6addr2(a, b interface{}) (interface{}, error) {
	return joinChars(a) + b.(string), nil

}

func (p *parser) callonip6addr2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onip6addr2(stack["a"], stack["b"])
}

func (c *current) onip6addr9(a, b, d, e interface{}) (interface{}, error) {
	return a.(string) + joinChars(b) + "::" + joinChars(d) + e.(string), nil

}

func (p *parser) callonip6addr9() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onip6addr9(stack["a"], stack["b"], stack["d"], stack["e"])
}

func (c *current) onip6addr22(a, b interface{}) (interface{}, error) {
	return "::" + joinChars(a) + b.(string), nil

}

func (p *parser) callonip6addr22() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onip6addr22(stack["a"], stack["b"])
}

func (c *current) onip6addr30(a, b interface{}) (interface{}, error) {
	return a.(string) + joinChars(b) + "::", nil

}

func (p *parser) callonip6addr30() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onip6addr30(stack["a"], stack["b"])
}

func (c *current) onip6addr38() (interface{}, error) {
	return "::", nil

}

func (p *parser) callonip6addr38() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onip6addr38()
}

func (c *current) onh_append1(v interface{}) (interface{}, error) {
	return ":" + v.(string), nil
}

func (p *parser) callonh_append1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onh_append1(stack["v"])
}

func (c *current) onh_prepend1(v interface{}) (interface{}, error) {
	return v.(string) + ":", nil
}

func (p *parser) callonh_prepend1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onh_prepend1(stack["v"])
}

func (c *current) onsub_addr3(a interface{}) (interface{}, error) {
	return string(c.text) + ".0", nil
}

func (p *parser) callonsub_addr3() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsub_addr3(stack["a"])
}

func (c *current) onsub_addr11(a interface{}) (interface{}, error) {
	return string(c.text) + ".0.0", nil
}

func (p *parser) callonsub_addr11() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsub_addr11(stack["a"])
}

func (c *current) onsub_addr17(a interface{}) (interface{}, error) {
	return string(c.text) + ".0.0.0", nil
}

func (p *parser) callonsub_addr17() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsub_addr17(stack["a"])
}

func (c *current) onsubnet1(a, m interface{}) (interface{}, error) {
	return a.(string) + "/" + fmt.Sprintf("%v", m), nil

}

func (p *parser) callonsubnet1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsubnet1(stack["a"], stack["m"])
}

func (c *current) onip6subnet1(a, m interface{}) (interface{}, error) {
	return a.(string) + "/" + m.(string), nil

}

func (p *parser) callonip6subnet1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onip6subnet1(stack["a"], stack["m"])
}

func (c *current) oninteger1(s interface{}) (interface{}, error) {
	return parseInt(s), nil

}

func (p *parser) calloninteger1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.oninteger1(stack["s"])
}

func (c *current) onsinteger1(chars interface{}) (interface{}, error) {
	return string(c.text), nil

}

func (p *parser) callonsinteger1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsinteger1(stack["chars"])
}

func (c *current) ondouble1(s interface{}) (interface{}, error) {
	return parseFloat(s), nil

}

func (p *parser) callondouble1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.ondouble1(stack["s"])
}

func (c *current) onsdouble2() (interface{}, error) {
	return string(c.text), nil

}

func (p *parser) callonsdouble2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsdouble2()
}

func (c *current) onsdouble11() (interface{}, error) {
	return string(c.text), nil

}

func (p *parser) callonsdouble11() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsdouble11()
}

func (c *current) onh161(chars interface{}) (interface{}, error) {
	return string(c.text), nil
}

func (p *parser) callonh161() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onh161(stack["chars"])
}

func (c *current) onboomWord1(chars interface{}) (interface{}, error) {
	return string(c.text), nil
}

func (p *parser) callonboomWord1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onboomWord1(stack["chars"])
}

func (c *current) onquotedString2(v interface{}) (interface{}, error) {
	return joinChars(v), nil
}

func (p *parser) callonquotedString2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onquotedString2(stack["v"])
}

func (c *current) onquotedString9(v interface{}) (interface{}, error) {
	return joinChars(v), nil
}

func (p *parser) callonquotedString9() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onquotedString9(stack["v"])
}

func (c *current) ondoubleQuotedChar2() (interface{}, error) {
	return string(c.text), nil
}

func (p *parser) callondoubleQuotedChar2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.ondoubleQuotedChar2()
}

func (c *current) ondoubleQuotedChar9(s interface{}) (interface{}, error) {
	return s, nil
}

func (p *parser) callondoubleQuotedChar9() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.ondoubleQuotedChar9(stack["s"])
}

func (c *current) onsingleQuotedChar2() (interface{}, error) {
	return string(c.text), nil
}

func (p *parser) callonsingleQuotedChar2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsingleQuotedChar2()
}

func (c *current) onsingleQuotedChar9(s interface{}) (interface{}, error) {
	return s, nil
}

func (p *parser) callonsingleQuotedChar9() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsingleQuotedChar9(stack["s"])
}

func (c *current) onsingleCharEscape5() (interface{}, error) {
	return "\b", nil
}

func (p *parser) callonsingleCharEscape5() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsingleCharEscape5()
}

func (c *current) onsingleCharEscape7() (interface{}, error) {
	return "\f", nil
}

func (p *parser) callonsingleCharEscape7() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsingleCharEscape7()
}

func (c *current) onsingleCharEscape9() (interface{}, error) {
	return "\n", nil
}

func (p *parser) callonsingleCharEscape9() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsingleCharEscape9()
}

func (c *current) onsingleCharEscape11() (interface{}, error) {
	return "\r", nil
}

func (p *parser) callonsingleCharEscape11() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsingleCharEscape11()
}

func (c *current) onsingleCharEscape13() (interface{}, error) {
	return "\t", nil
}

func (p *parser) callonsingleCharEscape13() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsingleCharEscape13()
}

func (c *current) onsingleCharEscape15() (interface{}, error) {
	return "\v", nil
}

func (p *parser) callonsingleCharEscape15() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsingleCharEscape15()
}

func (c *current) onreString1(v interface{}) (interface{}, error) {
	return v, nil
}

func (p *parser) callonreString1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onreString1(stack["v"])
}

func (c *current) onreBody1() (interface{}, error) {
	return string(c.text), nil
}

func (p *parser) callonreBody1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onreBody1()
}

var (
	// errNoRule is returned when the grammar to parse has no rule.
	errNoRule = errors.New("grammar has no rule")

	// errInvalidEncoding is returned when the source is not properly
	// utf8-encoded.
	errInvalidEncoding = errors.New("invalid encoding")

	// errNoMatch is returned if no match could be found.
	errNoMatch = errors.New("no match found")
)

// Option is a function that can set an option on the parser. It returns
// the previous setting as an Option.
type Option func(*parser) Option

// Debug creates an Option to set the debug flag to b. When set to true,
// debugging information is printed to stdout while parsing.
//
// The default is false.
func Debug(b bool) Option {
	return func(p *parser) Option {
		old := p.debug
		p.debug = b
		return Debug(old)
	}
}

// Memoize creates an Option to set the memoize flag to b. When set to true,
// the parser will cache all results so each expression is evaluated only
// once. This guarantees linear parsing time even for pathological cases,
// at the expense of more memory and slower times for typical cases.
//
// The default is false.
func Memoize(b bool) Option {
	return func(p *parser) Option {
		old := p.memoize
		p.memoize = b
		return Memoize(old)
	}
}

// Recover creates an Option to set the recover flag to b. When set to
// true, this causes the parser to recover from panics and convert it
// to an error. Setting it to false can be useful while debugging to
// access the full stack trace.
//
// The default is true.
func Recover(b bool) Option {
	return func(p *parser) Option {
		old := p.recover
		p.recover = b
		return Recover(old)
	}
}

// ParseFile parses the file identified by filename.
func ParseFile(filename string, opts ...Option) (interface{}, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ParseReader(filename, f, opts...)
}

// ParseReader parses the data from r using filename as information in the
// error messages.
func ParseReader(filename string, r io.Reader, opts ...Option) (interface{}, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return Parse(filename, b, opts...)
}

// Parse parses the data from b using filename as information in the
// error messages.
func Parse(filename string, b []byte, opts ...Option) (interface{}, error) {
	return newParser(filename, b, opts...).parse(g)
}

// position records a position in the text.
type position struct {
	line, col, offset int
}

func (p position) String() string {
	return fmt.Sprintf("%d:%d [%d]", p.line, p.col, p.offset)
}

// savepoint stores all state required to go back to this point in the
// parser.
type savepoint struct {
	position
	rn rune
	w  int
}

type current struct {
	pos  position // start position of the match
	text []byte   // raw text of the match
}

// the AST types...

type grammar struct {
	pos   position
	rules []*rule
}

type rule struct {
	pos         position
	name        string
	displayName string
	expr        interface{}
}

type choiceExpr struct {
	pos          position
	alternatives []interface{}
}

type actionExpr struct {
	pos  position
	expr interface{}
	run  func(*parser) (interface{}, error)
}

type seqExpr struct {
	pos   position
	exprs []interface{}
}

type labeledExpr struct {
	pos   position
	label string
	expr  interface{}
}

type expr struct {
	pos  position
	expr interface{}
}

type andExpr expr
type notExpr expr
type zeroOrOneExpr expr
type zeroOrMoreExpr expr
type oneOrMoreExpr expr

type ruleRefExpr struct {
	pos  position
	name string
}

type andCodeExpr struct {
	pos position
	run func(*parser) (bool, error)
}

type notCodeExpr struct {
	pos position
	run func(*parser) (bool, error)
}

type litMatcher struct {
	pos        position
	val        string
	ignoreCase bool
}

type charClassMatcher struct {
	pos        position
	val        string
	chars      []rune
	ranges     []rune
	classes    []*unicode.RangeTable
	ignoreCase bool
	inverted   bool
}

type anyMatcher position

// errList cumulates the errors found by the parser.
type errList []error

func (e *errList) add(err error) {
	*e = append(*e, err)
}

func (e errList) err() error {
	if len(e) == 0 {
		return nil
	}
	e.dedupe()
	return e
}

func (e *errList) dedupe() {
	var cleaned []error
	set := make(map[string]bool)
	for _, err := range *e {
		if msg := err.Error(); !set[msg] {
			set[msg] = true
			cleaned = append(cleaned, err)
		}
	}
	*e = cleaned
}

func (e errList) Error() string {
	switch len(e) {
	case 0:
		return ""
	case 1:
		return e[0].Error()
	default:
		var buf bytes.Buffer

		for i, err := range e {
			if i > 0 {
				buf.WriteRune('\n')
			}
			buf.WriteString(err.Error())
		}
		return buf.String()
	}
}

// parserError wraps an error with a prefix indicating the rule in which
// the error occurred. The original error is stored in the Inner field.
type parserError struct {
	Inner  error
	pos    position
	prefix string
}

// Error returns the error message.
func (p *parserError) Error() string {
	return p.prefix + ": " + p.Inner.Error()
}

// newParser creates a parser with the specified input source and options.
func newParser(filename string, b []byte, opts ...Option) *parser {
	p := &parser{
		filename: filename,
		errs:     new(errList),
		data:     b,
		pt:       savepoint{position: position{line: 1}},
		recover:  true,
	}
	p.setOptions(opts)
	return p
}

// setOptions applies the options to the parser.
func (p *parser) setOptions(opts []Option) {
	for _, opt := range opts {
		opt(p)
	}
}

type resultTuple struct {
	v   interface{}
	b   bool
	end savepoint
}

type parser struct {
	filename string
	pt       savepoint
	cur      current

	data []byte
	errs *errList

	recover bool
	debug   bool
	depth   int

	memoize bool
	// memoization table for the packrat algorithm:
	// map[offset in source] map[expression or rule] {value, match}
	memo map[int]map[interface{}]resultTuple

	// rules table, maps the rule identifier to the rule node
	rules map[string]*rule
	// variables stack, map of label to value
	vstack []map[string]interface{}
	// rule stack, allows identification of the current rule in errors
	rstack []*rule

	// stats
	exprCnt int
}

// push a variable set on the vstack.
func (p *parser) pushV() {
	if cap(p.vstack) == len(p.vstack) {
		// create new empty slot in the stack
		p.vstack = append(p.vstack, nil)
	} else {
		// slice to 1 more
		p.vstack = p.vstack[:len(p.vstack)+1]
	}

	// get the last args set
	m := p.vstack[len(p.vstack)-1]
	if m != nil && len(m) == 0 {
		// empty map, all good
		return
	}

	m = make(map[string]interface{})
	p.vstack[len(p.vstack)-1] = m
}

// pop a variable set from the vstack.
func (p *parser) popV() {
	// if the map is not empty, clear it
	m := p.vstack[len(p.vstack)-1]
	if len(m) > 0 {
		// GC that map
		p.vstack[len(p.vstack)-1] = nil
	}
	p.vstack = p.vstack[:len(p.vstack)-1]
}

func (p *parser) print(prefix, s string) string {
	if !p.debug {
		return s
	}

	fmt.Printf("%s %d:%d:%d: %s [%#U]\n",
		prefix, p.pt.line, p.pt.col, p.pt.offset, s, p.pt.rn)
	return s
}

func (p *parser) in(s string) string {
	p.depth++
	return p.print(strings.Repeat(" ", p.depth)+">", s)
}

func (p *parser) out(s string) string {
	p.depth--
	return p.print(strings.Repeat(" ", p.depth)+"<", s)
}

func (p *parser) addErr(err error) {
	p.addErrAt(err, p.pt.position)
}

func (p *parser) addErrAt(err error, pos position) {
	var buf bytes.Buffer
	if p.filename != "" {
		buf.WriteString(p.filename)
	}
	if buf.Len() > 0 {
		buf.WriteString(":")
	}
	buf.WriteString(fmt.Sprintf("%d:%d (%d)", pos.line, pos.col, pos.offset))
	if len(p.rstack) > 0 {
		if buf.Len() > 0 {
			buf.WriteString(": ")
		}
		rule := p.rstack[len(p.rstack)-1]
		if rule.displayName != "" {
			buf.WriteString("rule " + rule.displayName)
		} else {
			buf.WriteString("rule " + rule.name)
		}
	}
	pe := &parserError{Inner: err, pos: pos, prefix: buf.String()}
	p.errs.add(pe)
}

// read advances the parser to the next rune.
func (p *parser) read() {
	p.pt.offset += p.pt.w
	rn, n := utf8.DecodeRune(p.data[p.pt.offset:])
	p.pt.rn = rn
	p.pt.w = n
	p.pt.col++
	if rn == '\n' {
		p.pt.line++
		p.pt.col = 0
	}

	if rn == utf8.RuneError {
		if n == 1 {
			p.addErr(errInvalidEncoding)
		}
	}
}

// restore parser position to the savepoint pt.
func (p *parser) restore(pt savepoint) {
	if p.debug {
		defer p.out(p.in("restore"))
	}
	if pt.offset == p.pt.offset {
		return
	}
	p.pt = pt
}

// get the slice of bytes from the savepoint start to the current position.
func (p *parser) sliceFrom(start savepoint) []byte {
	return p.data[start.position.offset:p.pt.position.offset]
}

func (p *parser) getMemoized(node interface{}) (resultTuple, bool) {
	if len(p.memo) == 0 {
		return resultTuple{}, false
	}
	m := p.memo[p.pt.offset]
	if len(m) == 0 {
		return resultTuple{}, false
	}
	res, ok := m[node]
	return res, ok
}

func (p *parser) setMemoized(pt savepoint, node interface{}, tuple resultTuple) {
	if p.memo == nil {
		p.memo = make(map[int]map[interface{}]resultTuple)
	}
	m := p.memo[pt.offset]
	if m == nil {
		m = make(map[interface{}]resultTuple)
		p.memo[pt.offset] = m
	}
	m[node] = tuple
}

func (p *parser) buildRulesTable(g *grammar) {
	p.rules = make(map[string]*rule, len(g.rules))
	for _, r := range g.rules {
		p.rules[r.name] = r
	}
}

func (p *parser) parse(g *grammar) (val interface{}, err error) {
	if len(g.rules) == 0 {
		p.addErr(errNoRule)
		return nil, p.errs.err()
	}

	// TODO : not super critical but this could be generated
	p.buildRulesTable(g)

	if p.recover {
		// panic can be used in action code to stop parsing immediately
		// and return the panic as an error.
		defer func() {
			if e := recover(); e != nil {
				if p.debug {
					defer p.out(p.in("panic handler"))
				}
				val = nil
				switch e := e.(type) {
				case error:
					p.addErr(e)
				default:
					p.addErr(fmt.Errorf("%v", e))
				}
				err = p.errs.err()
			}
		}()
	}

	// start rule is rule [0]
	p.read() // advance to first rune
	val, ok := p.parseRule(g.rules[0])
	if !ok {
		if len(*p.errs) == 0 {
			// make sure this doesn't go out silently
			p.addErr(errNoMatch)
		}
		return nil, p.errs.err()
	}
	return val, p.errs.err()
}

func (p *parser) parseRule(rule *rule) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseRule " + rule.name))
	}

	if p.memoize {
		res, ok := p.getMemoized(rule)
		if ok {
			p.restore(res.end)
			return res.v, res.b
		}
	}

	start := p.pt
	p.rstack = append(p.rstack, rule)
	p.pushV()
	val, ok := p.parseExpr(rule.expr)
	p.popV()
	p.rstack = p.rstack[:len(p.rstack)-1]
	if ok && p.debug {
		p.print(strings.Repeat(" ", p.depth)+"MATCH", string(p.sliceFrom(start)))
	}

	if p.memoize {
		p.setMemoized(start, rule, resultTuple{val, ok, p.pt})
	}
	return val, ok
}

func (p *parser) parseExpr(expr interface{}) (interface{}, bool) {
	var pt savepoint
	var ok bool

	if p.memoize {
		res, ok := p.getMemoized(expr)
		if ok {
			p.restore(res.end)
			return res.v, res.b
		}
		pt = p.pt
	}

	p.exprCnt++
	var val interface{}
	switch expr := expr.(type) {
	case *actionExpr:
		val, ok = p.parseActionExpr(expr)
	case *andCodeExpr:
		val, ok = p.parseAndCodeExpr(expr)
	case *andExpr:
		val, ok = p.parseAndExpr(expr)
	case *anyMatcher:
		val, ok = p.parseAnyMatcher(expr)
	case *charClassMatcher:
		val, ok = p.parseCharClassMatcher(expr)
	case *choiceExpr:
		val, ok = p.parseChoiceExpr(expr)
	case *labeledExpr:
		val, ok = p.parseLabeledExpr(expr)
	case *litMatcher:
		val, ok = p.parseLitMatcher(expr)
	case *notCodeExpr:
		val, ok = p.parseNotCodeExpr(expr)
	case *notExpr:
		val, ok = p.parseNotExpr(expr)
	case *oneOrMoreExpr:
		val, ok = p.parseOneOrMoreExpr(expr)
	case *ruleRefExpr:
		val, ok = p.parseRuleRefExpr(expr)
	case *seqExpr:
		val, ok = p.parseSeqExpr(expr)
	case *zeroOrMoreExpr:
		val, ok = p.parseZeroOrMoreExpr(expr)
	case *zeroOrOneExpr:
		val, ok = p.parseZeroOrOneExpr(expr)
	default:
		panic(fmt.Sprintf("unknown expression type %T", expr))
	}
	if p.memoize {
		p.setMemoized(pt, expr, resultTuple{val, ok, p.pt})
	}
	return val, ok
}

func (p *parser) parseActionExpr(act *actionExpr) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseActionExpr"))
	}

	start := p.pt
	val, ok := p.parseExpr(act.expr)
	if ok {
		p.cur.pos = start.position
		p.cur.text = p.sliceFrom(start)
		actVal, err := act.run(p)
		if err != nil {
			p.addErrAt(err, start.position)
		}
		val = actVal
	}
	if ok && p.debug {
		p.print(strings.Repeat(" ", p.depth)+"MATCH", string(p.sliceFrom(start)))
	}
	return val, ok
}

func (p *parser) parseAndCodeExpr(and *andCodeExpr) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseAndCodeExpr"))
	}

	ok, err := and.run(p)
	if err != nil {
		p.addErr(err)
	}
	return nil, ok
}

func (p *parser) parseAndExpr(and *andExpr) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseAndExpr"))
	}

	pt := p.pt
	p.pushV()
	_, ok := p.parseExpr(and.expr)
	p.popV()
	p.restore(pt)
	return nil, ok
}

func (p *parser) parseAnyMatcher(any *anyMatcher) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseAnyMatcher"))
	}

	if p.pt.rn != utf8.RuneError {
		start := p.pt
		p.read()
		return p.sliceFrom(start), true
	}
	return nil, false
}

func (p *parser) parseCharClassMatcher(chr *charClassMatcher) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseCharClassMatcher"))
	}

	cur := p.pt.rn
	// can't match EOF
	if cur == utf8.RuneError {
		return nil, false
	}
	start := p.pt
	if chr.ignoreCase {
		cur = unicode.ToLower(cur)
	}

	// try to match in the list of available chars
	for _, rn := range chr.chars {
		if rn == cur {
			if chr.inverted {
				return nil, false
			}
			p.read()
			return p.sliceFrom(start), true
		}
	}

	// try to match in the list of ranges
	for i := 0; i < len(chr.ranges); i += 2 {
		if cur >= chr.ranges[i] && cur <= chr.ranges[i+1] {
			if chr.inverted {
				return nil, false
			}
			p.read()
			return p.sliceFrom(start), true
		}
	}

	// try to match in the list of Unicode classes
	for _, cl := range chr.classes {
		if unicode.Is(cl, cur) {
			if chr.inverted {
				return nil, false
			}
			p.read()
			return p.sliceFrom(start), true
		}
	}

	if chr.inverted {
		p.read()
		return p.sliceFrom(start), true
	}
	return nil, false
}

func (p *parser) parseChoiceExpr(ch *choiceExpr) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseChoiceExpr"))
	}

	for _, alt := range ch.alternatives {
		p.pushV()
		val, ok := p.parseExpr(alt)
		p.popV()
		if ok {
			return val, ok
		}
	}
	return nil, false
}

func (p *parser) parseLabeledExpr(lab *labeledExpr) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseLabeledExpr"))
	}

	p.pushV()
	val, ok := p.parseExpr(lab.expr)
	p.popV()
	if ok && lab.label != "" {
		m := p.vstack[len(p.vstack)-1]
		m[lab.label] = val
	}
	return val, ok
}

func (p *parser) parseLitMatcher(lit *litMatcher) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseLitMatcher"))
	}

	start := p.pt
	for _, want := range lit.val {
		cur := p.pt.rn
		if lit.ignoreCase {
			cur = unicode.ToLower(cur)
		}
		if cur != want {
			p.restore(start)
			return nil, false
		}
		p.read()
	}
	return p.sliceFrom(start), true
}

func (p *parser) parseNotCodeExpr(not *notCodeExpr) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseNotCodeExpr"))
	}

	ok, err := not.run(p)
	if err != nil {
		p.addErr(err)
	}
	return nil, !ok
}

func (p *parser) parseNotExpr(not *notExpr) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseNotExpr"))
	}

	pt := p.pt
	p.pushV()
	_, ok := p.parseExpr(not.expr)
	p.popV()
	p.restore(pt)
	return nil, !ok
}

func (p *parser) parseOneOrMoreExpr(expr *oneOrMoreExpr) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseOneOrMoreExpr"))
	}

	var vals []interface{}

	for {
		p.pushV()
		val, ok := p.parseExpr(expr.expr)
		p.popV()
		if !ok {
			if len(vals) == 0 {
				// did not match once, no match
				return nil, false
			}
			return vals, true
		}
		vals = append(vals, val)
	}
}

func (p *parser) parseRuleRefExpr(ref *ruleRefExpr) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseRuleRefExpr " + ref.name))
	}

	if ref.name == "" {
		panic(fmt.Sprintf("%s: invalid rule: missing name", ref.pos))
	}

	rule := p.rules[ref.name]
	if rule == nil {
		p.addErr(fmt.Errorf("undefined rule: %s", ref.name))
		return nil, false
	}
	return p.parseRule(rule)
}

func (p *parser) parseSeqExpr(seq *seqExpr) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseSeqExpr"))
	}

	var vals []interface{}

	pt := p.pt
	for _, expr := range seq.exprs {
		val, ok := p.parseExpr(expr)
		if !ok {
			p.restore(pt)
			return nil, false
		}
		vals = append(vals, val)
	}
	return vals, true
}

func (p *parser) parseZeroOrMoreExpr(expr *zeroOrMoreExpr) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseZeroOrMoreExpr"))
	}

	var vals []interface{}

	for {
		p.pushV()
		val, ok := p.parseExpr(expr.expr)
		p.popV()
		if !ok {
			return vals, true
		}
		vals = append(vals, val)
	}
}

func (p *parser) parseZeroOrOneExpr(expr *zeroOrOneExpr) (interface{}, bool) {
	if p.debug {
		defer p.out(p.in("parseZeroOrOneExpr"))
	}

	p.pushV()
	val, _ := p.parseExpr(expr.expr)
	p.popV()
	// whether it matched or not, consider it a match
	return val, true
}

func rangeTable(class string) *unicode.RangeTable {
	if rt, ok := unicode.Categories[class]; ok {
		return rt
	}
	if rt, ok := unicode.Properties[class]; ok {
		return rt
	}
	if rt, ok := unicode.Scripts[class]; ok {
		return rt
	}

	// cannot happen
	panic(fmt.Sprintf("invalid Unicode class: %s", class))
}

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

func makeBooleanLiteral(val bool) *ast.BooleanLiteral {
	return &ast.BooleanLiteral{ast.Node{"BooleanLiteral"}, val}
}

func makeCompareField(comparatorIn, fieldIn, valueIn interface{}) *ast.CompareField {
	comparator := comparatorIn.(string)
	field := fieldIn.(ast.FieldExpr)
	value := valueIn.(*ast.TypedValue)
	return &ast.CompareField{ast.Node{"CompareField"}, comparator, field, *value}
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

func fieldExprArray(val interface{}) []ast.FieldExpr {
	var ret []ast.FieldExpr
	if val != nil {
		for _, f := range val.([]interface{}) {
			ret = append(ret, f.(ast.FieldExpr))
		}
	}
	return ret
}

func makeSortProc(fieldsIn, dirIn, limitIn interface{}) *ast.SortProc {
	fields := fieldExprArray(fieldsIn)
	sortdir := dirIn.(int)
	var limit int
	if limitIn != nil {
		limit = limitIn.(int)
	}
	return &ast.SortProc{ast.Node{"SortProc"}, limit, fields, sortdir}
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
			pos:  position{line: 313, col: 1, offset: 8975},
			expr: &actionExpr{
				pos: position{line: 313, col: 9, offset: 8983},
				run: (*parser).callonstart1,
				expr: &seqExpr{
					pos: position{line: 313, col: 9, offset: 8983},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 313, col: 9, offset: 8983},
							expr: &ruleRefExpr{
								pos:  position{line: 313, col: 9, offset: 8983},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 313, col: 12, offset: 8986},
							label: "ast",
							expr: &ruleRefExpr{
								pos:  position{line: 313, col: 16, offset: 8990},
								name: "boomCommand",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 313, col: 28, offset: 9002},
							expr: &ruleRefExpr{
								pos:  position{line: 313, col: 28, offset: 9002},
								name: "_",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 313, col: 31, offset: 9005},
							name: "EOF",
						},
					},
				},
			},
		},
		{
			name: "boomCommand",
			pos:  position{line: 315, col: 1, offset: 9030},
			expr: &choiceExpr{
				pos: position{line: 316, col: 5, offset: 9046},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 316, col: 5, offset: 9046},
						run: (*parser).callonboomCommand2,
						expr: &labeledExpr{
							pos:   position{line: 316, col: 5, offset: 9046},
							label: "procs",
							expr: &ruleRefExpr{
								pos:  position{line: 316, col: 11, offset: 9052},
								name: "procChain",
							},
						},
					},
					&actionExpr{
						pos: position{line: 320, col: 5, offset: 9225},
						run: (*parser).callonboomCommand5,
						expr: &seqExpr{
							pos: position{line: 320, col: 5, offset: 9225},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 320, col: 5, offset: 9225},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 320, col: 7, offset: 9227},
										name: "search",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 320, col: 14, offset: 9234},
									expr: &ruleRefExpr{
										pos:  position{line: 320, col: 14, offset: 9234},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 320, col: 17, offset: 9237},
									label: "rest",
									expr: &zeroOrMoreExpr{
										pos: position{line: 320, col: 22, offset: 9242},
										expr: &ruleRefExpr{
											pos:  position{line: 320, col: 22, offset: 9242},
											name: "chainedProc",
										},
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 327, col: 5, offset: 9452},
						run: (*parser).callonboomCommand14,
						expr: &labeledExpr{
							pos:   position{line: 327, col: 5, offset: 9452},
							label: "s",
							expr: &ruleRefExpr{
								pos:  position{line: 327, col: 7, offset: 9454},
								name: "search",
							},
						},
					},
				},
			},
		},
		{
			name: "procChain",
			pos:  position{line: 331, col: 1, offset: 9525},
			expr: &actionExpr{
				pos: position{line: 332, col: 5, offset: 9539},
				run: (*parser).callonprocChain1,
				expr: &seqExpr{
					pos: position{line: 332, col: 5, offset: 9539},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 332, col: 5, offset: 9539},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 332, col: 11, offset: 9545},
								name: "proc",
							},
						},
						&labeledExpr{
							pos:   position{line: 332, col: 16, offset: 9550},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 332, col: 21, offset: 9555},
								expr: &ruleRefExpr{
									pos:  position{line: 332, col: 21, offset: 9555},
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
			pos:  position{line: 340, col: 1, offset: 9741},
			expr: &actionExpr{
				pos: position{line: 340, col: 15, offset: 9755},
				run: (*parser).callonchainedProc1,
				expr: &seqExpr{
					pos: position{line: 340, col: 15, offset: 9755},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 340, col: 15, offset: 9755},
							expr: &ruleRefExpr{
								pos:  position{line: 340, col: 15, offset: 9755},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 340, col: 18, offset: 9758},
							val:        "|",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 340, col: 22, offset: 9762},
							expr: &ruleRefExpr{
								pos:  position{line: 340, col: 22, offset: 9762},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 340, col: 25, offset: 9765},
							label: "p",
							expr: &ruleRefExpr{
								pos:  position{line: 340, col: 27, offset: 9767},
								name: "proc",
							},
						},
					},
				},
			},
		},
		{
			name: "search",
			pos:  position{line: 342, col: 1, offset: 9791},
			expr: &actionExpr{
				pos: position{line: 343, col: 5, offset: 9802},
				run: (*parser).callonsearch1,
				expr: &labeledExpr{
					pos:   position{line: 343, col: 5, offset: 9802},
					label: "expr",
					expr: &ruleRefExpr{
						pos:  position{line: 343, col: 10, offset: 9807},
						name: "searchExpr",
					},
				},
			},
		},
		{
			name: "searchExpr",
			pos:  position{line: 347, col: 1, offset: 9866},
			expr: &actionExpr{
				pos: position{line: 348, col: 5, offset: 9881},
				run: (*parser).callonsearchExpr1,
				expr: &seqExpr{
					pos: position{line: 348, col: 5, offset: 9881},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 348, col: 5, offset: 9881},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 348, col: 11, offset: 9887},
								name: "searchTerm",
							},
						},
						&labeledExpr{
							pos:   position{line: 348, col: 22, offset: 9898},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 348, col: 27, offset: 9903},
								expr: &ruleRefExpr{
									pos:  position{line: 348, col: 27, offset: 9903},
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
			pos:  position{line: 352, col: 1, offset: 9971},
			expr: &actionExpr{
				pos: position{line: 352, col: 18, offset: 9988},
				run: (*parser).callonoredSearchTerm1,
				expr: &seqExpr{
					pos: position{line: 352, col: 18, offset: 9988},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 352, col: 18, offset: 9988},
							name: "_",
						},
						&ruleRefExpr{
							pos:  position{line: 352, col: 20, offset: 9990},
							name: "orToken",
						},
						&ruleRefExpr{
							pos:  position{line: 352, col: 28, offset: 9998},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 352, col: 30, offset: 10000},
							label: "t",
							expr: &ruleRefExpr{
								pos:  position{line: 352, col: 32, offset: 10002},
								name: "searchTerm",
							},
						},
					},
				},
			},
		},
		{
			name: "searchTerm",
			pos:  position{line: 354, col: 1, offset: 10032},
			expr: &actionExpr{
				pos: position{line: 355, col: 5, offset: 10047},
				run: (*parser).callonsearchTerm1,
				expr: &seqExpr{
					pos: position{line: 355, col: 5, offset: 10047},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 355, col: 5, offset: 10047},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 355, col: 11, offset: 10053},
								name: "searchFactor",
							},
						},
						&labeledExpr{
							pos:   position{line: 355, col: 24, offset: 10066},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 355, col: 29, offset: 10071},
								expr: &ruleRefExpr{
									pos:  position{line: 355, col: 29, offset: 10071},
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
			pos:  position{line: 359, col: 1, offset: 10141},
			expr: &actionExpr{
				pos: position{line: 359, col: 19, offset: 10159},
				run: (*parser).callonandedSearchTerm1,
				expr: &seqExpr{
					pos: position{line: 359, col: 19, offset: 10159},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 359, col: 19, offset: 10159},
							name: "_",
						},
						&zeroOrOneExpr{
							pos: position{line: 359, col: 21, offset: 10161},
							expr: &seqExpr{
								pos: position{line: 359, col: 22, offset: 10162},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 359, col: 22, offset: 10162},
										name: "andToken",
									},
									&ruleRefExpr{
										pos:  position{line: 359, col: 31, offset: 10171},
										name: "_",
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 359, col: 35, offset: 10175},
							label: "f",
							expr: &ruleRefExpr{
								pos:  position{line: 359, col: 37, offset: 10177},
								name: "searchFactor",
							},
						},
					},
				},
			},
		},
		{
			name: "searchFactor",
			pos:  position{line: 361, col: 1, offset: 10209},
			expr: &choiceExpr{
				pos: position{line: 362, col: 5, offset: 10226},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 362, col: 5, offset: 10226},
						run: (*parser).callonsearchFactor2,
						expr: &seqExpr{
							pos: position{line: 362, col: 5, offset: 10226},
							exprs: []interface{}{
								&choiceExpr{
									pos: position{line: 362, col: 6, offset: 10227},
									alternatives: []interface{}{
										&seqExpr{
											pos: position{line: 362, col: 6, offset: 10227},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 362, col: 6, offset: 10227},
													name: "notToken",
												},
												&ruleRefExpr{
													pos:  position{line: 362, col: 15, offset: 10236},
													name: "_",
												},
											},
										},
										&seqExpr{
											pos: position{line: 362, col: 19, offset: 10240},
											exprs: []interface{}{
												&litMatcher{
													pos:        position{line: 362, col: 19, offset: 10240},
													val:        "!",
													ignoreCase: false,
												},
												&zeroOrOneExpr{
													pos: position{line: 362, col: 23, offset: 10244},
													expr: &ruleRefExpr{
														pos:  position{line: 362, col: 23, offset: 10244},
														name: "_",
													},
												},
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 362, col: 27, offset: 10248},
									label: "e",
									expr: &ruleRefExpr{
										pos:  position{line: 362, col: 29, offset: 10250},
										name: "searchExpr",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 365, col: 5, offset: 10309},
						run: (*parser).callonsearchFactor14,
						expr: &seqExpr{
							pos: position{line: 365, col: 5, offset: 10309},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 365, col: 5, offset: 10309},
									expr: &litMatcher{
										pos:        position{line: 365, col: 7, offset: 10311},
										val:        "-",
										ignoreCase: false,
									},
								},
								&labeledExpr{
									pos:   position{line: 365, col: 12, offset: 10316},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 365, col: 14, offset: 10318},
										name: "searchPred",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 366, col: 5, offset: 10351},
						run: (*parser).callonsearchFactor20,
						expr: &seqExpr{
							pos: position{line: 366, col: 5, offset: 10351},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 366, col: 5, offset: 10351},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 366, col: 9, offset: 10355},
									expr: &ruleRefExpr{
										pos:  position{line: 366, col: 9, offset: 10355},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 366, col: 12, offset: 10358},
									label: "expr",
									expr: &ruleRefExpr{
										pos:  position{line: 366, col: 17, offset: 10363},
										name: "searchExpr",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 366, col: 28, offset: 10374},
									expr: &ruleRefExpr{
										pos:  position{line: 366, col: 28, offset: 10374},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 366, col: 31, offset: 10377},
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
			pos:  position{line: 368, col: 1, offset: 10403},
			expr: &choiceExpr{
				pos: position{line: 369, col: 5, offset: 10418},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 369, col: 5, offset: 10418},
						run: (*parser).callonsearchPred2,
						expr: &seqExpr{
							pos: position{line: 369, col: 5, offset: 10418},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 369, col: 5, offset: 10418},
									val:        "*",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 369, col: 9, offset: 10422},
									expr: &ruleRefExpr{
										pos:  position{line: 369, col: 9, offset: 10422},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 369, col: 12, offset: 10425},
									label: "fieldComparator",
									expr: &ruleRefExpr{
										pos:  position{line: 369, col: 28, offset: 10441},
										name: "equalityToken",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 369, col: 42, offset: 10455},
									expr: &ruleRefExpr{
										pos:  position{line: 369, col: 42, offset: 10455},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 369, col: 45, offset: 10458},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 369, col: 47, offset: 10460},
										name: "searchValue",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 372, col: 5, offset: 10537},
						run: (*parser).callonsearchPred13,
						expr: &litMatcher{
							pos:        position{line: 372, col: 5, offset: 10537},
							val:        "*",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 375, col: 5, offset: 10596},
						run: (*parser).callonsearchPred15,
						expr: &seqExpr{
							pos: position{line: 375, col: 5, offset: 10596},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 375, col: 5, offset: 10596},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 375, col: 7, offset: 10598},
										name: "fieldExpr",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 375, col: 17, offset: 10608},
									expr: &ruleRefExpr{
										pos:  position{line: 375, col: 17, offset: 10608},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 375, col: 20, offset: 10611},
									label: "fieldComparator",
									expr: &ruleRefExpr{
										pos:  position{line: 375, col: 36, offset: 10627},
										name: "equalityToken",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 375, col: 50, offset: 10641},
									expr: &ruleRefExpr{
										pos:  position{line: 375, col: 50, offset: 10641},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 375, col: 53, offset: 10644},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 375, col: 55, offset: 10646},
										name: "searchValue",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 378, col: 5, offset: 10728},
						run: (*parser).callonsearchPred27,
						expr: &seqExpr{
							pos: position{line: 378, col: 5, offset: 10728},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 378, col: 5, offset: 10728},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 378, col: 7, offset: 10730},
										name: "searchValue",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 378, col: 19, offset: 10742},
									expr: &ruleRefExpr{
										pos:  position{line: 378, col: 19, offset: 10742},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 378, col: 22, offset: 10745},
									name: "inToken",
								},
								&zeroOrOneExpr{
									pos: position{line: 378, col: 30, offset: 10753},
									expr: &ruleRefExpr{
										pos:  position{line: 378, col: 30, offset: 10753},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 378, col: 33, offset: 10756},
									val:        "*",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 381, col: 5, offset: 10814},
						run: (*parser).callonsearchPred37,
						expr: &seqExpr{
							pos: position{line: 381, col: 5, offset: 10814},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 381, col: 5, offset: 10814},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 381, col: 7, offset: 10816},
										name: "searchValue",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 381, col: 19, offset: 10828},
									expr: &ruleRefExpr{
										pos:  position{line: 381, col: 19, offset: 10828},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 381, col: 22, offset: 10831},
									name: "inToken",
								},
								&zeroOrOneExpr{
									pos: position{line: 381, col: 30, offset: 10839},
									expr: &ruleRefExpr{
										pos:  position{line: 381, col: 30, offset: 10839},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 381, col: 33, offset: 10842},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 381, col: 35, offset: 10844},
										name: "fieldReference",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 384, col: 5, offset: 10918},
						run: (*parser).callonsearchPred48,
						expr: &labeledExpr{
							pos:   position{line: 384, col: 5, offset: 10918},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 384, col: 7, offset: 10920},
								name: "searchValue",
							},
						},
					},
				},
			},
		},
		{
			name: "searchValue",
			pos:  position{line: 393, col: 1, offset: 11214},
			expr: &choiceExpr{
				pos: position{line: 394, col: 5, offset: 11230},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 394, col: 5, offset: 11230},
						run: (*parser).callonsearchValue2,
						expr: &labeledExpr{
							pos:   position{line: 394, col: 5, offset: 11230},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 394, col: 7, offset: 11232},
								name: "quotedString",
							},
						},
					},
					&actionExpr{
						pos: position{line: 397, col: 5, offset: 11303},
						run: (*parser).callonsearchValue5,
						expr: &labeledExpr{
							pos:   position{line: 397, col: 5, offset: 11303},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 397, col: 7, offset: 11305},
								name: "reString",
							},
						},
					},
					&actionExpr{
						pos: position{line: 400, col: 5, offset: 11372},
						run: (*parser).callonsearchValue8,
						expr: &labeledExpr{
							pos:   position{line: 400, col: 5, offset: 11372},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 400, col: 7, offset: 11374},
								name: "port",
							},
						},
					},
					&actionExpr{
						pos: position{line: 403, col: 5, offset: 11433},
						run: (*parser).callonsearchValue11,
						expr: &labeledExpr{
							pos:   position{line: 403, col: 5, offset: 11433},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 403, col: 7, offset: 11435},
								name: "ip6subnet",
							},
						},
					},
					&actionExpr{
						pos: position{line: 406, col: 5, offset: 11503},
						run: (*parser).callonsearchValue14,
						expr: &labeledExpr{
							pos:   position{line: 406, col: 5, offset: 11503},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 406, col: 7, offset: 11505},
								name: "ip6addr",
							},
						},
					},
					&actionExpr{
						pos: position{line: 409, col: 5, offset: 11569},
						run: (*parser).callonsearchValue17,
						expr: &labeledExpr{
							pos:   position{line: 409, col: 5, offset: 11569},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 409, col: 7, offset: 11571},
								name: "subnet",
							},
						},
					},
					&actionExpr{
						pos: position{line: 412, col: 5, offset: 11636},
						run: (*parser).callonsearchValue20,
						expr: &labeledExpr{
							pos:   position{line: 412, col: 5, offset: 11636},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 412, col: 7, offset: 11638},
								name: "addr",
							},
						},
					},
					&actionExpr{
						pos: position{line: 415, col: 5, offset: 11699},
						run: (*parser).callonsearchValue23,
						expr: &labeledExpr{
							pos:   position{line: 415, col: 5, offset: 11699},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 415, col: 7, offset: 11701},
								name: "sdouble",
							},
						},
					},
					&actionExpr{
						pos: position{line: 418, col: 5, offset: 11767},
						run: (*parser).callonsearchValue26,
						expr: &seqExpr{
							pos: position{line: 418, col: 5, offset: 11767},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 418, col: 5, offset: 11767},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 418, col: 7, offset: 11769},
										name: "sinteger",
									},
								},
								&notExpr{
									pos: position{line: 418, col: 16, offset: 11778},
									expr: &ruleRefExpr{
										pos:  position{line: 418, col: 17, offset: 11779},
										name: "boomWord",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 421, col: 5, offset: 11843},
						run: (*parser).callonsearchValue32,
						expr: &seqExpr{
							pos: position{line: 421, col: 5, offset: 11843},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 421, col: 5, offset: 11843},
									expr: &seqExpr{
										pos: position{line: 421, col: 7, offset: 11845},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 421, col: 7, offset: 11845},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 421, col: 22, offset: 11860},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 421, col: 25, offset: 11863},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 421, col: 27, offset: 11865},
										name: "booleanLiteral",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 422, col: 5, offset: 11902},
						run: (*parser).callonsearchValue40,
						expr: &seqExpr{
							pos: position{line: 422, col: 5, offset: 11902},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 422, col: 5, offset: 11902},
									expr: &seqExpr{
										pos: position{line: 422, col: 7, offset: 11904},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 422, col: 7, offset: 11904},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 422, col: 22, offset: 11919},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 422, col: 25, offset: 11922},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 422, col: 27, offset: 11924},
										name: "unsetLiteral",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 423, col: 5, offset: 11959},
						run: (*parser).callonsearchValue48,
						expr: &seqExpr{
							pos: position{line: 423, col: 5, offset: 11959},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 423, col: 5, offset: 11959},
									expr: &seqExpr{
										pos: position{line: 423, col: 7, offset: 11961},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 423, col: 7, offset: 11961},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 423, col: 22, offset: 11976},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 423, col: 25, offset: 11979},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 423, col: 27, offset: 11981},
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
			pos:  position{line: 431, col: 1, offset: 12205},
			expr: &choiceExpr{
				pos: position{line: 432, col: 5, offset: 12224},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 432, col: 5, offset: 12224},
						name: "andToken",
					},
					&ruleRefExpr{
						pos:  position{line: 433, col: 5, offset: 12237},
						name: "orToken",
					},
					&ruleRefExpr{
						pos:  position{line: 434, col: 5, offset: 12249},
						name: "inToken",
					},
				},
			},
		},
		{
			name: "booleanLiteral",
			pos:  position{line: 436, col: 1, offset: 12258},
			expr: &choiceExpr{
				pos: position{line: 437, col: 5, offset: 12277},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 437, col: 5, offset: 12277},
						run: (*parser).callonbooleanLiteral2,
						expr: &litMatcher{
							pos:        position{line: 437, col: 5, offset: 12277},
							val:        "true",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 438, col: 5, offset: 12345},
						run: (*parser).callonbooleanLiteral4,
						expr: &litMatcher{
							pos:        position{line: 438, col: 5, offset: 12345},
							val:        "false",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "unsetLiteral",
			pos:  position{line: 440, col: 1, offset: 12411},
			expr: &actionExpr{
				pos: position{line: 441, col: 5, offset: 12428},
				run: (*parser).callonunsetLiteral1,
				expr: &litMatcher{
					pos:        position{line: 441, col: 5, offset: 12428},
					val:        "nil",
					ignoreCase: false,
				},
			},
		},
		{
			name: "procList",
			pos:  position{line: 443, col: 1, offset: 12489},
			expr: &actionExpr{
				pos: position{line: 444, col: 5, offset: 12502},
				run: (*parser).callonprocList1,
				expr: &seqExpr{
					pos: position{line: 444, col: 5, offset: 12502},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 444, col: 5, offset: 12502},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 444, col: 11, offset: 12508},
								name: "procChain",
							},
						},
						&labeledExpr{
							pos:   position{line: 444, col: 21, offset: 12518},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 444, col: 26, offset: 12523},
								expr: &ruleRefExpr{
									pos:  position{line: 444, col: 26, offset: 12523},
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
			pos:  position{line: 453, col: 1, offset: 12747},
			expr: &actionExpr{
				pos: position{line: 454, col: 5, offset: 12765},
				run: (*parser).callonparallelChain1,
				expr: &seqExpr{
					pos: position{line: 454, col: 5, offset: 12765},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 454, col: 5, offset: 12765},
							expr: &ruleRefExpr{
								pos:  position{line: 454, col: 5, offset: 12765},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 454, col: 8, offset: 12768},
							val:        ";",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 454, col: 12, offset: 12772},
							expr: &ruleRefExpr{
								pos:  position{line: 454, col: 12, offset: 12772},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 454, col: 15, offset: 12775},
							label: "ch",
							expr: &ruleRefExpr{
								pos:  position{line: 454, col: 18, offset: 12778},
								name: "procChain",
							},
						},
					},
				},
			},
		},
		{
			name: "proc",
			pos:  position{line: 456, col: 1, offset: 12828},
			expr: &choiceExpr{
				pos: position{line: 457, col: 5, offset: 12837},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 457, col: 5, offset: 12837},
						name: "simpleProc",
					},
					&ruleRefExpr{
						pos:  position{line: 458, col: 5, offset: 12852},
						name: "reducerProc",
					},
					&actionExpr{
						pos: position{line: 459, col: 5, offset: 12868},
						run: (*parser).callonproc4,
						expr: &seqExpr{
							pos: position{line: 459, col: 5, offset: 12868},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 459, col: 5, offset: 12868},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 459, col: 9, offset: 12872},
									expr: &ruleRefExpr{
										pos:  position{line: 459, col: 9, offset: 12872},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 459, col: 12, offset: 12875},
									label: "proc",
									expr: &ruleRefExpr{
										pos:  position{line: 459, col: 17, offset: 12880},
										name: "procList",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 459, col: 26, offset: 12889},
									expr: &ruleRefExpr{
										pos:  position{line: 459, col: 26, offset: 12889},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 459, col: 29, offset: 12892},
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
			pos:  position{line: 463, col: 1, offset: 12928},
			expr: &actionExpr{
				pos: position{line: 464, col: 5, offset: 12940},
				run: (*parser).callongroupBy1,
				expr: &seqExpr{
					pos: position{line: 464, col: 5, offset: 12940},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 464, col: 5, offset: 12940},
							val:        "by",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 464, col: 11, offset: 12946},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 464, col: 13, offset: 12948},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 464, col: 18, offset: 12953},
								name: "fieldNameList",
							},
						},
					},
				},
			},
		},
		{
			name: "everyDur",
			pos:  position{line: 466, col: 1, offset: 12989},
			expr: &actionExpr{
				pos: position{line: 467, col: 5, offset: 13002},
				run: (*parser).calloneveryDur1,
				expr: &seqExpr{
					pos: position{line: 467, col: 5, offset: 13002},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 467, col: 5, offset: 13002},
							val:        "every",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 467, col: 14, offset: 13011},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 467, col: 16, offset: 13013},
							label: "dur",
							expr: &ruleRefExpr{
								pos:  position{line: 467, col: 20, offset: 13017},
								name: "duration",
							},
						},
					},
				},
			},
		},
		{
			name: "equalityToken",
			pos:  position{line: 469, col: 1, offset: 13047},
			expr: &choiceExpr{
				pos: position{line: 470, col: 5, offset: 13065},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 470, col: 5, offset: 13065},
						run: (*parser).callonequalityToken2,
						expr: &litMatcher{
							pos:        position{line: 470, col: 5, offset: 13065},
							val:        "=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 471, col: 5, offset: 13095},
						run: (*parser).callonequalityToken4,
						expr: &litMatcher{
							pos:        position{line: 471, col: 5, offset: 13095},
							val:        "!=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 472, col: 5, offset: 13127},
						run: (*parser).callonequalityToken6,
						expr: &litMatcher{
							pos:        position{line: 472, col: 5, offset: 13127},
							val:        "<=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 473, col: 5, offset: 13158},
						run: (*parser).callonequalityToken8,
						expr: &litMatcher{
							pos:        position{line: 473, col: 5, offset: 13158},
							val:        ">=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 474, col: 5, offset: 13189},
						run: (*parser).callonequalityToken10,
						expr: &litMatcher{
							pos:        position{line: 474, col: 5, offset: 13189},
							val:        "<",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 475, col: 5, offset: 13218},
						run: (*parser).callonequalityToken12,
						expr: &litMatcher{
							pos:        position{line: 475, col: 5, offset: 13218},
							val:        ">",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "types",
			pos:  position{line: 477, col: 1, offset: 13244},
			expr: &choiceExpr{
				pos: position{line: 478, col: 5, offset: 13254},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 478, col: 5, offset: 13254},
						val:        "bool",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 479, col: 5, offset: 13265},
						val:        "int",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 480, col: 5, offset: 13275},
						val:        "count",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 481, col: 5, offset: 13287},
						val:        "double",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 482, col: 5, offset: 13300},
						val:        "string",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 483, col: 5, offset: 13313},
						val:        "addr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 484, col: 5, offset: 13324},
						val:        "subnet",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 485, col: 5, offset: 13337},
						val:        "port",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "dash",
			pos:  position{line: 487, col: 1, offset: 13345},
			expr: &choiceExpr{
				pos: position{line: 487, col: 8, offset: 13352},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 487, col: 8, offset: 13352},
						val:        "-",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 487, col: 14, offset: 13358},
						val:        "—",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 487, col: 25, offset: 13369},
						val:        "–",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 487, col: 36, offset: 13380},
						exprs: []interface{}{
							&litMatcher{
								pos:        position{line: 487, col: 36, offset: 13380},
								val:        "to",
								ignoreCase: false,
							},
							&andExpr{
								pos: position{line: 487, col: 40, offset: 13384},
								expr: &ruleRefExpr{
									pos:  position{line: 487, col: 42, offset: 13386},
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
			pos:  position{line: 489, col: 1, offset: 13390},
			expr: &litMatcher{
				pos:        position{line: 489, col: 12, offset: 13401},
				val:        "and",
				ignoreCase: false,
			},
		},
		{
			name: "orToken",
			pos:  position{line: 490, col: 1, offset: 13407},
			expr: &litMatcher{
				pos:        position{line: 490, col: 11, offset: 13417},
				val:        "or",
				ignoreCase: false,
			},
		},
		{
			name: "inToken",
			pos:  position{line: 491, col: 1, offset: 13422},
			expr: &litMatcher{
				pos:        position{line: 491, col: 11, offset: 13432},
				val:        "in",
				ignoreCase: false,
			},
		},
		{
			name: "notToken",
			pos:  position{line: 492, col: 1, offset: 13437},
			expr: &litMatcher{
				pos:        position{line: 492, col: 12, offset: 13448},
				val:        "not",
				ignoreCase: false,
			},
		},
		{
			name: "fieldName",
			pos:  position{line: 494, col: 1, offset: 13455},
			expr: &actionExpr{
				pos: position{line: 494, col: 13, offset: 13467},
				run: (*parser).callonfieldName1,
				expr: &seqExpr{
					pos: position{line: 494, col: 13, offset: 13467},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 494, col: 13, offset: 13467},
							name: "fieldNameStart",
						},
						&zeroOrMoreExpr{
							pos: position{line: 494, col: 28, offset: 13482},
							expr: &ruleRefExpr{
								pos:  position{line: 494, col: 28, offset: 13482},
								name: "fieldNameRest",
							},
						},
					},
				},
			},
		},
		{
			name: "fieldNameStart",
			pos:  position{line: 496, col: 1, offset: 13529},
			expr: &charClassMatcher{
				pos:        position{line: 496, col: 18, offset: 13546},
				val:        "[A-Za-z_$]",
				chars:      []rune{'_', '$'},
				ranges:     []rune{'A', 'Z', 'a', 'z'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "fieldNameRest",
			pos:  position{line: 497, col: 1, offset: 13557},
			expr: &choiceExpr{
				pos: position{line: 497, col: 17, offset: 13573},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 497, col: 17, offset: 13573},
						name: "fieldNameStart",
					},
					&charClassMatcher{
						pos:        position{line: 497, col: 34, offset: 13590},
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
			pos:  position{line: 499, col: 1, offset: 13597},
			expr: &actionExpr{
				pos: position{line: 500, col: 4, offset: 13615},
				run: (*parser).callonfieldReference1,
				expr: &seqExpr{
					pos: position{line: 500, col: 4, offset: 13615},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 500, col: 4, offset: 13615},
							label: "base",
							expr: &ruleRefExpr{
								pos:  position{line: 500, col: 9, offset: 13620},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 500, col: 19, offset: 13630},
							label: "derefs",
							expr: &zeroOrMoreExpr{
								pos: position{line: 500, col: 26, offset: 13637},
								expr: &choiceExpr{
									pos: position{line: 501, col: 8, offset: 13646},
									alternatives: []interface{}{
										&actionExpr{
											pos: position{line: 501, col: 8, offset: 13646},
											run: (*parser).callonfieldReference8,
											expr: &seqExpr{
												pos: position{line: 501, col: 8, offset: 13646},
												exprs: []interface{}{
													&litMatcher{
														pos:        position{line: 501, col: 8, offset: 13646},
														val:        ".",
														ignoreCase: false,
													},
													&labeledExpr{
														pos:   position{line: 501, col: 12, offset: 13650},
														label: "field",
														expr: &ruleRefExpr{
															pos:  position{line: 501, col: 18, offset: 13656},
															name: "fieldName",
														},
													},
												},
											},
										},
										&actionExpr{
											pos: position{line: 502, col: 8, offset: 13737},
											run: (*parser).callonfieldReference13,
											expr: &seqExpr{
												pos: position{line: 502, col: 8, offset: 13737},
												exprs: []interface{}{
													&litMatcher{
														pos:        position{line: 502, col: 8, offset: 13737},
														val:        "[",
														ignoreCase: false,
													},
													&labeledExpr{
														pos:   position{line: 502, col: 12, offset: 13741},
														label: "index",
														expr: &ruleRefExpr{
															pos:  position{line: 502, col: 18, offset: 13747},
															name: "sinteger",
														},
													},
													&litMatcher{
														pos:        position{line: 502, col: 27, offset: 13756},
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
			pos:  position{line: 507, col: 1, offset: 13872},
			expr: &choiceExpr{
				pos: position{line: 508, col: 5, offset: 13886},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 508, col: 5, offset: 13886},
						run: (*parser).callonfieldExpr2,
						expr: &seqExpr{
							pos: position{line: 508, col: 5, offset: 13886},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 508, col: 5, offset: 13886},
									label: "op",
									expr: &ruleRefExpr{
										pos:  position{line: 508, col: 8, offset: 13889},
										name: "fieldOp",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 508, col: 16, offset: 13897},
									expr: &ruleRefExpr{
										pos:  position{line: 508, col: 16, offset: 13897},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 508, col: 19, offset: 13900},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 508, col: 23, offset: 13904},
									expr: &ruleRefExpr{
										pos:  position{line: 508, col: 23, offset: 13904},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 508, col: 26, offset: 13907},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 508, col: 32, offset: 13913},
										name: "fieldReference",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 508, col: 47, offset: 13928},
									expr: &ruleRefExpr{
										pos:  position{line: 508, col: 47, offset: 13928},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 508, col: 50, offset: 13931},
									val:        ")",
									ignoreCase: false,
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 511, col: 5, offset: 13995},
						name: "fieldReference",
					},
				},
			},
		},
		{
			name: "fieldOp",
			pos:  position{line: 513, col: 1, offset: 14011},
			expr: &actionExpr{
				pos: position{line: 514, col: 5, offset: 14023},
				run: (*parser).callonfieldOp1,
				expr: &litMatcher{
					pos:        position{line: 514, col: 5, offset: 14023},
					val:        "len",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldExprList",
			pos:  position{line: 516, col: 1, offset: 14053},
			expr: &actionExpr{
				pos: position{line: 517, col: 5, offset: 14071},
				run: (*parser).callonfieldExprList1,
				expr: &seqExpr{
					pos: position{line: 517, col: 5, offset: 14071},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 517, col: 5, offset: 14071},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 517, col: 11, offset: 14077},
								name: "fieldExpr",
							},
						},
						&labeledExpr{
							pos:   position{line: 517, col: 21, offset: 14087},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 517, col: 26, offset: 14092},
								expr: &seqExpr{
									pos: position{line: 517, col: 27, offset: 14093},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 517, col: 27, offset: 14093},
											expr: &ruleRefExpr{
												pos:  position{line: 517, col: 27, offset: 14093},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 517, col: 30, offset: 14096},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 517, col: 34, offset: 14100},
											expr: &ruleRefExpr{
												pos:  position{line: 517, col: 34, offset: 14100},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 517, col: 37, offset: 14103},
											name: "fieldExpr",
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
			name: "fieldNameList",
			pos:  position{line: 527, col: 1, offset: 14298},
			expr: &actionExpr{
				pos: position{line: 528, col: 5, offset: 14316},
				run: (*parser).callonfieldNameList1,
				expr: &seqExpr{
					pos: position{line: 528, col: 5, offset: 14316},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 528, col: 5, offset: 14316},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 528, col: 11, offset: 14322},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 528, col: 21, offset: 14332},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 528, col: 26, offset: 14337},
								expr: &seqExpr{
									pos: position{line: 528, col: 27, offset: 14338},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 528, col: 27, offset: 14338},
											expr: &ruleRefExpr{
												pos:  position{line: 528, col: 27, offset: 14338},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 528, col: 30, offset: 14341},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 528, col: 34, offset: 14345},
											expr: &ruleRefExpr{
												pos:  position{line: 528, col: 34, offset: 14345},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 528, col: 37, offset: 14348},
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
			pos:  position{line: 536, col: 1, offset: 14541},
			expr: &actionExpr{
				pos: position{line: 537, col: 5, offset: 14553},
				run: (*parser).calloncountOp1,
				expr: &litMatcher{
					pos:        position{line: 537, col: 5, offset: 14553},
					val:        "count",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldReducerOp",
			pos:  position{line: 539, col: 1, offset: 14587},
			expr: &choiceExpr{
				pos: position{line: 540, col: 5, offset: 14606},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 540, col: 5, offset: 14606},
						run: (*parser).callonfieldReducerOp2,
						expr: &litMatcher{
							pos:        position{line: 540, col: 5, offset: 14606},
							val:        "sum",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 541, col: 5, offset: 14640},
						run: (*parser).callonfieldReducerOp4,
						expr: &litMatcher{
							pos:        position{line: 541, col: 5, offset: 14640},
							val:        "avg",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 542, col: 5, offset: 14674},
						run: (*parser).callonfieldReducerOp6,
						expr: &litMatcher{
							pos:        position{line: 542, col: 5, offset: 14674},
							val:        "stdev",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 543, col: 5, offset: 14711},
						run: (*parser).callonfieldReducerOp8,
						expr: &litMatcher{
							pos:        position{line: 543, col: 5, offset: 14711},
							val:        "sd",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 544, col: 5, offset: 14747},
						run: (*parser).callonfieldReducerOp10,
						expr: &litMatcher{
							pos:        position{line: 544, col: 5, offset: 14747},
							val:        "var",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 545, col: 5, offset: 14781},
						run: (*parser).callonfieldReducerOp12,
						expr: &litMatcher{
							pos:        position{line: 545, col: 5, offset: 14781},
							val:        "entropy",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 546, col: 5, offset: 14822},
						run: (*parser).callonfieldReducerOp14,
						expr: &litMatcher{
							pos:        position{line: 546, col: 5, offset: 14822},
							val:        "min",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 547, col: 5, offset: 14856},
						run: (*parser).callonfieldReducerOp16,
						expr: &litMatcher{
							pos:        position{line: 547, col: 5, offset: 14856},
							val:        "max",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 548, col: 5, offset: 14890},
						run: (*parser).callonfieldReducerOp18,
						expr: &litMatcher{
							pos:        position{line: 548, col: 5, offset: 14890},
							val:        "first",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 549, col: 5, offset: 14928},
						run: (*parser).callonfieldReducerOp20,
						expr: &litMatcher{
							pos:        position{line: 549, col: 5, offset: 14928},
							val:        "last",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 550, col: 5, offset: 14964},
						run: (*parser).callonfieldReducerOp22,
						expr: &litMatcher{
							pos:        position{line: 550, col: 5, offset: 14964},
							val:        "countdistinct",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "paddedFieldName",
			pos:  position{line: 552, col: 1, offset: 15014},
			expr: &actionExpr{
				pos: position{line: 552, col: 19, offset: 15032},
				run: (*parser).callonpaddedFieldName1,
				expr: &seqExpr{
					pos: position{line: 552, col: 19, offset: 15032},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 552, col: 19, offset: 15032},
							expr: &ruleRefExpr{
								pos:  position{line: 552, col: 19, offset: 15032},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 552, col: 22, offset: 15035},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 552, col: 28, offset: 15041},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 552, col: 38, offset: 15051},
							expr: &ruleRefExpr{
								pos:  position{line: 552, col: 38, offset: 15051},
								name: "_",
							},
						},
					},
				},
			},
		},
		{
			name: "countReducer",
			pos:  position{line: 554, col: 1, offset: 15077},
			expr: &actionExpr{
				pos: position{line: 555, col: 5, offset: 15094},
				run: (*parser).calloncountReducer1,
				expr: &seqExpr{
					pos: position{line: 555, col: 5, offset: 15094},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 555, col: 5, offset: 15094},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 555, col: 8, offset: 15097},
								name: "countOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 555, col: 16, offset: 15105},
							expr: &ruleRefExpr{
								pos:  position{line: 555, col: 16, offset: 15105},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 555, col: 19, offset: 15108},
							val:        "(",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 555, col: 23, offset: 15112},
							label: "field",
							expr: &zeroOrOneExpr{
								pos: position{line: 555, col: 29, offset: 15118},
								expr: &ruleRefExpr{
									pos:  position{line: 555, col: 29, offset: 15118},
									name: "paddedFieldName",
								},
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 555, col: 47, offset: 15136},
							expr: &ruleRefExpr{
								pos:  position{line: 555, col: 47, offset: 15136},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 555, col: 50, offset: 15139},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "fieldReducer",
			pos:  position{line: 559, col: 1, offset: 15198},
			expr: &actionExpr{
				pos: position{line: 560, col: 5, offset: 15215},
				run: (*parser).callonfieldReducer1,
				expr: &seqExpr{
					pos: position{line: 560, col: 5, offset: 15215},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 560, col: 5, offset: 15215},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 560, col: 8, offset: 15218},
								name: "fieldReducerOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 560, col: 23, offset: 15233},
							expr: &ruleRefExpr{
								pos:  position{line: 560, col: 23, offset: 15233},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 560, col: 26, offset: 15236},
							val:        "(",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 560, col: 30, offset: 15240},
							expr: &ruleRefExpr{
								pos:  position{line: 560, col: 30, offset: 15240},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 560, col: 33, offset: 15243},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 560, col: 39, offset: 15249},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 560, col: 50, offset: 15260},
							expr: &ruleRefExpr{
								pos:  position{line: 560, col: 50, offset: 15260},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 560, col: 53, offset: 15263},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reducerProc",
			pos:  position{line: 564, col: 1, offset: 15330},
			expr: &actionExpr{
				pos: position{line: 565, col: 5, offset: 15346},
				run: (*parser).callonreducerProc1,
				expr: &seqExpr{
					pos: position{line: 565, col: 5, offset: 15346},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 565, col: 5, offset: 15346},
							label: "every",
							expr: &zeroOrOneExpr{
								pos: position{line: 565, col: 11, offset: 15352},
								expr: &seqExpr{
									pos: position{line: 565, col: 12, offset: 15353},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 565, col: 12, offset: 15353},
											name: "everyDur",
										},
										&ruleRefExpr{
											pos:  position{line: 565, col: 21, offset: 15362},
											name: "_",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 565, col: 25, offset: 15366},
							label: "reducers",
							expr: &ruleRefExpr{
								pos:  position{line: 565, col: 34, offset: 15375},
								name: "reducerList",
							},
						},
						&labeledExpr{
							pos:   position{line: 565, col: 46, offset: 15387},
							label: "keys",
							expr: &zeroOrOneExpr{
								pos: position{line: 565, col: 51, offset: 15392},
								expr: &seqExpr{
									pos: position{line: 565, col: 52, offset: 15393},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 565, col: 52, offset: 15393},
											name: "_",
										},
										&ruleRefExpr{
											pos:  position{line: 565, col: 54, offset: 15395},
											name: "groupBy",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 565, col: 64, offset: 15405},
							label: "limit",
							expr: &zeroOrOneExpr{
								pos: position{line: 565, col: 70, offset: 15411},
								expr: &ruleRefExpr{
									pos:  position{line: 565, col: 70, offset: 15411},
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
			pos:  position{line: 583, col: 1, offset: 15768},
			expr: &actionExpr{
				pos: position{line: 584, col: 5, offset: 15781},
				run: (*parser).callonasClause1,
				expr: &seqExpr{
					pos: position{line: 584, col: 5, offset: 15781},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 584, col: 5, offset: 15781},
							val:        "as",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 584, col: 11, offset: 15787},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 584, col: 13, offset: 15789},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 584, col: 15, offset: 15791},
								name: "fieldName",
							},
						},
					},
				},
			},
		},
		{
			name: "reducerExpr",
			pos:  position{line: 586, col: 1, offset: 15820},
			expr: &choiceExpr{
				pos: position{line: 587, col: 5, offset: 15836},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 587, col: 5, offset: 15836},
						run: (*parser).callonreducerExpr2,
						expr: &seqExpr{
							pos: position{line: 587, col: 5, offset: 15836},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 587, col: 5, offset: 15836},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 587, col: 11, offset: 15842},
										name: "fieldName",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 587, col: 21, offset: 15852},
									expr: &ruleRefExpr{
										pos:  position{line: 587, col: 21, offset: 15852},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 587, col: 24, offset: 15855},
									val:        "=",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 587, col: 28, offset: 15859},
									expr: &ruleRefExpr{
										pos:  position{line: 587, col: 28, offset: 15859},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 587, col: 31, offset: 15862},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 587, col: 33, offset: 15864},
										name: "reducer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 590, col: 5, offset: 15927},
						run: (*parser).callonreducerExpr13,
						expr: &seqExpr{
							pos: position{line: 590, col: 5, offset: 15927},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 590, col: 5, offset: 15927},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 590, col: 7, offset: 15929},
										name: "reducer",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 590, col: 15, offset: 15937},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 590, col: 17, offset: 15939},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 590, col: 23, offset: 15945},
										name: "asClause",
									},
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 593, col: 5, offset: 16009},
						name: "reducer",
					},
				},
			},
		},
		{
			name: "reducer",
			pos:  position{line: 595, col: 1, offset: 16018},
			expr: &choiceExpr{
				pos: position{line: 596, col: 5, offset: 16030},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 596, col: 5, offset: 16030},
						name: "countReducer",
					},
					&ruleRefExpr{
						pos:  position{line: 597, col: 5, offset: 16047},
						name: "fieldReducer",
					},
				},
			},
		},
		{
			name: "reducerList",
			pos:  position{line: 599, col: 1, offset: 16061},
			expr: &actionExpr{
				pos: position{line: 600, col: 5, offset: 16077},
				run: (*parser).callonreducerList1,
				expr: &seqExpr{
					pos: position{line: 600, col: 5, offset: 16077},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 600, col: 5, offset: 16077},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 600, col: 11, offset: 16083},
								name: "reducerExpr",
							},
						},
						&labeledExpr{
							pos:   position{line: 600, col: 23, offset: 16095},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 600, col: 28, offset: 16100},
								expr: &seqExpr{
									pos: position{line: 600, col: 29, offset: 16101},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 600, col: 29, offset: 16101},
											expr: &ruleRefExpr{
												pos:  position{line: 600, col: 29, offset: 16101},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 600, col: 32, offset: 16104},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 600, col: 36, offset: 16108},
											expr: &ruleRefExpr{
												pos:  position{line: 600, col: 36, offset: 16108},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 600, col: 39, offset: 16111},
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
			pos:  position{line: 608, col: 1, offset: 16308},
			expr: &choiceExpr{
				pos: position{line: 609, col: 5, offset: 16323},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 609, col: 5, offset: 16323},
						name: "sort",
					},
					&ruleRefExpr{
						pos:  position{line: 610, col: 5, offset: 16332},
						name: "top",
					},
					&ruleRefExpr{
						pos:  position{line: 611, col: 5, offset: 16340},
						name: "cut",
					},
					&ruleRefExpr{
						pos:  position{line: 612, col: 5, offset: 16348},
						name: "head",
					},
					&ruleRefExpr{
						pos:  position{line: 613, col: 5, offset: 16357},
						name: "tail",
					},
					&ruleRefExpr{
						pos:  position{line: 614, col: 5, offset: 16366},
						name: "filter",
					},
					&ruleRefExpr{
						pos:  position{line: 615, col: 5, offset: 16377},
						name: "uniq",
					},
				},
			},
		},
		{
			name: "sort",
			pos:  position{line: 617, col: 1, offset: 16383},
			expr: &choiceExpr{
				pos: position{line: 618, col: 5, offset: 16392},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 618, col: 5, offset: 16392},
						run: (*parser).callonsort2,
						expr: &seqExpr{
							pos: position{line: 618, col: 5, offset: 16392},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 618, col: 5, offset: 16392},
									val:        "sort",
									ignoreCase: true,
								},
								&labeledExpr{
									pos:   position{line: 618, col: 13, offset: 16400},
									label: "rev",
									expr: &zeroOrOneExpr{
										pos: position{line: 618, col: 17, offset: 16404},
										expr: &seqExpr{
											pos: position{line: 618, col: 18, offset: 16405},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 618, col: 18, offset: 16405},
													name: "_",
												},
												&litMatcher{
													pos:        position{line: 618, col: 20, offset: 16407},
													val:        "-r",
													ignoreCase: false,
												},
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 618, col: 27, offset: 16414},
									label: "limit",
									expr: &zeroOrOneExpr{
										pos: position{line: 618, col: 33, offset: 16420},
										expr: &ruleRefExpr{
											pos:  position{line: 618, col: 33, offset: 16420},
											name: "procLimitArg",
										},
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 618, col: 48, offset: 16435},
									expr: &ruleRefExpr{
										pos:  position{line: 618, col: 48, offset: 16435},
										name: "_",
									},
								},
								&notExpr{
									pos: position{line: 618, col: 51, offset: 16438},
									expr: &litMatcher{
										pos:        position{line: 618, col: 52, offset: 16439},
										val:        "-r",
										ignoreCase: false,
									},
								},
								&labeledExpr{
									pos:   position{line: 618, col: 57, offset: 16444},
									label: "list",
									expr: &zeroOrOneExpr{
										pos: position{line: 618, col: 62, offset: 16449},
										expr: &ruleRefExpr{
											pos:  position{line: 618, col: 63, offset: 16450},
											name: "fieldExprList",
										},
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 623, col: 5, offset: 16580},
						run: (*parser).callonsort20,
						expr: &seqExpr{
							pos: position{line: 623, col: 5, offset: 16580},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 623, col: 5, offset: 16580},
									val:        "sort",
									ignoreCase: true,
								},
								&labeledExpr{
									pos:   position{line: 623, col: 13, offset: 16588},
									label: "limit",
									expr: &zeroOrOneExpr{
										pos: position{line: 623, col: 19, offset: 16594},
										expr: &ruleRefExpr{
											pos:  position{line: 623, col: 19, offset: 16594},
											name: "procLimitArg",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 623, col: 33, offset: 16608},
									label: "rev",
									expr: &zeroOrOneExpr{
										pos: position{line: 623, col: 37, offset: 16612},
										expr: &seqExpr{
											pos: position{line: 623, col: 38, offset: 16613},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 623, col: 38, offset: 16613},
													name: "_",
												},
												&litMatcher{
													pos:        position{line: 623, col: 40, offset: 16615},
													val:        "-r",
													ignoreCase: false,
												},
											},
										},
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 623, col: 47, offset: 16622},
									expr: &ruleRefExpr{
										pos:  position{line: 623, col: 47, offset: 16622},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 623, col: 50, offset: 16625},
									label: "list",
									expr: &zeroOrOneExpr{
										pos: position{line: 623, col: 55, offset: 16630},
										expr: &ruleRefExpr{
											pos:  position{line: 623, col: 56, offset: 16631},
											name: "fieldExprList",
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
			pos:  position{line: 629, col: 1, offset: 16758},
			expr: &actionExpr{
				pos: position{line: 630, col: 5, offset: 16766},
				run: (*parser).callontop1,
				expr: &seqExpr{
					pos: position{line: 630, col: 5, offset: 16766},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 630, col: 5, offset: 16766},
							val:        "top",
							ignoreCase: true,
						},
						&labeledExpr{
							pos:   position{line: 630, col: 12, offset: 16773},
							label: "limit",
							expr: &zeroOrOneExpr{
								pos: position{line: 630, col: 18, offset: 16779},
								expr: &ruleRefExpr{
									pos:  position{line: 630, col: 18, offset: 16779},
									name: "procLimitArg",
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 630, col: 32, offset: 16793},
							label: "flush",
							expr: &zeroOrOneExpr{
								pos: position{line: 630, col: 38, offset: 16799},
								expr: &seqExpr{
									pos: position{line: 630, col: 39, offset: 16800},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 630, col: 39, offset: 16800},
											name: "_",
										},
										&litMatcher{
											pos:        position{line: 630, col: 41, offset: 16802},
											val:        "-flush",
											ignoreCase: false,
										},
									},
								},
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 630, col: 52, offset: 16813},
							expr: &ruleRefExpr{
								pos:  position{line: 630, col: 52, offset: 16813},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 630, col: 55, offset: 16816},
							label: "list",
							expr: &zeroOrOneExpr{
								pos: position{line: 630, col: 60, offset: 16821},
								expr: &ruleRefExpr{
									pos:  position{line: 630, col: 61, offset: 16822},
									name: "fieldExprList",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "procLimitArg",
			pos:  position{line: 634, col: 1, offset: 16893},
			expr: &actionExpr{
				pos: position{line: 635, col: 5, offset: 16910},
				run: (*parser).callonprocLimitArg1,
				expr: &seqExpr{
					pos: position{line: 635, col: 5, offset: 16910},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 635, col: 5, offset: 16910},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 635, col: 7, offset: 16912},
							val:        "-limit",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 635, col: 16, offset: 16921},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 635, col: 18, offset: 16923},
							label: "limit",
							expr: &ruleRefExpr{
								pos:  position{line: 635, col: 24, offset: 16929},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "cut",
			pos:  position{line: 637, col: 1, offset: 16960},
			expr: &actionExpr{
				pos: position{line: 638, col: 5, offset: 16968},
				run: (*parser).calloncut1,
				expr: &seqExpr{
					pos: position{line: 638, col: 5, offset: 16968},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 638, col: 5, offset: 16968},
							val:        "cut",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 638, col: 12, offset: 16975},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 638, col: 14, offset: 16977},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 638, col: 19, offset: 16982},
								name: "fieldNameList",
							},
						},
					},
				},
			},
		},
		{
			name: "head",
			pos:  position{line: 639, col: 1, offset: 17030},
			expr: &choiceExpr{
				pos: position{line: 640, col: 5, offset: 17039},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 640, col: 5, offset: 17039},
						run: (*parser).callonhead2,
						expr: &seqExpr{
							pos: position{line: 640, col: 5, offset: 17039},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 640, col: 5, offset: 17039},
									val:        "head",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 640, col: 13, offset: 17047},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 640, col: 15, offset: 17049},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 640, col: 21, offset: 17055},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 641, col: 5, offset: 17103},
						run: (*parser).callonhead8,
						expr: &litMatcher{
							pos:        position{line: 641, col: 5, offset: 17103},
							val:        "head",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "tail",
			pos:  position{line: 642, col: 1, offset: 17143},
			expr: &choiceExpr{
				pos: position{line: 643, col: 5, offset: 17152},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 643, col: 5, offset: 17152},
						run: (*parser).callontail2,
						expr: &seqExpr{
							pos: position{line: 643, col: 5, offset: 17152},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 643, col: 5, offset: 17152},
									val:        "tail",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 643, col: 13, offset: 17160},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 643, col: 15, offset: 17162},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 643, col: 21, offset: 17168},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 644, col: 5, offset: 17216},
						run: (*parser).callontail8,
						expr: &litMatcher{
							pos:        position{line: 644, col: 5, offset: 17216},
							val:        "tail",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "filter",
			pos:  position{line: 646, col: 1, offset: 17257},
			expr: &actionExpr{
				pos: position{line: 647, col: 5, offset: 17268},
				run: (*parser).callonfilter1,
				expr: &seqExpr{
					pos: position{line: 647, col: 5, offset: 17268},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 647, col: 5, offset: 17268},
							val:        "filter",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 647, col: 15, offset: 17278},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 647, col: 17, offset: 17280},
							label: "expr",
							expr: &ruleRefExpr{
								pos:  position{line: 647, col: 22, offset: 17285},
								name: "searchExpr",
							},
						},
					},
				},
			},
		},
		{
			name: "uniq",
			pos:  position{line: 650, col: 1, offset: 17343},
			expr: &choiceExpr{
				pos: position{line: 651, col: 5, offset: 17352},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 651, col: 5, offset: 17352},
						run: (*parser).callonuniq2,
						expr: &seqExpr{
							pos: position{line: 651, col: 5, offset: 17352},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 651, col: 5, offset: 17352},
									val:        "uniq",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 651, col: 13, offset: 17360},
									name: "_",
								},
								&litMatcher{
									pos:        position{line: 651, col: 15, offset: 17362},
									val:        "-c",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 654, col: 5, offset: 17416},
						run: (*parser).callonuniq7,
						expr: &litMatcher{
							pos:        position{line: 654, col: 5, offset: 17416},
							val:        "uniq",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "duration",
			pos:  position{line: 658, col: 1, offset: 17471},
			expr: &choiceExpr{
				pos: position{line: 659, col: 5, offset: 17484},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 659, col: 5, offset: 17484},
						name: "seconds",
					},
					&ruleRefExpr{
						pos:  position{line: 660, col: 5, offset: 17496},
						name: "minutes",
					},
					&ruleRefExpr{
						pos:  position{line: 661, col: 5, offset: 17508},
						name: "hours",
					},
					&seqExpr{
						pos: position{line: 662, col: 5, offset: 17518},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 662, col: 5, offset: 17518},
								name: "hours",
							},
							&ruleRefExpr{
								pos:  position{line: 662, col: 11, offset: 17524},
								name: "_",
							},
							&litMatcher{
								pos:        position{line: 662, col: 13, offset: 17526},
								val:        "and",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 662, col: 19, offset: 17532},
								name: "_",
							},
							&ruleRefExpr{
								pos:  position{line: 662, col: 21, offset: 17534},
								name: "minutes",
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 663, col: 5, offset: 17546},
						name: "days",
					},
					&ruleRefExpr{
						pos:  position{line: 664, col: 5, offset: 17555},
						name: "weeks",
					},
				},
			},
		},
		{
			name: "sec_abbrev",
			pos:  position{line: 666, col: 1, offset: 17562},
			expr: &choiceExpr{
				pos: position{line: 667, col: 5, offset: 17577},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 667, col: 5, offset: 17577},
						val:        "seconds",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 668, col: 5, offset: 17591},
						val:        "second",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 669, col: 5, offset: 17604},
						val:        "secs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 670, col: 5, offset: 17615},
						val:        "sec",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 671, col: 5, offset: 17625},
						val:        "s",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "min_abbrev",
			pos:  position{line: 673, col: 1, offset: 17630},
			expr: &choiceExpr{
				pos: position{line: 674, col: 5, offset: 17645},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 674, col: 5, offset: 17645},
						val:        "minutes",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 675, col: 5, offset: 17659},
						val:        "minute",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 676, col: 5, offset: 17672},
						val:        "mins",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 677, col: 5, offset: 17683},
						val:        "min",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 678, col: 5, offset: 17693},
						val:        "m",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "hour_abbrev",
			pos:  position{line: 680, col: 1, offset: 17698},
			expr: &choiceExpr{
				pos: position{line: 681, col: 5, offset: 17714},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 681, col: 5, offset: 17714},
						val:        "hours",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 682, col: 5, offset: 17726},
						val:        "hrs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 683, col: 5, offset: 17736},
						val:        "hr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 684, col: 5, offset: 17745},
						val:        "h",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 685, col: 5, offset: 17753},
						val:        "hour",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "day_abbrev",
			pos:  position{line: 687, col: 1, offset: 17761},
			expr: &choiceExpr{
				pos: position{line: 687, col: 14, offset: 17774},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 687, col: 14, offset: 17774},
						val:        "days",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 687, col: 21, offset: 17781},
						val:        "day",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 687, col: 27, offset: 17787},
						val:        "d",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "week_abbrev",
			pos:  position{line: 688, col: 1, offset: 17791},
			expr: &choiceExpr{
				pos: position{line: 688, col: 15, offset: 17805},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 688, col: 15, offset: 17805},
						val:        "weeks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 688, col: 23, offset: 17813},
						val:        "week",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 688, col: 30, offset: 17820},
						val:        "wks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 688, col: 36, offset: 17826},
						val:        "wk",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 688, col: 41, offset: 17831},
						val:        "w",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "seconds",
			pos:  position{line: 690, col: 1, offset: 17836},
			expr: &choiceExpr{
				pos: position{line: 691, col: 5, offset: 17848},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 691, col: 5, offset: 17848},
						run: (*parser).callonseconds2,
						expr: &litMatcher{
							pos:        position{line: 691, col: 5, offset: 17848},
							val:        "second",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 692, col: 5, offset: 17893},
						run: (*parser).callonseconds4,
						expr: &seqExpr{
							pos: position{line: 692, col: 5, offset: 17893},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 692, col: 5, offset: 17893},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 692, col: 9, offset: 17897},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 692, col: 16, offset: 17904},
									expr: &ruleRefExpr{
										pos:  position{line: 692, col: 16, offset: 17904},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 692, col: 19, offset: 17907},
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
			pos:  position{line: 694, col: 1, offset: 17953},
			expr: &choiceExpr{
				pos: position{line: 695, col: 5, offset: 17965},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 695, col: 5, offset: 17965},
						run: (*parser).callonminutes2,
						expr: &litMatcher{
							pos:        position{line: 695, col: 5, offset: 17965},
							val:        "minute",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 696, col: 5, offset: 18011},
						run: (*parser).callonminutes4,
						expr: &seqExpr{
							pos: position{line: 696, col: 5, offset: 18011},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 696, col: 5, offset: 18011},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 696, col: 9, offset: 18015},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 696, col: 16, offset: 18022},
									expr: &ruleRefExpr{
										pos:  position{line: 696, col: 16, offset: 18022},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 696, col: 19, offset: 18025},
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
			pos:  position{line: 698, col: 1, offset: 18080},
			expr: &choiceExpr{
				pos: position{line: 699, col: 5, offset: 18090},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 699, col: 5, offset: 18090},
						run: (*parser).callonhours2,
						expr: &litMatcher{
							pos:        position{line: 699, col: 5, offset: 18090},
							val:        "hour",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 700, col: 5, offset: 18136},
						run: (*parser).callonhours4,
						expr: &seqExpr{
							pos: position{line: 700, col: 5, offset: 18136},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 700, col: 5, offset: 18136},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 700, col: 9, offset: 18140},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 700, col: 16, offset: 18147},
									expr: &ruleRefExpr{
										pos:  position{line: 700, col: 16, offset: 18147},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 700, col: 19, offset: 18150},
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
			pos:  position{line: 702, col: 1, offset: 18208},
			expr: &choiceExpr{
				pos: position{line: 703, col: 5, offset: 18217},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 703, col: 5, offset: 18217},
						run: (*parser).callondays2,
						expr: &litMatcher{
							pos:        position{line: 703, col: 5, offset: 18217},
							val:        "day",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 704, col: 5, offset: 18265},
						run: (*parser).callondays4,
						expr: &seqExpr{
							pos: position{line: 704, col: 5, offset: 18265},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 704, col: 5, offset: 18265},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 704, col: 9, offset: 18269},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 704, col: 16, offset: 18276},
									expr: &ruleRefExpr{
										pos:  position{line: 704, col: 16, offset: 18276},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 704, col: 19, offset: 18279},
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
			pos:  position{line: 706, col: 1, offset: 18339},
			expr: &actionExpr{
				pos: position{line: 707, col: 5, offset: 18349},
				run: (*parser).callonweeks1,
				expr: &seqExpr{
					pos: position{line: 707, col: 5, offset: 18349},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 707, col: 5, offset: 18349},
							label: "num",
							expr: &ruleRefExpr{
								pos:  position{line: 707, col: 9, offset: 18353},
								name: "number",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 707, col: 16, offset: 18360},
							expr: &ruleRefExpr{
								pos:  position{line: 707, col: 16, offset: 18360},
								name: "_",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 707, col: 19, offset: 18363},
							name: "week_abbrev",
						},
					},
				},
			},
		},
		{
			name: "number",
			pos:  position{line: 709, col: 1, offset: 18426},
			expr: &ruleRefExpr{
				pos:  position{line: 709, col: 10, offset: 18435},
				name: "integer",
			},
		},
		{
			name: "addr",
			pos:  position{line: 713, col: 1, offset: 18473},
			expr: &actionExpr{
				pos: position{line: 714, col: 5, offset: 18482},
				run: (*parser).callonaddr1,
				expr: &labeledExpr{
					pos:   position{line: 714, col: 5, offset: 18482},
					label: "a",
					expr: &seqExpr{
						pos: position{line: 714, col: 8, offset: 18485},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 714, col: 8, offset: 18485},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 714, col: 16, offset: 18493},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 714, col: 20, offset: 18497},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 714, col: 28, offset: 18505},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 714, col: 32, offset: 18509},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 714, col: 40, offset: 18517},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 714, col: 44, offset: 18521},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "port",
			pos:  position{line: 716, col: 1, offset: 18562},
			expr: &actionExpr{
				pos: position{line: 717, col: 5, offset: 18571},
				run: (*parser).callonport1,
				expr: &seqExpr{
					pos: position{line: 717, col: 5, offset: 18571},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 717, col: 5, offset: 18571},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 717, col: 9, offset: 18575},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 717, col: 11, offset: 18577},
								name: "sinteger",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6addr",
			pos:  position{line: 721, col: 1, offset: 18736},
			expr: &choiceExpr{
				pos: position{line: 722, col: 5, offset: 18748},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 722, col: 5, offset: 18748},
						run: (*parser).callonip6addr2,
						expr: &seqExpr{
							pos: position{line: 722, col: 5, offset: 18748},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 722, col: 5, offset: 18748},
									label: "a",
									expr: &oneOrMoreExpr{
										pos: position{line: 722, col: 7, offset: 18750},
										expr: &ruleRefExpr{
											pos:  position{line: 722, col: 8, offset: 18751},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 722, col: 20, offset: 18763},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 722, col: 22, offset: 18765},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 725, col: 5, offset: 18829},
						run: (*parser).callonip6addr9,
						expr: &seqExpr{
							pos: position{line: 725, col: 5, offset: 18829},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 725, col: 5, offset: 18829},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 725, col: 7, offset: 18831},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 725, col: 11, offset: 18835},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 725, col: 13, offset: 18837},
										expr: &ruleRefExpr{
											pos:  position{line: 725, col: 14, offset: 18838},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 725, col: 25, offset: 18849},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 725, col: 30, offset: 18854},
									label: "d",
									expr: &zeroOrMoreExpr{
										pos: position{line: 725, col: 32, offset: 18856},
										expr: &ruleRefExpr{
											pos:  position{line: 725, col: 33, offset: 18857},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 725, col: 45, offset: 18869},
									label: "e",
									expr: &ruleRefExpr{
										pos:  position{line: 725, col: 47, offset: 18871},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 728, col: 5, offset: 18970},
						run: (*parser).callonip6addr22,
						expr: &seqExpr{
							pos: position{line: 728, col: 5, offset: 18970},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 728, col: 5, offset: 18970},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 728, col: 10, offset: 18975},
									label: "a",
									expr: &zeroOrMoreExpr{
										pos: position{line: 728, col: 12, offset: 18977},
										expr: &ruleRefExpr{
											pos:  position{line: 728, col: 13, offset: 18978},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 728, col: 25, offset: 18990},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 728, col: 27, offset: 18992},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 731, col: 5, offset: 19063},
						run: (*parser).callonip6addr30,
						expr: &seqExpr{
							pos: position{line: 731, col: 5, offset: 19063},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 731, col: 5, offset: 19063},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 731, col: 7, offset: 19065},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 731, col: 11, offset: 19069},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 731, col: 13, offset: 19071},
										expr: &ruleRefExpr{
											pos:  position{line: 731, col: 14, offset: 19072},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 731, col: 25, offset: 19083},
									val:        "::",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 734, col: 5, offset: 19151},
						run: (*parser).callonip6addr38,
						expr: &litMatcher{
							pos:        position{line: 734, col: 5, offset: 19151},
							val:        "::",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "ip6tail",
			pos:  position{line: 738, col: 1, offset: 19188},
			expr: &choiceExpr{
				pos: position{line: 739, col: 5, offset: 19200},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 739, col: 5, offset: 19200},
						name: "addr",
					},
					&ruleRefExpr{
						pos:  position{line: 740, col: 5, offset: 19209},
						name: "h16",
					},
				},
			},
		},
		{
			name: "h_append",
			pos:  position{line: 742, col: 1, offset: 19214},
			expr: &actionExpr{
				pos: position{line: 742, col: 12, offset: 19225},
				run: (*parser).callonh_append1,
				expr: &seqExpr{
					pos: position{line: 742, col: 12, offset: 19225},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 742, col: 12, offset: 19225},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 742, col: 16, offset: 19229},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 742, col: 18, offset: 19231},
								name: "h16",
							},
						},
					},
				},
			},
		},
		{
			name: "h_prepend",
			pos:  position{line: 743, col: 1, offset: 19268},
			expr: &actionExpr{
				pos: position{line: 743, col: 13, offset: 19280},
				run: (*parser).callonh_prepend1,
				expr: &seqExpr{
					pos: position{line: 743, col: 13, offset: 19280},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 743, col: 13, offset: 19280},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 743, col: 15, offset: 19282},
								name: "h16",
							},
						},
						&litMatcher{
							pos:        position{line: 743, col: 19, offset: 19286},
							val:        ":",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "sub_addr",
			pos:  position{line: 745, col: 1, offset: 19324},
			expr: &choiceExpr{
				pos: position{line: 746, col: 5, offset: 19337},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 746, col: 5, offset: 19337},
						name: "addr",
					},
					&actionExpr{
						pos: position{line: 747, col: 5, offset: 19346},
						run: (*parser).callonsub_addr3,
						expr: &labeledExpr{
							pos:   position{line: 747, col: 5, offset: 19346},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 747, col: 8, offset: 19349},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 747, col: 8, offset: 19349},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 747, col: 16, offset: 19357},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 747, col: 20, offset: 19361},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 747, col: 28, offset: 19369},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 747, col: 32, offset: 19373},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 748, col: 5, offset: 19425},
						run: (*parser).callonsub_addr11,
						expr: &labeledExpr{
							pos:   position{line: 748, col: 5, offset: 19425},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 748, col: 8, offset: 19428},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 748, col: 8, offset: 19428},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 748, col: 16, offset: 19436},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 748, col: 20, offset: 19440},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 749, col: 5, offset: 19494},
						run: (*parser).callonsub_addr17,
						expr: &labeledExpr{
							pos:   position{line: 749, col: 5, offset: 19494},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 749, col: 7, offset: 19496},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "subnet",
			pos:  position{line: 751, col: 1, offset: 19547},
			expr: &actionExpr{
				pos: position{line: 752, col: 5, offset: 19558},
				run: (*parser).callonsubnet1,
				expr: &seqExpr{
					pos: position{line: 752, col: 5, offset: 19558},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 752, col: 5, offset: 19558},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 752, col: 7, offset: 19560},
								name: "sub_addr",
							},
						},
						&litMatcher{
							pos:        position{line: 752, col: 16, offset: 19569},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 752, col: 20, offset: 19573},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 752, col: 22, offset: 19575},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6subnet",
			pos:  position{line: 756, col: 1, offset: 19651},
			expr: &actionExpr{
				pos: position{line: 757, col: 5, offset: 19665},
				run: (*parser).callonip6subnet1,
				expr: &seqExpr{
					pos: position{line: 757, col: 5, offset: 19665},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 757, col: 5, offset: 19665},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 757, col: 7, offset: 19667},
								name: "ip6addr",
							},
						},
						&litMatcher{
							pos:        position{line: 757, col: 15, offset: 19675},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 757, col: 19, offset: 19679},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 757, col: 21, offset: 19681},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "integer",
			pos:  position{line: 761, col: 1, offset: 19747},
			expr: &actionExpr{
				pos: position{line: 762, col: 5, offset: 19759},
				run: (*parser).calloninteger1,
				expr: &labeledExpr{
					pos:   position{line: 762, col: 5, offset: 19759},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 762, col: 7, offset: 19761},
						name: "sinteger",
					},
				},
			},
		},
		{
			name: "sinteger",
			pos:  position{line: 766, col: 1, offset: 19805},
			expr: &actionExpr{
				pos: position{line: 767, col: 5, offset: 19818},
				run: (*parser).callonsinteger1,
				expr: &labeledExpr{
					pos:   position{line: 767, col: 5, offset: 19818},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 767, col: 11, offset: 19824},
						expr: &charClassMatcher{
							pos:        position{line: 767, col: 11, offset: 19824},
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
			pos:  position{line: 771, col: 1, offset: 19869},
			expr: &actionExpr{
				pos: position{line: 772, col: 5, offset: 19880},
				run: (*parser).callondouble1,
				expr: &labeledExpr{
					pos:   position{line: 772, col: 5, offset: 19880},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 772, col: 7, offset: 19882},
						name: "sdouble",
					},
				},
			},
		},
		{
			name: "sdouble",
			pos:  position{line: 776, col: 1, offset: 19929},
			expr: &choiceExpr{
				pos: position{line: 777, col: 5, offset: 19941},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 777, col: 5, offset: 19941},
						run: (*parser).callonsdouble2,
						expr: &seqExpr{
							pos: position{line: 777, col: 5, offset: 19941},
							exprs: []interface{}{
								&oneOrMoreExpr{
									pos: position{line: 777, col: 5, offset: 19941},
									expr: &ruleRefExpr{
										pos:  position{line: 777, col: 5, offset: 19941},
										name: "doubleInteger",
									},
								},
								&litMatcher{
									pos:        position{line: 777, col: 20, offset: 19956},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 777, col: 24, offset: 19960},
									expr: &ruleRefExpr{
										pos:  position{line: 777, col: 24, offset: 19960},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 777, col: 37, offset: 19973},
									expr: &ruleRefExpr{
										pos:  position{line: 777, col: 37, offset: 19973},
										name: "exponentPart",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 780, col: 5, offset: 20032},
						run: (*parser).callonsdouble11,
						expr: &seqExpr{
							pos: position{line: 780, col: 5, offset: 20032},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 780, col: 5, offset: 20032},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 780, col: 9, offset: 20036},
									expr: &ruleRefExpr{
										pos:  position{line: 780, col: 9, offset: 20036},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 780, col: 22, offset: 20049},
									expr: &ruleRefExpr{
										pos:  position{line: 780, col: 22, offset: 20049},
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
			pos:  position{line: 784, col: 1, offset: 20105},
			expr: &choiceExpr{
				pos: position{line: 785, col: 5, offset: 20123},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 785, col: 5, offset: 20123},
						val:        "0",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 786, col: 5, offset: 20131},
						exprs: []interface{}{
							&charClassMatcher{
								pos:        position{line: 786, col: 5, offset: 20131},
								val:        "[1-9]",
								ranges:     []rune{'1', '9'},
								ignoreCase: false,
								inverted:   false,
							},
							&zeroOrMoreExpr{
								pos: position{line: 786, col: 11, offset: 20137},
								expr: &charClassMatcher{
									pos:        position{line: 786, col: 11, offset: 20137},
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
			pos:  position{line: 788, col: 1, offset: 20145},
			expr: &charClassMatcher{
				pos:        position{line: 788, col: 15, offset: 20159},
				val:        "[0-9]",
				ranges:     []rune{'0', '9'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "signedInteger",
			pos:  position{line: 790, col: 1, offset: 20166},
			expr: &seqExpr{
				pos: position{line: 790, col: 17, offset: 20182},
				exprs: []interface{}{
					&zeroOrOneExpr{
						pos: position{line: 790, col: 17, offset: 20182},
						expr: &charClassMatcher{
							pos:        position{line: 790, col: 17, offset: 20182},
							val:        "[+-]",
							chars:      []rune{'+', '-'},
							ignoreCase: false,
							inverted:   false,
						},
					},
					&oneOrMoreExpr{
						pos: position{line: 790, col: 23, offset: 20188},
						expr: &ruleRefExpr{
							pos:  position{line: 790, col: 23, offset: 20188},
							name: "doubleDigit",
						},
					},
				},
			},
		},
		{
			name: "exponentPart",
			pos:  position{line: 792, col: 1, offset: 20202},
			expr: &seqExpr{
				pos: position{line: 792, col: 16, offset: 20217},
				exprs: []interface{}{
					&litMatcher{
						pos:        position{line: 792, col: 16, offset: 20217},
						val:        "e",
						ignoreCase: true,
					},
					&ruleRefExpr{
						pos:  position{line: 792, col: 21, offset: 20222},
						name: "signedInteger",
					},
				},
			},
		},
		{
			name: "h16",
			pos:  position{line: 794, col: 1, offset: 20237},
			expr: &actionExpr{
				pos: position{line: 794, col: 7, offset: 20243},
				run: (*parser).callonh161,
				expr: &labeledExpr{
					pos:   position{line: 794, col: 7, offset: 20243},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 794, col: 13, offset: 20249},
						expr: &ruleRefExpr{
							pos:  position{line: 794, col: 13, offset: 20249},
							name: "hexdigit",
						},
					},
				},
			},
		},
		{
			name: "hexdigit",
			pos:  position{line: 796, col: 1, offset: 20291},
			expr: &charClassMatcher{
				pos:        position{line: 796, col: 12, offset: 20302},
				val:        "[0-9a-fA-F]",
				ranges:     []rune{'0', '9', 'a', 'f', 'A', 'F'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name:        "boomWord",
			displayName: "\"boomWord\"",
			pos:         position{line: 798, col: 1, offset: 20315},
			expr: &actionExpr{
				pos: position{line: 798, col: 23, offset: 20337},
				run: (*parser).callonboomWord1,
				expr: &labeledExpr{
					pos:   position{line: 798, col: 23, offset: 20337},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 798, col: 29, offset: 20343},
						expr: &ruleRefExpr{
							pos:  position{line: 798, col: 29, offset: 20343},
							name: "boomWordPart",
						},
					},
				},
			},
		},
		{
			name: "boomWordPart",
			pos:  position{line: 800, col: 1, offset: 20389},
			expr: &seqExpr{
				pos: position{line: 801, col: 5, offset: 20406},
				exprs: []interface{}{
					&notExpr{
						pos: position{line: 801, col: 5, offset: 20406},
						expr: &choiceExpr{
							pos: position{line: 801, col: 7, offset: 20408},
							alternatives: []interface{}{
								&charClassMatcher{
									pos:        position{line: 801, col: 7, offset: 20408},
									val:        "[\\x00-\\x1F\\x5C(),!><=\\x22|\\x27;]",
									chars:      []rune{'\\', '(', ')', ',', '!', '>', '<', '=', '"', '|', '\'', ';'},
									ranges:     []rune{'\x00', '\x1f'},
									ignoreCase: false,
									inverted:   false,
								},
								&ruleRefExpr{
									pos:  position{line: 801, col: 42, offset: 20443},
									name: "ws",
								},
							},
						},
					},
					&anyMatcher{
						line: 801, col: 46, offset: 20447,
					},
				},
			},
		},
		{
			name: "quotedString",
			pos:  position{line: 803, col: 1, offset: 20450},
			expr: &choiceExpr{
				pos: position{line: 804, col: 5, offset: 20467},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 804, col: 5, offset: 20467},
						run: (*parser).callonquotedString2,
						expr: &seqExpr{
							pos: position{line: 804, col: 5, offset: 20467},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 804, col: 5, offset: 20467},
									val:        "\"",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 804, col: 9, offset: 20471},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 804, col: 11, offset: 20473},
										expr: &ruleRefExpr{
											pos:  position{line: 804, col: 11, offset: 20473},
											name: "doubleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 804, col: 29, offset: 20491},
									val:        "\"",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 805, col: 5, offset: 20528},
						run: (*parser).callonquotedString9,
						expr: &seqExpr{
							pos: position{line: 805, col: 5, offset: 20528},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 805, col: 5, offset: 20528},
									val:        "'",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 805, col: 9, offset: 20532},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 805, col: 11, offset: 20534},
										expr: &ruleRefExpr{
											pos:  position{line: 805, col: 11, offset: 20534},
											name: "singleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 805, col: 29, offset: 20552},
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
			pos:  position{line: 807, col: 1, offset: 20586},
			expr: &choiceExpr{
				pos: position{line: 808, col: 5, offset: 20607},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 808, col: 5, offset: 20607},
						run: (*parser).callondoubleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 808, col: 5, offset: 20607},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 808, col: 5, offset: 20607},
									expr: &choiceExpr{
										pos: position{line: 808, col: 7, offset: 20609},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 808, col: 7, offset: 20609},
												val:        "\"",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 808, col: 13, offset: 20615},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 808, col: 26, offset: 20628,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 809, col: 5, offset: 20665},
						run: (*parser).callondoubleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 809, col: 5, offset: 20665},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 809, col: 5, offset: 20665},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 809, col: 10, offset: 20670},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 809, col: 12, offset: 20672},
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
			pos:  position{line: 811, col: 1, offset: 20706},
			expr: &choiceExpr{
				pos: position{line: 812, col: 5, offset: 20727},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 812, col: 5, offset: 20727},
						run: (*parser).callonsingleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 812, col: 5, offset: 20727},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 812, col: 5, offset: 20727},
									expr: &choiceExpr{
										pos: position{line: 812, col: 7, offset: 20729},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 812, col: 7, offset: 20729},
												val:        "'",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 812, col: 13, offset: 20735},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 812, col: 26, offset: 20748,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 813, col: 5, offset: 20785},
						run: (*parser).callonsingleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 813, col: 5, offset: 20785},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 813, col: 5, offset: 20785},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 813, col: 10, offset: 20790},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 813, col: 12, offset: 20792},
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
			pos:  position{line: 815, col: 1, offset: 20826},
			expr: &choiceExpr{
				pos: position{line: 815, col: 18, offset: 20843},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 815, col: 18, offset: 20843},
						name: "singleCharEscape",
					},
					&ruleRefExpr{
						pos:  position{line: 815, col: 37, offset: 20862},
						name: "unicodeEscape",
					},
				},
			},
		},
		{
			name: "singleCharEscape",
			pos:  position{line: 817, col: 1, offset: 20877},
			expr: &choiceExpr{
				pos: position{line: 818, col: 5, offset: 20898},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 818, col: 5, offset: 20898},
						val:        "'",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 819, col: 5, offset: 20906},
						val:        "\"",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 820, col: 5, offset: 20914},
						val:        "\\",
						ignoreCase: false,
					},
					&actionExpr{
						pos: position{line: 821, col: 5, offset: 20923},
						run: (*parser).callonsingleCharEscape5,
						expr: &litMatcher{
							pos:        position{line: 821, col: 5, offset: 20923},
							val:        "b",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 822, col: 5, offset: 20952},
						run: (*parser).callonsingleCharEscape7,
						expr: &litMatcher{
							pos:        position{line: 822, col: 5, offset: 20952},
							val:        "f",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 823, col: 5, offset: 20981},
						run: (*parser).callonsingleCharEscape9,
						expr: &litMatcher{
							pos:        position{line: 823, col: 5, offset: 20981},
							val:        "n",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 824, col: 5, offset: 21010},
						run: (*parser).callonsingleCharEscape11,
						expr: &litMatcher{
							pos:        position{line: 824, col: 5, offset: 21010},
							val:        "r",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 825, col: 5, offset: 21039},
						run: (*parser).callonsingleCharEscape13,
						expr: &litMatcher{
							pos:        position{line: 825, col: 5, offset: 21039},
							val:        "t",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 826, col: 5, offset: 21068},
						run: (*parser).callonsingleCharEscape15,
						expr: &litMatcher{
							pos:        position{line: 826, col: 5, offset: 21068},
							val:        "v",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "unicodeEscape",
			pos:  position{line: 828, col: 1, offset: 21094},
			expr: &seqExpr{
				pos: position{line: 829, col: 5, offset: 21112},
				exprs: []interface{}{
					&litMatcher{
						pos:        position{line: 829, col: 5, offset: 21112},
						val:        "u",
						ignoreCase: false,
					},
					&ruleRefExpr{
						pos:  position{line: 829, col: 9, offset: 21116},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 829, col: 18, offset: 21125},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 829, col: 27, offset: 21134},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 829, col: 36, offset: 21143},
						name: "hexdigit",
					},
				},
			},
		},
		{
			name: "reString",
			pos:  position{line: 831, col: 1, offset: 21153},
			expr: &actionExpr{
				pos: position{line: 832, col: 5, offset: 21166},
				run: (*parser).callonreString1,
				expr: &seqExpr{
					pos: position{line: 832, col: 5, offset: 21166},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 832, col: 5, offset: 21166},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 832, col: 9, offset: 21170},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 832, col: 11, offset: 21172},
								name: "reBody",
							},
						},
						&litMatcher{
							pos:        position{line: 832, col: 18, offset: 21179},
							val:        "/",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reBody",
			pos:  position{line: 834, col: 1, offset: 21202},
			expr: &actionExpr{
				pos: position{line: 835, col: 5, offset: 21213},
				run: (*parser).callonreBody1,
				expr: &oneOrMoreExpr{
					pos: position{line: 835, col: 5, offset: 21213},
					expr: &choiceExpr{
						pos: position{line: 835, col: 6, offset: 21214},
						alternatives: []interface{}{
							&charClassMatcher{
								pos:        position{line: 835, col: 6, offset: 21214},
								val:        "[^/\\\\]",
								chars:      []rune{'/', '\\'},
								ignoreCase: false,
								inverted:   true,
							},
							&litMatcher{
								pos:        position{line: 835, col: 13, offset: 21221},
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
			pos:  position{line: 837, col: 1, offset: 21261},
			expr: &charClassMatcher{
				pos:        position{line: 838, col: 5, offset: 21277},
				val:        "[\\x00-\\x1f\\\\]",
				chars:      []rune{'\\'},
				ranges:     []rune{'\x00', '\x1f'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "ws",
			pos:  position{line: 840, col: 1, offset: 21292},
			expr: &choiceExpr{
				pos: position{line: 841, col: 5, offset: 21299},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 841, col: 5, offset: 21299},
						val:        "\t",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 842, col: 5, offset: 21308},
						val:        "\v",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 843, col: 5, offset: 21317},
						val:        "\f",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 844, col: 5, offset: 21326},
						val:        " ",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 845, col: 5, offset: 21334},
						val:        "\u00a0",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 846, col: 5, offset: 21347},
						val:        "\ufeff",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name:        "_",
			displayName: "\"whitespace\"",
			pos:         position{line: 848, col: 1, offset: 21357},
			expr: &oneOrMoreExpr{
				pos: position{line: 848, col: 18, offset: 21374},
				expr: &ruleRefExpr{
					pos:  position{line: 848, col: 18, offset: 21374},
					name: "ws",
				},
			},
		},
		{
			name: "EOF",
			pos:  position{line: 850, col: 1, offset: 21379},
			expr: &notExpr{
				pos: position{line: 850, col: 7, offset: 21385},
				expr: &anyMatcher{
					line: 850, col: 8, offset: 21386,
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

func (c *current) onfieldExprList1(first, rest interface{}) (interface{}, error) {
	result := []interface{}{first}

	for _, r := range rest.([]interface{}) {
		result = append(result, r.([]interface{})[3])
	}

	return result, nil

}

func (p *parser) callonfieldExprList1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldExprList1(stack["first"], stack["rest"])
}

func (c *current) onfieldNameList1(first, rest interface{}) (interface{}, error) {
	result := []interface{}{first}
	for _, r := range rest.([]interface{}) {
		result = append(result, r.([]interface{})[3])
	}
	return result, nil

}

func (p *parser) callonfieldNameList1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldNameList1(stack["first"], stack["rest"])
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

func (c *current) onfieldReducerOp22() (interface{}, error) {
	return "CountDistinct", nil
}

func (p *parser) callonfieldReducerOp22() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldReducerOp22()
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

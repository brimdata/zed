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
			pos:  position{line: 313, col: 1, offset: 8978},
			expr: &actionExpr{
				pos: position{line: 313, col: 9, offset: 8986},
				run: (*parser).callonstart1,
				expr: &seqExpr{
					pos: position{line: 313, col: 9, offset: 8986},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 313, col: 9, offset: 8986},
							expr: &ruleRefExpr{
								pos:  position{line: 313, col: 9, offset: 8986},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 313, col: 12, offset: 8989},
							label: "ast",
							expr: &ruleRefExpr{
								pos:  position{line: 313, col: 16, offset: 8993},
								name: "boomCommand",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 313, col: 28, offset: 9005},
							expr: &ruleRefExpr{
								pos:  position{line: 313, col: 28, offset: 9005},
								name: "_",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 313, col: 31, offset: 9008},
							name: "EOF",
						},
					},
				},
			},
		},
		{
			name: "boomCommand",
			pos:  position{line: 315, col: 1, offset: 9033},
			expr: &choiceExpr{
				pos: position{line: 316, col: 5, offset: 9049},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 316, col: 5, offset: 9049},
						run: (*parser).callonboomCommand2,
						expr: &labeledExpr{
							pos:   position{line: 316, col: 5, offset: 9049},
							label: "procs",
							expr: &ruleRefExpr{
								pos:  position{line: 316, col: 11, offset: 9055},
								name: "procChain",
							},
						},
					},
					&actionExpr{
						pos: position{line: 320, col: 5, offset: 9228},
						run: (*parser).callonboomCommand5,
						expr: &seqExpr{
							pos: position{line: 320, col: 5, offset: 9228},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 320, col: 5, offset: 9228},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 320, col: 7, offset: 9230},
										name: "search",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 320, col: 14, offset: 9237},
									expr: &ruleRefExpr{
										pos:  position{line: 320, col: 14, offset: 9237},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 320, col: 17, offset: 9240},
									label: "rest",
									expr: &zeroOrMoreExpr{
										pos: position{line: 320, col: 22, offset: 9245},
										expr: &ruleRefExpr{
											pos:  position{line: 320, col: 22, offset: 9245},
											name: "chainedProc",
										},
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 327, col: 5, offset: 9455},
						run: (*parser).callonboomCommand14,
						expr: &labeledExpr{
							pos:   position{line: 327, col: 5, offset: 9455},
							label: "s",
							expr: &ruleRefExpr{
								pos:  position{line: 327, col: 7, offset: 9457},
								name: "search",
							},
						},
					},
				},
			},
		},
		{
			name: "procChain",
			pos:  position{line: 331, col: 1, offset: 9528},
			expr: &actionExpr{
				pos: position{line: 332, col: 5, offset: 9542},
				run: (*parser).callonprocChain1,
				expr: &seqExpr{
					pos: position{line: 332, col: 5, offset: 9542},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 332, col: 5, offset: 9542},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 332, col: 11, offset: 9548},
								name: "proc",
							},
						},
						&labeledExpr{
							pos:   position{line: 332, col: 16, offset: 9553},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 332, col: 21, offset: 9558},
								expr: &ruleRefExpr{
									pos:  position{line: 332, col: 21, offset: 9558},
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
			pos:  position{line: 340, col: 1, offset: 9744},
			expr: &actionExpr{
				pos: position{line: 340, col: 15, offset: 9758},
				run: (*parser).callonchainedProc1,
				expr: &seqExpr{
					pos: position{line: 340, col: 15, offset: 9758},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 340, col: 15, offset: 9758},
							expr: &ruleRefExpr{
								pos:  position{line: 340, col: 15, offset: 9758},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 340, col: 18, offset: 9761},
							val:        "|",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 340, col: 22, offset: 9765},
							expr: &ruleRefExpr{
								pos:  position{line: 340, col: 22, offset: 9765},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 340, col: 25, offset: 9768},
							label: "p",
							expr: &ruleRefExpr{
								pos:  position{line: 340, col: 27, offset: 9770},
								name: "proc",
							},
						},
					},
				},
			},
		},
		{
			name: "search",
			pos:  position{line: 342, col: 1, offset: 9794},
			expr: &actionExpr{
				pos: position{line: 343, col: 5, offset: 9805},
				run: (*parser).callonsearch1,
				expr: &labeledExpr{
					pos:   position{line: 343, col: 5, offset: 9805},
					label: "expr",
					expr: &ruleRefExpr{
						pos:  position{line: 343, col: 10, offset: 9810},
						name: "searchExpr",
					},
				},
			},
		},
		{
			name: "searchExpr",
			pos:  position{line: 347, col: 1, offset: 9869},
			expr: &actionExpr{
				pos: position{line: 348, col: 5, offset: 9884},
				run: (*parser).callonsearchExpr1,
				expr: &seqExpr{
					pos: position{line: 348, col: 5, offset: 9884},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 348, col: 5, offset: 9884},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 348, col: 11, offset: 9890},
								name: "searchTerm",
							},
						},
						&labeledExpr{
							pos:   position{line: 348, col: 22, offset: 9901},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 348, col: 27, offset: 9906},
								expr: &ruleRefExpr{
									pos:  position{line: 348, col: 27, offset: 9906},
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
			pos:  position{line: 352, col: 1, offset: 9974},
			expr: &actionExpr{
				pos: position{line: 352, col: 18, offset: 9991},
				run: (*parser).callonoredSearchTerm1,
				expr: &seqExpr{
					pos: position{line: 352, col: 18, offset: 9991},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 352, col: 18, offset: 9991},
							name: "_",
						},
						&ruleRefExpr{
							pos:  position{line: 352, col: 20, offset: 9993},
							name: "orToken",
						},
						&ruleRefExpr{
							pos:  position{line: 352, col: 28, offset: 10001},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 352, col: 30, offset: 10003},
							label: "t",
							expr: &ruleRefExpr{
								pos:  position{line: 352, col: 32, offset: 10005},
								name: "searchTerm",
							},
						},
					},
				},
			},
		},
		{
			name: "searchTerm",
			pos:  position{line: 354, col: 1, offset: 10035},
			expr: &actionExpr{
				pos: position{line: 355, col: 5, offset: 10050},
				run: (*parser).callonsearchTerm1,
				expr: &seqExpr{
					pos: position{line: 355, col: 5, offset: 10050},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 355, col: 5, offset: 10050},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 355, col: 11, offset: 10056},
								name: "searchFactor",
							},
						},
						&labeledExpr{
							pos:   position{line: 355, col: 24, offset: 10069},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 355, col: 29, offset: 10074},
								expr: &ruleRefExpr{
									pos:  position{line: 355, col: 29, offset: 10074},
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
			pos:  position{line: 359, col: 1, offset: 10144},
			expr: &actionExpr{
				pos: position{line: 359, col: 19, offset: 10162},
				run: (*parser).callonandedSearchTerm1,
				expr: &seqExpr{
					pos: position{line: 359, col: 19, offset: 10162},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 359, col: 19, offset: 10162},
							name: "_",
						},
						&zeroOrOneExpr{
							pos: position{line: 359, col: 21, offset: 10164},
							expr: &seqExpr{
								pos: position{line: 359, col: 22, offset: 10165},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 359, col: 22, offset: 10165},
										name: "andToken",
									},
									&ruleRefExpr{
										pos:  position{line: 359, col: 31, offset: 10174},
										name: "_",
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 359, col: 35, offset: 10178},
							label: "f",
							expr: &ruleRefExpr{
								pos:  position{line: 359, col: 37, offset: 10180},
								name: "searchFactor",
							},
						},
					},
				},
			},
		},
		{
			name: "searchFactor",
			pos:  position{line: 361, col: 1, offset: 10212},
			expr: &choiceExpr{
				pos: position{line: 362, col: 5, offset: 10229},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 362, col: 5, offset: 10229},
						run: (*parser).callonsearchFactor2,
						expr: &seqExpr{
							pos: position{line: 362, col: 5, offset: 10229},
							exprs: []interface{}{
								&choiceExpr{
									pos: position{line: 362, col: 6, offset: 10230},
									alternatives: []interface{}{
										&seqExpr{
											pos: position{line: 362, col: 6, offset: 10230},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 362, col: 6, offset: 10230},
													name: "notToken",
												},
												&ruleRefExpr{
													pos:  position{line: 362, col: 15, offset: 10239},
													name: "_",
												},
											},
										},
										&seqExpr{
											pos: position{line: 362, col: 19, offset: 10243},
											exprs: []interface{}{
												&litMatcher{
													pos:        position{line: 362, col: 19, offset: 10243},
													val:        "!",
													ignoreCase: false,
												},
												&zeroOrOneExpr{
													pos: position{line: 362, col: 23, offset: 10247},
													expr: &ruleRefExpr{
														pos:  position{line: 362, col: 23, offset: 10247},
														name: "_",
													},
												},
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 362, col: 27, offset: 10251},
									label: "e",
									expr: &ruleRefExpr{
										pos:  position{line: 362, col: 29, offset: 10253},
										name: "searchExpr",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 365, col: 5, offset: 10312},
						run: (*parser).callonsearchFactor14,
						expr: &seqExpr{
							pos: position{line: 365, col: 5, offset: 10312},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 365, col: 5, offset: 10312},
									expr: &litMatcher{
										pos:        position{line: 365, col: 7, offset: 10314},
										val:        "-",
										ignoreCase: false,
									},
								},
								&labeledExpr{
									pos:   position{line: 365, col: 12, offset: 10319},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 365, col: 14, offset: 10321},
										name: "searchPred",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 366, col: 5, offset: 10354},
						run: (*parser).callonsearchFactor20,
						expr: &seqExpr{
							pos: position{line: 366, col: 5, offset: 10354},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 366, col: 5, offset: 10354},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 366, col: 9, offset: 10358},
									expr: &ruleRefExpr{
										pos:  position{line: 366, col: 9, offset: 10358},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 366, col: 12, offset: 10361},
									label: "expr",
									expr: &ruleRefExpr{
										pos:  position{line: 366, col: 17, offset: 10366},
										name: "searchExpr",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 366, col: 28, offset: 10377},
									expr: &ruleRefExpr{
										pos:  position{line: 366, col: 28, offset: 10377},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 366, col: 31, offset: 10380},
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
			pos:  position{line: 368, col: 1, offset: 10406},
			expr: &choiceExpr{
				pos: position{line: 369, col: 5, offset: 10421},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 369, col: 5, offset: 10421},
						run: (*parser).callonsearchPred2,
						expr: &seqExpr{
							pos: position{line: 369, col: 5, offset: 10421},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 369, col: 5, offset: 10421},
									val:        "*",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 369, col: 9, offset: 10425},
									expr: &ruleRefExpr{
										pos:  position{line: 369, col: 9, offset: 10425},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 369, col: 12, offset: 10428},
									label: "fieldComparator",
									expr: &ruleRefExpr{
										pos:  position{line: 369, col: 28, offset: 10444},
										name: "equalityToken",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 369, col: 42, offset: 10458},
									expr: &ruleRefExpr{
										pos:  position{line: 369, col: 42, offset: 10458},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 369, col: 45, offset: 10461},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 369, col: 47, offset: 10463},
										name: "searchValue",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 372, col: 5, offset: 10540},
						run: (*parser).callonsearchPred13,
						expr: &litMatcher{
							pos:        position{line: 372, col: 5, offset: 10540},
							val:        "*",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 375, col: 5, offset: 10599},
						run: (*parser).callonsearchPred15,
						expr: &seqExpr{
							pos: position{line: 375, col: 5, offset: 10599},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 375, col: 5, offset: 10599},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 375, col: 7, offset: 10601},
										name: "fieldExpr",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 375, col: 17, offset: 10611},
									expr: &ruleRefExpr{
										pos:  position{line: 375, col: 17, offset: 10611},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 375, col: 20, offset: 10614},
									label: "fieldComparator",
									expr: &ruleRefExpr{
										pos:  position{line: 375, col: 36, offset: 10630},
										name: "equalityToken",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 375, col: 50, offset: 10644},
									expr: &ruleRefExpr{
										pos:  position{line: 375, col: 50, offset: 10644},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 375, col: 53, offset: 10647},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 375, col: 55, offset: 10649},
										name: "searchValue",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 378, col: 5, offset: 10731},
						run: (*parser).callonsearchPred27,
						expr: &seqExpr{
							pos: position{line: 378, col: 5, offset: 10731},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 378, col: 5, offset: 10731},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 378, col: 7, offset: 10733},
										name: "searchValue",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 378, col: 19, offset: 10745},
									expr: &ruleRefExpr{
										pos:  position{line: 378, col: 19, offset: 10745},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 378, col: 22, offset: 10748},
									name: "inToken",
								},
								&zeroOrOneExpr{
									pos: position{line: 378, col: 30, offset: 10756},
									expr: &ruleRefExpr{
										pos:  position{line: 378, col: 30, offset: 10756},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 378, col: 33, offset: 10759},
									val:        "*",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 381, col: 5, offset: 10817},
						run: (*parser).callonsearchPred37,
						expr: &seqExpr{
							pos: position{line: 381, col: 5, offset: 10817},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 381, col: 5, offset: 10817},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 381, col: 7, offset: 10819},
										name: "searchValue",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 381, col: 19, offset: 10831},
									expr: &ruleRefExpr{
										pos:  position{line: 381, col: 19, offset: 10831},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 381, col: 22, offset: 10834},
									name: "inToken",
								},
								&zeroOrOneExpr{
									pos: position{line: 381, col: 30, offset: 10842},
									expr: &ruleRefExpr{
										pos:  position{line: 381, col: 30, offset: 10842},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 381, col: 33, offset: 10845},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 381, col: 35, offset: 10847},
										name: "fieldReference",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 384, col: 5, offset: 10921},
						run: (*parser).callonsearchPred48,
						expr: &labeledExpr{
							pos:   position{line: 384, col: 5, offset: 10921},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 384, col: 7, offset: 10923},
								name: "searchValue",
							},
						},
					},
				},
			},
		},
		{
			name: "searchValue",
			pos:  position{line: 393, col: 1, offset: 11217},
			expr: &choiceExpr{
				pos: position{line: 394, col: 5, offset: 11233},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 394, col: 5, offset: 11233},
						run: (*parser).callonsearchValue2,
						expr: &labeledExpr{
							pos:   position{line: 394, col: 5, offset: 11233},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 394, col: 7, offset: 11235},
								name: "quotedString",
							},
						},
					},
					&actionExpr{
						pos: position{line: 397, col: 5, offset: 11306},
						run: (*parser).callonsearchValue5,
						expr: &labeledExpr{
							pos:   position{line: 397, col: 5, offset: 11306},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 397, col: 7, offset: 11308},
								name: "reString",
							},
						},
					},
					&actionExpr{
						pos: position{line: 400, col: 5, offset: 11375},
						run: (*parser).callonsearchValue8,
						expr: &labeledExpr{
							pos:   position{line: 400, col: 5, offset: 11375},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 400, col: 7, offset: 11377},
								name: "port",
							},
						},
					},
					&actionExpr{
						pos: position{line: 403, col: 5, offset: 11436},
						run: (*parser).callonsearchValue11,
						expr: &labeledExpr{
							pos:   position{line: 403, col: 5, offset: 11436},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 403, col: 7, offset: 11438},
								name: "ip6subnet",
							},
						},
					},
					&actionExpr{
						pos: position{line: 406, col: 5, offset: 11506},
						run: (*parser).callonsearchValue14,
						expr: &labeledExpr{
							pos:   position{line: 406, col: 5, offset: 11506},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 406, col: 7, offset: 11508},
								name: "ip6addr",
							},
						},
					},
					&actionExpr{
						pos: position{line: 409, col: 5, offset: 11572},
						run: (*parser).callonsearchValue17,
						expr: &labeledExpr{
							pos:   position{line: 409, col: 5, offset: 11572},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 409, col: 7, offset: 11574},
								name: "subnet",
							},
						},
					},
					&actionExpr{
						pos: position{line: 412, col: 5, offset: 11639},
						run: (*parser).callonsearchValue20,
						expr: &labeledExpr{
							pos:   position{line: 412, col: 5, offset: 11639},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 412, col: 7, offset: 11641},
								name: "addr",
							},
						},
					},
					&actionExpr{
						pos: position{line: 415, col: 5, offset: 11702},
						run: (*parser).callonsearchValue23,
						expr: &labeledExpr{
							pos:   position{line: 415, col: 5, offset: 11702},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 415, col: 7, offset: 11704},
								name: "sdouble",
							},
						},
					},
					&actionExpr{
						pos: position{line: 418, col: 5, offset: 11770},
						run: (*parser).callonsearchValue26,
						expr: &seqExpr{
							pos: position{line: 418, col: 5, offset: 11770},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 418, col: 5, offset: 11770},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 418, col: 7, offset: 11772},
										name: "sinteger",
									},
								},
								&notExpr{
									pos: position{line: 418, col: 16, offset: 11781},
									expr: &ruleRefExpr{
										pos:  position{line: 418, col: 17, offset: 11782},
										name: "boomWord",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 421, col: 5, offset: 11846},
						run: (*parser).callonsearchValue32,
						expr: &seqExpr{
							pos: position{line: 421, col: 5, offset: 11846},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 421, col: 5, offset: 11846},
									expr: &seqExpr{
										pos: position{line: 421, col: 7, offset: 11848},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 421, col: 7, offset: 11848},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 421, col: 22, offset: 11863},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 421, col: 25, offset: 11866},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 421, col: 27, offset: 11868},
										name: "booleanLiteral",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 422, col: 5, offset: 11905},
						run: (*parser).callonsearchValue40,
						expr: &seqExpr{
							pos: position{line: 422, col: 5, offset: 11905},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 422, col: 5, offset: 11905},
									expr: &seqExpr{
										pos: position{line: 422, col: 7, offset: 11907},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 422, col: 7, offset: 11907},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 422, col: 22, offset: 11922},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 422, col: 25, offset: 11925},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 422, col: 27, offset: 11927},
										name: "unsetLiteral",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 423, col: 5, offset: 11962},
						run: (*parser).callonsearchValue48,
						expr: &seqExpr{
							pos: position{line: 423, col: 5, offset: 11962},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 423, col: 5, offset: 11962},
									expr: &seqExpr{
										pos: position{line: 423, col: 7, offset: 11964},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 423, col: 7, offset: 11964},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 423, col: 22, offset: 11979},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 423, col: 25, offset: 11982},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 423, col: 27, offset: 11984},
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
			pos:  position{line: 431, col: 1, offset: 12208},
			expr: &choiceExpr{
				pos: position{line: 432, col: 5, offset: 12227},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 432, col: 5, offset: 12227},
						name: "andToken",
					},
					&ruleRefExpr{
						pos:  position{line: 433, col: 5, offset: 12240},
						name: "orToken",
					},
					&ruleRefExpr{
						pos:  position{line: 434, col: 5, offset: 12252},
						name: "inToken",
					},
				},
			},
		},
		{
			name: "booleanLiteral",
			pos:  position{line: 436, col: 1, offset: 12261},
			expr: &choiceExpr{
				pos: position{line: 437, col: 5, offset: 12280},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 437, col: 5, offset: 12280},
						run: (*parser).callonbooleanLiteral2,
						expr: &litMatcher{
							pos:        position{line: 437, col: 5, offset: 12280},
							val:        "true",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 438, col: 5, offset: 12348},
						run: (*parser).callonbooleanLiteral4,
						expr: &litMatcher{
							pos:        position{line: 438, col: 5, offset: 12348},
							val:        "false",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "unsetLiteral",
			pos:  position{line: 440, col: 1, offset: 12414},
			expr: &actionExpr{
				pos: position{line: 441, col: 5, offset: 12431},
				run: (*parser).callonunsetLiteral1,
				expr: &litMatcher{
					pos:        position{line: 441, col: 5, offset: 12431},
					val:        "nil",
					ignoreCase: false,
				},
			},
		},
		{
			name: "procList",
			pos:  position{line: 443, col: 1, offset: 12492},
			expr: &actionExpr{
				pos: position{line: 444, col: 5, offset: 12505},
				run: (*parser).callonprocList1,
				expr: &seqExpr{
					pos: position{line: 444, col: 5, offset: 12505},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 444, col: 5, offset: 12505},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 444, col: 11, offset: 12511},
								name: "procChain",
							},
						},
						&labeledExpr{
							pos:   position{line: 444, col: 21, offset: 12521},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 444, col: 26, offset: 12526},
								expr: &ruleRefExpr{
									pos:  position{line: 444, col: 26, offset: 12526},
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
			pos:  position{line: 453, col: 1, offset: 12750},
			expr: &actionExpr{
				pos: position{line: 454, col: 5, offset: 12768},
				run: (*parser).callonparallelChain1,
				expr: &seqExpr{
					pos: position{line: 454, col: 5, offset: 12768},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 454, col: 5, offset: 12768},
							expr: &ruleRefExpr{
								pos:  position{line: 454, col: 5, offset: 12768},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 454, col: 8, offset: 12771},
							val:        ";",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 454, col: 12, offset: 12775},
							expr: &ruleRefExpr{
								pos:  position{line: 454, col: 12, offset: 12775},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 454, col: 15, offset: 12778},
							label: "ch",
							expr: &ruleRefExpr{
								pos:  position{line: 454, col: 18, offset: 12781},
								name: "procChain",
							},
						},
					},
				},
			},
		},
		{
			name: "proc",
			pos:  position{line: 456, col: 1, offset: 12831},
			expr: &choiceExpr{
				pos: position{line: 457, col: 5, offset: 12840},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 457, col: 5, offset: 12840},
						name: "simpleProc",
					},
					&ruleRefExpr{
						pos:  position{line: 458, col: 5, offset: 12855},
						name: "reducerProc",
					},
					&actionExpr{
						pos: position{line: 459, col: 5, offset: 12871},
						run: (*parser).callonproc4,
						expr: &seqExpr{
							pos: position{line: 459, col: 5, offset: 12871},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 459, col: 5, offset: 12871},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 459, col: 9, offset: 12875},
									expr: &ruleRefExpr{
										pos:  position{line: 459, col: 9, offset: 12875},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 459, col: 12, offset: 12878},
									label: "proc",
									expr: &ruleRefExpr{
										pos:  position{line: 459, col: 17, offset: 12883},
										name: "procList",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 459, col: 26, offset: 12892},
									expr: &ruleRefExpr{
										pos:  position{line: 459, col: 26, offset: 12892},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 459, col: 29, offset: 12895},
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
			pos:  position{line: 463, col: 1, offset: 12931},
			expr: &actionExpr{
				pos: position{line: 464, col: 5, offset: 12943},
				run: (*parser).callongroupBy1,
				expr: &seqExpr{
					pos: position{line: 464, col: 5, offset: 12943},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 464, col: 5, offset: 12943},
							val:        "by",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 464, col: 11, offset: 12949},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 464, col: 13, offset: 12951},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 464, col: 18, offset: 12956},
								name: "fieldExprList",
							},
						},
					},
				},
			},
		},
		{
			name: "everyDur",
			pos:  position{line: 466, col: 1, offset: 12992},
			expr: &actionExpr{
				pos: position{line: 467, col: 5, offset: 13005},
				run: (*parser).calloneveryDur1,
				expr: &seqExpr{
					pos: position{line: 467, col: 5, offset: 13005},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 467, col: 5, offset: 13005},
							val:        "every",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 467, col: 14, offset: 13014},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 467, col: 16, offset: 13016},
							label: "dur",
							expr: &ruleRefExpr{
								pos:  position{line: 467, col: 20, offset: 13020},
								name: "duration",
							},
						},
					},
				},
			},
		},
		{
			name: "equalityToken",
			pos:  position{line: 469, col: 1, offset: 13050},
			expr: &choiceExpr{
				pos: position{line: 470, col: 5, offset: 13068},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 470, col: 5, offset: 13068},
						run: (*parser).callonequalityToken2,
						expr: &litMatcher{
							pos:        position{line: 470, col: 5, offset: 13068},
							val:        "=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 471, col: 5, offset: 13098},
						run: (*parser).callonequalityToken4,
						expr: &litMatcher{
							pos:        position{line: 471, col: 5, offset: 13098},
							val:        "!=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 472, col: 5, offset: 13130},
						run: (*parser).callonequalityToken6,
						expr: &litMatcher{
							pos:        position{line: 472, col: 5, offset: 13130},
							val:        "<=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 473, col: 5, offset: 13161},
						run: (*parser).callonequalityToken8,
						expr: &litMatcher{
							pos:        position{line: 473, col: 5, offset: 13161},
							val:        ">=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 474, col: 5, offset: 13192},
						run: (*parser).callonequalityToken10,
						expr: &litMatcher{
							pos:        position{line: 474, col: 5, offset: 13192},
							val:        "<",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 475, col: 5, offset: 13221},
						run: (*parser).callonequalityToken12,
						expr: &litMatcher{
							pos:        position{line: 475, col: 5, offset: 13221},
							val:        ">",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "types",
			pos:  position{line: 477, col: 1, offset: 13247},
			expr: &choiceExpr{
				pos: position{line: 478, col: 5, offset: 13257},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 478, col: 5, offset: 13257},
						val:        "bool",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 479, col: 5, offset: 13268},
						val:        "int",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 480, col: 5, offset: 13278},
						val:        "count",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 481, col: 5, offset: 13290},
						val:        "double",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 482, col: 5, offset: 13303},
						val:        "string",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 483, col: 5, offset: 13316},
						val:        "addr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 484, col: 5, offset: 13327},
						val:        "subnet",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 485, col: 5, offset: 13340},
						val:        "port",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "dash",
			pos:  position{line: 487, col: 1, offset: 13348},
			expr: &choiceExpr{
				pos: position{line: 487, col: 8, offset: 13355},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 487, col: 8, offset: 13355},
						val:        "-",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 487, col: 14, offset: 13361},
						val:        "—",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 487, col: 25, offset: 13372},
						val:        "–",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 487, col: 36, offset: 13383},
						exprs: []interface{}{
							&litMatcher{
								pos:        position{line: 487, col: 36, offset: 13383},
								val:        "to",
								ignoreCase: false,
							},
							&andExpr{
								pos: position{line: 487, col: 40, offset: 13387},
								expr: &ruleRefExpr{
									pos:  position{line: 487, col: 42, offset: 13389},
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
			pos:  position{line: 489, col: 1, offset: 13393},
			expr: &litMatcher{
				pos:        position{line: 489, col: 12, offset: 13404},
				val:        "and",
				ignoreCase: false,
			},
		},
		{
			name: "orToken",
			pos:  position{line: 490, col: 1, offset: 13410},
			expr: &litMatcher{
				pos:        position{line: 490, col: 11, offset: 13420},
				val:        "or",
				ignoreCase: false,
			},
		},
		{
			name: "inToken",
			pos:  position{line: 491, col: 1, offset: 13425},
			expr: &litMatcher{
				pos:        position{line: 491, col: 11, offset: 13435},
				val:        "in",
				ignoreCase: false,
			},
		},
		{
			name: "notToken",
			pos:  position{line: 492, col: 1, offset: 13440},
			expr: &litMatcher{
				pos:        position{line: 492, col: 12, offset: 13451},
				val:        "not",
				ignoreCase: false,
			},
		},
		{
			name: "fieldName",
			pos:  position{line: 494, col: 1, offset: 13458},
			expr: &actionExpr{
				pos: position{line: 494, col: 13, offset: 13470},
				run: (*parser).callonfieldName1,
				expr: &seqExpr{
					pos: position{line: 494, col: 13, offset: 13470},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 494, col: 13, offset: 13470},
							name: "fieldNameStart",
						},
						&zeroOrMoreExpr{
							pos: position{line: 494, col: 28, offset: 13485},
							expr: &ruleRefExpr{
								pos:  position{line: 494, col: 28, offset: 13485},
								name: "fieldNameRest",
							},
						},
					},
				},
			},
		},
		{
			name: "fieldNameStart",
			pos:  position{line: 496, col: 1, offset: 13532},
			expr: &charClassMatcher{
				pos:        position{line: 496, col: 18, offset: 13549},
				val:        "[A-Za-z_$]",
				chars:      []rune{'_', '$'},
				ranges:     []rune{'A', 'Z', 'a', 'z'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "fieldNameRest",
			pos:  position{line: 497, col: 1, offset: 13560},
			expr: &choiceExpr{
				pos: position{line: 497, col: 17, offset: 13576},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 497, col: 17, offset: 13576},
						name: "fieldNameStart",
					},
					&charClassMatcher{
						pos:        position{line: 497, col: 34, offset: 13593},
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
			pos:  position{line: 499, col: 1, offset: 13600},
			expr: &actionExpr{
				pos: position{line: 500, col: 4, offset: 13618},
				run: (*parser).callonfieldReference1,
				expr: &seqExpr{
					pos: position{line: 500, col: 4, offset: 13618},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 500, col: 4, offset: 13618},
							label: "base",
							expr: &ruleRefExpr{
								pos:  position{line: 500, col: 9, offset: 13623},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 500, col: 19, offset: 13633},
							label: "derefs",
							expr: &zeroOrMoreExpr{
								pos: position{line: 500, col: 26, offset: 13640},
								expr: &choiceExpr{
									pos: position{line: 501, col: 8, offset: 13649},
									alternatives: []interface{}{
										&actionExpr{
											pos: position{line: 501, col: 8, offset: 13649},
											run: (*parser).callonfieldReference8,
											expr: &seqExpr{
												pos: position{line: 501, col: 8, offset: 13649},
												exprs: []interface{}{
													&litMatcher{
														pos:        position{line: 501, col: 8, offset: 13649},
														val:        ".",
														ignoreCase: false,
													},
													&labeledExpr{
														pos:   position{line: 501, col: 12, offset: 13653},
														label: "field",
														expr: &ruleRefExpr{
															pos:  position{line: 501, col: 18, offset: 13659},
															name: "fieldName",
														},
													},
												},
											},
										},
										&actionExpr{
											pos: position{line: 502, col: 8, offset: 13740},
											run: (*parser).callonfieldReference13,
											expr: &seqExpr{
												pos: position{line: 502, col: 8, offset: 13740},
												exprs: []interface{}{
													&litMatcher{
														pos:        position{line: 502, col: 8, offset: 13740},
														val:        "[",
														ignoreCase: false,
													},
													&labeledExpr{
														pos:   position{line: 502, col: 12, offset: 13744},
														label: "index",
														expr: &ruleRefExpr{
															pos:  position{line: 502, col: 18, offset: 13750},
															name: "sinteger",
														},
													},
													&litMatcher{
														pos:        position{line: 502, col: 27, offset: 13759},
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
			pos:  position{line: 507, col: 1, offset: 13875},
			expr: &choiceExpr{
				pos: position{line: 508, col: 5, offset: 13889},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 508, col: 5, offset: 13889},
						run: (*parser).callonfieldExpr2,
						expr: &seqExpr{
							pos: position{line: 508, col: 5, offset: 13889},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 508, col: 5, offset: 13889},
									label: "op",
									expr: &ruleRefExpr{
										pos:  position{line: 508, col: 8, offset: 13892},
										name: "fieldOp",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 508, col: 16, offset: 13900},
									expr: &ruleRefExpr{
										pos:  position{line: 508, col: 16, offset: 13900},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 508, col: 19, offset: 13903},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 508, col: 23, offset: 13907},
									expr: &ruleRefExpr{
										pos:  position{line: 508, col: 23, offset: 13907},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 508, col: 26, offset: 13910},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 508, col: 32, offset: 13916},
										name: "fieldReference",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 508, col: 47, offset: 13931},
									expr: &ruleRefExpr{
										pos:  position{line: 508, col: 47, offset: 13931},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 508, col: 50, offset: 13934},
									val:        ")",
									ignoreCase: false,
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 511, col: 5, offset: 13998},
						name: "fieldReference",
					},
				},
			},
		},
		{
			name: "fieldOp",
			pos:  position{line: 513, col: 1, offset: 14014},
			expr: &actionExpr{
				pos: position{line: 514, col: 5, offset: 14026},
				run: (*parser).callonfieldOp1,
				expr: &litMatcher{
					pos:        position{line: 514, col: 5, offset: 14026},
					val:        "len",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldExprList",
			pos:  position{line: 516, col: 1, offset: 14056},
			expr: &actionExpr{
				pos: position{line: 517, col: 5, offset: 14074},
				run: (*parser).callonfieldExprList1,
				expr: &seqExpr{
					pos: position{line: 517, col: 5, offset: 14074},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 517, col: 5, offset: 14074},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 517, col: 11, offset: 14080},
								name: "fieldExpr",
							},
						},
						&labeledExpr{
							pos:   position{line: 517, col: 21, offset: 14090},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 517, col: 26, offset: 14095},
								expr: &seqExpr{
									pos: position{line: 517, col: 27, offset: 14096},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 517, col: 27, offset: 14096},
											expr: &ruleRefExpr{
												pos:  position{line: 517, col: 27, offset: 14096},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 517, col: 30, offset: 14099},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 517, col: 34, offset: 14103},
											expr: &ruleRefExpr{
												pos:  position{line: 517, col: 34, offset: 14103},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 517, col: 37, offset: 14106},
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
			pos:  position{line: 527, col: 1, offset: 14301},
			expr: &actionExpr{
				pos: position{line: 528, col: 5, offset: 14319},
				run: (*parser).callonfieldNameList1,
				expr: &seqExpr{
					pos: position{line: 528, col: 5, offset: 14319},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 528, col: 5, offset: 14319},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 528, col: 11, offset: 14325},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 528, col: 21, offset: 14335},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 528, col: 26, offset: 14340},
								expr: &seqExpr{
									pos: position{line: 528, col: 27, offset: 14341},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 528, col: 27, offset: 14341},
											expr: &ruleRefExpr{
												pos:  position{line: 528, col: 27, offset: 14341},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 528, col: 30, offset: 14344},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 528, col: 34, offset: 14348},
											expr: &ruleRefExpr{
												pos:  position{line: 528, col: 34, offset: 14348},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 528, col: 37, offset: 14351},
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
			pos:  position{line: 536, col: 1, offset: 14544},
			expr: &actionExpr{
				pos: position{line: 537, col: 5, offset: 14556},
				run: (*parser).calloncountOp1,
				expr: &litMatcher{
					pos:        position{line: 537, col: 5, offset: 14556},
					val:        "count",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldReducerOp",
			pos:  position{line: 539, col: 1, offset: 14590},
			expr: &choiceExpr{
				pos: position{line: 540, col: 5, offset: 14609},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 540, col: 5, offset: 14609},
						run: (*parser).callonfieldReducerOp2,
						expr: &litMatcher{
							pos:        position{line: 540, col: 5, offset: 14609},
							val:        "sum",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 541, col: 5, offset: 14643},
						run: (*parser).callonfieldReducerOp4,
						expr: &litMatcher{
							pos:        position{line: 541, col: 5, offset: 14643},
							val:        "avg",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 542, col: 5, offset: 14677},
						run: (*parser).callonfieldReducerOp6,
						expr: &litMatcher{
							pos:        position{line: 542, col: 5, offset: 14677},
							val:        "stdev",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 543, col: 5, offset: 14714},
						run: (*parser).callonfieldReducerOp8,
						expr: &litMatcher{
							pos:        position{line: 543, col: 5, offset: 14714},
							val:        "sd",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 544, col: 5, offset: 14750},
						run: (*parser).callonfieldReducerOp10,
						expr: &litMatcher{
							pos:        position{line: 544, col: 5, offset: 14750},
							val:        "var",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 545, col: 5, offset: 14784},
						run: (*parser).callonfieldReducerOp12,
						expr: &litMatcher{
							pos:        position{line: 545, col: 5, offset: 14784},
							val:        "entropy",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 546, col: 5, offset: 14825},
						run: (*parser).callonfieldReducerOp14,
						expr: &litMatcher{
							pos:        position{line: 546, col: 5, offset: 14825},
							val:        "min",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 547, col: 5, offset: 14859},
						run: (*parser).callonfieldReducerOp16,
						expr: &litMatcher{
							pos:        position{line: 547, col: 5, offset: 14859},
							val:        "max",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 548, col: 5, offset: 14893},
						run: (*parser).callonfieldReducerOp18,
						expr: &litMatcher{
							pos:        position{line: 548, col: 5, offset: 14893},
							val:        "first",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 549, col: 5, offset: 14931},
						run: (*parser).callonfieldReducerOp20,
						expr: &litMatcher{
							pos:        position{line: 549, col: 5, offset: 14931},
							val:        "last",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 550, col: 5, offset: 14967},
						run: (*parser).callonfieldReducerOp22,
						expr: &litMatcher{
							pos:        position{line: 550, col: 5, offset: 14967},
							val:        "countdistinct",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "paddedFieldName",
			pos:  position{line: 552, col: 1, offset: 15017},
			expr: &actionExpr{
				pos: position{line: 552, col: 19, offset: 15035},
				run: (*parser).callonpaddedFieldName1,
				expr: &seqExpr{
					pos: position{line: 552, col: 19, offset: 15035},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 552, col: 19, offset: 15035},
							expr: &ruleRefExpr{
								pos:  position{line: 552, col: 19, offset: 15035},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 552, col: 22, offset: 15038},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 552, col: 28, offset: 15044},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 552, col: 38, offset: 15054},
							expr: &ruleRefExpr{
								pos:  position{line: 552, col: 38, offset: 15054},
								name: "_",
							},
						},
					},
				},
			},
		},
		{
			name: "countReducer",
			pos:  position{line: 554, col: 1, offset: 15080},
			expr: &actionExpr{
				pos: position{line: 555, col: 5, offset: 15097},
				run: (*parser).calloncountReducer1,
				expr: &seqExpr{
					pos: position{line: 555, col: 5, offset: 15097},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 555, col: 5, offset: 15097},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 555, col: 8, offset: 15100},
								name: "countOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 555, col: 16, offset: 15108},
							expr: &ruleRefExpr{
								pos:  position{line: 555, col: 16, offset: 15108},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 555, col: 19, offset: 15111},
							val:        "(",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 555, col: 23, offset: 15115},
							label: "field",
							expr: &zeroOrOneExpr{
								pos: position{line: 555, col: 29, offset: 15121},
								expr: &ruleRefExpr{
									pos:  position{line: 555, col: 29, offset: 15121},
									name: "paddedFieldName",
								},
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 555, col: 47, offset: 15139},
							expr: &ruleRefExpr{
								pos:  position{line: 555, col: 47, offset: 15139},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 555, col: 50, offset: 15142},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "fieldReducer",
			pos:  position{line: 559, col: 1, offset: 15201},
			expr: &actionExpr{
				pos: position{line: 560, col: 5, offset: 15218},
				run: (*parser).callonfieldReducer1,
				expr: &seqExpr{
					pos: position{line: 560, col: 5, offset: 15218},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 560, col: 5, offset: 15218},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 560, col: 8, offset: 15221},
								name: "fieldReducerOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 560, col: 23, offset: 15236},
							expr: &ruleRefExpr{
								pos:  position{line: 560, col: 23, offset: 15236},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 560, col: 26, offset: 15239},
							val:        "(",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 560, col: 30, offset: 15243},
							expr: &ruleRefExpr{
								pos:  position{line: 560, col: 30, offset: 15243},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 560, col: 33, offset: 15246},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 560, col: 39, offset: 15252},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 560, col: 50, offset: 15263},
							expr: &ruleRefExpr{
								pos:  position{line: 560, col: 50, offset: 15263},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 560, col: 53, offset: 15266},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reducerProc",
			pos:  position{line: 564, col: 1, offset: 15333},
			expr: &actionExpr{
				pos: position{line: 565, col: 5, offset: 15349},
				run: (*parser).callonreducerProc1,
				expr: &seqExpr{
					pos: position{line: 565, col: 5, offset: 15349},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 565, col: 5, offset: 15349},
							label: "every",
							expr: &zeroOrOneExpr{
								pos: position{line: 565, col: 11, offset: 15355},
								expr: &seqExpr{
									pos: position{line: 565, col: 12, offset: 15356},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 565, col: 12, offset: 15356},
											name: "everyDur",
										},
										&ruleRefExpr{
											pos:  position{line: 565, col: 21, offset: 15365},
											name: "_",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 565, col: 25, offset: 15369},
							label: "reducers",
							expr: &ruleRefExpr{
								pos:  position{line: 565, col: 34, offset: 15378},
								name: "reducerList",
							},
						},
						&labeledExpr{
							pos:   position{line: 565, col: 46, offset: 15390},
							label: "keys",
							expr: &zeroOrOneExpr{
								pos: position{line: 565, col: 51, offset: 15395},
								expr: &seqExpr{
									pos: position{line: 565, col: 52, offset: 15396},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 565, col: 52, offset: 15396},
											name: "_",
										},
										&ruleRefExpr{
											pos:  position{line: 565, col: 54, offset: 15398},
											name: "groupBy",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 565, col: 64, offset: 15408},
							label: "limit",
							expr: &zeroOrOneExpr{
								pos: position{line: 565, col: 70, offset: 15414},
								expr: &ruleRefExpr{
									pos:  position{line: 565, col: 70, offset: 15414},
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
			pos:  position{line: 583, col: 1, offset: 15771},
			expr: &actionExpr{
				pos: position{line: 584, col: 5, offset: 15784},
				run: (*parser).callonasClause1,
				expr: &seqExpr{
					pos: position{line: 584, col: 5, offset: 15784},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 584, col: 5, offset: 15784},
							val:        "as",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 584, col: 11, offset: 15790},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 584, col: 13, offset: 15792},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 584, col: 15, offset: 15794},
								name: "fieldName",
							},
						},
					},
				},
			},
		},
		{
			name: "reducerExpr",
			pos:  position{line: 586, col: 1, offset: 15823},
			expr: &choiceExpr{
				pos: position{line: 587, col: 5, offset: 15839},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 587, col: 5, offset: 15839},
						run: (*parser).callonreducerExpr2,
						expr: &seqExpr{
							pos: position{line: 587, col: 5, offset: 15839},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 587, col: 5, offset: 15839},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 587, col: 11, offset: 15845},
										name: "fieldName",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 587, col: 21, offset: 15855},
									expr: &ruleRefExpr{
										pos:  position{line: 587, col: 21, offset: 15855},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 587, col: 24, offset: 15858},
									val:        "=",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 587, col: 28, offset: 15862},
									expr: &ruleRefExpr{
										pos:  position{line: 587, col: 28, offset: 15862},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 587, col: 31, offset: 15865},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 587, col: 33, offset: 15867},
										name: "reducer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 590, col: 5, offset: 15930},
						run: (*parser).callonreducerExpr13,
						expr: &seqExpr{
							pos: position{line: 590, col: 5, offset: 15930},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 590, col: 5, offset: 15930},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 590, col: 7, offset: 15932},
										name: "reducer",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 590, col: 15, offset: 15940},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 590, col: 17, offset: 15942},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 590, col: 23, offset: 15948},
										name: "asClause",
									},
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 593, col: 5, offset: 16012},
						name: "reducer",
					},
				},
			},
		},
		{
			name: "reducer",
			pos:  position{line: 595, col: 1, offset: 16021},
			expr: &choiceExpr{
				pos: position{line: 596, col: 5, offset: 16033},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 596, col: 5, offset: 16033},
						name: "countReducer",
					},
					&ruleRefExpr{
						pos:  position{line: 597, col: 5, offset: 16050},
						name: "fieldReducer",
					},
				},
			},
		},
		{
			name: "reducerList",
			pos:  position{line: 599, col: 1, offset: 16064},
			expr: &actionExpr{
				pos: position{line: 600, col: 5, offset: 16080},
				run: (*parser).callonreducerList1,
				expr: &seqExpr{
					pos: position{line: 600, col: 5, offset: 16080},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 600, col: 5, offset: 16080},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 600, col: 11, offset: 16086},
								name: "reducerExpr",
							},
						},
						&labeledExpr{
							pos:   position{line: 600, col: 23, offset: 16098},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 600, col: 28, offset: 16103},
								expr: &seqExpr{
									pos: position{line: 600, col: 29, offset: 16104},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 600, col: 29, offset: 16104},
											expr: &ruleRefExpr{
												pos:  position{line: 600, col: 29, offset: 16104},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 600, col: 32, offset: 16107},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 600, col: 36, offset: 16111},
											expr: &ruleRefExpr{
												pos:  position{line: 600, col: 36, offset: 16111},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 600, col: 39, offset: 16114},
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
			pos:  position{line: 608, col: 1, offset: 16311},
			expr: &choiceExpr{
				pos: position{line: 609, col: 5, offset: 16326},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 609, col: 5, offset: 16326},
						name: "sort",
					},
					&ruleRefExpr{
						pos:  position{line: 610, col: 5, offset: 16335},
						name: "top",
					},
					&ruleRefExpr{
						pos:  position{line: 611, col: 5, offset: 16343},
						name: "cut",
					},
					&ruleRefExpr{
						pos:  position{line: 612, col: 5, offset: 16351},
						name: "head",
					},
					&ruleRefExpr{
						pos:  position{line: 613, col: 5, offset: 16360},
						name: "tail",
					},
					&ruleRefExpr{
						pos:  position{line: 614, col: 5, offset: 16369},
						name: "filter",
					},
					&ruleRefExpr{
						pos:  position{line: 615, col: 5, offset: 16380},
						name: "uniq",
					},
				},
			},
		},
		{
			name: "sort",
			pos:  position{line: 617, col: 1, offset: 16386},
			expr: &choiceExpr{
				pos: position{line: 618, col: 5, offset: 16395},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 618, col: 5, offset: 16395},
						run: (*parser).callonsort2,
						expr: &seqExpr{
							pos: position{line: 618, col: 5, offset: 16395},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 618, col: 5, offset: 16395},
									val:        "sort",
									ignoreCase: true,
								},
								&labeledExpr{
									pos:   position{line: 618, col: 13, offset: 16403},
									label: "rev",
									expr: &zeroOrOneExpr{
										pos: position{line: 618, col: 17, offset: 16407},
										expr: &seqExpr{
											pos: position{line: 618, col: 18, offset: 16408},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 618, col: 18, offset: 16408},
													name: "_",
												},
												&litMatcher{
													pos:        position{line: 618, col: 20, offset: 16410},
													val:        "-r",
													ignoreCase: false,
												},
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 618, col: 27, offset: 16417},
									label: "limit",
									expr: &zeroOrOneExpr{
										pos: position{line: 618, col: 33, offset: 16423},
										expr: &ruleRefExpr{
											pos:  position{line: 618, col: 33, offset: 16423},
											name: "procLimitArg",
										},
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 618, col: 48, offset: 16438},
									expr: &ruleRefExpr{
										pos:  position{line: 618, col: 48, offset: 16438},
										name: "_",
									},
								},
								&notExpr{
									pos: position{line: 618, col: 51, offset: 16441},
									expr: &litMatcher{
										pos:        position{line: 618, col: 52, offset: 16442},
										val:        "-r",
										ignoreCase: false,
									},
								},
								&labeledExpr{
									pos:   position{line: 618, col: 57, offset: 16447},
									label: "list",
									expr: &zeroOrOneExpr{
										pos: position{line: 618, col: 62, offset: 16452},
										expr: &ruleRefExpr{
											pos:  position{line: 618, col: 63, offset: 16453},
											name: "fieldExprList",
										},
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 623, col: 5, offset: 16583},
						run: (*parser).callonsort20,
						expr: &seqExpr{
							pos: position{line: 623, col: 5, offset: 16583},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 623, col: 5, offset: 16583},
									val:        "sort",
									ignoreCase: true,
								},
								&labeledExpr{
									pos:   position{line: 623, col: 13, offset: 16591},
									label: "limit",
									expr: &zeroOrOneExpr{
										pos: position{line: 623, col: 19, offset: 16597},
										expr: &ruleRefExpr{
											pos:  position{line: 623, col: 19, offset: 16597},
											name: "procLimitArg",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 623, col: 33, offset: 16611},
									label: "rev",
									expr: &zeroOrOneExpr{
										pos: position{line: 623, col: 37, offset: 16615},
										expr: &seqExpr{
											pos: position{line: 623, col: 38, offset: 16616},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 623, col: 38, offset: 16616},
													name: "_",
												},
												&litMatcher{
													pos:        position{line: 623, col: 40, offset: 16618},
													val:        "-r",
													ignoreCase: false,
												},
											},
										},
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 623, col: 47, offset: 16625},
									expr: &ruleRefExpr{
										pos:  position{line: 623, col: 47, offset: 16625},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 623, col: 50, offset: 16628},
									label: "list",
									expr: &zeroOrOneExpr{
										pos: position{line: 623, col: 55, offset: 16633},
										expr: &ruleRefExpr{
											pos:  position{line: 623, col: 56, offset: 16634},
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
			pos:  position{line: 629, col: 1, offset: 16761},
			expr: &actionExpr{
				pos: position{line: 630, col: 5, offset: 16769},
				run: (*parser).callontop1,
				expr: &seqExpr{
					pos: position{line: 630, col: 5, offset: 16769},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 630, col: 5, offset: 16769},
							val:        "top",
							ignoreCase: true,
						},
						&labeledExpr{
							pos:   position{line: 630, col: 12, offset: 16776},
							label: "limit",
							expr: &zeroOrOneExpr{
								pos: position{line: 630, col: 18, offset: 16782},
								expr: &ruleRefExpr{
									pos:  position{line: 630, col: 18, offset: 16782},
									name: "procLimitArg",
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 630, col: 32, offset: 16796},
							label: "flush",
							expr: &zeroOrOneExpr{
								pos: position{line: 630, col: 38, offset: 16802},
								expr: &seqExpr{
									pos: position{line: 630, col: 39, offset: 16803},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 630, col: 39, offset: 16803},
											name: "_",
										},
										&litMatcher{
											pos:        position{line: 630, col: 41, offset: 16805},
											val:        "-flush",
											ignoreCase: false,
										},
									},
								},
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 630, col: 52, offset: 16816},
							expr: &ruleRefExpr{
								pos:  position{line: 630, col: 52, offset: 16816},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 630, col: 55, offset: 16819},
							label: "list",
							expr: &zeroOrOneExpr{
								pos: position{line: 630, col: 60, offset: 16824},
								expr: &ruleRefExpr{
									pos:  position{line: 630, col: 61, offset: 16825},
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
			pos:  position{line: 634, col: 1, offset: 16896},
			expr: &actionExpr{
				pos: position{line: 635, col: 5, offset: 16913},
				run: (*parser).callonprocLimitArg1,
				expr: &seqExpr{
					pos: position{line: 635, col: 5, offset: 16913},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 635, col: 5, offset: 16913},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 635, col: 7, offset: 16915},
							val:        "-limit",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 635, col: 16, offset: 16924},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 635, col: 18, offset: 16926},
							label: "limit",
							expr: &ruleRefExpr{
								pos:  position{line: 635, col: 24, offset: 16932},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "cut",
			pos:  position{line: 637, col: 1, offset: 16963},
			expr: &actionExpr{
				pos: position{line: 638, col: 5, offset: 16971},
				run: (*parser).calloncut1,
				expr: &seqExpr{
					pos: position{line: 638, col: 5, offset: 16971},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 638, col: 5, offset: 16971},
							val:        "cut",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 638, col: 12, offset: 16978},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 638, col: 14, offset: 16980},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 638, col: 19, offset: 16985},
								name: "fieldNameList",
							},
						},
					},
				},
			},
		},
		{
			name: "head",
			pos:  position{line: 639, col: 1, offset: 17033},
			expr: &choiceExpr{
				pos: position{line: 640, col: 5, offset: 17042},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 640, col: 5, offset: 17042},
						run: (*parser).callonhead2,
						expr: &seqExpr{
							pos: position{line: 640, col: 5, offset: 17042},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 640, col: 5, offset: 17042},
									val:        "head",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 640, col: 13, offset: 17050},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 640, col: 15, offset: 17052},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 640, col: 21, offset: 17058},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 641, col: 5, offset: 17106},
						run: (*parser).callonhead8,
						expr: &litMatcher{
							pos:        position{line: 641, col: 5, offset: 17106},
							val:        "head",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "tail",
			pos:  position{line: 642, col: 1, offset: 17146},
			expr: &choiceExpr{
				pos: position{line: 643, col: 5, offset: 17155},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 643, col: 5, offset: 17155},
						run: (*parser).callontail2,
						expr: &seqExpr{
							pos: position{line: 643, col: 5, offset: 17155},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 643, col: 5, offset: 17155},
									val:        "tail",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 643, col: 13, offset: 17163},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 643, col: 15, offset: 17165},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 643, col: 21, offset: 17171},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 644, col: 5, offset: 17219},
						run: (*parser).callontail8,
						expr: &litMatcher{
							pos:        position{line: 644, col: 5, offset: 17219},
							val:        "tail",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "filter",
			pos:  position{line: 646, col: 1, offset: 17260},
			expr: &actionExpr{
				pos: position{line: 647, col: 5, offset: 17271},
				run: (*parser).callonfilter1,
				expr: &seqExpr{
					pos: position{line: 647, col: 5, offset: 17271},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 647, col: 5, offset: 17271},
							val:        "filter",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 647, col: 15, offset: 17281},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 647, col: 17, offset: 17283},
							label: "expr",
							expr: &ruleRefExpr{
								pos:  position{line: 647, col: 22, offset: 17288},
								name: "searchExpr",
							},
						},
					},
				},
			},
		},
		{
			name: "uniq",
			pos:  position{line: 650, col: 1, offset: 17346},
			expr: &choiceExpr{
				pos: position{line: 651, col: 5, offset: 17355},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 651, col: 5, offset: 17355},
						run: (*parser).callonuniq2,
						expr: &seqExpr{
							pos: position{line: 651, col: 5, offset: 17355},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 651, col: 5, offset: 17355},
									val:        "uniq",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 651, col: 13, offset: 17363},
									name: "_",
								},
								&litMatcher{
									pos:        position{line: 651, col: 15, offset: 17365},
									val:        "-c",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 654, col: 5, offset: 17419},
						run: (*parser).callonuniq7,
						expr: &litMatcher{
							pos:        position{line: 654, col: 5, offset: 17419},
							val:        "uniq",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "duration",
			pos:  position{line: 658, col: 1, offset: 17474},
			expr: &choiceExpr{
				pos: position{line: 659, col: 5, offset: 17487},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 659, col: 5, offset: 17487},
						name: "seconds",
					},
					&ruleRefExpr{
						pos:  position{line: 660, col: 5, offset: 17499},
						name: "minutes",
					},
					&ruleRefExpr{
						pos:  position{line: 661, col: 5, offset: 17511},
						name: "hours",
					},
					&seqExpr{
						pos: position{line: 662, col: 5, offset: 17521},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 662, col: 5, offset: 17521},
								name: "hours",
							},
							&ruleRefExpr{
								pos:  position{line: 662, col: 11, offset: 17527},
								name: "_",
							},
							&litMatcher{
								pos:        position{line: 662, col: 13, offset: 17529},
								val:        "and",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 662, col: 19, offset: 17535},
								name: "_",
							},
							&ruleRefExpr{
								pos:  position{line: 662, col: 21, offset: 17537},
								name: "minutes",
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 663, col: 5, offset: 17549},
						name: "days",
					},
					&ruleRefExpr{
						pos:  position{line: 664, col: 5, offset: 17558},
						name: "weeks",
					},
				},
			},
		},
		{
			name: "sec_abbrev",
			pos:  position{line: 666, col: 1, offset: 17565},
			expr: &choiceExpr{
				pos: position{line: 667, col: 5, offset: 17580},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 667, col: 5, offset: 17580},
						val:        "seconds",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 668, col: 5, offset: 17594},
						val:        "second",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 669, col: 5, offset: 17607},
						val:        "secs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 670, col: 5, offset: 17618},
						val:        "sec",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 671, col: 5, offset: 17628},
						val:        "s",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "min_abbrev",
			pos:  position{line: 673, col: 1, offset: 17633},
			expr: &choiceExpr{
				pos: position{line: 674, col: 5, offset: 17648},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 674, col: 5, offset: 17648},
						val:        "minutes",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 675, col: 5, offset: 17662},
						val:        "minute",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 676, col: 5, offset: 17675},
						val:        "mins",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 677, col: 5, offset: 17686},
						val:        "min",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 678, col: 5, offset: 17696},
						val:        "m",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "hour_abbrev",
			pos:  position{line: 680, col: 1, offset: 17701},
			expr: &choiceExpr{
				pos: position{line: 681, col: 5, offset: 17717},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 681, col: 5, offset: 17717},
						val:        "hours",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 682, col: 5, offset: 17729},
						val:        "hrs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 683, col: 5, offset: 17739},
						val:        "hr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 684, col: 5, offset: 17748},
						val:        "h",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 685, col: 5, offset: 17756},
						val:        "hour",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "day_abbrev",
			pos:  position{line: 687, col: 1, offset: 17764},
			expr: &choiceExpr{
				pos: position{line: 687, col: 14, offset: 17777},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 687, col: 14, offset: 17777},
						val:        "days",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 687, col: 21, offset: 17784},
						val:        "day",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 687, col: 27, offset: 17790},
						val:        "d",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "week_abbrev",
			pos:  position{line: 688, col: 1, offset: 17794},
			expr: &choiceExpr{
				pos: position{line: 688, col: 15, offset: 17808},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 688, col: 15, offset: 17808},
						val:        "weeks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 688, col: 23, offset: 17816},
						val:        "week",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 688, col: 30, offset: 17823},
						val:        "wks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 688, col: 36, offset: 17829},
						val:        "wk",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 688, col: 41, offset: 17834},
						val:        "w",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "seconds",
			pos:  position{line: 690, col: 1, offset: 17839},
			expr: &choiceExpr{
				pos: position{line: 691, col: 5, offset: 17851},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 691, col: 5, offset: 17851},
						run: (*parser).callonseconds2,
						expr: &litMatcher{
							pos:        position{line: 691, col: 5, offset: 17851},
							val:        "second",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 692, col: 5, offset: 17896},
						run: (*parser).callonseconds4,
						expr: &seqExpr{
							pos: position{line: 692, col: 5, offset: 17896},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 692, col: 5, offset: 17896},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 692, col: 9, offset: 17900},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 692, col: 16, offset: 17907},
									expr: &ruleRefExpr{
										pos:  position{line: 692, col: 16, offset: 17907},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 692, col: 19, offset: 17910},
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
			pos:  position{line: 694, col: 1, offset: 17956},
			expr: &choiceExpr{
				pos: position{line: 695, col: 5, offset: 17968},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 695, col: 5, offset: 17968},
						run: (*parser).callonminutes2,
						expr: &litMatcher{
							pos:        position{line: 695, col: 5, offset: 17968},
							val:        "minute",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 696, col: 5, offset: 18014},
						run: (*parser).callonminutes4,
						expr: &seqExpr{
							pos: position{line: 696, col: 5, offset: 18014},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 696, col: 5, offset: 18014},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 696, col: 9, offset: 18018},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 696, col: 16, offset: 18025},
									expr: &ruleRefExpr{
										pos:  position{line: 696, col: 16, offset: 18025},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 696, col: 19, offset: 18028},
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
			pos:  position{line: 698, col: 1, offset: 18083},
			expr: &choiceExpr{
				pos: position{line: 699, col: 5, offset: 18093},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 699, col: 5, offset: 18093},
						run: (*parser).callonhours2,
						expr: &litMatcher{
							pos:        position{line: 699, col: 5, offset: 18093},
							val:        "hour",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 700, col: 5, offset: 18139},
						run: (*parser).callonhours4,
						expr: &seqExpr{
							pos: position{line: 700, col: 5, offset: 18139},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 700, col: 5, offset: 18139},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 700, col: 9, offset: 18143},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 700, col: 16, offset: 18150},
									expr: &ruleRefExpr{
										pos:  position{line: 700, col: 16, offset: 18150},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 700, col: 19, offset: 18153},
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
			pos:  position{line: 702, col: 1, offset: 18211},
			expr: &choiceExpr{
				pos: position{line: 703, col: 5, offset: 18220},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 703, col: 5, offset: 18220},
						run: (*parser).callondays2,
						expr: &litMatcher{
							pos:        position{line: 703, col: 5, offset: 18220},
							val:        "day",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 704, col: 5, offset: 18268},
						run: (*parser).callondays4,
						expr: &seqExpr{
							pos: position{line: 704, col: 5, offset: 18268},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 704, col: 5, offset: 18268},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 704, col: 9, offset: 18272},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 704, col: 16, offset: 18279},
									expr: &ruleRefExpr{
										pos:  position{line: 704, col: 16, offset: 18279},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 704, col: 19, offset: 18282},
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
			pos:  position{line: 706, col: 1, offset: 18342},
			expr: &actionExpr{
				pos: position{line: 707, col: 5, offset: 18352},
				run: (*parser).callonweeks1,
				expr: &seqExpr{
					pos: position{line: 707, col: 5, offset: 18352},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 707, col: 5, offset: 18352},
							label: "num",
							expr: &ruleRefExpr{
								pos:  position{line: 707, col: 9, offset: 18356},
								name: "number",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 707, col: 16, offset: 18363},
							expr: &ruleRefExpr{
								pos:  position{line: 707, col: 16, offset: 18363},
								name: "_",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 707, col: 19, offset: 18366},
							name: "week_abbrev",
						},
					},
				},
			},
		},
		{
			name: "number",
			pos:  position{line: 709, col: 1, offset: 18429},
			expr: &ruleRefExpr{
				pos:  position{line: 709, col: 10, offset: 18438},
				name: "integer",
			},
		},
		{
			name: "addr",
			pos:  position{line: 713, col: 1, offset: 18476},
			expr: &actionExpr{
				pos: position{line: 714, col: 5, offset: 18485},
				run: (*parser).callonaddr1,
				expr: &labeledExpr{
					pos:   position{line: 714, col: 5, offset: 18485},
					label: "a",
					expr: &seqExpr{
						pos: position{line: 714, col: 8, offset: 18488},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 714, col: 8, offset: 18488},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 714, col: 16, offset: 18496},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 714, col: 20, offset: 18500},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 714, col: 28, offset: 18508},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 714, col: 32, offset: 18512},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 714, col: 40, offset: 18520},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 714, col: 44, offset: 18524},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "port",
			pos:  position{line: 716, col: 1, offset: 18565},
			expr: &actionExpr{
				pos: position{line: 717, col: 5, offset: 18574},
				run: (*parser).callonport1,
				expr: &seqExpr{
					pos: position{line: 717, col: 5, offset: 18574},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 717, col: 5, offset: 18574},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 717, col: 9, offset: 18578},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 717, col: 11, offset: 18580},
								name: "sinteger",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6addr",
			pos:  position{line: 721, col: 1, offset: 18739},
			expr: &choiceExpr{
				pos: position{line: 722, col: 5, offset: 18751},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 722, col: 5, offset: 18751},
						run: (*parser).callonip6addr2,
						expr: &seqExpr{
							pos: position{line: 722, col: 5, offset: 18751},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 722, col: 5, offset: 18751},
									label: "a",
									expr: &oneOrMoreExpr{
										pos: position{line: 722, col: 7, offset: 18753},
										expr: &ruleRefExpr{
											pos:  position{line: 722, col: 8, offset: 18754},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 722, col: 20, offset: 18766},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 722, col: 22, offset: 18768},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 725, col: 5, offset: 18832},
						run: (*parser).callonip6addr9,
						expr: &seqExpr{
							pos: position{line: 725, col: 5, offset: 18832},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 725, col: 5, offset: 18832},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 725, col: 7, offset: 18834},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 725, col: 11, offset: 18838},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 725, col: 13, offset: 18840},
										expr: &ruleRefExpr{
											pos:  position{line: 725, col: 14, offset: 18841},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 725, col: 25, offset: 18852},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 725, col: 30, offset: 18857},
									label: "d",
									expr: &zeroOrMoreExpr{
										pos: position{line: 725, col: 32, offset: 18859},
										expr: &ruleRefExpr{
											pos:  position{line: 725, col: 33, offset: 18860},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 725, col: 45, offset: 18872},
									label: "e",
									expr: &ruleRefExpr{
										pos:  position{line: 725, col: 47, offset: 18874},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 728, col: 5, offset: 18973},
						run: (*parser).callonip6addr22,
						expr: &seqExpr{
							pos: position{line: 728, col: 5, offset: 18973},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 728, col: 5, offset: 18973},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 728, col: 10, offset: 18978},
									label: "a",
									expr: &zeroOrMoreExpr{
										pos: position{line: 728, col: 12, offset: 18980},
										expr: &ruleRefExpr{
											pos:  position{line: 728, col: 13, offset: 18981},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 728, col: 25, offset: 18993},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 728, col: 27, offset: 18995},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 731, col: 5, offset: 19066},
						run: (*parser).callonip6addr30,
						expr: &seqExpr{
							pos: position{line: 731, col: 5, offset: 19066},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 731, col: 5, offset: 19066},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 731, col: 7, offset: 19068},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 731, col: 11, offset: 19072},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 731, col: 13, offset: 19074},
										expr: &ruleRefExpr{
											pos:  position{line: 731, col: 14, offset: 19075},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 731, col: 25, offset: 19086},
									val:        "::",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 734, col: 5, offset: 19154},
						run: (*parser).callonip6addr38,
						expr: &litMatcher{
							pos:        position{line: 734, col: 5, offset: 19154},
							val:        "::",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "ip6tail",
			pos:  position{line: 738, col: 1, offset: 19191},
			expr: &choiceExpr{
				pos: position{line: 739, col: 5, offset: 19203},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 739, col: 5, offset: 19203},
						name: "addr",
					},
					&ruleRefExpr{
						pos:  position{line: 740, col: 5, offset: 19212},
						name: "h16",
					},
				},
			},
		},
		{
			name: "h_append",
			pos:  position{line: 742, col: 1, offset: 19217},
			expr: &actionExpr{
				pos: position{line: 742, col: 12, offset: 19228},
				run: (*parser).callonh_append1,
				expr: &seqExpr{
					pos: position{line: 742, col: 12, offset: 19228},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 742, col: 12, offset: 19228},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 742, col: 16, offset: 19232},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 742, col: 18, offset: 19234},
								name: "h16",
							},
						},
					},
				},
			},
		},
		{
			name: "h_prepend",
			pos:  position{line: 743, col: 1, offset: 19271},
			expr: &actionExpr{
				pos: position{line: 743, col: 13, offset: 19283},
				run: (*parser).callonh_prepend1,
				expr: &seqExpr{
					pos: position{line: 743, col: 13, offset: 19283},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 743, col: 13, offset: 19283},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 743, col: 15, offset: 19285},
								name: "h16",
							},
						},
						&litMatcher{
							pos:        position{line: 743, col: 19, offset: 19289},
							val:        ":",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "sub_addr",
			pos:  position{line: 745, col: 1, offset: 19327},
			expr: &choiceExpr{
				pos: position{line: 746, col: 5, offset: 19340},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 746, col: 5, offset: 19340},
						name: "addr",
					},
					&actionExpr{
						pos: position{line: 747, col: 5, offset: 19349},
						run: (*parser).callonsub_addr3,
						expr: &labeledExpr{
							pos:   position{line: 747, col: 5, offset: 19349},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 747, col: 8, offset: 19352},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 747, col: 8, offset: 19352},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 747, col: 16, offset: 19360},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 747, col: 20, offset: 19364},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 747, col: 28, offset: 19372},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 747, col: 32, offset: 19376},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 748, col: 5, offset: 19428},
						run: (*parser).callonsub_addr11,
						expr: &labeledExpr{
							pos:   position{line: 748, col: 5, offset: 19428},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 748, col: 8, offset: 19431},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 748, col: 8, offset: 19431},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 748, col: 16, offset: 19439},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 748, col: 20, offset: 19443},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 749, col: 5, offset: 19497},
						run: (*parser).callonsub_addr17,
						expr: &labeledExpr{
							pos:   position{line: 749, col: 5, offset: 19497},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 749, col: 7, offset: 19499},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "subnet",
			pos:  position{line: 751, col: 1, offset: 19550},
			expr: &actionExpr{
				pos: position{line: 752, col: 5, offset: 19561},
				run: (*parser).callonsubnet1,
				expr: &seqExpr{
					pos: position{line: 752, col: 5, offset: 19561},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 752, col: 5, offset: 19561},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 752, col: 7, offset: 19563},
								name: "sub_addr",
							},
						},
						&litMatcher{
							pos:        position{line: 752, col: 16, offset: 19572},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 752, col: 20, offset: 19576},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 752, col: 22, offset: 19578},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6subnet",
			pos:  position{line: 756, col: 1, offset: 19654},
			expr: &actionExpr{
				pos: position{line: 757, col: 5, offset: 19668},
				run: (*parser).callonip6subnet1,
				expr: &seqExpr{
					pos: position{line: 757, col: 5, offset: 19668},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 757, col: 5, offset: 19668},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 757, col: 7, offset: 19670},
								name: "ip6addr",
							},
						},
						&litMatcher{
							pos:        position{line: 757, col: 15, offset: 19678},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 757, col: 19, offset: 19682},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 757, col: 21, offset: 19684},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "integer",
			pos:  position{line: 761, col: 1, offset: 19750},
			expr: &actionExpr{
				pos: position{line: 762, col: 5, offset: 19762},
				run: (*parser).calloninteger1,
				expr: &labeledExpr{
					pos:   position{line: 762, col: 5, offset: 19762},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 762, col: 7, offset: 19764},
						name: "sinteger",
					},
				},
			},
		},
		{
			name: "sinteger",
			pos:  position{line: 766, col: 1, offset: 19808},
			expr: &actionExpr{
				pos: position{line: 767, col: 5, offset: 19821},
				run: (*parser).callonsinteger1,
				expr: &labeledExpr{
					pos:   position{line: 767, col: 5, offset: 19821},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 767, col: 11, offset: 19827},
						expr: &charClassMatcher{
							pos:        position{line: 767, col: 11, offset: 19827},
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
			pos:  position{line: 771, col: 1, offset: 19872},
			expr: &actionExpr{
				pos: position{line: 772, col: 5, offset: 19883},
				run: (*parser).callondouble1,
				expr: &labeledExpr{
					pos:   position{line: 772, col: 5, offset: 19883},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 772, col: 7, offset: 19885},
						name: "sdouble",
					},
				},
			},
		},
		{
			name: "sdouble",
			pos:  position{line: 776, col: 1, offset: 19932},
			expr: &choiceExpr{
				pos: position{line: 777, col: 5, offset: 19944},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 777, col: 5, offset: 19944},
						run: (*parser).callonsdouble2,
						expr: &seqExpr{
							pos: position{line: 777, col: 5, offset: 19944},
							exprs: []interface{}{
								&oneOrMoreExpr{
									pos: position{line: 777, col: 5, offset: 19944},
									expr: &ruleRefExpr{
										pos:  position{line: 777, col: 5, offset: 19944},
										name: "doubleInteger",
									},
								},
								&litMatcher{
									pos:        position{line: 777, col: 20, offset: 19959},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 777, col: 24, offset: 19963},
									expr: &ruleRefExpr{
										pos:  position{line: 777, col: 24, offset: 19963},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 777, col: 37, offset: 19976},
									expr: &ruleRefExpr{
										pos:  position{line: 777, col: 37, offset: 19976},
										name: "exponentPart",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 780, col: 5, offset: 20035},
						run: (*parser).callonsdouble11,
						expr: &seqExpr{
							pos: position{line: 780, col: 5, offset: 20035},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 780, col: 5, offset: 20035},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 780, col: 9, offset: 20039},
									expr: &ruleRefExpr{
										pos:  position{line: 780, col: 9, offset: 20039},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 780, col: 22, offset: 20052},
									expr: &ruleRefExpr{
										pos:  position{line: 780, col: 22, offset: 20052},
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
			pos:  position{line: 784, col: 1, offset: 20108},
			expr: &choiceExpr{
				pos: position{line: 785, col: 5, offset: 20126},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 785, col: 5, offset: 20126},
						val:        "0",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 786, col: 5, offset: 20134},
						exprs: []interface{}{
							&charClassMatcher{
								pos:        position{line: 786, col: 5, offset: 20134},
								val:        "[1-9]",
								ranges:     []rune{'1', '9'},
								ignoreCase: false,
								inverted:   false,
							},
							&zeroOrMoreExpr{
								pos: position{line: 786, col: 11, offset: 20140},
								expr: &charClassMatcher{
									pos:        position{line: 786, col: 11, offset: 20140},
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
			pos:  position{line: 788, col: 1, offset: 20148},
			expr: &charClassMatcher{
				pos:        position{line: 788, col: 15, offset: 20162},
				val:        "[0-9]",
				ranges:     []rune{'0', '9'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "signedInteger",
			pos:  position{line: 790, col: 1, offset: 20169},
			expr: &seqExpr{
				pos: position{line: 790, col: 17, offset: 20185},
				exprs: []interface{}{
					&zeroOrOneExpr{
						pos: position{line: 790, col: 17, offset: 20185},
						expr: &charClassMatcher{
							pos:        position{line: 790, col: 17, offset: 20185},
							val:        "[+-]",
							chars:      []rune{'+', '-'},
							ignoreCase: false,
							inverted:   false,
						},
					},
					&oneOrMoreExpr{
						pos: position{line: 790, col: 23, offset: 20191},
						expr: &ruleRefExpr{
							pos:  position{line: 790, col: 23, offset: 20191},
							name: "doubleDigit",
						},
					},
				},
			},
		},
		{
			name: "exponentPart",
			pos:  position{line: 792, col: 1, offset: 20205},
			expr: &seqExpr{
				pos: position{line: 792, col: 16, offset: 20220},
				exprs: []interface{}{
					&litMatcher{
						pos:        position{line: 792, col: 16, offset: 20220},
						val:        "e",
						ignoreCase: true,
					},
					&ruleRefExpr{
						pos:  position{line: 792, col: 21, offset: 20225},
						name: "signedInteger",
					},
				},
			},
		},
		{
			name: "h16",
			pos:  position{line: 794, col: 1, offset: 20240},
			expr: &actionExpr{
				pos: position{line: 794, col: 7, offset: 20246},
				run: (*parser).callonh161,
				expr: &labeledExpr{
					pos:   position{line: 794, col: 7, offset: 20246},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 794, col: 13, offset: 20252},
						expr: &ruleRefExpr{
							pos:  position{line: 794, col: 13, offset: 20252},
							name: "hexdigit",
						},
					},
				},
			},
		},
		{
			name: "hexdigit",
			pos:  position{line: 796, col: 1, offset: 20294},
			expr: &charClassMatcher{
				pos:        position{line: 796, col: 12, offset: 20305},
				val:        "[0-9a-fA-F]",
				ranges:     []rune{'0', '9', 'a', 'f', 'A', 'F'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name:        "boomWord",
			displayName: "\"boomWord\"",
			pos:         position{line: 798, col: 1, offset: 20318},
			expr: &actionExpr{
				pos: position{line: 798, col: 23, offset: 20340},
				run: (*parser).callonboomWord1,
				expr: &labeledExpr{
					pos:   position{line: 798, col: 23, offset: 20340},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 798, col: 29, offset: 20346},
						expr: &ruleRefExpr{
							pos:  position{line: 798, col: 29, offset: 20346},
							name: "boomWordPart",
						},
					},
				},
			},
		},
		{
			name: "boomWordPart",
			pos:  position{line: 800, col: 1, offset: 20392},
			expr: &seqExpr{
				pos: position{line: 801, col: 5, offset: 20409},
				exprs: []interface{}{
					&notExpr{
						pos: position{line: 801, col: 5, offset: 20409},
						expr: &choiceExpr{
							pos: position{line: 801, col: 7, offset: 20411},
							alternatives: []interface{}{
								&charClassMatcher{
									pos:        position{line: 801, col: 7, offset: 20411},
									val:        "[\\x00-\\x1F\\x5C(),!><=\\x22|\\x27;]",
									chars:      []rune{'\\', '(', ')', ',', '!', '>', '<', '=', '"', '|', '\'', ';'},
									ranges:     []rune{'\x00', '\x1f'},
									ignoreCase: false,
									inverted:   false,
								},
								&ruleRefExpr{
									pos:  position{line: 801, col: 42, offset: 20446},
									name: "ws",
								},
							},
						},
					},
					&anyMatcher{
						line: 801, col: 46, offset: 20450,
					},
				},
			},
		},
		{
			name: "quotedString",
			pos:  position{line: 803, col: 1, offset: 20453},
			expr: &choiceExpr{
				pos: position{line: 804, col: 5, offset: 20470},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 804, col: 5, offset: 20470},
						run: (*parser).callonquotedString2,
						expr: &seqExpr{
							pos: position{line: 804, col: 5, offset: 20470},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 804, col: 5, offset: 20470},
									val:        "\"",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 804, col: 9, offset: 20474},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 804, col: 11, offset: 20476},
										expr: &ruleRefExpr{
											pos:  position{line: 804, col: 11, offset: 20476},
											name: "doubleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 804, col: 29, offset: 20494},
									val:        "\"",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 805, col: 5, offset: 20531},
						run: (*parser).callonquotedString9,
						expr: &seqExpr{
							pos: position{line: 805, col: 5, offset: 20531},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 805, col: 5, offset: 20531},
									val:        "'",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 805, col: 9, offset: 20535},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 805, col: 11, offset: 20537},
										expr: &ruleRefExpr{
											pos:  position{line: 805, col: 11, offset: 20537},
											name: "singleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 805, col: 29, offset: 20555},
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
			pos:  position{line: 807, col: 1, offset: 20589},
			expr: &choiceExpr{
				pos: position{line: 808, col: 5, offset: 20610},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 808, col: 5, offset: 20610},
						run: (*parser).callondoubleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 808, col: 5, offset: 20610},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 808, col: 5, offset: 20610},
									expr: &choiceExpr{
										pos: position{line: 808, col: 7, offset: 20612},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 808, col: 7, offset: 20612},
												val:        "\"",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 808, col: 13, offset: 20618},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 808, col: 26, offset: 20631,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 809, col: 5, offset: 20668},
						run: (*parser).callondoubleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 809, col: 5, offset: 20668},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 809, col: 5, offset: 20668},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 809, col: 10, offset: 20673},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 809, col: 12, offset: 20675},
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
			pos:  position{line: 811, col: 1, offset: 20709},
			expr: &choiceExpr{
				pos: position{line: 812, col: 5, offset: 20730},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 812, col: 5, offset: 20730},
						run: (*parser).callonsingleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 812, col: 5, offset: 20730},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 812, col: 5, offset: 20730},
									expr: &choiceExpr{
										pos: position{line: 812, col: 7, offset: 20732},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 812, col: 7, offset: 20732},
												val:        "'",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 812, col: 13, offset: 20738},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 812, col: 26, offset: 20751,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 813, col: 5, offset: 20788},
						run: (*parser).callonsingleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 813, col: 5, offset: 20788},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 813, col: 5, offset: 20788},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 813, col: 10, offset: 20793},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 813, col: 12, offset: 20795},
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
			pos:  position{line: 815, col: 1, offset: 20829},
			expr: &choiceExpr{
				pos: position{line: 815, col: 18, offset: 20846},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 815, col: 18, offset: 20846},
						name: "singleCharEscape",
					},
					&ruleRefExpr{
						pos:  position{line: 815, col: 37, offset: 20865},
						name: "unicodeEscape",
					},
				},
			},
		},
		{
			name: "singleCharEscape",
			pos:  position{line: 817, col: 1, offset: 20880},
			expr: &choiceExpr{
				pos: position{line: 818, col: 5, offset: 20901},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 818, col: 5, offset: 20901},
						val:        "'",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 819, col: 5, offset: 20909},
						val:        "\"",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 820, col: 5, offset: 20917},
						val:        "\\",
						ignoreCase: false,
					},
					&actionExpr{
						pos: position{line: 821, col: 5, offset: 20926},
						run: (*parser).callonsingleCharEscape5,
						expr: &litMatcher{
							pos:        position{line: 821, col: 5, offset: 20926},
							val:        "b",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 822, col: 5, offset: 20955},
						run: (*parser).callonsingleCharEscape7,
						expr: &litMatcher{
							pos:        position{line: 822, col: 5, offset: 20955},
							val:        "f",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 823, col: 5, offset: 20984},
						run: (*parser).callonsingleCharEscape9,
						expr: &litMatcher{
							pos:        position{line: 823, col: 5, offset: 20984},
							val:        "n",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 824, col: 5, offset: 21013},
						run: (*parser).callonsingleCharEscape11,
						expr: &litMatcher{
							pos:        position{line: 824, col: 5, offset: 21013},
							val:        "r",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 825, col: 5, offset: 21042},
						run: (*parser).callonsingleCharEscape13,
						expr: &litMatcher{
							pos:        position{line: 825, col: 5, offset: 21042},
							val:        "t",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 826, col: 5, offset: 21071},
						run: (*parser).callonsingleCharEscape15,
						expr: &litMatcher{
							pos:        position{line: 826, col: 5, offset: 21071},
							val:        "v",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "unicodeEscape",
			pos:  position{line: 828, col: 1, offset: 21097},
			expr: &seqExpr{
				pos: position{line: 829, col: 5, offset: 21115},
				exprs: []interface{}{
					&litMatcher{
						pos:        position{line: 829, col: 5, offset: 21115},
						val:        "u",
						ignoreCase: false,
					},
					&ruleRefExpr{
						pos:  position{line: 829, col: 9, offset: 21119},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 829, col: 18, offset: 21128},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 829, col: 27, offset: 21137},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 829, col: 36, offset: 21146},
						name: "hexdigit",
					},
				},
			},
		},
		{
			name: "reString",
			pos:  position{line: 831, col: 1, offset: 21156},
			expr: &actionExpr{
				pos: position{line: 832, col: 5, offset: 21169},
				run: (*parser).callonreString1,
				expr: &seqExpr{
					pos: position{line: 832, col: 5, offset: 21169},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 832, col: 5, offset: 21169},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 832, col: 9, offset: 21173},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 832, col: 11, offset: 21175},
								name: "reBody",
							},
						},
						&litMatcher{
							pos:        position{line: 832, col: 18, offset: 21182},
							val:        "/",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reBody",
			pos:  position{line: 834, col: 1, offset: 21205},
			expr: &actionExpr{
				pos: position{line: 835, col: 5, offset: 21216},
				run: (*parser).callonreBody1,
				expr: &oneOrMoreExpr{
					pos: position{line: 835, col: 5, offset: 21216},
					expr: &choiceExpr{
						pos: position{line: 835, col: 6, offset: 21217},
						alternatives: []interface{}{
							&charClassMatcher{
								pos:        position{line: 835, col: 6, offset: 21217},
								val:        "[^/\\\\]",
								chars:      []rune{'/', '\\'},
								ignoreCase: false,
								inverted:   true,
							},
							&litMatcher{
								pos:        position{line: 835, col: 13, offset: 21224},
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
			pos:  position{line: 837, col: 1, offset: 21264},
			expr: &charClassMatcher{
				pos:        position{line: 838, col: 5, offset: 21280},
				val:        "[\\x00-\\x1f\\\\]",
				chars:      []rune{'\\'},
				ranges:     []rune{'\x00', '\x1f'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "ws",
			pos:  position{line: 840, col: 1, offset: 21295},
			expr: &choiceExpr{
				pos: position{line: 841, col: 5, offset: 21302},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 841, col: 5, offset: 21302},
						val:        "\t",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 842, col: 5, offset: 21311},
						val:        "\v",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 843, col: 5, offset: 21320},
						val:        "\f",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 844, col: 5, offset: 21329},
						val:        " ",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 845, col: 5, offset: 21337},
						val:        "\u00a0",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 846, col: 5, offset: 21350},
						val:        "\ufeff",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name:        "_",
			displayName: "\"whitespace\"",
			pos:         position{line: 848, col: 1, offset: 21360},
			expr: &oneOrMoreExpr{
				pos: position{line: 848, col: 18, offset: 21377},
				expr: &ruleRefExpr{
					pos:  position{line: 848, col: 18, offset: 21377},
					name: "ws",
				},
			},
		},
		{
			name: "EOF",
			pos:  position{line: 850, col: 1, offset: 21382},
			expr: &notExpr{
				pos: position{line: 850, col: 7, offset: 21388},
				expr: &anyMatcher{
					line: 850, col: 8, offset: 21389,
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

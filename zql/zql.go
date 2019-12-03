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

func makeCompareAny(comparatorIn, recurseIn, valueIn interface{}) *ast.CompareAny {
	comparator := comparatorIn.(string)
	recurse := recurseIn.(bool)
	value := valueIn.(*ast.TypedValue)
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
			pos:  position{line: 314, col: 1, offset: 9034},
			expr: &actionExpr{
				pos: position{line: 314, col: 9, offset: 9042},
				run: (*parser).callonstart1,
				expr: &seqExpr{
					pos: position{line: 314, col: 9, offset: 9042},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 314, col: 9, offset: 9042},
							expr: &ruleRefExpr{
								pos:  position{line: 314, col: 9, offset: 9042},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 314, col: 12, offset: 9045},
							label: "ast",
							expr: &ruleRefExpr{
								pos:  position{line: 314, col: 16, offset: 9049},
								name: "boomCommand",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 314, col: 28, offset: 9061},
							expr: &ruleRefExpr{
								pos:  position{line: 314, col: 28, offset: 9061},
								name: "_",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 314, col: 31, offset: 9064},
							name: "EOF",
						},
					},
				},
			},
		},
		{
			name: "boomCommand",
			pos:  position{line: 316, col: 1, offset: 9089},
			expr: &choiceExpr{
				pos: position{line: 317, col: 5, offset: 9105},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 317, col: 5, offset: 9105},
						run: (*parser).callonboomCommand2,
						expr: &labeledExpr{
							pos:   position{line: 317, col: 5, offset: 9105},
							label: "procs",
							expr: &ruleRefExpr{
								pos:  position{line: 317, col: 11, offset: 9111},
								name: "procChain",
							},
						},
					},
					&actionExpr{
						pos: position{line: 321, col: 5, offset: 9284},
						run: (*parser).callonboomCommand5,
						expr: &seqExpr{
							pos: position{line: 321, col: 5, offset: 9284},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 321, col: 5, offset: 9284},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 321, col: 7, offset: 9286},
										name: "search",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 321, col: 14, offset: 9293},
									expr: &ruleRefExpr{
										pos:  position{line: 321, col: 14, offset: 9293},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 321, col: 17, offset: 9296},
									label: "rest",
									expr: &zeroOrMoreExpr{
										pos: position{line: 321, col: 22, offset: 9301},
										expr: &ruleRefExpr{
											pos:  position{line: 321, col: 22, offset: 9301},
											name: "chainedProc",
										},
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 328, col: 5, offset: 9511},
						run: (*parser).callonboomCommand14,
						expr: &labeledExpr{
							pos:   position{line: 328, col: 5, offset: 9511},
							label: "s",
							expr: &ruleRefExpr{
								pos:  position{line: 328, col: 7, offset: 9513},
								name: "search",
							},
						},
					},
				},
			},
		},
		{
			name: "procChain",
			pos:  position{line: 332, col: 1, offset: 9584},
			expr: &actionExpr{
				pos: position{line: 333, col: 5, offset: 9598},
				run: (*parser).callonprocChain1,
				expr: &seqExpr{
					pos: position{line: 333, col: 5, offset: 9598},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 333, col: 5, offset: 9598},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 333, col: 11, offset: 9604},
								name: "proc",
							},
						},
						&labeledExpr{
							pos:   position{line: 333, col: 16, offset: 9609},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 333, col: 21, offset: 9614},
								expr: &ruleRefExpr{
									pos:  position{line: 333, col: 21, offset: 9614},
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
			pos:  position{line: 341, col: 1, offset: 9800},
			expr: &actionExpr{
				pos: position{line: 341, col: 15, offset: 9814},
				run: (*parser).callonchainedProc1,
				expr: &seqExpr{
					pos: position{line: 341, col: 15, offset: 9814},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 341, col: 15, offset: 9814},
							expr: &ruleRefExpr{
								pos:  position{line: 341, col: 15, offset: 9814},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 341, col: 18, offset: 9817},
							val:        "|",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 341, col: 22, offset: 9821},
							expr: &ruleRefExpr{
								pos:  position{line: 341, col: 22, offset: 9821},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 341, col: 25, offset: 9824},
							label: "p",
							expr: &ruleRefExpr{
								pos:  position{line: 341, col: 27, offset: 9826},
								name: "proc",
							},
						},
					},
				},
			},
		},
		{
			name: "search",
			pos:  position{line: 343, col: 1, offset: 9850},
			expr: &actionExpr{
				pos: position{line: 344, col: 5, offset: 9861},
				run: (*parser).callonsearch1,
				expr: &labeledExpr{
					pos:   position{line: 344, col: 5, offset: 9861},
					label: "expr",
					expr: &ruleRefExpr{
						pos:  position{line: 344, col: 10, offset: 9866},
						name: "searchExpr",
					},
				},
			},
		},
		{
			name: "searchExpr",
			pos:  position{line: 348, col: 1, offset: 9925},
			expr: &actionExpr{
				pos: position{line: 349, col: 5, offset: 9940},
				run: (*parser).callonsearchExpr1,
				expr: &seqExpr{
					pos: position{line: 349, col: 5, offset: 9940},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 349, col: 5, offset: 9940},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 349, col: 11, offset: 9946},
								name: "searchTerm",
							},
						},
						&labeledExpr{
							pos:   position{line: 349, col: 22, offset: 9957},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 349, col: 27, offset: 9962},
								expr: &ruleRefExpr{
									pos:  position{line: 349, col: 27, offset: 9962},
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
			pos:  position{line: 353, col: 1, offset: 10030},
			expr: &actionExpr{
				pos: position{line: 353, col: 18, offset: 10047},
				run: (*parser).callonoredSearchTerm1,
				expr: &seqExpr{
					pos: position{line: 353, col: 18, offset: 10047},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 353, col: 18, offset: 10047},
							name: "_",
						},
						&ruleRefExpr{
							pos:  position{line: 353, col: 20, offset: 10049},
							name: "orToken",
						},
						&ruleRefExpr{
							pos:  position{line: 353, col: 28, offset: 10057},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 353, col: 30, offset: 10059},
							label: "t",
							expr: &ruleRefExpr{
								pos:  position{line: 353, col: 32, offset: 10061},
								name: "searchTerm",
							},
						},
					},
				},
			},
		},
		{
			name: "searchTerm",
			pos:  position{line: 355, col: 1, offset: 10091},
			expr: &actionExpr{
				pos: position{line: 356, col: 5, offset: 10106},
				run: (*parser).callonsearchTerm1,
				expr: &seqExpr{
					pos: position{line: 356, col: 5, offset: 10106},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 356, col: 5, offset: 10106},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 356, col: 11, offset: 10112},
								name: "searchFactor",
							},
						},
						&labeledExpr{
							pos:   position{line: 356, col: 24, offset: 10125},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 356, col: 29, offset: 10130},
								expr: &ruleRefExpr{
									pos:  position{line: 356, col: 29, offset: 10130},
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
			pos:  position{line: 360, col: 1, offset: 10200},
			expr: &actionExpr{
				pos: position{line: 360, col: 19, offset: 10218},
				run: (*parser).callonandedSearchTerm1,
				expr: &seqExpr{
					pos: position{line: 360, col: 19, offset: 10218},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 360, col: 19, offset: 10218},
							name: "_",
						},
						&zeroOrOneExpr{
							pos: position{line: 360, col: 21, offset: 10220},
							expr: &seqExpr{
								pos: position{line: 360, col: 22, offset: 10221},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 360, col: 22, offset: 10221},
										name: "andToken",
									},
									&ruleRefExpr{
										pos:  position{line: 360, col: 31, offset: 10230},
										name: "_",
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 360, col: 35, offset: 10234},
							label: "f",
							expr: &ruleRefExpr{
								pos:  position{line: 360, col: 37, offset: 10236},
								name: "searchFactor",
							},
						},
					},
				},
			},
		},
		{
			name: "searchFactor",
			pos:  position{line: 362, col: 1, offset: 10268},
			expr: &choiceExpr{
				pos: position{line: 363, col: 5, offset: 10285},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 363, col: 5, offset: 10285},
						run: (*parser).callonsearchFactor2,
						expr: &seqExpr{
							pos: position{line: 363, col: 5, offset: 10285},
							exprs: []interface{}{
								&choiceExpr{
									pos: position{line: 363, col: 6, offset: 10286},
									alternatives: []interface{}{
										&seqExpr{
											pos: position{line: 363, col: 6, offset: 10286},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 363, col: 6, offset: 10286},
													name: "notToken",
												},
												&ruleRefExpr{
													pos:  position{line: 363, col: 15, offset: 10295},
													name: "_",
												},
											},
										},
										&seqExpr{
											pos: position{line: 363, col: 19, offset: 10299},
											exprs: []interface{}{
												&litMatcher{
													pos:        position{line: 363, col: 19, offset: 10299},
													val:        "!",
													ignoreCase: false,
												},
												&zeroOrOneExpr{
													pos: position{line: 363, col: 23, offset: 10303},
													expr: &ruleRefExpr{
														pos:  position{line: 363, col: 23, offset: 10303},
														name: "_",
													},
												},
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 363, col: 27, offset: 10307},
									label: "e",
									expr: &ruleRefExpr{
										pos:  position{line: 363, col: 29, offset: 10309},
										name: "searchExpr",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 366, col: 5, offset: 10368},
						run: (*parser).callonsearchFactor14,
						expr: &seqExpr{
							pos: position{line: 366, col: 5, offset: 10368},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 366, col: 5, offset: 10368},
									expr: &litMatcher{
										pos:        position{line: 366, col: 7, offset: 10370},
										val:        "-",
										ignoreCase: false,
									},
								},
								&labeledExpr{
									pos:   position{line: 366, col: 12, offset: 10375},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 366, col: 14, offset: 10377},
										name: "searchPred",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 367, col: 5, offset: 10410},
						run: (*parser).callonsearchFactor20,
						expr: &seqExpr{
							pos: position{line: 367, col: 5, offset: 10410},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 367, col: 5, offset: 10410},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 367, col: 9, offset: 10414},
									expr: &ruleRefExpr{
										pos:  position{line: 367, col: 9, offset: 10414},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 367, col: 12, offset: 10417},
									label: "expr",
									expr: &ruleRefExpr{
										pos:  position{line: 367, col: 17, offset: 10422},
										name: "searchExpr",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 367, col: 28, offset: 10433},
									expr: &ruleRefExpr{
										pos:  position{line: 367, col: 28, offset: 10433},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 367, col: 31, offset: 10436},
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
			pos:  position{line: 369, col: 1, offset: 10462},
			expr: &choiceExpr{
				pos: position{line: 370, col: 5, offset: 10477},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 370, col: 5, offset: 10477},
						run: (*parser).callonsearchPred2,
						expr: &seqExpr{
							pos: position{line: 370, col: 5, offset: 10477},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 370, col: 5, offset: 10477},
									val:        "*",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 370, col: 9, offset: 10481},
									expr: &ruleRefExpr{
										pos:  position{line: 370, col: 9, offset: 10481},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 370, col: 12, offset: 10484},
									label: "fieldComparator",
									expr: &ruleRefExpr{
										pos:  position{line: 370, col: 28, offset: 10500},
										name: "equalityToken",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 370, col: 42, offset: 10514},
									expr: &ruleRefExpr{
										pos:  position{line: 370, col: 42, offset: 10514},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 370, col: 45, offset: 10517},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 370, col: 47, offset: 10519},
										name: "searchValue",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 373, col: 5, offset: 10603},
						run: (*parser).callonsearchPred13,
						expr: &seqExpr{
							pos: position{line: 373, col: 5, offset: 10603},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 373, col: 5, offset: 10603},
									val:        "**",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 373, col: 10, offset: 10608},
									expr: &ruleRefExpr{
										pos:  position{line: 373, col: 10, offset: 10608},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 373, col: 13, offset: 10611},
									label: "fieldComparator",
									expr: &ruleRefExpr{
										pos:  position{line: 373, col: 29, offset: 10627},
										name: "equalityToken",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 373, col: 43, offset: 10641},
									expr: &ruleRefExpr{
										pos:  position{line: 373, col: 43, offset: 10641},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 373, col: 46, offset: 10644},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 373, col: 48, offset: 10646},
										name: "searchValue",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 376, col: 5, offset: 10729},
						run: (*parser).callonsearchPred24,
						expr: &litMatcher{
							pos:        position{line: 376, col: 5, offset: 10729},
							val:        "*",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 379, col: 5, offset: 10788},
						run: (*parser).callonsearchPred26,
						expr: &seqExpr{
							pos: position{line: 379, col: 5, offset: 10788},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 379, col: 5, offset: 10788},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 379, col: 7, offset: 10790},
										name: "fieldExpr",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 379, col: 17, offset: 10800},
									expr: &ruleRefExpr{
										pos:  position{line: 379, col: 17, offset: 10800},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 379, col: 20, offset: 10803},
									label: "fieldComparator",
									expr: &ruleRefExpr{
										pos:  position{line: 379, col: 36, offset: 10819},
										name: "equalityToken",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 379, col: 50, offset: 10833},
									expr: &ruleRefExpr{
										pos:  position{line: 379, col: 50, offset: 10833},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 379, col: 53, offset: 10836},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 379, col: 55, offset: 10838},
										name: "searchValue",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 382, col: 5, offset: 10920},
						run: (*parser).callonsearchPred38,
						expr: &seqExpr{
							pos: position{line: 382, col: 5, offset: 10920},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 382, col: 5, offset: 10920},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 382, col: 7, offset: 10922},
										name: "searchValue",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 382, col: 19, offset: 10934},
									expr: &ruleRefExpr{
										pos:  position{line: 382, col: 19, offset: 10934},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 382, col: 22, offset: 10937},
									name: "inToken",
								},
								&zeroOrOneExpr{
									pos: position{line: 382, col: 30, offset: 10945},
									expr: &ruleRefExpr{
										pos:  position{line: 382, col: 30, offset: 10945},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 382, col: 33, offset: 10948},
									val:        "*",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 385, col: 5, offset: 11013},
						run: (*parser).callonsearchPred48,
						expr: &seqExpr{
							pos: position{line: 385, col: 5, offset: 11013},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 385, col: 5, offset: 11013},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 385, col: 7, offset: 11015},
										name: "searchValue",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 385, col: 19, offset: 11027},
									expr: &ruleRefExpr{
										pos:  position{line: 385, col: 19, offset: 11027},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 385, col: 22, offset: 11030},
									name: "inToken",
								},
								&zeroOrOneExpr{
									pos: position{line: 385, col: 30, offset: 11038},
									expr: &ruleRefExpr{
										pos:  position{line: 385, col: 30, offset: 11038},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 385, col: 33, offset: 11041},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 385, col: 35, offset: 11043},
										name: "fieldReference",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 388, col: 5, offset: 11117},
						run: (*parser).callonsearchPred59,
						expr: &labeledExpr{
							pos:   position{line: 388, col: 5, offset: 11117},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 388, col: 7, offset: 11119},
								name: "searchValue",
							},
						},
					},
				},
			},
		},
		{
			name: "searchValue",
			pos:  position{line: 397, col: 1, offset: 11425},
			expr: &choiceExpr{
				pos: position{line: 398, col: 5, offset: 11441},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 398, col: 5, offset: 11441},
						run: (*parser).callonsearchValue2,
						expr: &labeledExpr{
							pos:   position{line: 398, col: 5, offset: 11441},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 398, col: 7, offset: 11443},
								name: "quotedString",
							},
						},
					},
					&actionExpr{
						pos: position{line: 401, col: 5, offset: 11514},
						run: (*parser).callonsearchValue5,
						expr: &labeledExpr{
							pos:   position{line: 401, col: 5, offset: 11514},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 401, col: 7, offset: 11516},
								name: "reString",
							},
						},
					},
					&actionExpr{
						pos: position{line: 404, col: 5, offset: 11583},
						run: (*parser).callonsearchValue8,
						expr: &labeledExpr{
							pos:   position{line: 404, col: 5, offset: 11583},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 404, col: 7, offset: 11585},
								name: "port",
							},
						},
					},
					&actionExpr{
						pos: position{line: 407, col: 5, offset: 11644},
						run: (*parser).callonsearchValue11,
						expr: &labeledExpr{
							pos:   position{line: 407, col: 5, offset: 11644},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 407, col: 7, offset: 11646},
								name: "ip6subnet",
							},
						},
					},
					&actionExpr{
						pos: position{line: 410, col: 5, offset: 11714},
						run: (*parser).callonsearchValue14,
						expr: &labeledExpr{
							pos:   position{line: 410, col: 5, offset: 11714},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 410, col: 7, offset: 11716},
								name: "ip6addr",
							},
						},
					},
					&actionExpr{
						pos: position{line: 413, col: 5, offset: 11780},
						run: (*parser).callonsearchValue17,
						expr: &labeledExpr{
							pos:   position{line: 413, col: 5, offset: 11780},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 413, col: 7, offset: 11782},
								name: "subnet",
							},
						},
					},
					&actionExpr{
						pos: position{line: 416, col: 5, offset: 11847},
						run: (*parser).callonsearchValue20,
						expr: &labeledExpr{
							pos:   position{line: 416, col: 5, offset: 11847},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 416, col: 7, offset: 11849},
								name: "addr",
							},
						},
					},
					&actionExpr{
						pos: position{line: 419, col: 5, offset: 11910},
						run: (*parser).callonsearchValue23,
						expr: &labeledExpr{
							pos:   position{line: 419, col: 5, offset: 11910},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 419, col: 7, offset: 11912},
								name: "sdouble",
							},
						},
					},
					&actionExpr{
						pos: position{line: 422, col: 5, offset: 11978},
						run: (*parser).callonsearchValue26,
						expr: &seqExpr{
							pos: position{line: 422, col: 5, offset: 11978},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 422, col: 5, offset: 11978},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 422, col: 7, offset: 11980},
										name: "sinteger",
									},
								},
								&notExpr{
									pos: position{line: 422, col: 16, offset: 11989},
									expr: &ruleRefExpr{
										pos:  position{line: 422, col: 17, offset: 11990},
										name: "boomWord",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 425, col: 5, offset: 12054},
						run: (*parser).callonsearchValue32,
						expr: &seqExpr{
							pos: position{line: 425, col: 5, offset: 12054},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 425, col: 5, offset: 12054},
									expr: &seqExpr{
										pos: position{line: 425, col: 7, offset: 12056},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 425, col: 7, offset: 12056},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 425, col: 22, offset: 12071},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 425, col: 25, offset: 12074},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 425, col: 27, offset: 12076},
										name: "booleanLiteral",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 426, col: 5, offset: 12113},
						run: (*parser).callonsearchValue40,
						expr: &seqExpr{
							pos: position{line: 426, col: 5, offset: 12113},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 426, col: 5, offset: 12113},
									expr: &seqExpr{
										pos: position{line: 426, col: 7, offset: 12115},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 426, col: 7, offset: 12115},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 426, col: 22, offset: 12130},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 426, col: 25, offset: 12133},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 426, col: 27, offset: 12135},
										name: "unsetLiteral",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 427, col: 5, offset: 12170},
						run: (*parser).callonsearchValue48,
						expr: &seqExpr{
							pos: position{line: 427, col: 5, offset: 12170},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 427, col: 5, offset: 12170},
									expr: &seqExpr{
										pos: position{line: 427, col: 7, offset: 12172},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 427, col: 7, offset: 12172},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 427, col: 22, offset: 12187},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 427, col: 25, offset: 12190},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 427, col: 27, offset: 12192},
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
			pos:  position{line: 435, col: 1, offset: 12416},
			expr: &choiceExpr{
				pos: position{line: 436, col: 5, offset: 12435},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 436, col: 5, offset: 12435},
						name: "andToken",
					},
					&ruleRefExpr{
						pos:  position{line: 437, col: 5, offset: 12448},
						name: "orToken",
					},
					&ruleRefExpr{
						pos:  position{line: 438, col: 5, offset: 12460},
						name: "inToken",
					},
				},
			},
		},
		{
			name: "booleanLiteral",
			pos:  position{line: 440, col: 1, offset: 12469},
			expr: &choiceExpr{
				pos: position{line: 441, col: 5, offset: 12488},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 441, col: 5, offset: 12488},
						run: (*parser).callonbooleanLiteral2,
						expr: &litMatcher{
							pos:        position{line: 441, col: 5, offset: 12488},
							val:        "true",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 442, col: 5, offset: 12556},
						run: (*parser).callonbooleanLiteral4,
						expr: &litMatcher{
							pos:        position{line: 442, col: 5, offset: 12556},
							val:        "false",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "unsetLiteral",
			pos:  position{line: 444, col: 1, offset: 12622},
			expr: &actionExpr{
				pos: position{line: 445, col: 5, offset: 12639},
				run: (*parser).callonunsetLiteral1,
				expr: &litMatcher{
					pos:        position{line: 445, col: 5, offset: 12639},
					val:        "nil",
					ignoreCase: false,
				},
			},
		},
		{
			name: "procList",
			pos:  position{line: 447, col: 1, offset: 12700},
			expr: &actionExpr{
				pos: position{line: 448, col: 5, offset: 12713},
				run: (*parser).callonprocList1,
				expr: &seqExpr{
					pos: position{line: 448, col: 5, offset: 12713},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 448, col: 5, offset: 12713},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 448, col: 11, offset: 12719},
								name: "procChain",
							},
						},
						&labeledExpr{
							pos:   position{line: 448, col: 21, offset: 12729},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 448, col: 26, offset: 12734},
								expr: &ruleRefExpr{
									pos:  position{line: 448, col: 26, offset: 12734},
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
			pos:  position{line: 457, col: 1, offset: 12958},
			expr: &actionExpr{
				pos: position{line: 458, col: 5, offset: 12976},
				run: (*parser).callonparallelChain1,
				expr: &seqExpr{
					pos: position{line: 458, col: 5, offset: 12976},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 458, col: 5, offset: 12976},
							expr: &ruleRefExpr{
								pos:  position{line: 458, col: 5, offset: 12976},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 458, col: 8, offset: 12979},
							val:        ";",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 458, col: 12, offset: 12983},
							expr: &ruleRefExpr{
								pos:  position{line: 458, col: 12, offset: 12983},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 458, col: 15, offset: 12986},
							label: "ch",
							expr: &ruleRefExpr{
								pos:  position{line: 458, col: 18, offset: 12989},
								name: "procChain",
							},
						},
					},
				},
			},
		},
		{
			name: "proc",
			pos:  position{line: 460, col: 1, offset: 13039},
			expr: &choiceExpr{
				pos: position{line: 461, col: 5, offset: 13048},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 461, col: 5, offset: 13048},
						name: "simpleProc",
					},
					&ruleRefExpr{
						pos:  position{line: 462, col: 5, offset: 13063},
						name: "reducerProc",
					},
					&actionExpr{
						pos: position{line: 463, col: 5, offset: 13079},
						run: (*parser).callonproc4,
						expr: &seqExpr{
							pos: position{line: 463, col: 5, offset: 13079},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 463, col: 5, offset: 13079},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 463, col: 9, offset: 13083},
									expr: &ruleRefExpr{
										pos:  position{line: 463, col: 9, offset: 13083},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 463, col: 12, offset: 13086},
									label: "proc",
									expr: &ruleRefExpr{
										pos:  position{line: 463, col: 17, offset: 13091},
										name: "procList",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 463, col: 26, offset: 13100},
									expr: &ruleRefExpr{
										pos:  position{line: 463, col: 26, offset: 13100},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 463, col: 29, offset: 13103},
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
			pos:  position{line: 467, col: 1, offset: 13139},
			expr: &actionExpr{
				pos: position{line: 468, col: 5, offset: 13151},
				run: (*parser).callongroupBy1,
				expr: &seqExpr{
					pos: position{line: 468, col: 5, offset: 13151},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 468, col: 5, offset: 13151},
							val:        "by",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 468, col: 11, offset: 13157},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 468, col: 13, offset: 13159},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 468, col: 18, offset: 13164},
								name: "fieldExprList",
							},
						},
					},
				},
			},
		},
		{
			name: "everyDur",
			pos:  position{line: 470, col: 1, offset: 13200},
			expr: &actionExpr{
				pos: position{line: 471, col: 5, offset: 13213},
				run: (*parser).calloneveryDur1,
				expr: &seqExpr{
					pos: position{line: 471, col: 5, offset: 13213},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 471, col: 5, offset: 13213},
							val:        "every",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 471, col: 14, offset: 13222},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 471, col: 16, offset: 13224},
							label: "dur",
							expr: &ruleRefExpr{
								pos:  position{line: 471, col: 20, offset: 13228},
								name: "duration",
							},
						},
					},
				},
			},
		},
		{
			name: "equalityToken",
			pos:  position{line: 473, col: 1, offset: 13258},
			expr: &choiceExpr{
				pos: position{line: 474, col: 5, offset: 13276},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 474, col: 5, offset: 13276},
						run: (*parser).callonequalityToken2,
						expr: &litMatcher{
							pos:        position{line: 474, col: 5, offset: 13276},
							val:        "=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 475, col: 5, offset: 13306},
						run: (*parser).callonequalityToken4,
						expr: &litMatcher{
							pos:        position{line: 475, col: 5, offset: 13306},
							val:        "!=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 476, col: 5, offset: 13338},
						run: (*parser).callonequalityToken6,
						expr: &litMatcher{
							pos:        position{line: 476, col: 5, offset: 13338},
							val:        "<=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 477, col: 5, offset: 13369},
						run: (*parser).callonequalityToken8,
						expr: &litMatcher{
							pos:        position{line: 477, col: 5, offset: 13369},
							val:        ">=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 478, col: 5, offset: 13400},
						run: (*parser).callonequalityToken10,
						expr: &litMatcher{
							pos:        position{line: 478, col: 5, offset: 13400},
							val:        "<",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 479, col: 5, offset: 13429},
						run: (*parser).callonequalityToken12,
						expr: &litMatcher{
							pos:        position{line: 479, col: 5, offset: 13429},
							val:        ">",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "types",
			pos:  position{line: 481, col: 1, offset: 13455},
			expr: &choiceExpr{
				pos: position{line: 482, col: 5, offset: 13465},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 482, col: 5, offset: 13465},
						val:        "bool",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 483, col: 5, offset: 13476},
						val:        "int",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 484, col: 5, offset: 13486},
						val:        "count",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 485, col: 5, offset: 13498},
						val:        "double",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 486, col: 5, offset: 13511},
						val:        "string",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 487, col: 5, offset: 13524},
						val:        "addr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 488, col: 5, offset: 13535},
						val:        "subnet",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 489, col: 5, offset: 13548},
						val:        "port",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "dash",
			pos:  position{line: 491, col: 1, offset: 13556},
			expr: &choiceExpr{
				pos: position{line: 491, col: 8, offset: 13563},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 491, col: 8, offset: 13563},
						val:        "-",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 491, col: 14, offset: 13569},
						val:        "—",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 491, col: 25, offset: 13580},
						val:        "–",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 491, col: 36, offset: 13591},
						exprs: []interface{}{
							&litMatcher{
								pos:        position{line: 491, col: 36, offset: 13591},
								val:        "to",
								ignoreCase: false,
							},
							&andExpr{
								pos: position{line: 491, col: 40, offset: 13595},
								expr: &ruleRefExpr{
									pos:  position{line: 491, col: 42, offset: 13597},
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
			pos:  position{line: 493, col: 1, offset: 13601},
			expr: &litMatcher{
				pos:        position{line: 493, col: 12, offset: 13612},
				val:        "and",
				ignoreCase: false,
			},
		},
		{
			name: "orToken",
			pos:  position{line: 494, col: 1, offset: 13618},
			expr: &litMatcher{
				pos:        position{line: 494, col: 11, offset: 13628},
				val:        "or",
				ignoreCase: false,
			},
		},
		{
			name: "inToken",
			pos:  position{line: 495, col: 1, offset: 13633},
			expr: &litMatcher{
				pos:        position{line: 495, col: 11, offset: 13643},
				val:        "in",
				ignoreCase: false,
			},
		},
		{
			name: "notToken",
			pos:  position{line: 496, col: 1, offset: 13648},
			expr: &litMatcher{
				pos:        position{line: 496, col: 12, offset: 13659},
				val:        "not",
				ignoreCase: false,
			},
		},
		{
			name: "fieldName",
			pos:  position{line: 498, col: 1, offset: 13666},
			expr: &actionExpr{
				pos: position{line: 498, col: 13, offset: 13678},
				run: (*parser).callonfieldName1,
				expr: &seqExpr{
					pos: position{line: 498, col: 13, offset: 13678},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 498, col: 13, offset: 13678},
							name: "fieldNameStart",
						},
						&zeroOrMoreExpr{
							pos: position{line: 498, col: 28, offset: 13693},
							expr: &ruleRefExpr{
								pos:  position{line: 498, col: 28, offset: 13693},
								name: "fieldNameRest",
							},
						},
					},
				},
			},
		},
		{
			name: "fieldNameStart",
			pos:  position{line: 500, col: 1, offset: 13740},
			expr: &charClassMatcher{
				pos:        position{line: 500, col: 18, offset: 13757},
				val:        "[A-Za-z_$]",
				chars:      []rune{'_', '$'},
				ranges:     []rune{'A', 'Z', 'a', 'z'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "fieldNameRest",
			pos:  position{line: 501, col: 1, offset: 13768},
			expr: &choiceExpr{
				pos: position{line: 501, col: 17, offset: 13784},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 501, col: 17, offset: 13784},
						name: "fieldNameStart",
					},
					&charClassMatcher{
						pos:        position{line: 501, col: 34, offset: 13801},
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
			pos:  position{line: 503, col: 1, offset: 13808},
			expr: &actionExpr{
				pos: position{line: 504, col: 4, offset: 13826},
				run: (*parser).callonfieldReference1,
				expr: &seqExpr{
					pos: position{line: 504, col: 4, offset: 13826},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 504, col: 4, offset: 13826},
							label: "base",
							expr: &ruleRefExpr{
								pos:  position{line: 504, col: 9, offset: 13831},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 504, col: 19, offset: 13841},
							label: "derefs",
							expr: &zeroOrMoreExpr{
								pos: position{line: 504, col: 26, offset: 13848},
								expr: &choiceExpr{
									pos: position{line: 505, col: 8, offset: 13857},
									alternatives: []interface{}{
										&actionExpr{
											pos: position{line: 505, col: 8, offset: 13857},
											run: (*parser).callonfieldReference8,
											expr: &seqExpr{
												pos: position{line: 505, col: 8, offset: 13857},
												exprs: []interface{}{
													&litMatcher{
														pos:        position{line: 505, col: 8, offset: 13857},
														val:        ".",
														ignoreCase: false,
													},
													&labeledExpr{
														pos:   position{line: 505, col: 12, offset: 13861},
														label: "field",
														expr: &ruleRefExpr{
															pos:  position{line: 505, col: 18, offset: 13867},
															name: "fieldName",
														},
													},
												},
											},
										},
										&actionExpr{
											pos: position{line: 506, col: 8, offset: 13948},
											run: (*parser).callonfieldReference13,
											expr: &seqExpr{
												pos: position{line: 506, col: 8, offset: 13948},
												exprs: []interface{}{
													&litMatcher{
														pos:        position{line: 506, col: 8, offset: 13948},
														val:        "[",
														ignoreCase: false,
													},
													&labeledExpr{
														pos:   position{line: 506, col: 12, offset: 13952},
														label: "index",
														expr: &ruleRefExpr{
															pos:  position{line: 506, col: 18, offset: 13958},
															name: "sinteger",
														},
													},
													&litMatcher{
														pos:        position{line: 506, col: 27, offset: 13967},
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
			pos:  position{line: 511, col: 1, offset: 14083},
			expr: &choiceExpr{
				pos: position{line: 512, col: 5, offset: 14097},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 512, col: 5, offset: 14097},
						run: (*parser).callonfieldExpr2,
						expr: &seqExpr{
							pos: position{line: 512, col: 5, offset: 14097},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 512, col: 5, offset: 14097},
									label: "op",
									expr: &ruleRefExpr{
										pos:  position{line: 512, col: 8, offset: 14100},
										name: "fieldOp",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 512, col: 16, offset: 14108},
									expr: &ruleRefExpr{
										pos:  position{line: 512, col: 16, offset: 14108},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 512, col: 19, offset: 14111},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 512, col: 23, offset: 14115},
									expr: &ruleRefExpr{
										pos:  position{line: 512, col: 23, offset: 14115},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 512, col: 26, offset: 14118},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 512, col: 32, offset: 14124},
										name: "fieldReference",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 512, col: 47, offset: 14139},
									expr: &ruleRefExpr{
										pos:  position{line: 512, col: 47, offset: 14139},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 512, col: 50, offset: 14142},
									val:        ")",
									ignoreCase: false,
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 515, col: 5, offset: 14206},
						name: "fieldReference",
					},
				},
			},
		},
		{
			name: "fieldOp",
			pos:  position{line: 517, col: 1, offset: 14222},
			expr: &actionExpr{
				pos: position{line: 518, col: 5, offset: 14234},
				run: (*parser).callonfieldOp1,
				expr: &litMatcher{
					pos:        position{line: 518, col: 5, offset: 14234},
					val:        "len",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldExprList",
			pos:  position{line: 520, col: 1, offset: 14264},
			expr: &actionExpr{
				pos: position{line: 521, col: 5, offset: 14282},
				run: (*parser).callonfieldExprList1,
				expr: &seqExpr{
					pos: position{line: 521, col: 5, offset: 14282},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 521, col: 5, offset: 14282},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 521, col: 11, offset: 14288},
								name: "fieldExpr",
							},
						},
						&labeledExpr{
							pos:   position{line: 521, col: 21, offset: 14298},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 521, col: 26, offset: 14303},
								expr: &seqExpr{
									pos: position{line: 521, col: 27, offset: 14304},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 521, col: 27, offset: 14304},
											expr: &ruleRefExpr{
												pos:  position{line: 521, col: 27, offset: 14304},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 521, col: 30, offset: 14307},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 521, col: 34, offset: 14311},
											expr: &ruleRefExpr{
												pos:  position{line: 521, col: 34, offset: 14311},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 521, col: 37, offset: 14314},
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
			pos:  position{line: 531, col: 1, offset: 14509},
			expr: &actionExpr{
				pos: position{line: 532, col: 5, offset: 14527},
				run: (*parser).callonfieldNameList1,
				expr: &seqExpr{
					pos: position{line: 532, col: 5, offset: 14527},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 532, col: 5, offset: 14527},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 532, col: 11, offset: 14533},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 532, col: 21, offset: 14543},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 532, col: 26, offset: 14548},
								expr: &seqExpr{
									pos: position{line: 532, col: 27, offset: 14549},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 532, col: 27, offset: 14549},
											expr: &ruleRefExpr{
												pos:  position{line: 532, col: 27, offset: 14549},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 532, col: 30, offset: 14552},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 532, col: 34, offset: 14556},
											expr: &ruleRefExpr{
												pos:  position{line: 532, col: 34, offset: 14556},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 532, col: 37, offset: 14559},
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
			pos:  position{line: 540, col: 1, offset: 14752},
			expr: &actionExpr{
				pos: position{line: 541, col: 5, offset: 14764},
				run: (*parser).calloncountOp1,
				expr: &litMatcher{
					pos:        position{line: 541, col: 5, offset: 14764},
					val:        "count",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldReducerOp",
			pos:  position{line: 543, col: 1, offset: 14798},
			expr: &choiceExpr{
				pos: position{line: 544, col: 5, offset: 14817},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 544, col: 5, offset: 14817},
						run: (*parser).callonfieldReducerOp2,
						expr: &litMatcher{
							pos:        position{line: 544, col: 5, offset: 14817},
							val:        "sum",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 545, col: 5, offset: 14851},
						run: (*parser).callonfieldReducerOp4,
						expr: &litMatcher{
							pos:        position{line: 545, col: 5, offset: 14851},
							val:        "avg",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 546, col: 5, offset: 14885},
						run: (*parser).callonfieldReducerOp6,
						expr: &litMatcher{
							pos:        position{line: 546, col: 5, offset: 14885},
							val:        "stdev",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 547, col: 5, offset: 14922},
						run: (*parser).callonfieldReducerOp8,
						expr: &litMatcher{
							pos:        position{line: 547, col: 5, offset: 14922},
							val:        "sd",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 548, col: 5, offset: 14958},
						run: (*parser).callonfieldReducerOp10,
						expr: &litMatcher{
							pos:        position{line: 548, col: 5, offset: 14958},
							val:        "var",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 549, col: 5, offset: 14992},
						run: (*parser).callonfieldReducerOp12,
						expr: &litMatcher{
							pos:        position{line: 549, col: 5, offset: 14992},
							val:        "entropy",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 550, col: 5, offset: 15033},
						run: (*parser).callonfieldReducerOp14,
						expr: &litMatcher{
							pos:        position{line: 550, col: 5, offset: 15033},
							val:        "min",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 551, col: 5, offset: 15067},
						run: (*parser).callonfieldReducerOp16,
						expr: &litMatcher{
							pos:        position{line: 551, col: 5, offset: 15067},
							val:        "max",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 552, col: 5, offset: 15101},
						run: (*parser).callonfieldReducerOp18,
						expr: &litMatcher{
							pos:        position{line: 552, col: 5, offset: 15101},
							val:        "first",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 553, col: 5, offset: 15139},
						run: (*parser).callonfieldReducerOp20,
						expr: &litMatcher{
							pos:        position{line: 553, col: 5, offset: 15139},
							val:        "last",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 554, col: 5, offset: 15175},
						run: (*parser).callonfieldReducerOp22,
						expr: &litMatcher{
							pos:        position{line: 554, col: 5, offset: 15175},
							val:        "countdistinct",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "paddedFieldName",
			pos:  position{line: 556, col: 1, offset: 15225},
			expr: &actionExpr{
				pos: position{line: 556, col: 19, offset: 15243},
				run: (*parser).callonpaddedFieldName1,
				expr: &seqExpr{
					pos: position{line: 556, col: 19, offset: 15243},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 556, col: 19, offset: 15243},
							expr: &ruleRefExpr{
								pos:  position{line: 556, col: 19, offset: 15243},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 556, col: 22, offset: 15246},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 556, col: 28, offset: 15252},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 556, col: 38, offset: 15262},
							expr: &ruleRefExpr{
								pos:  position{line: 556, col: 38, offset: 15262},
								name: "_",
							},
						},
					},
				},
			},
		},
		{
			name: "countReducer",
			pos:  position{line: 558, col: 1, offset: 15288},
			expr: &actionExpr{
				pos: position{line: 559, col: 5, offset: 15305},
				run: (*parser).calloncountReducer1,
				expr: &seqExpr{
					pos: position{line: 559, col: 5, offset: 15305},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 559, col: 5, offset: 15305},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 559, col: 8, offset: 15308},
								name: "countOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 559, col: 16, offset: 15316},
							expr: &ruleRefExpr{
								pos:  position{line: 559, col: 16, offset: 15316},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 559, col: 19, offset: 15319},
							val:        "(",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 559, col: 23, offset: 15323},
							label: "field",
							expr: &zeroOrOneExpr{
								pos: position{line: 559, col: 29, offset: 15329},
								expr: &ruleRefExpr{
									pos:  position{line: 559, col: 29, offset: 15329},
									name: "paddedFieldName",
								},
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 559, col: 47, offset: 15347},
							expr: &ruleRefExpr{
								pos:  position{line: 559, col: 47, offset: 15347},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 559, col: 50, offset: 15350},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "fieldReducer",
			pos:  position{line: 563, col: 1, offset: 15409},
			expr: &actionExpr{
				pos: position{line: 564, col: 5, offset: 15426},
				run: (*parser).callonfieldReducer1,
				expr: &seqExpr{
					pos: position{line: 564, col: 5, offset: 15426},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 564, col: 5, offset: 15426},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 564, col: 8, offset: 15429},
								name: "fieldReducerOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 564, col: 23, offset: 15444},
							expr: &ruleRefExpr{
								pos:  position{line: 564, col: 23, offset: 15444},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 564, col: 26, offset: 15447},
							val:        "(",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 564, col: 30, offset: 15451},
							expr: &ruleRefExpr{
								pos:  position{line: 564, col: 30, offset: 15451},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 564, col: 33, offset: 15454},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 564, col: 39, offset: 15460},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 564, col: 50, offset: 15471},
							expr: &ruleRefExpr{
								pos:  position{line: 564, col: 50, offset: 15471},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 564, col: 53, offset: 15474},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reducerProc",
			pos:  position{line: 568, col: 1, offset: 15541},
			expr: &actionExpr{
				pos: position{line: 569, col: 5, offset: 15557},
				run: (*parser).callonreducerProc1,
				expr: &seqExpr{
					pos: position{line: 569, col: 5, offset: 15557},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 569, col: 5, offset: 15557},
							label: "every",
							expr: &zeroOrOneExpr{
								pos: position{line: 569, col: 11, offset: 15563},
								expr: &seqExpr{
									pos: position{line: 569, col: 12, offset: 15564},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 569, col: 12, offset: 15564},
											name: "everyDur",
										},
										&ruleRefExpr{
											pos:  position{line: 569, col: 21, offset: 15573},
											name: "_",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 569, col: 25, offset: 15577},
							label: "reducers",
							expr: &ruleRefExpr{
								pos:  position{line: 569, col: 34, offset: 15586},
								name: "reducerList",
							},
						},
						&labeledExpr{
							pos:   position{line: 569, col: 46, offset: 15598},
							label: "keys",
							expr: &zeroOrOneExpr{
								pos: position{line: 569, col: 51, offset: 15603},
								expr: &seqExpr{
									pos: position{line: 569, col: 52, offset: 15604},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 569, col: 52, offset: 15604},
											name: "_",
										},
										&ruleRefExpr{
											pos:  position{line: 569, col: 54, offset: 15606},
											name: "groupBy",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 569, col: 64, offset: 15616},
							label: "limit",
							expr: &zeroOrOneExpr{
								pos: position{line: 569, col: 70, offset: 15622},
								expr: &ruleRefExpr{
									pos:  position{line: 569, col: 70, offset: 15622},
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
			pos:  position{line: 587, col: 1, offset: 15979},
			expr: &actionExpr{
				pos: position{line: 588, col: 5, offset: 15992},
				run: (*parser).callonasClause1,
				expr: &seqExpr{
					pos: position{line: 588, col: 5, offset: 15992},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 588, col: 5, offset: 15992},
							val:        "as",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 588, col: 11, offset: 15998},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 588, col: 13, offset: 16000},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 588, col: 15, offset: 16002},
								name: "fieldName",
							},
						},
					},
				},
			},
		},
		{
			name: "reducerExpr",
			pos:  position{line: 590, col: 1, offset: 16031},
			expr: &choiceExpr{
				pos: position{line: 591, col: 5, offset: 16047},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 591, col: 5, offset: 16047},
						run: (*parser).callonreducerExpr2,
						expr: &seqExpr{
							pos: position{line: 591, col: 5, offset: 16047},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 591, col: 5, offset: 16047},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 591, col: 11, offset: 16053},
										name: "fieldName",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 591, col: 21, offset: 16063},
									expr: &ruleRefExpr{
										pos:  position{line: 591, col: 21, offset: 16063},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 591, col: 24, offset: 16066},
									val:        "=",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 591, col: 28, offset: 16070},
									expr: &ruleRefExpr{
										pos:  position{line: 591, col: 28, offset: 16070},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 591, col: 31, offset: 16073},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 591, col: 33, offset: 16075},
										name: "reducer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 594, col: 5, offset: 16138},
						run: (*parser).callonreducerExpr13,
						expr: &seqExpr{
							pos: position{line: 594, col: 5, offset: 16138},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 594, col: 5, offset: 16138},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 594, col: 7, offset: 16140},
										name: "reducer",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 594, col: 15, offset: 16148},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 594, col: 17, offset: 16150},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 594, col: 23, offset: 16156},
										name: "asClause",
									},
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 597, col: 5, offset: 16220},
						name: "reducer",
					},
				},
			},
		},
		{
			name: "reducer",
			pos:  position{line: 599, col: 1, offset: 16229},
			expr: &choiceExpr{
				pos: position{line: 600, col: 5, offset: 16241},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 600, col: 5, offset: 16241},
						name: "countReducer",
					},
					&ruleRefExpr{
						pos:  position{line: 601, col: 5, offset: 16258},
						name: "fieldReducer",
					},
				},
			},
		},
		{
			name: "reducerList",
			pos:  position{line: 603, col: 1, offset: 16272},
			expr: &actionExpr{
				pos: position{line: 604, col: 5, offset: 16288},
				run: (*parser).callonreducerList1,
				expr: &seqExpr{
					pos: position{line: 604, col: 5, offset: 16288},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 604, col: 5, offset: 16288},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 604, col: 11, offset: 16294},
								name: "reducerExpr",
							},
						},
						&labeledExpr{
							pos:   position{line: 604, col: 23, offset: 16306},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 604, col: 28, offset: 16311},
								expr: &seqExpr{
									pos: position{line: 604, col: 29, offset: 16312},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 604, col: 29, offset: 16312},
											expr: &ruleRefExpr{
												pos:  position{line: 604, col: 29, offset: 16312},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 604, col: 32, offset: 16315},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 604, col: 36, offset: 16319},
											expr: &ruleRefExpr{
												pos:  position{line: 604, col: 36, offset: 16319},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 604, col: 39, offset: 16322},
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
			pos:  position{line: 612, col: 1, offset: 16519},
			expr: &choiceExpr{
				pos: position{line: 613, col: 5, offset: 16534},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 613, col: 5, offset: 16534},
						name: "sort",
					},
					&ruleRefExpr{
						pos:  position{line: 614, col: 5, offset: 16543},
						name: "top",
					},
					&ruleRefExpr{
						pos:  position{line: 615, col: 5, offset: 16551},
						name: "cut",
					},
					&ruleRefExpr{
						pos:  position{line: 616, col: 5, offset: 16559},
						name: "head",
					},
					&ruleRefExpr{
						pos:  position{line: 617, col: 5, offset: 16568},
						name: "tail",
					},
					&ruleRefExpr{
						pos:  position{line: 618, col: 5, offset: 16577},
						name: "filter",
					},
					&ruleRefExpr{
						pos:  position{line: 619, col: 5, offset: 16588},
						name: "uniq",
					},
				},
			},
		},
		{
			name: "sort",
			pos:  position{line: 621, col: 1, offset: 16594},
			expr: &choiceExpr{
				pos: position{line: 622, col: 5, offset: 16603},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 622, col: 5, offset: 16603},
						run: (*parser).callonsort2,
						expr: &seqExpr{
							pos: position{line: 622, col: 5, offset: 16603},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 622, col: 5, offset: 16603},
									val:        "sort",
									ignoreCase: true,
								},
								&labeledExpr{
									pos:   position{line: 622, col: 13, offset: 16611},
									label: "rev",
									expr: &zeroOrOneExpr{
										pos: position{line: 622, col: 17, offset: 16615},
										expr: &seqExpr{
											pos: position{line: 622, col: 18, offset: 16616},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 622, col: 18, offset: 16616},
													name: "_",
												},
												&litMatcher{
													pos:        position{line: 622, col: 20, offset: 16618},
													val:        "-r",
													ignoreCase: false,
												},
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 622, col: 27, offset: 16625},
									label: "limit",
									expr: &zeroOrOneExpr{
										pos: position{line: 622, col: 33, offset: 16631},
										expr: &ruleRefExpr{
											pos:  position{line: 622, col: 33, offset: 16631},
											name: "procLimitArg",
										},
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 622, col: 48, offset: 16646},
									expr: &ruleRefExpr{
										pos:  position{line: 622, col: 48, offset: 16646},
										name: "_",
									},
								},
								&notExpr{
									pos: position{line: 622, col: 51, offset: 16649},
									expr: &litMatcher{
										pos:        position{line: 622, col: 52, offset: 16650},
										val:        "-r",
										ignoreCase: false,
									},
								},
								&labeledExpr{
									pos:   position{line: 622, col: 57, offset: 16655},
									label: "list",
									expr: &zeroOrOneExpr{
										pos: position{line: 622, col: 62, offset: 16660},
										expr: &ruleRefExpr{
											pos:  position{line: 622, col: 63, offset: 16661},
											name: "fieldExprList",
										},
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 627, col: 5, offset: 16791},
						run: (*parser).callonsort20,
						expr: &seqExpr{
							pos: position{line: 627, col: 5, offset: 16791},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 627, col: 5, offset: 16791},
									val:        "sort",
									ignoreCase: true,
								},
								&labeledExpr{
									pos:   position{line: 627, col: 13, offset: 16799},
									label: "limit",
									expr: &zeroOrOneExpr{
										pos: position{line: 627, col: 19, offset: 16805},
										expr: &ruleRefExpr{
											pos:  position{line: 627, col: 19, offset: 16805},
											name: "procLimitArg",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 627, col: 33, offset: 16819},
									label: "rev",
									expr: &zeroOrOneExpr{
										pos: position{line: 627, col: 37, offset: 16823},
										expr: &seqExpr{
											pos: position{line: 627, col: 38, offset: 16824},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 627, col: 38, offset: 16824},
													name: "_",
												},
												&litMatcher{
													pos:        position{line: 627, col: 40, offset: 16826},
													val:        "-r",
													ignoreCase: false,
												},
											},
										},
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 627, col: 47, offset: 16833},
									expr: &ruleRefExpr{
										pos:  position{line: 627, col: 47, offset: 16833},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 627, col: 50, offset: 16836},
									label: "list",
									expr: &zeroOrOneExpr{
										pos: position{line: 627, col: 55, offset: 16841},
										expr: &ruleRefExpr{
											pos:  position{line: 627, col: 56, offset: 16842},
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
			pos:  position{line: 633, col: 1, offset: 16969},
			expr: &actionExpr{
				pos: position{line: 634, col: 5, offset: 16977},
				run: (*parser).callontop1,
				expr: &seqExpr{
					pos: position{line: 634, col: 5, offset: 16977},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 634, col: 5, offset: 16977},
							val:        "top",
							ignoreCase: true,
						},
						&labeledExpr{
							pos:   position{line: 634, col: 12, offset: 16984},
							label: "limit",
							expr: &zeroOrOneExpr{
								pos: position{line: 634, col: 18, offset: 16990},
								expr: &ruleRefExpr{
									pos:  position{line: 634, col: 18, offset: 16990},
									name: "procLimitArg",
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 634, col: 32, offset: 17004},
							label: "flush",
							expr: &zeroOrOneExpr{
								pos: position{line: 634, col: 38, offset: 17010},
								expr: &seqExpr{
									pos: position{line: 634, col: 39, offset: 17011},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 634, col: 39, offset: 17011},
											name: "_",
										},
										&litMatcher{
											pos:        position{line: 634, col: 41, offset: 17013},
											val:        "-flush",
											ignoreCase: false,
										},
									},
								},
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 634, col: 52, offset: 17024},
							expr: &ruleRefExpr{
								pos:  position{line: 634, col: 52, offset: 17024},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 634, col: 55, offset: 17027},
							label: "list",
							expr: &zeroOrOneExpr{
								pos: position{line: 634, col: 60, offset: 17032},
								expr: &ruleRefExpr{
									pos:  position{line: 634, col: 61, offset: 17033},
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
			pos:  position{line: 638, col: 1, offset: 17104},
			expr: &actionExpr{
				pos: position{line: 639, col: 5, offset: 17121},
				run: (*parser).callonprocLimitArg1,
				expr: &seqExpr{
					pos: position{line: 639, col: 5, offset: 17121},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 639, col: 5, offset: 17121},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 639, col: 7, offset: 17123},
							val:        "-limit",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 639, col: 16, offset: 17132},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 639, col: 18, offset: 17134},
							label: "limit",
							expr: &ruleRefExpr{
								pos:  position{line: 639, col: 24, offset: 17140},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "cut",
			pos:  position{line: 641, col: 1, offset: 17171},
			expr: &actionExpr{
				pos: position{line: 642, col: 5, offset: 17179},
				run: (*parser).calloncut1,
				expr: &seqExpr{
					pos: position{line: 642, col: 5, offset: 17179},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 642, col: 5, offset: 17179},
							val:        "cut",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 642, col: 12, offset: 17186},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 642, col: 14, offset: 17188},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 642, col: 19, offset: 17193},
								name: "fieldNameList",
							},
						},
					},
				},
			},
		},
		{
			name: "head",
			pos:  position{line: 643, col: 1, offset: 17241},
			expr: &choiceExpr{
				pos: position{line: 644, col: 5, offset: 17250},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 644, col: 5, offset: 17250},
						run: (*parser).callonhead2,
						expr: &seqExpr{
							pos: position{line: 644, col: 5, offset: 17250},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 644, col: 5, offset: 17250},
									val:        "head",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 644, col: 13, offset: 17258},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 644, col: 15, offset: 17260},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 644, col: 21, offset: 17266},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 645, col: 5, offset: 17314},
						run: (*parser).callonhead8,
						expr: &litMatcher{
							pos:        position{line: 645, col: 5, offset: 17314},
							val:        "head",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "tail",
			pos:  position{line: 646, col: 1, offset: 17354},
			expr: &choiceExpr{
				pos: position{line: 647, col: 5, offset: 17363},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 647, col: 5, offset: 17363},
						run: (*parser).callontail2,
						expr: &seqExpr{
							pos: position{line: 647, col: 5, offset: 17363},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 647, col: 5, offset: 17363},
									val:        "tail",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 647, col: 13, offset: 17371},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 647, col: 15, offset: 17373},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 647, col: 21, offset: 17379},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 648, col: 5, offset: 17427},
						run: (*parser).callontail8,
						expr: &litMatcher{
							pos:        position{line: 648, col: 5, offset: 17427},
							val:        "tail",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "filter",
			pos:  position{line: 650, col: 1, offset: 17468},
			expr: &actionExpr{
				pos: position{line: 651, col: 5, offset: 17479},
				run: (*parser).callonfilter1,
				expr: &seqExpr{
					pos: position{line: 651, col: 5, offset: 17479},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 651, col: 5, offset: 17479},
							val:        "filter",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 651, col: 15, offset: 17489},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 651, col: 17, offset: 17491},
							label: "expr",
							expr: &ruleRefExpr{
								pos:  position{line: 651, col: 22, offset: 17496},
								name: "searchExpr",
							},
						},
					},
				},
			},
		},
		{
			name: "uniq",
			pos:  position{line: 654, col: 1, offset: 17554},
			expr: &choiceExpr{
				pos: position{line: 655, col: 5, offset: 17563},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 655, col: 5, offset: 17563},
						run: (*parser).callonuniq2,
						expr: &seqExpr{
							pos: position{line: 655, col: 5, offset: 17563},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 655, col: 5, offset: 17563},
									val:        "uniq",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 655, col: 13, offset: 17571},
									name: "_",
								},
								&litMatcher{
									pos:        position{line: 655, col: 15, offset: 17573},
									val:        "-c",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 658, col: 5, offset: 17627},
						run: (*parser).callonuniq7,
						expr: &litMatcher{
							pos:        position{line: 658, col: 5, offset: 17627},
							val:        "uniq",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "duration",
			pos:  position{line: 662, col: 1, offset: 17682},
			expr: &choiceExpr{
				pos: position{line: 663, col: 5, offset: 17695},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 663, col: 5, offset: 17695},
						name: "seconds",
					},
					&ruleRefExpr{
						pos:  position{line: 664, col: 5, offset: 17707},
						name: "minutes",
					},
					&ruleRefExpr{
						pos:  position{line: 665, col: 5, offset: 17719},
						name: "hours",
					},
					&seqExpr{
						pos: position{line: 666, col: 5, offset: 17729},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 666, col: 5, offset: 17729},
								name: "hours",
							},
							&ruleRefExpr{
								pos:  position{line: 666, col: 11, offset: 17735},
								name: "_",
							},
							&litMatcher{
								pos:        position{line: 666, col: 13, offset: 17737},
								val:        "and",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 666, col: 19, offset: 17743},
								name: "_",
							},
							&ruleRefExpr{
								pos:  position{line: 666, col: 21, offset: 17745},
								name: "minutes",
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 667, col: 5, offset: 17757},
						name: "days",
					},
					&ruleRefExpr{
						pos:  position{line: 668, col: 5, offset: 17766},
						name: "weeks",
					},
				},
			},
		},
		{
			name: "sec_abbrev",
			pos:  position{line: 670, col: 1, offset: 17773},
			expr: &choiceExpr{
				pos: position{line: 671, col: 5, offset: 17788},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 671, col: 5, offset: 17788},
						val:        "seconds",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 672, col: 5, offset: 17802},
						val:        "second",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 673, col: 5, offset: 17815},
						val:        "secs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 674, col: 5, offset: 17826},
						val:        "sec",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 675, col: 5, offset: 17836},
						val:        "s",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "min_abbrev",
			pos:  position{line: 677, col: 1, offset: 17841},
			expr: &choiceExpr{
				pos: position{line: 678, col: 5, offset: 17856},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 678, col: 5, offset: 17856},
						val:        "minutes",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 679, col: 5, offset: 17870},
						val:        "minute",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 680, col: 5, offset: 17883},
						val:        "mins",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 681, col: 5, offset: 17894},
						val:        "min",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 682, col: 5, offset: 17904},
						val:        "m",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "hour_abbrev",
			pos:  position{line: 684, col: 1, offset: 17909},
			expr: &choiceExpr{
				pos: position{line: 685, col: 5, offset: 17925},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 685, col: 5, offset: 17925},
						val:        "hours",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 686, col: 5, offset: 17937},
						val:        "hrs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 687, col: 5, offset: 17947},
						val:        "hr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 688, col: 5, offset: 17956},
						val:        "h",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 689, col: 5, offset: 17964},
						val:        "hour",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "day_abbrev",
			pos:  position{line: 691, col: 1, offset: 17972},
			expr: &choiceExpr{
				pos: position{line: 691, col: 14, offset: 17985},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 691, col: 14, offset: 17985},
						val:        "days",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 691, col: 21, offset: 17992},
						val:        "day",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 691, col: 27, offset: 17998},
						val:        "d",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "week_abbrev",
			pos:  position{line: 692, col: 1, offset: 18002},
			expr: &choiceExpr{
				pos: position{line: 692, col: 15, offset: 18016},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 692, col: 15, offset: 18016},
						val:        "weeks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 692, col: 23, offset: 18024},
						val:        "week",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 692, col: 30, offset: 18031},
						val:        "wks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 692, col: 36, offset: 18037},
						val:        "wk",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 692, col: 41, offset: 18042},
						val:        "w",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "seconds",
			pos:  position{line: 694, col: 1, offset: 18047},
			expr: &choiceExpr{
				pos: position{line: 695, col: 5, offset: 18059},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 695, col: 5, offset: 18059},
						run: (*parser).callonseconds2,
						expr: &litMatcher{
							pos:        position{line: 695, col: 5, offset: 18059},
							val:        "second",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 696, col: 5, offset: 18104},
						run: (*parser).callonseconds4,
						expr: &seqExpr{
							pos: position{line: 696, col: 5, offset: 18104},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 696, col: 5, offset: 18104},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 696, col: 9, offset: 18108},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 696, col: 16, offset: 18115},
									expr: &ruleRefExpr{
										pos:  position{line: 696, col: 16, offset: 18115},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 696, col: 19, offset: 18118},
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
			pos:  position{line: 698, col: 1, offset: 18164},
			expr: &choiceExpr{
				pos: position{line: 699, col: 5, offset: 18176},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 699, col: 5, offset: 18176},
						run: (*parser).callonminutes2,
						expr: &litMatcher{
							pos:        position{line: 699, col: 5, offset: 18176},
							val:        "minute",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 700, col: 5, offset: 18222},
						run: (*parser).callonminutes4,
						expr: &seqExpr{
							pos: position{line: 700, col: 5, offset: 18222},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 700, col: 5, offset: 18222},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 700, col: 9, offset: 18226},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 700, col: 16, offset: 18233},
									expr: &ruleRefExpr{
										pos:  position{line: 700, col: 16, offset: 18233},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 700, col: 19, offset: 18236},
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
			pos:  position{line: 702, col: 1, offset: 18291},
			expr: &choiceExpr{
				pos: position{line: 703, col: 5, offset: 18301},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 703, col: 5, offset: 18301},
						run: (*parser).callonhours2,
						expr: &litMatcher{
							pos:        position{line: 703, col: 5, offset: 18301},
							val:        "hour",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 704, col: 5, offset: 18347},
						run: (*parser).callonhours4,
						expr: &seqExpr{
							pos: position{line: 704, col: 5, offset: 18347},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 704, col: 5, offset: 18347},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 704, col: 9, offset: 18351},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 704, col: 16, offset: 18358},
									expr: &ruleRefExpr{
										pos:  position{line: 704, col: 16, offset: 18358},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 704, col: 19, offset: 18361},
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
			pos:  position{line: 706, col: 1, offset: 18419},
			expr: &choiceExpr{
				pos: position{line: 707, col: 5, offset: 18428},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 707, col: 5, offset: 18428},
						run: (*parser).callondays2,
						expr: &litMatcher{
							pos:        position{line: 707, col: 5, offset: 18428},
							val:        "day",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 708, col: 5, offset: 18476},
						run: (*parser).callondays4,
						expr: &seqExpr{
							pos: position{line: 708, col: 5, offset: 18476},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 708, col: 5, offset: 18476},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 708, col: 9, offset: 18480},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 708, col: 16, offset: 18487},
									expr: &ruleRefExpr{
										pos:  position{line: 708, col: 16, offset: 18487},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 708, col: 19, offset: 18490},
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
			pos:  position{line: 710, col: 1, offset: 18550},
			expr: &actionExpr{
				pos: position{line: 711, col: 5, offset: 18560},
				run: (*parser).callonweeks1,
				expr: &seqExpr{
					pos: position{line: 711, col: 5, offset: 18560},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 711, col: 5, offset: 18560},
							label: "num",
							expr: &ruleRefExpr{
								pos:  position{line: 711, col: 9, offset: 18564},
								name: "number",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 711, col: 16, offset: 18571},
							expr: &ruleRefExpr{
								pos:  position{line: 711, col: 16, offset: 18571},
								name: "_",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 711, col: 19, offset: 18574},
							name: "week_abbrev",
						},
					},
				},
			},
		},
		{
			name: "number",
			pos:  position{line: 713, col: 1, offset: 18637},
			expr: &ruleRefExpr{
				pos:  position{line: 713, col: 10, offset: 18646},
				name: "integer",
			},
		},
		{
			name: "addr",
			pos:  position{line: 717, col: 1, offset: 18684},
			expr: &actionExpr{
				pos: position{line: 718, col: 5, offset: 18693},
				run: (*parser).callonaddr1,
				expr: &labeledExpr{
					pos:   position{line: 718, col: 5, offset: 18693},
					label: "a",
					expr: &seqExpr{
						pos: position{line: 718, col: 8, offset: 18696},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 718, col: 8, offset: 18696},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 718, col: 16, offset: 18704},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 718, col: 20, offset: 18708},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 718, col: 28, offset: 18716},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 718, col: 32, offset: 18720},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 718, col: 40, offset: 18728},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 718, col: 44, offset: 18732},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "port",
			pos:  position{line: 720, col: 1, offset: 18773},
			expr: &actionExpr{
				pos: position{line: 721, col: 5, offset: 18782},
				run: (*parser).callonport1,
				expr: &seqExpr{
					pos: position{line: 721, col: 5, offset: 18782},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 721, col: 5, offset: 18782},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 721, col: 9, offset: 18786},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 721, col: 11, offset: 18788},
								name: "sinteger",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6addr",
			pos:  position{line: 725, col: 1, offset: 18947},
			expr: &choiceExpr{
				pos: position{line: 726, col: 5, offset: 18959},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 726, col: 5, offset: 18959},
						run: (*parser).callonip6addr2,
						expr: &seqExpr{
							pos: position{line: 726, col: 5, offset: 18959},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 726, col: 5, offset: 18959},
									label: "a",
									expr: &oneOrMoreExpr{
										pos: position{line: 726, col: 7, offset: 18961},
										expr: &ruleRefExpr{
											pos:  position{line: 726, col: 8, offset: 18962},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 726, col: 20, offset: 18974},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 726, col: 22, offset: 18976},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 729, col: 5, offset: 19040},
						run: (*parser).callonip6addr9,
						expr: &seqExpr{
							pos: position{line: 729, col: 5, offset: 19040},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 729, col: 5, offset: 19040},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 729, col: 7, offset: 19042},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 729, col: 11, offset: 19046},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 729, col: 13, offset: 19048},
										expr: &ruleRefExpr{
											pos:  position{line: 729, col: 14, offset: 19049},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 729, col: 25, offset: 19060},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 729, col: 30, offset: 19065},
									label: "d",
									expr: &zeroOrMoreExpr{
										pos: position{line: 729, col: 32, offset: 19067},
										expr: &ruleRefExpr{
											pos:  position{line: 729, col: 33, offset: 19068},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 729, col: 45, offset: 19080},
									label: "e",
									expr: &ruleRefExpr{
										pos:  position{line: 729, col: 47, offset: 19082},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 732, col: 5, offset: 19181},
						run: (*parser).callonip6addr22,
						expr: &seqExpr{
							pos: position{line: 732, col: 5, offset: 19181},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 732, col: 5, offset: 19181},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 732, col: 10, offset: 19186},
									label: "a",
									expr: &zeroOrMoreExpr{
										pos: position{line: 732, col: 12, offset: 19188},
										expr: &ruleRefExpr{
											pos:  position{line: 732, col: 13, offset: 19189},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 732, col: 25, offset: 19201},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 732, col: 27, offset: 19203},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 735, col: 5, offset: 19274},
						run: (*parser).callonip6addr30,
						expr: &seqExpr{
							pos: position{line: 735, col: 5, offset: 19274},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 735, col: 5, offset: 19274},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 735, col: 7, offset: 19276},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 735, col: 11, offset: 19280},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 735, col: 13, offset: 19282},
										expr: &ruleRefExpr{
											pos:  position{line: 735, col: 14, offset: 19283},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 735, col: 25, offset: 19294},
									val:        "::",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 738, col: 5, offset: 19362},
						run: (*parser).callonip6addr38,
						expr: &litMatcher{
							pos:        position{line: 738, col: 5, offset: 19362},
							val:        "::",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "ip6tail",
			pos:  position{line: 742, col: 1, offset: 19399},
			expr: &choiceExpr{
				pos: position{line: 743, col: 5, offset: 19411},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 743, col: 5, offset: 19411},
						name: "addr",
					},
					&ruleRefExpr{
						pos:  position{line: 744, col: 5, offset: 19420},
						name: "h16",
					},
				},
			},
		},
		{
			name: "h_append",
			pos:  position{line: 746, col: 1, offset: 19425},
			expr: &actionExpr{
				pos: position{line: 746, col: 12, offset: 19436},
				run: (*parser).callonh_append1,
				expr: &seqExpr{
					pos: position{line: 746, col: 12, offset: 19436},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 746, col: 12, offset: 19436},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 746, col: 16, offset: 19440},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 746, col: 18, offset: 19442},
								name: "h16",
							},
						},
					},
				},
			},
		},
		{
			name: "h_prepend",
			pos:  position{line: 747, col: 1, offset: 19479},
			expr: &actionExpr{
				pos: position{line: 747, col: 13, offset: 19491},
				run: (*parser).callonh_prepend1,
				expr: &seqExpr{
					pos: position{line: 747, col: 13, offset: 19491},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 747, col: 13, offset: 19491},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 747, col: 15, offset: 19493},
								name: "h16",
							},
						},
						&litMatcher{
							pos:        position{line: 747, col: 19, offset: 19497},
							val:        ":",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "sub_addr",
			pos:  position{line: 749, col: 1, offset: 19535},
			expr: &choiceExpr{
				pos: position{line: 750, col: 5, offset: 19548},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 750, col: 5, offset: 19548},
						name: "addr",
					},
					&actionExpr{
						pos: position{line: 751, col: 5, offset: 19557},
						run: (*parser).callonsub_addr3,
						expr: &labeledExpr{
							pos:   position{line: 751, col: 5, offset: 19557},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 751, col: 8, offset: 19560},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 751, col: 8, offset: 19560},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 751, col: 16, offset: 19568},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 751, col: 20, offset: 19572},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 751, col: 28, offset: 19580},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 751, col: 32, offset: 19584},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 752, col: 5, offset: 19636},
						run: (*parser).callonsub_addr11,
						expr: &labeledExpr{
							pos:   position{line: 752, col: 5, offset: 19636},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 752, col: 8, offset: 19639},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 752, col: 8, offset: 19639},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 752, col: 16, offset: 19647},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 752, col: 20, offset: 19651},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 753, col: 5, offset: 19705},
						run: (*parser).callonsub_addr17,
						expr: &labeledExpr{
							pos:   position{line: 753, col: 5, offset: 19705},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 753, col: 7, offset: 19707},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "subnet",
			pos:  position{line: 755, col: 1, offset: 19758},
			expr: &actionExpr{
				pos: position{line: 756, col: 5, offset: 19769},
				run: (*parser).callonsubnet1,
				expr: &seqExpr{
					pos: position{line: 756, col: 5, offset: 19769},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 756, col: 5, offset: 19769},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 756, col: 7, offset: 19771},
								name: "sub_addr",
							},
						},
						&litMatcher{
							pos:        position{line: 756, col: 16, offset: 19780},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 756, col: 20, offset: 19784},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 756, col: 22, offset: 19786},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6subnet",
			pos:  position{line: 760, col: 1, offset: 19862},
			expr: &actionExpr{
				pos: position{line: 761, col: 5, offset: 19876},
				run: (*parser).callonip6subnet1,
				expr: &seqExpr{
					pos: position{line: 761, col: 5, offset: 19876},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 761, col: 5, offset: 19876},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 761, col: 7, offset: 19878},
								name: "ip6addr",
							},
						},
						&litMatcher{
							pos:        position{line: 761, col: 15, offset: 19886},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 761, col: 19, offset: 19890},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 761, col: 21, offset: 19892},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "integer",
			pos:  position{line: 765, col: 1, offset: 19958},
			expr: &actionExpr{
				pos: position{line: 766, col: 5, offset: 19970},
				run: (*parser).calloninteger1,
				expr: &labeledExpr{
					pos:   position{line: 766, col: 5, offset: 19970},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 766, col: 7, offset: 19972},
						name: "sinteger",
					},
				},
			},
		},
		{
			name: "sinteger",
			pos:  position{line: 770, col: 1, offset: 20016},
			expr: &actionExpr{
				pos: position{line: 771, col: 5, offset: 20029},
				run: (*parser).callonsinteger1,
				expr: &labeledExpr{
					pos:   position{line: 771, col: 5, offset: 20029},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 771, col: 11, offset: 20035},
						expr: &charClassMatcher{
							pos:        position{line: 771, col: 11, offset: 20035},
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
			pos:  position{line: 775, col: 1, offset: 20080},
			expr: &actionExpr{
				pos: position{line: 776, col: 5, offset: 20091},
				run: (*parser).callondouble1,
				expr: &labeledExpr{
					pos:   position{line: 776, col: 5, offset: 20091},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 776, col: 7, offset: 20093},
						name: "sdouble",
					},
				},
			},
		},
		{
			name: "sdouble",
			pos:  position{line: 780, col: 1, offset: 20140},
			expr: &choiceExpr{
				pos: position{line: 781, col: 5, offset: 20152},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 781, col: 5, offset: 20152},
						run: (*parser).callonsdouble2,
						expr: &seqExpr{
							pos: position{line: 781, col: 5, offset: 20152},
							exprs: []interface{}{
								&oneOrMoreExpr{
									pos: position{line: 781, col: 5, offset: 20152},
									expr: &ruleRefExpr{
										pos:  position{line: 781, col: 5, offset: 20152},
										name: "doubleInteger",
									},
								},
								&litMatcher{
									pos:        position{line: 781, col: 20, offset: 20167},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 781, col: 24, offset: 20171},
									expr: &ruleRefExpr{
										pos:  position{line: 781, col: 24, offset: 20171},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 781, col: 37, offset: 20184},
									expr: &ruleRefExpr{
										pos:  position{line: 781, col: 37, offset: 20184},
										name: "exponentPart",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 784, col: 5, offset: 20243},
						run: (*parser).callonsdouble11,
						expr: &seqExpr{
							pos: position{line: 784, col: 5, offset: 20243},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 784, col: 5, offset: 20243},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 784, col: 9, offset: 20247},
									expr: &ruleRefExpr{
										pos:  position{line: 784, col: 9, offset: 20247},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 784, col: 22, offset: 20260},
									expr: &ruleRefExpr{
										pos:  position{line: 784, col: 22, offset: 20260},
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
			pos:  position{line: 788, col: 1, offset: 20316},
			expr: &choiceExpr{
				pos: position{line: 789, col: 5, offset: 20334},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 789, col: 5, offset: 20334},
						val:        "0",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 790, col: 5, offset: 20342},
						exprs: []interface{}{
							&charClassMatcher{
								pos:        position{line: 790, col: 5, offset: 20342},
								val:        "[1-9]",
								ranges:     []rune{'1', '9'},
								ignoreCase: false,
								inverted:   false,
							},
							&zeroOrMoreExpr{
								pos: position{line: 790, col: 11, offset: 20348},
								expr: &charClassMatcher{
									pos:        position{line: 790, col: 11, offset: 20348},
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
			pos:  position{line: 792, col: 1, offset: 20356},
			expr: &charClassMatcher{
				pos:        position{line: 792, col: 15, offset: 20370},
				val:        "[0-9]",
				ranges:     []rune{'0', '9'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "signedInteger",
			pos:  position{line: 794, col: 1, offset: 20377},
			expr: &seqExpr{
				pos: position{line: 794, col: 17, offset: 20393},
				exprs: []interface{}{
					&zeroOrOneExpr{
						pos: position{line: 794, col: 17, offset: 20393},
						expr: &charClassMatcher{
							pos:        position{line: 794, col: 17, offset: 20393},
							val:        "[+-]",
							chars:      []rune{'+', '-'},
							ignoreCase: false,
							inverted:   false,
						},
					},
					&oneOrMoreExpr{
						pos: position{line: 794, col: 23, offset: 20399},
						expr: &ruleRefExpr{
							pos:  position{line: 794, col: 23, offset: 20399},
							name: "doubleDigit",
						},
					},
				},
			},
		},
		{
			name: "exponentPart",
			pos:  position{line: 796, col: 1, offset: 20413},
			expr: &seqExpr{
				pos: position{line: 796, col: 16, offset: 20428},
				exprs: []interface{}{
					&litMatcher{
						pos:        position{line: 796, col: 16, offset: 20428},
						val:        "e",
						ignoreCase: true,
					},
					&ruleRefExpr{
						pos:  position{line: 796, col: 21, offset: 20433},
						name: "signedInteger",
					},
				},
			},
		},
		{
			name: "h16",
			pos:  position{line: 798, col: 1, offset: 20448},
			expr: &actionExpr{
				pos: position{line: 798, col: 7, offset: 20454},
				run: (*parser).callonh161,
				expr: &labeledExpr{
					pos:   position{line: 798, col: 7, offset: 20454},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 798, col: 13, offset: 20460},
						expr: &ruleRefExpr{
							pos:  position{line: 798, col: 13, offset: 20460},
							name: "hexdigit",
						},
					},
				},
			},
		},
		{
			name: "hexdigit",
			pos:  position{line: 800, col: 1, offset: 20502},
			expr: &charClassMatcher{
				pos:        position{line: 800, col: 12, offset: 20513},
				val:        "[0-9a-fA-F]",
				ranges:     []rune{'0', '9', 'a', 'f', 'A', 'F'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name:        "boomWord",
			displayName: "\"boomWord\"",
			pos:         position{line: 802, col: 1, offset: 20526},
			expr: &actionExpr{
				pos: position{line: 802, col: 23, offset: 20548},
				run: (*parser).callonboomWord1,
				expr: &labeledExpr{
					pos:   position{line: 802, col: 23, offset: 20548},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 802, col: 29, offset: 20554},
						expr: &ruleRefExpr{
							pos:  position{line: 802, col: 29, offset: 20554},
							name: "boomWordPart",
						},
					},
				},
			},
		},
		{
			name: "boomWordPart",
			pos:  position{line: 804, col: 1, offset: 20600},
			expr: &seqExpr{
				pos: position{line: 805, col: 5, offset: 20617},
				exprs: []interface{}{
					&notExpr{
						pos: position{line: 805, col: 5, offset: 20617},
						expr: &choiceExpr{
							pos: position{line: 805, col: 7, offset: 20619},
							alternatives: []interface{}{
								&charClassMatcher{
									pos:        position{line: 805, col: 7, offset: 20619},
									val:        "[\\x00-\\x1F\\x5C(),!><=\\x22|\\x27;]",
									chars:      []rune{'\\', '(', ')', ',', '!', '>', '<', '=', '"', '|', '\'', ';'},
									ranges:     []rune{'\x00', '\x1f'},
									ignoreCase: false,
									inverted:   false,
								},
								&ruleRefExpr{
									pos:  position{line: 805, col: 42, offset: 20654},
									name: "ws",
								},
							},
						},
					},
					&anyMatcher{
						line: 805, col: 46, offset: 20658,
					},
				},
			},
		},
		{
			name: "quotedString",
			pos:  position{line: 807, col: 1, offset: 20661},
			expr: &choiceExpr{
				pos: position{line: 808, col: 5, offset: 20678},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 808, col: 5, offset: 20678},
						run: (*parser).callonquotedString2,
						expr: &seqExpr{
							pos: position{line: 808, col: 5, offset: 20678},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 808, col: 5, offset: 20678},
									val:        "\"",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 808, col: 9, offset: 20682},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 808, col: 11, offset: 20684},
										expr: &ruleRefExpr{
											pos:  position{line: 808, col: 11, offset: 20684},
											name: "doubleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 808, col: 29, offset: 20702},
									val:        "\"",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 809, col: 5, offset: 20739},
						run: (*parser).callonquotedString9,
						expr: &seqExpr{
							pos: position{line: 809, col: 5, offset: 20739},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 809, col: 5, offset: 20739},
									val:        "'",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 809, col: 9, offset: 20743},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 809, col: 11, offset: 20745},
										expr: &ruleRefExpr{
											pos:  position{line: 809, col: 11, offset: 20745},
											name: "singleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 809, col: 29, offset: 20763},
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
			pos:  position{line: 811, col: 1, offset: 20797},
			expr: &choiceExpr{
				pos: position{line: 812, col: 5, offset: 20818},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 812, col: 5, offset: 20818},
						run: (*parser).callondoubleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 812, col: 5, offset: 20818},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 812, col: 5, offset: 20818},
									expr: &choiceExpr{
										pos: position{line: 812, col: 7, offset: 20820},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 812, col: 7, offset: 20820},
												val:        "\"",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 812, col: 13, offset: 20826},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 812, col: 26, offset: 20839,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 813, col: 5, offset: 20876},
						run: (*parser).callondoubleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 813, col: 5, offset: 20876},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 813, col: 5, offset: 20876},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 813, col: 10, offset: 20881},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 813, col: 12, offset: 20883},
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
			pos:  position{line: 815, col: 1, offset: 20917},
			expr: &choiceExpr{
				pos: position{line: 816, col: 5, offset: 20938},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 816, col: 5, offset: 20938},
						run: (*parser).callonsingleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 816, col: 5, offset: 20938},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 816, col: 5, offset: 20938},
									expr: &choiceExpr{
										pos: position{line: 816, col: 7, offset: 20940},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 816, col: 7, offset: 20940},
												val:        "'",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 816, col: 13, offset: 20946},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 816, col: 26, offset: 20959,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 817, col: 5, offset: 20996},
						run: (*parser).callonsingleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 817, col: 5, offset: 20996},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 817, col: 5, offset: 20996},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 817, col: 10, offset: 21001},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 817, col: 12, offset: 21003},
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
			pos:  position{line: 819, col: 1, offset: 21037},
			expr: &choiceExpr{
				pos: position{line: 819, col: 18, offset: 21054},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 819, col: 18, offset: 21054},
						name: "singleCharEscape",
					},
					&ruleRefExpr{
						pos:  position{line: 819, col: 37, offset: 21073},
						name: "unicodeEscape",
					},
				},
			},
		},
		{
			name: "singleCharEscape",
			pos:  position{line: 821, col: 1, offset: 21088},
			expr: &choiceExpr{
				pos: position{line: 822, col: 5, offset: 21109},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 822, col: 5, offset: 21109},
						val:        "'",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 823, col: 5, offset: 21117},
						val:        "\"",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 824, col: 5, offset: 21125},
						val:        "\\",
						ignoreCase: false,
					},
					&actionExpr{
						pos: position{line: 825, col: 5, offset: 21134},
						run: (*parser).callonsingleCharEscape5,
						expr: &litMatcher{
							pos:        position{line: 825, col: 5, offset: 21134},
							val:        "b",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 826, col: 5, offset: 21163},
						run: (*parser).callonsingleCharEscape7,
						expr: &litMatcher{
							pos:        position{line: 826, col: 5, offset: 21163},
							val:        "f",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 827, col: 5, offset: 21192},
						run: (*parser).callonsingleCharEscape9,
						expr: &litMatcher{
							pos:        position{line: 827, col: 5, offset: 21192},
							val:        "n",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 828, col: 5, offset: 21221},
						run: (*parser).callonsingleCharEscape11,
						expr: &litMatcher{
							pos:        position{line: 828, col: 5, offset: 21221},
							val:        "r",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 829, col: 5, offset: 21250},
						run: (*parser).callonsingleCharEscape13,
						expr: &litMatcher{
							pos:        position{line: 829, col: 5, offset: 21250},
							val:        "t",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 830, col: 5, offset: 21279},
						run: (*parser).callonsingleCharEscape15,
						expr: &litMatcher{
							pos:        position{line: 830, col: 5, offset: 21279},
							val:        "v",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "unicodeEscape",
			pos:  position{line: 832, col: 1, offset: 21305},
			expr: &seqExpr{
				pos: position{line: 833, col: 5, offset: 21323},
				exprs: []interface{}{
					&litMatcher{
						pos:        position{line: 833, col: 5, offset: 21323},
						val:        "u",
						ignoreCase: false,
					},
					&ruleRefExpr{
						pos:  position{line: 833, col: 9, offset: 21327},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 833, col: 18, offset: 21336},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 833, col: 27, offset: 21345},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 833, col: 36, offset: 21354},
						name: "hexdigit",
					},
				},
			},
		},
		{
			name: "reString",
			pos:  position{line: 835, col: 1, offset: 21364},
			expr: &actionExpr{
				pos: position{line: 836, col: 5, offset: 21377},
				run: (*parser).callonreString1,
				expr: &seqExpr{
					pos: position{line: 836, col: 5, offset: 21377},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 836, col: 5, offset: 21377},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 836, col: 9, offset: 21381},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 836, col: 11, offset: 21383},
								name: "reBody",
							},
						},
						&litMatcher{
							pos:        position{line: 836, col: 18, offset: 21390},
							val:        "/",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reBody",
			pos:  position{line: 838, col: 1, offset: 21413},
			expr: &actionExpr{
				pos: position{line: 839, col: 5, offset: 21424},
				run: (*parser).callonreBody1,
				expr: &oneOrMoreExpr{
					pos: position{line: 839, col: 5, offset: 21424},
					expr: &choiceExpr{
						pos: position{line: 839, col: 6, offset: 21425},
						alternatives: []interface{}{
							&charClassMatcher{
								pos:        position{line: 839, col: 6, offset: 21425},
								val:        "[^/\\\\]",
								chars:      []rune{'/', '\\'},
								ignoreCase: false,
								inverted:   true,
							},
							&litMatcher{
								pos:        position{line: 839, col: 13, offset: 21432},
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
			pos:  position{line: 841, col: 1, offset: 21472},
			expr: &charClassMatcher{
				pos:        position{line: 842, col: 5, offset: 21488},
				val:        "[\\x00-\\x1f\\\\]",
				chars:      []rune{'\\'},
				ranges:     []rune{'\x00', '\x1f'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "ws",
			pos:  position{line: 844, col: 1, offset: 21503},
			expr: &choiceExpr{
				pos: position{line: 845, col: 5, offset: 21510},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 845, col: 5, offset: 21510},
						val:        "\t",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 846, col: 5, offset: 21519},
						val:        "\v",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 847, col: 5, offset: 21528},
						val:        "\f",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 848, col: 5, offset: 21537},
						val:        " ",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 849, col: 5, offset: 21545},
						val:        "\u00a0",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 850, col: 5, offset: 21558},
						val:        "\ufeff",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name:        "_",
			displayName: "\"whitespace\"",
			pos:         position{line: 852, col: 1, offset: 21568},
			expr: &oneOrMoreExpr{
				pos: position{line: 852, col: 18, offset: 21585},
				expr: &ruleRefExpr{
					pos:  position{line: 852, col: 18, offset: 21585},
					name: "ws",
				},
			},
		},
		{
			name: "EOF",
			pos:  position{line: 854, col: 1, offset: 21590},
			expr: &notExpr{
				pos: position{line: 854, col: 7, offset: 21596},
				expr: &anyMatcher{
					line: 854, col: 8, offset: 21597,
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
	return makeCompareAny(fieldComparator, false, v), nil

}

func (p *parser) callonsearchPred2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchPred2(stack["fieldComparator"], stack["v"])
}

func (c *current) onsearchPred13(fieldComparator, v interface{}) (interface{}, error) {
	return makeCompareAny(fieldComparator, true, v), nil

}

func (p *parser) callonsearchPred13() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchPred13(stack["fieldComparator"], stack["v"])
}

func (c *current) onsearchPred24() (interface{}, error) {
	return makeBooleanLiteral(true), nil

}

func (p *parser) callonsearchPred24() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchPred24()
}

func (c *current) onsearchPred26(f, fieldComparator, v interface{}) (interface{}, error) {
	return makeCompareField(fieldComparator, f, v), nil

}

func (p *parser) callonsearchPred26() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchPred26(stack["f"], stack["fieldComparator"], stack["v"])
}

func (c *current) onsearchPred38(v interface{}) (interface{}, error) {
	return makeCompareAny("in", false, v), nil

}

func (p *parser) callonsearchPred38() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchPred38(stack["v"])
}

func (c *current) onsearchPred48(v, f interface{}) (interface{}, error) {
	return makeCompareField("in", f, v), nil

}

func (p *parser) callonsearchPred48() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchPred48(stack["v"], stack["f"])
}

func (c *current) onsearchPred59(v interface{}) (interface{}, error) {
	ss := makeSearchString(v)
	if getValueType(v) == "string" {
		return ss, nil
	}
	ss = makeSearchString(makeTypedValue("string", string(c.text)))
	return makeOrChain(ss, []interface{}{makeCompareAny("eql", true, v), makeCompareAny("in", true, v)}), nil

}

func (p *parser) callonsearchPred59() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchPred59(stack["v"])
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

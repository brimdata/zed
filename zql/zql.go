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

func makeFieldRead(field interface{}) *ast.FieldRead {
	return &ast.FieldRead{ast.Node{"FieldRead"}, field.(string)}
}

func makeFieldCall(fn, field, paramIn interface{}) *ast.FieldCall {
	var param string
	if paramIn != nil {
		param = paramIn.(string)
	}
	return &ast.FieldCall{ast.Node{"FieldCall"}, fn.(string), field.(string), param}
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

func makeSortProc(fieldsIn, dirIn, limitIn interface{}) *ast.SortProc {
	fields := stringArray(fieldsIn)
	sortdir := dirIn.(int)
	var limit int
	if limitIn != nil {
		limit = limitIn.(int)
	}
	return &ast.SortProc{ast.Node{"SortProc"}, limit, fields, sortdir}
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
			pos:  position{line: 277, col: 1, offset: 7884},
			expr: &actionExpr{
				pos: position{line: 277, col: 9, offset: 7892},
				run: (*parser).callonstart1,
				expr: &seqExpr{
					pos: position{line: 277, col: 9, offset: 7892},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 277, col: 9, offset: 7892},
							expr: &ruleRefExpr{
								pos:  position{line: 277, col: 9, offset: 7892},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 277, col: 12, offset: 7895},
							label: "ast",
							expr: &ruleRefExpr{
								pos:  position{line: 277, col: 16, offset: 7899},
								name: "boomCommand",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 277, col: 28, offset: 7911},
							expr: &ruleRefExpr{
								pos:  position{line: 277, col: 28, offset: 7911},
								name: "_",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 277, col: 31, offset: 7914},
							name: "EOF",
						},
					},
				},
			},
		},
		{
			name: "boomCommand",
			pos:  position{line: 279, col: 1, offset: 7939},
			expr: &choiceExpr{
				pos: position{line: 280, col: 5, offset: 7955},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 280, col: 5, offset: 7955},
						run: (*parser).callonboomCommand2,
						expr: &labeledExpr{
							pos:   position{line: 280, col: 5, offset: 7955},
							label: "procs",
							expr: &ruleRefExpr{
								pos:  position{line: 280, col: 11, offset: 7961},
								name: "procChain",
							},
						},
					},
					&actionExpr{
						pos: position{line: 284, col: 5, offset: 8134},
						run: (*parser).callonboomCommand5,
						expr: &seqExpr{
							pos: position{line: 284, col: 5, offset: 8134},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 284, col: 5, offset: 8134},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 284, col: 7, offset: 8136},
										name: "search",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 284, col: 14, offset: 8143},
									expr: &ruleRefExpr{
										pos:  position{line: 284, col: 14, offset: 8143},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 284, col: 17, offset: 8146},
									label: "rest",
									expr: &zeroOrMoreExpr{
										pos: position{line: 284, col: 22, offset: 8151},
										expr: &ruleRefExpr{
											pos:  position{line: 284, col: 22, offset: 8151},
											name: "chainedProc",
										},
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 291, col: 5, offset: 8361},
						run: (*parser).callonboomCommand14,
						expr: &labeledExpr{
							pos:   position{line: 291, col: 5, offset: 8361},
							label: "s",
							expr: &ruleRefExpr{
								pos:  position{line: 291, col: 7, offset: 8363},
								name: "search",
							},
						},
					},
				},
			},
		},
		{
			name: "procChain",
			pos:  position{line: 295, col: 1, offset: 8434},
			expr: &actionExpr{
				pos: position{line: 296, col: 5, offset: 8448},
				run: (*parser).callonprocChain1,
				expr: &seqExpr{
					pos: position{line: 296, col: 5, offset: 8448},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 296, col: 5, offset: 8448},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 296, col: 11, offset: 8454},
								name: "proc",
							},
						},
						&labeledExpr{
							pos:   position{line: 296, col: 16, offset: 8459},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 296, col: 21, offset: 8464},
								expr: &ruleRefExpr{
									pos:  position{line: 296, col: 21, offset: 8464},
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
			pos:  position{line: 304, col: 1, offset: 8650},
			expr: &actionExpr{
				pos: position{line: 304, col: 15, offset: 8664},
				run: (*parser).callonchainedProc1,
				expr: &seqExpr{
					pos: position{line: 304, col: 15, offset: 8664},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 304, col: 15, offset: 8664},
							expr: &ruleRefExpr{
								pos:  position{line: 304, col: 15, offset: 8664},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 304, col: 18, offset: 8667},
							val:        "|",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 304, col: 22, offset: 8671},
							expr: &ruleRefExpr{
								pos:  position{line: 304, col: 22, offset: 8671},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 304, col: 25, offset: 8674},
							label: "p",
							expr: &ruleRefExpr{
								pos:  position{line: 304, col: 27, offset: 8676},
								name: "proc",
							},
						},
					},
				},
			},
		},
		{
			name: "search",
			pos:  position{line: 306, col: 1, offset: 8700},
			expr: &actionExpr{
				pos: position{line: 307, col: 5, offset: 8711},
				run: (*parser).callonsearch1,
				expr: &labeledExpr{
					pos:   position{line: 307, col: 5, offset: 8711},
					label: "expr",
					expr: &ruleRefExpr{
						pos:  position{line: 307, col: 10, offset: 8716},
						name: "searchExpr",
					},
				},
			},
		},
		{
			name: "searchExpr",
			pos:  position{line: 311, col: 1, offset: 8775},
			expr: &actionExpr{
				pos: position{line: 312, col: 5, offset: 8790},
				run: (*parser).callonsearchExpr1,
				expr: &seqExpr{
					pos: position{line: 312, col: 5, offset: 8790},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 312, col: 5, offset: 8790},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 312, col: 11, offset: 8796},
								name: "searchTerm",
							},
						},
						&labeledExpr{
							pos:   position{line: 312, col: 22, offset: 8807},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 312, col: 27, offset: 8812},
								expr: &ruleRefExpr{
									pos:  position{line: 312, col: 27, offset: 8812},
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
			pos:  position{line: 316, col: 1, offset: 8880},
			expr: &actionExpr{
				pos: position{line: 316, col: 18, offset: 8897},
				run: (*parser).callonoredSearchTerm1,
				expr: &seqExpr{
					pos: position{line: 316, col: 18, offset: 8897},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 316, col: 18, offset: 8897},
							name: "_",
						},
						&ruleRefExpr{
							pos:  position{line: 316, col: 20, offset: 8899},
							name: "orToken",
						},
						&ruleRefExpr{
							pos:  position{line: 316, col: 28, offset: 8907},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 316, col: 30, offset: 8909},
							label: "t",
							expr: &ruleRefExpr{
								pos:  position{line: 316, col: 32, offset: 8911},
								name: "searchTerm",
							},
						},
					},
				},
			},
		},
		{
			name: "searchTerm",
			pos:  position{line: 318, col: 1, offset: 8941},
			expr: &actionExpr{
				pos: position{line: 319, col: 5, offset: 8956},
				run: (*parser).callonsearchTerm1,
				expr: &seqExpr{
					pos: position{line: 319, col: 5, offset: 8956},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 319, col: 5, offset: 8956},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 319, col: 11, offset: 8962},
								name: "searchFactor",
							},
						},
						&labeledExpr{
							pos:   position{line: 319, col: 24, offset: 8975},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 319, col: 29, offset: 8980},
								expr: &ruleRefExpr{
									pos:  position{line: 319, col: 29, offset: 8980},
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
			pos:  position{line: 323, col: 1, offset: 9050},
			expr: &actionExpr{
				pos: position{line: 323, col: 19, offset: 9068},
				run: (*parser).callonandedSearchTerm1,
				expr: &seqExpr{
					pos: position{line: 323, col: 19, offset: 9068},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 323, col: 19, offset: 9068},
							name: "_",
						},
						&zeroOrOneExpr{
							pos: position{line: 323, col: 21, offset: 9070},
							expr: &seqExpr{
								pos: position{line: 323, col: 22, offset: 9071},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 323, col: 22, offset: 9071},
										name: "andToken",
									},
									&ruleRefExpr{
										pos:  position{line: 323, col: 31, offset: 9080},
										name: "_",
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 323, col: 35, offset: 9084},
							label: "f",
							expr: &ruleRefExpr{
								pos:  position{line: 323, col: 37, offset: 9086},
								name: "searchFactor",
							},
						},
					},
				},
			},
		},
		{
			name: "searchFactor",
			pos:  position{line: 325, col: 1, offset: 9118},
			expr: &choiceExpr{
				pos: position{line: 326, col: 5, offset: 9135},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 326, col: 5, offset: 9135},
						run: (*parser).callonsearchFactor2,
						expr: &seqExpr{
							pos: position{line: 326, col: 5, offset: 9135},
							exprs: []interface{}{
								&choiceExpr{
									pos: position{line: 326, col: 6, offset: 9136},
									alternatives: []interface{}{
										&seqExpr{
											pos: position{line: 326, col: 6, offset: 9136},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 326, col: 6, offset: 9136},
													name: "notToken",
												},
												&ruleRefExpr{
													pos:  position{line: 326, col: 15, offset: 9145},
													name: "_",
												},
											},
										},
										&seqExpr{
											pos: position{line: 326, col: 19, offset: 9149},
											exprs: []interface{}{
												&litMatcher{
													pos:        position{line: 326, col: 19, offset: 9149},
													val:        "!",
													ignoreCase: false,
												},
												&zeroOrOneExpr{
													pos: position{line: 326, col: 23, offset: 9153},
													expr: &ruleRefExpr{
														pos:  position{line: 326, col: 23, offset: 9153},
														name: "_",
													},
												},
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 326, col: 27, offset: 9157},
									label: "e",
									expr: &ruleRefExpr{
										pos:  position{line: 326, col: 29, offset: 9159},
										name: "searchExpr",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 329, col: 5, offset: 9218},
						run: (*parser).callonsearchFactor14,
						expr: &seqExpr{
							pos: position{line: 329, col: 5, offset: 9218},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 329, col: 5, offset: 9218},
									expr: &litMatcher{
										pos:        position{line: 329, col: 7, offset: 9220},
										val:        "-",
										ignoreCase: false,
									},
								},
								&labeledExpr{
									pos:   position{line: 329, col: 12, offset: 9225},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 329, col: 14, offset: 9227},
										name: "searchPred",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 330, col: 5, offset: 9260},
						run: (*parser).callonsearchFactor20,
						expr: &seqExpr{
							pos: position{line: 330, col: 5, offset: 9260},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 330, col: 5, offset: 9260},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 330, col: 9, offset: 9264},
									expr: &ruleRefExpr{
										pos:  position{line: 330, col: 9, offset: 9264},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 330, col: 12, offset: 9267},
									label: "expr",
									expr: &ruleRefExpr{
										pos:  position{line: 330, col: 17, offset: 9272},
										name: "searchExpr",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 330, col: 28, offset: 9283},
									expr: &ruleRefExpr{
										pos:  position{line: 330, col: 28, offset: 9283},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 330, col: 31, offset: 9286},
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
			pos:  position{line: 332, col: 1, offset: 9312},
			expr: &choiceExpr{
				pos: position{line: 333, col: 5, offset: 9327},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 333, col: 5, offset: 9327},
						run: (*parser).callonsearchPred2,
						expr: &seqExpr{
							pos: position{line: 333, col: 5, offset: 9327},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 333, col: 5, offset: 9327},
									val:        "*",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 333, col: 9, offset: 9331},
									expr: &ruleRefExpr{
										pos:  position{line: 333, col: 9, offset: 9331},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 333, col: 12, offset: 9334},
									label: "fieldComparator",
									expr: &ruleRefExpr{
										pos:  position{line: 333, col: 28, offset: 9350},
										name: "equalityToken",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 333, col: 42, offset: 9364},
									expr: &ruleRefExpr{
										pos:  position{line: 333, col: 42, offset: 9364},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 333, col: 45, offset: 9367},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 333, col: 47, offset: 9369},
										name: "searchValue",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 336, col: 5, offset: 9446},
						run: (*parser).callonsearchPred13,
						expr: &litMatcher{
							pos:        position{line: 336, col: 5, offset: 9446},
							val:        "*",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 339, col: 5, offset: 9505},
						run: (*parser).callonsearchPred15,
						expr: &seqExpr{
							pos: position{line: 339, col: 5, offset: 9505},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 339, col: 5, offset: 9505},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 339, col: 7, offset: 9507},
										name: "fieldExpr",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 339, col: 17, offset: 9517},
									expr: &ruleRefExpr{
										pos:  position{line: 339, col: 17, offset: 9517},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 339, col: 20, offset: 9520},
									label: "fieldComparator",
									expr: &ruleRefExpr{
										pos:  position{line: 339, col: 36, offset: 9536},
										name: "equalityToken",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 339, col: 50, offset: 9550},
									expr: &ruleRefExpr{
										pos:  position{line: 339, col: 50, offset: 9550},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 339, col: 53, offset: 9553},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 339, col: 55, offset: 9555},
										name: "searchValue",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 342, col: 5, offset: 9637},
						run: (*parser).callonsearchPred27,
						expr: &seqExpr{
							pos: position{line: 342, col: 5, offset: 9637},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 342, col: 5, offset: 9637},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 342, col: 7, offset: 9639},
										name: "searchValue",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 342, col: 19, offset: 9651},
									expr: &ruleRefExpr{
										pos:  position{line: 342, col: 19, offset: 9651},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 342, col: 22, offset: 9654},
									name: "inToken",
								},
								&zeroOrOneExpr{
									pos: position{line: 342, col: 30, offset: 9662},
									expr: &ruleRefExpr{
										pos:  position{line: 342, col: 30, offset: 9662},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 342, col: 33, offset: 9665},
									val:        "*",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 345, col: 5, offset: 9723},
						run: (*parser).callonsearchPred37,
						expr: &seqExpr{
							pos: position{line: 345, col: 5, offset: 9723},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 345, col: 5, offset: 9723},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 345, col: 7, offset: 9725},
										name: "searchValue",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 345, col: 19, offset: 9737},
									expr: &ruleRefExpr{
										pos:  position{line: 345, col: 19, offset: 9737},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 345, col: 22, offset: 9740},
									name: "inToken",
								},
								&zeroOrOneExpr{
									pos: position{line: 345, col: 30, offset: 9748},
									expr: &ruleRefExpr{
										pos:  position{line: 345, col: 30, offset: 9748},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 345, col: 33, offset: 9751},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 345, col: 35, offset: 9753},
										name: "fieldRead",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 348, col: 5, offset: 9822},
						run: (*parser).callonsearchPred48,
						expr: &labeledExpr{
							pos:   position{line: 348, col: 5, offset: 9822},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 348, col: 7, offset: 9824},
								name: "searchValue",
							},
						},
					},
				},
			},
		},
		{
			name: "searchValue",
			pos:  position{line: 357, col: 1, offset: 10118},
			expr: &choiceExpr{
				pos: position{line: 358, col: 5, offset: 10134},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 358, col: 5, offset: 10134},
						run: (*parser).callonsearchValue2,
						expr: &labeledExpr{
							pos:   position{line: 358, col: 5, offset: 10134},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 358, col: 7, offset: 10136},
								name: "quotedString",
							},
						},
					},
					&actionExpr{
						pos: position{line: 361, col: 5, offset: 10207},
						run: (*parser).callonsearchValue5,
						expr: &labeledExpr{
							pos:   position{line: 361, col: 5, offset: 10207},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 361, col: 7, offset: 10209},
								name: "reString",
							},
						},
					},
					&actionExpr{
						pos: position{line: 364, col: 5, offset: 10276},
						run: (*parser).callonsearchValue8,
						expr: &labeledExpr{
							pos:   position{line: 364, col: 5, offset: 10276},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 364, col: 7, offset: 10278},
								name: "port",
							},
						},
					},
					&actionExpr{
						pos: position{line: 367, col: 5, offset: 10337},
						run: (*parser).callonsearchValue11,
						expr: &labeledExpr{
							pos:   position{line: 367, col: 5, offset: 10337},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 367, col: 7, offset: 10339},
								name: "ip6subnet",
							},
						},
					},
					&actionExpr{
						pos: position{line: 370, col: 5, offset: 10407},
						run: (*parser).callonsearchValue14,
						expr: &labeledExpr{
							pos:   position{line: 370, col: 5, offset: 10407},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 370, col: 7, offset: 10409},
								name: "ip6addr",
							},
						},
					},
					&actionExpr{
						pos: position{line: 373, col: 5, offset: 10473},
						run: (*parser).callonsearchValue17,
						expr: &labeledExpr{
							pos:   position{line: 373, col: 5, offset: 10473},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 373, col: 7, offset: 10475},
								name: "subnet",
							},
						},
					},
					&actionExpr{
						pos: position{line: 376, col: 5, offset: 10540},
						run: (*parser).callonsearchValue20,
						expr: &labeledExpr{
							pos:   position{line: 376, col: 5, offset: 10540},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 376, col: 7, offset: 10542},
								name: "addr",
							},
						},
					},
					&actionExpr{
						pos: position{line: 379, col: 5, offset: 10603},
						run: (*parser).callonsearchValue23,
						expr: &labeledExpr{
							pos:   position{line: 379, col: 5, offset: 10603},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 379, col: 7, offset: 10605},
								name: "sdouble",
							},
						},
					},
					&actionExpr{
						pos: position{line: 382, col: 5, offset: 10671},
						run: (*parser).callonsearchValue26,
						expr: &seqExpr{
							pos: position{line: 382, col: 5, offset: 10671},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 382, col: 5, offset: 10671},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 382, col: 7, offset: 10673},
										name: "sinteger",
									},
								},
								&notExpr{
									pos: position{line: 382, col: 16, offset: 10682},
									expr: &ruleRefExpr{
										pos:  position{line: 382, col: 17, offset: 10683},
										name: "boomWord",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 385, col: 5, offset: 10747},
						run: (*parser).callonsearchValue32,
						expr: &seqExpr{
							pos: position{line: 385, col: 5, offset: 10747},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 385, col: 5, offset: 10747},
									expr: &seqExpr{
										pos: position{line: 385, col: 7, offset: 10749},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 385, col: 7, offset: 10749},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 385, col: 22, offset: 10764},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 385, col: 25, offset: 10767},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 385, col: 27, offset: 10769},
										name: "booleanLiteral",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 386, col: 5, offset: 10806},
						run: (*parser).callonsearchValue40,
						expr: &seqExpr{
							pos: position{line: 386, col: 5, offset: 10806},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 386, col: 5, offset: 10806},
									expr: &seqExpr{
										pos: position{line: 386, col: 7, offset: 10808},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 386, col: 7, offset: 10808},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 386, col: 22, offset: 10823},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 386, col: 25, offset: 10826},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 386, col: 27, offset: 10828},
										name: "unsetLiteral",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 387, col: 5, offset: 10863},
						run: (*parser).callonsearchValue48,
						expr: &seqExpr{
							pos: position{line: 387, col: 5, offset: 10863},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 387, col: 5, offset: 10863},
									expr: &seqExpr{
										pos: position{line: 387, col: 7, offset: 10865},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 387, col: 7, offset: 10865},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 387, col: 22, offset: 10880},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 387, col: 25, offset: 10883},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 387, col: 27, offset: 10885},
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
			pos:  position{line: 395, col: 1, offset: 11109},
			expr: &choiceExpr{
				pos: position{line: 396, col: 5, offset: 11128},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 396, col: 5, offset: 11128},
						name: "andToken",
					},
					&ruleRefExpr{
						pos:  position{line: 397, col: 5, offset: 11141},
						name: "orToken",
					},
					&ruleRefExpr{
						pos:  position{line: 398, col: 5, offset: 11153},
						name: "inToken",
					},
				},
			},
		},
		{
			name: "booleanLiteral",
			pos:  position{line: 400, col: 1, offset: 11162},
			expr: &choiceExpr{
				pos: position{line: 401, col: 5, offset: 11181},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 401, col: 5, offset: 11181},
						run: (*parser).callonbooleanLiteral2,
						expr: &litMatcher{
							pos:        position{line: 401, col: 5, offset: 11181},
							val:        "true",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 402, col: 5, offset: 11249},
						run: (*parser).callonbooleanLiteral4,
						expr: &litMatcher{
							pos:        position{line: 402, col: 5, offset: 11249},
							val:        "false",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "unsetLiteral",
			pos:  position{line: 404, col: 1, offset: 11315},
			expr: &actionExpr{
				pos: position{line: 405, col: 5, offset: 11332},
				run: (*parser).callonunsetLiteral1,
				expr: &litMatcher{
					pos:        position{line: 405, col: 5, offset: 11332},
					val:        "nil",
					ignoreCase: false,
				},
			},
		},
		{
			name: "procList",
			pos:  position{line: 407, col: 1, offset: 11393},
			expr: &actionExpr{
				pos: position{line: 408, col: 5, offset: 11406},
				run: (*parser).callonprocList1,
				expr: &seqExpr{
					pos: position{line: 408, col: 5, offset: 11406},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 408, col: 5, offset: 11406},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 408, col: 11, offset: 11412},
								name: "procChain",
							},
						},
						&labeledExpr{
							pos:   position{line: 408, col: 21, offset: 11422},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 408, col: 26, offset: 11427},
								expr: &ruleRefExpr{
									pos:  position{line: 408, col: 26, offset: 11427},
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
			pos:  position{line: 417, col: 1, offset: 11651},
			expr: &actionExpr{
				pos: position{line: 418, col: 5, offset: 11669},
				run: (*parser).callonparallelChain1,
				expr: &seqExpr{
					pos: position{line: 418, col: 5, offset: 11669},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 418, col: 5, offset: 11669},
							expr: &ruleRefExpr{
								pos:  position{line: 418, col: 5, offset: 11669},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 418, col: 8, offset: 11672},
							val:        ";",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 418, col: 12, offset: 11676},
							expr: &ruleRefExpr{
								pos:  position{line: 418, col: 12, offset: 11676},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 418, col: 15, offset: 11679},
							label: "ch",
							expr: &ruleRefExpr{
								pos:  position{line: 418, col: 18, offset: 11682},
								name: "procChain",
							},
						},
					},
				},
			},
		},
		{
			name: "proc",
			pos:  position{line: 420, col: 1, offset: 11732},
			expr: &choiceExpr{
				pos: position{line: 421, col: 5, offset: 11741},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 421, col: 5, offset: 11741},
						name: "simpleProc",
					},
					&ruleRefExpr{
						pos:  position{line: 422, col: 5, offset: 11756},
						name: "reducerProc",
					},
					&actionExpr{
						pos: position{line: 423, col: 5, offset: 11772},
						run: (*parser).callonproc4,
						expr: &seqExpr{
							pos: position{line: 423, col: 5, offset: 11772},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 423, col: 5, offset: 11772},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 423, col: 9, offset: 11776},
									expr: &ruleRefExpr{
										pos:  position{line: 423, col: 9, offset: 11776},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 423, col: 12, offset: 11779},
									label: "proc",
									expr: &ruleRefExpr{
										pos:  position{line: 423, col: 17, offset: 11784},
										name: "procList",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 423, col: 26, offset: 11793},
									expr: &ruleRefExpr{
										pos:  position{line: 423, col: 26, offset: 11793},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 423, col: 29, offset: 11796},
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
			pos:  position{line: 427, col: 1, offset: 11832},
			expr: &actionExpr{
				pos: position{line: 428, col: 5, offset: 11844},
				run: (*parser).callongroupBy1,
				expr: &seqExpr{
					pos: position{line: 428, col: 5, offset: 11844},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 428, col: 5, offset: 11844},
							val:        "by",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 428, col: 11, offset: 11850},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 428, col: 13, offset: 11852},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 428, col: 18, offset: 11857},
								name: "fieldList",
							},
						},
					},
				},
			},
		},
		{
			name: "everyDur",
			pos:  position{line: 430, col: 1, offset: 11889},
			expr: &actionExpr{
				pos: position{line: 431, col: 5, offset: 11902},
				run: (*parser).calloneveryDur1,
				expr: &seqExpr{
					pos: position{line: 431, col: 5, offset: 11902},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 431, col: 5, offset: 11902},
							val:        "every",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 431, col: 14, offset: 11911},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 431, col: 16, offset: 11913},
							label: "dur",
							expr: &ruleRefExpr{
								pos:  position{line: 431, col: 20, offset: 11917},
								name: "duration",
							},
						},
					},
				},
			},
		},
		{
			name: "equalityToken",
			pos:  position{line: 433, col: 1, offset: 11947},
			expr: &choiceExpr{
				pos: position{line: 434, col: 5, offset: 11965},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 434, col: 5, offset: 11965},
						run: (*parser).callonequalityToken2,
						expr: &litMatcher{
							pos:        position{line: 434, col: 5, offset: 11965},
							val:        "=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 435, col: 5, offset: 11995},
						run: (*parser).callonequalityToken4,
						expr: &litMatcher{
							pos:        position{line: 435, col: 5, offset: 11995},
							val:        "!=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 436, col: 5, offset: 12027},
						run: (*parser).callonequalityToken6,
						expr: &litMatcher{
							pos:        position{line: 436, col: 5, offset: 12027},
							val:        "<=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 437, col: 5, offset: 12058},
						run: (*parser).callonequalityToken8,
						expr: &litMatcher{
							pos:        position{line: 437, col: 5, offset: 12058},
							val:        ">=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 438, col: 5, offset: 12089},
						run: (*parser).callonequalityToken10,
						expr: &litMatcher{
							pos:        position{line: 438, col: 5, offset: 12089},
							val:        "<",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 439, col: 5, offset: 12118},
						run: (*parser).callonequalityToken12,
						expr: &litMatcher{
							pos:        position{line: 439, col: 5, offset: 12118},
							val:        ">",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "types",
			pos:  position{line: 441, col: 1, offset: 12144},
			expr: &choiceExpr{
				pos: position{line: 442, col: 5, offset: 12154},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 442, col: 5, offset: 12154},
						val:        "bool",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 443, col: 5, offset: 12165},
						val:        "int",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 444, col: 5, offset: 12175},
						val:        "count",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 445, col: 5, offset: 12187},
						val:        "double",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 446, col: 5, offset: 12200},
						val:        "string",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 447, col: 5, offset: 12213},
						val:        "addr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 448, col: 5, offset: 12224},
						val:        "subnet",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 449, col: 5, offset: 12237},
						val:        "port",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "dash",
			pos:  position{line: 451, col: 1, offset: 12245},
			expr: &choiceExpr{
				pos: position{line: 451, col: 8, offset: 12252},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 451, col: 8, offset: 12252},
						val:        "-",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 451, col: 14, offset: 12258},
						val:        "—",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 451, col: 25, offset: 12269},
						val:        "–",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 451, col: 36, offset: 12280},
						exprs: []interface{}{
							&litMatcher{
								pos:        position{line: 451, col: 36, offset: 12280},
								val:        "to",
								ignoreCase: false,
							},
							&andExpr{
								pos: position{line: 451, col: 40, offset: 12284},
								expr: &ruleRefExpr{
									pos:  position{line: 451, col: 42, offset: 12286},
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
			pos:  position{line: 453, col: 1, offset: 12290},
			expr: &litMatcher{
				pos:        position{line: 453, col: 12, offset: 12301},
				val:        "and",
				ignoreCase: false,
			},
		},
		{
			name: "orToken",
			pos:  position{line: 454, col: 1, offset: 12307},
			expr: &litMatcher{
				pos:        position{line: 454, col: 11, offset: 12317},
				val:        "or",
				ignoreCase: false,
			},
		},
		{
			name: "inToken",
			pos:  position{line: 455, col: 1, offset: 12322},
			expr: &litMatcher{
				pos:        position{line: 455, col: 11, offset: 12332},
				val:        "in",
				ignoreCase: false,
			},
		},
		{
			name: "notToken",
			pos:  position{line: 456, col: 1, offset: 12337},
			expr: &litMatcher{
				pos:        position{line: 456, col: 12, offset: 12348},
				val:        "not",
				ignoreCase: false,
			},
		},
		{
			name: "fieldName",
			pos:  position{line: 458, col: 1, offset: 12355},
			expr: &actionExpr{
				pos: position{line: 458, col: 13, offset: 12367},
				run: (*parser).callonfieldName1,
				expr: &seqExpr{
					pos: position{line: 458, col: 13, offset: 12367},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 458, col: 13, offset: 12367},
							name: "fieldNameStart",
						},
						&zeroOrMoreExpr{
							pos: position{line: 458, col: 28, offset: 12382},
							expr: &ruleRefExpr{
								pos:  position{line: 458, col: 28, offset: 12382},
								name: "fieldNameRest",
							},
						},
					},
				},
			},
		},
		{
			name: "fieldNameStart",
			pos:  position{line: 460, col: 1, offset: 12429},
			expr: &charClassMatcher{
				pos:        position{line: 460, col: 18, offset: 12446},
				val:        "[A-Za-z_$]",
				chars:      []rune{'_', '$'},
				ranges:     []rune{'A', 'Z', 'a', 'z'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "fieldNameRest",
			pos:  position{line: 461, col: 1, offset: 12457},
			expr: &choiceExpr{
				pos: position{line: 461, col: 17, offset: 12473},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 461, col: 17, offset: 12473},
						name: "fieldNameStart",
					},
					&charClassMatcher{
						pos:        position{line: 461, col: 34, offset: 12490},
						val:        "[0-9.]",
						chars:      []rune{'.'},
						ranges:     []rune{'0', '9'},
						ignoreCase: false,
						inverted:   false,
					},
				},
			},
		},
		{
			name: "fieldRead",
			pos:  position{line: 463, col: 1, offset: 12498},
			expr: &actionExpr{
				pos: position{line: 464, col: 5, offset: 12512},
				run: (*parser).callonfieldRead1,
				expr: &labeledExpr{
					pos:   position{line: 464, col: 5, offset: 12512},
					label: "field",
					expr: &ruleRefExpr{
						pos:  position{line: 464, col: 11, offset: 12518},
						name: "fieldName",
					},
				},
			},
		},
		{
			name: "fieldExpr",
			pos:  position{line: 468, col: 1, offset: 12575},
			expr: &choiceExpr{
				pos: position{line: 469, col: 5, offset: 12589},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 469, col: 5, offset: 12589},
						run: (*parser).callonfieldExpr2,
						expr: &seqExpr{
							pos: position{line: 469, col: 5, offset: 12589},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 469, col: 5, offset: 12589},
									label: "op",
									expr: &ruleRefExpr{
										pos:  position{line: 469, col: 8, offset: 12592},
										name: "fieldOp",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 469, col: 16, offset: 12600},
									expr: &ruleRefExpr{
										pos:  position{line: 469, col: 16, offset: 12600},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 469, col: 19, offset: 12603},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 469, col: 23, offset: 12607},
									expr: &ruleRefExpr{
										pos:  position{line: 469, col: 23, offset: 12607},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 469, col: 26, offset: 12610},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 469, col: 32, offset: 12616},
										name: "fieldName",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 469, col: 42, offset: 12626},
									expr: &ruleRefExpr{
										pos:  position{line: 469, col: 42, offset: 12626},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 469, col: 45, offset: 12629},
									val:        ")",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 472, col: 5, offset: 12693},
						run: (*parser).callonfieldExpr16,
						expr: &seqExpr{
							pos: position{line: 472, col: 5, offset: 12693},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 472, col: 5, offset: 12693},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 472, col: 11, offset: 12699},
										name: "fieldName",
									},
								},
								&litMatcher{
									pos:        position{line: 472, col: 21, offset: 12709},
									val:        "[",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 472, col: 25, offset: 12713},
									label: "index",
									expr: &ruleRefExpr{
										pos:  position{line: 472, col: 31, offset: 12719},
										name: "sinteger",
									},
								},
								&litMatcher{
									pos:        position{line: 472, col: 40, offset: 12728},
									val:        "]",
									ignoreCase: false,
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 475, col: 5, offset: 12799},
						name: "fieldRead",
					},
				},
			},
		},
		{
			name: "fieldOp",
			pos:  position{line: 477, col: 1, offset: 12810},
			expr: &actionExpr{
				pos: position{line: 478, col: 5, offset: 12822},
				run: (*parser).callonfieldOp1,
				expr: &litMatcher{
					pos:        position{line: 478, col: 5, offset: 12822},
					val:        "len",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldList",
			pos:  position{line: 480, col: 1, offset: 12852},
			expr: &actionExpr{
				pos: position{line: 481, col: 5, offset: 12866},
				run: (*parser).callonfieldList1,
				expr: &seqExpr{
					pos: position{line: 481, col: 5, offset: 12866},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 481, col: 5, offset: 12866},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 481, col: 11, offset: 12872},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 481, col: 21, offset: 12882},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 481, col: 26, offset: 12887},
								expr: &seqExpr{
									pos: position{line: 481, col: 27, offset: 12888},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 481, col: 27, offset: 12888},
											expr: &ruleRefExpr{
												pos:  position{line: 481, col: 27, offset: 12888},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 481, col: 30, offset: 12891},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 481, col: 34, offset: 12895},
											expr: &ruleRefExpr{
												pos:  position{line: 481, col: 34, offset: 12895},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 481, col: 37, offset: 12898},
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
			pos:  position{line: 491, col: 1, offset: 13093},
			expr: &actionExpr{
				pos: position{line: 492, col: 5, offset: 13105},
				run: (*parser).calloncountOp1,
				expr: &litMatcher{
					pos:        position{line: 492, col: 5, offset: 13105},
					val:        "count",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldReducerOp",
			pos:  position{line: 494, col: 1, offset: 13139},
			expr: &choiceExpr{
				pos: position{line: 495, col: 5, offset: 13158},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 495, col: 5, offset: 13158},
						run: (*parser).callonfieldReducerOp2,
						expr: &litMatcher{
							pos:        position{line: 495, col: 5, offset: 13158},
							val:        "sum",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 496, col: 5, offset: 13192},
						run: (*parser).callonfieldReducerOp4,
						expr: &litMatcher{
							pos:        position{line: 496, col: 5, offset: 13192},
							val:        "avg",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 497, col: 5, offset: 13226},
						run: (*parser).callonfieldReducerOp6,
						expr: &litMatcher{
							pos:        position{line: 497, col: 5, offset: 13226},
							val:        "stdev",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 498, col: 5, offset: 13263},
						run: (*parser).callonfieldReducerOp8,
						expr: &litMatcher{
							pos:        position{line: 498, col: 5, offset: 13263},
							val:        "sd",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 499, col: 5, offset: 13299},
						run: (*parser).callonfieldReducerOp10,
						expr: &litMatcher{
							pos:        position{line: 499, col: 5, offset: 13299},
							val:        "var",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 500, col: 5, offset: 13333},
						run: (*parser).callonfieldReducerOp12,
						expr: &litMatcher{
							pos:        position{line: 500, col: 5, offset: 13333},
							val:        "entropy",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 501, col: 5, offset: 13374},
						run: (*parser).callonfieldReducerOp14,
						expr: &litMatcher{
							pos:        position{line: 501, col: 5, offset: 13374},
							val:        "min",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 502, col: 5, offset: 13408},
						run: (*parser).callonfieldReducerOp16,
						expr: &litMatcher{
							pos:        position{line: 502, col: 5, offset: 13408},
							val:        "max",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 503, col: 5, offset: 13442},
						run: (*parser).callonfieldReducerOp18,
						expr: &litMatcher{
							pos:        position{line: 503, col: 5, offset: 13442},
							val:        "first",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 504, col: 5, offset: 13480},
						run: (*parser).callonfieldReducerOp20,
						expr: &litMatcher{
							pos:        position{line: 504, col: 5, offset: 13480},
							val:        "last",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "paddedFieldName",
			pos:  position{line: 506, col: 1, offset: 13513},
			expr: &actionExpr{
				pos: position{line: 506, col: 19, offset: 13531},
				run: (*parser).callonpaddedFieldName1,
				expr: &seqExpr{
					pos: position{line: 506, col: 19, offset: 13531},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 506, col: 19, offset: 13531},
							expr: &ruleRefExpr{
								pos:  position{line: 506, col: 19, offset: 13531},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 506, col: 22, offset: 13534},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 506, col: 28, offset: 13540},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 506, col: 38, offset: 13550},
							expr: &ruleRefExpr{
								pos:  position{line: 506, col: 38, offset: 13550},
								name: "_",
							},
						},
					},
				},
			},
		},
		{
			name: "countReducer",
			pos:  position{line: 508, col: 1, offset: 13576},
			expr: &actionExpr{
				pos: position{line: 509, col: 5, offset: 13593},
				run: (*parser).calloncountReducer1,
				expr: &seqExpr{
					pos: position{line: 509, col: 5, offset: 13593},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 509, col: 5, offset: 13593},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 509, col: 8, offset: 13596},
								name: "countOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 509, col: 16, offset: 13604},
							expr: &ruleRefExpr{
								pos:  position{line: 509, col: 16, offset: 13604},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 509, col: 19, offset: 13607},
							val:        "(",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 509, col: 23, offset: 13611},
							label: "field",
							expr: &zeroOrOneExpr{
								pos: position{line: 509, col: 29, offset: 13617},
								expr: &ruleRefExpr{
									pos:  position{line: 509, col: 29, offset: 13617},
									name: "paddedFieldName",
								},
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 509, col: 47, offset: 13635},
							expr: &ruleRefExpr{
								pos:  position{line: 509, col: 47, offset: 13635},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 509, col: 50, offset: 13638},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "fieldReducer",
			pos:  position{line: 513, col: 1, offset: 13697},
			expr: &actionExpr{
				pos: position{line: 514, col: 5, offset: 13714},
				run: (*parser).callonfieldReducer1,
				expr: &seqExpr{
					pos: position{line: 514, col: 5, offset: 13714},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 514, col: 5, offset: 13714},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 514, col: 8, offset: 13717},
								name: "fieldReducerOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 514, col: 23, offset: 13732},
							expr: &ruleRefExpr{
								pos:  position{line: 514, col: 23, offset: 13732},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 514, col: 26, offset: 13735},
							val:        "(",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 514, col: 30, offset: 13739},
							expr: &ruleRefExpr{
								pos:  position{line: 514, col: 30, offset: 13739},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 514, col: 33, offset: 13742},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 514, col: 39, offset: 13748},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 514, col: 50, offset: 13759},
							expr: &ruleRefExpr{
								pos:  position{line: 514, col: 50, offset: 13759},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 514, col: 53, offset: 13762},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reducerProc",
			pos:  position{line: 518, col: 1, offset: 13829},
			expr: &actionExpr{
				pos: position{line: 519, col: 5, offset: 13845},
				run: (*parser).callonreducerProc1,
				expr: &seqExpr{
					pos: position{line: 519, col: 5, offset: 13845},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 519, col: 5, offset: 13845},
							label: "every",
							expr: &zeroOrOneExpr{
								pos: position{line: 519, col: 11, offset: 13851},
								expr: &seqExpr{
									pos: position{line: 519, col: 12, offset: 13852},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 519, col: 12, offset: 13852},
											name: "everyDur",
										},
										&ruleRefExpr{
											pos:  position{line: 519, col: 21, offset: 13861},
											name: "_",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 519, col: 25, offset: 13865},
							label: "reducers",
							expr: &ruleRefExpr{
								pos:  position{line: 519, col: 34, offset: 13874},
								name: "reducerList",
							},
						},
						&labeledExpr{
							pos:   position{line: 519, col: 46, offset: 13886},
							label: "keys",
							expr: &zeroOrOneExpr{
								pos: position{line: 519, col: 51, offset: 13891},
								expr: &seqExpr{
									pos: position{line: 519, col: 52, offset: 13892},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 519, col: 52, offset: 13892},
											name: "_",
										},
										&ruleRefExpr{
											pos:  position{line: 519, col: 54, offset: 13894},
											name: "groupBy",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 519, col: 64, offset: 13904},
							label: "limit",
							expr: &zeroOrOneExpr{
								pos: position{line: 519, col: 70, offset: 13910},
								expr: &ruleRefExpr{
									pos:  position{line: 519, col: 70, offset: 13910},
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
			pos:  position{line: 537, col: 1, offset: 14267},
			expr: &actionExpr{
				pos: position{line: 538, col: 5, offset: 14280},
				run: (*parser).callonasClause1,
				expr: &seqExpr{
					pos: position{line: 538, col: 5, offset: 14280},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 538, col: 5, offset: 14280},
							val:        "as",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 538, col: 11, offset: 14286},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 538, col: 13, offset: 14288},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 538, col: 15, offset: 14290},
								name: "fieldName",
							},
						},
					},
				},
			},
		},
		{
			name: "reducerExpr",
			pos:  position{line: 540, col: 1, offset: 14319},
			expr: &choiceExpr{
				pos: position{line: 541, col: 5, offset: 14335},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 541, col: 5, offset: 14335},
						run: (*parser).callonreducerExpr2,
						expr: &seqExpr{
							pos: position{line: 541, col: 5, offset: 14335},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 541, col: 5, offset: 14335},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 541, col: 11, offset: 14341},
										name: "fieldName",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 541, col: 21, offset: 14351},
									expr: &ruleRefExpr{
										pos:  position{line: 541, col: 21, offset: 14351},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 541, col: 24, offset: 14354},
									val:        "=",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 541, col: 28, offset: 14358},
									expr: &ruleRefExpr{
										pos:  position{line: 541, col: 28, offset: 14358},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 541, col: 31, offset: 14361},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 541, col: 33, offset: 14363},
										name: "reducer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 544, col: 5, offset: 14426},
						run: (*parser).callonreducerExpr13,
						expr: &seqExpr{
							pos: position{line: 544, col: 5, offset: 14426},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 544, col: 5, offset: 14426},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 544, col: 7, offset: 14428},
										name: "reducer",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 544, col: 15, offset: 14436},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 544, col: 17, offset: 14438},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 544, col: 23, offset: 14444},
										name: "asClause",
									},
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 547, col: 5, offset: 14508},
						name: "reducer",
					},
				},
			},
		},
		{
			name: "reducer",
			pos:  position{line: 549, col: 1, offset: 14517},
			expr: &choiceExpr{
				pos: position{line: 550, col: 5, offset: 14529},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 550, col: 5, offset: 14529},
						name: "countReducer",
					},
					&ruleRefExpr{
						pos:  position{line: 551, col: 5, offset: 14546},
						name: "fieldReducer",
					},
				},
			},
		},
		{
			name: "reducerList",
			pos:  position{line: 553, col: 1, offset: 14560},
			expr: &actionExpr{
				pos: position{line: 554, col: 5, offset: 14576},
				run: (*parser).callonreducerList1,
				expr: &seqExpr{
					pos: position{line: 554, col: 5, offset: 14576},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 554, col: 5, offset: 14576},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 554, col: 11, offset: 14582},
								name: "reducerExpr",
							},
						},
						&labeledExpr{
							pos:   position{line: 554, col: 23, offset: 14594},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 554, col: 28, offset: 14599},
								expr: &seqExpr{
									pos: position{line: 554, col: 29, offset: 14600},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 554, col: 29, offset: 14600},
											expr: &ruleRefExpr{
												pos:  position{line: 554, col: 29, offset: 14600},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 554, col: 32, offset: 14603},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 554, col: 36, offset: 14607},
											expr: &ruleRefExpr{
												pos:  position{line: 554, col: 36, offset: 14607},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 554, col: 39, offset: 14610},
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
			pos:  position{line: 562, col: 1, offset: 14807},
			expr: &choiceExpr{
				pos: position{line: 563, col: 5, offset: 14822},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 563, col: 5, offset: 14822},
						name: "sort",
					},
					&ruleRefExpr{
						pos:  position{line: 564, col: 5, offset: 14831},
						name: "cut",
					},
					&ruleRefExpr{
						pos:  position{line: 565, col: 5, offset: 14839},
						name: "head",
					},
					&ruleRefExpr{
						pos:  position{line: 566, col: 5, offset: 14848},
						name: "tail",
					},
					&ruleRefExpr{
						pos:  position{line: 567, col: 5, offset: 14857},
						name: "filter",
					},
					&ruleRefExpr{
						pos:  position{line: 568, col: 5, offset: 14868},
						name: "uniq",
					},
				},
			},
		},
		{
			name: "sort",
			pos:  position{line: 570, col: 1, offset: 14874},
			expr: &choiceExpr{
				pos: position{line: 571, col: 5, offset: 14883},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 571, col: 5, offset: 14883},
						run: (*parser).callonsort2,
						expr: &seqExpr{
							pos: position{line: 571, col: 5, offset: 14883},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 571, col: 5, offset: 14883},
									val:        "sort",
									ignoreCase: true,
								},
								&labeledExpr{
									pos:   position{line: 571, col: 13, offset: 14891},
									label: "rev",
									expr: &zeroOrOneExpr{
										pos: position{line: 571, col: 17, offset: 14895},
										expr: &seqExpr{
											pos: position{line: 571, col: 18, offset: 14896},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 571, col: 18, offset: 14896},
													name: "_",
												},
												&litMatcher{
													pos:        position{line: 571, col: 20, offset: 14898},
													val:        "-r",
													ignoreCase: false,
												},
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 571, col: 27, offset: 14905},
									label: "limit",
									expr: &zeroOrOneExpr{
										pos: position{line: 571, col: 33, offset: 14911},
										expr: &ruleRefExpr{
											pos:  position{line: 571, col: 33, offset: 14911},
											name: "procLimitArg",
										},
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 571, col: 48, offset: 14926},
									expr: &ruleRefExpr{
										pos:  position{line: 571, col: 48, offset: 14926},
										name: "_",
									},
								},
								&notExpr{
									pos: position{line: 571, col: 51, offset: 14929},
									expr: &litMatcher{
										pos:        position{line: 571, col: 52, offset: 14930},
										val:        "-r",
										ignoreCase: false,
									},
								},
								&labeledExpr{
									pos:   position{line: 571, col: 57, offset: 14935},
									label: "list",
									expr: &zeroOrOneExpr{
										pos: position{line: 571, col: 62, offset: 14940},
										expr: &ruleRefExpr{
											pos:  position{line: 571, col: 63, offset: 14941},
											name: "fieldList",
										},
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 576, col: 5, offset: 15067},
						run: (*parser).callonsort20,
						expr: &seqExpr{
							pos: position{line: 576, col: 5, offset: 15067},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 576, col: 5, offset: 15067},
									val:        "sort",
									ignoreCase: true,
								},
								&labeledExpr{
									pos:   position{line: 576, col: 13, offset: 15075},
									label: "limit",
									expr: &zeroOrOneExpr{
										pos: position{line: 576, col: 19, offset: 15081},
										expr: &ruleRefExpr{
											pos:  position{line: 576, col: 19, offset: 15081},
											name: "procLimitArg",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 576, col: 33, offset: 15095},
									label: "rev",
									expr: &zeroOrOneExpr{
										pos: position{line: 576, col: 37, offset: 15099},
										expr: &seqExpr{
											pos: position{line: 576, col: 38, offset: 15100},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 576, col: 38, offset: 15100},
													name: "_",
												},
												&litMatcher{
													pos:        position{line: 576, col: 40, offset: 15102},
													val:        "-r",
													ignoreCase: false,
												},
											},
										},
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 576, col: 47, offset: 15109},
									expr: &ruleRefExpr{
										pos:  position{line: 576, col: 47, offset: 15109},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 576, col: 50, offset: 15112},
									label: "list",
									expr: &zeroOrOneExpr{
										pos: position{line: 576, col: 55, offset: 15117},
										expr: &ruleRefExpr{
											pos:  position{line: 576, col: 56, offset: 15118},
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
			name: "procLimitArg",
			pos:  position{line: 582, col: 1, offset: 15241},
			expr: &actionExpr{
				pos: position{line: 583, col: 5, offset: 15258},
				run: (*parser).callonprocLimitArg1,
				expr: &seqExpr{
					pos: position{line: 583, col: 5, offset: 15258},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 583, col: 5, offset: 15258},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 583, col: 7, offset: 15260},
							val:        "-limit",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 583, col: 16, offset: 15269},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 583, col: 18, offset: 15271},
							label: "limit",
							expr: &ruleRefExpr{
								pos:  position{line: 583, col: 24, offset: 15277},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "cut",
			pos:  position{line: 585, col: 1, offset: 15308},
			expr: &actionExpr{
				pos: position{line: 586, col: 5, offset: 15316},
				run: (*parser).calloncut1,
				expr: &seqExpr{
					pos: position{line: 586, col: 5, offset: 15316},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 586, col: 5, offset: 15316},
							val:        "cut",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 586, col: 12, offset: 15323},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 586, col: 14, offset: 15325},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 586, col: 19, offset: 15330},
								name: "fieldList",
							},
						},
					},
				},
			},
		},
		{
			name: "head",
			pos:  position{line: 587, col: 1, offset: 15374},
			expr: &choiceExpr{
				pos: position{line: 588, col: 5, offset: 15383},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 588, col: 5, offset: 15383},
						run: (*parser).callonhead2,
						expr: &seqExpr{
							pos: position{line: 588, col: 5, offset: 15383},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 588, col: 5, offset: 15383},
									val:        "head",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 588, col: 13, offset: 15391},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 588, col: 15, offset: 15393},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 588, col: 21, offset: 15399},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 589, col: 5, offset: 15447},
						run: (*parser).callonhead8,
						expr: &litMatcher{
							pos:        position{line: 589, col: 5, offset: 15447},
							val:        "head",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "tail",
			pos:  position{line: 590, col: 1, offset: 15487},
			expr: &choiceExpr{
				pos: position{line: 591, col: 5, offset: 15496},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 591, col: 5, offset: 15496},
						run: (*parser).callontail2,
						expr: &seqExpr{
							pos: position{line: 591, col: 5, offset: 15496},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 591, col: 5, offset: 15496},
									val:        "tail",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 591, col: 13, offset: 15504},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 591, col: 15, offset: 15506},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 591, col: 21, offset: 15512},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 592, col: 5, offset: 15560},
						run: (*parser).callontail8,
						expr: &litMatcher{
							pos:        position{line: 592, col: 5, offset: 15560},
							val:        "tail",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "filter",
			pos:  position{line: 594, col: 1, offset: 15601},
			expr: &actionExpr{
				pos: position{line: 595, col: 5, offset: 15612},
				run: (*parser).callonfilter1,
				expr: &seqExpr{
					pos: position{line: 595, col: 5, offset: 15612},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 595, col: 5, offset: 15612},
							val:        "filter",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 595, col: 15, offset: 15622},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 595, col: 17, offset: 15624},
							label: "expr",
							expr: &ruleRefExpr{
								pos:  position{line: 595, col: 22, offset: 15629},
								name: "searchExpr",
							},
						},
					},
				},
			},
		},
		{
			name: "uniq",
			pos:  position{line: 598, col: 1, offset: 15687},
			expr: &choiceExpr{
				pos: position{line: 599, col: 5, offset: 15696},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 599, col: 5, offset: 15696},
						run: (*parser).callonuniq2,
						expr: &seqExpr{
							pos: position{line: 599, col: 5, offset: 15696},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 599, col: 5, offset: 15696},
									val:        "uniq",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 599, col: 13, offset: 15704},
									name: "_",
								},
								&litMatcher{
									pos:        position{line: 599, col: 15, offset: 15706},
									val:        "-c",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 602, col: 5, offset: 15760},
						run: (*parser).callonuniq7,
						expr: &litMatcher{
							pos:        position{line: 602, col: 5, offset: 15760},
							val:        "uniq",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "duration",
			pos:  position{line: 606, col: 1, offset: 15815},
			expr: &choiceExpr{
				pos: position{line: 607, col: 5, offset: 15828},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 607, col: 5, offset: 15828},
						name: "seconds",
					},
					&ruleRefExpr{
						pos:  position{line: 608, col: 5, offset: 15840},
						name: "minutes",
					},
					&ruleRefExpr{
						pos:  position{line: 609, col: 5, offset: 15852},
						name: "hours",
					},
					&seqExpr{
						pos: position{line: 610, col: 5, offset: 15862},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 610, col: 5, offset: 15862},
								name: "hours",
							},
							&ruleRefExpr{
								pos:  position{line: 610, col: 11, offset: 15868},
								name: "_",
							},
							&litMatcher{
								pos:        position{line: 610, col: 13, offset: 15870},
								val:        "and",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 610, col: 19, offset: 15876},
								name: "_",
							},
							&ruleRefExpr{
								pos:  position{line: 610, col: 21, offset: 15878},
								name: "minutes",
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 611, col: 5, offset: 15890},
						name: "days",
					},
					&ruleRefExpr{
						pos:  position{line: 612, col: 5, offset: 15899},
						name: "weeks",
					},
				},
			},
		},
		{
			name: "sec_abbrev",
			pos:  position{line: 614, col: 1, offset: 15906},
			expr: &choiceExpr{
				pos: position{line: 615, col: 5, offset: 15921},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 615, col: 5, offset: 15921},
						val:        "seconds",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 616, col: 5, offset: 15935},
						val:        "second",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 617, col: 5, offset: 15948},
						val:        "secs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 618, col: 5, offset: 15959},
						val:        "sec",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 619, col: 5, offset: 15969},
						val:        "s",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "min_abbrev",
			pos:  position{line: 621, col: 1, offset: 15974},
			expr: &choiceExpr{
				pos: position{line: 622, col: 5, offset: 15989},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 622, col: 5, offset: 15989},
						val:        "minutes",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 623, col: 5, offset: 16003},
						val:        "minute",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 624, col: 5, offset: 16016},
						val:        "mins",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 625, col: 5, offset: 16027},
						val:        "min",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 626, col: 5, offset: 16037},
						val:        "m",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "hour_abbrev",
			pos:  position{line: 628, col: 1, offset: 16042},
			expr: &choiceExpr{
				pos: position{line: 629, col: 5, offset: 16058},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 629, col: 5, offset: 16058},
						val:        "hours",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 630, col: 5, offset: 16070},
						val:        "hrs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 631, col: 5, offset: 16080},
						val:        "hr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 632, col: 5, offset: 16089},
						val:        "h",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 633, col: 5, offset: 16097},
						val:        "hour",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "day_abbrev",
			pos:  position{line: 635, col: 1, offset: 16105},
			expr: &choiceExpr{
				pos: position{line: 635, col: 14, offset: 16118},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 635, col: 14, offset: 16118},
						val:        "days",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 635, col: 21, offset: 16125},
						val:        "day",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 635, col: 27, offset: 16131},
						val:        "d",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "week_abbrev",
			pos:  position{line: 636, col: 1, offset: 16135},
			expr: &choiceExpr{
				pos: position{line: 636, col: 15, offset: 16149},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 636, col: 15, offset: 16149},
						val:        "weeks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 636, col: 23, offset: 16157},
						val:        "week",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 636, col: 30, offset: 16164},
						val:        "wks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 636, col: 36, offset: 16170},
						val:        "wk",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 636, col: 41, offset: 16175},
						val:        "w",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "seconds",
			pos:  position{line: 638, col: 1, offset: 16180},
			expr: &choiceExpr{
				pos: position{line: 639, col: 5, offset: 16192},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 639, col: 5, offset: 16192},
						run: (*parser).callonseconds2,
						expr: &litMatcher{
							pos:        position{line: 639, col: 5, offset: 16192},
							val:        "second",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 640, col: 5, offset: 16237},
						run: (*parser).callonseconds4,
						expr: &seqExpr{
							pos: position{line: 640, col: 5, offset: 16237},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 640, col: 5, offset: 16237},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 640, col: 9, offset: 16241},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 640, col: 16, offset: 16248},
									expr: &ruleRefExpr{
										pos:  position{line: 640, col: 16, offset: 16248},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 640, col: 19, offset: 16251},
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
			pos:  position{line: 642, col: 1, offset: 16297},
			expr: &choiceExpr{
				pos: position{line: 643, col: 5, offset: 16309},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 643, col: 5, offset: 16309},
						run: (*parser).callonminutes2,
						expr: &litMatcher{
							pos:        position{line: 643, col: 5, offset: 16309},
							val:        "minute",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 644, col: 5, offset: 16355},
						run: (*parser).callonminutes4,
						expr: &seqExpr{
							pos: position{line: 644, col: 5, offset: 16355},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 644, col: 5, offset: 16355},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 644, col: 9, offset: 16359},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 644, col: 16, offset: 16366},
									expr: &ruleRefExpr{
										pos:  position{line: 644, col: 16, offset: 16366},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 644, col: 19, offset: 16369},
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
			pos:  position{line: 646, col: 1, offset: 16424},
			expr: &choiceExpr{
				pos: position{line: 647, col: 5, offset: 16434},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 647, col: 5, offset: 16434},
						run: (*parser).callonhours2,
						expr: &litMatcher{
							pos:        position{line: 647, col: 5, offset: 16434},
							val:        "hour",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 648, col: 5, offset: 16480},
						run: (*parser).callonhours4,
						expr: &seqExpr{
							pos: position{line: 648, col: 5, offset: 16480},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 648, col: 5, offset: 16480},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 648, col: 9, offset: 16484},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 648, col: 16, offset: 16491},
									expr: &ruleRefExpr{
										pos:  position{line: 648, col: 16, offset: 16491},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 648, col: 19, offset: 16494},
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
			pos:  position{line: 650, col: 1, offset: 16552},
			expr: &choiceExpr{
				pos: position{line: 651, col: 5, offset: 16561},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 651, col: 5, offset: 16561},
						run: (*parser).callondays2,
						expr: &litMatcher{
							pos:        position{line: 651, col: 5, offset: 16561},
							val:        "day",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 652, col: 5, offset: 16609},
						run: (*parser).callondays4,
						expr: &seqExpr{
							pos: position{line: 652, col: 5, offset: 16609},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 652, col: 5, offset: 16609},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 652, col: 9, offset: 16613},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 652, col: 16, offset: 16620},
									expr: &ruleRefExpr{
										pos:  position{line: 652, col: 16, offset: 16620},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 652, col: 19, offset: 16623},
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
			pos:  position{line: 654, col: 1, offset: 16683},
			expr: &actionExpr{
				pos: position{line: 655, col: 5, offset: 16693},
				run: (*parser).callonweeks1,
				expr: &seqExpr{
					pos: position{line: 655, col: 5, offset: 16693},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 655, col: 5, offset: 16693},
							label: "num",
							expr: &ruleRefExpr{
								pos:  position{line: 655, col: 9, offset: 16697},
								name: "number",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 655, col: 16, offset: 16704},
							expr: &ruleRefExpr{
								pos:  position{line: 655, col: 16, offset: 16704},
								name: "_",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 655, col: 19, offset: 16707},
							name: "week_abbrev",
						},
					},
				},
			},
		},
		{
			name: "number",
			pos:  position{line: 657, col: 1, offset: 16770},
			expr: &ruleRefExpr{
				pos:  position{line: 657, col: 10, offset: 16779},
				name: "integer",
			},
		},
		{
			name: "addr",
			pos:  position{line: 661, col: 1, offset: 16817},
			expr: &actionExpr{
				pos: position{line: 662, col: 5, offset: 16826},
				run: (*parser).callonaddr1,
				expr: &labeledExpr{
					pos:   position{line: 662, col: 5, offset: 16826},
					label: "a",
					expr: &seqExpr{
						pos: position{line: 662, col: 8, offset: 16829},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 662, col: 8, offset: 16829},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 662, col: 16, offset: 16837},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 662, col: 20, offset: 16841},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 662, col: 28, offset: 16849},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 662, col: 32, offset: 16853},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 662, col: 40, offset: 16861},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 662, col: 44, offset: 16865},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "port",
			pos:  position{line: 664, col: 1, offset: 16906},
			expr: &actionExpr{
				pos: position{line: 665, col: 5, offset: 16915},
				run: (*parser).callonport1,
				expr: &seqExpr{
					pos: position{line: 665, col: 5, offset: 16915},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 665, col: 5, offset: 16915},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 665, col: 9, offset: 16919},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 665, col: 11, offset: 16921},
								name: "sinteger",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6addr",
			pos:  position{line: 669, col: 1, offset: 17080},
			expr: &choiceExpr{
				pos: position{line: 670, col: 5, offset: 17092},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 670, col: 5, offset: 17092},
						run: (*parser).callonip6addr2,
						expr: &seqExpr{
							pos: position{line: 670, col: 5, offset: 17092},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 670, col: 5, offset: 17092},
									label: "a",
									expr: &oneOrMoreExpr{
										pos: position{line: 670, col: 7, offset: 17094},
										expr: &ruleRefExpr{
											pos:  position{line: 670, col: 8, offset: 17095},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 670, col: 20, offset: 17107},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 670, col: 22, offset: 17109},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 673, col: 5, offset: 17173},
						run: (*parser).callonip6addr9,
						expr: &seqExpr{
							pos: position{line: 673, col: 5, offset: 17173},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 673, col: 5, offset: 17173},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 673, col: 7, offset: 17175},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 673, col: 11, offset: 17179},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 673, col: 13, offset: 17181},
										expr: &ruleRefExpr{
											pos:  position{line: 673, col: 14, offset: 17182},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 673, col: 25, offset: 17193},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 673, col: 30, offset: 17198},
									label: "d",
									expr: &zeroOrMoreExpr{
										pos: position{line: 673, col: 32, offset: 17200},
										expr: &ruleRefExpr{
											pos:  position{line: 673, col: 33, offset: 17201},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 673, col: 45, offset: 17213},
									label: "e",
									expr: &ruleRefExpr{
										pos:  position{line: 673, col: 47, offset: 17215},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 676, col: 5, offset: 17314},
						run: (*parser).callonip6addr22,
						expr: &seqExpr{
							pos: position{line: 676, col: 5, offset: 17314},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 676, col: 5, offset: 17314},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 676, col: 10, offset: 17319},
									label: "a",
									expr: &zeroOrMoreExpr{
										pos: position{line: 676, col: 12, offset: 17321},
										expr: &ruleRefExpr{
											pos:  position{line: 676, col: 13, offset: 17322},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 676, col: 25, offset: 17334},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 676, col: 27, offset: 17336},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 679, col: 5, offset: 17407},
						run: (*parser).callonip6addr30,
						expr: &seqExpr{
							pos: position{line: 679, col: 5, offset: 17407},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 679, col: 5, offset: 17407},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 679, col: 7, offset: 17409},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 679, col: 11, offset: 17413},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 679, col: 13, offset: 17415},
										expr: &ruleRefExpr{
											pos:  position{line: 679, col: 14, offset: 17416},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 679, col: 25, offset: 17427},
									val:        "::",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 682, col: 5, offset: 17495},
						run: (*parser).callonip6addr38,
						expr: &litMatcher{
							pos:        position{line: 682, col: 5, offset: 17495},
							val:        "::",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "ip6tail",
			pos:  position{line: 686, col: 1, offset: 17532},
			expr: &choiceExpr{
				pos: position{line: 687, col: 5, offset: 17544},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 687, col: 5, offset: 17544},
						name: "addr",
					},
					&ruleRefExpr{
						pos:  position{line: 688, col: 5, offset: 17553},
						name: "h16",
					},
				},
			},
		},
		{
			name: "h_append",
			pos:  position{line: 690, col: 1, offset: 17558},
			expr: &actionExpr{
				pos: position{line: 690, col: 12, offset: 17569},
				run: (*parser).callonh_append1,
				expr: &seqExpr{
					pos: position{line: 690, col: 12, offset: 17569},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 690, col: 12, offset: 17569},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 690, col: 16, offset: 17573},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 690, col: 18, offset: 17575},
								name: "h16",
							},
						},
					},
				},
			},
		},
		{
			name: "h_prepend",
			pos:  position{line: 691, col: 1, offset: 17612},
			expr: &actionExpr{
				pos: position{line: 691, col: 13, offset: 17624},
				run: (*parser).callonh_prepend1,
				expr: &seqExpr{
					pos: position{line: 691, col: 13, offset: 17624},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 691, col: 13, offset: 17624},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 691, col: 15, offset: 17626},
								name: "h16",
							},
						},
						&litMatcher{
							pos:        position{line: 691, col: 19, offset: 17630},
							val:        ":",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "sub_addr",
			pos:  position{line: 693, col: 1, offset: 17668},
			expr: &choiceExpr{
				pos: position{line: 694, col: 5, offset: 17681},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 694, col: 5, offset: 17681},
						name: "addr",
					},
					&actionExpr{
						pos: position{line: 695, col: 5, offset: 17690},
						run: (*parser).callonsub_addr3,
						expr: &labeledExpr{
							pos:   position{line: 695, col: 5, offset: 17690},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 695, col: 8, offset: 17693},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 695, col: 8, offset: 17693},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 695, col: 16, offset: 17701},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 695, col: 20, offset: 17705},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 695, col: 28, offset: 17713},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 695, col: 32, offset: 17717},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 696, col: 5, offset: 17769},
						run: (*parser).callonsub_addr11,
						expr: &labeledExpr{
							pos:   position{line: 696, col: 5, offset: 17769},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 696, col: 8, offset: 17772},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 696, col: 8, offset: 17772},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 696, col: 16, offset: 17780},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 696, col: 20, offset: 17784},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 697, col: 5, offset: 17838},
						run: (*parser).callonsub_addr17,
						expr: &labeledExpr{
							pos:   position{line: 697, col: 5, offset: 17838},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 697, col: 7, offset: 17840},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "subnet",
			pos:  position{line: 699, col: 1, offset: 17891},
			expr: &actionExpr{
				pos: position{line: 700, col: 5, offset: 17902},
				run: (*parser).callonsubnet1,
				expr: &seqExpr{
					pos: position{line: 700, col: 5, offset: 17902},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 700, col: 5, offset: 17902},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 700, col: 7, offset: 17904},
								name: "sub_addr",
							},
						},
						&litMatcher{
							pos:        position{line: 700, col: 16, offset: 17913},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 700, col: 20, offset: 17917},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 700, col: 22, offset: 17919},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6subnet",
			pos:  position{line: 704, col: 1, offset: 17995},
			expr: &actionExpr{
				pos: position{line: 705, col: 5, offset: 18009},
				run: (*parser).callonip6subnet1,
				expr: &seqExpr{
					pos: position{line: 705, col: 5, offset: 18009},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 705, col: 5, offset: 18009},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 705, col: 7, offset: 18011},
								name: "ip6addr",
							},
						},
						&litMatcher{
							pos:        position{line: 705, col: 15, offset: 18019},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 705, col: 19, offset: 18023},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 705, col: 21, offset: 18025},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "integer",
			pos:  position{line: 709, col: 1, offset: 18091},
			expr: &actionExpr{
				pos: position{line: 710, col: 5, offset: 18103},
				run: (*parser).calloninteger1,
				expr: &labeledExpr{
					pos:   position{line: 710, col: 5, offset: 18103},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 710, col: 7, offset: 18105},
						name: "sinteger",
					},
				},
			},
		},
		{
			name: "sinteger",
			pos:  position{line: 714, col: 1, offset: 18149},
			expr: &actionExpr{
				pos: position{line: 715, col: 5, offset: 18162},
				run: (*parser).callonsinteger1,
				expr: &labeledExpr{
					pos:   position{line: 715, col: 5, offset: 18162},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 715, col: 11, offset: 18168},
						expr: &charClassMatcher{
							pos:        position{line: 715, col: 11, offset: 18168},
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
			pos:  position{line: 719, col: 1, offset: 18213},
			expr: &actionExpr{
				pos: position{line: 720, col: 5, offset: 18224},
				run: (*parser).callondouble1,
				expr: &labeledExpr{
					pos:   position{line: 720, col: 5, offset: 18224},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 720, col: 7, offset: 18226},
						name: "sdouble",
					},
				},
			},
		},
		{
			name: "sdouble",
			pos:  position{line: 724, col: 1, offset: 18273},
			expr: &choiceExpr{
				pos: position{line: 725, col: 5, offset: 18285},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 725, col: 5, offset: 18285},
						run: (*parser).callonsdouble2,
						expr: &seqExpr{
							pos: position{line: 725, col: 5, offset: 18285},
							exprs: []interface{}{
								&oneOrMoreExpr{
									pos: position{line: 725, col: 5, offset: 18285},
									expr: &ruleRefExpr{
										pos:  position{line: 725, col: 5, offset: 18285},
										name: "doubleInteger",
									},
								},
								&litMatcher{
									pos:        position{line: 725, col: 20, offset: 18300},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 725, col: 24, offset: 18304},
									expr: &ruleRefExpr{
										pos:  position{line: 725, col: 24, offset: 18304},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 725, col: 37, offset: 18317},
									expr: &ruleRefExpr{
										pos:  position{line: 725, col: 37, offset: 18317},
										name: "exponentPart",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 728, col: 5, offset: 18376},
						run: (*parser).callonsdouble11,
						expr: &seqExpr{
							pos: position{line: 728, col: 5, offset: 18376},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 728, col: 5, offset: 18376},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 728, col: 9, offset: 18380},
									expr: &ruleRefExpr{
										pos:  position{line: 728, col: 9, offset: 18380},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 728, col: 22, offset: 18393},
									expr: &ruleRefExpr{
										pos:  position{line: 728, col: 22, offset: 18393},
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
			pos:  position{line: 732, col: 1, offset: 18449},
			expr: &choiceExpr{
				pos: position{line: 733, col: 5, offset: 18467},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 733, col: 5, offset: 18467},
						val:        "0",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 734, col: 5, offset: 18475},
						exprs: []interface{}{
							&charClassMatcher{
								pos:        position{line: 734, col: 5, offset: 18475},
								val:        "[1-9]",
								ranges:     []rune{'1', '9'},
								ignoreCase: false,
								inverted:   false,
							},
							&zeroOrMoreExpr{
								pos: position{line: 734, col: 11, offset: 18481},
								expr: &charClassMatcher{
									pos:        position{line: 734, col: 11, offset: 18481},
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
			pos:  position{line: 736, col: 1, offset: 18489},
			expr: &charClassMatcher{
				pos:        position{line: 736, col: 15, offset: 18503},
				val:        "[0-9]",
				ranges:     []rune{'0', '9'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "signedInteger",
			pos:  position{line: 738, col: 1, offset: 18510},
			expr: &seqExpr{
				pos: position{line: 738, col: 17, offset: 18526},
				exprs: []interface{}{
					&zeroOrOneExpr{
						pos: position{line: 738, col: 17, offset: 18526},
						expr: &charClassMatcher{
							pos:        position{line: 738, col: 17, offset: 18526},
							val:        "[+-]",
							chars:      []rune{'+', '-'},
							ignoreCase: false,
							inverted:   false,
						},
					},
					&oneOrMoreExpr{
						pos: position{line: 738, col: 23, offset: 18532},
						expr: &ruleRefExpr{
							pos:  position{line: 738, col: 23, offset: 18532},
							name: "doubleDigit",
						},
					},
				},
			},
		},
		{
			name: "exponentPart",
			pos:  position{line: 740, col: 1, offset: 18546},
			expr: &seqExpr{
				pos: position{line: 740, col: 16, offset: 18561},
				exprs: []interface{}{
					&litMatcher{
						pos:        position{line: 740, col: 16, offset: 18561},
						val:        "e",
						ignoreCase: true,
					},
					&ruleRefExpr{
						pos:  position{line: 740, col: 21, offset: 18566},
						name: "signedInteger",
					},
				},
			},
		},
		{
			name: "h16",
			pos:  position{line: 742, col: 1, offset: 18581},
			expr: &actionExpr{
				pos: position{line: 742, col: 7, offset: 18587},
				run: (*parser).callonh161,
				expr: &labeledExpr{
					pos:   position{line: 742, col: 7, offset: 18587},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 742, col: 13, offset: 18593},
						expr: &ruleRefExpr{
							pos:  position{line: 742, col: 13, offset: 18593},
							name: "hexdigit",
						},
					},
				},
			},
		},
		{
			name: "hexdigit",
			pos:  position{line: 744, col: 1, offset: 18635},
			expr: &charClassMatcher{
				pos:        position{line: 744, col: 12, offset: 18646},
				val:        "[0-9a-fA-F]",
				ranges:     []rune{'0', '9', 'a', 'f', 'A', 'F'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name:        "boomWord",
			displayName: "\"boomWord\"",
			pos:         position{line: 746, col: 1, offset: 18659},
			expr: &actionExpr{
				pos: position{line: 746, col: 23, offset: 18681},
				run: (*parser).callonboomWord1,
				expr: &labeledExpr{
					pos:   position{line: 746, col: 23, offset: 18681},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 746, col: 29, offset: 18687},
						expr: &ruleRefExpr{
							pos:  position{line: 746, col: 29, offset: 18687},
							name: "boomWordPart",
						},
					},
				},
			},
		},
		{
			name: "boomWordPart",
			pos:  position{line: 748, col: 1, offset: 18733},
			expr: &seqExpr{
				pos: position{line: 749, col: 5, offset: 18750},
				exprs: []interface{}{
					&notExpr{
						pos: position{line: 749, col: 5, offset: 18750},
						expr: &choiceExpr{
							pos: position{line: 749, col: 7, offset: 18752},
							alternatives: []interface{}{
								&charClassMatcher{
									pos:        position{line: 749, col: 7, offset: 18752},
									val:        "[\\x00-\\x1F\\x5C(),!><=\\x22|\\x27;]",
									chars:      []rune{'\\', '(', ')', ',', '!', '>', '<', '=', '"', '|', '\'', ';'},
									ranges:     []rune{'\x00', '\x1f'},
									ignoreCase: false,
									inverted:   false,
								},
								&ruleRefExpr{
									pos:  position{line: 749, col: 42, offset: 18787},
									name: "ws",
								},
							},
						},
					},
					&anyMatcher{
						line: 749, col: 46, offset: 18791,
					},
				},
			},
		},
		{
			name: "quotedString",
			pos:  position{line: 751, col: 1, offset: 18794},
			expr: &choiceExpr{
				pos: position{line: 752, col: 5, offset: 18811},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 752, col: 5, offset: 18811},
						run: (*parser).callonquotedString2,
						expr: &seqExpr{
							pos: position{line: 752, col: 5, offset: 18811},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 752, col: 5, offset: 18811},
									val:        "\"",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 752, col: 9, offset: 18815},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 752, col: 11, offset: 18817},
										expr: &ruleRefExpr{
											pos:  position{line: 752, col: 11, offset: 18817},
											name: "doubleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 752, col: 29, offset: 18835},
									val:        "\"",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 753, col: 5, offset: 18872},
						run: (*parser).callonquotedString9,
						expr: &seqExpr{
							pos: position{line: 753, col: 5, offset: 18872},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 753, col: 5, offset: 18872},
									val:        "'",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 753, col: 9, offset: 18876},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 753, col: 11, offset: 18878},
										expr: &ruleRefExpr{
											pos:  position{line: 753, col: 11, offset: 18878},
											name: "singleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 753, col: 29, offset: 18896},
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
			pos:  position{line: 755, col: 1, offset: 18930},
			expr: &choiceExpr{
				pos: position{line: 756, col: 5, offset: 18951},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 756, col: 5, offset: 18951},
						run: (*parser).callondoubleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 756, col: 5, offset: 18951},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 756, col: 5, offset: 18951},
									expr: &choiceExpr{
										pos: position{line: 756, col: 7, offset: 18953},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 756, col: 7, offset: 18953},
												val:        "\"",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 756, col: 13, offset: 18959},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 756, col: 26, offset: 18972,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 757, col: 5, offset: 19009},
						run: (*parser).callondoubleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 757, col: 5, offset: 19009},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 757, col: 5, offset: 19009},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 757, col: 10, offset: 19014},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 757, col: 12, offset: 19016},
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
			pos:  position{line: 759, col: 1, offset: 19050},
			expr: &choiceExpr{
				pos: position{line: 760, col: 5, offset: 19071},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 760, col: 5, offset: 19071},
						run: (*parser).callonsingleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 760, col: 5, offset: 19071},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 760, col: 5, offset: 19071},
									expr: &choiceExpr{
										pos: position{line: 760, col: 7, offset: 19073},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 760, col: 7, offset: 19073},
												val:        "'",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 760, col: 13, offset: 19079},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 760, col: 26, offset: 19092,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 761, col: 5, offset: 19129},
						run: (*parser).callonsingleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 761, col: 5, offset: 19129},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 761, col: 5, offset: 19129},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 761, col: 10, offset: 19134},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 761, col: 12, offset: 19136},
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
			pos:  position{line: 763, col: 1, offset: 19170},
			expr: &choiceExpr{
				pos: position{line: 763, col: 18, offset: 19187},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 763, col: 18, offset: 19187},
						name: "singleCharEscape",
					},
					&ruleRefExpr{
						pos:  position{line: 763, col: 37, offset: 19206},
						name: "unicodeEscape",
					},
				},
			},
		},
		{
			name: "singleCharEscape",
			pos:  position{line: 765, col: 1, offset: 19221},
			expr: &choiceExpr{
				pos: position{line: 766, col: 5, offset: 19242},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 766, col: 5, offset: 19242},
						val:        "'",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 767, col: 5, offset: 19250},
						val:        "\"",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 768, col: 5, offset: 19258},
						val:        "\\",
						ignoreCase: false,
					},
					&actionExpr{
						pos: position{line: 769, col: 5, offset: 19267},
						run: (*parser).callonsingleCharEscape5,
						expr: &litMatcher{
							pos:        position{line: 769, col: 5, offset: 19267},
							val:        "b",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 770, col: 5, offset: 19296},
						run: (*parser).callonsingleCharEscape7,
						expr: &litMatcher{
							pos:        position{line: 770, col: 5, offset: 19296},
							val:        "f",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 771, col: 5, offset: 19325},
						run: (*parser).callonsingleCharEscape9,
						expr: &litMatcher{
							pos:        position{line: 771, col: 5, offset: 19325},
							val:        "n",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 772, col: 5, offset: 19354},
						run: (*parser).callonsingleCharEscape11,
						expr: &litMatcher{
							pos:        position{line: 772, col: 5, offset: 19354},
							val:        "r",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 773, col: 5, offset: 19383},
						run: (*parser).callonsingleCharEscape13,
						expr: &litMatcher{
							pos:        position{line: 773, col: 5, offset: 19383},
							val:        "t",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 774, col: 5, offset: 19412},
						run: (*parser).callonsingleCharEscape15,
						expr: &litMatcher{
							pos:        position{line: 774, col: 5, offset: 19412},
							val:        "v",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "unicodeEscape",
			pos:  position{line: 776, col: 1, offset: 19438},
			expr: &seqExpr{
				pos: position{line: 777, col: 5, offset: 19456},
				exprs: []interface{}{
					&litMatcher{
						pos:        position{line: 777, col: 5, offset: 19456},
						val:        "u",
						ignoreCase: false,
					},
					&ruleRefExpr{
						pos:  position{line: 777, col: 9, offset: 19460},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 777, col: 18, offset: 19469},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 777, col: 27, offset: 19478},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 777, col: 36, offset: 19487},
						name: "hexdigit",
					},
				},
			},
		},
		{
			name: "reString",
			pos:  position{line: 779, col: 1, offset: 19497},
			expr: &actionExpr{
				pos: position{line: 780, col: 5, offset: 19510},
				run: (*parser).callonreString1,
				expr: &seqExpr{
					pos: position{line: 780, col: 5, offset: 19510},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 780, col: 5, offset: 19510},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 780, col: 9, offset: 19514},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 780, col: 11, offset: 19516},
								name: "reBody",
							},
						},
						&litMatcher{
							pos:        position{line: 780, col: 18, offset: 19523},
							val:        "/",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reBody",
			pos:  position{line: 782, col: 1, offset: 19546},
			expr: &actionExpr{
				pos: position{line: 783, col: 5, offset: 19557},
				run: (*parser).callonreBody1,
				expr: &oneOrMoreExpr{
					pos: position{line: 783, col: 5, offset: 19557},
					expr: &choiceExpr{
						pos: position{line: 783, col: 6, offset: 19558},
						alternatives: []interface{}{
							&charClassMatcher{
								pos:        position{line: 783, col: 6, offset: 19558},
								val:        "[^/\\\\]",
								chars:      []rune{'/', '\\'},
								ignoreCase: false,
								inverted:   true,
							},
							&litMatcher{
								pos:        position{line: 783, col: 13, offset: 19565},
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
			pos:  position{line: 785, col: 1, offset: 19605},
			expr: &charClassMatcher{
				pos:        position{line: 786, col: 5, offset: 19621},
				val:        "[\\x00-\\x1f\\\\]",
				chars:      []rune{'\\'},
				ranges:     []rune{'\x00', '\x1f'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "ws",
			pos:  position{line: 788, col: 1, offset: 19636},
			expr: &choiceExpr{
				pos: position{line: 789, col: 5, offset: 19643},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 789, col: 5, offset: 19643},
						val:        "\t",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 790, col: 5, offset: 19652},
						val:        "\v",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 791, col: 5, offset: 19661},
						val:        "\f",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 792, col: 5, offset: 19670},
						val:        " ",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 793, col: 5, offset: 19678},
						val:        "\u00a0",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 794, col: 5, offset: 19691},
						val:        "\ufeff",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name:        "_",
			displayName: "\"whitespace\"",
			pos:         position{line: 796, col: 1, offset: 19701},
			expr: &oneOrMoreExpr{
				pos: position{line: 796, col: 18, offset: 19718},
				expr: &ruleRefExpr{
					pos:  position{line: 796, col: 18, offset: 19718},
					name: "ws",
				},
			},
		},
		{
			name: "EOF",
			pos:  position{line: 798, col: 1, offset: 19723},
			expr: &notExpr{
				pos: position{line: 798, col: 7, offset: 19729},
				expr: &anyMatcher{
					line: 798, col: 8, offset: 19730,
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

func (c *current) onfieldRead1(field interface{}) (interface{}, error) {
	return makeFieldRead(field), nil

}

func (p *parser) callonfieldRead1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldRead1(stack["field"])
}

func (c *current) onfieldExpr2(op, field interface{}) (interface{}, error) {
	return makeFieldCall(op, field, nil), nil

}

func (p *parser) callonfieldExpr2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldExpr2(stack["op"], stack["field"])
}

func (c *current) onfieldExpr16(field, index interface{}) (interface{}, error) {
	return makeFieldCall("Index", field, index), nil

}

func (p *parser) callonfieldExpr16() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldExpr16(stack["field"], stack["index"])
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

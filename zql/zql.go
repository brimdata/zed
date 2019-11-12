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

func makeSequentialProc(firstIn, procsIn interface{}) *ast.SequentialProc {
	procs := []ast.Proc{firstIn.(ast.Proc)}
	for _, p := range procsIn.([]interface{}) {
		procs = append(procs, p.(ast.Proc))
	}
	return &ast.SequentialProc{ast.Node{"SequentialProc"}, procs}
}

func makeParallelProc(firstIn, procsIn interface{}) *ast.ParallelProc {
	procs := []ast.Proc{firstIn.(ast.Proc)}
	for _, p := range procsIn.([]interface{}) {
		procs = append(procs, p.(ast.Proc))
	}
	return &ast.ParallelProc{ast.Node{"ParallelProc"}, procs}
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

// Helper to get a properly-typed array of strings from an interface{}
func stringArray(val interface{}) []string {
	if val == nil {
		return nil
	}
	arr := val.([]interface{})
	ret := make([]string, len(arr))
	for i, s := range arr {
		ret[i] = s.(string)
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
			pos:  position{line: 270, col: 1, offset: 7778},
			expr: &actionExpr{
				pos: position{line: 270, col: 9, offset: 7786},
				run: (*parser).callonstart1,
				expr: &seqExpr{
					pos: position{line: 270, col: 9, offset: 7786},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 270, col: 9, offset: 7786},
							expr: &ruleRefExpr{
								pos:  position{line: 270, col: 9, offset: 7786},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 270, col: 12, offset: 7789},
							label: "ast",
							expr: &ruleRefExpr{
								pos:  position{line: 270, col: 16, offset: 7793},
								name: "boomCommand",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 270, col: 28, offset: 7805},
							expr: &ruleRefExpr{
								pos:  position{line: 270, col: 28, offset: 7805},
								name: "_",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 270, col: 31, offset: 7808},
							name: "EOF",
						},
					},
				},
			},
		},
		{
			name: "boomCommand",
			pos:  position{line: 272, col: 1, offset: 7833},
			expr: &choiceExpr{
				pos: position{line: 273, col: 6, offset: 7850},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 273, col: 6, offset: 7850},
						run: (*parser).callonboomCommand2,
						expr: &seqExpr{
							pos: position{line: 273, col: 6, offset: 7850},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 273, col: 6, offset: 7850},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 273, col: 8, offset: 7852},
										name: "search",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 273, col: 15, offset: 7859},
									expr: &ruleRefExpr{
										pos:  position{line: 273, col: 15, offset: 7859},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 273, col: 18, offset: 7862},
									label: "rest",
									expr: &zeroOrMoreExpr{
										pos: position{line: 273, col: 23, offset: 7867},
										expr: &ruleRefExpr{
											pos:  position{line: 273, col: 23, offset: 7867},
											name: "chainedProc",
										},
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 280, col: 6, offset: 8033},
						run: (*parser).callonboomCommand11,
						expr: &labeledExpr{
							pos:   position{line: 280, col: 6, offset: 8033},
							label: "s",
							expr: &ruleRefExpr{
								pos:  position{line: 280, col: 8, offset: 8035},
								name: "search",
							},
						},
					},
				},
			},
		},
		{
			name: "chainedProc",
			pos:  position{line: 284, col: 1, offset: 8071},
			expr: &actionExpr{
				pos: position{line: 284, col: 15, offset: 8085},
				run: (*parser).callonchainedProc1,
				expr: &seqExpr{
					pos: position{line: 284, col: 15, offset: 8085},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 284, col: 15, offset: 8085},
							expr: &ruleRefExpr{
								pos:  position{line: 284, col: 15, offset: 8085},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 284, col: 18, offset: 8088},
							val:        "|",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 284, col: 22, offset: 8092},
							expr: &ruleRefExpr{
								pos:  position{line: 284, col: 22, offset: 8092},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 284, col: 25, offset: 8095},
							label: "p",
							expr: &ruleRefExpr{
								pos:  position{line: 284, col: 27, offset: 8097},
								name: "proc",
							},
						},
					},
				},
			},
		},
		{
			name: "search",
			pos:  position{line: 286, col: 1, offset: 8121},
			expr: &actionExpr{
				pos: position{line: 287, col: 5, offset: 8132},
				run: (*parser).callonsearch1,
				expr: &labeledExpr{
					pos:   position{line: 287, col: 5, offset: 8132},
					label: "expr",
					expr: &ruleRefExpr{
						pos:  position{line: 287, col: 10, offset: 8137},
						name: "searchExpr",
					},
				},
			},
		},
		{
			name: "searchExpr",
			pos:  position{line: 291, col: 1, offset: 8196},
			expr: &actionExpr{
				pos: position{line: 292, col: 5, offset: 8211},
				run: (*parser).callonsearchExpr1,
				expr: &seqExpr{
					pos: position{line: 292, col: 5, offset: 8211},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 292, col: 5, offset: 8211},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 292, col: 11, offset: 8217},
								name: "searchTerm",
							},
						},
						&labeledExpr{
							pos:   position{line: 292, col: 22, offset: 8228},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 292, col: 27, offset: 8233},
								expr: &ruleRefExpr{
									pos:  position{line: 292, col: 27, offset: 8233},
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
			pos:  position{line: 296, col: 1, offset: 8301},
			expr: &actionExpr{
				pos: position{line: 296, col: 18, offset: 8318},
				run: (*parser).callonoredSearchTerm1,
				expr: &seqExpr{
					pos: position{line: 296, col: 18, offset: 8318},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 296, col: 18, offset: 8318},
							name: "_",
						},
						&ruleRefExpr{
							pos:  position{line: 296, col: 20, offset: 8320},
							name: "orToken",
						},
						&ruleRefExpr{
							pos:  position{line: 296, col: 28, offset: 8328},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 296, col: 30, offset: 8330},
							label: "t",
							expr: &ruleRefExpr{
								pos:  position{line: 296, col: 32, offset: 8332},
								name: "searchTerm",
							},
						},
					},
				},
			},
		},
		{
			name: "searchTerm",
			pos:  position{line: 298, col: 1, offset: 8362},
			expr: &actionExpr{
				pos: position{line: 299, col: 5, offset: 8377},
				run: (*parser).callonsearchTerm1,
				expr: &seqExpr{
					pos: position{line: 299, col: 5, offset: 8377},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 299, col: 5, offset: 8377},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 299, col: 11, offset: 8383},
								name: "searchFactor",
							},
						},
						&labeledExpr{
							pos:   position{line: 299, col: 24, offset: 8396},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 299, col: 29, offset: 8401},
								expr: &ruleRefExpr{
									pos:  position{line: 299, col: 29, offset: 8401},
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
			pos:  position{line: 303, col: 1, offset: 8471},
			expr: &actionExpr{
				pos: position{line: 303, col: 19, offset: 8489},
				run: (*parser).callonandedSearchTerm1,
				expr: &seqExpr{
					pos: position{line: 303, col: 19, offset: 8489},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 303, col: 19, offset: 8489},
							name: "_",
						},
						&zeroOrOneExpr{
							pos: position{line: 303, col: 21, offset: 8491},
							expr: &seqExpr{
								pos: position{line: 303, col: 22, offset: 8492},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 303, col: 22, offset: 8492},
										name: "andToken",
									},
									&ruleRefExpr{
										pos:  position{line: 303, col: 31, offset: 8501},
										name: "_",
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 303, col: 35, offset: 8505},
							label: "f",
							expr: &ruleRefExpr{
								pos:  position{line: 303, col: 37, offset: 8507},
								name: "searchFactor",
							},
						},
					},
				},
			},
		},
		{
			name: "searchFactor",
			pos:  position{line: 305, col: 1, offset: 8539},
			expr: &choiceExpr{
				pos: position{line: 306, col: 5, offset: 8556},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 306, col: 5, offset: 8556},
						run: (*parser).callonsearchFactor2,
						expr: &seqExpr{
							pos: position{line: 306, col: 5, offset: 8556},
							exprs: []interface{}{
								&choiceExpr{
									pos: position{line: 306, col: 6, offset: 8557},
									alternatives: []interface{}{
										&seqExpr{
											pos: position{line: 306, col: 6, offset: 8557},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 306, col: 6, offset: 8557},
													name: "notToken",
												},
												&ruleRefExpr{
													pos:  position{line: 306, col: 15, offset: 8566},
													name: "_",
												},
											},
										},
										&seqExpr{
											pos: position{line: 306, col: 19, offset: 8570},
											exprs: []interface{}{
												&litMatcher{
													pos:        position{line: 306, col: 19, offset: 8570},
													val:        "!",
													ignoreCase: false,
												},
												&zeroOrOneExpr{
													pos: position{line: 306, col: 23, offset: 8574},
													expr: &ruleRefExpr{
														pos:  position{line: 306, col: 23, offset: 8574},
														name: "_",
													},
												},
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 306, col: 27, offset: 8578},
									label: "e",
									expr: &ruleRefExpr{
										pos:  position{line: 306, col: 29, offset: 8580},
										name: "searchExpr",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 309, col: 5, offset: 8639},
						run: (*parser).callonsearchFactor14,
						expr: &seqExpr{
							pos: position{line: 309, col: 5, offset: 8639},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 309, col: 5, offset: 8639},
									expr: &litMatcher{
										pos:        position{line: 309, col: 7, offset: 8641},
										val:        "-",
										ignoreCase: false,
									},
								},
								&labeledExpr{
									pos:   position{line: 309, col: 12, offset: 8646},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 309, col: 14, offset: 8648},
										name: "searchPred",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 310, col: 5, offset: 8681},
						run: (*parser).callonsearchFactor20,
						expr: &seqExpr{
							pos: position{line: 310, col: 5, offset: 8681},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 310, col: 5, offset: 8681},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 310, col: 9, offset: 8685},
									expr: &ruleRefExpr{
										pos:  position{line: 310, col: 9, offset: 8685},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 310, col: 12, offset: 8688},
									label: "expr",
									expr: &ruleRefExpr{
										pos:  position{line: 310, col: 17, offset: 8693},
										name: "searchExpr",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 310, col: 28, offset: 8704},
									expr: &ruleRefExpr{
										pos:  position{line: 310, col: 28, offset: 8704},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 310, col: 31, offset: 8707},
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
			pos:  position{line: 312, col: 1, offset: 8733},
			expr: &choiceExpr{
				pos: position{line: 313, col: 5, offset: 8748},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 313, col: 5, offset: 8748},
						run: (*parser).callonsearchPred2,
						expr: &seqExpr{
							pos: position{line: 313, col: 5, offset: 8748},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 313, col: 5, offset: 8748},
									val:        "*",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 313, col: 9, offset: 8752},
									expr: &ruleRefExpr{
										pos:  position{line: 313, col: 9, offset: 8752},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 313, col: 12, offset: 8755},
									label: "fieldComparator",
									expr: &ruleRefExpr{
										pos:  position{line: 313, col: 28, offset: 8771},
										name: "equalityToken",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 313, col: 42, offset: 8785},
									expr: &ruleRefExpr{
										pos:  position{line: 313, col: 42, offset: 8785},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 313, col: 45, offset: 8788},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 313, col: 47, offset: 8790},
										name: "searchValue",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 316, col: 5, offset: 8867},
						run: (*parser).callonsearchPred13,
						expr: &litMatcher{
							pos:        position{line: 316, col: 5, offset: 8867},
							val:        "*",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 319, col: 5, offset: 8926},
						run: (*parser).callonsearchPred15,
						expr: &seqExpr{
							pos: position{line: 319, col: 5, offset: 8926},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 319, col: 5, offset: 8926},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 319, col: 7, offset: 8928},
										name: "fieldExpr",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 319, col: 17, offset: 8938},
									expr: &ruleRefExpr{
										pos:  position{line: 319, col: 17, offset: 8938},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 319, col: 20, offset: 8941},
									label: "fieldComparator",
									expr: &ruleRefExpr{
										pos:  position{line: 319, col: 36, offset: 8957},
										name: "equalityToken",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 319, col: 50, offset: 8971},
									expr: &ruleRefExpr{
										pos:  position{line: 319, col: 50, offset: 8971},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 319, col: 53, offset: 8974},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 319, col: 55, offset: 8976},
										name: "searchValue",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 322, col: 5, offset: 9058},
						run: (*parser).callonsearchPred27,
						expr: &seqExpr{
							pos: position{line: 322, col: 5, offset: 9058},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 322, col: 5, offset: 9058},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 322, col: 7, offset: 9060},
										name: "searchValue",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 322, col: 19, offset: 9072},
									expr: &ruleRefExpr{
										pos:  position{line: 322, col: 19, offset: 9072},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 322, col: 22, offset: 9075},
									name: "inToken",
								},
								&zeroOrOneExpr{
									pos: position{line: 322, col: 30, offset: 9083},
									expr: &ruleRefExpr{
										pos:  position{line: 322, col: 30, offset: 9083},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 322, col: 33, offset: 9086},
									val:        "*",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 325, col: 5, offset: 9144},
						run: (*parser).callonsearchPred37,
						expr: &seqExpr{
							pos: position{line: 325, col: 5, offset: 9144},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 325, col: 5, offset: 9144},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 325, col: 7, offset: 9146},
										name: "searchValue",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 325, col: 19, offset: 9158},
									expr: &ruleRefExpr{
										pos:  position{line: 325, col: 19, offset: 9158},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 325, col: 22, offset: 9161},
									name: "inToken",
								},
								&zeroOrOneExpr{
									pos: position{line: 325, col: 30, offset: 9169},
									expr: &ruleRefExpr{
										pos:  position{line: 325, col: 30, offset: 9169},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 325, col: 33, offset: 9172},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 325, col: 35, offset: 9174},
										name: "fieldRead",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 328, col: 5, offset: 9243},
						run: (*parser).callonsearchPred48,
						expr: &labeledExpr{
							pos:   position{line: 328, col: 5, offset: 9243},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 328, col: 7, offset: 9245},
								name: "searchValue",
							},
						},
					},
				},
			},
		},
		{
			name: "searchValue",
			pos:  position{line: 337, col: 1, offset: 9539},
			expr: &choiceExpr{
				pos: position{line: 338, col: 5, offset: 9555},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 338, col: 5, offset: 9555},
						run: (*parser).callonsearchValue2,
						expr: &labeledExpr{
							pos:   position{line: 338, col: 5, offset: 9555},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 338, col: 7, offset: 9557},
								name: "quotedString",
							},
						},
					},
					&actionExpr{
						pos: position{line: 341, col: 5, offset: 9628},
						run: (*parser).callonsearchValue5,
						expr: &labeledExpr{
							pos:   position{line: 341, col: 5, offset: 9628},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 341, col: 7, offset: 9630},
								name: "reString",
							},
						},
					},
					&actionExpr{
						pos: position{line: 344, col: 5, offset: 9697},
						run: (*parser).callonsearchValue8,
						expr: &labeledExpr{
							pos:   position{line: 344, col: 5, offset: 9697},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 344, col: 7, offset: 9699},
								name: "port",
							},
						},
					},
					&actionExpr{
						pos: position{line: 347, col: 5, offset: 9758},
						run: (*parser).callonsearchValue11,
						expr: &labeledExpr{
							pos:   position{line: 347, col: 5, offset: 9758},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 347, col: 7, offset: 9760},
								name: "ip6subnet",
							},
						},
					},
					&actionExpr{
						pos: position{line: 350, col: 5, offset: 9828},
						run: (*parser).callonsearchValue14,
						expr: &labeledExpr{
							pos:   position{line: 350, col: 5, offset: 9828},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 350, col: 7, offset: 9830},
								name: "ip6addr",
							},
						},
					},
					&actionExpr{
						pos: position{line: 353, col: 5, offset: 9894},
						run: (*parser).callonsearchValue17,
						expr: &labeledExpr{
							pos:   position{line: 353, col: 5, offset: 9894},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 353, col: 7, offset: 9896},
								name: "subnet",
							},
						},
					},
					&actionExpr{
						pos: position{line: 356, col: 5, offset: 9961},
						run: (*parser).callonsearchValue20,
						expr: &labeledExpr{
							pos:   position{line: 356, col: 5, offset: 9961},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 356, col: 7, offset: 9963},
								name: "addr",
							},
						},
					},
					&actionExpr{
						pos: position{line: 359, col: 5, offset: 10024},
						run: (*parser).callonsearchValue23,
						expr: &labeledExpr{
							pos:   position{line: 359, col: 5, offset: 10024},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 359, col: 7, offset: 10026},
								name: "sdouble",
							},
						},
					},
					&actionExpr{
						pos: position{line: 362, col: 5, offset: 10092},
						run: (*parser).callonsearchValue26,
						expr: &seqExpr{
							pos: position{line: 362, col: 5, offset: 10092},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 362, col: 5, offset: 10092},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 362, col: 7, offset: 10094},
										name: "sinteger",
									},
								},
								&notExpr{
									pos: position{line: 362, col: 16, offset: 10103},
									expr: &ruleRefExpr{
										pos:  position{line: 362, col: 17, offset: 10104},
										name: "boomWord",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 365, col: 5, offset: 10168},
						run: (*parser).callonsearchValue32,
						expr: &seqExpr{
							pos: position{line: 365, col: 5, offset: 10168},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 365, col: 5, offset: 10168},
									expr: &seqExpr{
										pos: position{line: 365, col: 7, offset: 10170},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 365, col: 7, offset: 10170},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 365, col: 22, offset: 10185},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 365, col: 25, offset: 10188},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 365, col: 27, offset: 10190},
										name: "booleanLiteral",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 366, col: 5, offset: 10227},
						run: (*parser).callonsearchValue40,
						expr: &seqExpr{
							pos: position{line: 366, col: 5, offset: 10227},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 366, col: 5, offset: 10227},
									expr: &seqExpr{
										pos: position{line: 366, col: 7, offset: 10229},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 366, col: 7, offset: 10229},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 366, col: 22, offset: 10244},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 366, col: 25, offset: 10247},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 366, col: 27, offset: 10249},
										name: "unsetLiteral",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 367, col: 5, offset: 10284},
						run: (*parser).callonsearchValue48,
						expr: &seqExpr{
							pos: position{line: 367, col: 5, offset: 10284},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 367, col: 5, offset: 10284},
									expr: &seqExpr{
										pos: position{line: 367, col: 7, offset: 10286},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 367, col: 7, offset: 10286},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 367, col: 22, offset: 10301},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 367, col: 25, offset: 10304},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 367, col: 27, offset: 10306},
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
			pos:  position{line: 375, col: 1, offset: 10530},
			expr: &choiceExpr{
				pos: position{line: 376, col: 5, offset: 10549},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 376, col: 5, offset: 10549},
						name: "andToken",
					},
					&ruleRefExpr{
						pos:  position{line: 377, col: 5, offset: 10562},
						name: "orToken",
					},
					&ruleRefExpr{
						pos:  position{line: 378, col: 5, offset: 10574},
						name: "inToken",
					},
				},
			},
		},
		{
			name: "booleanLiteral",
			pos:  position{line: 380, col: 1, offset: 10583},
			expr: &choiceExpr{
				pos: position{line: 381, col: 5, offset: 10602},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 381, col: 5, offset: 10602},
						run: (*parser).callonbooleanLiteral2,
						expr: &litMatcher{
							pos:        position{line: 381, col: 5, offset: 10602},
							val:        "true",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 382, col: 5, offset: 10670},
						run: (*parser).callonbooleanLiteral4,
						expr: &litMatcher{
							pos:        position{line: 382, col: 5, offset: 10670},
							val:        "false",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "unsetLiteral",
			pos:  position{line: 384, col: 1, offset: 10736},
			expr: &actionExpr{
				pos: position{line: 385, col: 5, offset: 10753},
				run: (*parser).callonunsetLiteral1,
				expr: &litMatcher{
					pos:        position{line: 385, col: 5, offset: 10753},
					val:        "nil",
					ignoreCase: false,
				},
			},
		},
		{
			name: "procList",
			pos:  position{line: 387, col: 1, offset: 10814},
			expr: &actionExpr{
				pos: position{line: 388, col: 5, offset: 10827},
				run: (*parser).callonprocList1,
				expr: &seqExpr{
					pos: position{line: 388, col: 5, offset: 10827},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 388, col: 5, offset: 10827},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 388, col: 11, offset: 10833},
								name: "procChain",
							},
						},
						&labeledExpr{
							pos:   position{line: 388, col: 21, offset: 10843},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 388, col: 26, offset: 10848},
								expr: &ruleRefExpr{
									pos:  position{line: 388, col: 26, offset: 10848},
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
			pos:  position{line: 392, col: 1, offset: 10920},
			expr: &actionExpr{
				pos: position{line: 392, col: 17, offset: 10936},
				run: (*parser).callonparallelChain1,
				expr: &seqExpr{
					pos: position{line: 392, col: 17, offset: 10936},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 392, col: 17, offset: 10936},
							expr: &ruleRefExpr{
								pos:  position{line: 392, col: 17, offset: 10936},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 392, col: 20, offset: 10939},
							val:        ";",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 392, col: 24, offset: 10943},
							expr: &ruleRefExpr{
								pos:  position{line: 392, col: 24, offset: 10943},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 392, col: 27, offset: 10946},
							label: "ch",
							expr: &ruleRefExpr{
								pos:  position{line: 392, col: 30, offset: 10949},
								name: "procChain",
							},
						},
					},
				},
			},
		},
		{
			name: "procChain",
			pos:  position{line: 394, col: 1, offset: 10979},
			expr: &actionExpr{
				pos: position{line: 395, col: 5, offset: 10993},
				run: (*parser).callonprocChain1,
				expr: &seqExpr{
					pos: position{line: 395, col: 5, offset: 10993},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 395, col: 5, offset: 10993},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 395, col: 11, offset: 10999},
								name: "proc",
							},
						},
						&labeledExpr{
							pos:   position{line: 395, col: 16, offset: 11004},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 395, col: 21, offset: 11009},
								expr: &ruleRefExpr{
									pos:  position{line: 395, col: 21, offset: 11009},
									name: "chainedProc",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "proc",
			pos:  position{line: 403, col: 1, offset: 11178},
			expr: &choiceExpr{
				pos: position{line: 404, col: 5, offset: 11187},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 404, col: 5, offset: 11187},
						name: "simpleProc",
					},
					&ruleRefExpr{
						pos:  position{line: 405, col: 5, offset: 11202},
						name: "reducerProc",
					},
					&actionExpr{
						pos: position{line: 406, col: 5, offset: 11218},
						run: (*parser).callonproc4,
						expr: &seqExpr{
							pos: position{line: 406, col: 5, offset: 11218},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 406, col: 5, offset: 11218},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 406, col: 9, offset: 11222},
									expr: &ruleRefExpr{
										pos:  position{line: 406, col: 9, offset: 11222},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 406, col: 12, offset: 11225},
									label: "proc",
									expr: &ruleRefExpr{
										pos:  position{line: 406, col: 17, offset: 11230},
										name: "procList",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 406, col: 26, offset: 11239},
									expr: &ruleRefExpr{
										pos:  position{line: 406, col: 26, offset: 11239},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 406, col: 29, offset: 11242},
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
			pos:  position{line: 410, col: 1, offset: 11278},
			expr: &actionExpr{
				pos: position{line: 411, col: 5, offset: 11290},
				run: (*parser).callongroupBy1,
				expr: &seqExpr{
					pos: position{line: 411, col: 5, offset: 11290},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 411, col: 5, offset: 11290},
							val:        "by",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 411, col: 11, offset: 11296},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 411, col: 13, offset: 11298},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 411, col: 18, offset: 11303},
								name: "fieldList",
							},
						},
					},
				},
			},
		},
		{
			name: "everyDur",
			pos:  position{line: 413, col: 1, offset: 11335},
			expr: &actionExpr{
				pos: position{line: 414, col: 5, offset: 11348},
				run: (*parser).calloneveryDur1,
				expr: &seqExpr{
					pos: position{line: 414, col: 5, offset: 11348},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 414, col: 5, offset: 11348},
							val:        "every",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 414, col: 14, offset: 11357},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 414, col: 16, offset: 11359},
							label: "dur",
							expr: &ruleRefExpr{
								pos:  position{line: 414, col: 20, offset: 11363},
								name: "duration",
							},
						},
					},
				},
			},
		},
		{
			name: "equalityToken",
			pos:  position{line: 416, col: 1, offset: 11393},
			expr: &choiceExpr{
				pos: position{line: 417, col: 5, offset: 11411},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 417, col: 5, offset: 11411},
						run: (*parser).callonequalityToken2,
						expr: &litMatcher{
							pos:        position{line: 417, col: 5, offset: 11411},
							val:        "=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 418, col: 5, offset: 11441},
						run: (*parser).callonequalityToken4,
						expr: &litMatcher{
							pos:        position{line: 418, col: 5, offset: 11441},
							val:        "!=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 419, col: 5, offset: 11473},
						run: (*parser).callonequalityToken6,
						expr: &litMatcher{
							pos:        position{line: 419, col: 5, offset: 11473},
							val:        "<=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 420, col: 5, offset: 11504},
						run: (*parser).callonequalityToken8,
						expr: &litMatcher{
							pos:        position{line: 420, col: 5, offset: 11504},
							val:        ">=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 421, col: 5, offset: 11535},
						run: (*parser).callonequalityToken10,
						expr: &litMatcher{
							pos:        position{line: 421, col: 5, offset: 11535},
							val:        "<",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 422, col: 5, offset: 11564},
						run: (*parser).callonequalityToken12,
						expr: &litMatcher{
							pos:        position{line: 422, col: 5, offset: 11564},
							val:        ">",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "types",
			pos:  position{line: 424, col: 1, offset: 11590},
			expr: &choiceExpr{
				pos: position{line: 425, col: 5, offset: 11600},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 425, col: 5, offset: 11600},
						val:        "bool",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 426, col: 5, offset: 11611},
						val:        "int",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 427, col: 5, offset: 11621},
						val:        "count",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 428, col: 5, offset: 11633},
						val:        "double",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 429, col: 5, offset: 11646},
						val:        "string",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 430, col: 5, offset: 11659},
						val:        "addr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 431, col: 5, offset: 11670},
						val:        "subnet",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 432, col: 5, offset: 11683},
						val:        "port",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "dash",
			pos:  position{line: 434, col: 1, offset: 11691},
			expr: &choiceExpr{
				pos: position{line: 434, col: 8, offset: 11698},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 434, col: 8, offset: 11698},
						val:        "-",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 434, col: 14, offset: 11704},
						val:        "—",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 434, col: 25, offset: 11715},
						val:        "–",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 434, col: 36, offset: 11726},
						exprs: []interface{}{
							&litMatcher{
								pos:        position{line: 434, col: 36, offset: 11726},
								val:        "to",
								ignoreCase: false,
							},
							&andExpr{
								pos: position{line: 434, col: 40, offset: 11730},
								expr: &ruleRefExpr{
									pos:  position{line: 434, col: 42, offset: 11732},
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
			pos:  position{line: 436, col: 1, offset: 11736},
			expr: &litMatcher{
				pos:        position{line: 436, col: 12, offset: 11747},
				val:        "and",
				ignoreCase: false,
			},
		},
		{
			name: "orToken",
			pos:  position{line: 437, col: 1, offset: 11753},
			expr: &litMatcher{
				pos:        position{line: 437, col: 11, offset: 11763},
				val:        "or",
				ignoreCase: false,
			},
		},
		{
			name: "inToken",
			pos:  position{line: 438, col: 1, offset: 11768},
			expr: &litMatcher{
				pos:        position{line: 438, col: 11, offset: 11778},
				val:        "in",
				ignoreCase: false,
			},
		},
		{
			name: "notToken",
			pos:  position{line: 439, col: 1, offset: 11783},
			expr: &litMatcher{
				pos:        position{line: 439, col: 12, offset: 11794},
				val:        "not",
				ignoreCase: false,
			},
		},
		{
			name: "fieldName",
			pos:  position{line: 441, col: 1, offset: 11801},
			expr: &actionExpr{
				pos: position{line: 441, col: 13, offset: 11813},
				run: (*parser).callonfieldName1,
				expr: &seqExpr{
					pos: position{line: 441, col: 13, offset: 11813},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 441, col: 13, offset: 11813},
							name: "fieldNameStart",
						},
						&zeroOrMoreExpr{
							pos: position{line: 441, col: 28, offset: 11828},
							expr: &ruleRefExpr{
								pos:  position{line: 441, col: 28, offset: 11828},
								name: "fieldNameRest",
							},
						},
					},
				},
			},
		},
		{
			name: "fieldNameStart",
			pos:  position{line: 443, col: 1, offset: 11875},
			expr: &charClassMatcher{
				pos:        position{line: 443, col: 18, offset: 11892},
				val:        "[A-Za-z_$]",
				chars:      []rune{'_', '$'},
				ranges:     []rune{'A', 'Z', 'a', 'z'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "fieldNameRest",
			pos:  position{line: 444, col: 1, offset: 11903},
			expr: &choiceExpr{
				pos: position{line: 444, col: 17, offset: 11919},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 444, col: 17, offset: 11919},
						name: "fieldNameStart",
					},
					&charClassMatcher{
						pos:        position{line: 444, col: 34, offset: 11936},
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
			pos:  position{line: 446, col: 1, offset: 11944},
			expr: &actionExpr{
				pos: position{line: 447, col: 5, offset: 11958},
				run: (*parser).callonfieldRead1,
				expr: &labeledExpr{
					pos:   position{line: 447, col: 5, offset: 11958},
					label: "field",
					expr: &ruleRefExpr{
						pos:  position{line: 447, col: 11, offset: 11964},
						name: "fieldName",
					},
				},
			},
		},
		{
			name: "fieldExpr",
			pos:  position{line: 451, col: 1, offset: 12021},
			expr: &choiceExpr{
				pos: position{line: 452, col: 5, offset: 12035},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 452, col: 5, offset: 12035},
						run: (*parser).callonfieldExpr2,
						expr: &seqExpr{
							pos: position{line: 452, col: 5, offset: 12035},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 452, col: 5, offset: 12035},
									label: "op",
									expr: &ruleRefExpr{
										pos:  position{line: 452, col: 8, offset: 12038},
										name: "fieldOp",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 452, col: 16, offset: 12046},
									expr: &ruleRefExpr{
										pos:  position{line: 452, col: 16, offset: 12046},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 452, col: 19, offset: 12049},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 452, col: 23, offset: 12053},
									expr: &ruleRefExpr{
										pos:  position{line: 452, col: 23, offset: 12053},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 452, col: 26, offset: 12056},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 452, col: 32, offset: 12062},
										name: "fieldName",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 452, col: 42, offset: 12072},
									expr: &ruleRefExpr{
										pos:  position{line: 452, col: 42, offset: 12072},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 452, col: 45, offset: 12075},
									val:        ")",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 455, col: 5, offset: 12139},
						run: (*parser).callonfieldExpr16,
						expr: &seqExpr{
							pos: position{line: 455, col: 5, offset: 12139},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 455, col: 5, offset: 12139},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 455, col: 11, offset: 12145},
										name: "fieldName",
									},
								},
								&litMatcher{
									pos:        position{line: 455, col: 21, offset: 12155},
									val:        "[",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 455, col: 25, offset: 12159},
									label: "index",
									expr: &ruleRefExpr{
										pos:  position{line: 455, col: 31, offset: 12165},
										name: "sinteger",
									},
								},
								&litMatcher{
									pos:        position{line: 455, col: 40, offset: 12174},
									val:        "]",
									ignoreCase: false,
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 458, col: 5, offset: 12245},
						name: "fieldRead",
					},
				},
			},
		},
		{
			name: "fieldOp",
			pos:  position{line: 460, col: 1, offset: 12256},
			expr: &actionExpr{
				pos: position{line: 461, col: 5, offset: 12268},
				run: (*parser).callonfieldOp1,
				expr: &litMatcher{
					pos:        position{line: 461, col: 5, offset: 12268},
					val:        "len",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldList",
			pos:  position{line: 463, col: 1, offset: 12298},
			expr: &actionExpr{
				pos: position{line: 464, col: 5, offset: 12312},
				run: (*parser).callonfieldList1,
				expr: &seqExpr{
					pos: position{line: 464, col: 5, offset: 12312},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 464, col: 5, offset: 12312},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 464, col: 11, offset: 12318},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 464, col: 21, offset: 12328},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 464, col: 26, offset: 12333},
								expr: &seqExpr{
									pos: position{line: 464, col: 27, offset: 12334},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 464, col: 27, offset: 12334},
											expr: &ruleRefExpr{
												pos:  position{line: 464, col: 27, offset: 12334},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 464, col: 30, offset: 12337},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 464, col: 34, offset: 12341},
											expr: &ruleRefExpr{
												pos:  position{line: 464, col: 34, offset: 12341},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 464, col: 37, offset: 12344},
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
			pos:  position{line: 474, col: 1, offset: 12539},
			expr: &actionExpr{
				pos: position{line: 475, col: 5, offset: 12551},
				run: (*parser).calloncountOp1,
				expr: &litMatcher{
					pos:        position{line: 475, col: 5, offset: 12551},
					val:        "count",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldReducerOp",
			pos:  position{line: 477, col: 1, offset: 12585},
			expr: &choiceExpr{
				pos: position{line: 478, col: 5, offset: 12604},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 478, col: 5, offset: 12604},
						run: (*parser).callonfieldReducerOp2,
						expr: &litMatcher{
							pos:        position{line: 478, col: 5, offset: 12604},
							val:        "sum",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 479, col: 5, offset: 12638},
						run: (*parser).callonfieldReducerOp4,
						expr: &litMatcher{
							pos:        position{line: 479, col: 5, offset: 12638},
							val:        "avg",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 480, col: 5, offset: 12672},
						run: (*parser).callonfieldReducerOp6,
						expr: &litMatcher{
							pos:        position{line: 480, col: 5, offset: 12672},
							val:        "stdev",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 481, col: 5, offset: 12709},
						run: (*parser).callonfieldReducerOp8,
						expr: &litMatcher{
							pos:        position{line: 481, col: 5, offset: 12709},
							val:        "sd",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 482, col: 5, offset: 12745},
						run: (*parser).callonfieldReducerOp10,
						expr: &litMatcher{
							pos:        position{line: 482, col: 5, offset: 12745},
							val:        "var",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 483, col: 5, offset: 12779},
						run: (*parser).callonfieldReducerOp12,
						expr: &litMatcher{
							pos:        position{line: 483, col: 5, offset: 12779},
							val:        "entropy",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 484, col: 5, offset: 12820},
						run: (*parser).callonfieldReducerOp14,
						expr: &litMatcher{
							pos:        position{line: 484, col: 5, offset: 12820},
							val:        "min",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 485, col: 5, offset: 12854},
						run: (*parser).callonfieldReducerOp16,
						expr: &litMatcher{
							pos:        position{line: 485, col: 5, offset: 12854},
							val:        "max",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 486, col: 5, offset: 12888},
						run: (*parser).callonfieldReducerOp18,
						expr: &litMatcher{
							pos:        position{line: 486, col: 5, offset: 12888},
							val:        "first",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 487, col: 5, offset: 12926},
						run: (*parser).callonfieldReducerOp20,
						expr: &litMatcher{
							pos:        position{line: 487, col: 5, offset: 12926},
							val:        "last",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "paddedFieldName",
			pos:  position{line: 489, col: 1, offset: 12959},
			expr: &actionExpr{
				pos: position{line: 489, col: 19, offset: 12977},
				run: (*parser).callonpaddedFieldName1,
				expr: &seqExpr{
					pos: position{line: 489, col: 19, offset: 12977},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 489, col: 19, offset: 12977},
							expr: &ruleRefExpr{
								pos:  position{line: 489, col: 19, offset: 12977},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 489, col: 22, offset: 12980},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 489, col: 28, offset: 12986},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 489, col: 38, offset: 12996},
							expr: &ruleRefExpr{
								pos:  position{line: 489, col: 38, offset: 12996},
								name: "_",
							},
						},
					},
				},
			},
		},
		{
			name: "countReducer",
			pos:  position{line: 491, col: 1, offset: 13022},
			expr: &actionExpr{
				pos: position{line: 492, col: 5, offset: 13039},
				run: (*parser).calloncountReducer1,
				expr: &seqExpr{
					pos: position{line: 492, col: 5, offset: 13039},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 492, col: 5, offset: 13039},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 492, col: 8, offset: 13042},
								name: "countOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 492, col: 16, offset: 13050},
							expr: &ruleRefExpr{
								pos:  position{line: 492, col: 16, offset: 13050},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 492, col: 19, offset: 13053},
							val:        "(",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 492, col: 23, offset: 13057},
							label: "field",
							expr: &zeroOrOneExpr{
								pos: position{line: 492, col: 29, offset: 13063},
								expr: &ruleRefExpr{
									pos:  position{line: 492, col: 29, offset: 13063},
									name: "paddedFieldName",
								},
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 492, col: 47, offset: 13081},
							expr: &ruleRefExpr{
								pos:  position{line: 492, col: 47, offset: 13081},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 492, col: 50, offset: 13084},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "fieldReducer",
			pos:  position{line: 496, col: 1, offset: 13143},
			expr: &actionExpr{
				pos: position{line: 497, col: 5, offset: 13160},
				run: (*parser).callonfieldReducer1,
				expr: &seqExpr{
					pos: position{line: 497, col: 5, offset: 13160},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 497, col: 5, offset: 13160},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 497, col: 8, offset: 13163},
								name: "fieldReducerOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 497, col: 23, offset: 13178},
							expr: &ruleRefExpr{
								pos:  position{line: 497, col: 23, offset: 13178},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 497, col: 26, offset: 13181},
							val:        "(",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 497, col: 30, offset: 13185},
							expr: &ruleRefExpr{
								pos:  position{line: 497, col: 30, offset: 13185},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 497, col: 33, offset: 13188},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 497, col: 39, offset: 13194},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 497, col: 50, offset: 13205},
							expr: &ruleRefExpr{
								pos:  position{line: 497, col: 50, offset: 13205},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 497, col: 53, offset: 13208},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reducerProc",
			pos:  position{line: 501, col: 1, offset: 13275},
			expr: &actionExpr{
				pos: position{line: 502, col: 5, offset: 13291},
				run: (*parser).callonreducerProc1,
				expr: &seqExpr{
					pos: position{line: 502, col: 5, offset: 13291},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 502, col: 5, offset: 13291},
							label: "every",
							expr: &zeroOrOneExpr{
								pos: position{line: 502, col: 11, offset: 13297},
								expr: &seqExpr{
									pos: position{line: 502, col: 12, offset: 13298},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 502, col: 12, offset: 13298},
											name: "everyDur",
										},
										&ruleRefExpr{
											pos:  position{line: 502, col: 21, offset: 13307},
											name: "_",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 502, col: 25, offset: 13311},
							label: "reducers",
							expr: &ruleRefExpr{
								pos:  position{line: 502, col: 34, offset: 13320},
								name: "reducerList",
							},
						},
						&labeledExpr{
							pos:   position{line: 502, col: 46, offset: 13332},
							label: "keys",
							expr: &zeroOrOneExpr{
								pos: position{line: 502, col: 51, offset: 13337},
								expr: &seqExpr{
									pos: position{line: 502, col: 52, offset: 13338},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 502, col: 52, offset: 13338},
											name: "_",
										},
										&ruleRefExpr{
											pos:  position{line: 502, col: 54, offset: 13340},
											name: "groupBy",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 502, col: 64, offset: 13350},
							label: "limit",
							expr: &zeroOrOneExpr{
								pos: position{line: 502, col: 70, offset: 13356},
								expr: &ruleRefExpr{
									pos:  position{line: 502, col: 70, offset: 13356},
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
			pos:  position{line: 520, col: 1, offset: 13713},
			expr: &actionExpr{
				pos: position{line: 521, col: 5, offset: 13726},
				run: (*parser).callonasClause1,
				expr: &seqExpr{
					pos: position{line: 521, col: 5, offset: 13726},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 521, col: 5, offset: 13726},
							val:        "as",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 521, col: 11, offset: 13732},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 521, col: 13, offset: 13734},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 521, col: 15, offset: 13736},
								name: "fieldName",
							},
						},
					},
				},
			},
		},
		{
			name: "reducerExpr",
			pos:  position{line: 523, col: 1, offset: 13765},
			expr: &choiceExpr{
				pos: position{line: 524, col: 5, offset: 13781},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 524, col: 5, offset: 13781},
						run: (*parser).callonreducerExpr2,
						expr: &seqExpr{
							pos: position{line: 524, col: 5, offset: 13781},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 524, col: 5, offset: 13781},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 524, col: 11, offset: 13787},
										name: "fieldName",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 524, col: 21, offset: 13797},
									expr: &ruleRefExpr{
										pos:  position{line: 524, col: 21, offset: 13797},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 524, col: 24, offset: 13800},
									val:        "=",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 524, col: 28, offset: 13804},
									expr: &ruleRefExpr{
										pos:  position{line: 524, col: 28, offset: 13804},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 524, col: 31, offset: 13807},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 524, col: 33, offset: 13809},
										name: "reducer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 527, col: 5, offset: 13872},
						run: (*parser).callonreducerExpr13,
						expr: &seqExpr{
							pos: position{line: 527, col: 5, offset: 13872},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 527, col: 5, offset: 13872},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 527, col: 7, offset: 13874},
										name: "reducer",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 527, col: 15, offset: 13882},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 527, col: 17, offset: 13884},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 527, col: 23, offset: 13890},
										name: "asClause",
									},
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 530, col: 5, offset: 13954},
						name: "reducer",
					},
				},
			},
		},
		{
			name: "reducer",
			pos:  position{line: 532, col: 1, offset: 13963},
			expr: &choiceExpr{
				pos: position{line: 533, col: 5, offset: 13975},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 533, col: 5, offset: 13975},
						name: "countReducer",
					},
					&ruleRefExpr{
						pos:  position{line: 534, col: 5, offset: 13992},
						name: "fieldReducer",
					},
				},
			},
		},
		{
			name: "reducerList",
			pos:  position{line: 536, col: 1, offset: 14006},
			expr: &actionExpr{
				pos: position{line: 537, col: 5, offset: 14022},
				run: (*parser).callonreducerList1,
				expr: &seqExpr{
					pos: position{line: 537, col: 5, offset: 14022},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 537, col: 5, offset: 14022},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 537, col: 11, offset: 14028},
								name: "reducerExpr",
							},
						},
						&labeledExpr{
							pos:   position{line: 537, col: 23, offset: 14040},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 537, col: 28, offset: 14045},
								expr: &seqExpr{
									pos: position{line: 537, col: 29, offset: 14046},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 537, col: 29, offset: 14046},
											expr: &ruleRefExpr{
												pos:  position{line: 537, col: 29, offset: 14046},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 537, col: 32, offset: 14049},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 537, col: 36, offset: 14053},
											expr: &ruleRefExpr{
												pos:  position{line: 537, col: 36, offset: 14053},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 537, col: 39, offset: 14056},
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
			pos:  position{line: 545, col: 1, offset: 14253},
			expr: &choiceExpr{
				pos: position{line: 546, col: 5, offset: 14268},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 546, col: 5, offset: 14268},
						name: "sort",
					},
					&ruleRefExpr{
						pos:  position{line: 547, col: 5, offset: 14277},
						name: "cut",
					},
					&ruleRefExpr{
						pos:  position{line: 548, col: 5, offset: 14285},
						name: "head",
					},
					&ruleRefExpr{
						pos:  position{line: 549, col: 5, offset: 14294},
						name: "tail",
					},
					&ruleRefExpr{
						pos:  position{line: 550, col: 5, offset: 14303},
						name: "filter",
					},
					&ruleRefExpr{
						pos:  position{line: 551, col: 5, offset: 14314},
						name: "uniq",
					},
				},
			},
		},
		{
			name: "sort",
			pos:  position{line: 553, col: 1, offset: 14320},
			expr: &choiceExpr{
				pos: position{line: 554, col: 5, offset: 14329},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 554, col: 5, offset: 14329},
						run: (*parser).callonsort2,
						expr: &seqExpr{
							pos: position{line: 554, col: 5, offset: 14329},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 554, col: 5, offset: 14329},
									val:        "sort",
									ignoreCase: true,
								},
								&labeledExpr{
									pos:   position{line: 554, col: 13, offset: 14337},
									label: "rev",
									expr: &zeroOrOneExpr{
										pos: position{line: 554, col: 17, offset: 14341},
										expr: &seqExpr{
											pos: position{line: 554, col: 18, offset: 14342},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 554, col: 18, offset: 14342},
													name: "_",
												},
												&litMatcher{
													pos:        position{line: 554, col: 20, offset: 14344},
													val:        "-r",
													ignoreCase: false,
												},
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 554, col: 27, offset: 14351},
									label: "limit",
									expr: &zeroOrOneExpr{
										pos: position{line: 554, col: 33, offset: 14357},
										expr: &ruleRefExpr{
											pos:  position{line: 554, col: 33, offset: 14357},
											name: "procLimitArg",
										},
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 554, col: 48, offset: 14372},
									expr: &ruleRefExpr{
										pos:  position{line: 554, col: 48, offset: 14372},
										name: "_",
									},
								},
								&notExpr{
									pos: position{line: 554, col: 51, offset: 14375},
									expr: &litMatcher{
										pos:        position{line: 554, col: 52, offset: 14376},
										val:        "-r",
										ignoreCase: false,
									},
								},
								&labeledExpr{
									pos:   position{line: 554, col: 57, offset: 14381},
									label: "list",
									expr: &zeroOrOneExpr{
										pos: position{line: 554, col: 62, offset: 14386},
										expr: &ruleRefExpr{
											pos:  position{line: 554, col: 63, offset: 14387},
											name: "fieldList",
										},
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 559, col: 5, offset: 14513},
						run: (*parser).callonsort20,
						expr: &seqExpr{
							pos: position{line: 559, col: 5, offset: 14513},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 559, col: 5, offset: 14513},
									val:        "sort",
									ignoreCase: true,
								},
								&labeledExpr{
									pos:   position{line: 559, col: 13, offset: 14521},
									label: "limit",
									expr: &zeroOrOneExpr{
										pos: position{line: 559, col: 19, offset: 14527},
										expr: &ruleRefExpr{
											pos:  position{line: 559, col: 19, offset: 14527},
											name: "procLimitArg",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 559, col: 33, offset: 14541},
									label: "rev",
									expr: &zeroOrOneExpr{
										pos: position{line: 559, col: 37, offset: 14545},
										expr: &seqExpr{
											pos: position{line: 559, col: 38, offset: 14546},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 559, col: 38, offset: 14546},
													name: "_",
												},
												&litMatcher{
													pos:        position{line: 559, col: 40, offset: 14548},
													val:        "-r",
													ignoreCase: false,
												},
											},
										},
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 559, col: 47, offset: 14555},
									expr: &ruleRefExpr{
										pos:  position{line: 559, col: 47, offset: 14555},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 559, col: 50, offset: 14558},
									label: "list",
									expr: &zeroOrOneExpr{
										pos: position{line: 559, col: 55, offset: 14563},
										expr: &ruleRefExpr{
											pos:  position{line: 559, col: 56, offset: 14564},
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
			pos:  position{line: 565, col: 1, offset: 14687},
			expr: &actionExpr{
				pos: position{line: 566, col: 5, offset: 14704},
				run: (*parser).callonprocLimitArg1,
				expr: &seqExpr{
					pos: position{line: 566, col: 5, offset: 14704},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 566, col: 5, offset: 14704},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 566, col: 7, offset: 14706},
							val:        "-limit",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 566, col: 16, offset: 14715},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 566, col: 18, offset: 14717},
							label: "limit",
							expr: &ruleRefExpr{
								pos:  position{line: 566, col: 24, offset: 14723},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "cut",
			pos:  position{line: 568, col: 1, offset: 14754},
			expr: &actionExpr{
				pos: position{line: 569, col: 5, offset: 14762},
				run: (*parser).calloncut1,
				expr: &seqExpr{
					pos: position{line: 569, col: 5, offset: 14762},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 569, col: 5, offset: 14762},
							val:        "cut",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 569, col: 12, offset: 14769},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 569, col: 14, offset: 14771},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 569, col: 19, offset: 14776},
								name: "fieldList",
							},
						},
					},
				},
			},
		},
		{
			name: "head",
			pos:  position{line: 570, col: 1, offset: 14820},
			expr: &choiceExpr{
				pos: position{line: 571, col: 5, offset: 14829},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 571, col: 5, offset: 14829},
						run: (*parser).callonhead2,
						expr: &seqExpr{
							pos: position{line: 571, col: 5, offset: 14829},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 571, col: 5, offset: 14829},
									val:        "head",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 571, col: 13, offset: 14837},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 571, col: 15, offset: 14839},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 571, col: 21, offset: 14845},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 572, col: 5, offset: 14893},
						run: (*parser).callonhead8,
						expr: &litMatcher{
							pos:        position{line: 572, col: 5, offset: 14893},
							val:        "head",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "tail",
			pos:  position{line: 573, col: 1, offset: 14933},
			expr: &choiceExpr{
				pos: position{line: 574, col: 5, offset: 14942},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 574, col: 5, offset: 14942},
						run: (*parser).callontail2,
						expr: &seqExpr{
							pos: position{line: 574, col: 5, offset: 14942},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 574, col: 5, offset: 14942},
									val:        "tail",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 574, col: 13, offset: 14950},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 574, col: 15, offset: 14952},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 574, col: 21, offset: 14958},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 575, col: 5, offset: 15006},
						run: (*parser).callontail8,
						expr: &litMatcher{
							pos:        position{line: 575, col: 5, offset: 15006},
							val:        "tail",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "filter",
			pos:  position{line: 577, col: 1, offset: 15047},
			expr: &actionExpr{
				pos: position{line: 578, col: 5, offset: 15058},
				run: (*parser).callonfilter1,
				expr: &seqExpr{
					pos: position{line: 578, col: 5, offset: 15058},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 578, col: 5, offset: 15058},
							val:        "filter",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 578, col: 15, offset: 15068},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 578, col: 17, offset: 15070},
							label: "expr",
							expr: &ruleRefExpr{
								pos:  position{line: 578, col: 22, offset: 15075},
								name: "searchExpr",
							},
						},
					},
				},
			},
		},
		{
			name: "uniq",
			pos:  position{line: 581, col: 1, offset: 15133},
			expr: &choiceExpr{
				pos: position{line: 582, col: 5, offset: 15142},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 582, col: 5, offset: 15142},
						run: (*parser).callonuniq2,
						expr: &seqExpr{
							pos: position{line: 582, col: 5, offset: 15142},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 582, col: 5, offset: 15142},
									val:        "uniq",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 582, col: 13, offset: 15150},
									name: "_",
								},
								&litMatcher{
									pos:        position{line: 582, col: 15, offset: 15152},
									val:        "-c",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 585, col: 5, offset: 15206},
						run: (*parser).callonuniq7,
						expr: &litMatcher{
							pos:        position{line: 585, col: 5, offset: 15206},
							val:        "uniq",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "duration",
			pos:  position{line: 589, col: 1, offset: 15261},
			expr: &choiceExpr{
				pos: position{line: 590, col: 5, offset: 15274},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 590, col: 5, offset: 15274},
						name: "seconds",
					},
					&ruleRefExpr{
						pos:  position{line: 591, col: 5, offset: 15286},
						name: "minutes",
					},
					&ruleRefExpr{
						pos:  position{line: 592, col: 5, offset: 15298},
						name: "hours",
					},
					&seqExpr{
						pos: position{line: 593, col: 5, offset: 15308},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 593, col: 5, offset: 15308},
								name: "hours",
							},
							&ruleRefExpr{
								pos:  position{line: 593, col: 11, offset: 15314},
								name: "_",
							},
							&litMatcher{
								pos:        position{line: 593, col: 13, offset: 15316},
								val:        "and",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 593, col: 19, offset: 15322},
								name: "_",
							},
							&ruleRefExpr{
								pos:  position{line: 593, col: 21, offset: 15324},
								name: "minutes",
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 594, col: 5, offset: 15336},
						name: "days",
					},
					&ruleRefExpr{
						pos:  position{line: 595, col: 5, offset: 15345},
						name: "weeks",
					},
				},
			},
		},
		{
			name: "sec_abbrev",
			pos:  position{line: 597, col: 1, offset: 15352},
			expr: &choiceExpr{
				pos: position{line: 598, col: 5, offset: 15367},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 598, col: 5, offset: 15367},
						val:        "seconds",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 599, col: 5, offset: 15381},
						val:        "second",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 600, col: 5, offset: 15394},
						val:        "secs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 601, col: 5, offset: 15405},
						val:        "sec",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 602, col: 5, offset: 15415},
						val:        "s",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "min_abbrev",
			pos:  position{line: 604, col: 1, offset: 15420},
			expr: &choiceExpr{
				pos: position{line: 605, col: 5, offset: 15435},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 605, col: 5, offset: 15435},
						val:        "minutes",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 606, col: 5, offset: 15449},
						val:        "minute",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 607, col: 5, offset: 15462},
						val:        "mins",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 608, col: 5, offset: 15473},
						val:        "min",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 609, col: 5, offset: 15483},
						val:        "m",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "hour_abbrev",
			pos:  position{line: 611, col: 1, offset: 15488},
			expr: &choiceExpr{
				pos: position{line: 612, col: 5, offset: 15504},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 612, col: 5, offset: 15504},
						val:        "hours",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 613, col: 5, offset: 15516},
						val:        "hrs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 614, col: 5, offset: 15526},
						val:        "hr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 615, col: 5, offset: 15535},
						val:        "h",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 616, col: 5, offset: 15543},
						val:        "hour",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "day_abbrev",
			pos:  position{line: 618, col: 1, offset: 15551},
			expr: &choiceExpr{
				pos: position{line: 618, col: 14, offset: 15564},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 618, col: 14, offset: 15564},
						val:        "days",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 618, col: 21, offset: 15571},
						val:        "day",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 618, col: 27, offset: 15577},
						val:        "d",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "week_abbrev",
			pos:  position{line: 619, col: 1, offset: 15581},
			expr: &choiceExpr{
				pos: position{line: 619, col: 15, offset: 15595},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 619, col: 15, offset: 15595},
						val:        "weeks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 619, col: 23, offset: 15603},
						val:        "week",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 619, col: 30, offset: 15610},
						val:        "wks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 619, col: 36, offset: 15616},
						val:        "wk",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 619, col: 41, offset: 15621},
						val:        "w",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "seconds",
			pos:  position{line: 621, col: 1, offset: 15626},
			expr: &choiceExpr{
				pos: position{line: 622, col: 5, offset: 15638},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 622, col: 5, offset: 15638},
						run: (*parser).callonseconds2,
						expr: &litMatcher{
							pos:        position{line: 622, col: 5, offset: 15638},
							val:        "second",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 623, col: 5, offset: 15683},
						run: (*parser).callonseconds4,
						expr: &seqExpr{
							pos: position{line: 623, col: 5, offset: 15683},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 623, col: 5, offset: 15683},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 623, col: 9, offset: 15687},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 623, col: 16, offset: 15694},
									expr: &ruleRefExpr{
										pos:  position{line: 623, col: 16, offset: 15694},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 623, col: 19, offset: 15697},
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
			pos:  position{line: 625, col: 1, offset: 15743},
			expr: &choiceExpr{
				pos: position{line: 626, col: 5, offset: 15755},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 626, col: 5, offset: 15755},
						run: (*parser).callonminutes2,
						expr: &litMatcher{
							pos:        position{line: 626, col: 5, offset: 15755},
							val:        "minute",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 627, col: 5, offset: 15801},
						run: (*parser).callonminutes4,
						expr: &seqExpr{
							pos: position{line: 627, col: 5, offset: 15801},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 627, col: 5, offset: 15801},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 627, col: 9, offset: 15805},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 627, col: 16, offset: 15812},
									expr: &ruleRefExpr{
										pos:  position{line: 627, col: 16, offset: 15812},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 627, col: 19, offset: 15815},
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
			pos:  position{line: 629, col: 1, offset: 15870},
			expr: &choiceExpr{
				pos: position{line: 630, col: 5, offset: 15880},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 630, col: 5, offset: 15880},
						run: (*parser).callonhours2,
						expr: &litMatcher{
							pos:        position{line: 630, col: 5, offset: 15880},
							val:        "hour",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 631, col: 5, offset: 15926},
						run: (*parser).callonhours4,
						expr: &seqExpr{
							pos: position{line: 631, col: 5, offset: 15926},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 631, col: 5, offset: 15926},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 631, col: 9, offset: 15930},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 631, col: 16, offset: 15937},
									expr: &ruleRefExpr{
										pos:  position{line: 631, col: 16, offset: 15937},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 631, col: 19, offset: 15940},
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
			pos:  position{line: 633, col: 1, offset: 15998},
			expr: &choiceExpr{
				pos: position{line: 634, col: 5, offset: 16007},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 634, col: 5, offset: 16007},
						run: (*parser).callondays2,
						expr: &litMatcher{
							pos:        position{line: 634, col: 5, offset: 16007},
							val:        "day",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 635, col: 5, offset: 16055},
						run: (*parser).callondays4,
						expr: &seqExpr{
							pos: position{line: 635, col: 5, offset: 16055},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 635, col: 5, offset: 16055},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 635, col: 9, offset: 16059},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 635, col: 16, offset: 16066},
									expr: &ruleRefExpr{
										pos:  position{line: 635, col: 16, offset: 16066},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 635, col: 19, offset: 16069},
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
			pos:  position{line: 637, col: 1, offset: 16129},
			expr: &actionExpr{
				pos: position{line: 638, col: 5, offset: 16139},
				run: (*parser).callonweeks1,
				expr: &seqExpr{
					pos: position{line: 638, col: 5, offset: 16139},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 638, col: 5, offset: 16139},
							label: "num",
							expr: &ruleRefExpr{
								pos:  position{line: 638, col: 9, offset: 16143},
								name: "number",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 638, col: 16, offset: 16150},
							expr: &ruleRefExpr{
								pos:  position{line: 638, col: 16, offset: 16150},
								name: "_",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 638, col: 19, offset: 16153},
							name: "week_abbrev",
						},
					},
				},
			},
		},
		{
			name: "number",
			pos:  position{line: 640, col: 1, offset: 16216},
			expr: &ruleRefExpr{
				pos:  position{line: 640, col: 10, offset: 16225},
				name: "integer",
			},
		},
		{
			name: "addr",
			pos:  position{line: 644, col: 1, offset: 16263},
			expr: &actionExpr{
				pos: position{line: 645, col: 5, offset: 16272},
				run: (*parser).callonaddr1,
				expr: &labeledExpr{
					pos:   position{line: 645, col: 5, offset: 16272},
					label: "a",
					expr: &seqExpr{
						pos: position{line: 645, col: 8, offset: 16275},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 645, col: 8, offset: 16275},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 645, col: 16, offset: 16283},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 645, col: 20, offset: 16287},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 645, col: 28, offset: 16295},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 645, col: 32, offset: 16299},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 645, col: 40, offset: 16307},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 645, col: 44, offset: 16311},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "port",
			pos:  position{line: 647, col: 1, offset: 16352},
			expr: &actionExpr{
				pos: position{line: 648, col: 5, offset: 16361},
				run: (*parser).callonport1,
				expr: &seqExpr{
					pos: position{line: 648, col: 5, offset: 16361},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 648, col: 5, offset: 16361},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 648, col: 9, offset: 16365},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 648, col: 11, offset: 16367},
								name: "sinteger",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6addr",
			pos:  position{line: 652, col: 1, offset: 16526},
			expr: &choiceExpr{
				pos: position{line: 653, col: 5, offset: 16538},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 653, col: 5, offset: 16538},
						run: (*parser).callonip6addr2,
						expr: &seqExpr{
							pos: position{line: 653, col: 5, offset: 16538},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 653, col: 5, offset: 16538},
									label: "a",
									expr: &oneOrMoreExpr{
										pos: position{line: 653, col: 7, offset: 16540},
										expr: &ruleRefExpr{
											pos:  position{line: 653, col: 8, offset: 16541},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 653, col: 20, offset: 16553},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 653, col: 22, offset: 16555},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 656, col: 5, offset: 16619},
						run: (*parser).callonip6addr9,
						expr: &seqExpr{
							pos: position{line: 656, col: 5, offset: 16619},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 656, col: 5, offset: 16619},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 656, col: 7, offset: 16621},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 656, col: 11, offset: 16625},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 656, col: 13, offset: 16627},
										expr: &ruleRefExpr{
											pos:  position{line: 656, col: 14, offset: 16628},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 656, col: 25, offset: 16639},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 656, col: 30, offset: 16644},
									label: "d",
									expr: &zeroOrMoreExpr{
										pos: position{line: 656, col: 32, offset: 16646},
										expr: &ruleRefExpr{
											pos:  position{line: 656, col: 33, offset: 16647},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 656, col: 45, offset: 16659},
									label: "e",
									expr: &ruleRefExpr{
										pos:  position{line: 656, col: 47, offset: 16661},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 659, col: 5, offset: 16760},
						run: (*parser).callonip6addr22,
						expr: &seqExpr{
							pos: position{line: 659, col: 5, offset: 16760},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 659, col: 5, offset: 16760},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 659, col: 10, offset: 16765},
									label: "a",
									expr: &zeroOrMoreExpr{
										pos: position{line: 659, col: 12, offset: 16767},
										expr: &ruleRefExpr{
											pos:  position{line: 659, col: 13, offset: 16768},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 659, col: 25, offset: 16780},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 659, col: 27, offset: 16782},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 662, col: 5, offset: 16853},
						run: (*parser).callonip6addr30,
						expr: &seqExpr{
							pos: position{line: 662, col: 5, offset: 16853},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 662, col: 5, offset: 16853},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 662, col: 7, offset: 16855},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 662, col: 11, offset: 16859},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 662, col: 13, offset: 16861},
										expr: &ruleRefExpr{
											pos:  position{line: 662, col: 14, offset: 16862},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 662, col: 25, offset: 16873},
									val:        "::",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 665, col: 5, offset: 16941},
						run: (*parser).callonip6addr38,
						expr: &litMatcher{
							pos:        position{line: 665, col: 5, offset: 16941},
							val:        "::",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "ip6tail",
			pos:  position{line: 669, col: 1, offset: 16978},
			expr: &choiceExpr{
				pos: position{line: 670, col: 5, offset: 16990},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 670, col: 5, offset: 16990},
						name: "addr",
					},
					&ruleRefExpr{
						pos:  position{line: 671, col: 5, offset: 16999},
						name: "h16",
					},
				},
			},
		},
		{
			name: "h_append",
			pos:  position{line: 673, col: 1, offset: 17004},
			expr: &actionExpr{
				pos: position{line: 673, col: 12, offset: 17015},
				run: (*parser).callonh_append1,
				expr: &seqExpr{
					pos: position{line: 673, col: 12, offset: 17015},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 673, col: 12, offset: 17015},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 673, col: 16, offset: 17019},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 673, col: 18, offset: 17021},
								name: "h16",
							},
						},
					},
				},
			},
		},
		{
			name: "h_prepend",
			pos:  position{line: 674, col: 1, offset: 17058},
			expr: &actionExpr{
				pos: position{line: 674, col: 13, offset: 17070},
				run: (*parser).callonh_prepend1,
				expr: &seqExpr{
					pos: position{line: 674, col: 13, offset: 17070},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 674, col: 13, offset: 17070},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 674, col: 15, offset: 17072},
								name: "h16",
							},
						},
						&litMatcher{
							pos:        position{line: 674, col: 19, offset: 17076},
							val:        ":",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "sub_addr",
			pos:  position{line: 676, col: 1, offset: 17114},
			expr: &choiceExpr{
				pos: position{line: 677, col: 5, offset: 17127},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 677, col: 5, offset: 17127},
						name: "addr",
					},
					&actionExpr{
						pos: position{line: 678, col: 5, offset: 17136},
						run: (*parser).callonsub_addr3,
						expr: &labeledExpr{
							pos:   position{line: 678, col: 5, offset: 17136},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 678, col: 8, offset: 17139},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 678, col: 8, offset: 17139},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 678, col: 16, offset: 17147},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 678, col: 20, offset: 17151},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 678, col: 28, offset: 17159},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 678, col: 32, offset: 17163},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 679, col: 5, offset: 17215},
						run: (*parser).callonsub_addr11,
						expr: &labeledExpr{
							pos:   position{line: 679, col: 5, offset: 17215},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 679, col: 8, offset: 17218},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 679, col: 8, offset: 17218},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 679, col: 16, offset: 17226},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 679, col: 20, offset: 17230},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 680, col: 5, offset: 17284},
						run: (*parser).callonsub_addr17,
						expr: &labeledExpr{
							pos:   position{line: 680, col: 5, offset: 17284},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 680, col: 7, offset: 17286},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "subnet",
			pos:  position{line: 682, col: 1, offset: 17337},
			expr: &actionExpr{
				pos: position{line: 683, col: 5, offset: 17348},
				run: (*parser).callonsubnet1,
				expr: &seqExpr{
					pos: position{line: 683, col: 5, offset: 17348},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 683, col: 5, offset: 17348},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 683, col: 7, offset: 17350},
								name: "sub_addr",
							},
						},
						&litMatcher{
							pos:        position{line: 683, col: 16, offset: 17359},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 683, col: 20, offset: 17363},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 683, col: 22, offset: 17365},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6subnet",
			pos:  position{line: 687, col: 1, offset: 17441},
			expr: &actionExpr{
				pos: position{line: 688, col: 5, offset: 17455},
				run: (*parser).callonip6subnet1,
				expr: &seqExpr{
					pos: position{line: 688, col: 5, offset: 17455},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 688, col: 5, offset: 17455},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 688, col: 7, offset: 17457},
								name: "ip6addr",
							},
						},
						&litMatcher{
							pos:        position{line: 688, col: 15, offset: 17465},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 688, col: 19, offset: 17469},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 688, col: 21, offset: 17471},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "integer",
			pos:  position{line: 692, col: 1, offset: 17537},
			expr: &actionExpr{
				pos: position{line: 693, col: 5, offset: 17549},
				run: (*parser).calloninteger1,
				expr: &labeledExpr{
					pos:   position{line: 693, col: 5, offset: 17549},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 693, col: 7, offset: 17551},
						name: "sinteger",
					},
				},
			},
		},
		{
			name: "sinteger",
			pos:  position{line: 697, col: 1, offset: 17595},
			expr: &actionExpr{
				pos: position{line: 698, col: 5, offset: 17608},
				run: (*parser).callonsinteger1,
				expr: &labeledExpr{
					pos:   position{line: 698, col: 5, offset: 17608},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 698, col: 11, offset: 17614},
						expr: &charClassMatcher{
							pos:        position{line: 698, col: 11, offset: 17614},
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
			pos:  position{line: 702, col: 1, offset: 17659},
			expr: &actionExpr{
				pos: position{line: 703, col: 5, offset: 17670},
				run: (*parser).callondouble1,
				expr: &labeledExpr{
					pos:   position{line: 703, col: 5, offset: 17670},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 703, col: 7, offset: 17672},
						name: "sdouble",
					},
				},
			},
		},
		{
			name: "sdouble",
			pos:  position{line: 707, col: 1, offset: 17719},
			expr: &choiceExpr{
				pos: position{line: 708, col: 5, offset: 17731},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 708, col: 5, offset: 17731},
						run: (*parser).callonsdouble2,
						expr: &seqExpr{
							pos: position{line: 708, col: 5, offset: 17731},
							exprs: []interface{}{
								&oneOrMoreExpr{
									pos: position{line: 708, col: 5, offset: 17731},
									expr: &ruleRefExpr{
										pos:  position{line: 708, col: 5, offset: 17731},
										name: "doubleInteger",
									},
								},
								&litMatcher{
									pos:        position{line: 708, col: 20, offset: 17746},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 708, col: 24, offset: 17750},
									expr: &ruleRefExpr{
										pos:  position{line: 708, col: 24, offset: 17750},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 708, col: 37, offset: 17763},
									expr: &ruleRefExpr{
										pos:  position{line: 708, col: 37, offset: 17763},
										name: "exponentPart",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 711, col: 5, offset: 17822},
						run: (*parser).callonsdouble11,
						expr: &seqExpr{
							pos: position{line: 711, col: 5, offset: 17822},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 711, col: 5, offset: 17822},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 711, col: 9, offset: 17826},
									expr: &ruleRefExpr{
										pos:  position{line: 711, col: 9, offset: 17826},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 711, col: 22, offset: 17839},
									expr: &ruleRefExpr{
										pos:  position{line: 711, col: 22, offset: 17839},
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
			pos:  position{line: 715, col: 1, offset: 17895},
			expr: &choiceExpr{
				pos: position{line: 716, col: 5, offset: 17913},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 716, col: 5, offset: 17913},
						val:        "0",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 717, col: 5, offset: 17921},
						exprs: []interface{}{
							&charClassMatcher{
								pos:        position{line: 717, col: 5, offset: 17921},
								val:        "[1-9]",
								ranges:     []rune{'1', '9'},
								ignoreCase: false,
								inverted:   false,
							},
							&zeroOrMoreExpr{
								pos: position{line: 717, col: 11, offset: 17927},
								expr: &charClassMatcher{
									pos:        position{line: 717, col: 11, offset: 17927},
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
			pos:  position{line: 719, col: 1, offset: 17935},
			expr: &charClassMatcher{
				pos:        position{line: 719, col: 15, offset: 17949},
				val:        "[0-9]",
				ranges:     []rune{'0', '9'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "signedInteger",
			pos:  position{line: 721, col: 1, offset: 17956},
			expr: &seqExpr{
				pos: position{line: 721, col: 17, offset: 17972},
				exprs: []interface{}{
					&zeroOrOneExpr{
						pos: position{line: 721, col: 17, offset: 17972},
						expr: &charClassMatcher{
							pos:        position{line: 721, col: 17, offset: 17972},
							val:        "[+-]",
							chars:      []rune{'+', '-'},
							ignoreCase: false,
							inverted:   false,
						},
					},
					&oneOrMoreExpr{
						pos: position{line: 721, col: 23, offset: 17978},
						expr: &ruleRefExpr{
							pos:  position{line: 721, col: 23, offset: 17978},
							name: "doubleDigit",
						},
					},
				},
			},
		},
		{
			name: "exponentPart",
			pos:  position{line: 723, col: 1, offset: 17992},
			expr: &seqExpr{
				pos: position{line: 723, col: 16, offset: 18007},
				exprs: []interface{}{
					&litMatcher{
						pos:        position{line: 723, col: 16, offset: 18007},
						val:        "e",
						ignoreCase: true,
					},
					&ruleRefExpr{
						pos:  position{line: 723, col: 21, offset: 18012},
						name: "signedInteger",
					},
				},
			},
		},
		{
			name: "h16",
			pos:  position{line: 725, col: 1, offset: 18027},
			expr: &actionExpr{
				pos: position{line: 725, col: 7, offset: 18033},
				run: (*parser).callonh161,
				expr: &labeledExpr{
					pos:   position{line: 725, col: 7, offset: 18033},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 725, col: 13, offset: 18039},
						expr: &ruleRefExpr{
							pos:  position{line: 725, col: 13, offset: 18039},
							name: "hexdigit",
						},
					},
				},
			},
		},
		{
			name: "hexdigit",
			pos:  position{line: 727, col: 1, offset: 18081},
			expr: &charClassMatcher{
				pos:        position{line: 727, col: 12, offset: 18092},
				val:        "[0-9a-fA-F]",
				ranges:     []rune{'0', '9', 'a', 'f', 'A', 'F'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name:        "boomWord",
			displayName: "\"boomWord\"",
			pos:         position{line: 729, col: 1, offset: 18105},
			expr: &actionExpr{
				pos: position{line: 729, col: 23, offset: 18127},
				run: (*parser).callonboomWord1,
				expr: &labeledExpr{
					pos:   position{line: 729, col: 23, offset: 18127},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 729, col: 29, offset: 18133},
						expr: &ruleRefExpr{
							pos:  position{line: 729, col: 29, offset: 18133},
							name: "boomWordPart",
						},
					},
				},
			},
		},
		{
			name: "boomWordPart",
			pos:  position{line: 731, col: 1, offset: 18179},
			expr: &seqExpr{
				pos: position{line: 732, col: 5, offset: 18196},
				exprs: []interface{}{
					&notExpr{
						pos: position{line: 732, col: 5, offset: 18196},
						expr: &choiceExpr{
							pos: position{line: 732, col: 7, offset: 18198},
							alternatives: []interface{}{
								&charClassMatcher{
									pos:        position{line: 732, col: 7, offset: 18198},
									val:        "[\\x00-\\x1F\\x5C(),!><=\\x22|\\x27;]",
									chars:      []rune{'\\', '(', ')', ',', '!', '>', '<', '=', '"', '|', '\'', ';'},
									ranges:     []rune{'\x00', '\x1f'},
									ignoreCase: false,
									inverted:   false,
								},
								&ruleRefExpr{
									pos:  position{line: 732, col: 42, offset: 18233},
									name: "ws",
								},
							},
						},
					},
					&anyMatcher{
						line: 732, col: 46, offset: 18237,
					},
				},
			},
		},
		{
			name: "quotedString",
			pos:  position{line: 734, col: 1, offset: 18240},
			expr: &choiceExpr{
				pos: position{line: 735, col: 5, offset: 18257},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 735, col: 5, offset: 18257},
						run: (*parser).callonquotedString2,
						expr: &seqExpr{
							pos: position{line: 735, col: 5, offset: 18257},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 735, col: 5, offset: 18257},
									val:        "\"",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 735, col: 9, offset: 18261},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 735, col: 11, offset: 18263},
										expr: &ruleRefExpr{
											pos:  position{line: 735, col: 11, offset: 18263},
											name: "doubleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 735, col: 29, offset: 18281},
									val:        "\"",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 736, col: 5, offset: 18318},
						run: (*parser).callonquotedString9,
						expr: &seqExpr{
							pos: position{line: 736, col: 5, offset: 18318},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 736, col: 5, offset: 18318},
									val:        "'",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 736, col: 9, offset: 18322},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 736, col: 11, offset: 18324},
										expr: &ruleRefExpr{
											pos:  position{line: 736, col: 11, offset: 18324},
											name: "singleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 736, col: 29, offset: 18342},
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
			pos:  position{line: 738, col: 1, offset: 18376},
			expr: &choiceExpr{
				pos: position{line: 739, col: 5, offset: 18397},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 739, col: 5, offset: 18397},
						run: (*parser).callondoubleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 739, col: 5, offset: 18397},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 739, col: 5, offset: 18397},
									expr: &choiceExpr{
										pos: position{line: 739, col: 7, offset: 18399},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 739, col: 7, offset: 18399},
												val:        "\"",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 739, col: 13, offset: 18405},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 739, col: 26, offset: 18418,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 740, col: 5, offset: 18455},
						run: (*parser).callondoubleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 740, col: 5, offset: 18455},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 740, col: 5, offset: 18455},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 740, col: 10, offset: 18460},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 740, col: 12, offset: 18462},
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
			pos:  position{line: 742, col: 1, offset: 18496},
			expr: &choiceExpr{
				pos: position{line: 743, col: 5, offset: 18517},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 743, col: 5, offset: 18517},
						run: (*parser).callonsingleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 743, col: 5, offset: 18517},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 743, col: 5, offset: 18517},
									expr: &choiceExpr{
										pos: position{line: 743, col: 7, offset: 18519},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 743, col: 7, offset: 18519},
												val:        "'",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 743, col: 13, offset: 18525},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 743, col: 26, offset: 18538,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 744, col: 5, offset: 18575},
						run: (*parser).callonsingleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 744, col: 5, offset: 18575},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 744, col: 5, offset: 18575},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 744, col: 10, offset: 18580},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 744, col: 12, offset: 18582},
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
			pos:  position{line: 746, col: 1, offset: 18616},
			expr: &choiceExpr{
				pos: position{line: 746, col: 18, offset: 18633},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 746, col: 18, offset: 18633},
						name: "singleCharEscape",
					},
					&ruleRefExpr{
						pos:  position{line: 746, col: 37, offset: 18652},
						name: "unicodeEscape",
					},
				},
			},
		},
		{
			name: "singleCharEscape",
			pos:  position{line: 748, col: 1, offset: 18667},
			expr: &choiceExpr{
				pos: position{line: 749, col: 5, offset: 18688},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 749, col: 5, offset: 18688},
						val:        "'",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 750, col: 5, offset: 18696},
						val:        "\"",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 751, col: 5, offset: 18704},
						val:        "\\",
						ignoreCase: false,
					},
					&actionExpr{
						pos: position{line: 752, col: 5, offset: 18713},
						run: (*parser).callonsingleCharEscape5,
						expr: &litMatcher{
							pos:        position{line: 752, col: 5, offset: 18713},
							val:        "b",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 753, col: 5, offset: 18742},
						run: (*parser).callonsingleCharEscape7,
						expr: &litMatcher{
							pos:        position{line: 753, col: 5, offset: 18742},
							val:        "f",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 754, col: 5, offset: 18771},
						run: (*parser).callonsingleCharEscape9,
						expr: &litMatcher{
							pos:        position{line: 754, col: 5, offset: 18771},
							val:        "n",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 755, col: 5, offset: 18800},
						run: (*parser).callonsingleCharEscape11,
						expr: &litMatcher{
							pos:        position{line: 755, col: 5, offset: 18800},
							val:        "r",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 756, col: 5, offset: 18829},
						run: (*parser).callonsingleCharEscape13,
						expr: &litMatcher{
							pos:        position{line: 756, col: 5, offset: 18829},
							val:        "t",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 757, col: 5, offset: 18858},
						run: (*parser).callonsingleCharEscape15,
						expr: &litMatcher{
							pos:        position{line: 757, col: 5, offset: 18858},
							val:        "v",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "unicodeEscape",
			pos:  position{line: 759, col: 1, offset: 18884},
			expr: &seqExpr{
				pos: position{line: 760, col: 5, offset: 18902},
				exprs: []interface{}{
					&litMatcher{
						pos:        position{line: 760, col: 5, offset: 18902},
						val:        "u",
						ignoreCase: false,
					},
					&ruleRefExpr{
						pos:  position{line: 760, col: 9, offset: 18906},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 760, col: 18, offset: 18915},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 760, col: 27, offset: 18924},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 760, col: 36, offset: 18933},
						name: "hexdigit",
					},
				},
			},
		},
		{
			name: "reString",
			pos:  position{line: 762, col: 1, offset: 18943},
			expr: &actionExpr{
				pos: position{line: 763, col: 5, offset: 18956},
				run: (*parser).callonreString1,
				expr: &seqExpr{
					pos: position{line: 763, col: 5, offset: 18956},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 763, col: 5, offset: 18956},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 763, col: 9, offset: 18960},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 763, col: 11, offset: 18962},
								name: "reBody",
							},
						},
						&litMatcher{
							pos:        position{line: 763, col: 18, offset: 18969},
							val:        "/",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reBody",
			pos:  position{line: 765, col: 1, offset: 18992},
			expr: &actionExpr{
				pos: position{line: 766, col: 5, offset: 19003},
				run: (*parser).callonreBody1,
				expr: &oneOrMoreExpr{
					pos: position{line: 766, col: 5, offset: 19003},
					expr: &choiceExpr{
						pos: position{line: 766, col: 6, offset: 19004},
						alternatives: []interface{}{
							&charClassMatcher{
								pos:        position{line: 766, col: 6, offset: 19004},
								val:        "[^/\\\\]",
								chars:      []rune{'/', '\\'},
								ignoreCase: false,
								inverted:   true,
							},
							&litMatcher{
								pos:        position{line: 766, col: 13, offset: 19011},
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
			pos:  position{line: 768, col: 1, offset: 19051},
			expr: &charClassMatcher{
				pos:        position{line: 769, col: 5, offset: 19067},
				val:        "[\\x00-\\x1f\\\\]",
				chars:      []rune{'\\'},
				ranges:     []rune{'\x00', '\x1f'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "ws",
			pos:  position{line: 771, col: 1, offset: 19082},
			expr: &choiceExpr{
				pos: position{line: 772, col: 5, offset: 19089},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 772, col: 5, offset: 19089},
						val:        "\t",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 773, col: 5, offset: 19098},
						val:        "\v",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 774, col: 5, offset: 19107},
						val:        "\f",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 775, col: 5, offset: 19116},
						val:        " ",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 776, col: 5, offset: 19124},
						val:        "\u00a0",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 777, col: 5, offset: 19137},
						val:        "\ufeff",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name:        "_",
			displayName: "\"whitespace\"",
			pos:         position{line: 779, col: 1, offset: 19147},
			expr: &oneOrMoreExpr{
				pos: position{line: 779, col: 18, offset: 19164},
				expr: &ruleRefExpr{
					pos:  position{line: 779, col: 18, offset: 19164},
					name: "ws",
				},
			},
		},
		{
			name: "EOF",
			pos:  position{line: 781, col: 1, offset: 19169},
			expr: &notExpr{
				pos: position{line: 781, col: 7, offset: 19175},
				expr: &anyMatcher{
					line: 781, col: 8, offset: 19176,
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

func (c *current) onboomCommand2(s, rest interface{}) (interface{}, error) {
	if len(rest.([]interface{})) == 0 {
		return s, nil
	} else {
		return makeSequentialProc(s, rest), nil
	}

}

func (p *parser) callonboomCommand2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onboomCommand2(stack["s"], stack["rest"])
}

func (c *current) onboomCommand11(s interface{}) (interface{}, error) {
	return s, nil

}

func (p *parser) callonboomCommand11() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onboomCommand11(stack["s"])
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
	return makeParallelProc(first, rest), nil

}

func (p *parser) callonprocList1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onprocList1(stack["first"], stack["rest"])
}

func (c *current) onparallelChain1(ch interface{}) (interface{}, error) {
	return ch, nil
}

func (p *parser) callonparallelChain1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onparallelChain1(stack["ch"])
}

func (c *current) onprocChain1(first, rest interface{}) (interface{}, error) {
	if len(rest.([]interface{})) == 0 {
		return first, nil
	} else {
		return makeSequentialProc(first, rest), nil
	}

}

func (p *parser) callonprocChain1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onprocChain1(stack["first"], stack["rest"])
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

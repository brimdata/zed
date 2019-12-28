package zql

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mccanne/zq/reglob"
)

var g = &grammar{
	rules: []*rule{
		{
			name: "start",
			pos:  position{line: 12, col: 1, offset: 69},
			expr: &actionExpr{
				pos: position{line: 12, col: 9, offset: 77},
				run: (*parser).callonstart1,
				expr: &seqExpr{
					pos: position{line: 12, col: 9, offset: 77},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 12, col: 9, offset: 77},
							expr: &ruleRefExpr{
								pos:  position{line: 12, col: 9, offset: 77},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 12, col: 12, offset: 80},
							label: "ast",
							expr: &ruleRefExpr{
								pos:  position{line: 12, col: 16, offset: 84},
								name: "boomCommand",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 12, col: 28, offset: 96},
							expr: &ruleRefExpr{
								pos:  position{line: 12, col: 28, offset: 96},
								name: "_",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 12, col: 31, offset: 99},
							name: "EOF",
						},
					},
				},
			},
		},
		{
			name: "boomCommand",
			pos:  position{line: 14, col: 1, offset: 124},
			expr: &choiceExpr{
				pos: position{line: 15, col: 5, offset: 140},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 15, col: 5, offset: 140},
						run: (*parser).callonboomCommand2,
						expr: &labeledExpr{
							pos:   position{line: 15, col: 5, offset: 140},
							label: "procs",
							expr: &ruleRefExpr{
								pos:  position{line: 15, col: 11, offset: 146},
								name: "procChain",
							},
						},
					},
					&actionExpr{
						pos: position{line: 19, col: 5, offset: 319},
						run: (*parser).callonboomCommand5,
						expr: &seqExpr{
							pos: position{line: 19, col: 5, offset: 319},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 19, col: 5, offset: 319},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 19, col: 7, offset: 321},
										name: "search",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 19, col: 14, offset: 328},
									expr: &ruleRefExpr{
										pos:  position{line: 19, col: 14, offset: 328},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 19, col: 17, offset: 331},
									label: "rest",
									expr: &zeroOrMoreExpr{
										pos: position{line: 19, col: 22, offset: 336},
										expr: &ruleRefExpr{
											pos:  position{line: 19, col: 22, offset: 336},
											name: "chainedProc",
										},
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 26, col: 5, offset: 546},
						run: (*parser).callonboomCommand14,
						expr: &labeledExpr{
							pos:   position{line: 26, col: 5, offset: 546},
							label: "s",
							expr: &ruleRefExpr{
								pos:  position{line: 26, col: 7, offset: 548},
								name: "search",
							},
						},
					},
				},
			},
		},
		{
			name: "procChain",
			pos:  position{line: 30, col: 1, offset: 619},
			expr: &actionExpr{
				pos: position{line: 31, col: 5, offset: 633},
				run: (*parser).callonprocChain1,
				expr: &seqExpr{
					pos: position{line: 31, col: 5, offset: 633},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 31, col: 5, offset: 633},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 31, col: 11, offset: 639},
								name: "proc",
							},
						},
						&labeledExpr{
							pos:   position{line: 31, col: 16, offset: 644},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 31, col: 21, offset: 649},
								expr: &ruleRefExpr{
									pos:  position{line: 31, col: 21, offset: 649},
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
			pos:  position{line: 39, col: 1, offset: 835},
			expr: &actionExpr{
				pos: position{line: 39, col: 15, offset: 849},
				run: (*parser).callonchainedProc1,
				expr: &seqExpr{
					pos: position{line: 39, col: 15, offset: 849},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 39, col: 15, offset: 849},
							expr: &ruleRefExpr{
								pos:  position{line: 39, col: 15, offset: 849},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 39, col: 18, offset: 852},
							val:        "|",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 39, col: 22, offset: 856},
							expr: &ruleRefExpr{
								pos:  position{line: 39, col: 22, offset: 856},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 39, col: 25, offset: 859},
							label: "p",
							expr: &ruleRefExpr{
								pos:  position{line: 39, col: 27, offset: 861},
								name: "proc",
							},
						},
					},
				},
			},
		},
		{
			name: "search",
			pos:  position{line: 41, col: 1, offset: 885},
			expr: &actionExpr{
				pos: position{line: 42, col: 5, offset: 896},
				run: (*parser).callonsearch1,
				expr: &labeledExpr{
					pos:   position{line: 42, col: 5, offset: 896},
					label: "expr",
					expr: &ruleRefExpr{
						pos:  position{line: 42, col: 10, offset: 901},
						name: "searchExpr",
					},
				},
			},
		},
		{
			name: "searchExpr",
			pos:  position{line: 46, col: 1, offset: 960},
			expr: &actionExpr{
				pos: position{line: 47, col: 5, offset: 975},
				run: (*parser).callonsearchExpr1,
				expr: &seqExpr{
					pos: position{line: 47, col: 5, offset: 975},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 47, col: 5, offset: 975},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 47, col: 11, offset: 981},
								name: "searchTerm",
							},
						},
						&labeledExpr{
							pos:   position{line: 47, col: 22, offset: 992},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 47, col: 27, offset: 997},
								expr: &ruleRefExpr{
									pos:  position{line: 47, col: 27, offset: 997},
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
			pos:  position{line: 51, col: 1, offset: 1065},
			expr: &actionExpr{
				pos: position{line: 51, col: 18, offset: 1082},
				run: (*parser).callonoredSearchTerm1,
				expr: &seqExpr{
					pos: position{line: 51, col: 18, offset: 1082},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 51, col: 18, offset: 1082},
							name: "_",
						},
						&ruleRefExpr{
							pos:  position{line: 51, col: 20, offset: 1084},
							name: "orToken",
						},
						&ruleRefExpr{
							pos:  position{line: 51, col: 28, offset: 1092},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 51, col: 30, offset: 1094},
							label: "t",
							expr: &ruleRefExpr{
								pos:  position{line: 51, col: 32, offset: 1096},
								name: "searchTerm",
							},
						},
					},
				},
			},
		},
		{
			name: "searchTerm",
			pos:  position{line: 53, col: 1, offset: 1126},
			expr: &actionExpr{
				pos: position{line: 54, col: 5, offset: 1141},
				run: (*parser).callonsearchTerm1,
				expr: &seqExpr{
					pos: position{line: 54, col: 5, offset: 1141},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 54, col: 5, offset: 1141},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 54, col: 11, offset: 1147},
								name: "searchFactor",
							},
						},
						&labeledExpr{
							pos:   position{line: 54, col: 24, offset: 1160},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 54, col: 29, offset: 1165},
								expr: &ruleRefExpr{
									pos:  position{line: 54, col: 29, offset: 1165},
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
			pos:  position{line: 58, col: 1, offset: 1235},
			expr: &actionExpr{
				pos: position{line: 58, col: 19, offset: 1253},
				run: (*parser).callonandedSearchTerm1,
				expr: &seqExpr{
					pos: position{line: 58, col: 19, offset: 1253},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 58, col: 19, offset: 1253},
							name: "_",
						},
						&zeroOrOneExpr{
							pos: position{line: 58, col: 21, offset: 1255},
							expr: &seqExpr{
								pos: position{line: 58, col: 22, offset: 1256},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 58, col: 22, offset: 1256},
										name: "andToken",
									},
									&ruleRefExpr{
										pos:  position{line: 58, col: 31, offset: 1265},
										name: "_",
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 58, col: 35, offset: 1269},
							label: "f",
							expr: &ruleRefExpr{
								pos:  position{line: 58, col: 37, offset: 1271},
								name: "searchFactor",
							},
						},
					},
				},
			},
		},
		{
			name: "searchFactor",
			pos:  position{line: 60, col: 1, offset: 1303},
			expr: &choiceExpr{
				pos: position{line: 61, col: 5, offset: 1320},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 61, col: 5, offset: 1320},
						run: (*parser).callonsearchFactor2,
						expr: &seqExpr{
							pos: position{line: 61, col: 5, offset: 1320},
							exprs: []interface{}{
								&choiceExpr{
									pos: position{line: 61, col: 6, offset: 1321},
									alternatives: []interface{}{
										&seqExpr{
											pos: position{line: 61, col: 6, offset: 1321},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 61, col: 6, offset: 1321},
													name: "notToken",
												},
												&ruleRefExpr{
													pos:  position{line: 61, col: 15, offset: 1330},
													name: "_",
												},
											},
										},
										&seqExpr{
											pos: position{line: 61, col: 19, offset: 1334},
											exprs: []interface{}{
												&litMatcher{
													pos:        position{line: 61, col: 19, offset: 1334},
													val:        "!",
													ignoreCase: false,
												},
												&zeroOrOneExpr{
													pos: position{line: 61, col: 23, offset: 1338},
													expr: &ruleRefExpr{
														pos:  position{line: 61, col: 23, offset: 1338},
														name: "_",
													},
												},
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 61, col: 27, offset: 1342},
									label: "e",
									expr: &ruleRefExpr{
										pos:  position{line: 61, col: 29, offset: 1344},
										name: "searchExpr",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 64, col: 5, offset: 1403},
						run: (*parser).callonsearchFactor14,
						expr: &seqExpr{
							pos: position{line: 64, col: 5, offset: 1403},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 64, col: 5, offset: 1403},
									expr: &litMatcher{
										pos:        position{line: 64, col: 7, offset: 1405},
										val:        "-",
										ignoreCase: false,
									},
								},
								&labeledExpr{
									pos:   position{line: 64, col: 12, offset: 1410},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 64, col: 14, offset: 1412},
										name: "searchPred",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 65, col: 5, offset: 1445},
						run: (*parser).callonsearchFactor20,
						expr: &seqExpr{
							pos: position{line: 65, col: 5, offset: 1445},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 65, col: 5, offset: 1445},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 65, col: 9, offset: 1449},
									expr: &ruleRefExpr{
										pos:  position{line: 65, col: 9, offset: 1449},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 65, col: 12, offset: 1452},
									label: "expr",
									expr: &ruleRefExpr{
										pos:  position{line: 65, col: 17, offset: 1457},
										name: "searchExpr",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 65, col: 28, offset: 1468},
									expr: &ruleRefExpr{
										pos:  position{line: 65, col: 28, offset: 1468},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 65, col: 31, offset: 1471},
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
			pos:  position{line: 67, col: 1, offset: 1497},
			expr: &choiceExpr{
				pos: position{line: 68, col: 5, offset: 1512},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 68, col: 5, offset: 1512},
						run: (*parser).callonsearchPred2,
						expr: &seqExpr{
							pos: position{line: 68, col: 5, offset: 1512},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 68, col: 5, offset: 1512},
									val:        "*",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 68, col: 9, offset: 1516},
									expr: &ruleRefExpr{
										pos:  position{line: 68, col: 9, offset: 1516},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 68, col: 12, offset: 1519},
									label: "fieldComparator",
									expr: &ruleRefExpr{
										pos:  position{line: 68, col: 28, offset: 1535},
										name: "equalityToken",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 68, col: 42, offset: 1549},
									expr: &ruleRefExpr{
										pos:  position{line: 68, col: 42, offset: 1549},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 68, col: 45, offset: 1552},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 68, col: 47, offset: 1554},
										name: "searchValue",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 71, col: 5, offset: 1638},
						run: (*parser).callonsearchPred13,
						expr: &seqExpr{
							pos: position{line: 71, col: 5, offset: 1638},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 71, col: 5, offset: 1638},
									val:        "**",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 71, col: 10, offset: 1643},
									expr: &ruleRefExpr{
										pos:  position{line: 71, col: 10, offset: 1643},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 71, col: 13, offset: 1646},
									label: "fieldComparator",
									expr: &ruleRefExpr{
										pos:  position{line: 71, col: 29, offset: 1662},
										name: "equalityToken",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 71, col: 43, offset: 1676},
									expr: &ruleRefExpr{
										pos:  position{line: 71, col: 43, offset: 1676},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 71, col: 46, offset: 1679},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 71, col: 48, offset: 1681},
										name: "searchValue",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 74, col: 5, offset: 1764},
						run: (*parser).callonsearchPred24,
						expr: &litMatcher{
							pos:        position{line: 74, col: 5, offset: 1764},
							val:        "*",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 77, col: 5, offset: 1823},
						run: (*parser).callonsearchPred26,
						expr: &seqExpr{
							pos: position{line: 77, col: 5, offset: 1823},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 77, col: 5, offset: 1823},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 77, col: 7, offset: 1825},
										name: "fieldExpr",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 77, col: 17, offset: 1835},
									expr: &ruleRefExpr{
										pos:  position{line: 77, col: 17, offset: 1835},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 77, col: 20, offset: 1838},
									label: "fieldComparator",
									expr: &ruleRefExpr{
										pos:  position{line: 77, col: 36, offset: 1854},
										name: "equalityToken",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 77, col: 50, offset: 1868},
									expr: &ruleRefExpr{
										pos:  position{line: 77, col: 50, offset: 1868},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 77, col: 53, offset: 1871},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 77, col: 55, offset: 1873},
										name: "searchValue",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 80, col: 5, offset: 1955},
						run: (*parser).callonsearchPred38,
						expr: &seqExpr{
							pos: position{line: 80, col: 5, offset: 1955},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 80, col: 5, offset: 1955},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 80, col: 7, offset: 1957},
										name: "searchValue",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 80, col: 19, offset: 1969},
									expr: &ruleRefExpr{
										pos:  position{line: 80, col: 19, offset: 1969},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 80, col: 22, offset: 1972},
									name: "inToken",
								},
								&zeroOrOneExpr{
									pos: position{line: 80, col: 30, offset: 1980},
									expr: &ruleRefExpr{
										pos:  position{line: 80, col: 30, offset: 1980},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 80, col: 33, offset: 1983},
									val:        "*",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 83, col: 5, offset: 2048},
						run: (*parser).callonsearchPred48,
						expr: &seqExpr{
							pos: position{line: 83, col: 5, offset: 2048},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 83, col: 5, offset: 2048},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 83, col: 7, offset: 2050},
										name: "searchValue",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 83, col: 19, offset: 2062},
									expr: &ruleRefExpr{
										pos:  position{line: 83, col: 19, offset: 2062},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 83, col: 22, offset: 2065},
									name: "inToken",
								},
								&zeroOrOneExpr{
									pos: position{line: 83, col: 30, offset: 2073},
									expr: &ruleRefExpr{
										pos:  position{line: 83, col: 30, offset: 2073},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 83, col: 33, offset: 2076},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 83, col: 35, offset: 2078},
										name: "fieldReference",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 86, col: 5, offset: 2152},
						run: (*parser).callonsearchPred59,
						expr: &labeledExpr{
							pos:   position{line: 86, col: 5, offset: 2152},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 86, col: 7, offset: 2154},
								name: "searchValue",
							},
						},
					},
				},
			},
		},
		{
			name: "searchValue",
			pos:  position{line: 95, col: 1, offset: 2629},
			expr: &choiceExpr{
				pos: position{line: 96, col: 5, offset: 2645},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 96, col: 5, offset: 2645},
						run: (*parser).callonsearchValue2,
						expr: &labeledExpr{
							pos:   position{line: 96, col: 5, offset: 2645},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 96, col: 7, offset: 2647},
								name: "quotedString",
							},
						},
					},
					&actionExpr{
						pos: position{line: 99, col: 5, offset: 2718},
						run: (*parser).callonsearchValue5,
						expr: &labeledExpr{
							pos:   position{line: 99, col: 5, offset: 2718},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 99, col: 7, offset: 2720},
								name: "reString",
							},
						},
					},
					&actionExpr{
						pos: position{line: 102, col: 5, offset: 2787},
						run: (*parser).callonsearchValue8,
						expr: &labeledExpr{
							pos:   position{line: 102, col: 5, offset: 2787},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 102, col: 7, offset: 2789},
								name: "port",
							},
						},
					},
					&actionExpr{
						pos: position{line: 105, col: 5, offset: 2848},
						run: (*parser).callonsearchValue11,
						expr: &labeledExpr{
							pos:   position{line: 105, col: 5, offset: 2848},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 105, col: 7, offset: 2850},
								name: "ip6subnet",
							},
						},
					},
					&actionExpr{
						pos: position{line: 108, col: 5, offset: 2918},
						run: (*parser).callonsearchValue14,
						expr: &labeledExpr{
							pos:   position{line: 108, col: 5, offset: 2918},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 108, col: 7, offset: 2920},
								name: "ip6addr",
							},
						},
					},
					&actionExpr{
						pos: position{line: 111, col: 5, offset: 2984},
						run: (*parser).callonsearchValue17,
						expr: &labeledExpr{
							pos:   position{line: 111, col: 5, offset: 2984},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 111, col: 7, offset: 2986},
								name: "subnet",
							},
						},
					},
					&actionExpr{
						pos: position{line: 114, col: 5, offset: 3051},
						run: (*parser).callonsearchValue20,
						expr: &labeledExpr{
							pos:   position{line: 114, col: 5, offset: 3051},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 114, col: 7, offset: 3053},
								name: "addr",
							},
						},
					},
					&actionExpr{
						pos: position{line: 117, col: 5, offset: 3114},
						run: (*parser).callonsearchValue23,
						expr: &labeledExpr{
							pos:   position{line: 117, col: 5, offset: 3114},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 117, col: 7, offset: 3116},
								name: "sdouble",
							},
						},
					},
					&actionExpr{
						pos: position{line: 120, col: 5, offset: 3182},
						run: (*parser).callonsearchValue26,
						expr: &seqExpr{
							pos: position{line: 120, col: 5, offset: 3182},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 120, col: 5, offset: 3182},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 120, col: 7, offset: 3184},
										name: "sinteger",
									},
								},
								&notExpr{
									pos: position{line: 120, col: 16, offset: 3193},
									expr: &ruleRefExpr{
										pos:  position{line: 120, col: 17, offset: 3194},
										name: "boomWord",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 123, col: 5, offset: 3258},
						run: (*parser).callonsearchValue32,
						expr: &seqExpr{
							pos: position{line: 123, col: 5, offset: 3258},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 123, col: 5, offset: 3258},
									expr: &seqExpr{
										pos: position{line: 123, col: 7, offset: 3260},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 123, col: 7, offset: 3260},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 123, col: 22, offset: 3275},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 123, col: 25, offset: 3278},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 123, col: 27, offset: 3280},
										name: "booleanLiteral",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 124, col: 5, offset: 3317},
						run: (*parser).callonsearchValue40,
						expr: &seqExpr{
							pos: position{line: 124, col: 5, offset: 3317},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 124, col: 5, offset: 3317},
									expr: &seqExpr{
										pos: position{line: 124, col: 7, offset: 3319},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 124, col: 7, offset: 3319},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 124, col: 22, offset: 3334},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 124, col: 25, offset: 3337},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 124, col: 27, offset: 3339},
										name: "unsetLiteral",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 125, col: 5, offset: 3374},
						run: (*parser).callonsearchValue48,
						expr: &seqExpr{
							pos: position{line: 125, col: 5, offset: 3374},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 125, col: 5, offset: 3374},
									expr: &seqExpr{
										pos: position{line: 125, col: 7, offset: 3376},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 125, col: 7, offset: 3376},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 125, col: 22, offset: 3391},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 125, col: 25, offset: 3394},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 125, col: 27, offset: 3396},
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
			pos:  position{line: 133, col: 1, offset: 3620},
			expr: &choiceExpr{
				pos: position{line: 134, col: 5, offset: 3639},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 134, col: 5, offset: 3639},
						name: "andToken",
					},
					&ruleRefExpr{
						pos:  position{line: 135, col: 5, offset: 3652},
						name: "orToken",
					},
					&ruleRefExpr{
						pos:  position{line: 136, col: 5, offset: 3664},
						name: "inToken",
					},
				},
			},
		},
		{
			name: "booleanLiteral",
			pos:  position{line: 138, col: 1, offset: 3673},
			expr: &choiceExpr{
				pos: position{line: 139, col: 5, offset: 3692},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 139, col: 5, offset: 3692},
						run: (*parser).callonbooleanLiteral2,
						expr: &litMatcher{
							pos:        position{line: 139, col: 5, offset: 3692},
							val:        "true",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 140, col: 5, offset: 3760},
						run: (*parser).callonbooleanLiteral4,
						expr: &litMatcher{
							pos:        position{line: 140, col: 5, offset: 3760},
							val:        "false",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "unsetLiteral",
			pos:  position{line: 142, col: 1, offset: 3826},
			expr: &actionExpr{
				pos: position{line: 143, col: 5, offset: 3843},
				run: (*parser).callonunsetLiteral1,
				expr: &litMatcher{
					pos:        position{line: 143, col: 5, offset: 3843},
					val:        "nil",
					ignoreCase: false,
				},
			},
		},
		{
			name: "procList",
			pos:  position{line: 145, col: 1, offset: 3904},
			expr: &actionExpr{
				pos: position{line: 146, col: 5, offset: 3917},
				run: (*parser).callonprocList1,
				expr: &seqExpr{
					pos: position{line: 146, col: 5, offset: 3917},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 146, col: 5, offset: 3917},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 146, col: 11, offset: 3923},
								name: "procChain",
							},
						},
						&labeledExpr{
							pos:   position{line: 146, col: 21, offset: 3933},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 146, col: 26, offset: 3938},
								expr: &ruleRefExpr{
									pos:  position{line: 146, col: 26, offset: 3938},
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
			pos:  position{line: 155, col: 1, offset: 4162},
			expr: &actionExpr{
				pos: position{line: 156, col: 5, offset: 4180},
				run: (*parser).callonparallelChain1,
				expr: &seqExpr{
					pos: position{line: 156, col: 5, offset: 4180},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 156, col: 5, offset: 4180},
							expr: &ruleRefExpr{
								pos:  position{line: 156, col: 5, offset: 4180},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 156, col: 8, offset: 4183},
							val:        ";",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 156, col: 12, offset: 4187},
							expr: &ruleRefExpr{
								pos:  position{line: 156, col: 12, offset: 4187},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 156, col: 15, offset: 4190},
							label: "ch",
							expr: &ruleRefExpr{
								pos:  position{line: 156, col: 18, offset: 4193},
								name: "procChain",
							},
						},
					},
				},
			},
		},
		{
			name: "proc",
			pos:  position{line: 158, col: 1, offset: 4243},
			expr: &choiceExpr{
				pos: position{line: 159, col: 5, offset: 4252},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 159, col: 5, offset: 4252},
						name: "simpleProc",
					},
					&ruleRefExpr{
						pos:  position{line: 160, col: 5, offset: 4267},
						name: "reducerProc",
					},
					&actionExpr{
						pos: position{line: 161, col: 5, offset: 4283},
						run: (*parser).callonproc4,
						expr: &seqExpr{
							pos: position{line: 161, col: 5, offset: 4283},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 161, col: 5, offset: 4283},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 161, col: 9, offset: 4287},
									expr: &ruleRefExpr{
										pos:  position{line: 161, col: 9, offset: 4287},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 161, col: 12, offset: 4290},
									label: "proc",
									expr: &ruleRefExpr{
										pos:  position{line: 161, col: 17, offset: 4295},
										name: "procList",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 161, col: 26, offset: 4304},
									expr: &ruleRefExpr{
										pos:  position{line: 161, col: 26, offset: 4304},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 161, col: 29, offset: 4307},
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
			pos:  position{line: 165, col: 1, offset: 4343},
			expr: &actionExpr{
				pos: position{line: 166, col: 5, offset: 4355},
				run: (*parser).callongroupBy1,
				expr: &seqExpr{
					pos: position{line: 166, col: 5, offset: 4355},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 166, col: 5, offset: 4355},
							val:        "by",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 166, col: 11, offset: 4361},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 166, col: 13, offset: 4363},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 166, col: 18, offset: 4368},
								name: "fieldExprList",
							},
						},
					},
				},
			},
		},
		{
			name: "everyDur",
			pos:  position{line: 168, col: 1, offset: 4404},
			expr: &actionExpr{
				pos: position{line: 169, col: 5, offset: 4417},
				run: (*parser).calloneveryDur1,
				expr: &seqExpr{
					pos: position{line: 169, col: 5, offset: 4417},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 169, col: 5, offset: 4417},
							val:        "every",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 169, col: 14, offset: 4426},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 169, col: 16, offset: 4428},
							label: "dur",
							expr: &ruleRefExpr{
								pos:  position{line: 169, col: 20, offset: 4432},
								name: "duration",
							},
						},
					},
				},
			},
		},
		{
			name: "equalityToken",
			pos:  position{line: 171, col: 1, offset: 4462},
			expr: &choiceExpr{
				pos: position{line: 172, col: 5, offset: 4480},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 172, col: 5, offset: 4480},
						run: (*parser).callonequalityToken2,
						expr: &litMatcher{
							pos:        position{line: 172, col: 5, offset: 4480},
							val:        "=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 173, col: 5, offset: 4510},
						run: (*parser).callonequalityToken4,
						expr: &litMatcher{
							pos:        position{line: 173, col: 5, offset: 4510},
							val:        "!=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 174, col: 5, offset: 4542},
						run: (*parser).callonequalityToken6,
						expr: &litMatcher{
							pos:        position{line: 174, col: 5, offset: 4542},
							val:        "<=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 175, col: 5, offset: 4573},
						run: (*parser).callonequalityToken8,
						expr: &litMatcher{
							pos:        position{line: 175, col: 5, offset: 4573},
							val:        ">=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 176, col: 5, offset: 4604},
						run: (*parser).callonequalityToken10,
						expr: &litMatcher{
							pos:        position{line: 176, col: 5, offset: 4604},
							val:        "<",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 177, col: 5, offset: 4633},
						run: (*parser).callonequalityToken12,
						expr: &litMatcher{
							pos:        position{line: 177, col: 5, offset: 4633},
							val:        ">",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "types",
			pos:  position{line: 179, col: 1, offset: 4659},
			expr: &choiceExpr{
				pos: position{line: 180, col: 5, offset: 4669},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 180, col: 5, offset: 4669},
						val:        "bool",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 181, col: 5, offset: 4680},
						val:        "int",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 182, col: 5, offset: 4690},
						val:        "count",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 183, col: 5, offset: 4702},
						val:        "double",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 184, col: 5, offset: 4715},
						val:        "string",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 185, col: 5, offset: 4728},
						val:        "addr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 186, col: 5, offset: 4739},
						val:        "subnet",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 187, col: 5, offset: 4752},
						val:        "port",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "dash",
			pos:  position{line: 189, col: 1, offset: 4760},
			expr: &choiceExpr{
				pos: position{line: 189, col: 8, offset: 4767},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 189, col: 8, offset: 4767},
						val:        "-",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 189, col: 14, offset: 4773},
						val:        "—",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 189, col: 25, offset: 4784},
						val:        "–",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 189, col: 36, offset: 4795},
						exprs: []interface{}{
							&litMatcher{
								pos:        position{line: 189, col: 36, offset: 4795},
								val:        "to",
								ignoreCase: false,
							},
							&andExpr{
								pos: position{line: 189, col: 40, offset: 4799},
								expr: &ruleRefExpr{
									pos:  position{line: 189, col: 42, offset: 4801},
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
			pos:  position{line: 191, col: 1, offset: 4805},
			expr: &litMatcher{
				pos:        position{line: 191, col: 12, offset: 4816},
				val:        "and",
				ignoreCase: false,
			},
		},
		{
			name: "orToken",
			pos:  position{line: 192, col: 1, offset: 4822},
			expr: &litMatcher{
				pos:        position{line: 192, col: 11, offset: 4832},
				val:        "or",
				ignoreCase: false,
			},
		},
		{
			name: "inToken",
			pos:  position{line: 193, col: 1, offset: 4837},
			expr: &litMatcher{
				pos:        position{line: 193, col: 11, offset: 4847},
				val:        "in",
				ignoreCase: false,
			},
		},
		{
			name: "notToken",
			pos:  position{line: 194, col: 1, offset: 4852},
			expr: &litMatcher{
				pos:        position{line: 194, col: 12, offset: 4863},
				val:        "not",
				ignoreCase: false,
			},
		},
		{
			name: "fieldName",
			pos:  position{line: 196, col: 1, offset: 4870},
			expr: &actionExpr{
				pos: position{line: 196, col: 13, offset: 4882},
				run: (*parser).callonfieldName1,
				expr: &seqExpr{
					pos: position{line: 196, col: 13, offset: 4882},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 196, col: 13, offset: 4882},
							name: "fieldNameStart",
						},
						&zeroOrMoreExpr{
							pos: position{line: 196, col: 28, offset: 4897},
							expr: &ruleRefExpr{
								pos:  position{line: 196, col: 28, offset: 4897},
								name: "fieldNameRest",
							},
						},
					},
				},
			},
		},
		{
			name: "fieldNameStart",
			pos:  position{line: 198, col: 1, offset: 4944},
			expr: &charClassMatcher{
				pos:        position{line: 198, col: 18, offset: 4961},
				val:        "[A-Za-z_$]",
				chars:      []rune{'_', '$'},
				ranges:     []rune{'A', 'Z', 'a', 'z'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "fieldNameRest",
			pos:  position{line: 199, col: 1, offset: 4972},
			expr: &choiceExpr{
				pos: position{line: 199, col: 17, offset: 4988},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 199, col: 17, offset: 4988},
						name: "fieldNameStart",
					},
					&charClassMatcher{
						pos:        position{line: 199, col: 34, offset: 5005},
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
			pos:  position{line: 201, col: 1, offset: 5012},
			expr: &actionExpr{
				pos: position{line: 202, col: 4, offset: 5030},
				run: (*parser).callonfieldReference1,
				expr: &seqExpr{
					pos: position{line: 202, col: 4, offset: 5030},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 202, col: 4, offset: 5030},
							label: "base",
							expr: &ruleRefExpr{
								pos:  position{line: 202, col: 9, offset: 5035},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 202, col: 19, offset: 5045},
							label: "derefs",
							expr: &zeroOrMoreExpr{
								pos: position{line: 202, col: 26, offset: 5052},
								expr: &choiceExpr{
									pos: position{line: 203, col: 8, offset: 5061},
									alternatives: []interface{}{
										&actionExpr{
											pos: position{line: 203, col: 8, offset: 5061},
											run: (*parser).callonfieldReference8,
											expr: &seqExpr{
												pos: position{line: 203, col: 8, offset: 5061},
												exprs: []interface{}{
													&litMatcher{
														pos:        position{line: 203, col: 8, offset: 5061},
														val:        ".",
														ignoreCase: false,
													},
													&labeledExpr{
														pos:   position{line: 203, col: 12, offset: 5065},
														label: "field",
														expr: &ruleRefExpr{
															pos:  position{line: 203, col: 18, offset: 5071},
															name: "fieldName",
														},
													},
												},
											},
										},
										&actionExpr{
											pos: position{line: 204, col: 8, offset: 5152},
											run: (*parser).callonfieldReference13,
											expr: &seqExpr{
												pos: position{line: 204, col: 8, offset: 5152},
												exprs: []interface{}{
													&litMatcher{
														pos:        position{line: 204, col: 8, offset: 5152},
														val:        "[",
														ignoreCase: false,
													},
													&labeledExpr{
														pos:   position{line: 204, col: 12, offset: 5156},
														label: "index",
														expr: &ruleRefExpr{
															pos:  position{line: 204, col: 18, offset: 5162},
															name: "sinteger",
														},
													},
													&litMatcher{
														pos:        position{line: 204, col: 27, offset: 5171},
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
			pos:  position{line: 209, col: 1, offset: 5287},
			expr: &choiceExpr{
				pos: position{line: 210, col: 5, offset: 5301},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 210, col: 5, offset: 5301},
						run: (*parser).callonfieldExpr2,
						expr: &seqExpr{
							pos: position{line: 210, col: 5, offset: 5301},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 210, col: 5, offset: 5301},
									label: "op",
									expr: &ruleRefExpr{
										pos:  position{line: 210, col: 8, offset: 5304},
										name: "fieldOp",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 210, col: 16, offset: 5312},
									expr: &ruleRefExpr{
										pos:  position{line: 210, col: 16, offset: 5312},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 210, col: 19, offset: 5315},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 210, col: 23, offset: 5319},
									expr: &ruleRefExpr{
										pos:  position{line: 210, col: 23, offset: 5319},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 210, col: 26, offset: 5322},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 210, col: 32, offset: 5328},
										name: "fieldReference",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 210, col: 47, offset: 5343},
									expr: &ruleRefExpr{
										pos:  position{line: 210, col: 47, offset: 5343},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 210, col: 50, offset: 5346},
									val:        ")",
									ignoreCase: false,
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 213, col: 5, offset: 5410},
						name: "fieldReference",
					},
				},
			},
		},
		{
			name: "fieldOp",
			pos:  position{line: 215, col: 1, offset: 5426},
			expr: &actionExpr{
				pos: position{line: 216, col: 5, offset: 5438},
				run: (*parser).callonfieldOp1,
				expr: &litMatcher{
					pos:        position{line: 216, col: 5, offset: 5438},
					val:        "len",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldExprList",
			pos:  position{line: 218, col: 1, offset: 5468},
			expr: &actionExpr{
				pos: position{line: 219, col: 5, offset: 5486},
				run: (*parser).callonfieldExprList1,
				expr: &seqExpr{
					pos: position{line: 219, col: 5, offset: 5486},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 219, col: 5, offset: 5486},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 219, col: 11, offset: 5492},
								name: "fieldExpr",
							},
						},
						&labeledExpr{
							pos:   position{line: 219, col: 21, offset: 5502},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 219, col: 26, offset: 5507},
								expr: &seqExpr{
									pos: position{line: 219, col: 27, offset: 5508},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 219, col: 27, offset: 5508},
											expr: &ruleRefExpr{
												pos:  position{line: 219, col: 27, offset: 5508},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 219, col: 30, offset: 5511},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 219, col: 34, offset: 5515},
											expr: &ruleRefExpr{
												pos:  position{line: 219, col: 34, offset: 5515},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 219, col: 37, offset: 5518},
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
			name: "fieldRefDotOnly",
			pos:  position{line: 229, col: 1, offset: 5713},
			expr: &actionExpr{
				pos: position{line: 230, col: 5, offset: 5733},
				run: (*parser).callonfieldRefDotOnly1,
				expr: &seqExpr{
					pos: position{line: 230, col: 5, offset: 5733},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 230, col: 5, offset: 5733},
							label: "base",
							expr: &ruleRefExpr{
								pos:  position{line: 230, col: 10, offset: 5738},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 230, col: 20, offset: 5748},
							label: "refs",
							expr: &zeroOrMoreExpr{
								pos: position{line: 230, col: 25, offset: 5753},
								expr: &actionExpr{
									pos: position{line: 230, col: 26, offset: 5754},
									run: (*parser).callonfieldRefDotOnly7,
									expr: &seqExpr{
										pos: position{line: 230, col: 26, offset: 5754},
										exprs: []interface{}{
											&litMatcher{
												pos:        position{line: 230, col: 26, offset: 5754},
												val:        ".",
												ignoreCase: false,
											},
											&labeledExpr{
												pos:   position{line: 230, col: 30, offset: 5758},
												label: "field",
												expr: &ruleRefExpr{
													pos:  position{line: 230, col: 36, offset: 5764},
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
			},
		},
		{
			name: "fieldRefDotOnlyList",
			pos:  position{line: 234, col: 1, offset: 5889},
			expr: &actionExpr{
				pos: position{line: 235, col: 5, offset: 5913},
				run: (*parser).callonfieldRefDotOnlyList1,
				expr: &seqExpr{
					pos: position{line: 235, col: 5, offset: 5913},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 235, col: 5, offset: 5913},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 235, col: 11, offset: 5919},
								name: "fieldRefDotOnly",
							},
						},
						&labeledExpr{
							pos:   position{line: 235, col: 27, offset: 5935},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 235, col: 32, offset: 5940},
								expr: &actionExpr{
									pos: position{line: 235, col: 33, offset: 5941},
									run: (*parser).callonfieldRefDotOnlyList7,
									expr: &seqExpr{
										pos: position{line: 235, col: 33, offset: 5941},
										exprs: []interface{}{
											&zeroOrOneExpr{
												pos: position{line: 235, col: 33, offset: 5941},
												expr: &ruleRefExpr{
													pos:  position{line: 235, col: 33, offset: 5941},
													name: "_",
												},
											},
											&litMatcher{
												pos:        position{line: 235, col: 36, offset: 5944},
												val:        ",",
												ignoreCase: false,
											},
											&zeroOrOneExpr{
												pos: position{line: 235, col: 40, offset: 5948},
												expr: &ruleRefExpr{
													pos:  position{line: 235, col: 40, offset: 5948},
													name: "_",
												},
											},
											&labeledExpr{
												pos:   position{line: 235, col: 43, offset: 5951},
												label: "ref",
												expr: &ruleRefExpr{
													pos:  position{line: 235, col: 47, offset: 5955},
													name: "fieldRefDotOnly",
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
			name: "fieldNameList",
			pos:  position{line: 243, col: 1, offset: 6135},
			expr: &actionExpr{
				pos: position{line: 244, col: 5, offset: 6153},
				run: (*parser).callonfieldNameList1,
				expr: &seqExpr{
					pos: position{line: 244, col: 5, offset: 6153},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 244, col: 5, offset: 6153},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 244, col: 11, offset: 6159},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 244, col: 21, offset: 6169},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 244, col: 26, offset: 6174},
								expr: &seqExpr{
									pos: position{line: 244, col: 27, offset: 6175},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 244, col: 27, offset: 6175},
											expr: &ruleRefExpr{
												pos:  position{line: 244, col: 27, offset: 6175},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 244, col: 30, offset: 6178},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 244, col: 34, offset: 6182},
											expr: &ruleRefExpr{
												pos:  position{line: 244, col: 34, offset: 6182},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 244, col: 37, offset: 6185},
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
			pos:  position{line: 252, col: 1, offset: 6378},
			expr: &actionExpr{
				pos: position{line: 253, col: 5, offset: 6390},
				run: (*parser).calloncountOp1,
				expr: &litMatcher{
					pos:        position{line: 253, col: 5, offset: 6390},
					val:        "count",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldReducerOp",
			pos:  position{line: 255, col: 1, offset: 6424},
			expr: &choiceExpr{
				pos: position{line: 256, col: 5, offset: 6443},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 256, col: 5, offset: 6443},
						run: (*parser).callonfieldReducerOp2,
						expr: &litMatcher{
							pos:        position{line: 256, col: 5, offset: 6443},
							val:        "sum",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 257, col: 5, offset: 6477},
						run: (*parser).callonfieldReducerOp4,
						expr: &litMatcher{
							pos:        position{line: 257, col: 5, offset: 6477},
							val:        "avg",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 258, col: 5, offset: 6511},
						run: (*parser).callonfieldReducerOp6,
						expr: &litMatcher{
							pos:        position{line: 258, col: 5, offset: 6511},
							val:        "stdev",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 259, col: 5, offset: 6548},
						run: (*parser).callonfieldReducerOp8,
						expr: &litMatcher{
							pos:        position{line: 259, col: 5, offset: 6548},
							val:        "sd",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 260, col: 5, offset: 6584},
						run: (*parser).callonfieldReducerOp10,
						expr: &litMatcher{
							pos:        position{line: 260, col: 5, offset: 6584},
							val:        "var",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 261, col: 5, offset: 6618},
						run: (*parser).callonfieldReducerOp12,
						expr: &litMatcher{
							pos:        position{line: 261, col: 5, offset: 6618},
							val:        "entropy",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 262, col: 5, offset: 6659},
						run: (*parser).callonfieldReducerOp14,
						expr: &litMatcher{
							pos:        position{line: 262, col: 5, offset: 6659},
							val:        "min",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 263, col: 5, offset: 6693},
						run: (*parser).callonfieldReducerOp16,
						expr: &litMatcher{
							pos:        position{line: 263, col: 5, offset: 6693},
							val:        "max",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 264, col: 5, offset: 6727},
						run: (*parser).callonfieldReducerOp18,
						expr: &litMatcher{
							pos:        position{line: 264, col: 5, offset: 6727},
							val:        "first",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 265, col: 5, offset: 6765},
						run: (*parser).callonfieldReducerOp20,
						expr: &litMatcher{
							pos:        position{line: 265, col: 5, offset: 6765},
							val:        "last",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 266, col: 5, offset: 6801},
						run: (*parser).callonfieldReducerOp22,
						expr: &litMatcher{
							pos:        position{line: 266, col: 5, offset: 6801},
							val:        "countdistinct",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "paddedFieldName",
			pos:  position{line: 268, col: 1, offset: 6851},
			expr: &actionExpr{
				pos: position{line: 268, col: 19, offset: 6869},
				run: (*parser).callonpaddedFieldName1,
				expr: &seqExpr{
					pos: position{line: 268, col: 19, offset: 6869},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 268, col: 19, offset: 6869},
							expr: &ruleRefExpr{
								pos:  position{line: 268, col: 19, offset: 6869},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 268, col: 22, offset: 6872},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 268, col: 28, offset: 6878},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 268, col: 38, offset: 6888},
							expr: &ruleRefExpr{
								pos:  position{line: 268, col: 38, offset: 6888},
								name: "_",
							},
						},
					},
				},
			},
		},
		{
			name: "countReducer",
			pos:  position{line: 270, col: 1, offset: 6914},
			expr: &actionExpr{
				pos: position{line: 271, col: 5, offset: 6931},
				run: (*parser).calloncountReducer1,
				expr: &seqExpr{
					pos: position{line: 271, col: 5, offset: 6931},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 271, col: 5, offset: 6931},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 271, col: 8, offset: 6934},
								name: "countOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 271, col: 16, offset: 6942},
							expr: &ruleRefExpr{
								pos:  position{line: 271, col: 16, offset: 6942},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 271, col: 19, offset: 6945},
							val:        "(",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 271, col: 23, offset: 6949},
							label: "field",
							expr: &zeroOrOneExpr{
								pos: position{line: 271, col: 29, offset: 6955},
								expr: &ruleRefExpr{
									pos:  position{line: 271, col: 29, offset: 6955},
									name: "paddedFieldName",
								},
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 271, col: 47, offset: 6973},
							expr: &ruleRefExpr{
								pos:  position{line: 271, col: 47, offset: 6973},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 271, col: 50, offset: 6976},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "fieldReducer",
			pos:  position{line: 275, col: 1, offset: 7035},
			expr: &actionExpr{
				pos: position{line: 276, col: 5, offset: 7052},
				run: (*parser).callonfieldReducer1,
				expr: &seqExpr{
					pos: position{line: 276, col: 5, offset: 7052},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 276, col: 5, offset: 7052},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 276, col: 8, offset: 7055},
								name: "fieldReducerOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 276, col: 23, offset: 7070},
							expr: &ruleRefExpr{
								pos:  position{line: 276, col: 23, offset: 7070},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 276, col: 26, offset: 7073},
							val:        "(",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 276, col: 30, offset: 7077},
							expr: &ruleRefExpr{
								pos:  position{line: 276, col: 30, offset: 7077},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 276, col: 33, offset: 7080},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 276, col: 39, offset: 7086},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 276, col: 50, offset: 7097},
							expr: &ruleRefExpr{
								pos:  position{line: 276, col: 50, offset: 7097},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 276, col: 53, offset: 7100},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reducerProc",
			pos:  position{line: 280, col: 1, offset: 7167},
			expr: &actionExpr{
				pos: position{line: 281, col: 5, offset: 7183},
				run: (*parser).callonreducerProc1,
				expr: &seqExpr{
					pos: position{line: 281, col: 5, offset: 7183},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 281, col: 5, offset: 7183},
							label: "every",
							expr: &zeroOrOneExpr{
								pos: position{line: 281, col: 11, offset: 7189},
								expr: &seqExpr{
									pos: position{line: 281, col: 12, offset: 7190},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 281, col: 12, offset: 7190},
											name: "everyDur",
										},
										&ruleRefExpr{
											pos:  position{line: 281, col: 21, offset: 7199},
											name: "_",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 281, col: 25, offset: 7203},
							label: "reducers",
							expr: &ruleRefExpr{
								pos:  position{line: 281, col: 34, offset: 7212},
								name: "reducerList",
							},
						},
						&labeledExpr{
							pos:   position{line: 281, col: 46, offset: 7224},
							label: "keys",
							expr: &zeroOrOneExpr{
								pos: position{line: 281, col: 51, offset: 7229},
								expr: &seqExpr{
									pos: position{line: 281, col: 52, offset: 7230},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 281, col: 52, offset: 7230},
											name: "_",
										},
										&ruleRefExpr{
											pos:  position{line: 281, col: 54, offset: 7232},
											name: "groupBy",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 281, col: 64, offset: 7242},
							label: "limit",
							expr: &zeroOrOneExpr{
								pos: position{line: 281, col: 70, offset: 7248},
								expr: &ruleRefExpr{
									pos:  position{line: 281, col: 70, offset: 7248},
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
			pos:  position{line: 299, col: 1, offset: 7605},
			expr: &actionExpr{
				pos: position{line: 300, col: 5, offset: 7618},
				run: (*parser).callonasClause1,
				expr: &seqExpr{
					pos: position{line: 300, col: 5, offset: 7618},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 300, col: 5, offset: 7618},
							val:        "as",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 300, col: 11, offset: 7624},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 300, col: 13, offset: 7626},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 300, col: 15, offset: 7628},
								name: "fieldName",
							},
						},
					},
				},
			},
		},
		{
			name: "reducerExpr",
			pos:  position{line: 302, col: 1, offset: 7657},
			expr: &choiceExpr{
				pos: position{line: 303, col: 5, offset: 7673},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 303, col: 5, offset: 7673},
						run: (*parser).callonreducerExpr2,
						expr: &seqExpr{
							pos: position{line: 303, col: 5, offset: 7673},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 303, col: 5, offset: 7673},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 303, col: 11, offset: 7679},
										name: "fieldName",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 303, col: 21, offset: 7689},
									expr: &ruleRefExpr{
										pos:  position{line: 303, col: 21, offset: 7689},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 303, col: 24, offset: 7692},
									val:        "=",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 303, col: 28, offset: 7696},
									expr: &ruleRefExpr{
										pos:  position{line: 303, col: 28, offset: 7696},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 303, col: 31, offset: 7699},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 303, col: 33, offset: 7701},
										name: "reducer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 306, col: 5, offset: 7764},
						run: (*parser).callonreducerExpr13,
						expr: &seqExpr{
							pos: position{line: 306, col: 5, offset: 7764},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 306, col: 5, offset: 7764},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 306, col: 7, offset: 7766},
										name: "reducer",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 306, col: 15, offset: 7774},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 306, col: 17, offset: 7776},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 306, col: 23, offset: 7782},
										name: "asClause",
									},
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 309, col: 5, offset: 7846},
						name: "reducer",
					},
				},
			},
		},
		{
			name: "reducer",
			pos:  position{line: 311, col: 1, offset: 7855},
			expr: &choiceExpr{
				pos: position{line: 312, col: 5, offset: 7867},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 312, col: 5, offset: 7867},
						name: "countReducer",
					},
					&ruleRefExpr{
						pos:  position{line: 313, col: 5, offset: 7884},
						name: "fieldReducer",
					},
				},
			},
		},
		{
			name: "reducerList",
			pos:  position{line: 315, col: 1, offset: 7898},
			expr: &actionExpr{
				pos: position{line: 316, col: 5, offset: 7914},
				run: (*parser).callonreducerList1,
				expr: &seqExpr{
					pos: position{line: 316, col: 5, offset: 7914},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 316, col: 5, offset: 7914},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 316, col: 11, offset: 7920},
								name: "reducerExpr",
							},
						},
						&labeledExpr{
							pos:   position{line: 316, col: 23, offset: 7932},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 316, col: 28, offset: 7937},
								expr: &seqExpr{
									pos: position{line: 316, col: 29, offset: 7938},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 316, col: 29, offset: 7938},
											expr: &ruleRefExpr{
												pos:  position{line: 316, col: 29, offset: 7938},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 316, col: 32, offset: 7941},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 316, col: 36, offset: 7945},
											expr: &ruleRefExpr{
												pos:  position{line: 316, col: 36, offset: 7945},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 316, col: 39, offset: 7948},
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
			pos:  position{line: 324, col: 1, offset: 8145},
			expr: &choiceExpr{
				pos: position{line: 325, col: 5, offset: 8160},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 325, col: 5, offset: 8160},
						name: "sort",
					},
					&ruleRefExpr{
						pos:  position{line: 326, col: 5, offset: 8169},
						name: "top",
					},
					&ruleRefExpr{
						pos:  position{line: 327, col: 5, offset: 8177},
						name: "cut",
					},
					&ruleRefExpr{
						pos:  position{line: 328, col: 5, offset: 8185},
						name: "head",
					},
					&ruleRefExpr{
						pos:  position{line: 329, col: 5, offset: 8194},
						name: "tail",
					},
					&ruleRefExpr{
						pos:  position{line: 330, col: 5, offset: 8203},
						name: "filter",
					},
					&ruleRefExpr{
						pos:  position{line: 331, col: 5, offset: 8214},
						name: "uniq",
					},
				},
			},
		},
		{
			name: "sort",
			pos:  position{line: 333, col: 1, offset: 8220},
			expr: &choiceExpr{
				pos: position{line: 334, col: 5, offset: 8229},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 334, col: 5, offset: 8229},
						run: (*parser).callonsort2,
						expr: &seqExpr{
							pos: position{line: 334, col: 5, offset: 8229},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 334, col: 5, offset: 8229},
									val:        "sort",
									ignoreCase: true,
								},
								&labeledExpr{
									pos:   position{line: 334, col: 13, offset: 8237},
									label: "rev",
									expr: &zeroOrOneExpr{
										pos: position{line: 334, col: 17, offset: 8241},
										expr: &seqExpr{
											pos: position{line: 334, col: 18, offset: 8242},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 334, col: 18, offset: 8242},
													name: "_",
												},
												&litMatcher{
													pos:        position{line: 334, col: 20, offset: 8244},
													val:        "-r",
													ignoreCase: false,
												},
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 334, col: 27, offset: 8251},
									label: "limit",
									expr: &zeroOrOneExpr{
										pos: position{line: 334, col: 33, offset: 8257},
										expr: &ruleRefExpr{
											pos:  position{line: 334, col: 33, offset: 8257},
											name: "procLimitArg",
										},
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 334, col: 48, offset: 8272},
									expr: &ruleRefExpr{
										pos:  position{line: 334, col: 48, offset: 8272},
										name: "_",
									},
								},
								&notExpr{
									pos: position{line: 334, col: 51, offset: 8275},
									expr: &litMatcher{
										pos:        position{line: 334, col: 52, offset: 8276},
										val:        "-r",
										ignoreCase: false,
									},
								},
								&labeledExpr{
									pos:   position{line: 334, col: 57, offset: 8281},
									label: "list",
									expr: &zeroOrOneExpr{
										pos: position{line: 334, col: 62, offset: 8286},
										expr: &ruleRefExpr{
											pos:  position{line: 334, col: 63, offset: 8287},
											name: "fieldExprList",
										},
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 339, col: 5, offset: 8417},
						run: (*parser).callonsort20,
						expr: &seqExpr{
							pos: position{line: 339, col: 5, offset: 8417},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 339, col: 5, offset: 8417},
									val:        "sort",
									ignoreCase: true,
								},
								&labeledExpr{
									pos:   position{line: 339, col: 13, offset: 8425},
									label: "limit",
									expr: &zeroOrOneExpr{
										pos: position{line: 339, col: 19, offset: 8431},
										expr: &ruleRefExpr{
											pos:  position{line: 339, col: 19, offset: 8431},
											name: "procLimitArg",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 339, col: 33, offset: 8445},
									label: "rev",
									expr: &zeroOrOneExpr{
										pos: position{line: 339, col: 37, offset: 8449},
										expr: &seqExpr{
											pos: position{line: 339, col: 38, offset: 8450},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 339, col: 38, offset: 8450},
													name: "_",
												},
												&litMatcher{
													pos:        position{line: 339, col: 40, offset: 8452},
													val:        "-r",
													ignoreCase: false,
												},
											},
										},
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 339, col: 47, offset: 8459},
									expr: &ruleRefExpr{
										pos:  position{line: 339, col: 47, offset: 8459},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 339, col: 50, offset: 8462},
									label: "list",
									expr: &zeroOrOneExpr{
										pos: position{line: 339, col: 55, offset: 8467},
										expr: &ruleRefExpr{
											pos:  position{line: 339, col: 56, offset: 8468},
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
			pos:  position{line: 345, col: 1, offset: 8595},
			expr: &actionExpr{
				pos: position{line: 346, col: 5, offset: 8603},
				run: (*parser).callontop1,
				expr: &seqExpr{
					pos: position{line: 346, col: 5, offset: 8603},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 346, col: 5, offset: 8603},
							val:        "top",
							ignoreCase: true,
						},
						&labeledExpr{
							pos:   position{line: 346, col: 12, offset: 8610},
							label: "limit",
							expr: &zeroOrOneExpr{
								pos: position{line: 346, col: 18, offset: 8616},
								expr: &ruleRefExpr{
									pos:  position{line: 346, col: 18, offset: 8616},
									name: "procLimitArg",
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 346, col: 32, offset: 8630},
							label: "flush",
							expr: &zeroOrOneExpr{
								pos: position{line: 346, col: 38, offset: 8636},
								expr: &seqExpr{
									pos: position{line: 346, col: 39, offset: 8637},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 346, col: 39, offset: 8637},
											name: "_",
										},
										&litMatcher{
											pos:        position{line: 346, col: 41, offset: 8639},
											val:        "-flush",
											ignoreCase: false,
										},
									},
								},
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 346, col: 52, offset: 8650},
							expr: &ruleRefExpr{
								pos:  position{line: 346, col: 52, offset: 8650},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 346, col: 55, offset: 8653},
							label: "list",
							expr: &zeroOrOneExpr{
								pos: position{line: 346, col: 60, offset: 8658},
								expr: &ruleRefExpr{
									pos:  position{line: 346, col: 61, offset: 8659},
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
			pos:  position{line: 350, col: 1, offset: 8730},
			expr: &actionExpr{
				pos: position{line: 351, col: 5, offset: 8747},
				run: (*parser).callonprocLimitArg1,
				expr: &seqExpr{
					pos: position{line: 351, col: 5, offset: 8747},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 351, col: 5, offset: 8747},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 351, col: 7, offset: 8749},
							val:        "-limit",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 351, col: 16, offset: 8758},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 351, col: 18, offset: 8760},
							label: "limit",
							expr: &ruleRefExpr{
								pos:  position{line: 351, col: 24, offset: 8766},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "cut",
			pos:  position{line: 353, col: 1, offset: 8797},
			expr: &actionExpr{
				pos: position{line: 354, col: 5, offset: 8805},
				run: (*parser).calloncut1,
				expr: &seqExpr{
					pos: position{line: 354, col: 5, offset: 8805},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 354, col: 5, offset: 8805},
							val:        "cut",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 354, col: 12, offset: 8812},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 354, col: 14, offset: 8814},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 354, col: 19, offset: 8819},
								name: "fieldRefDotOnlyList",
							},
						},
					},
				},
			},
		},
		{
			name: "head",
			pos:  position{line: 355, col: 1, offset: 8873},
			expr: &choiceExpr{
				pos: position{line: 356, col: 5, offset: 8882},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 356, col: 5, offset: 8882},
						run: (*parser).callonhead2,
						expr: &seqExpr{
							pos: position{line: 356, col: 5, offset: 8882},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 356, col: 5, offset: 8882},
									val:        "head",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 356, col: 13, offset: 8890},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 356, col: 15, offset: 8892},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 356, col: 21, offset: 8898},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 357, col: 5, offset: 8946},
						run: (*parser).callonhead8,
						expr: &litMatcher{
							pos:        position{line: 357, col: 5, offset: 8946},
							val:        "head",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "tail",
			pos:  position{line: 358, col: 1, offset: 8986},
			expr: &choiceExpr{
				pos: position{line: 359, col: 5, offset: 8995},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 359, col: 5, offset: 8995},
						run: (*parser).callontail2,
						expr: &seqExpr{
							pos: position{line: 359, col: 5, offset: 8995},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 359, col: 5, offset: 8995},
									val:        "tail",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 359, col: 13, offset: 9003},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 359, col: 15, offset: 9005},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 359, col: 21, offset: 9011},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 360, col: 5, offset: 9059},
						run: (*parser).callontail8,
						expr: &litMatcher{
							pos:        position{line: 360, col: 5, offset: 9059},
							val:        "tail",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "filter",
			pos:  position{line: 362, col: 1, offset: 9100},
			expr: &actionExpr{
				pos: position{line: 363, col: 5, offset: 9111},
				run: (*parser).callonfilter1,
				expr: &seqExpr{
					pos: position{line: 363, col: 5, offset: 9111},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 363, col: 5, offset: 9111},
							val:        "filter",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 363, col: 15, offset: 9121},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 363, col: 17, offset: 9123},
							label: "expr",
							expr: &ruleRefExpr{
								pos:  position{line: 363, col: 22, offset: 9128},
								name: "searchExpr",
							},
						},
					},
				},
			},
		},
		{
			name: "uniq",
			pos:  position{line: 366, col: 1, offset: 9186},
			expr: &choiceExpr{
				pos: position{line: 367, col: 5, offset: 9195},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 367, col: 5, offset: 9195},
						run: (*parser).callonuniq2,
						expr: &seqExpr{
							pos: position{line: 367, col: 5, offset: 9195},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 367, col: 5, offset: 9195},
									val:        "uniq",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 367, col: 13, offset: 9203},
									name: "_",
								},
								&litMatcher{
									pos:        position{line: 367, col: 15, offset: 9205},
									val:        "-c",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 370, col: 5, offset: 9259},
						run: (*parser).callonuniq7,
						expr: &litMatcher{
							pos:        position{line: 370, col: 5, offset: 9259},
							val:        "uniq",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "duration",
			pos:  position{line: 374, col: 1, offset: 9314},
			expr: &choiceExpr{
				pos: position{line: 375, col: 5, offset: 9327},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 375, col: 5, offset: 9327},
						name: "seconds",
					},
					&ruleRefExpr{
						pos:  position{line: 376, col: 5, offset: 9339},
						name: "minutes",
					},
					&ruleRefExpr{
						pos:  position{line: 377, col: 5, offset: 9351},
						name: "hours",
					},
					&seqExpr{
						pos: position{line: 378, col: 5, offset: 9361},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 378, col: 5, offset: 9361},
								name: "hours",
							},
							&ruleRefExpr{
								pos:  position{line: 378, col: 11, offset: 9367},
								name: "_",
							},
							&litMatcher{
								pos:        position{line: 378, col: 13, offset: 9369},
								val:        "and",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 378, col: 19, offset: 9375},
								name: "_",
							},
							&ruleRefExpr{
								pos:  position{line: 378, col: 21, offset: 9377},
								name: "minutes",
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 379, col: 5, offset: 9389},
						name: "days",
					},
					&ruleRefExpr{
						pos:  position{line: 380, col: 5, offset: 9398},
						name: "weeks",
					},
				},
			},
		},
		{
			name: "sec_abbrev",
			pos:  position{line: 382, col: 1, offset: 9405},
			expr: &choiceExpr{
				pos: position{line: 383, col: 5, offset: 9420},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 383, col: 5, offset: 9420},
						val:        "seconds",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 384, col: 5, offset: 9434},
						val:        "second",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 385, col: 5, offset: 9447},
						val:        "secs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 386, col: 5, offset: 9458},
						val:        "sec",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 387, col: 5, offset: 9468},
						val:        "s",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "min_abbrev",
			pos:  position{line: 389, col: 1, offset: 9473},
			expr: &choiceExpr{
				pos: position{line: 390, col: 5, offset: 9488},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 390, col: 5, offset: 9488},
						val:        "minutes",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 391, col: 5, offset: 9502},
						val:        "minute",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 392, col: 5, offset: 9515},
						val:        "mins",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 393, col: 5, offset: 9526},
						val:        "min",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 394, col: 5, offset: 9536},
						val:        "m",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "hour_abbrev",
			pos:  position{line: 396, col: 1, offset: 9541},
			expr: &choiceExpr{
				pos: position{line: 397, col: 5, offset: 9557},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 397, col: 5, offset: 9557},
						val:        "hours",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 398, col: 5, offset: 9569},
						val:        "hrs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 399, col: 5, offset: 9579},
						val:        "hr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 400, col: 5, offset: 9588},
						val:        "h",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 401, col: 5, offset: 9596},
						val:        "hour",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "day_abbrev",
			pos:  position{line: 403, col: 1, offset: 9604},
			expr: &choiceExpr{
				pos: position{line: 403, col: 14, offset: 9617},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 403, col: 14, offset: 9617},
						val:        "days",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 403, col: 21, offset: 9624},
						val:        "day",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 403, col: 27, offset: 9630},
						val:        "d",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "week_abbrev",
			pos:  position{line: 404, col: 1, offset: 9634},
			expr: &choiceExpr{
				pos: position{line: 404, col: 15, offset: 9648},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 404, col: 15, offset: 9648},
						val:        "weeks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 404, col: 23, offset: 9656},
						val:        "week",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 404, col: 30, offset: 9663},
						val:        "wks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 404, col: 36, offset: 9669},
						val:        "wk",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 404, col: 41, offset: 9674},
						val:        "w",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "seconds",
			pos:  position{line: 406, col: 1, offset: 9679},
			expr: &choiceExpr{
				pos: position{line: 407, col: 5, offset: 9691},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 407, col: 5, offset: 9691},
						run: (*parser).callonseconds2,
						expr: &litMatcher{
							pos:        position{line: 407, col: 5, offset: 9691},
							val:        "second",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 408, col: 5, offset: 9736},
						run: (*parser).callonseconds4,
						expr: &seqExpr{
							pos: position{line: 408, col: 5, offset: 9736},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 408, col: 5, offset: 9736},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 408, col: 9, offset: 9740},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 408, col: 16, offset: 9747},
									expr: &ruleRefExpr{
										pos:  position{line: 408, col: 16, offset: 9747},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 408, col: 19, offset: 9750},
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
			pos:  position{line: 410, col: 1, offset: 9796},
			expr: &choiceExpr{
				pos: position{line: 411, col: 5, offset: 9808},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 411, col: 5, offset: 9808},
						run: (*parser).callonminutes2,
						expr: &litMatcher{
							pos:        position{line: 411, col: 5, offset: 9808},
							val:        "minute",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 412, col: 5, offset: 9854},
						run: (*parser).callonminutes4,
						expr: &seqExpr{
							pos: position{line: 412, col: 5, offset: 9854},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 412, col: 5, offset: 9854},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 412, col: 9, offset: 9858},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 412, col: 16, offset: 9865},
									expr: &ruleRefExpr{
										pos:  position{line: 412, col: 16, offset: 9865},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 412, col: 19, offset: 9868},
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
			pos:  position{line: 414, col: 1, offset: 9923},
			expr: &choiceExpr{
				pos: position{line: 415, col: 5, offset: 9933},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 415, col: 5, offset: 9933},
						run: (*parser).callonhours2,
						expr: &litMatcher{
							pos:        position{line: 415, col: 5, offset: 9933},
							val:        "hour",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 416, col: 5, offset: 9979},
						run: (*parser).callonhours4,
						expr: &seqExpr{
							pos: position{line: 416, col: 5, offset: 9979},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 416, col: 5, offset: 9979},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 416, col: 9, offset: 9983},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 416, col: 16, offset: 9990},
									expr: &ruleRefExpr{
										pos:  position{line: 416, col: 16, offset: 9990},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 416, col: 19, offset: 9993},
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
			pos:  position{line: 418, col: 1, offset: 10051},
			expr: &choiceExpr{
				pos: position{line: 419, col: 5, offset: 10060},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 419, col: 5, offset: 10060},
						run: (*parser).callondays2,
						expr: &litMatcher{
							pos:        position{line: 419, col: 5, offset: 10060},
							val:        "day",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 420, col: 5, offset: 10108},
						run: (*parser).callondays4,
						expr: &seqExpr{
							pos: position{line: 420, col: 5, offset: 10108},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 420, col: 5, offset: 10108},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 420, col: 9, offset: 10112},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 420, col: 16, offset: 10119},
									expr: &ruleRefExpr{
										pos:  position{line: 420, col: 16, offset: 10119},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 420, col: 19, offset: 10122},
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
			pos:  position{line: 422, col: 1, offset: 10182},
			expr: &actionExpr{
				pos: position{line: 423, col: 5, offset: 10192},
				run: (*parser).callonweeks1,
				expr: &seqExpr{
					pos: position{line: 423, col: 5, offset: 10192},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 423, col: 5, offset: 10192},
							label: "num",
							expr: &ruleRefExpr{
								pos:  position{line: 423, col: 9, offset: 10196},
								name: "number",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 423, col: 16, offset: 10203},
							expr: &ruleRefExpr{
								pos:  position{line: 423, col: 16, offset: 10203},
								name: "_",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 423, col: 19, offset: 10206},
							name: "week_abbrev",
						},
					},
				},
			},
		},
		{
			name: "number",
			pos:  position{line: 425, col: 1, offset: 10269},
			expr: &ruleRefExpr{
				pos:  position{line: 425, col: 10, offset: 10278},
				name: "integer",
			},
		},
		{
			name: "addr",
			pos:  position{line: 429, col: 1, offset: 10316},
			expr: &actionExpr{
				pos: position{line: 430, col: 5, offset: 10325},
				run: (*parser).callonaddr1,
				expr: &labeledExpr{
					pos:   position{line: 430, col: 5, offset: 10325},
					label: "a",
					expr: &seqExpr{
						pos: position{line: 430, col: 8, offset: 10328},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 430, col: 8, offset: 10328},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 430, col: 16, offset: 10336},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 430, col: 20, offset: 10340},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 430, col: 28, offset: 10348},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 430, col: 32, offset: 10352},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 430, col: 40, offset: 10360},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 430, col: 44, offset: 10364},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "port",
			pos:  position{line: 432, col: 1, offset: 10405},
			expr: &actionExpr{
				pos: position{line: 433, col: 5, offset: 10414},
				run: (*parser).callonport1,
				expr: &seqExpr{
					pos: position{line: 433, col: 5, offset: 10414},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 433, col: 5, offset: 10414},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 433, col: 9, offset: 10418},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 433, col: 11, offset: 10420},
								name: "sinteger",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6addr",
			pos:  position{line: 437, col: 1, offset: 10579},
			expr: &choiceExpr{
				pos: position{line: 438, col: 5, offset: 10591},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 438, col: 5, offset: 10591},
						run: (*parser).callonip6addr2,
						expr: &seqExpr{
							pos: position{line: 438, col: 5, offset: 10591},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 438, col: 5, offset: 10591},
									label: "a",
									expr: &oneOrMoreExpr{
										pos: position{line: 438, col: 7, offset: 10593},
										expr: &ruleRefExpr{
											pos:  position{line: 438, col: 8, offset: 10594},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 438, col: 20, offset: 10606},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 438, col: 22, offset: 10608},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 441, col: 5, offset: 10672},
						run: (*parser).callonip6addr9,
						expr: &seqExpr{
							pos: position{line: 441, col: 5, offset: 10672},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 441, col: 5, offset: 10672},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 441, col: 7, offset: 10674},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 441, col: 11, offset: 10678},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 441, col: 13, offset: 10680},
										expr: &ruleRefExpr{
											pos:  position{line: 441, col: 14, offset: 10681},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 441, col: 25, offset: 10692},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 441, col: 30, offset: 10697},
									label: "d",
									expr: &zeroOrMoreExpr{
										pos: position{line: 441, col: 32, offset: 10699},
										expr: &ruleRefExpr{
											pos:  position{line: 441, col: 33, offset: 10700},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 441, col: 45, offset: 10712},
									label: "e",
									expr: &ruleRefExpr{
										pos:  position{line: 441, col: 47, offset: 10714},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 444, col: 5, offset: 10813},
						run: (*parser).callonip6addr22,
						expr: &seqExpr{
							pos: position{line: 444, col: 5, offset: 10813},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 444, col: 5, offset: 10813},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 444, col: 10, offset: 10818},
									label: "a",
									expr: &zeroOrMoreExpr{
										pos: position{line: 444, col: 12, offset: 10820},
										expr: &ruleRefExpr{
											pos:  position{line: 444, col: 13, offset: 10821},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 444, col: 25, offset: 10833},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 444, col: 27, offset: 10835},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 447, col: 5, offset: 10906},
						run: (*parser).callonip6addr30,
						expr: &seqExpr{
							pos: position{line: 447, col: 5, offset: 10906},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 447, col: 5, offset: 10906},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 447, col: 7, offset: 10908},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 447, col: 11, offset: 10912},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 447, col: 13, offset: 10914},
										expr: &ruleRefExpr{
											pos:  position{line: 447, col: 14, offset: 10915},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 447, col: 25, offset: 10926},
									val:        "::",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 450, col: 5, offset: 10994},
						run: (*parser).callonip6addr38,
						expr: &litMatcher{
							pos:        position{line: 450, col: 5, offset: 10994},
							val:        "::",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "ip6tail",
			pos:  position{line: 454, col: 1, offset: 11031},
			expr: &choiceExpr{
				pos: position{line: 455, col: 5, offset: 11043},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 455, col: 5, offset: 11043},
						name: "addr",
					},
					&ruleRefExpr{
						pos:  position{line: 456, col: 5, offset: 11052},
						name: "h16",
					},
				},
			},
		},
		{
			name: "h_append",
			pos:  position{line: 458, col: 1, offset: 11057},
			expr: &actionExpr{
				pos: position{line: 458, col: 12, offset: 11068},
				run: (*parser).callonh_append1,
				expr: &seqExpr{
					pos: position{line: 458, col: 12, offset: 11068},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 458, col: 12, offset: 11068},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 458, col: 16, offset: 11072},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 458, col: 18, offset: 11074},
								name: "h16",
							},
						},
					},
				},
			},
		},
		{
			name: "h_prepend",
			pos:  position{line: 459, col: 1, offset: 11111},
			expr: &actionExpr{
				pos: position{line: 459, col: 13, offset: 11123},
				run: (*parser).callonh_prepend1,
				expr: &seqExpr{
					pos: position{line: 459, col: 13, offset: 11123},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 459, col: 13, offset: 11123},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 459, col: 15, offset: 11125},
								name: "h16",
							},
						},
						&litMatcher{
							pos:        position{line: 459, col: 19, offset: 11129},
							val:        ":",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "sub_addr",
			pos:  position{line: 461, col: 1, offset: 11167},
			expr: &choiceExpr{
				pos: position{line: 462, col: 5, offset: 11180},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 462, col: 5, offset: 11180},
						name: "addr",
					},
					&actionExpr{
						pos: position{line: 463, col: 5, offset: 11189},
						run: (*parser).callonsub_addr3,
						expr: &labeledExpr{
							pos:   position{line: 463, col: 5, offset: 11189},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 463, col: 8, offset: 11192},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 463, col: 8, offset: 11192},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 463, col: 16, offset: 11200},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 463, col: 20, offset: 11204},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 463, col: 28, offset: 11212},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 463, col: 32, offset: 11216},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 464, col: 5, offset: 11268},
						run: (*parser).callonsub_addr11,
						expr: &labeledExpr{
							pos:   position{line: 464, col: 5, offset: 11268},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 464, col: 8, offset: 11271},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 464, col: 8, offset: 11271},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 464, col: 16, offset: 11279},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 464, col: 20, offset: 11283},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 465, col: 5, offset: 11337},
						run: (*parser).callonsub_addr17,
						expr: &labeledExpr{
							pos:   position{line: 465, col: 5, offset: 11337},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 465, col: 7, offset: 11339},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "subnet",
			pos:  position{line: 467, col: 1, offset: 11390},
			expr: &actionExpr{
				pos: position{line: 468, col: 5, offset: 11401},
				run: (*parser).callonsubnet1,
				expr: &seqExpr{
					pos: position{line: 468, col: 5, offset: 11401},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 468, col: 5, offset: 11401},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 468, col: 7, offset: 11403},
								name: "sub_addr",
							},
						},
						&litMatcher{
							pos:        position{line: 468, col: 16, offset: 11412},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 468, col: 20, offset: 11416},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 468, col: 22, offset: 11418},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6subnet",
			pos:  position{line: 472, col: 1, offset: 11494},
			expr: &actionExpr{
				pos: position{line: 473, col: 5, offset: 11508},
				run: (*parser).callonip6subnet1,
				expr: &seqExpr{
					pos: position{line: 473, col: 5, offset: 11508},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 473, col: 5, offset: 11508},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 473, col: 7, offset: 11510},
								name: "ip6addr",
							},
						},
						&litMatcher{
							pos:        position{line: 473, col: 15, offset: 11518},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 473, col: 19, offset: 11522},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 473, col: 21, offset: 11524},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "integer",
			pos:  position{line: 477, col: 1, offset: 11590},
			expr: &actionExpr{
				pos: position{line: 478, col: 5, offset: 11602},
				run: (*parser).calloninteger1,
				expr: &labeledExpr{
					pos:   position{line: 478, col: 5, offset: 11602},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 478, col: 7, offset: 11604},
						name: "sinteger",
					},
				},
			},
		},
		{
			name: "sinteger",
			pos:  position{line: 482, col: 1, offset: 11648},
			expr: &actionExpr{
				pos: position{line: 483, col: 5, offset: 11661},
				run: (*parser).callonsinteger1,
				expr: &labeledExpr{
					pos:   position{line: 483, col: 5, offset: 11661},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 483, col: 11, offset: 11667},
						expr: &charClassMatcher{
							pos:        position{line: 483, col: 11, offset: 11667},
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
			pos:  position{line: 487, col: 1, offset: 11712},
			expr: &actionExpr{
				pos: position{line: 488, col: 5, offset: 11723},
				run: (*parser).callondouble1,
				expr: &labeledExpr{
					pos:   position{line: 488, col: 5, offset: 11723},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 488, col: 7, offset: 11725},
						name: "sdouble",
					},
				},
			},
		},
		{
			name: "sdouble",
			pos:  position{line: 492, col: 1, offset: 11772},
			expr: &choiceExpr{
				pos: position{line: 493, col: 5, offset: 11784},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 493, col: 5, offset: 11784},
						run: (*parser).callonsdouble2,
						expr: &seqExpr{
							pos: position{line: 493, col: 5, offset: 11784},
							exprs: []interface{}{
								&oneOrMoreExpr{
									pos: position{line: 493, col: 5, offset: 11784},
									expr: &ruleRefExpr{
										pos:  position{line: 493, col: 5, offset: 11784},
										name: "doubleInteger",
									},
								},
								&litMatcher{
									pos:        position{line: 493, col: 20, offset: 11799},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 493, col: 24, offset: 11803},
									expr: &ruleRefExpr{
										pos:  position{line: 493, col: 24, offset: 11803},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 493, col: 37, offset: 11816},
									expr: &ruleRefExpr{
										pos:  position{line: 493, col: 37, offset: 11816},
										name: "exponentPart",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 496, col: 5, offset: 11875},
						run: (*parser).callonsdouble11,
						expr: &seqExpr{
							pos: position{line: 496, col: 5, offset: 11875},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 496, col: 5, offset: 11875},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 496, col: 9, offset: 11879},
									expr: &ruleRefExpr{
										pos:  position{line: 496, col: 9, offset: 11879},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 496, col: 22, offset: 11892},
									expr: &ruleRefExpr{
										pos:  position{line: 496, col: 22, offset: 11892},
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
			pos:  position{line: 500, col: 1, offset: 11948},
			expr: &choiceExpr{
				pos: position{line: 501, col: 5, offset: 11966},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 501, col: 5, offset: 11966},
						val:        "0",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 502, col: 5, offset: 11974},
						exprs: []interface{}{
							&charClassMatcher{
								pos:        position{line: 502, col: 5, offset: 11974},
								val:        "[1-9]",
								ranges:     []rune{'1', '9'},
								ignoreCase: false,
								inverted:   false,
							},
							&zeroOrMoreExpr{
								pos: position{line: 502, col: 11, offset: 11980},
								expr: &charClassMatcher{
									pos:        position{line: 502, col: 11, offset: 11980},
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
			pos:  position{line: 504, col: 1, offset: 11988},
			expr: &charClassMatcher{
				pos:        position{line: 504, col: 15, offset: 12002},
				val:        "[0-9]",
				ranges:     []rune{'0', '9'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "signedInteger",
			pos:  position{line: 506, col: 1, offset: 12009},
			expr: &seqExpr{
				pos: position{line: 506, col: 17, offset: 12025},
				exprs: []interface{}{
					&zeroOrOneExpr{
						pos: position{line: 506, col: 17, offset: 12025},
						expr: &charClassMatcher{
							pos:        position{line: 506, col: 17, offset: 12025},
							val:        "[+-]",
							chars:      []rune{'+', '-'},
							ignoreCase: false,
							inverted:   false,
						},
					},
					&oneOrMoreExpr{
						pos: position{line: 506, col: 23, offset: 12031},
						expr: &ruleRefExpr{
							pos:  position{line: 506, col: 23, offset: 12031},
							name: "doubleDigit",
						},
					},
				},
			},
		},
		{
			name: "exponentPart",
			pos:  position{line: 508, col: 1, offset: 12045},
			expr: &seqExpr{
				pos: position{line: 508, col: 16, offset: 12060},
				exprs: []interface{}{
					&litMatcher{
						pos:        position{line: 508, col: 16, offset: 12060},
						val:        "e",
						ignoreCase: true,
					},
					&ruleRefExpr{
						pos:  position{line: 508, col: 21, offset: 12065},
						name: "signedInteger",
					},
				},
			},
		},
		{
			name: "h16",
			pos:  position{line: 510, col: 1, offset: 12080},
			expr: &actionExpr{
				pos: position{line: 510, col: 7, offset: 12086},
				run: (*parser).callonh161,
				expr: &labeledExpr{
					pos:   position{line: 510, col: 7, offset: 12086},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 510, col: 13, offset: 12092},
						expr: &ruleRefExpr{
							pos:  position{line: 510, col: 13, offset: 12092},
							name: "hexdigit",
						},
					},
				},
			},
		},
		{
			name: "hexdigit",
			pos:  position{line: 512, col: 1, offset: 12134},
			expr: &charClassMatcher{
				pos:        position{line: 512, col: 12, offset: 12145},
				val:        "[0-9a-fA-F]",
				ranges:     []rune{'0', '9', 'a', 'f', 'A', 'F'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name:        "boomWord",
			displayName: "\"boomWord\"",
			pos:         position{line: 514, col: 1, offset: 12158},
			expr: &actionExpr{
				pos: position{line: 514, col: 23, offset: 12180},
				run: (*parser).callonboomWord1,
				expr: &labeledExpr{
					pos:   position{line: 514, col: 23, offset: 12180},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 514, col: 29, offset: 12186},
						expr: &ruleRefExpr{
							pos:  position{line: 514, col: 29, offset: 12186},
							name: "boomWordPart",
						},
					},
				},
			},
		},
		{
			name: "boomWordPart",
			pos:  position{line: 516, col: 1, offset: 12234},
			expr: &choiceExpr{
				pos: position{line: 517, col: 5, offset: 12251},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 517, col: 5, offset: 12251},
						run: (*parser).callonboomWordPart2,
						expr: &seqExpr{
							pos: position{line: 517, col: 5, offset: 12251},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 517, col: 5, offset: 12251},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 517, col: 10, offset: 12256},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 517, col: 12, offset: 12258},
										name: "escapeSequence",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 518, col: 5, offset: 12296},
						run: (*parser).callonboomWordPart7,
						expr: &seqExpr{
							pos: position{line: 518, col: 5, offset: 12296},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 518, col: 5, offset: 12296},
									expr: &choiceExpr{
										pos: position{line: 518, col: 7, offset: 12298},
										alternatives: []interface{}{
											&charClassMatcher{
												pos:        position{line: 518, col: 7, offset: 12298},
												val:        "[\\x00-\\x1F\\x5C(),!><=\\x22|\\x27;]",
												chars:      []rune{'\\', '(', ')', ',', '!', '>', '<', '=', '"', '|', '\'', ';'},
												ranges:     []rune{'\x00', '\x1f'},
												ignoreCase: false,
												inverted:   false,
											},
											&ruleRefExpr{
												pos:  position{line: 518, col: 42, offset: 12333},
												name: "ws",
											},
										},
									},
								},
								&anyMatcher{
									line: 518, col: 46, offset: 12337,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "quotedString",
			pos:  position{line: 520, col: 1, offset: 12371},
			expr: &choiceExpr{
				pos: position{line: 521, col: 5, offset: 12388},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 521, col: 5, offset: 12388},
						run: (*parser).callonquotedString2,
						expr: &seqExpr{
							pos: position{line: 521, col: 5, offset: 12388},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 521, col: 5, offset: 12388},
									val:        "\"",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 521, col: 9, offset: 12392},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 521, col: 11, offset: 12394},
										expr: &ruleRefExpr{
											pos:  position{line: 521, col: 11, offset: 12394},
											name: "doubleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 521, col: 29, offset: 12412},
									val:        "\"",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 522, col: 5, offset: 12449},
						run: (*parser).callonquotedString9,
						expr: &seqExpr{
							pos: position{line: 522, col: 5, offset: 12449},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 522, col: 5, offset: 12449},
									val:        "'",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 522, col: 9, offset: 12453},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 522, col: 11, offset: 12455},
										expr: &ruleRefExpr{
											pos:  position{line: 522, col: 11, offset: 12455},
											name: "singleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 522, col: 29, offset: 12473},
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
			pos:  position{line: 524, col: 1, offset: 12507},
			expr: &choiceExpr{
				pos: position{line: 525, col: 5, offset: 12528},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 525, col: 5, offset: 12528},
						run: (*parser).callondoubleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 525, col: 5, offset: 12528},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 525, col: 5, offset: 12528},
									expr: &choiceExpr{
										pos: position{line: 525, col: 7, offset: 12530},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 525, col: 7, offset: 12530},
												val:        "\"",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 525, col: 13, offset: 12536},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 525, col: 26, offset: 12549,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 526, col: 5, offset: 12586},
						run: (*parser).callondoubleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 526, col: 5, offset: 12586},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 526, col: 5, offset: 12586},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 526, col: 10, offset: 12591},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 526, col: 12, offset: 12593},
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
			pos:  position{line: 528, col: 1, offset: 12627},
			expr: &choiceExpr{
				pos: position{line: 529, col: 5, offset: 12648},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 529, col: 5, offset: 12648},
						run: (*parser).callonsingleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 529, col: 5, offset: 12648},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 529, col: 5, offset: 12648},
									expr: &choiceExpr{
										pos: position{line: 529, col: 7, offset: 12650},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 529, col: 7, offset: 12650},
												val:        "'",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 529, col: 13, offset: 12656},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 529, col: 26, offset: 12669,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 530, col: 5, offset: 12706},
						run: (*parser).callonsingleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 530, col: 5, offset: 12706},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 530, col: 5, offset: 12706},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 530, col: 10, offset: 12711},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 530, col: 12, offset: 12713},
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
			pos:  position{line: 532, col: 1, offset: 12747},
			expr: &choiceExpr{
				pos: position{line: 533, col: 5, offset: 12766},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 533, col: 5, offset: 12766},
						run: (*parser).callonescapeSequence2,
						expr: &seqExpr{
							pos: position{line: 533, col: 5, offset: 12766},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 533, col: 5, offset: 12766},
									val:        "x",
									ignoreCase: false,
								},
								&ruleRefExpr{
									pos:  position{line: 533, col: 9, offset: 12770},
									name: "hexdigit",
								},
								&ruleRefExpr{
									pos:  position{line: 533, col: 18, offset: 12779},
									name: "hexdigit",
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 534, col: 5, offset: 12830},
						name: "singleCharEscape",
					},
					&ruleRefExpr{
						pos:  position{line: 535, col: 5, offset: 12851},
						name: "unicodeEscape",
					},
				},
			},
		},
		{
			name: "singleCharEscape",
			pos:  position{line: 537, col: 1, offset: 12866},
			expr: &choiceExpr{
				pos: position{line: 538, col: 5, offset: 12887},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 538, col: 5, offset: 12887},
						val:        "'",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 539, col: 5, offset: 12895},
						val:        "\"",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 540, col: 5, offset: 12903},
						val:        "\\",
						ignoreCase: false,
					},
					&actionExpr{
						pos: position{line: 541, col: 5, offset: 12912},
						run: (*parser).callonsingleCharEscape5,
						expr: &litMatcher{
							pos:        position{line: 541, col: 5, offset: 12912},
							val:        "b",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 542, col: 5, offset: 12941},
						run: (*parser).callonsingleCharEscape7,
						expr: &litMatcher{
							pos:        position{line: 542, col: 5, offset: 12941},
							val:        "f",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 543, col: 5, offset: 12970},
						run: (*parser).callonsingleCharEscape9,
						expr: &litMatcher{
							pos:        position{line: 543, col: 5, offset: 12970},
							val:        "n",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 544, col: 5, offset: 12999},
						run: (*parser).callonsingleCharEscape11,
						expr: &litMatcher{
							pos:        position{line: 544, col: 5, offset: 12999},
							val:        "r",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 545, col: 5, offset: 13028},
						run: (*parser).callonsingleCharEscape13,
						expr: &litMatcher{
							pos:        position{line: 545, col: 5, offset: 13028},
							val:        "t",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 546, col: 5, offset: 13057},
						run: (*parser).callonsingleCharEscape15,
						expr: &litMatcher{
							pos:        position{line: 546, col: 5, offset: 13057},
							val:        "v",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "unicodeEscape",
			pos:  position{line: 548, col: 1, offset: 13083},
			expr: &seqExpr{
				pos: position{line: 549, col: 5, offset: 13101},
				exprs: []interface{}{
					&litMatcher{
						pos:        position{line: 549, col: 5, offset: 13101},
						val:        "u",
						ignoreCase: false,
					},
					&ruleRefExpr{
						pos:  position{line: 549, col: 9, offset: 13105},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 549, col: 18, offset: 13114},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 549, col: 27, offset: 13123},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 549, col: 36, offset: 13132},
						name: "hexdigit",
					},
				},
			},
		},
		{
			name: "reString",
			pos:  position{line: 551, col: 1, offset: 13142},
			expr: &actionExpr{
				pos: position{line: 552, col: 5, offset: 13155},
				run: (*parser).callonreString1,
				expr: &seqExpr{
					pos: position{line: 552, col: 5, offset: 13155},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 552, col: 5, offset: 13155},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 552, col: 9, offset: 13159},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 552, col: 11, offset: 13161},
								name: "reBody",
							},
						},
						&litMatcher{
							pos:        position{line: 552, col: 18, offset: 13168},
							val:        "/",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reBody",
			pos:  position{line: 554, col: 1, offset: 13191},
			expr: &actionExpr{
				pos: position{line: 555, col: 5, offset: 13202},
				run: (*parser).callonreBody1,
				expr: &oneOrMoreExpr{
					pos: position{line: 555, col: 5, offset: 13202},
					expr: &choiceExpr{
						pos: position{line: 555, col: 6, offset: 13203},
						alternatives: []interface{}{
							&charClassMatcher{
								pos:        position{line: 555, col: 6, offset: 13203},
								val:        "[^/\\\\]",
								chars:      []rune{'/', '\\'},
								ignoreCase: false,
								inverted:   true,
							},
							&litMatcher{
								pos:        position{line: 555, col: 13, offset: 13210},
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
			pos:  position{line: 557, col: 1, offset: 13250},
			expr: &charClassMatcher{
				pos:        position{line: 558, col: 5, offset: 13266},
				val:        "[\\x00-\\x1f\\\\]",
				chars:      []rune{'\\'},
				ranges:     []rune{'\x00', '\x1f'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "ws",
			pos:  position{line: 560, col: 1, offset: 13281},
			expr: &choiceExpr{
				pos: position{line: 561, col: 5, offset: 13288},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 561, col: 5, offset: 13288},
						val:        "\t",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 562, col: 5, offset: 13297},
						val:        "\v",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 563, col: 5, offset: 13306},
						val:        "\f",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 564, col: 5, offset: 13315},
						val:        " ",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 565, col: 5, offset: 13323},
						val:        "\u00a0",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 566, col: 5, offset: 13336},
						val:        "\ufeff",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name:        "_",
			displayName: "\"whitespace\"",
			pos:         position{line: 568, col: 1, offset: 13346},
			expr: &oneOrMoreExpr{
				pos: position{line: 568, col: 18, offset: 13363},
				expr: &ruleRefExpr{
					pos:  position{line: 568, col: 18, offset: 13363},
					name: "ws",
				},
			},
		},
		{
			name: "EOF",
			pos:  position{line: 570, col: 1, offset: 13368},
			expr: &notExpr{
				pos: position{line: 570, col: 7, offset: 13374},
				expr: &anyMatcher{
					line: 570, col: 8, offset: 13375,
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
	ss := makeCompareAny("search", true, v)
	if getValueType(v) == "string" {
		return makeOrChain(ss, []interface{}{makeCompareAny("searchin", true, v)}), nil
	}
	ss = makeCompareAny("search", true, makeTypedValue("string", string(c.text)))
	return makeOrChain(ss, []interface{}{makeCompareAny("searchin", true, makeTypedValue("string", string(c.text))), makeCompareAny("eql", true, v), makeCompareAny("in", true, v)}), nil

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

func (c *current) onfieldRefDotOnly7(field interface{}) (interface{}, error) {
	return makeFieldCall("RecordFieldRead", nil, field), nil
}

func (p *parser) callonfieldRefDotOnly7() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldRefDotOnly7(stack["field"])
}

func (c *current) onfieldRefDotOnly1(base, refs interface{}) (interface{}, error) {
	return chainFieldCalls(base, refs), nil

}

func (p *parser) callonfieldRefDotOnly1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldRefDotOnly1(stack["base"], stack["refs"])
}

func (c *current) onfieldRefDotOnlyList7(ref interface{}) (interface{}, error) {
	return ref, nil
}

func (p *parser) callonfieldRefDotOnlyList7() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldRefDotOnlyList7(stack["ref"])
}

func (c *current) onfieldRefDotOnlyList1(first, rest interface{}) (interface{}, error) {
	result := []interface{}{first}
	for _, r := range rest.([]interface{}) {
		result = append(result, r)
	}
	return result, nil

}

func (p *parser) callonfieldRefDotOnlyList1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onfieldRefDotOnlyList1(stack["first"], stack["rest"])
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
	return joinChars(chars), nil
}

func (p *parser) callonboomWord1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onboomWord1(stack["chars"])
}

func (c *current) onboomWordPart2(s interface{}) (interface{}, error) {
	return s, nil
}

func (p *parser) callonboomWordPart2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onboomWordPart2(stack["s"])
}

func (c *current) onboomWordPart7() (interface{}, error) {
	return string(c.text), nil
}

func (p *parser) callonboomWordPart7() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onboomWordPart7()
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

func (c *current) onescapeSequence2() (interface{}, error) {
	return "\\" + string(c.text), nil
}

func (p *parser) callonescapeSequence2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onescapeSequence2()
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

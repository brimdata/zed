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
			pos:  position{line: 95, col: 1, offset: 2460},
			expr: &choiceExpr{
				pos: position{line: 96, col: 5, offset: 2476},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 96, col: 5, offset: 2476},
						run: (*parser).callonsearchValue2,
						expr: &labeledExpr{
							pos:   position{line: 96, col: 5, offset: 2476},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 96, col: 7, offset: 2478},
								name: "quotedString",
							},
						},
					},
					&actionExpr{
						pos: position{line: 99, col: 5, offset: 2549},
						run: (*parser).callonsearchValue5,
						expr: &labeledExpr{
							pos:   position{line: 99, col: 5, offset: 2549},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 99, col: 7, offset: 2551},
								name: "reString",
							},
						},
					},
					&actionExpr{
						pos: position{line: 102, col: 5, offset: 2618},
						run: (*parser).callonsearchValue8,
						expr: &labeledExpr{
							pos:   position{line: 102, col: 5, offset: 2618},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 102, col: 7, offset: 2620},
								name: "port",
							},
						},
					},
					&actionExpr{
						pos: position{line: 105, col: 5, offset: 2679},
						run: (*parser).callonsearchValue11,
						expr: &labeledExpr{
							pos:   position{line: 105, col: 5, offset: 2679},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 105, col: 7, offset: 2681},
								name: "ip6subnet",
							},
						},
					},
					&actionExpr{
						pos: position{line: 108, col: 5, offset: 2749},
						run: (*parser).callonsearchValue14,
						expr: &labeledExpr{
							pos:   position{line: 108, col: 5, offset: 2749},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 108, col: 7, offset: 2751},
								name: "ip6addr",
							},
						},
					},
					&actionExpr{
						pos: position{line: 111, col: 5, offset: 2815},
						run: (*parser).callonsearchValue17,
						expr: &labeledExpr{
							pos:   position{line: 111, col: 5, offset: 2815},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 111, col: 7, offset: 2817},
								name: "subnet",
							},
						},
					},
					&actionExpr{
						pos: position{line: 114, col: 5, offset: 2882},
						run: (*parser).callonsearchValue20,
						expr: &labeledExpr{
							pos:   position{line: 114, col: 5, offset: 2882},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 114, col: 7, offset: 2884},
								name: "addr",
							},
						},
					},
					&actionExpr{
						pos: position{line: 117, col: 5, offset: 2945},
						run: (*parser).callonsearchValue23,
						expr: &labeledExpr{
							pos:   position{line: 117, col: 5, offset: 2945},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 117, col: 7, offset: 2947},
								name: "sdouble",
							},
						},
					},
					&actionExpr{
						pos: position{line: 120, col: 5, offset: 3013},
						run: (*parser).callonsearchValue26,
						expr: &seqExpr{
							pos: position{line: 120, col: 5, offset: 3013},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 120, col: 5, offset: 3013},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 120, col: 7, offset: 3015},
										name: "sinteger",
									},
								},
								&notExpr{
									pos: position{line: 120, col: 16, offset: 3024},
									expr: &ruleRefExpr{
										pos:  position{line: 120, col: 17, offset: 3025},
										name: "boomWord",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 123, col: 5, offset: 3089},
						run: (*parser).callonsearchValue32,
						expr: &seqExpr{
							pos: position{line: 123, col: 5, offset: 3089},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 123, col: 5, offset: 3089},
									expr: &seqExpr{
										pos: position{line: 123, col: 7, offset: 3091},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 123, col: 7, offset: 3091},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 123, col: 22, offset: 3106},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 123, col: 25, offset: 3109},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 123, col: 27, offset: 3111},
										name: "booleanLiteral",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 124, col: 5, offset: 3148},
						run: (*parser).callonsearchValue40,
						expr: &seqExpr{
							pos: position{line: 124, col: 5, offset: 3148},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 124, col: 5, offset: 3148},
									expr: &seqExpr{
										pos: position{line: 124, col: 7, offset: 3150},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 124, col: 7, offset: 3150},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 124, col: 22, offset: 3165},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 124, col: 25, offset: 3168},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 124, col: 27, offset: 3170},
										name: "unsetLiteral",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 125, col: 5, offset: 3205},
						run: (*parser).callonsearchValue48,
						expr: &seqExpr{
							pos: position{line: 125, col: 5, offset: 3205},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 125, col: 5, offset: 3205},
									expr: &seqExpr{
										pos: position{line: 125, col: 7, offset: 3207},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 125, col: 7, offset: 3207},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 125, col: 22, offset: 3222},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 125, col: 25, offset: 3225},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 125, col: 27, offset: 3227},
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
			pos:  position{line: 133, col: 1, offset: 3451},
			expr: &choiceExpr{
				pos: position{line: 134, col: 5, offset: 3470},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 134, col: 5, offset: 3470},
						name: "andToken",
					},
					&ruleRefExpr{
						pos:  position{line: 135, col: 5, offset: 3483},
						name: "orToken",
					},
					&ruleRefExpr{
						pos:  position{line: 136, col: 5, offset: 3495},
						name: "inToken",
					},
				},
			},
		},
		{
			name: "booleanLiteral",
			pos:  position{line: 138, col: 1, offset: 3504},
			expr: &choiceExpr{
				pos: position{line: 139, col: 5, offset: 3523},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 139, col: 5, offset: 3523},
						run: (*parser).callonbooleanLiteral2,
						expr: &litMatcher{
							pos:        position{line: 139, col: 5, offset: 3523},
							val:        "true",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 140, col: 5, offset: 3591},
						run: (*parser).callonbooleanLiteral4,
						expr: &litMatcher{
							pos:        position{line: 140, col: 5, offset: 3591},
							val:        "false",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "unsetLiteral",
			pos:  position{line: 142, col: 1, offset: 3657},
			expr: &actionExpr{
				pos: position{line: 143, col: 5, offset: 3674},
				run: (*parser).callonunsetLiteral1,
				expr: &litMatcher{
					pos:        position{line: 143, col: 5, offset: 3674},
					val:        "nil",
					ignoreCase: false,
				},
			},
		},
		{
			name: "procList",
			pos:  position{line: 145, col: 1, offset: 3735},
			expr: &actionExpr{
				pos: position{line: 146, col: 5, offset: 3748},
				run: (*parser).callonprocList1,
				expr: &seqExpr{
					pos: position{line: 146, col: 5, offset: 3748},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 146, col: 5, offset: 3748},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 146, col: 11, offset: 3754},
								name: "procChain",
							},
						},
						&labeledExpr{
							pos:   position{line: 146, col: 21, offset: 3764},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 146, col: 26, offset: 3769},
								expr: &ruleRefExpr{
									pos:  position{line: 146, col: 26, offset: 3769},
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
			pos:  position{line: 155, col: 1, offset: 3993},
			expr: &actionExpr{
				pos: position{line: 156, col: 5, offset: 4011},
				run: (*parser).callonparallelChain1,
				expr: &seqExpr{
					pos: position{line: 156, col: 5, offset: 4011},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 156, col: 5, offset: 4011},
							expr: &ruleRefExpr{
								pos:  position{line: 156, col: 5, offset: 4011},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 156, col: 8, offset: 4014},
							val:        ";",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 156, col: 12, offset: 4018},
							expr: &ruleRefExpr{
								pos:  position{line: 156, col: 12, offset: 4018},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 156, col: 15, offset: 4021},
							label: "ch",
							expr: &ruleRefExpr{
								pos:  position{line: 156, col: 18, offset: 4024},
								name: "procChain",
							},
						},
					},
				},
			},
		},
		{
			name: "proc",
			pos:  position{line: 158, col: 1, offset: 4074},
			expr: &choiceExpr{
				pos: position{line: 159, col: 5, offset: 4083},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 159, col: 5, offset: 4083},
						name: "simpleProc",
					},
					&ruleRefExpr{
						pos:  position{line: 160, col: 5, offset: 4098},
						name: "reducerProc",
					},
					&actionExpr{
						pos: position{line: 161, col: 5, offset: 4114},
						run: (*parser).callonproc4,
						expr: &seqExpr{
							pos: position{line: 161, col: 5, offset: 4114},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 161, col: 5, offset: 4114},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 161, col: 9, offset: 4118},
									expr: &ruleRefExpr{
										pos:  position{line: 161, col: 9, offset: 4118},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 161, col: 12, offset: 4121},
									label: "proc",
									expr: &ruleRefExpr{
										pos:  position{line: 161, col: 17, offset: 4126},
										name: "procList",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 161, col: 26, offset: 4135},
									expr: &ruleRefExpr{
										pos:  position{line: 161, col: 26, offset: 4135},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 161, col: 29, offset: 4138},
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
			pos:  position{line: 165, col: 1, offset: 4174},
			expr: &actionExpr{
				pos: position{line: 166, col: 5, offset: 4186},
				run: (*parser).callongroupBy1,
				expr: &seqExpr{
					pos: position{line: 166, col: 5, offset: 4186},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 166, col: 5, offset: 4186},
							val:        "by",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 166, col: 11, offset: 4192},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 166, col: 13, offset: 4194},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 166, col: 18, offset: 4199},
								name: "fieldExprList",
							},
						},
					},
				},
			},
		},
		{
			name: "everyDur",
			pos:  position{line: 168, col: 1, offset: 4235},
			expr: &actionExpr{
				pos: position{line: 169, col: 5, offset: 4248},
				run: (*parser).calloneveryDur1,
				expr: &seqExpr{
					pos: position{line: 169, col: 5, offset: 4248},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 169, col: 5, offset: 4248},
							val:        "every",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 169, col: 14, offset: 4257},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 169, col: 16, offset: 4259},
							label: "dur",
							expr: &ruleRefExpr{
								pos:  position{line: 169, col: 20, offset: 4263},
								name: "duration",
							},
						},
					},
				},
			},
		},
		{
			name: "equalityToken",
			pos:  position{line: 171, col: 1, offset: 4293},
			expr: &choiceExpr{
				pos: position{line: 172, col: 5, offset: 4311},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 172, col: 5, offset: 4311},
						run: (*parser).callonequalityToken2,
						expr: &litMatcher{
							pos:        position{line: 172, col: 5, offset: 4311},
							val:        "=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 173, col: 5, offset: 4341},
						run: (*parser).callonequalityToken4,
						expr: &litMatcher{
							pos:        position{line: 173, col: 5, offset: 4341},
							val:        "!=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 174, col: 5, offset: 4373},
						run: (*parser).callonequalityToken6,
						expr: &litMatcher{
							pos:        position{line: 174, col: 5, offset: 4373},
							val:        "<=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 175, col: 5, offset: 4404},
						run: (*parser).callonequalityToken8,
						expr: &litMatcher{
							pos:        position{line: 175, col: 5, offset: 4404},
							val:        ">=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 176, col: 5, offset: 4435},
						run: (*parser).callonequalityToken10,
						expr: &litMatcher{
							pos:        position{line: 176, col: 5, offset: 4435},
							val:        "<",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 177, col: 5, offset: 4464},
						run: (*parser).callonequalityToken12,
						expr: &litMatcher{
							pos:        position{line: 177, col: 5, offset: 4464},
							val:        ">",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "types",
			pos:  position{line: 179, col: 1, offset: 4490},
			expr: &choiceExpr{
				pos: position{line: 180, col: 5, offset: 4500},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 180, col: 5, offset: 4500},
						val:        "bool",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 181, col: 5, offset: 4511},
						val:        "int",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 182, col: 5, offset: 4521},
						val:        "count",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 183, col: 5, offset: 4533},
						val:        "double",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 184, col: 5, offset: 4546},
						val:        "string",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 185, col: 5, offset: 4559},
						val:        "addr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 186, col: 5, offset: 4570},
						val:        "subnet",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 187, col: 5, offset: 4583},
						val:        "port",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "dash",
			pos:  position{line: 189, col: 1, offset: 4591},
			expr: &choiceExpr{
				pos: position{line: 189, col: 8, offset: 4598},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 189, col: 8, offset: 4598},
						val:        "-",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 189, col: 14, offset: 4604},
						val:        "—",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 189, col: 25, offset: 4615},
						val:        "–",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 189, col: 36, offset: 4626},
						exprs: []interface{}{
							&litMatcher{
								pos:        position{line: 189, col: 36, offset: 4626},
								val:        "to",
								ignoreCase: false,
							},
							&andExpr{
								pos: position{line: 189, col: 40, offset: 4630},
								expr: &ruleRefExpr{
									pos:  position{line: 189, col: 42, offset: 4632},
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
			pos:  position{line: 191, col: 1, offset: 4636},
			expr: &litMatcher{
				pos:        position{line: 191, col: 12, offset: 4647},
				val:        "and",
				ignoreCase: false,
			},
		},
		{
			name: "orToken",
			pos:  position{line: 192, col: 1, offset: 4653},
			expr: &litMatcher{
				pos:        position{line: 192, col: 11, offset: 4663},
				val:        "or",
				ignoreCase: false,
			},
		},
		{
			name: "inToken",
			pos:  position{line: 193, col: 1, offset: 4668},
			expr: &litMatcher{
				pos:        position{line: 193, col: 11, offset: 4678},
				val:        "in",
				ignoreCase: false,
			},
		},
		{
			name: "notToken",
			pos:  position{line: 194, col: 1, offset: 4683},
			expr: &litMatcher{
				pos:        position{line: 194, col: 12, offset: 4694},
				val:        "not",
				ignoreCase: false,
			},
		},
		{
			name: "fieldName",
			pos:  position{line: 196, col: 1, offset: 4701},
			expr: &actionExpr{
				pos: position{line: 196, col: 13, offset: 4713},
				run: (*parser).callonfieldName1,
				expr: &seqExpr{
					pos: position{line: 196, col: 13, offset: 4713},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 196, col: 13, offset: 4713},
							name: "fieldNameStart",
						},
						&zeroOrMoreExpr{
							pos: position{line: 196, col: 28, offset: 4728},
							expr: &ruleRefExpr{
								pos:  position{line: 196, col: 28, offset: 4728},
								name: "fieldNameRest",
							},
						},
					},
				},
			},
		},
		{
			name: "fieldNameStart",
			pos:  position{line: 198, col: 1, offset: 4775},
			expr: &charClassMatcher{
				pos:        position{line: 198, col: 18, offset: 4792},
				val:        "[A-Za-z_$]",
				chars:      []rune{'_', '$'},
				ranges:     []rune{'A', 'Z', 'a', 'z'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "fieldNameRest",
			pos:  position{line: 199, col: 1, offset: 4803},
			expr: &choiceExpr{
				pos: position{line: 199, col: 17, offset: 4819},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 199, col: 17, offset: 4819},
						name: "fieldNameStart",
					},
					&charClassMatcher{
						pos:        position{line: 199, col: 34, offset: 4836},
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
			pos:  position{line: 201, col: 1, offset: 4843},
			expr: &actionExpr{
				pos: position{line: 202, col: 4, offset: 4861},
				run: (*parser).callonfieldReference1,
				expr: &seqExpr{
					pos: position{line: 202, col: 4, offset: 4861},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 202, col: 4, offset: 4861},
							label: "base",
							expr: &ruleRefExpr{
								pos:  position{line: 202, col: 9, offset: 4866},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 202, col: 19, offset: 4876},
							label: "derefs",
							expr: &zeroOrMoreExpr{
								pos: position{line: 202, col: 26, offset: 4883},
								expr: &choiceExpr{
									pos: position{line: 203, col: 8, offset: 4892},
									alternatives: []interface{}{
										&actionExpr{
											pos: position{line: 203, col: 8, offset: 4892},
											run: (*parser).callonfieldReference8,
											expr: &seqExpr{
												pos: position{line: 203, col: 8, offset: 4892},
												exprs: []interface{}{
													&litMatcher{
														pos:        position{line: 203, col: 8, offset: 4892},
														val:        ".",
														ignoreCase: false,
													},
													&labeledExpr{
														pos:   position{line: 203, col: 12, offset: 4896},
														label: "field",
														expr: &ruleRefExpr{
															pos:  position{line: 203, col: 18, offset: 4902},
															name: "fieldName",
														},
													},
												},
											},
										},
										&actionExpr{
											pos: position{line: 204, col: 8, offset: 4983},
											run: (*parser).callonfieldReference13,
											expr: &seqExpr{
												pos: position{line: 204, col: 8, offset: 4983},
												exprs: []interface{}{
													&litMatcher{
														pos:        position{line: 204, col: 8, offset: 4983},
														val:        "[",
														ignoreCase: false,
													},
													&labeledExpr{
														pos:   position{line: 204, col: 12, offset: 4987},
														label: "index",
														expr: &ruleRefExpr{
															pos:  position{line: 204, col: 18, offset: 4993},
															name: "sinteger",
														},
													},
													&litMatcher{
														pos:        position{line: 204, col: 27, offset: 5002},
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
			pos:  position{line: 209, col: 1, offset: 5118},
			expr: &choiceExpr{
				pos: position{line: 210, col: 5, offset: 5132},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 210, col: 5, offset: 5132},
						run: (*parser).callonfieldExpr2,
						expr: &seqExpr{
							pos: position{line: 210, col: 5, offset: 5132},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 210, col: 5, offset: 5132},
									label: "op",
									expr: &ruleRefExpr{
										pos:  position{line: 210, col: 8, offset: 5135},
										name: "fieldOp",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 210, col: 16, offset: 5143},
									expr: &ruleRefExpr{
										pos:  position{line: 210, col: 16, offset: 5143},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 210, col: 19, offset: 5146},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 210, col: 23, offset: 5150},
									expr: &ruleRefExpr{
										pos:  position{line: 210, col: 23, offset: 5150},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 210, col: 26, offset: 5153},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 210, col: 32, offset: 5159},
										name: "fieldReference",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 210, col: 47, offset: 5174},
									expr: &ruleRefExpr{
										pos:  position{line: 210, col: 47, offset: 5174},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 210, col: 50, offset: 5177},
									val:        ")",
									ignoreCase: false,
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 213, col: 5, offset: 5241},
						name: "fieldReference",
					},
				},
			},
		},
		{
			name: "fieldOp",
			pos:  position{line: 215, col: 1, offset: 5257},
			expr: &actionExpr{
				pos: position{line: 216, col: 5, offset: 5269},
				run: (*parser).callonfieldOp1,
				expr: &litMatcher{
					pos:        position{line: 216, col: 5, offset: 5269},
					val:        "len",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldExprList",
			pos:  position{line: 218, col: 1, offset: 5299},
			expr: &actionExpr{
				pos: position{line: 219, col: 5, offset: 5317},
				run: (*parser).callonfieldExprList1,
				expr: &seqExpr{
					pos: position{line: 219, col: 5, offset: 5317},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 219, col: 5, offset: 5317},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 219, col: 11, offset: 5323},
								name: "fieldExpr",
							},
						},
						&labeledExpr{
							pos:   position{line: 219, col: 21, offset: 5333},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 219, col: 26, offset: 5338},
								expr: &seqExpr{
									pos: position{line: 219, col: 27, offset: 5339},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 219, col: 27, offset: 5339},
											expr: &ruleRefExpr{
												pos:  position{line: 219, col: 27, offset: 5339},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 219, col: 30, offset: 5342},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 219, col: 34, offset: 5346},
											expr: &ruleRefExpr{
												pos:  position{line: 219, col: 34, offset: 5346},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 219, col: 37, offset: 5349},
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
			pos:  position{line: 229, col: 1, offset: 5544},
			expr: &actionExpr{
				pos: position{line: 230, col: 5, offset: 5564},
				run: (*parser).callonfieldRefDotOnly1,
				expr: &seqExpr{
					pos: position{line: 230, col: 5, offset: 5564},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 230, col: 5, offset: 5564},
							label: "base",
							expr: &ruleRefExpr{
								pos:  position{line: 230, col: 10, offset: 5569},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 230, col: 20, offset: 5579},
							label: "refs",
							expr: &zeroOrMoreExpr{
								pos: position{line: 230, col: 25, offset: 5584},
								expr: &actionExpr{
									pos: position{line: 230, col: 26, offset: 5585},
									run: (*parser).callonfieldRefDotOnly7,
									expr: &seqExpr{
										pos: position{line: 230, col: 26, offset: 5585},
										exprs: []interface{}{
											&litMatcher{
												pos:        position{line: 230, col: 26, offset: 5585},
												val:        ".",
												ignoreCase: false,
											},
											&labeledExpr{
												pos:   position{line: 230, col: 30, offset: 5589},
												label: "field",
												expr: &ruleRefExpr{
													pos:  position{line: 230, col: 36, offset: 5595},
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
			pos:  position{line: 234, col: 1, offset: 5720},
			expr: &actionExpr{
				pos: position{line: 235, col: 5, offset: 5744},
				run: (*parser).callonfieldRefDotOnlyList1,
				expr: &seqExpr{
					pos: position{line: 235, col: 5, offset: 5744},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 235, col: 5, offset: 5744},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 235, col: 11, offset: 5750},
								name: "fieldRefDotOnly",
							},
						},
						&labeledExpr{
							pos:   position{line: 235, col: 27, offset: 5766},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 235, col: 32, offset: 5771},
								expr: &actionExpr{
									pos: position{line: 235, col: 33, offset: 5772},
									run: (*parser).callonfieldRefDotOnlyList7,
									expr: &seqExpr{
										pos: position{line: 235, col: 33, offset: 5772},
										exprs: []interface{}{
											&zeroOrOneExpr{
												pos: position{line: 235, col: 33, offset: 5772},
												expr: &ruleRefExpr{
													pos:  position{line: 235, col: 33, offset: 5772},
													name: "_",
												},
											},
											&litMatcher{
												pos:        position{line: 235, col: 36, offset: 5775},
												val:        ",",
												ignoreCase: false,
											},
											&zeroOrOneExpr{
												pos: position{line: 235, col: 40, offset: 5779},
												expr: &ruleRefExpr{
													pos:  position{line: 235, col: 40, offset: 5779},
													name: "_",
												},
											},
											&labeledExpr{
												pos:   position{line: 235, col: 43, offset: 5782},
												label: "ref",
												expr: &ruleRefExpr{
													pos:  position{line: 235, col: 47, offset: 5786},
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
			pos:  position{line: 243, col: 1, offset: 5966},
			expr: &actionExpr{
				pos: position{line: 244, col: 5, offset: 5984},
				run: (*parser).callonfieldNameList1,
				expr: &seqExpr{
					pos: position{line: 244, col: 5, offset: 5984},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 244, col: 5, offset: 5984},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 244, col: 11, offset: 5990},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 244, col: 21, offset: 6000},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 244, col: 26, offset: 6005},
								expr: &seqExpr{
									pos: position{line: 244, col: 27, offset: 6006},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 244, col: 27, offset: 6006},
											expr: &ruleRefExpr{
												pos:  position{line: 244, col: 27, offset: 6006},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 244, col: 30, offset: 6009},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 244, col: 34, offset: 6013},
											expr: &ruleRefExpr{
												pos:  position{line: 244, col: 34, offset: 6013},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 244, col: 37, offset: 6016},
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
			pos:  position{line: 252, col: 1, offset: 6209},
			expr: &actionExpr{
				pos: position{line: 253, col: 5, offset: 6221},
				run: (*parser).calloncountOp1,
				expr: &litMatcher{
					pos:        position{line: 253, col: 5, offset: 6221},
					val:        "count",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldReducerOp",
			pos:  position{line: 255, col: 1, offset: 6255},
			expr: &choiceExpr{
				pos: position{line: 256, col: 5, offset: 6274},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 256, col: 5, offset: 6274},
						run: (*parser).callonfieldReducerOp2,
						expr: &litMatcher{
							pos:        position{line: 256, col: 5, offset: 6274},
							val:        "sum",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 257, col: 5, offset: 6308},
						run: (*parser).callonfieldReducerOp4,
						expr: &litMatcher{
							pos:        position{line: 257, col: 5, offset: 6308},
							val:        "avg",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 258, col: 5, offset: 6342},
						run: (*parser).callonfieldReducerOp6,
						expr: &litMatcher{
							pos:        position{line: 258, col: 5, offset: 6342},
							val:        "stdev",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 259, col: 5, offset: 6379},
						run: (*parser).callonfieldReducerOp8,
						expr: &litMatcher{
							pos:        position{line: 259, col: 5, offset: 6379},
							val:        "sd",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 260, col: 5, offset: 6415},
						run: (*parser).callonfieldReducerOp10,
						expr: &litMatcher{
							pos:        position{line: 260, col: 5, offset: 6415},
							val:        "var",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 261, col: 5, offset: 6449},
						run: (*parser).callonfieldReducerOp12,
						expr: &litMatcher{
							pos:        position{line: 261, col: 5, offset: 6449},
							val:        "entropy",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 262, col: 5, offset: 6490},
						run: (*parser).callonfieldReducerOp14,
						expr: &litMatcher{
							pos:        position{line: 262, col: 5, offset: 6490},
							val:        "min",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 263, col: 5, offset: 6524},
						run: (*parser).callonfieldReducerOp16,
						expr: &litMatcher{
							pos:        position{line: 263, col: 5, offset: 6524},
							val:        "max",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 264, col: 5, offset: 6558},
						run: (*parser).callonfieldReducerOp18,
						expr: &litMatcher{
							pos:        position{line: 264, col: 5, offset: 6558},
							val:        "first",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 265, col: 5, offset: 6596},
						run: (*parser).callonfieldReducerOp20,
						expr: &litMatcher{
							pos:        position{line: 265, col: 5, offset: 6596},
							val:        "last",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 266, col: 5, offset: 6632},
						run: (*parser).callonfieldReducerOp22,
						expr: &litMatcher{
							pos:        position{line: 266, col: 5, offset: 6632},
							val:        "countdistinct",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "paddedFieldName",
			pos:  position{line: 268, col: 1, offset: 6682},
			expr: &actionExpr{
				pos: position{line: 268, col: 19, offset: 6700},
				run: (*parser).callonpaddedFieldName1,
				expr: &seqExpr{
					pos: position{line: 268, col: 19, offset: 6700},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 268, col: 19, offset: 6700},
							expr: &ruleRefExpr{
								pos:  position{line: 268, col: 19, offset: 6700},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 268, col: 22, offset: 6703},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 268, col: 28, offset: 6709},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 268, col: 38, offset: 6719},
							expr: &ruleRefExpr{
								pos:  position{line: 268, col: 38, offset: 6719},
								name: "_",
							},
						},
					},
				},
			},
		},
		{
			name: "countReducer",
			pos:  position{line: 270, col: 1, offset: 6745},
			expr: &actionExpr{
				pos: position{line: 271, col: 5, offset: 6762},
				run: (*parser).calloncountReducer1,
				expr: &seqExpr{
					pos: position{line: 271, col: 5, offset: 6762},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 271, col: 5, offset: 6762},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 271, col: 8, offset: 6765},
								name: "countOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 271, col: 16, offset: 6773},
							expr: &ruleRefExpr{
								pos:  position{line: 271, col: 16, offset: 6773},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 271, col: 19, offset: 6776},
							val:        "(",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 271, col: 23, offset: 6780},
							label: "field",
							expr: &zeroOrOneExpr{
								pos: position{line: 271, col: 29, offset: 6786},
								expr: &ruleRefExpr{
									pos:  position{line: 271, col: 29, offset: 6786},
									name: "paddedFieldName",
								},
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 271, col: 47, offset: 6804},
							expr: &ruleRefExpr{
								pos:  position{line: 271, col: 47, offset: 6804},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 271, col: 50, offset: 6807},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "fieldReducer",
			pos:  position{line: 275, col: 1, offset: 6866},
			expr: &actionExpr{
				pos: position{line: 276, col: 5, offset: 6883},
				run: (*parser).callonfieldReducer1,
				expr: &seqExpr{
					pos: position{line: 276, col: 5, offset: 6883},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 276, col: 5, offset: 6883},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 276, col: 8, offset: 6886},
								name: "fieldReducerOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 276, col: 23, offset: 6901},
							expr: &ruleRefExpr{
								pos:  position{line: 276, col: 23, offset: 6901},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 276, col: 26, offset: 6904},
							val:        "(",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 276, col: 30, offset: 6908},
							expr: &ruleRefExpr{
								pos:  position{line: 276, col: 30, offset: 6908},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 276, col: 33, offset: 6911},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 276, col: 39, offset: 6917},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 276, col: 50, offset: 6928},
							expr: &ruleRefExpr{
								pos:  position{line: 276, col: 50, offset: 6928},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 276, col: 53, offset: 6931},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reducerProc",
			pos:  position{line: 280, col: 1, offset: 6998},
			expr: &actionExpr{
				pos: position{line: 281, col: 5, offset: 7014},
				run: (*parser).callonreducerProc1,
				expr: &seqExpr{
					pos: position{line: 281, col: 5, offset: 7014},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 281, col: 5, offset: 7014},
							label: "every",
							expr: &zeroOrOneExpr{
								pos: position{line: 281, col: 11, offset: 7020},
								expr: &seqExpr{
									pos: position{line: 281, col: 12, offset: 7021},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 281, col: 12, offset: 7021},
											name: "everyDur",
										},
										&ruleRefExpr{
											pos:  position{line: 281, col: 21, offset: 7030},
											name: "_",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 281, col: 25, offset: 7034},
							label: "reducers",
							expr: &ruleRefExpr{
								pos:  position{line: 281, col: 34, offset: 7043},
								name: "reducerList",
							},
						},
						&labeledExpr{
							pos:   position{line: 281, col: 46, offset: 7055},
							label: "keys",
							expr: &zeroOrOneExpr{
								pos: position{line: 281, col: 51, offset: 7060},
								expr: &seqExpr{
									pos: position{line: 281, col: 52, offset: 7061},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 281, col: 52, offset: 7061},
											name: "_",
										},
										&ruleRefExpr{
											pos:  position{line: 281, col: 54, offset: 7063},
											name: "groupBy",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 281, col: 64, offset: 7073},
							label: "limit",
							expr: &zeroOrOneExpr{
								pos: position{line: 281, col: 70, offset: 7079},
								expr: &ruleRefExpr{
									pos:  position{line: 281, col: 70, offset: 7079},
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
			pos:  position{line: 299, col: 1, offset: 7436},
			expr: &actionExpr{
				pos: position{line: 300, col: 5, offset: 7449},
				run: (*parser).callonasClause1,
				expr: &seqExpr{
					pos: position{line: 300, col: 5, offset: 7449},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 300, col: 5, offset: 7449},
							val:        "as",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 300, col: 11, offset: 7455},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 300, col: 13, offset: 7457},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 300, col: 15, offset: 7459},
								name: "fieldName",
							},
						},
					},
				},
			},
		},
		{
			name: "reducerExpr",
			pos:  position{line: 302, col: 1, offset: 7488},
			expr: &choiceExpr{
				pos: position{line: 303, col: 5, offset: 7504},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 303, col: 5, offset: 7504},
						run: (*parser).callonreducerExpr2,
						expr: &seqExpr{
							pos: position{line: 303, col: 5, offset: 7504},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 303, col: 5, offset: 7504},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 303, col: 11, offset: 7510},
										name: "fieldName",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 303, col: 21, offset: 7520},
									expr: &ruleRefExpr{
										pos:  position{line: 303, col: 21, offset: 7520},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 303, col: 24, offset: 7523},
									val:        "=",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 303, col: 28, offset: 7527},
									expr: &ruleRefExpr{
										pos:  position{line: 303, col: 28, offset: 7527},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 303, col: 31, offset: 7530},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 303, col: 33, offset: 7532},
										name: "reducer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 306, col: 5, offset: 7595},
						run: (*parser).callonreducerExpr13,
						expr: &seqExpr{
							pos: position{line: 306, col: 5, offset: 7595},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 306, col: 5, offset: 7595},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 306, col: 7, offset: 7597},
										name: "reducer",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 306, col: 15, offset: 7605},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 306, col: 17, offset: 7607},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 306, col: 23, offset: 7613},
										name: "asClause",
									},
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 309, col: 5, offset: 7677},
						name: "reducer",
					},
				},
			},
		},
		{
			name: "reducer",
			pos:  position{line: 311, col: 1, offset: 7686},
			expr: &choiceExpr{
				pos: position{line: 312, col: 5, offset: 7698},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 312, col: 5, offset: 7698},
						name: "countReducer",
					},
					&ruleRefExpr{
						pos:  position{line: 313, col: 5, offset: 7715},
						name: "fieldReducer",
					},
				},
			},
		},
		{
			name: "reducerList",
			pos:  position{line: 315, col: 1, offset: 7729},
			expr: &actionExpr{
				pos: position{line: 316, col: 5, offset: 7745},
				run: (*parser).callonreducerList1,
				expr: &seqExpr{
					pos: position{line: 316, col: 5, offset: 7745},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 316, col: 5, offset: 7745},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 316, col: 11, offset: 7751},
								name: "reducerExpr",
							},
						},
						&labeledExpr{
							pos:   position{line: 316, col: 23, offset: 7763},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 316, col: 28, offset: 7768},
								expr: &seqExpr{
									pos: position{line: 316, col: 29, offset: 7769},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 316, col: 29, offset: 7769},
											expr: &ruleRefExpr{
												pos:  position{line: 316, col: 29, offset: 7769},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 316, col: 32, offset: 7772},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 316, col: 36, offset: 7776},
											expr: &ruleRefExpr{
												pos:  position{line: 316, col: 36, offset: 7776},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 316, col: 39, offset: 7779},
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
			pos:  position{line: 324, col: 1, offset: 7976},
			expr: &choiceExpr{
				pos: position{line: 325, col: 5, offset: 7991},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 325, col: 5, offset: 7991},
						name: "sort",
					},
					&ruleRefExpr{
						pos:  position{line: 326, col: 5, offset: 8000},
						name: "top",
					},
					&ruleRefExpr{
						pos:  position{line: 327, col: 5, offset: 8008},
						name: "cut",
					},
					&ruleRefExpr{
						pos:  position{line: 328, col: 5, offset: 8016},
						name: "head",
					},
					&ruleRefExpr{
						pos:  position{line: 329, col: 5, offset: 8025},
						name: "tail",
					},
					&ruleRefExpr{
						pos:  position{line: 330, col: 5, offset: 8034},
						name: "filter",
					},
					&ruleRefExpr{
						pos:  position{line: 331, col: 5, offset: 8045},
						name: "uniq",
					},
				},
			},
		},
		{
			name: "sort",
			pos:  position{line: 333, col: 1, offset: 8051},
			expr: &choiceExpr{
				pos: position{line: 334, col: 5, offset: 8060},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 334, col: 5, offset: 8060},
						run: (*parser).callonsort2,
						expr: &seqExpr{
							pos: position{line: 334, col: 5, offset: 8060},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 334, col: 5, offset: 8060},
									val:        "sort",
									ignoreCase: true,
								},
								&labeledExpr{
									pos:   position{line: 334, col: 13, offset: 8068},
									label: "rev",
									expr: &zeroOrOneExpr{
										pos: position{line: 334, col: 17, offset: 8072},
										expr: &seqExpr{
											pos: position{line: 334, col: 18, offset: 8073},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 334, col: 18, offset: 8073},
													name: "_",
												},
												&litMatcher{
													pos:        position{line: 334, col: 20, offset: 8075},
													val:        "-r",
													ignoreCase: false,
												},
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 334, col: 27, offset: 8082},
									label: "limit",
									expr: &zeroOrOneExpr{
										pos: position{line: 334, col: 33, offset: 8088},
										expr: &ruleRefExpr{
											pos:  position{line: 334, col: 33, offset: 8088},
											name: "procLimitArg",
										},
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 334, col: 48, offset: 8103},
									expr: &ruleRefExpr{
										pos:  position{line: 334, col: 48, offset: 8103},
										name: "_",
									},
								},
								&notExpr{
									pos: position{line: 334, col: 51, offset: 8106},
									expr: &litMatcher{
										pos:        position{line: 334, col: 52, offset: 8107},
										val:        "-r",
										ignoreCase: false,
									},
								},
								&labeledExpr{
									pos:   position{line: 334, col: 57, offset: 8112},
									label: "list",
									expr: &zeroOrOneExpr{
										pos: position{line: 334, col: 62, offset: 8117},
										expr: &ruleRefExpr{
											pos:  position{line: 334, col: 63, offset: 8118},
											name: "fieldExprList",
										},
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 339, col: 5, offset: 8248},
						run: (*parser).callonsort20,
						expr: &seqExpr{
							pos: position{line: 339, col: 5, offset: 8248},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 339, col: 5, offset: 8248},
									val:        "sort",
									ignoreCase: true,
								},
								&labeledExpr{
									pos:   position{line: 339, col: 13, offset: 8256},
									label: "limit",
									expr: &zeroOrOneExpr{
										pos: position{line: 339, col: 19, offset: 8262},
										expr: &ruleRefExpr{
											pos:  position{line: 339, col: 19, offset: 8262},
											name: "procLimitArg",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 339, col: 33, offset: 8276},
									label: "rev",
									expr: &zeroOrOneExpr{
										pos: position{line: 339, col: 37, offset: 8280},
										expr: &seqExpr{
											pos: position{line: 339, col: 38, offset: 8281},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 339, col: 38, offset: 8281},
													name: "_",
												},
												&litMatcher{
													pos:        position{line: 339, col: 40, offset: 8283},
													val:        "-r",
													ignoreCase: false,
												},
											},
										},
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 339, col: 47, offset: 8290},
									expr: &ruleRefExpr{
										pos:  position{line: 339, col: 47, offset: 8290},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 339, col: 50, offset: 8293},
									label: "list",
									expr: &zeroOrOneExpr{
										pos: position{line: 339, col: 55, offset: 8298},
										expr: &ruleRefExpr{
											pos:  position{line: 339, col: 56, offset: 8299},
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
			pos:  position{line: 345, col: 1, offset: 8426},
			expr: &actionExpr{
				pos: position{line: 346, col: 5, offset: 8434},
				run: (*parser).callontop1,
				expr: &seqExpr{
					pos: position{line: 346, col: 5, offset: 8434},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 346, col: 5, offset: 8434},
							val:        "top",
							ignoreCase: true,
						},
						&labeledExpr{
							pos:   position{line: 346, col: 12, offset: 8441},
							label: "limit",
							expr: &zeroOrOneExpr{
								pos: position{line: 346, col: 18, offset: 8447},
								expr: &ruleRefExpr{
									pos:  position{line: 346, col: 18, offset: 8447},
									name: "procLimitArg",
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 346, col: 32, offset: 8461},
							label: "flush",
							expr: &zeroOrOneExpr{
								pos: position{line: 346, col: 38, offset: 8467},
								expr: &seqExpr{
									pos: position{line: 346, col: 39, offset: 8468},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 346, col: 39, offset: 8468},
											name: "_",
										},
										&litMatcher{
											pos:        position{line: 346, col: 41, offset: 8470},
											val:        "-flush",
											ignoreCase: false,
										},
									},
								},
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 346, col: 52, offset: 8481},
							expr: &ruleRefExpr{
								pos:  position{line: 346, col: 52, offset: 8481},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 346, col: 55, offset: 8484},
							label: "list",
							expr: &zeroOrOneExpr{
								pos: position{line: 346, col: 60, offset: 8489},
								expr: &ruleRefExpr{
									pos:  position{line: 346, col: 61, offset: 8490},
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
			pos:  position{line: 350, col: 1, offset: 8561},
			expr: &actionExpr{
				pos: position{line: 351, col: 5, offset: 8578},
				run: (*parser).callonprocLimitArg1,
				expr: &seqExpr{
					pos: position{line: 351, col: 5, offset: 8578},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 351, col: 5, offset: 8578},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 351, col: 7, offset: 8580},
							val:        "-limit",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 351, col: 16, offset: 8589},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 351, col: 18, offset: 8591},
							label: "limit",
							expr: &ruleRefExpr{
								pos:  position{line: 351, col: 24, offset: 8597},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "cut",
			pos:  position{line: 353, col: 1, offset: 8628},
			expr: &actionExpr{
				pos: position{line: 354, col: 5, offset: 8636},
				run: (*parser).calloncut1,
				expr: &seqExpr{
					pos: position{line: 354, col: 5, offset: 8636},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 354, col: 5, offset: 8636},
							val:        "cut",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 354, col: 12, offset: 8643},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 354, col: 14, offset: 8645},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 354, col: 19, offset: 8650},
								name: "fieldRefDotOnlyList",
							},
						},
					},
				},
			},
		},
		{
			name: "head",
			pos:  position{line: 355, col: 1, offset: 8704},
			expr: &choiceExpr{
				pos: position{line: 356, col: 5, offset: 8713},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 356, col: 5, offset: 8713},
						run: (*parser).callonhead2,
						expr: &seqExpr{
							pos: position{line: 356, col: 5, offset: 8713},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 356, col: 5, offset: 8713},
									val:        "head",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 356, col: 13, offset: 8721},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 356, col: 15, offset: 8723},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 356, col: 21, offset: 8729},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 357, col: 5, offset: 8777},
						run: (*parser).callonhead8,
						expr: &litMatcher{
							pos:        position{line: 357, col: 5, offset: 8777},
							val:        "head",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "tail",
			pos:  position{line: 358, col: 1, offset: 8817},
			expr: &choiceExpr{
				pos: position{line: 359, col: 5, offset: 8826},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 359, col: 5, offset: 8826},
						run: (*parser).callontail2,
						expr: &seqExpr{
							pos: position{line: 359, col: 5, offset: 8826},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 359, col: 5, offset: 8826},
									val:        "tail",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 359, col: 13, offset: 8834},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 359, col: 15, offset: 8836},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 359, col: 21, offset: 8842},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 360, col: 5, offset: 8890},
						run: (*parser).callontail8,
						expr: &litMatcher{
							pos:        position{line: 360, col: 5, offset: 8890},
							val:        "tail",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "filter",
			pos:  position{line: 362, col: 1, offset: 8931},
			expr: &actionExpr{
				pos: position{line: 363, col: 5, offset: 8942},
				run: (*parser).callonfilter1,
				expr: &seqExpr{
					pos: position{line: 363, col: 5, offset: 8942},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 363, col: 5, offset: 8942},
							val:        "filter",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 363, col: 15, offset: 8952},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 363, col: 17, offset: 8954},
							label: "expr",
							expr: &ruleRefExpr{
								pos:  position{line: 363, col: 22, offset: 8959},
								name: "searchExpr",
							},
						},
					},
				},
			},
		},
		{
			name: "uniq",
			pos:  position{line: 366, col: 1, offset: 9017},
			expr: &choiceExpr{
				pos: position{line: 367, col: 5, offset: 9026},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 367, col: 5, offset: 9026},
						run: (*parser).callonuniq2,
						expr: &seqExpr{
							pos: position{line: 367, col: 5, offset: 9026},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 367, col: 5, offset: 9026},
									val:        "uniq",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 367, col: 13, offset: 9034},
									name: "_",
								},
								&litMatcher{
									pos:        position{line: 367, col: 15, offset: 9036},
									val:        "-c",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 370, col: 5, offset: 9090},
						run: (*parser).callonuniq7,
						expr: &litMatcher{
							pos:        position{line: 370, col: 5, offset: 9090},
							val:        "uniq",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "duration",
			pos:  position{line: 374, col: 1, offset: 9145},
			expr: &choiceExpr{
				pos: position{line: 375, col: 5, offset: 9158},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 375, col: 5, offset: 9158},
						name: "seconds",
					},
					&ruleRefExpr{
						pos:  position{line: 376, col: 5, offset: 9170},
						name: "minutes",
					},
					&ruleRefExpr{
						pos:  position{line: 377, col: 5, offset: 9182},
						name: "hours",
					},
					&seqExpr{
						pos: position{line: 378, col: 5, offset: 9192},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 378, col: 5, offset: 9192},
								name: "hours",
							},
							&ruleRefExpr{
								pos:  position{line: 378, col: 11, offset: 9198},
								name: "_",
							},
							&litMatcher{
								pos:        position{line: 378, col: 13, offset: 9200},
								val:        "and",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 378, col: 19, offset: 9206},
								name: "_",
							},
							&ruleRefExpr{
								pos:  position{line: 378, col: 21, offset: 9208},
								name: "minutes",
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 379, col: 5, offset: 9220},
						name: "days",
					},
					&ruleRefExpr{
						pos:  position{line: 380, col: 5, offset: 9229},
						name: "weeks",
					},
				},
			},
		},
		{
			name: "sec_abbrev",
			pos:  position{line: 382, col: 1, offset: 9236},
			expr: &choiceExpr{
				pos: position{line: 383, col: 5, offset: 9251},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 383, col: 5, offset: 9251},
						val:        "seconds",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 384, col: 5, offset: 9265},
						val:        "second",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 385, col: 5, offset: 9278},
						val:        "secs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 386, col: 5, offset: 9289},
						val:        "sec",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 387, col: 5, offset: 9299},
						val:        "s",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "min_abbrev",
			pos:  position{line: 389, col: 1, offset: 9304},
			expr: &choiceExpr{
				pos: position{line: 390, col: 5, offset: 9319},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 390, col: 5, offset: 9319},
						val:        "minutes",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 391, col: 5, offset: 9333},
						val:        "minute",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 392, col: 5, offset: 9346},
						val:        "mins",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 393, col: 5, offset: 9357},
						val:        "min",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 394, col: 5, offset: 9367},
						val:        "m",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "hour_abbrev",
			pos:  position{line: 396, col: 1, offset: 9372},
			expr: &choiceExpr{
				pos: position{line: 397, col: 5, offset: 9388},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 397, col: 5, offset: 9388},
						val:        "hours",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 398, col: 5, offset: 9400},
						val:        "hrs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 399, col: 5, offset: 9410},
						val:        "hr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 400, col: 5, offset: 9419},
						val:        "h",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 401, col: 5, offset: 9427},
						val:        "hour",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "day_abbrev",
			pos:  position{line: 403, col: 1, offset: 9435},
			expr: &choiceExpr{
				pos: position{line: 403, col: 14, offset: 9448},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 403, col: 14, offset: 9448},
						val:        "days",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 403, col: 21, offset: 9455},
						val:        "day",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 403, col: 27, offset: 9461},
						val:        "d",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "week_abbrev",
			pos:  position{line: 404, col: 1, offset: 9465},
			expr: &choiceExpr{
				pos: position{line: 404, col: 15, offset: 9479},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 404, col: 15, offset: 9479},
						val:        "weeks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 404, col: 23, offset: 9487},
						val:        "week",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 404, col: 30, offset: 9494},
						val:        "wks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 404, col: 36, offset: 9500},
						val:        "wk",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 404, col: 41, offset: 9505},
						val:        "w",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "seconds",
			pos:  position{line: 406, col: 1, offset: 9510},
			expr: &choiceExpr{
				pos: position{line: 407, col: 5, offset: 9522},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 407, col: 5, offset: 9522},
						run: (*parser).callonseconds2,
						expr: &litMatcher{
							pos:        position{line: 407, col: 5, offset: 9522},
							val:        "second",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 408, col: 5, offset: 9567},
						run: (*parser).callonseconds4,
						expr: &seqExpr{
							pos: position{line: 408, col: 5, offset: 9567},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 408, col: 5, offset: 9567},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 408, col: 9, offset: 9571},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 408, col: 16, offset: 9578},
									expr: &ruleRefExpr{
										pos:  position{line: 408, col: 16, offset: 9578},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 408, col: 19, offset: 9581},
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
			pos:  position{line: 410, col: 1, offset: 9627},
			expr: &choiceExpr{
				pos: position{line: 411, col: 5, offset: 9639},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 411, col: 5, offset: 9639},
						run: (*parser).callonminutes2,
						expr: &litMatcher{
							pos:        position{line: 411, col: 5, offset: 9639},
							val:        "minute",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 412, col: 5, offset: 9685},
						run: (*parser).callonminutes4,
						expr: &seqExpr{
							pos: position{line: 412, col: 5, offset: 9685},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 412, col: 5, offset: 9685},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 412, col: 9, offset: 9689},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 412, col: 16, offset: 9696},
									expr: &ruleRefExpr{
										pos:  position{line: 412, col: 16, offset: 9696},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 412, col: 19, offset: 9699},
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
			pos:  position{line: 414, col: 1, offset: 9754},
			expr: &choiceExpr{
				pos: position{line: 415, col: 5, offset: 9764},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 415, col: 5, offset: 9764},
						run: (*parser).callonhours2,
						expr: &litMatcher{
							pos:        position{line: 415, col: 5, offset: 9764},
							val:        "hour",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 416, col: 5, offset: 9810},
						run: (*parser).callonhours4,
						expr: &seqExpr{
							pos: position{line: 416, col: 5, offset: 9810},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 416, col: 5, offset: 9810},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 416, col: 9, offset: 9814},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 416, col: 16, offset: 9821},
									expr: &ruleRefExpr{
										pos:  position{line: 416, col: 16, offset: 9821},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 416, col: 19, offset: 9824},
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
			pos:  position{line: 418, col: 1, offset: 9882},
			expr: &choiceExpr{
				pos: position{line: 419, col: 5, offset: 9891},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 419, col: 5, offset: 9891},
						run: (*parser).callondays2,
						expr: &litMatcher{
							pos:        position{line: 419, col: 5, offset: 9891},
							val:        "day",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 420, col: 5, offset: 9939},
						run: (*parser).callondays4,
						expr: &seqExpr{
							pos: position{line: 420, col: 5, offset: 9939},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 420, col: 5, offset: 9939},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 420, col: 9, offset: 9943},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 420, col: 16, offset: 9950},
									expr: &ruleRefExpr{
										pos:  position{line: 420, col: 16, offset: 9950},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 420, col: 19, offset: 9953},
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
			pos:  position{line: 422, col: 1, offset: 10013},
			expr: &actionExpr{
				pos: position{line: 423, col: 5, offset: 10023},
				run: (*parser).callonweeks1,
				expr: &seqExpr{
					pos: position{line: 423, col: 5, offset: 10023},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 423, col: 5, offset: 10023},
							label: "num",
							expr: &ruleRefExpr{
								pos:  position{line: 423, col: 9, offset: 10027},
								name: "number",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 423, col: 16, offset: 10034},
							expr: &ruleRefExpr{
								pos:  position{line: 423, col: 16, offset: 10034},
								name: "_",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 423, col: 19, offset: 10037},
							name: "week_abbrev",
						},
					},
				},
			},
		},
		{
			name: "number",
			pos:  position{line: 425, col: 1, offset: 10100},
			expr: &ruleRefExpr{
				pos:  position{line: 425, col: 10, offset: 10109},
				name: "integer",
			},
		},
		{
			name: "addr",
			pos:  position{line: 429, col: 1, offset: 10147},
			expr: &actionExpr{
				pos: position{line: 430, col: 5, offset: 10156},
				run: (*parser).callonaddr1,
				expr: &labeledExpr{
					pos:   position{line: 430, col: 5, offset: 10156},
					label: "a",
					expr: &seqExpr{
						pos: position{line: 430, col: 8, offset: 10159},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 430, col: 8, offset: 10159},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 430, col: 16, offset: 10167},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 430, col: 20, offset: 10171},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 430, col: 28, offset: 10179},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 430, col: 32, offset: 10183},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 430, col: 40, offset: 10191},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 430, col: 44, offset: 10195},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "port",
			pos:  position{line: 432, col: 1, offset: 10236},
			expr: &actionExpr{
				pos: position{line: 433, col: 5, offset: 10245},
				run: (*parser).callonport1,
				expr: &seqExpr{
					pos: position{line: 433, col: 5, offset: 10245},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 433, col: 5, offset: 10245},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 433, col: 9, offset: 10249},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 433, col: 11, offset: 10251},
								name: "sinteger",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6addr",
			pos:  position{line: 437, col: 1, offset: 10410},
			expr: &choiceExpr{
				pos: position{line: 438, col: 5, offset: 10422},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 438, col: 5, offset: 10422},
						run: (*parser).callonip6addr2,
						expr: &seqExpr{
							pos: position{line: 438, col: 5, offset: 10422},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 438, col: 5, offset: 10422},
									label: "a",
									expr: &oneOrMoreExpr{
										pos: position{line: 438, col: 7, offset: 10424},
										expr: &ruleRefExpr{
											pos:  position{line: 438, col: 8, offset: 10425},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 438, col: 20, offset: 10437},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 438, col: 22, offset: 10439},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 441, col: 5, offset: 10503},
						run: (*parser).callonip6addr9,
						expr: &seqExpr{
							pos: position{line: 441, col: 5, offset: 10503},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 441, col: 5, offset: 10503},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 441, col: 7, offset: 10505},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 441, col: 11, offset: 10509},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 441, col: 13, offset: 10511},
										expr: &ruleRefExpr{
											pos:  position{line: 441, col: 14, offset: 10512},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 441, col: 25, offset: 10523},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 441, col: 30, offset: 10528},
									label: "d",
									expr: &zeroOrMoreExpr{
										pos: position{line: 441, col: 32, offset: 10530},
										expr: &ruleRefExpr{
											pos:  position{line: 441, col: 33, offset: 10531},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 441, col: 45, offset: 10543},
									label: "e",
									expr: &ruleRefExpr{
										pos:  position{line: 441, col: 47, offset: 10545},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 444, col: 5, offset: 10644},
						run: (*parser).callonip6addr22,
						expr: &seqExpr{
							pos: position{line: 444, col: 5, offset: 10644},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 444, col: 5, offset: 10644},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 444, col: 10, offset: 10649},
									label: "a",
									expr: &zeroOrMoreExpr{
										pos: position{line: 444, col: 12, offset: 10651},
										expr: &ruleRefExpr{
											pos:  position{line: 444, col: 13, offset: 10652},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 444, col: 25, offset: 10664},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 444, col: 27, offset: 10666},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 447, col: 5, offset: 10737},
						run: (*parser).callonip6addr30,
						expr: &seqExpr{
							pos: position{line: 447, col: 5, offset: 10737},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 447, col: 5, offset: 10737},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 447, col: 7, offset: 10739},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 447, col: 11, offset: 10743},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 447, col: 13, offset: 10745},
										expr: &ruleRefExpr{
											pos:  position{line: 447, col: 14, offset: 10746},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 447, col: 25, offset: 10757},
									val:        "::",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 450, col: 5, offset: 10825},
						run: (*parser).callonip6addr38,
						expr: &litMatcher{
							pos:        position{line: 450, col: 5, offset: 10825},
							val:        "::",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "ip6tail",
			pos:  position{line: 454, col: 1, offset: 10862},
			expr: &choiceExpr{
				pos: position{line: 455, col: 5, offset: 10874},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 455, col: 5, offset: 10874},
						name: "addr",
					},
					&ruleRefExpr{
						pos:  position{line: 456, col: 5, offset: 10883},
						name: "h16",
					},
				},
			},
		},
		{
			name: "h_append",
			pos:  position{line: 458, col: 1, offset: 10888},
			expr: &actionExpr{
				pos: position{line: 458, col: 12, offset: 10899},
				run: (*parser).callonh_append1,
				expr: &seqExpr{
					pos: position{line: 458, col: 12, offset: 10899},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 458, col: 12, offset: 10899},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 458, col: 16, offset: 10903},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 458, col: 18, offset: 10905},
								name: "h16",
							},
						},
					},
				},
			},
		},
		{
			name: "h_prepend",
			pos:  position{line: 459, col: 1, offset: 10942},
			expr: &actionExpr{
				pos: position{line: 459, col: 13, offset: 10954},
				run: (*parser).callonh_prepend1,
				expr: &seqExpr{
					pos: position{line: 459, col: 13, offset: 10954},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 459, col: 13, offset: 10954},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 459, col: 15, offset: 10956},
								name: "h16",
							},
						},
						&litMatcher{
							pos:        position{line: 459, col: 19, offset: 10960},
							val:        ":",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "sub_addr",
			pos:  position{line: 461, col: 1, offset: 10998},
			expr: &choiceExpr{
				pos: position{line: 462, col: 5, offset: 11011},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 462, col: 5, offset: 11011},
						name: "addr",
					},
					&actionExpr{
						pos: position{line: 463, col: 5, offset: 11020},
						run: (*parser).callonsub_addr3,
						expr: &labeledExpr{
							pos:   position{line: 463, col: 5, offset: 11020},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 463, col: 8, offset: 11023},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 463, col: 8, offset: 11023},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 463, col: 16, offset: 11031},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 463, col: 20, offset: 11035},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 463, col: 28, offset: 11043},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 463, col: 32, offset: 11047},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 464, col: 5, offset: 11099},
						run: (*parser).callonsub_addr11,
						expr: &labeledExpr{
							pos:   position{line: 464, col: 5, offset: 11099},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 464, col: 8, offset: 11102},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 464, col: 8, offset: 11102},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 464, col: 16, offset: 11110},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 464, col: 20, offset: 11114},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 465, col: 5, offset: 11168},
						run: (*parser).callonsub_addr17,
						expr: &labeledExpr{
							pos:   position{line: 465, col: 5, offset: 11168},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 465, col: 7, offset: 11170},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "subnet",
			pos:  position{line: 467, col: 1, offset: 11221},
			expr: &actionExpr{
				pos: position{line: 468, col: 5, offset: 11232},
				run: (*parser).callonsubnet1,
				expr: &seqExpr{
					pos: position{line: 468, col: 5, offset: 11232},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 468, col: 5, offset: 11232},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 468, col: 7, offset: 11234},
								name: "sub_addr",
							},
						},
						&litMatcher{
							pos:        position{line: 468, col: 16, offset: 11243},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 468, col: 20, offset: 11247},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 468, col: 22, offset: 11249},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6subnet",
			pos:  position{line: 472, col: 1, offset: 11325},
			expr: &actionExpr{
				pos: position{line: 473, col: 5, offset: 11339},
				run: (*parser).callonip6subnet1,
				expr: &seqExpr{
					pos: position{line: 473, col: 5, offset: 11339},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 473, col: 5, offset: 11339},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 473, col: 7, offset: 11341},
								name: "ip6addr",
							},
						},
						&litMatcher{
							pos:        position{line: 473, col: 15, offset: 11349},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 473, col: 19, offset: 11353},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 473, col: 21, offset: 11355},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "integer",
			pos:  position{line: 477, col: 1, offset: 11421},
			expr: &actionExpr{
				pos: position{line: 478, col: 5, offset: 11433},
				run: (*parser).calloninteger1,
				expr: &labeledExpr{
					pos:   position{line: 478, col: 5, offset: 11433},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 478, col: 7, offset: 11435},
						name: "sinteger",
					},
				},
			},
		},
		{
			name: "sinteger",
			pos:  position{line: 482, col: 1, offset: 11479},
			expr: &actionExpr{
				pos: position{line: 483, col: 5, offset: 11492},
				run: (*parser).callonsinteger1,
				expr: &labeledExpr{
					pos:   position{line: 483, col: 5, offset: 11492},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 483, col: 11, offset: 11498},
						expr: &charClassMatcher{
							pos:        position{line: 483, col: 11, offset: 11498},
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
			pos:  position{line: 487, col: 1, offset: 11543},
			expr: &actionExpr{
				pos: position{line: 488, col: 5, offset: 11554},
				run: (*parser).callondouble1,
				expr: &labeledExpr{
					pos:   position{line: 488, col: 5, offset: 11554},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 488, col: 7, offset: 11556},
						name: "sdouble",
					},
				},
			},
		},
		{
			name: "sdouble",
			pos:  position{line: 492, col: 1, offset: 11603},
			expr: &choiceExpr{
				pos: position{line: 493, col: 5, offset: 11615},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 493, col: 5, offset: 11615},
						run: (*parser).callonsdouble2,
						expr: &seqExpr{
							pos: position{line: 493, col: 5, offset: 11615},
							exprs: []interface{}{
								&oneOrMoreExpr{
									pos: position{line: 493, col: 5, offset: 11615},
									expr: &ruleRefExpr{
										pos:  position{line: 493, col: 5, offset: 11615},
										name: "doubleInteger",
									},
								},
								&litMatcher{
									pos:        position{line: 493, col: 20, offset: 11630},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 493, col: 24, offset: 11634},
									expr: &ruleRefExpr{
										pos:  position{line: 493, col: 24, offset: 11634},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 493, col: 37, offset: 11647},
									expr: &ruleRefExpr{
										pos:  position{line: 493, col: 37, offset: 11647},
										name: "exponentPart",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 496, col: 5, offset: 11706},
						run: (*parser).callonsdouble11,
						expr: &seqExpr{
							pos: position{line: 496, col: 5, offset: 11706},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 496, col: 5, offset: 11706},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 496, col: 9, offset: 11710},
									expr: &ruleRefExpr{
										pos:  position{line: 496, col: 9, offset: 11710},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 496, col: 22, offset: 11723},
									expr: &ruleRefExpr{
										pos:  position{line: 496, col: 22, offset: 11723},
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
			pos:  position{line: 500, col: 1, offset: 11779},
			expr: &choiceExpr{
				pos: position{line: 501, col: 5, offset: 11797},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 501, col: 5, offset: 11797},
						val:        "0",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 502, col: 5, offset: 11805},
						exprs: []interface{}{
							&charClassMatcher{
								pos:        position{line: 502, col: 5, offset: 11805},
								val:        "[1-9]",
								ranges:     []rune{'1', '9'},
								ignoreCase: false,
								inverted:   false,
							},
							&zeroOrMoreExpr{
								pos: position{line: 502, col: 11, offset: 11811},
								expr: &charClassMatcher{
									pos:        position{line: 502, col: 11, offset: 11811},
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
			pos:  position{line: 504, col: 1, offset: 11819},
			expr: &charClassMatcher{
				pos:        position{line: 504, col: 15, offset: 11833},
				val:        "[0-9]",
				ranges:     []rune{'0', '9'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "signedInteger",
			pos:  position{line: 506, col: 1, offset: 11840},
			expr: &seqExpr{
				pos: position{line: 506, col: 17, offset: 11856},
				exprs: []interface{}{
					&zeroOrOneExpr{
						pos: position{line: 506, col: 17, offset: 11856},
						expr: &charClassMatcher{
							pos:        position{line: 506, col: 17, offset: 11856},
							val:        "[+-]",
							chars:      []rune{'+', '-'},
							ignoreCase: false,
							inverted:   false,
						},
					},
					&oneOrMoreExpr{
						pos: position{line: 506, col: 23, offset: 11862},
						expr: &ruleRefExpr{
							pos:  position{line: 506, col: 23, offset: 11862},
							name: "doubleDigit",
						},
					},
				},
			},
		},
		{
			name: "exponentPart",
			pos:  position{line: 508, col: 1, offset: 11876},
			expr: &seqExpr{
				pos: position{line: 508, col: 16, offset: 11891},
				exprs: []interface{}{
					&litMatcher{
						pos:        position{line: 508, col: 16, offset: 11891},
						val:        "e",
						ignoreCase: true,
					},
					&ruleRefExpr{
						pos:  position{line: 508, col: 21, offset: 11896},
						name: "signedInteger",
					},
				},
			},
		},
		{
			name: "h16",
			pos:  position{line: 510, col: 1, offset: 11911},
			expr: &actionExpr{
				pos: position{line: 510, col: 7, offset: 11917},
				run: (*parser).callonh161,
				expr: &labeledExpr{
					pos:   position{line: 510, col: 7, offset: 11917},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 510, col: 13, offset: 11923},
						expr: &ruleRefExpr{
							pos:  position{line: 510, col: 13, offset: 11923},
							name: "hexdigit",
						},
					},
				},
			},
		},
		{
			name: "hexdigit",
			pos:  position{line: 512, col: 1, offset: 11965},
			expr: &charClassMatcher{
				pos:        position{line: 512, col: 12, offset: 11976},
				val:        "[0-9a-fA-F]",
				ranges:     []rune{'0', '9', 'a', 'f', 'A', 'F'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name:        "boomWord",
			displayName: "\"boomWord\"",
			pos:         position{line: 514, col: 1, offset: 11989},
			expr: &actionExpr{
				pos: position{line: 514, col: 23, offset: 12011},
				run: (*parser).callonboomWord1,
				expr: &labeledExpr{
					pos:   position{line: 514, col: 23, offset: 12011},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 514, col: 29, offset: 12017},
						expr: &ruleRefExpr{
							pos:  position{line: 514, col: 29, offset: 12017},
							name: "boomWordPart",
						},
					},
				},
			},
		},
		{
			name: "boomWordPart",
			pos:  position{line: 516, col: 1, offset: 12065},
			expr: &choiceExpr{
				pos: position{line: 517, col: 5, offset: 12082},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 517, col: 5, offset: 12082},
						run: (*parser).callonboomWordPart2,
						expr: &seqExpr{
							pos: position{line: 517, col: 5, offset: 12082},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 517, col: 5, offset: 12082},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 517, col: 10, offset: 12087},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 517, col: 12, offset: 12089},
										name: "escapeSequence",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 518, col: 5, offset: 12127},
						run: (*parser).callonboomWordPart7,
						expr: &seqExpr{
							pos: position{line: 518, col: 5, offset: 12127},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 518, col: 5, offset: 12127},
									expr: &choiceExpr{
										pos: position{line: 518, col: 7, offset: 12129},
										alternatives: []interface{}{
											&charClassMatcher{
												pos:        position{line: 518, col: 7, offset: 12129},
												val:        "[\\x00-\\x1F\\x5C(),!><=\\x22|\\x27;]",
												chars:      []rune{'\\', '(', ')', ',', '!', '>', '<', '=', '"', '|', '\'', ';'},
												ranges:     []rune{'\x00', '\x1f'},
												ignoreCase: false,
												inverted:   false,
											},
											&ruleRefExpr{
												pos:  position{line: 518, col: 42, offset: 12164},
												name: "ws",
											},
										},
									},
								},
								&anyMatcher{
									line: 518, col: 46, offset: 12168,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "quotedString",
			pos:  position{line: 520, col: 1, offset: 12202},
			expr: &choiceExpr{
				pos: position{line: 521, col: 5, offset: 12219},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 521, col: 5, offset: 12219},
						run: (*parser).callonquotedString2,
						expr: &seqExpr{
							pos: position{line: 521, col: 5, offset: 12219},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 521, col: 5, offset: 12219},
									val:        "\"",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 521, col: 9, offset: 12223},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 521, col: 11, offset: 12225},
										expr: &ruleRefExpr{
											pos:  position{line: 521, col: 11, offset: 12225},
											name: "doubleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 521, col: 29, offset: 12243},
									val:        "\"",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 522, col: 5, offset: 12280},
						run: (*parser).callonquotedString9,
						expr: &seqExpr{
							pos: position{line: 522, col: 5, offset: 12280},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 522, col: 5, offset: 12280},
									val:        "'",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 522, col: 9, offset: 12284},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 522, col: 11, offset: 12286},
										expr: &ruleRefExpr{
											pos:  position{line: 522, col: 11, offset: 12286},
											name: "singleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 522, col: 29, offset: 12304},
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
			pos:  position{line: 524, col: 1, offset: 12338},
			expr: &choiceExpr{
				pos: position{line: 525, col: 5, offset: 12359},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 525, col: 5, offset: 12359},
						run: (*parser).callondoubleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 525, col: 5, offset: 12359},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 525, col: 5, offset: 12359},
									expr: &choiceExpr{
										pos: position{line: 525, col: 7, offset: 12361},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 525, col: 7, offset: 12361},
												val:        "\"",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 525, col: 13, offset: 12367},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 525, col: 26, offset: 12380,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 526, col: 5, offset: 12417},
						run: (*parser).callondoubleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 526, col: 5, offset: 12417},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 526, col: 5, offset: 12417},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 526, col: 10, offset: 12422},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 526, col: 12, offset: 12424},
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
			pos:  position{line: 528, col: 1, offset: 12458},
			expr: &choiceExpr{
				pos: position{line: 529, col: 5, offset: 12479},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 529, col: 5, offset: 12479},
						run: (*parser).callonsingleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 529, col: 5, offset: 12479},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 529, col: 5, offset: 12479},
									expr: &choiceExpr{
										pos: position{line: 529, col: 7, offset: 12481},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 529, col: 7, offset: 12481},
												val:        "'",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 529, col: 13, offset: 12487},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 529, col: 26, offset: 12500,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 530, col: 5, offset: 12537},
						run: (*parser).callonsingleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 530, col: 5, offset: 12537},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 530, col: 5, offset: 12537},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 530, col: 10, offset: 12542},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 530, col: 12, offset: 12544},
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
			pos:  position{line: 532, col: 1, offset: 12578},
			expr: &choiceExpr{
				pos: position{line: 533, col: 5, offset: 12597},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 533, col: 5, offset: 12597},
						run: (*parser).callonescapeSequence2,
						expr: &seqExpr{
							pos: position{line: 533, col: 5, offset: 12597},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 533, col: 5, offset: 12597},
									val:        "x",
									ignoreCase: false,
								},
								&ruleRefExpr{
									pos:  position{line: 533, col: 9, offset: 12601},
									name: "hexdigit",
								},
								&ruleRefExpr{
									pos:  position{line: 533, col: 18, offset: 12610},
									name: "hexdigit",
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 534, col: 5, offset: 12661},
						name: "singleCharEscape",
					},
					&ruleRefExpr{
						pos:  position{line: 535, col: 5, offset: 12682},
						name: "unicodeEscape",
					},
				},
			},
		},
		{
			name: "singleCharEscape",
			pos:  position{line: 537, col: 1, offset: 12697},
			expr: &choiceExpr{
				pos: position{line: 538, col: 5, offset: 12718},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 538, col: 5, offset: 12718},
						val:        "'",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 539, col: 5, offset: 12726},
						val:        "\"",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 540, col: 5, offset: 12734},
						val:        "\\",
						ignoreCase: false,
					},
					&actionExpr{
						pos: position{line: 541, col: 5, offset: 12743},
						run: (*parser).callonsingleCharEscape5,
						expr: &litMatcher{
							pos:        position{line: 541, col: 5, offset: 12743},
							val:        "b",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 542, col: 5, offset: 12772},
						run: (*parser).callonsingleCharEscape7,
						expr: &litMatcher{
							pos:        position{line: 542, col: 5, offset: 12772},
							val:        "f",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 543, col: 5, offset: 12801},
						run: (*parser).callonsingleCharEscape9,
						expr: &litMatcher{
							pos:        position{line: 543, col: 5, offset: 12801},
							val:        "n",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 544, col: 5, offset: 12830},
						run: (*parser).callonsingleCharEscape11,
						expr: &litMatcher{
							pos:        position{line: 544, col: 5, offset: 12830},
							val:        "r",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 545, col: 5, offset: 12859},
						run: (*parser).callonsingleCharEscape13,
						expr: &litMatcher{
							pos:        position{line: 545, col: 5, offset: 12859},
							val:        "t",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 546, col: 5, offset: 12888},
						run: (*parser).callonsingleCharEscape15,
						expr: &litMatcher{
							pos:        position{line: 546, col: 5, offset: 12888},
							val:        "v",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "unicodeEscape",
			pos:  position{line: 548, col: 1, offset: 12914},
			expr: &seqExpr{
				pos: position{line: 549, col: 5, offset: 12932},
				exprs: []interface{}{
					&litMatcher{
						pos:        position{line: 549, col: 5, offset: 12932},
						val:        "u",
						ignoreCase: false,
					},
					&ruleRefExpr{
						pos:  position{line: 549, col: 9, offset: 12936},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 549, col: 18, offset: 12945},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 549, col: 27, offset: 12954},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 549, col: 36, offset: 12963},
						name: "hexdigit",
					},
				},
			},
		},
		{
			name: "reString",
			pos:  position{line: 551, col: 1, offset: 12973},
			expr: &actionExpr{
				pos: position{line: 552, col: 5, offset: 12986},
				run: (*parser).callonreString1,
				expr: &seqExpr{
					pos: position{line: 552, col: 5, offset: 12986},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 552, col: 5, offset: 12986},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 552, col: 9, offset: 12990},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 552, col: 11, offset: 12992},
								name: "reBody",
							},
						},
						&litMatcher{
							pos:        position{line: 552, col: 18, offset: 12999},
							val:        "/",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reBody",
			pos:  position{line: 554, col: 1, offset: 13022},
			expr: &actionExpr{
				pos: position{line: 555, col: 5, offset: 13033},
				run: (*parser).callonreBody1,
				expr: &oneOrMoreExpr{
					pos: position{line: 555, col: 5, offset: 13033},
					expr: &choiceExpr{
						pos: position{line: 555, col: 6, offset: 13034},
						alternatives: []interface{}{
							&charClassMatcher{
								pos:        position{line: 555, col: 6, offset: 13034},
								val:        "[^/\\\\]",
								chars:      []rune{'/', '\\'},
								ignoreCase: false,
								inverted:   true,
							},
							&litMatcher{
								pos:        position{line: 555, col: 13, offset: 13041},
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
			pos:  position{line: 557, col: 1, offset: 13081},
			expr: &charClassMatcher{
				pos:        position{line: 558, col: 5, offset: 13097},
				val:        "[\\x00-\\x1f\\\\]",
				chars:      []rune{'\\'},
				ranges:     []rune{'\x00', '\x1f'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "ws",
			pos:  position{line: 560, col: 1, offset: 13112},
			expr: &choiceExpr{
				pos: position{line: 561, col: 5, offset: 13119},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 561, col: 5, offset: 13119},
						val:        "\t",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 562, col: 5, offset: 13128},
						val:        "\v",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 563, col: 5, offset: 13137},
						val:        "\f",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 564, col: 5, offset: 13146},
						val:        " ",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 565, col: 5, offset: 13154},
						val:        "\u00a0",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 566, col: 5, offset: 13167},
						val:        "\ufeff",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name:        "_",
			displayName: "\"whitespace\"",
			pos:         position{line: 568, col: 1, offset: 13177},
			expr: &oneOrMoreExpr{
				pos: position{line: 568, col: 18, offset: 13194},
				expr: &ruleRefExpr{
					pos:  position{line: 568, col: 18, offset: 13194},
					name: "ws",
				},
			},
		},
		{
			name: "EOF",
			pos:  position{line: 570, col: 1, offset: 13199},
			expr: &notExpr{
				pos: position{line: 570, col: 7, offset: 13205},
				expr: &anyMatcher{
					line: 570, col: 8, offset: 13206,
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

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
			name: "fieldNameList",
			pos:  position{line: 229, col: 1, offset: 5544},
			expr: &actionExpr{
				pos: position{line: 230, col: 5, offset: 5562},
				run: (*parser).callonfieldNameList1,
				expr: &seqExpr{
					pos: position{line: 230, col: 5, offset: 5562},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 230, col: 5, offset: 5562},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 230, col: 11, offset: 5568},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 230, col: 21, offset: 5578},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 230, col: 26, offset: 5583},
								expr: &seqExpr{
									pos: position{line: 230, col: 27, offset: 5584},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 230, col: 27, offset: 5584},
											expr: &ruleRefExpr{
												pos:  position{line: 230, col: 27, offset: 5584},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 230, col: 30, offset: 5587},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 230, col: 34, offset: 5591},
											expr: &ruleRefExpr{
												pos:  position{line: 230, col: 34, offset: 5591},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 230, col: 37, offset: 5594},
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
			pos:  position{line: 238, col: 1, offset: 5787},
			expr: &actionExpr{
				pos: position{line: 239, col: 5, offset: 5799},
				run: (*parser).calloncountOp1,
				expr: &litMatcher{
					pos:        position{line: 239, col: 5, offset: 5799},
					val:        "count",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldReducerOp",
			pos:  position{line: 241, col: 1, offset: 5833},
			expr: &choiceExpr{
				pos: position{line: 242, col: 5, offset: 5852},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 242, col: 5, offset: 5852},
						run: (*parser).callonfieldReducerOp2,
						expr: &litMatcher{
							pos:        position{line: 242, col: 5, offset: 5852},
							val:        "sum",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 243, col: 5, offset: 5886},
						run: (*parser).callonfieldReducerOp4,
						expr: &litMatcher{
							pos:        position{line: 243, col: 5, offset: 5886},
							val:        "avg",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 244, col: 5, offset: 5920},
						run: (*parser).callonfieldReducerOp6,
						expr: &litMatcher{
							pos:        position{line: 244, col: 5, offset: 5920},
							val:        "stdev",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 245, col: 5, offset: 5957},
						run: (*parser).callonfieldReducerOp8,
						expr: &litMatcher{
							pos:        position{line: 245, col: 5, offset: 5957},
							val:        "sd",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 246, col: 5, offset: 5993},
						run: (*parser).callonfieldReducerOp10,
						expr: &litMatcher{
							pos:        position{line: 246, col: 5, offset: 5993},
							val:        "var",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 247, col: 5, offset: 6027},
						run: (*parser).callonfieldReducerOp12,
						expr: &litMatcher{
							pos:        position{line: 247, col: 5, offset: 6027},
							val:        "entropy",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 248, col: 5, offset: 6068},
						run: (*parser).callonfieldReducerOp14,
						expr: &litMatcher{
							pos:        position{line: 248, col: 5, offset: 6068},
							val:        "min",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 249, col: 5, offset: 6102},
						run: (*parser).callonfieldReducerOp16,
						expr: &litMatcher{
							pos:        position{line: 249, col: 5, offset: 6102},
							val:        "max",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 250, col: 5, offset: 6136},
						run: (*parser).callonfieldReducerOp18,
						expr: &litMatcher{
							pos:        position{line: 250, col: 5, offset: 6136},
							val:        "first",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 251, col: 5, offset: 6174},
						run: (*parser).callonfieldReducerOp20,
						expr: &litMatcher{
							pos:        position{line: 251, col: 5, offset: 6174},
							val:        "last",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 252, col: 5, offset: 6210},
						run: (*parser).callonfieldReducerOp22,
						expr: &litMatcher{
							pos:        position{line: 252, col: 5, offset: 6210},
							val:        "countdistinct",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "paddedFieldName",
			pos:  position{line: 254, col: 1, offset: 6260},
			expr: &actionExpr{
				pos: position{line: 254, col: 19, offset: 6278},
				run: (*parser).callonpaddedFieldName1,
				expr: &seqExpr{
					pos: position{line: 254, col: 19, offset: 6278},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 254, col: 19, offset: 6278},
							expr: &ruleRefExpr{
								pos:  position{line: 254, col: 19, offset: 6278},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 254, col: 22, offset: 6281},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 254, col: 28, offset: 6287},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 254, col: 38, offset: 6297},
							expr: &ruleRefExpr{
								pos:  position{line: 254, col: 38, offset: 6297},
								name: "_",
							},
						},
					},
				},
			},
		},
		{
			name: "countReducer",
			pos:  position{line: 256, col: 1, offset: 6323},
			expr: &actionExpr{
				pos: position{line: 257, col: 5, offset: 6340},
				run: (*parser).calloncountReducer1,
				expr: &seqExpr{
					pos: position{line: 257, col: 5, offset: 6340},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 257, col: 5, offset: 6340},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 257, col: 8, offset: 6343},
								name: "countOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 257, col: 16, offset: 6351},
							expr: &ruleRefExpr{
								pos:  position{line: 257, col: 16, offset: 6351},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 257, col: 19, offset: 6354},
							val:        "(",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 257, col: 23, offset: 6358},
							label: "field",
							expr: &zeroOrOneExpr{
								pos: position{line: 257, col: 29, offset: 6364},
								expr: &ruleRefExpr{
									pos:  position{line: 257, col: 29, offset: 6364},
									name: "paddedFieldName",
								},
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 257, col: 47, offset: 6382},
							expr: &ruleRefExpr{
								pos:  position{line: 257, col: 47, offset: 6382},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 257, col: 50, offset: 6385},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "fieldReducer",
			pos:  position{line: 261, col: 1, offset: 6444},
			expr: &actionExpr{
				pos: position{line: 262, col: 5, offset: 6461},
				run: (*parser).callonfieldReducer1,
				expr: &seqExpr{
					pos: position{line: 262, col: 5, offset: 6461},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 262, col: 5, offset: 6461},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 262, col: 8, offset: 6464},
								name: "fieldReducerOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 262, col: 23, offset: 6479},
							expr: &ruleRefExpr{
								pos:  position{line: 262, col: 23, offset: 6479},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 262, col: 26, offset: 6482},
							val:        "(",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 262, col: 30, offset: 6486},
							expr: &ruleRefExpr{
								pos:  position{line: 262, col: 30, offset: 6486},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 262, col: 33, offset: 6489},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 262, col: 39, offset: 6495},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 262, col: 50, offset: 6506},
							expr: &ruleRefExpr{
								pos:  position{line: 262, col: 50, offset: 6506},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 262, col: 53, offset: 6509},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reducerProc",
			pos:  position{line: 266, col: 1, offset: 6576},
			expr: &actionExpr{
				pos: position{line: 267, col: 5, offset: 6592},
				run: (*parser).callonreducerProc1,
				expr: &seqExpr{
					pos: position{line: 267, col: 5, offset: 6592},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 267, col: 5, offset: 6592},
							label: "every",
							expr: &zeroOrOneExpr{
								pos: position{line: 267, col: 11, offset: 6598},
								expr: &seqExpr{
									pos: position{line: 267, col: 12, offset: 6599},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 267, col: 12, offset: 6599},
											name: "everyDur",
										},
										&ruleRefExpr{
											pos:  position{line: 267, col: 21, offset: 6608},
											name: "_",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 267, col: 25, offset: 6612},
							label: "reducers",
							expr: &ruleRefExpr{
								pos:  position{line: 267, col: 34, offset: 6621},
								name: "reducerList",
							},
						},
						&labeledExpr{
							pos:   position{line: 267, col: 46, offset: 6633},
							label: "keys",
							expr: &zeroOrOneExpr{
								pos: position{line: 267, col: 51, offset: 6638},
								expr: &seqExpr{
									pos: position{line: 267, col: 52, offset: 6639},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 267, col: 52, offset: 6639},
											name: "_",
										},
										&ruleRefExpr{
											pos:  position{line: 267, col: 54, offset: 6641},
											name: "groupBy",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 267, col: 64, offset: 6651},
							label: "limit",
							expr: &zeroOrOneExpr{
								pos: position{line: 267, col: 70, offset: 6657},
								expr: &ruleRefExpr{
									pos:  position{line: 267, col: 70, offset: 6657},
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
			pos:  position{line: 285, col: 1, offset: 7014},
			expr: &actionExpr{
				pos: position{line: 286, col: 5, offset: 7027},
				run: (*parser).callonasClause1,
				expr: &seqExpr{
					pos: position{line: 286, col: 5, offset: 7027},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 286, col: 5, offset: 7027},
							val:        "as",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 286, col: 11, offset: 7033},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 286, col: 13, offset: 7035},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 286, col: 15, offset: 7037},
								name: "fieldName",
							},
						},
					},
				},
			},
		},
		{
			name: "reducerExpr",
			pos:  position{line: 288, col: 1, offset: 7066},
			expr: &choiceExpr{
				pos: position{line: 289, col: 5, offset: 7082},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 289, col: 5, offset: 7082},
						run: (*parser).callonreducerExpr2,
						expr: &seqExpr{
							pos: position{line: 289, col: 5, offset: 7082},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 289, col: 5, offset: 7082},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 289, col: 11, offset: 7088},
										name: "fieldName",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 289, col: 21, offset: 7098},
									expr: &ruleRefExpr{
										pos:  position{line: 289, col: 21, offset: 7098},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 289, col: 24, offset: 7101},
									val:        "=",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 289, col: 28, offset: 7105},
									expr: &ruleRefExpr{
										pos:  position{line: 289, col: 28, offset: 7105},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 289, col: 31, offset: 7108},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 289, col: 33, offset: 7110},
										name: "reducer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 292, col: 5, offset: 7173},
						run: (*parser).callonreducerExpr13,
						expr: &seqExpr{
							pos: position{line: 292, col: 5, offset: 7173},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 292, col: 5, offset: 7173},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 292, col: 7, offset: 7175},
										name: "reducer",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 292, col: 15, offset: 7183},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 292, col: 17, offset: 7185},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 292, col: 23, offset: 7191},
										name: "asClause",
									},
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 295, col: 5, offset: 7255},
						name: "reducer",
					},
				},
			},
		},
		{
			name: "reducer",
			pos:  position{line: 297, col: 1, offset: 7264},
			expr: &choiceExpr{
				pos: position{line: 298, col: 5, offset: 7276},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 298, col: 5, offset: 7276},
						name: "countReducer",
					},
					&ruleRefExpr{
						pos:  position{line: 299, col: 5, offset: 7293},
						name: "fieldReducer",
					},
				},
			},
		},
		{
			name: "reducerList",
			pos:  position{line: 301, col: 1, offset: 7307},
			expr: &actionExpr{
				pos: position{line: 302, col: 5, offset: 7323},
				run: (*parser).callonreducerList1,
				expr: &seqExpr{
					pos: position{line: 302, col: 5, offset: 7323},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 302, col: 5, offset: 7323},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 302, col: 11, offset: 7329},
								name: "reducerExpr",
							},
						},
						&labeledExpr{
							pos:   position{line: 302, col: 23, offset: 7341},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 302, col: 28, offset: 7346},
								expr: &seqExpr{
									pos: position{line: 302, col: 29, offset: 7347},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 302, col: 29, offset: 7347},
											expr: &ruleRefExpr{
												pos:  position{line: 302, col: 29, offset: 7347},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 302, col: 32, offset: 7350},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 302, col: 36, offset: 7354},
											expr: &ruleRefExpr{
												pos:  position{line: 302, col: 36, offset: 7354},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 302, col: 39, offset: 7357},
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
			pos:  position{line: 310, col: 1, offset: 7554},
			expr: &choiceExpr{
				pos: position{line: 311, col: 5, offset: 7569},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 311, col: 5, offset: 7569},
						name: "sort",
					},
					&ruleRefExpr{
						pos:  position{line: 312, col: 5, offset: 7578},
						name: "top",
					},
					&ruleRefExpr{
						pos:  position{line: 313, col: 5, offset: 7586},
						name: "cut",
					},
					&ruleRefExpr{
						pos:  position{line: 314, col: 5, offset: 7594},
						name: "head",
					},
					&ruleRefExpr{
						pos:  position{line: 315, col: 5, offset: 7603},
						name: "tail",
					},
					&ruleRefExpr{
						pos:  position{line: 316, col: 5, offset: 7612},
						name: "filter",
					},
					&ruleRefExpr{
						pos:  position{line: 317, col: 5, offset: 7623},
						name: "uniq",
					},
				},
			},
		},
		{
			name: "sort",
			pos:  position{line: 319, col: 1, offset: 7629},
			expr: &choiceExpr{
				pos: position{line: 320, col: 5, offset: 7638},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 320, col: 5, offset: 7638},
						run: (*parser).callonsort2,
						expr: &seqExpr{
							pos: position{line: 320, col: 5, offset: 7638},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 320, col: 5, offset: 7638},
									val:        "sort",
									ignoreCase: true,
								},
								&labeledExpr{
									pos:   position{line: 320, col: 13, offset: 7646},
									label: "rev",
									expr: &zeroOrOneExpr{
										pos: position{line: 320, col: 17, offset: 7650},
										expr: &seqExpr{
											pos: position{line: 320, col: 18, offset: 7651},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 320, col: 18, offset: 7651},
													name: "_",
												},
												&litMatcher{
													pos:        position{line: 320, col: 20, offset: 7653},
													val:        "-r",
													ignoreCase: false,
												},
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 320, col: 27, offset: 7660},
									label: "limit",
									expr: &zeroOrOneExpr{
										pos: position{line: 320, col: 33, offset: 7666},
										expr: &ruleRefExpr{
											pos:  position{line: 320, col: 33, offset: 7666},
											name: "procLimitArg",
										},
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 320, col: 48, offset: 7681},
									expr: &ruleRefExpr{
										pos:  position{line: 320, col: 48, offset: 7681},
										name: "_",
									},
								},
								&notExpr{
									pos: position{line: 320, col: 51, offset: 7684},
									expr: &litMatcher{
										pos:        position{line: 320, col: 52, offset: 7685},
										val:        "-r",
										ignoreCase: false,
									},
								},
								&labeledExpr{
									pos:   position{line: 320, col: 57, offset: 7690},
									label: "list",
									expr: &zeroOrOneExpr{
										pos: position{line: 320, col: 62, offset: 7695},
										expr: &ruleRefExpr{
											pos:  position{line: 320, col: 63, offset: 7696},
											name: "fieldExprList",
										},
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 325, col: 5, offset: 7826},
						run: (*parser).callonsort20,
						expr: &seqExpr{
							pos: position{line: 325, col: 5, offset: 7826},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 325, col: 5, offset: 7826},
									val:        "sort",
									ignoreCase: true,
								},
								&labeledExpr{
									pos:   position{line: 325, col: 13, offset: 7834},
									label: "limit",
									expr: &zeroOrOneExpr{
										pos: position{line: 325, col: 19, offset: 7840},
										expr: &ruleRefExpr{
											pos:  position{line: 325, col: 19, offset: 7840},
											name: "procLimitArg",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 325, col: 33, offset: 7854},
									label: "rev",
									expr: &zeroOrOneExpr{
										pos: position{line: 325, col: 37, offset: 7858},
										expr: &seqExpr{
											pos: position{line: 325, col: 38, offset: 7859},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 325, col: 38, offset: 7859},
													name: "_",
												},
												&litMatcher{
													pos:        position{line: 325, col: 40, offset: 7861},
													val:        "-r",
													ignoreCase: false,
												},
											},
										},
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 325, col: 47, offset: 7868},
									expr: &ruleRefExpr{
										pos:  position{line: 325, col: 47, offset: 7868},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 325, col: 50, offset: 7871},
									label: "list",
									expr: &zeroOrOneExpr{
										pos: position{line: 325, col: 55, offset: 7876},
										expr: &ruleRefExpr{
											pos:  position{line: 325, col: 56, offset: 7877},
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
			pos:  position{line: 331, col: 1, offset: 8004},
			expr: &actionExpr{
				pos: position{line: 332, col: 5, offset: 8012},
				run: (*parser).callontop1,
				expr: &seqExpr{
					pos: position{line: 332, col: 5, offset: 8012},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 332, col: 5, offset: 8012},
							val:        "top",
							ignoreCase: true,
						},
						&labeledExpr{
							pos:   position{line: 332, col: 12, offset: 8019},
							label: "limit",
							expr: &zeroOrOneExpr{
								pos: position{line: 332, col: 18, offset: 8025},
								expr: &ruleRefExpr{
									pos:  position{line: 332, col: 18, offset: 8025},
									name: "procLimitArg",
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 332, col: 32, offset: 8039},
							label: "flush",
							expr: &zeroOrOneExpr{
								pos: position{line: 332, col: 38, offset: 8045},
								expr: &seqExpr{
									pos: position{line: 332, col: 39, offset: 8046},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 332, col: 39, offset: 8046},
											name: "_",
										},
										&litMatcher{
											pos:        position{line: 332, col: 41, offset: 8048},
											val:        "-flush",
											ignoreCase: false,
										},
									},
								},
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 332, col: 52, offset: 8059},
							expr: &ruleRefExpr{
								pos:  position{line: 332, col: 52, offset: 8059},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 332, col: 55, offset: 8062},
							label: "list",
							expr: &zeroOrOneExpr{
								pos: position{line: 332, col: 60, offset: 8067},
								expr: &ruleRefExpr{
									pos:  position{line: 332, col: 61, offset: 8068},
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
			pos:  position{line: 336, col: 1, offset: 8139},
			expr: &actionExpr{
				pos: position{line: 337, col: 5, offset: 8156},
				run: (*parser).callonprocLimitArg1,
				expr: &seqExpr{
					pos: position{line: 337, col: 5, offset: 8156},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 337, col: 5, offset: 8156},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 337, col: 7, offset: 8158},
							val:        "-limit",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 337, col: 16, offset: 8167},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 337, col: 18, offset: 8169},
							label: "limit",
							expr: &ruleRefExpr{
								pos:  position{line: 337, col: 24, offset: 8175},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "cut",
			pos:  position{line: 339, col: 1, offset: 8206},
			expr: &actionExpr{
				pos: position{line: 340, col: 5, offset: 8214},
				run: (*parser).calloncut1,
				expr: &seqExpr{
					pos: position{line: 340, col: 5, offset: 8214},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 340, col: 5, offset: 8214},
							val:        "cut",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 340, col: 12, offset: 8221},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 340, col: 14, offset: 8223},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 340, col: 19, offset: 8228},
								name: "fieldNameList",
							},
						},
					},
				},
			},
		},
		{
			name: "head",
			pos:  position{line: 341, col: 1, offset: 8276},
			expr: &choiceExpr{
				pos: position{line: 342, col: 5, offset: 8285},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 342, col: 5, offset: 8285},
						run: (*parser).callonhead2,
						expr: &seqExpr{
							pos: position{line: 342, col: 5, offset: 8285},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 342, col: 5, offset: 8285},
									val:        "head",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 342, col: 13, offset: 8293},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 342, col: 15, offset: 8295},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 342, col: 21, offset: 8301},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 343, col: 5, offset: 8349},
						run: (*parser).callonhead8,
						expr: &litMatcher{
							pos:        position{line: 343, col: 5, offset: 8349},
							val:        "head",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "tail",
			pos:  position{line: 344, col: 1, offset: 8389},
			expr: &choiceExpr{
				pos: position{line: 345, col: 5, offset: 8398},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 345, col: 5, offset: 8398},
						run: (*parser).callontail2,
						expr: &seqExpr{
							pos: position{line: 345, col: 5, offset: 8398},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 345, col: 5, offset: 8398},
									val:        "tail",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 345, col: 13, offset: 8406},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 345, col: 15, offset: 8408},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 345, col: 21, offset: 8414},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 346, col: 5, offset: 8462},
						run: (*parser).callontail8,
						expr: &litMatcher{
							pos:        position{line: 346, col: 5, offset: 8462},
							val:        "tail",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "filter",
			pos:  position{line: 348, col: 1, offset: 8503},
			expr: &actionExpr{
				pos: position{line: 349, col: 5, offset: 8514},
				run: (*parser).callonfilter1,
				expr: &seqExpr{
					pos: position{line: 349, col: 5, offset: 8514},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 349, col: 5, offset: 8514},
							val:        "filter",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 349, col: 15, offset: 8524},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 349, col: 17, offset: 8526},
							label: "expr",
							expr: &ruleRefExpr{
								pos:  position{line: 349, col: 22, offset: 8531},
								name: "searchExpr",
							},
						},
					},
				},
			},
		},
		{
			name: "uniq",
			pos:  position{line: 352, col: 1, offset: 8589},
			expr: &choiceExpr{
				pos: position{line: 353, col: 5, offset: 8598},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 353, col: 5, offset: 8598},
						run: (*parser).callonuniq2,
						expr: &seqExpr{
							pos: position{line: 353, col: 5, offset: 8598},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 353, col: 5, offset: 8598},
									val:        "uniq",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 353, col: 13, offset: 8606},
									name: "_",
								},
								&litMatcher{
									pos:        position{line: 353, col: 15, offset: 8608},
									val:        "-c",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 356, col: 5, offset: 8662},
						run: (*parser).callonuniq7,
						expr: &litMatcher{
							pos:        position{line: 356, col: 5, offset: 8662},
							val:        "uniq",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "duration",
			pos:  position{line: 360, col: 1, offset: 8717},
			expr: &choiceExpr{
				pos: position{line: 361, col: 5, offset: 8730},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 361, col: 5, offset: 8730},
						name: "seconds",
					},
					&ruleRefExpr{
						pos:  position{line: 362, col: 5, offset: 8742},
						name: "minutes",
					},
					&ruleRefExpr{
						pos:  position{line: 363, col: 5, offset: 8754},
						name: "hours",
					},
					&seqExpr{
						pos: position{line: 364, col: 5, offset: 8764},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 364, col: 5, offset: 8764},
								name: "hours",
							},
							&ruleRefExpr{
								pos:  position{line: 364, col: 11, offset: 8770},
								name: "_",
							},
							&litMatcher{
								pos:        position{line: 364, col: 13, offset: 8772},
								val:        "and",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 364, col: 19, offset: 8778},
								name: "_",
							},
							&ruleRefExpr{
								pos:  position{line: 364, col: 21, offset: 8780},
								name: "minutes",
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 365, col: 5, offset: 8792},
						name: "days",
					},
					&ruleRefExpr{
						pos:  position{line: 366, col: 5, offset: 8801},
						name: "weeks",
					},
				},
			},
		},
		{
			name: "sec_abbrev",
			pos:  position{line: 368, col: 1, offset: 8808},
			expr: &choiceExpr{
				pos: position{line: 369, col: 5, offset: 8823},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 369, col: 5, offset: 8823},
						val:        "seconds",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 370, col: 5, offset: 8837},
						val:        "second",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 371, col: 5, offset: 8850},
						val:        "secs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 372, col: 5, offset: 8861},
						val:        "sec",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 373, col: 5, offset: 8871},
						val:        "s",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "min_abbrev",
			pos:  position{line: 375, col: 1, offset: 8876},
			expr: &choiceExpr{
				pos: position{line: 376, col: 5, offset: 8891},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 376, col: 5, offset: 8891},
						val:        "minutes",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 377, col: 5, offset: 8905},
						val:        "minute",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 378, col: 5, offset: 8918},
						val:        "mins",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 379, col: 5, offset: 8929},
						val:        "min",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 380, col: 5, offset: 8939},
						val:        "m",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "hour_abbrev",
			pos:  position{line: 382, col: 1, offset: 8944},
			expr: &choiceExpr{
				pos: position{line: 383, col: 5, offset: 8960},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 383, col: 5, offset: 8960},
						val:        "hours",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 384, col: 5, offset: 8972},
						val:        "hrs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 385, col: 5, offset: 8982},
						val:        "hr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 386, col: 5, offset: 8991},
						val:        "h",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 387, col: 5, offset: 8999},
						val:        "hour",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "day_abbrev",
			pos:  position{line: 389, col: 1, offset: 9007},
			expr: &choiceExpr{
				pos: position{line: 389, col: 14, offset: 9020},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 389, col: 14, offset: 9020},
						val:        "days",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 389, col: 21, offset: 9027},
						val:        "day",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 389, col: 27, offset: 9033},
						val:        "d",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "week_abbrev",
			pos:  position{line: 390, col: 1, offset: 9037},
			expr: &choiceExpr{
				pos: position{line: 390, col: 15, offset: 9051},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 390, col: 15, offset: 9051},
						val:        "weeks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 390, col: 23, offset: 9059},
						val:        "week",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 390, col: 30, offset: 9066},
						val:        "wks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 390, col: 36, offset: 9072},
						val:        "wk",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 390, col: 41, offset: 9077},
						val:        "w",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "seconds",
			pos:  position{line: 392, col: 1, offset: 9082},
			expr: &choiceExpr{
				pos: position{line: 393, col: 5, offset: 9094},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 393, col: 5, offset: 9094},
						run: (*parser).callonseconds2,
						expr: &litMatcher{
							pos:        position{line: 393, col: 5, offset: 9094},
							val:        "second",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 394, col: 5, offset: 9139},
						run: (*parser).callonseconds4,
						expr: &seqExpr{
							pos: position{line: 394, col: 5, offset: 9139},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 394, col: 5, offset: 9139},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 394, col: 9, offset: 9143},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 394, col: 16, offset: 9150},
									expr: &ruleRefExpr{
										pos:  position{line: 394, col: 16, offset: 9150},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 394, col: 19, offset: 9153},
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
			pos:  position{line: 396, col: 1, offset: 9199},
			expr: &choiceExpr{
				pos: position{line: 397, col: 5, offset: 9211},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 397, col: 5, offset: 9211},
						run: (*parser).callonminutes2,
						expr: &litMatcher{
							pos:        position{line: 397, col: 5, offset: 9211},
							val:        "minute",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 398, col: 5, offset: 9257},
						run: (*parser).callonminutes4,
						expr: &seqExpr{
							pos: position{line: 398, col: 5, offset: 9257},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 398, col: 5, offset: 9257},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 398, col: 9, offset: 9261},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 398, col: 16, offset: 9268},
									expr: &ruleRefExpr{
										pos:  position{line: 398, col: 16, offset: 9268},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 398, col: 19, offset: 9271},
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
			pos:  position{line: 400, col: 1, offset: 9326},
			expr: &choiceExpr{
				pos: position{line: 401, col: 5, offset: 9336},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 401, col: 5, offset: 9336},
						run: (*parser).callonhours2,
						expr: &litMatcher{
							pos:        position{line: 401, col: 5, offset: 9336},
							val:        "hour",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 402, col: 5, offset: 9382},
						run: (*parser).callonhours4,
						expr: &seqExpr{
							pos: position{line: 402, col: 5, offset: 9382},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 402, col: 5, offset: 9382},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 402, col: 9, offset: 9386},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 402, col: 16, offset: 9393},
									expr: &ruleRefExpr{
										pos:  position{line: 402, col: 16, offset: 9393},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 402, col: 19, offset: 9396},
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
			pos:  position{line: 404, col: 1, offset: 9454},
			expr: &choiceExpr{
				pos: position{line: 405, col: 5, offset: 9463},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 405, col: 5, offset: 9463},
						run: (*parser).callondays2,
						expr: &litMatcher{
							pos:        position{line: 405, col: 5, offset: 9463},
							val:        "day",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 406, col: 5, offset: 9511},
						run: (*parser).callondays4,
						expr: &seqExpr{
							pos: position{line: 406, col: 5, offset: 9511},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 406, col: 5, offset: 9511},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 406, col: 9, offset: 9515},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 406, col: 16, offset: 9522},
									expr: &ruleRefExpr{
										pos:  position{line: 406, col: 16, offset: 9522},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 406, col: 19, offset: 9525},
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
			pos:  position{line: 408, col: 1, offset: 9585},
			expr: &actionExpr{
				pos: position{line: 409, col: 5, offset: 9595},
				run: (*parser).callonweeks1,
				expr: &seqExpr{
					pos: position{line: 409, col: 5, offset: 9595},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 409, col: 5, offset: 9595},
							label: "num",
							expr: &ruleRefExpr{
								pos:  position{line: 409, col: 9, offset: 9599},
								name: "number",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 409, col: 16, offset: 9606},
							expr: &ruleRefExpr{
								pos:  position{line: 409, col: 16, offset: 9606},
								name: "_",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 409, col: 19, offset: 9609},
							name: "week_abbrev",
						},
					},
				},
			},
		},
		{
			name: "number",
			pos:  position{line: 411, col: 1, offset: 9672},
			expr: &ruleRefExpr{
				pos:  position{line: 411, col: 10, offset: 9681},
				name: "integer",
			},
		},
		{
			name: "addr",
			pos:  position{line: 415, col: 1, offset: 9719},
			expr: &actionExpr{
				pos: position{line: 416, col: 5, offset: 9728},
				run: (*parser).callonaddr1,
				expr: &labeledExpr{
					pos:   position{line: 416, col: 5, offset: 9728},
					label: "a",
					expr: &seqExpr{
						pos: position{line: 416, col: 8, offset: 9731},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 416, col: 8, offset: 9731},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 416, col: 16, offset: 9739},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 416, col: 20, offset: 9743},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 416, col: 28, offset: 9751},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 416, col: 32, offset: 9755},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 416, col: 40, offset: 9763},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 416, col: 44, offset: 9767},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "port",
			pos:  position{line: 418, col: 1, offset: 9808},
			expr: &actionExpr{
				pos: position{line: 419, col: 5, offset: 9817},
				run: (*parser).callonport1,
				expr: &seqExpr{
					pos: position{line: 419, col: 5, offset: 9817},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 419, col: 5, offset: 9817},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 419, col: 9, offset: 9821},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 419, col: 11, offset: 9823},
								name: "sinteger",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6addr",
			pos:  position{line: 423, col: 1, offset: 9982},
			expr: &choiceExpr{
				pos: position{line: 424, col: 5, offset: 9994},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 424, col: 5, offset: 9994},
						run: (*parser).callonip6addr2,
						expr: &seqExpr{
							pos: position{line: 424, col: 5, offset: 9994},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 424, col: 5, offset: 9994},
									label: "a",
									expr: &oneOrMoreExpr{
										pos: position{line: 424, col: 7, offset: 9996},
										expr: &ruleRefExpr{
											pos:  position{line: 424, col: 8, offset: 9997},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 424, col: 20, offset: 10009},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 424, col: 22, offset: 10011},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 427, col: 5, offset: 10075},
						run: (*parser).callonip6addr9,
						expr: &seqExpr{
							pos: position{line: 427, col: 5, offset: 10075},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 427, col: 5, offset: 10075},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 427, col: 7, offset: 10077},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 427, col: 11, offset: 10081},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 427, col: 13, offset: 10083},
										expr: &ruleRefExpr{
											pos:  position{line: 427, col: 14, offset: 10084},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 427, col: 25, offset: 10095},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 427, col: 30, offset: 10100},
									label: "d",
									expr: &zeroOrMoreExpr{
										pos: position{line: 427, col: 32, offset: 10102},
										expr: &ruleRefExpr{
											pos:  position{line: 427, col: 33, offset: 10103},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 427, col: 45, offset: 10115},
									label: "e",
									expr: &ruleRefExpr{
										pos:  position{line: 427, col: 47, offset: 10117},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 430, col: 5, offset: 10216},
						run: (*parser).callonip6addr22,
						expr: &seqExpr{
							pos: position{line: 430, col: 5, offset: 10216},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 430, col: 5, offset: 10216},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 430, col: 10, offset: 10221},
									label: "a",
									expr: &zeroOrMoreExpr{
										pos: position{line: 430, col: 12, offset: 10223},
										expr: &ruleRefExpr{
											pos:  position{line: 430, col: 13, offset: 10224},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 430, col: 25, offset: 10236},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 430, col: 27, offset: 10238},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 433, col: 5, offset: 10309},
						run: (*parser).callonip6addr30,
						expr: &seqExpr{
							pos: position{line: 433, col: 5, offset: 10309},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 433, col: 5, offset: 10309},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 433, col: 7, offset: 10311},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 433, col: 11, offset: 10315},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 433, col: 13, offset: 10317},
										expr: &ruleRefExpr{
											pos:  position{line: 433, col: 14, offset: 10318},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 433, col: 25, offset: 10329},
									val:        "::",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 436, col: 5, offset: 10397},
						run: (*parser).callonip6addr38,
						expr: &litMatcher{
							pos:        position{line: 436, col: 5, offset: 10397},
							val:        "::",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "ip6tail",
			pos:  position{line: 440, col: 1, offset: 10434},
			expr: &choiceExpr{
				pos: position{line: 441, col: 5, offset: 10446},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 441, col: 5, offset: 10446},
						name: "addr",
					},
					&ruleRefExpr{
						pos:  position{line: 442, col: 5, offset: 10455},
						name: "h16",
					},
				},
			},
		},
		{
			name: "h_append",
			pos:  position{line: 444, col: 1, offset: 10460},
			expr: &actionExpr{
				pos: position{line: 444, col: 12, offset: 10471},
				run: (*parser).callonh_append1,
				expr: &seqExpr{
					pos: position{line: 444, col: 12, offset: 10471},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 444, col: 12, offset: 10471},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 444, col: 16, offset: 10475},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 444, col: 18, offset: 10477},
								name: "h16",
							},
						},
					},
				},
			},
		},
		{
			name: "h_prepend",
			pos:  position{line: 445, col: 1, offset: 10514},
			expr: &actionExpr{
				pos: position{line: 445, col: 13, offset: 10526},
				run: (*parser).callonh_prepend1,
				expr: &seqExpr{
					pos: position{line: 445, col: 13, offset: 10526},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 445, col: 13, offset: 10526},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 445, col: 15, offset: 10528},
								name: "h16",
							},
						},
						&litMatcher{
							pos:        position{line: 445, col: 19, offset: 10532},
							val:        ":",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "sub_addr",
			pos:  position{line: 447, col: 1, offset: 10570},
			expr: &choiceExpr{
				pos: position{line: 448, col: 5, offset: 10583},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 448, col: 5, offset: 10583},
						name: "addr",
					},
					&actionExpr{
						pos: position{line: 449, col: 5, offset: 10592},
						run: (*parser).callonsub_addr3,
						expr: &labeledExpr{
							pos:   position{line: 449, col: 5, offset: 10592},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 449, col: 8, offset: 10595},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 449, col: 8, offset: 10595},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 449, col: 16, offset: 10603},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 449, col: 20, offset: 10607},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 449, col: 28, offset: 10615},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 449, col: 32, offset: 10619},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 450, col: 5, offset: 10671},
						run: (*parser).callonsub_addr11,
						expr: &labeledExpr{
							pos:   position{line: 450, col: 5, offset: 10671},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 450, col: 8, offset: 10674},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 450, col: 8, offset: 10674},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 450, col: 16, offset: 10682},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 450, col: 20, offset: 10686},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 451, col: 5, offset: 10740},
						run: (*parser).callonsub_addr17,
						expr: &labeledExpr{
							pos:   position{line: 451, col: 5, offset: 10740},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 451, col: 7, offset: 10742},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "subnet",
			pos:  position{line: 453, col: 1, offset: 10793},
			expr: &actionExpr{
				pos: position{line: 454, col: 5, offset: 10804},
				run: (*parser).callonsubnet1,
				expr: &seqExpr{
					pos: position{line: 454, col: 5, offset: 10804},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 454, col: 5, offset: 10804},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 454, col: 7, offset: 10806},
								name: "sub_addr",
							},
						},
						&litMatcher{
							pos:        position{line: 454, col: 16, offset: 10815},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 454, col: 20, offset: 10819},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 454, col: 22, offset: 10821},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6subnet",
			pos:  position{line: 458, col: 1, offset: 10897},
			expr: &actionExpr{
				pos: position{line: 459, col: 5, offset: 10911},
				run: (*parser).callonip6subnet1,
				expr: &seqExpr{
					pos: position{line: 459, col: 5, offset: 10911},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 459, col: 5, offset: 10911},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 459, col: 7, offset: 10913},
								name: "ip6addr",
							},
						},
						&litMatcher{
							pos:        position{line: 459, col: 15, offset: 10921},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 459, col: 19, offset: 10925},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 459, col: 21, offset: 10927},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "integer",
			pos:  position{line: 463, col: 1, offset: 10993},
			expr: &actionExpr{
				pos: position{line: 464, col: 5, offset: 11005},
				run: (*parser).calloninteger1,
				expr: &labeledExpr{
					pos:   position{line: 464, col: 5, offset: 11005},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 464, col: 7, offset: 11007},
						name: "sinteger",
					},
				},
			},
		},
		{
			name: "sinteger",
			pos:  position{line: 468, col: 1, offset: 11051},
			expr: &actionExpr{
				pos: position{line: 469, col: 5, offset: 11064},
				run: (*parser).callonsinteger1,
				expr: &labeledExpr{
					pos:   position{line: 469, col: 5, offset: 11064},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 469, col: 11, offset: 11070},
						expr: &charClassMatcher{
							pos:        position{line: 469, col: 11, offset: 11070},
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
			pos:  position{line: 473, col: 1, offset: 11115},
			expr: &actionExpr{
				pos: position{line: 474, col: 5, offset: 11126},
				run: (*parser).callondouble1,
				expr: &labeledExpr{
					pos:   position{line: 474, col: 5, offset: 11126},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 474, col: 7, offset: 11128},
						name: "sdouble",
					},
				},
			},
		},
		{
			name: "sdouble",
			pos:  position{line: 478, col: 1, offset: 11175},
			expr: &choiceExpr{
				pos: position{line: 479, col: 5, offset: 11187},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 479, col: 5, offset: 11187},
						run: (*parser).callonsdouble2,
						expr: &seqExpr{
							pos: position{line: 479, col: 5, offset: 11187},
							exprs: []interface{}{
								&oneOrMoreExpr{
									pos: position{line: 479, col: 5, offset: 11187},
									expr: &ruleRefExpr{
										pos:  position{line: 479, col: 5, offset: 11187},
										name: "doubleInteger",
									},
								},
								&litMatcher{
									pos:        position{line: 479, col: 20, offset: 11202},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 479, col: 24, offset: 11206},
									expr: &ruleRefExpr{
										pos:  position{line: 479, col: 24, offset: 11206},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 479, col: 37, offset: 11219},
									expr: &ruleRefExpr{
										pos:  position{line: 479, col: 37, offset: 11219},
										name: "exponentPart",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 482, col: 5, offset: 11278},
						run: (*parser).callonsdouble11,
						expr: &seqExpr{
							pos: position{line: 482, col: 5, offset: 11278},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 482, col: 5, offset: 11278},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 482, col: 9, offset: 11282},
									expr: &ruleRefExpr{
										pos:  position{line: 482, col: 9, offset: 11282},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 482, col: 22, offset: 11295},
									expr: &ruleRefExpr{
										pos:  position{line: 482, col: 22, offset: 11295},
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
			pos:  position{line: 486, col: 1, offset: 11351},
			expr: &choiceExpr{
				pos: position{line: 487, col: 5, offset: 11369},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 487, col: 5, offset: 11369},
						val:        "0",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 488, col: 5, offset: 11377},
						exprs: []interface{}{
							&charClassMatcher{
								pos:        position{line: 488, col: 5, offset: 11377},
								val:        "[1-9]",
								ranges:     []rune{'1', '9'},
								ignoreCase: false,
								inverted:   false,
							},
							&zeroOrMoreExpr{
								pos: position{line: 488, col: 11, offset: 11383},
								expr: &charClassMatcher{
									pos:        position{line: 488, col: 11, offset: 11383},
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
			pos:  position{line: 490, col: 1, offset: 11391},
			expr: &charClassMatcher{
				pos:        position{line: 490, col: 15, offset: 11405},
				val:        "[0-9]",
				ranges:     []rune{'0', '9'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "signedInteger",
			pos:  position{line: 492, col: 1, offset: 11412},
			expr: &seqExpr{
				pos: position{line: 492, col: 17, offset: 11428},
				exprs: []interface{}{
					&zeroOrOneExpr{
						pos: position{line: 492, col: 17, offset: 11428},
						expr: &charClassMatcher{
							pos:        position{line: 492, col: 17, offset: 11428},
							val:        "[+-]",
							chars:      []rune{'+', '-'},
							ignoreCase: false,
							inverted:   false,
						},
					},
					&oneOrMoreExpr{
						pos: position{line: 492, col: 23, offset: 11434},
						expr: &ruleRefExpr{
							pos:  position{line: 492, col: 23, offset: 11434},
							name: "doubleDigit",
						},
					},
				},
			},
		},
		{
			name: "exponentPart",
			pos:  position{line: 494, col: 1, offset: 11448},
			expr: &seqExpr{
				pos: position{line: 494, col: 16, offset: 11463},
				exprs: []interface{}{
					&litMatcher{
						pos:        position{line: 494, col: 16, offset: 11463},
						val:        "e",
						ignoreCase: true,
					},
					&ruleRefExpr{
						pos:  position{line: 494, col: 21, offset: 11468},
						name: "signedInteger",
					},
				},
			},
		},
		{
			name: "h16",
			pos:  position{line: 496, col: 1, offset: 11483},
			expr: &actionExpr{
				pos: position{line: 496, col: 7, offset: 11489},
				run: (*parser).callonh161,
				expr: &labeledExpr{
					pos:   position{line: 496, col: 7, offset: 11489},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 496, col: 13, offset: 11495},
						expr: &ruleRefExpr{
							pos:  position{line: 496, col: 13, offset: 11495},
							name: "hexdigit",
						},
					},
				},
			},
		},
		{
			name: "hexdigit",
			pos:  position{line: 498, col: 1, offset: 11537},
			expr: &charClassMatcher{
				pos:        position{line: 498, col: 12, offset: 11548},
				val:        "[0-9a-fA-F]",
				ranges:     []rune{'0', '9', 'a', 'f', 'A', 'F'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name:        "boomWord",
			displayName: "\"boomWord\"",
			pos:         position{line: 500, col: 1, offset: 11561},
			expr: &actionExpr{
				pos: position{line: 500, col: 23, offset: 11583},
				run: (*parser).callonboomWord1,
				expr: &labeledExpr{
					pos:   position{line: 500, col: 23, offset: 11583},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 500, col: 29, offset: 11589},
						expr: &ruleRefExpr{
							pos:  position{line: 500, col: 29, offset: 11589},
							name: "boomWordPart",
						},
					},
				},
			},
		},
		{
			name: "boomWordPart",
			pos:  position{line: 502, col: 1, offset: 11637},
			expr: &choiceExpr{
				pos: position{line: 503, col: 5, offset: 11654},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 503, col: 5, offset: 11654},
						run: (*parser).callonboomWordPart2,
						expr: &seqExpr{
							pos: position{line: 503, col: 5, offset: 11654},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 503, col: 5, offset: 11654},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 503, col: 10, offset: 11659},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 503, col: 12, offset: 11661},
										name: "escapeSequence",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 504, col: 5, offset: 11699},
						run: (*parser).callonboomWordPart7,
						expr: &seqExpr{
							pos: position{line: 504, col: 5, offset: 11699},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 504, col: 5, offset: 11699},
									expr: &choiceExpr{
										pos: position{line: 504, col: 7, offset: 11701},
										alternatives: []interface{}{
											&charClassMatcher{
												pos:        position{line: 504, col: 7, offset: 11701},
												val:        "[\\x00-\\x1F\\x5C(),!><=\\x22|\\x27;]",
												chars:      []rune{'\\', '(', ')', ',', '!', '>', '<', '=', '"', '|', '\'', ';'},
												ranges:     []rune{'\x00', '\x1f'},
												ignoreCase: false,
												inverted:   false,
											},
											&ruleRefExpr{
												pos:  position{line: 504, col: 42, offset: 11736},
												name: "ws",
											},
										},
									},
								},
								&anyMatcher{
									line: 504, col: 46, offset: 11740,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "quotedString",
			pos:  position{line: 506, col: 1, offset: 11774},
			expr: &choiceExpr{
				pos: position{line: 507, col: 5, offset: 11791},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 507, col: 5, offset: 11791},
						run: (*parser).callonquotedString2,
						expr: &seqExpr{
							pos: position{line: 507, col: 5, offset: 11791},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 507, col: 5, offset: 11791},
									val:        "\"",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 507, col: 9, offset: 11795},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 507, col: 11, offset: 11797},
										expr: &ruleRefExpr{
											pos:  position{line: 507, col: 11, offset: 11797},
											name: "doubleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 507, col: 29, offset: 11815},
									val:        "\"",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 508, col: 5, offset: 11852},
						run: (*parser).callonquotedString9,
						expr: &seqExpr{
							pos: position{line: 508, col: 5, offset: 11852},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 508, col: 5, offset: 11852},
									val:        "'",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 508, col: 9, offset: 11856},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 508, col: 11, offset: 11858},
										expr: &ruleRefExpr{
											pos:  position{line: 508, col: 11, offset: 11858},
											name: "singleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 508, col: 29, offset: 11876},
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
			pos:  position{line: 510, col: 1, offset: 11910},
			expr: &choiceExpr{
				pos: position{line: 511, col: 5, offset: 11931},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 511, col: 5, offset: 11931},
						run: (*parser).callondoubleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 511, col: 5, offset: 11931},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 511, col: 5, offset: 11931},
									expr: &choiceExpr{
										pos: position{line: 511, col: 7, offset: 11933},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 511, col: 7, offset: 11933},
												val:        "\"",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 511, col: 13, offset: 11939},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 511, col: 26, offset: 11952,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 512, col: 5, offset: 11989},
						run: (*parser).callondoubleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 512, col: 5, offset: 11989},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 512, col: 5, offset: 11989},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 512, col: 10, offset: 11994},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 512, col: 12, offset: 11996},
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
			pos:  position{line: 514, col: 1, offset: 12030},
			expr: &choiceExpr{
				pos: position{line: 515, col: 5, offset: 12051},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 515, col: 5, offset: 12051},
						run: (*parser).callonsingleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 515, col: 5, offset: 12051},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 515, col: 5, offset: 12051},
									expr: &choiceExpr{
										pos: position{line: 515, col: 7, offset: 12053},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 515, col: 7, offset: 12053},
												val:        "'",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 515, col: 13, offset: 12059},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 515, col: 26, offset: 12072,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 516, col: 5, offset: 12109},
						run: (*parser).callonsingleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 516, col: 5, offset: 12109},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 516, col: 5, offset: 12109},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 516, col: 10, offset: 12114},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 516, col: 12, offset: 12116},
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
			pos:  position{line: 518, col: 1, offset: 12150},
			expr: &choiceExpr{
				pos: position{line: 519, col: 5, offset: 12169},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 519, col: 5, offset: 12169},
						run: (*parser).callonescapeSequence2,
						expr: &seqExpr{
							pos: position{line: 519, col: 5, offset: 12169},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 519, col: 5, offset: 12169},
									val:        "x",
									ignoreCase: false,
								},
								&ruleRefExpr{
									pos:  position{line: 519, col: 9, offset: 12173},
									name: "hexdigit",
								},
								&ruleRefExpr{
									pos:  position{line: 519, col: 18, offset: 12182},
									name: "hexdigit",
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 520, col: 5, offset: 12233},
						name: "singleCharEscape",
					},
					&ruleRefExpr{
						pos:  position{line: 521, col: 5, offset: 12254},
						name: "unicodeEscape",
					},
				},
			},
		},
		{
			name: "singleCharEscape",
			pos:  position{line: 523, col: 1, offset: 12269},
			expr: &choiceExpr{
				pos: position{line: 524, col: 5, offset: 12290},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 524, col: 5, offset: 12290},
						val:        "'",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 525, col: 5, offset: 12298},
						val:        "\"",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 526, col: 5, offset: 12306},
						val:        "\\",
						ignoreCase: false,
					},
					&actionExpr{
						pos: position{line: 527, col: 5, offset: 12315},
						run: (*parser).callonsingleCharEscape5,
						expr: &litMatcher{
							pos:        position{line: 527, col: 5, offset: 12315},
							val:        "b",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 528, col: 5, offset: 12344},
						run: (*parser).callonsingleCharEscape7,
						expr: &litMatcher{
							pos:        position{line: 528, col: 5, offset: 12344},
							val:        "f",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 529, col: 5, offset: 12373},
						run: (*parser).callonsingleCharEscape9,
						expr: &litMatcher{
							pos:        position{line: 529, col: 5, offset: 12373},
							val:        "n",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 530, col: 5, offset: 12402},
						run: (*parser).callonsingleCharEscape11,
						expr: &litMatcher{
							pos:        position{line: 530, col: 5, offset: 12402},
							val:        "r",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 531, col: 5, offset: 12431},
						run: (*parser).callonsingleCharEscape13,
						expr: &litMatcher{
							pos:        position{line: 531, col: 5, offset: 12431},
							val:        "t",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 532, col: 5, offset: 12460},
						run: (*parser).callonsingleCharEscape15,
						expr: &litMatcher{
							pos:        position{line: 532, col: 5, offset: 12460},
							val:        "v",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "unicodeEscape",
			pos:  position{line: 534, col: 1, offset: 12486},
			expr: &seqExpr{
				pos: position{line: 535, col: 5, offset: 12504},
				exprs: []interface{}{
					&litMatcher{
						pos:        position{line: 535, col: 5, offset: 12504},
						val:        "u",
						ignoreCase: false,
					},
					&ruleRefExpr{
						pos:  position{line: 535, col: 9, offset: 12508},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 535, col: 18, offset: 12517},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 535, col: 27, offset: 12526},
						name: "hexdigit",
					},
					&ruleRefExpr{
						pos:  position{line: 535, col: 36, offset: 12535},
						name: "hexdigit",
					},
				},
			},
		},
		{
			name: "reString",
			pos:  position{line: 537, col: 1, offset: 12545},
			expr: &actionExpr{
				pos: position{line: 538, col: 5, offset: 12558},
				run: (*parser).callonreString1,
				expr: &seqExpr{
					pos: position{line: 538, col: 5, offset: 12558},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 538, col: 5, offset: 12558},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 538, col: 9, offset: 12562},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 538, col: 11, offset: 12564},
								name: "reBody",
							},
						},
						&litMatcher{
							pos:        position{line: 538, col: 18, offset: 12571},
							val:        "/",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reBody",
			pos:  position{line: 540, col: 1, offset: 12594},
			expr: &actionExpr{
				pos: position{line: 541, col: 5, offset: 12605},
				run: (*parser).callonreBody1,
				expr: &oneOrMoreExpr{
					pos: position{line: 541, col: 5, offset: 12605},
					expr: &choiceExpr{
						pos: position{line: 541, col: 6, offset: 12606},
						alternatives: []interface{}{
							&charClassMatcher{
								pos:        position{line: 541, col: 6, offset: 12606},
								val:        "[^/\\\\]",
								chars:      []rune{'/', '\\'},
								ignoreCase: false,
								inverted:   true,
							},
							&litMatcher{
								pos:        position{line: 541, col: 13, offset: 12613},
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
			pos:  position{line: 543, col: 1, offset: 12653},
			expr: &charClassMatcher{
				pos:        position{line: 544, col: 5, offset: 12669},
				val:        "[\\x00-\\x1f\\\\]",
				chars:      []rune{'\\'},
				ranges:     []rune{'\x00', '\x1f'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "ws",
			pos:  position{line: 546, col: 1, offset: 12684},
			expr: &choiceExpr{
				pos: position{line: 547, col: 5, offset: 12691},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 547, col: 5, offset: 12691},
						val:        "\t",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 548, col: 5, offset: 12700},
						val:        "\v",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 549, col: 5, offset: 12709},
						val:        "\f",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 550, col: 5, offset: 12718},
						val:        " ",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 551, col: 5, offset: 12726},
						val:        "\u00a0",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 552, col: 5, offset: 12739},
						val:        "\ufeff",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name:        "_",
			displayName: "\"whitespace\"",
			pos:         position{line: 554, col: 1, offset: 12749},
			expr: &oneOrMoreExpr{
				pos: position{line: 554, col: 18, offset: 12766},
				expr: &ruleRefExpr{
					pos:  position{line: 554, col: 18, offset: 12766},
					name: "ws",
				},
			},
		},
		{
			name: "EOF",
			pos:  position{line: 556, col: 1, offset: 12771},
			expr: &notExpr{
				pos: position{line: 556, col: 7, offset: 12777},
				expr: &anyMatcher{
					line: 556, col: 8, offset: 12778,
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

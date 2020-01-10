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
						expr: &seqExpr{
							pos: position{line: 74, col: 5, offset: 1764},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 74, col: 5, offset: 1764},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 74, col: 7, offset: 1766},
										name: "fieldExpr",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 74, col: 17, offset: 1776},
									expr: &ruleRefExpr{
										pos:  position{line: 74, col: 17, offset: 1776},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 74, col: 20, offset: 1779},
									label: "fieldComparator",
									expr: &ruleRefExpr{
										pos:  position{line: 74, col: 36, offset: 1795},
										name: "equalityToken",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 74, col: 50, offset: 1809},
									expr: &ruleRefExpr{
										pos:  position{line: 74, col: 50, offset: 1809},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 74, col: 53, offset: 1812},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 74, col: 55, offset: 1814},
										name: "searchValue",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 77, col: 5, offset: 1896},
						run: (*parser).callonsearchPred36,
						expr: &seqExpr{
							pos: position{line: 77, col: 5, offset: 1896},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 77, col: 5, offset: 1896},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 77, col: 7, offset: 1898},
										name: "searchValue",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 77, col: 19, offset: 1910},
									expr: &ruleRefExpr{
										pos:  position{line: 77, col: 19, offset: 1910},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 77, col: 22, offset: 1913},
									name: "inToken",
								},
								&zeroOrOneExpr{
									pos: position{line: 77, col: 30, offset: 1921},
									expr: &ruleRefExpr{
										pos:  position{line: 77, col: 30, offset: 1921},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 77, col: 33, offset: 1924},
									val:        "*",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 80, col: 5, offset: 1989},
						run: (*parser).callonsearchPred46,
						expr: &seqExpr{
							pos: position{line: 80, col: 5, offset: 1989},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 80, col: 5, offset: 1989},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 80, col: 7, offset: 1991},
										name: "searchValue",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 80, col: 19, offset: 2003},
									expr: &ruleRefExpr{
										pos:  position{line: 80, col: 19, offset: 2003},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 80, col: 22, offset: 2006},
									name: "inToken",
								},
								&zeroOrOneExpr{
									pos: position{line: 80, col: 30, offset: 2014},
									expr: &ruleRefExpr{
										pos:  position{line: 80, col: 30, offset: 2014},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 80, col: 33, offset: 2017},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 80, col: 35, offset: 2019},
										name: "fieldReference",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 83, col: 5, offset: 2093},
						run: (*parser).callonsearchPred57,
						expr: &labeledExpr{
							pos:   position{line: 83, col: 5, offset: 2093},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 83, col: 7, offset: 2095},
								name: "searchValue",
							},
						},
					},
				},
			},
		},
		{
			name: "searchValue",
			pos:  position{line: 97, col: 1, offset: 2790},
			expr: &choiceExpr{
				pos: position{line: 98, col: 5, offset: 2806},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 98, col: 5, offset: 2806},
						run: (*parser).callonsearchValue2,
						expr: &labeledExpr{
							pos:   position{line: 98, col: 5, offset: 2806},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 98, col: 7, offset: 2808},
								name: "quotedString",
							},
						},
					},
					&actionExpr{
						pos: position{line: 101, col: 5, offset: 2879},
						run: (*parser).callonsearchValue5,
						expr: &labeledExpr{
							pos:   position{line: 101, col: 5, offset: 2879},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 101, col: 7, offset: 2881},
								name: "reString",
							},
						},
					},
					&actionExpr{
						pos: position{line: 104, col: 5, offset: 2948},
						run: (*parser).callonsearchValue8,
						expr: &labeledExpr{
							pos:   position{line: 104, col: 5, offset: 2948},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 104, col: 7, offset: 2950},
								name: "port",
							},
						},
					},
					&actionExpr{
						pos: position{line: 107, col: 5, offset: 3009},
						run: (*parser).callonsearchValue11,
						expr: &labeledExpr{
							pos:   position{line: 107, col: 5, offset: 3009},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 107, col: 7, offset: 3011},
								name: "ip6subnet",
							},
						},
					},
					&actionExpr{
						pos: position{line: 110, col: 5, offset: 3079},
						run: (*parser).callonsearchValue14,
						expr: &labeledExpr{
							pos:   position{line: 110, col: 5, offset: 3079},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 110, col: 7, offset: 3081},
								name: "ip6addr",
							},
						},
					},
					&actionExpr{
						pos: position{line: 113, col: 5, offset: 3145},
						run: (*parser).callonsearchValue17,
						expr: &labeledExpr{
							pos:   position{line: 113, col: 5, offset: 3145},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 113, col: 7, offset: 3147},
								name: "subnet",
							},
						},
					},
					&actionExpr{
						pos: position{line: 116, col: 5, offset: 3212},
						run: (*parser).callonsearchValue20,
						expr: &labeledExpr{
							pos:   position{line: 116, col: 5, offset: 3212},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 116, col: 7, offset: 3214},
								name: "addr",
							},
						},
					},
					&actionExpr{
						pos: position{line: 119, col: 5, offset: 3275},
						run: (*parser).callonsearchValue23,
						expr: &labeledExpr{
							pos:   position{line: 119, col: 5, offset: 3275},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 119, col: 7, offset: 3277},
								name: "sdouble",
							},
						},
					},
					&actionExpr{
						pos: position{line: 122, col: 5, offset: 3343},
						run: (*parser).callonsearchValue26,
						expr: &seqExpr{
							pos: position{line: 122, col: 5, offset: 3343},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 122, col: 5, offset: 3343},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 122, col: 7, offset: 3345},
										name: "sinteger",
									},
								},
								&notExpr{
									pos: position{line: 122, col: 16, offset: 3354},
									expr: &ruleRefExpr{
										pos:  position{line: 122, col: 17, offset: 3355},
										name: "boomWord",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 125, col: 5, offset: 3419},
						run: (*parser).callonsearchValue32,
						expr: &seqExpr{
							pos: position{line: 125, col: 5, offset: 3419},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 125, col: 5, offset: 3419},
									expr: &seqExpr{
										pos: position{line: 125, col: 7, offset: 3421},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 125, col: 7, offset: 3421},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 125, col: 22, offset: 3436},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 125, col: 25, offset: 3439},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 125, col: 27, offset: 3441},
										name: "booleanLiteral",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 126, col: 5, offset: 3478},
						run: (*parser).callonsearchValue40,
						expr: &seqExpr{
							pos: position{line: 126, col: 5, offset: 3478},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 126, col: 5, offset: 3478},
									expr: &seqExpr{
										pos: position{line: 126, col: 7, offset: 3480},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 126, col: 7, offset: 3480},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 126, col: 22, offset: 3495},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 126, col: 25, offset: 3498},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 126, col: 27, offset: 3500},
										name: "unsetLiteral",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 127, col: 5, offset: 3535},
						run: (*parser).callonsearchValue48,
						expr: &seqExpr{
							pos: position{line: 127, col: 5, offset: 3535},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 127, col: 5, offset: 3535},
									expr: &seqExpr{
										pos: position{line: 127, col: 7, offset: 3537},
										exprs: []interface{}{
											&choiceExpr{
												pos: position{line: 127, col: 8, offset: 3538},
												alternatives: []interface{}{
													&ruleRefExpr{
														pos:  position{line: 127, col: 8, offset: 3538},
														name: "searchKeywords",
													},
													&litMatcher{
														pos:        position{line: 127, col: 25, offset: 3555},
														val:        "|",
														ignoreCase: false,
													},
												},
											},
											&ruleRefExpr{
												pos:  position{line: 127, col: 30, offset: 3560},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 127, col: 33, offset: 3563},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 127, col: 35, offset: 3565},
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
			pos:  position{line: 135, col: 1, offset: 3768},
			expr: &choiceExpr{
				pos: position{line: 136, col: 5, offset: 3787},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 136, col: 5, offset: 3787},
						name: "andToken",
					},
					&ruleRefExpr{
						pos:  position{line: 137, col: 5, offset: 3800},
						name: "orToken",
					},
					&ruleRefExpr{
						pos:  position{line: 138, col: 5, offset: 3812},
						name: "inToken",
					},
				},
			},
		},
		{
			name: "booleanLiteral",
			pos:  position{line: 140, col: 1, offset: 3821},
			expr: &choiceExpr{
				pos: position{line: 141, col: 5, offset: 3840},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 141, col: 5, offset: 3840},
						run: (*parser).callonbooleanLiteral2,
						expr: &litMatcher{
							pos:        position{line: 141, col: 5, offset: 3840},
							val:        "true",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 142, col: 5, offset: 3908},
						run: (*parser).callonbooleanLiteral4,
						expr: &litMatcher{
							pos:        position{line: 142, col: 5, offset: 3908},
							val:        "false",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "unsetLiteral",
			pos:  position{line: 144, col: 1, offset: 3974},
			expr: &actionExpr{
				pos: position{line: 145, col: 5, offset: 3991},
				run: (*parser).callonunsetLiteral1,
				expr: &litMatcher{
					pos:        position{line: 145, col: 5, offset: 3991},
					val:        "nil",
					ignoreCase: false,
				},
			},
		},
		{
			name: "procList",
			pos:  position{line: 147, col: 1, offset: 4052},
			expr: &actionExpr{
				pos: position{line: 148, col: 5, offset: 4065},
				run: (*parser).callonprocList1,
				expr: &seqExpr{
					pos: position{line: 148, col: 5, offset: 4065},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 148, col: 5, offset: 4065},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 148, col: 11, offset: 4071},
								name: "procChain",
							},
						},
						&labeledExpr{
							pos:   position{line: 148, col: 21, offset: 4081},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 148, col: 26, offset: 4086},
								expr: &ruleRefExpr{
									pos:  position{line: 148, col: 26, offset: 4086},
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
			pos:  position{line: 157, col: 1, offset: 4310},
			expr: &actionExpr{
				pos: position{line: 158, col: 5, offset: 4328},
				run: (*parser).callonparallelChain1,
				expr: &seqExpr{
					pos: position{line: 158, col: 5, offset: 4328},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 158, col: 5, offset: 4328},
							expr: &ruleRefExpr{
								pos:  position{line: 158, col: 5, offset: 4328},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 158, col: 8, offset: 4331},
							val:        ";",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 158, col: 12, offset: 4335},
							expr: &ruleRefExpr{
								pos:  position{line: 158, col: 12, offset: 4335},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 158, col: 15, offset: 4338},
							label: "ch",
							expr: &ruleRefExpr{
								pos:  position{line: 158, col: 18, offset: 4341},
								name: "procChain",
							},
						},
					},
				},
			},
		},
		{
			name: "proc",
			pos:  position{line: 160, col: 1, offset: 4391},
			expr: &choiceExpr{
				pos: position{line: 161, col: 5, offset: 4400},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 161, col: 5, offset: 4400},
						name: "simpleProc",
					},
					&ruleRefExpr{
						pos:  position{line: 162, col: 5, offset: 4415},
						name: "reducerProc",
					},
					&actionExpr{
						pos: position{line: 163, col: 5, offset: 4431},
						run: (*parser).callonproc4,
						expr: &seqExpr{
							pos: position{line: 163, col: 5, offset: 4431},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 163, col: 5, offset: 4431},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 163, col: 9, offset: 4435},
									expr: &ruleRefExpr{
										pos:  position{line: 163, col: 9, offset: 4435},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 163, col: 12, offset: 4438},
									label: "proc",
									expr: &ruleRefExpr{
										pos:  position{line: 163, col: 17, offset: 4443},
										name: "procList",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 163, col: 26, offset: 4452},
									expr: &ruleRefExpr{
										pos:  position{line: 163, col: 26, offset: 4452},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 163, col: 29, offset: 4455},
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
			pos:  position{line: 167, col: 1, offset: 4491},
			expr: &actionExpr{
				pos: position{line: 168, col: 5, offset: 4503},
				run: (*parser).callongroupBy1,
				expr: &seqExpr{
					pos: position{line: 168, col: 5, offset: 4503},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 168, col: 5, offset: 4503},
							val:        "by",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 168, col: 11, offset: 4509},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 168, col: 13, offset: 4511},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 168, col: 18, offset: 4516},
								name: "fieldExprList",
							},
						},
					},
				},
			},
		},
		{
			name: "everyDur",
			pos:  position{line: 170, col: 1, offset: 4552},
			expr: &actionExpr{
				pos: position{line: 171, col: 5, offset: 4565},
				run: (*parser).calloneveryDur1,
				expr: &seqExpr{
					pos: position{line: 171, col: 5, offset: 4565},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 171, col: 5, offset: 4565},
							val:        "every",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 171, col: 14, offset: 4574},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 171, col: 16, offset: 4576},
							label: "dur",
							expr: &ruleRefExpr{
								pos:  position{line: 171, col: 20, offset: 4580},
								name: "duration",
							},
						},
					},
				},
			},
		},
		{
			name: "equalityToken",
			pos:  position{line: 173, col: 1, offset: 4610},
			expr: &choiceExpr{
				pos: position{line: 174, col: 5, offset: 4628},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 174, col: 5, offset: 4628},
						run: (*parser).callonequalityToken2,
						expr: &litMatcher{
							pos:        position{line: 174, col: 5, offset: 4628},
							val:        "=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 175, col: 5, offset: 4658},
						run: (*parser).callonequalityToken4,
						expr: &litMatcher{
							pos:        position{line: 175, col: 5, offset: 4658},
							val:        "!=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 176, col: 5, offset: 4690},
						run: (*parser).callonequalityToken6,
						expr: &litMatcher{
							pos:        position{line: 176, col: 5, offset: 4690},
							val:        "<=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 177, col: 5, offset: 4721},
						run: (*parser).callonequalityToken8,
						expr: &litMatcher{
							pos:        position{line: 177, col: 5, offset: 4721},
							val:        ">=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 178, col: 5, offset: 4752},
						run: (*parser).callonequalityToken10,
						expr: &litMatcher{
							pos:        position{line: 178, col: 5, offset: 4752},
							val:        "<",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 179, col: 5, offset: 4781},
						run: (*parser).callonequalityToken12,
						expr: &litMatcher{
							pos:        position{line: 179, col: 5, offset: 4781},
							val:        ">",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "types",
			pos:  position{line: 181, col: 1, offset: 4807},
			expr: &choiceExpr{
				pos: position{line: 182, col: 5, offset: 4817},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 182, col: 5, offset: 4817},
						val:        "bool",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 183, col: 5, offset: 4828},
						val:        "int",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 184, col: 5, offset: 4838},
						val:        "count",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 185, col: 5, offset: 4850},
						val:        "double",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 186, col: 5, offset: 4863},
						val:        "string",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 187, col: 5, offset: 4876},
						val:        "addr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 188, col: 5, offset: 4887},
						val:        "subnet",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 189, col: 5, offset: 4900},
						val:        "port",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "dash",
			pos:  position{line: 191, col: 1, offset: 4908},
			expr: &choiceExpr{
				pos: position{line: 191, col: 8, offset: 4915},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 191, col: 8, offset: 4915},
						val:        "-",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 191, col: 14, offset: 4921},
						val:        "—",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 191, col: 25, offset: 4932},
						val:        "–",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 191, col: 36, offset: 4943},
						exprs: []interface{}{
							&litMatcher{
								pos:        position{line: 191, col: 36, offset: 4943},
								val:        "to",
								ignoreCase: false,
							},
							&andExpr{
								pos: position{line: 191, col: 40, offset: 4947},
								expr: &ruleRefExpr{
									pos:  position{line: 191, col: 42, offset: 4949},
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
			pos:  position{line: 193, col: 1, offset: 4953},
			expr: &litMatcher{
				pos:        position{line: 193, col: 12, offset: 4964},
				val:        "and",
				ignoreCase: false,
			},
		},
		{
			name: "orToken",
			pos:  position{line: 194, col: 1, offset: 4970},
			expr: &litMatcher{
				pos:        position{line: 194, col: 11, offset: 4980},
				val:        "or",
				ignoreCase: false,
			},
		},
		{
			name: "inToken",
			pos:  position{line: 195, col: 1, offset: 4985},
			expr: &litMatcher{
				pos:        position{line: 195, col: 11, offset: 4995},
				val:        "in",
				ignoreCase: false,
			},
		},
		{
			name: "notToken",
			pos:  position{line: 196, col: 1, offset: 5000},
			expr: &litMatcher{
				pos:        position{line: 196, col: 12, offset: 5011},
				val:        "not",
				ignoreCase: false,
			},
		},
		{
			name: "fieldName",
			pos:  position{line: 198, col: 1, offset: 5018},
			expr: &actionExpr{
				pos: position{line: 198, col: 13, offset: 5030},
				run: (*parser).callonfieldName1,
				expr: &seqExpr{
					pos: position{line: 198, col: 13, offset: 5030},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 198, col: 13, offset: 5030},
							name: "fieldNameStart",
						},
						&zeroOrMoreExpr{
							pos: position{line: 198, col: 28, offset: 5045},
							expr: &ruleRefExpr{
								pos:  position{line: 198, col: 28, offset: 5045},
								name: "fieldNameRest",
							},
						},
					},
				},
			},
		},
		{
			name: "fieldNameStart",
			pos:  position{line: 200, col: 1, offset: 5092},
			expr: &charClassMatcher{
				pos:        position{line: 200, col: 18, offset: 5109},
				val:        "[A-Za-z_$]",
				chars:      []rune{'_', '$'},
				ranges:     []rune{'A', 'Z', 'a', 'z'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "fieldNameRest",
			pos:  position{line: 201, col: 1, offset: 5120},
			expr: &choiceExpr{
				pos: position{line: 201, col: 17, offset: 5136},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 201, col: 17, offset: 5136},
						name: "fieldNameStart",
					},
					&charClassMatcher{
						pos:        position{line: 201, col: 34, offset: 5153},
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
			pos:  position{line: 203, col: 1, offset: 5160},
			expr: &actionExpr{
				pos: position{line: 204, col: 4, offset: 5178},
				run: (*parser).callonfieldReference1,
				expr: &seqExpr{
					pos: position{line: 204, col: 4, offset: 5178},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 204, col: 4, offset: 5178},
							label: "base",
							expr: &ruleRefExpr{
								pos:  position{line: 204, col: 9, offset: 5183},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 204, col: 19, offset: 5193},
							label: "derefs",
							expr: &zeroOrMoreExpr{
								pos: position{line: 204, col: 26, offset: 5200},
								expr: &choiceExpr{
									pos: position{line: 205, col: 8, offset: 5209},
									alternatives: []interface{}{
										&actionExpr{
											pos: position{line: 205, col: 8, offset: 5209},
											run: (*parser).callonfieldReference8,
											expr: &seqExpr{
												pos: position{line: 205, col: 8, offset: 5209},
												exprs: []interface{}{
													&litMatcher{
														pos:        position{line: 205, col: 8, offset: 5209},
														val:        ".",
														ignoreCase: false,
													},
													&labeledExpr{
														pos:   position{line: 205, col: 12, offset: 5213},
														label: "field",
														expr: &ruleRefExpr{
															pos:  position{line: 205, col: 18, offset: 5219},
															name: "fieldName",
														},
													},
												},
											},
										},
										&actionExpr{
											pos: position{line: 206, col: 8, offset: 5300},
											run: (*parser).callonfieldReference13,
											expr: &seqExpr{
												pos: position{line: 206, col: 8, offset: 5300},
												exprs: []interface{}{
													&litMatcher{
														pos:        position{line: 206, col: 8, offset: 5300},
														val:        "[",
														ignoreCase: false,
													},
													&labeledExpr{
														pos:   position{line: 206, col: 12, offset: 5304},
														label: "index",
														expr: &ruleRefExpr{
															pos:  position{line: 206, col: 18, offset: 5310},
															name: "sinteger",
														},
													},
													&litMatcher{
														pos:        position{line: 206, col: 27, offset: 5319},
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
			pos:  position{line: 211, col: 1, offset: 5435},
			expr: &choiceExpr{
				pos: position{line: 212, col: 5, offset: 5449},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 212, col: 5, offset: 5449},
						run: (*parser).callonfieldExpr2,
						expr: &seqExpr{
							pos: position{line: 212, col: 5, offset: 5449},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 212, col: 5, offset: 5449},
									label: "op",
									expr: &ruleRefExpr{
										pos:  position{line: 212, col: 8, offset: 5452},
										name: "fieldOp",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 212, col: 16, offset: 5460},
									expr: &ruleRefExpr{
										pos:  position{line: 212, col: 16, offset: 5460},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 212, col: 19, offset: 5463},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 212, col: 23, offset: 5467},
									expr: &ruleRefExpr{
										pos:  position{line: 212, col: 23, offset: 5467},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 212, col: 26, offset: 5470},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 212, col: 32, offset: 5476},
										name: "fieldReference",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 212, col: 47, offset: 5491},
									expr: &ruleRefExpr{
										pos:  position{line: 212, col: 47, offset: 5491},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 212, col: 50, offset: 5494},
									val:        ")",
									ignoreCase: false,
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 215, col: 5, offset: 5558},
						name: "fieldReference",
					},
				},
			},
		},
		{
			name: "fieldOp",
			pos:  position{line: 217, col: 1, offset: 5574},
			expr: &actionExpr{
				pos: position{line: 218, col: 5, offset: 5586},
				run: (*parser).callonfieldOp1,
				expr: &litMatcher{
					pos:        position{line: 218, col: 5, offset: 5586},
					val:        "len",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldExprList",
			pos:  position{line: 220, col: 1, offset: 5616},
			expr: &actionExpr{
				pos: position{line: 221, col: 5, offset: 5634},
				run: (*parser).callonfieldExprList1,
				expr: &seqExpr{
					pos: position{line: 221, col: 5, offset: 5634},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 221, col: 5, offset: 5634},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 221, col: 11, offset: 5640},
								name: "fieldExpr",
							},
						},
						&labeledExpr{
							pos:   position{line: 221, col: 21, offset: 5650},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 221, col: 26, offset: 5655},
								expr: &seqExpr{
									pos: position{line: 221, col: 27, offset: 5656},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 221, col: 27, offset: 5656},
											expr: &ruleRefExpr{
												pos:  position{line: 221, col: 27, offset: 5656},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 221, col: 30, offset: 5659},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 221, col: 34, offset: 5663},
											expr: &ruleRefExpr{
												pos:  position{line: 221, col: 34, offset: 5663},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 221, col: 37, offset: 5666},
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
			pos:  position{line: 231, col: 1, offset: 5861},
			expr: &actionExpr{
				pos: position{line: 232, col: 5, offset: 5881},
				run: (*parser).callonfieldRefDotOnly1,
				expr: &seqExpr{
					pos: position{line: 232, col: 5, offset: 5881},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 232, col: 5, offset: 5881},
							label: "base",
							expr: &ruleRefExpr{
								pos:  position{line: 232, col: 10, offset: 5886},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 232, col: 20, offset: 5896},
							label: "refs",
							expr: &zeroOrMoreExpr{
								pos: position{line: 232, col: 25, offset: 5901},
								expr: &actionExpr{
									pos: position{line: 232, col: 26, offset: 5902},
									run: (*parser).callonfieldRefDotOnly7,
									expr: &seqExpr{
										pos: position{line: 232, col: 26, offset: 5902},
										exprs: []interface{}{
											&litMatcher{
												pos:        position{line: 232, col: 26, offset: 5902},
												val:        ".",
												ignoreCase: false,
											},
											&labeledExpr{
												pos:   position{line: 232, col: 30, offset: 5906},
												label: "field",
												expr: &ruleRefExpr{
													pos:  position{line: 232, col: 36, offset: 5912},
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
			pos:  position{line: 236, col: 1, offset: 6037},
			expr: &actionExpr{
				pos: position{line: 237, col: 5, offset: 6061},
				run: (*parser).callonfieldRefDotOnlyList1,
				expr: &seqExpr{
					pos: position{line: 237, col: 5, offset: 6061},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 237, col: 5, offset: 6061},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 237, col: 11, offset: 6067},
								name: "fieldRefDotOnly",
							},
						},
						&labeledExpr{
							pos:   position{line: 237, col: 27, offset: 6083},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 237, col: 32, offset: 6088},
								expr: &actionExpr{
									pos: position{line: 237, col: 33, offset: 6089},
									run: (*parser).callonfieldRefDotOnlyList7,
									expr: &seqExpr{
										pos: position{line: 237, col: 33, offset: 6089},
										exprs: []interface{}{
											&zeroOrOneExpr{
												pos: position{line: 237, col: 33, offset: 6089},
												expr: &ruleRefExpr{
													pos:  position{line: 237, col: 33, offset: 6089},
													name: "_",
												},
											},
											&litMatcher{
												pos:        position{line: 237, col: 36, offset: 6092},
												val:        ",",
												ignoreCase: false,
											},
											&zeroOrOneExpr{
												pos: position{line: 237, col: 40, offset: 6096},
												expr: &ruleRefExpr{
													pos:  position{line: 237, col: 40, offset: 6096},
													name: "_",
												},
											},
											&labeledExpr{
												pos:   position{line: 237, col: 43, offset: 6099},
												label: "ref",
												expr: &ruleRefExpr{
													pos:  position{line: 237, col: 47, offset: 6103},
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
			pos:  position{line: 245, col: 1, offset: 6283},
			expr: &actionExpr{
				pos: position{line: 246, col: 5, offset: 6301},
				run: (*parser).callonfieldNameList1,
				expr: &seqExpr{
					pos: position{line: 246, col: 5, offset: 6301},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 246, col: 5, offset: 6301},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 246, col: 11, offset: 6307},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 246, col: 21, offset: 6317},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 246, col: 26, offset: 6322},
								expr: &seqExpr{
									pos: position{line: 246, col: 27, offset: 6323},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 246, col: 27, offset: 6323},
											expr: &ruleRefExpr{
												pos:  position{line: 246, col: 27, offset: 6323},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 246, col: 30, offset: 6326},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 246, col: 34, offset: 6330},
											expr: &ruleRefExpr{
												pos:  position{line: 246, col: 34, offset: 6330},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 246, col: 37, offset: 6333},
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
			pos:  position{line: 254, col: 1, offset: 6526},
			expr: &actionExpr{
				pos: position{line: 255, col: 5, offset: 6538},
				run: (*parser).calloncountOp1,
				expr: &litMatcher{
					pos:        position{line: 255, col: 5, offset: 6538},
					val:        "count",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldReducerOp",
			pos:  position{line: 257, col: 1, offset: 6572},
			expr: &choiceExpr{
				pos: position{line: 258, col: 5, offset: 6591},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 258, col: 5, offset: 6591},
						run: (*parser).callonfieldReducerOp2,
						expr: &litMatcher{
							pos:        position{line: 258, col: 5, offset: 6591},
							val:        "sum",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 259, col: 5, offset: 6625},
						run: (*parser).callonfieldReducerOp4,
						expr: &litMatcher{
							pos:        position{line: 259, col: 5, offset: 6625},
							val:        "avg",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 260, col: 5, offset: 6659},
						run: (*parser).callonfieldReducerOp6,
						expr: &litMatcher{
							pos:        position{line: 260, col: 5, offset: 6659},
							val:        "stdev",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 261, col: 5, offset: 6696},
						run: (*parser).callonfieldReducerOp8,
						expr: &litMatcher{
							pos:        position{line: 261, col: 5, offset: 6696},
							val:        "sd",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 262, col: 5, offset: 6732},
						run: (*parser).callonfieldReducerOp10,
						expr: &litMatcher{
							pos:        position{line: 262, col: 5, offset: 6732},
							val:        "var",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 263, col: 5, offset: 6766},
						run: (*parser).callonfieldReducerOp12,
						expr: &litMatcher{
							pos:        position{line: 263, col: 5, offset: 6766},
							val:        "entropy",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 264, col: 5, offset: 6807},
						run: (*parser).callonfieldReducerOp14,
						expr: &litMatcher{
							pos:        position{line: 264, col: 5, offset: 6807},
							val:        "min",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 265, col: 5, offset: 6841},
						run: (*parser).callonfieldReducerOp16,
						expr: &litMatcher{
							pos:        position{line: 265, col: 5, offset: 6841},
							val:        "max",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 266, col: 5, offset: 6875},
						run: (*parser).callonfieldReducerOp18,
						expr: &litMatcher{
							pos:        position{line: 266, col: 5, offset: 6875},
							val:        "first",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 267, col: 5, offset: 6913},
						run: (*parser).callonfieldReducerOp20,
						expr: &litMatcher{
							pos:        position{line: 267, col: 5, offset: 6913},
							val:        "last",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 268, col: 5, offset: 6949},
						run: (*parser).callonfieldReducerOp22,
						expr: &litMatcher{
							pos:        position{line: 268, col: 5, offset: 6949},
							val:        "countdistinct",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "paddedFieldName",
			pos:  position{line: 270, col: 1, offset: 6999},
			expr: &actionExpr{
				pos: position{line: 270, col: 19, offset: 7017},
				run: (*parser).callonpaddedFieldName1,
				expr: &seqExpr{
					pos: position{line: 270, col: 19, offset: 7017},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 270, col: 19, offset: 7017},
							expr: &ruleRefExpr{
								pos:  position{line: 270, col: 19, offset: 7017},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 270, col: 22, offset: 7020},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 270, col: 28, offset: 7026},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 270, col: 38, offset: 7036},
							expr: &ruleRefExpr{
								pos:  position{line: 270, col: 38, offset: 7036},
								name: "_",
							},
						},
					},
				},
			},
		},
		{
			name: "countReducer",
			pos:  position{line: 272, col: 1, offset: 7062},
			expr: &actionExpr{
				pos: position{line: 273, col: 5, offset: 7079},
				run: (*parser).calloncountReducer1,
				expr: &seqExpr{
					pos: position{line: 273, col: 5, offset: 7079},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 273, col: 5, offset: 7079},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 273, col: 8, offset: 7082},
								name: "countOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 273, col: 16, offset: 7090},
							expr: &ruleRefExpr{
								pos:  position{line: 273, col: 16, offset: 7090},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 273, col: 19, offset: 7093},
							val:        "(",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 273, col: 23, offset: 7097},
							label: "field",
							expr: &zeroOrOneExpr{
								pos: position{line: 273, col: 29, offset: 7103},
								expr: &ruleRefExpr{
									pos:  position{line: 273, col: 29, offset: 7103},
									name: "paddedFieldName",
								},
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 273, col: 47, offset: 7121},
							expr: &ruleRefExpr{
								pos:  position{line: 273, col: 47, offset: 7121},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 273, col: 50, offset: 7124},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "fieldReducer",
			pos:  position{line: 277, col: 1, offset: 7183},
			expr: &actionExpr{
				pos: position{line: 278, col: 5, offset: 7200},
				run: (*parser).callonfieldReducer1,
				expr: &seqExpr{
					pos: position{line: 278, col: 5, offset: 7200},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 278, col: 5, offset: 7200},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 278, col: 8, offset: 7203},
								name: "fieldReducerOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 278, col: 23, offset: 7218},
							expr: &ruleRefExpr{
								pos:  position{line: 278, col: 23, offset: 7218},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 278, col: 26, offset: 7221},
							val:        "(",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 278, col: 30, offset: 7225},
							expr: &ruleRefExpr{
								pos:  position{line: 278, col: 30, offset: 7225},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 278, col: 33, offset: 7228},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 278, col: 39, offset: 7234},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 278, col: 50, offset: 7245},
							expr: &ruleRefExpr{
								pos:  position{line: 278, col: 50, offset: 7245},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 278, col: 53, offset: 7248},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reducerProc",
			pos:  position{line: 282, col: 1, offset: 7315},
			expr: &actionExpr{
				pos: position{line: 283, col: 5, offset: 7331},
				run: (*parser).callonreducerProc1,
				expr: &seqExpr{
					pos: position{line: 283, col: 5, offset: 7331},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 283, col: 5, offset: 7331},
							label: "every",
							expr: &zeroOrOneExpr{
								pos: position{line: 283, col: 11, offset: 7337},
								expr: &seqExpr{
									pos: position{line: 283, col: 12, offset: 7338},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 283, col: 12, offset: 7338},
											name: "everyDur",
										},
										&ruleRefExpr{
											pos:  position{line: 283, col: 21, offset: 7347},
											name: "_",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 283, col: 25, offset: 7351},
							label: "reducers",
							expr: &ruleRefExpr{
								pos:  position{line: 283, col: 34, offset: 7360},
								name: "reducerList",
							},
						},
						&labeledExpr{
							pos:   position{line: 283, col: 46, offset: 7372},
							label: "keys",
							expr: &zeroOrOneExpr{
								pos: position{line: 283, col: 51, offset: 7377},
								expr: &seqExpr{
									pos: position{line: 283, col: 52, offset: 7378},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 283, col: 52, offset: 7378},
											name: "_",
										},
										&ruleRefExpr{
											pos:  position{line: 283, col: 54, offset: 7380},
											name: "groupBy",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 283, col: 64, offset: 7390},
							label: "limit",
							expr: &zeroOrOneExpr{
								pos: position{line: 283, col: 70, offset: 7396},
								expr: &ruleRefExpr{
									pos:  position{line: 283, col: 70, offset: 7396},
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
			pos:  position{line: 301, col: 1, offset: 7753},
			expr: &actionExpr{
				pos: position{line: 302, col: 5, offset: 7766},
				run: (*parser).callonasClause1,
				expr: &seqExpr{
					pos: position{line: 302, col: 5, offset: 7766},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 302, col: 5, offset: 7766},
							val:        "as",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 302, col: 11, offset: 7772},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 302, col: 13, offset: 7774},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 302, col: 15, offset: 7776},
								name: "fieldName",
							},
						},
					},
				},
			},
		},
		{
			name: "reducerExpr",
			pos:  position{line: 304, col: 1, offset: 7805},
			expr: &choiceExpr{
				pos: position{line: 305, col: 5, offset: 7821},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 305, col: 5, offset: 7821},
						run: (*parser).callonreducerExpr2,
						expr: &seqExpr{
							pos: position{line: 305, col: 5, offset: 7821},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 305, col: 5, offset: 7821},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 305, col: 11, offset: 7827},
										name: "fieldName",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 305, col: 21, offset: 7837},
									expr: &ruleRefExpr{
										pos:  position{line: 305, col: 21, offset: 7837},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 305, col: 24, offset: 7840},
									val:        "=",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 305, col: 28, offset: 7844},
									expr: &ruleRefExpr{
										pos:  position{line: 305, col: 28, offset: 7844},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 305, col: 31, offset: 7847},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 305, col: 33, offset: 7849},
										name: "reducer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 308, col: 5, offset: 7912},
						run: (*parser).callonreducerExpr13,
						expr: &seqExpr{
							pos: position{line: 308, col: 5, offset: 7912},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 308, col: 5, offset: 7912},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 308, col: 7, offset: 7914},
										name: "reducer",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 308, col: 15, offset: 7922},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 308, col: 17, offset: 7924},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 308, col: 23, offset: 7930},
										name: "asClause",
									},
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 311, col: 5, offset: 7994},
						name: "reducer",
					},
				},
			},
		},
		{
			name: "reducer",
			pos:  position{line: 313, col: 1, offset: 8003},
			expr: &choiceExpr{
				pos: position{line: 314, col: 5, offset: 8015},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 314, col: 5, offset: 8015},
						name: "countReducer",
					},
					&ruleRefExpr{
						pos:  position{line: 315, col: 5, offset: 8032},
						name: "fieldReducer",
					},
				},
			},
		},
		{
			name: "reducerList",
			pos:  position{line: 317, col: 1, offset: 8046},
			expr: &actionExpr{
				pos: position{line: 318, col: 5, offset: 8062},
				run: (*parser).callonreducerList1,
				expr: &seqExpr{
					pos: position{line: 318, col: 5, offset: 8062},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 318, col: 5, offset: 8062},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 318, col: 11, offset: 8068},
								name: "reducerExpr",
							},
						},
						&labeledExpr{
							pos:   position{line: 318, col: 23, offset: 8080},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 318, col: 28, offset: 8085},
								expr: &seqExpr{
									pos: position{line: 318, col: 29, offset: 8086},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 318, col: 29, offset: 8086},
											expr: &ruleRefExpr{
												pos:  position{line: 318, col: 29, offset: 8086},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 318, col: 32, offset: 8089},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 318, col: 36, offset: 8093},
											expr: &ruleRefExpr{
												pos:  position{line: 318, col: 36, offset: 8093},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 318, col: 39, offset: 8096},
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
			pos:  position{line: 326, col: 1, offset: 8293},
			expr: &choiceExpr{
				pos: position{line: 327, col: 5, offset: 8308},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 327, col: 5, offset: 8308},
						name: "sort",
					},
					&ruleRefExpr{
						pos:  position{line: 328, col: 5, offset: 8317},
						name: "top",
					},
					&ruleRefExpr{
						pos:  position{line: 329, col: 5, offset: 8325},
						name: "cut",
					},
					&ruleRefExpr{
						pos:  position{line: 330, col: 5, offset: 8333},
						name: "head",
					},
					&ruleRefExpr{
						pos:  position{line: 331, col: 5, offset: 8342},
						name: "tail",
					},
					&ruleRefExpr{
						pos:  position{line: 332, col: 5, offset: 8351},
						name: "filter",
					},
					&ruleRefExpr{
						pos:  position{line: 333, col: 5, offset: 8362},
						name: "uniq",
					},
				},
			},
		},
		{
			name: "sort",
			pos:  position{line: 335, col: 1, offset: 8368},
			expr: &choiceExpr{
				pos: position{line: 336, col: 5, offset: 8377},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 336, col: 5, offset: 8377},
						run: (*parser).callonsort2,
						expr: &seqExpr{
							pos: position{line: 336, col: 5, offset: 8377},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 336, col: 5, offset: 8377},
									val:        "sort",
									ignoreCase: true,
								},
								&labeledExpr{
									pos:   position{line: 336, col: 13, offset: 8385},
									label: "rev",
									expr: &zeroOrOneExpr{
										pos: position{line: 336, col: 17, offset: 8389},
										expr: &seqExpr{
											pos: position{line: 336, col: 18, offset: 8390},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 336, col: 18, offset: 8390},
													name: "_",
												},
												&litMatcher{
													pos:        position{line: 336, col: 20, offset: 8392},
													val:        "-r",
													ignoreCase: false,
												},
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 336, col: 27, offset: 8399},
									label: "limit",
									expr: &zeroOrOneExpr{
										pos: position{line: 336, col: 33, offset: 8405},
										expr: &ruleRefExpr{
											pos:  position{line: 336, col: 33, offset: 8405},
											name: "procLimitArg",
										},
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 336, col: 48, offset: 8420},
									expr: &ruleRefExpr{
										pos:  position{line: 336, col: 48, offset: 8420},
										name: "_",
									},
								},
								&notExpr{
									pos: position{line: 336, col: 51, offset: 8423},
									expr: &litMatcher{
										pos:        position{line: 336, col: 52, offset: 8424},
										val:        "-r",
										ignoreCase: false,
									},
								},
								&labeledExpr{
									pos:   position{line: 336, col: 57, offset: 8429},
									label: "list",
									expr: &zeroOrOneExpr{
										pos: position{line: 336, col: 62, offset: 8434},
										expr: &ruleRefExpr{
											pos:  position{line: 336, col: 63, offset: 8435},
											name: "fieldExprList",
										},
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 341, col: 5, offset: 8565},
						run: (*parser).callonsort20,
						expr: &seqExpr{
							pos: position{line: 341, col: 5, offset: 8565},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 341, col: 5, offset: 8565},
									val:        "sort",
									ignoreCase: true,
								},
								&labeledExpr{
									pos:   position{line: 341, col: 13, offset: 8573},
									label: "limit",
									expr: &zeroOrOneExpr{
										pos: position{line: 341, col: 19, offset: 8579},
										expr: &ruleRefExpr{
											pos:  position{line: 341, col: 19, offset: 8579},
											name: "procLimitArg",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 341, col: 33, offset: 8593},
									label: "rev",
									expr: &zeroOrOneExpr{
										pos: position{line: 341, col: 37, offset: 8597},
										expr: &seqExpr{
											pos: position{line: 341, col: 38, offset: 8598},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 341, col: 38, offset: 8598},
													name: "_",
												},
												&litMatcher{
													pos:        position{line: 341, col: 40, offset: 8600},
													val:        "-r",
													ignoreCase: false,
												},
											},
										},
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 341, col: 47, offset: 8607},
									expr: &ruleRefExpr{
										pos:  position{line: 341, col: 47, offset: 8607},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 341, col: 50, offset: 8610},
									label: "list",
									expr: &zeroOrOneExpr{
										pos: position{line: 341, col: 55, offset: 8615},
										expr: &ruleRefExpr{
											pos:  position{line: 341, col: 56, offset: 8616},
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
			pos:  position{line: 347, col: 1, offset: 8743},
			expr: &actionExpr{
				pos: position{line: 348, col: 5, offset: 8751},
				run: (*parser).callontop1,
				expr: &seqExpr{
					pos: position{line: 348, col: 5, offset: 8751},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 348, col: 5, offset: 8751},
							val:        "top",
							ignoreCase: true,
						},
						&labeledExpr{
							pos:   position{line: 348, col: 12, offset: 8758},
							label: "limit",
							expr: &zeroOrOneExpr{
								pos: position{line: 348, col: 18, offset: 8764},
								expr: &actionExpr{
									pos: position{line: 348, col: 19, offset: 8765},
									run: (*parser).callontop6,
									expr: &seqExpr{
										pos: position{line: 348, col: 19, offset: 8765},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 348, col: 19, offset: 8765},
												name: "_",
											},
											&labeledExpr{
												pos:   position{line: 348, col: 21, offset: 8767},
												label: "n",
												expr: &ruleRefExpr{
													pos:  position{line: 348, col: 23, offset: 8769},
													name: "integer",
												},
											},
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 348, col: 50, offset: 8796},
							label: "flush",
							expr: &zeroOrOneExpr{
								pos: position{line: 348, col: 56, offset: 8802},
								expr: &seqExpr{
									pos: position{line: 348, col: 57, offset: 8803},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 348, col: 57, offset: 8803},
											name: "_",
										},
										&litMatcher{
											pos:        position{line: 348, col: 59, offset: 8805},
											val:        "-flush",
											ignoreCase: false,
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 348, col: 70, offset: 8816},
							label: "list",
							expr: &zeroOrOneExpr{
								pos: position{line: 348, col: 75, offset: 8821},
								expr: &actionExpr{
									pos: position{line: 348, col: 76, offset: 8822},
									run: (*parser).callontop18,
									expr: &seqExpr{
										pos: position{line: 348, col: 76, offset: 8822},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 348, col: 76, offset: 8822},
												name: "_",
											},
											&labeledExpr{
												pos:   position{line: 348, col: 78, offset: 8824},
												label: "f",
												expr: &ruleRefExpr{
													pos:  position{line: 348, col: 80, offset: 8826},
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
			},
		},
		{
			name: "procLimitArg",
			pos:  position{line: 352, col: 1, offset: 8915},
			expr: &actionExpr{
				pos: position{line: 353, col: 5, offset: 8932},
				run: (*parser).callonprocLimitArg1,
				expr: &seqExpr{
					pos: position{line: 353, col: 5, offset: 8932},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 353, col: 5, offset: 8932},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 353, col: 7, offset: 8934},
							val:        "-limit",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 353, col: 16, offset: 8943},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 353, col: 18, offset: 8945},
							label: "limit",
							expr: &ruleRefExpr{
								pos:  position{line: 353, col: 24, offset: 8951},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "cut",
			pos:  position{line: 355, col: 1, offset: 8982},
			expr: &actionExpr{
				pos: position{line: 356, col: 5, offset: 8990},
				run: (*parser).calloncut1,
				expr: &seqExpr{
					pos: position{line: 356, col: 5, offset: 8990},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 356, col: 5, offset: 8990},
							val:        "cut",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 356, col: 12, offset: 8997},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 356, col: 14, offset: 8999},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 356, col: 19, offset: 9004},
								name: "fieldRefDotOnlyList",
							},
						},
					},
				},
			},
		},
		{
			name: "head",
			pos:  position{line: 357, col: 1, offset: 9058},
			expr: &choiceExpr{
				pos: position{line: 358, col: 5, offset: 9067},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 358, col: 5, offset: 9067},
						run: (*parser).callonhead2,
						expr: &seqExpr{
							pos: position{line: 358, col: 5, offset: 9067},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 358, col: 5, offset: 9067},
									val:        "head",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 358, col: 13, offset: 9075},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 358, col: 15, offset: 9077},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 358, col: 21, offset: 9083},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 359, col: 5, offset: 9131},
						run: (*parser).callonhead8,
						expr: &litMatcher{
							pos:        position{line: 359, col: 5, offset: 9131},
							val:        "head",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "tail",
			pos:  position{line: 360, col: 1, offset: 9171},
			expr: &choiceExpr{
				pos: position{line: 361, col: 5, offset: 9180},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 361, col: 5, offset: 9180},
						run: (*parser).callontail2,
						expr: &seqExpr{
							pos: position{line: 361, col: 5, offset: 9180},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 361, col: 5, offset: 9180},
									val:        "tail",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 361, col: 13, offset: 9188},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 361, col: 15, offset: 9190},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 361, col: 21, offset: 9196},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 362, col: 5, offset: 9244},
						run: (*parser).callontail8,
						expr: &litMatcher{
							pos:        position{line: 362, col: 5, offset: 9244},
							val:        "tail",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "filter",
			pos:  position{line: 364, col: 1, offset: 9285},
			expr: &actionExpr{
				pos: position{line: 365, col: 5, offset: 9296},
				run: (*parser).callonfilter1,
				expr: &seqExpr{
					pos: position{line: 365, col: 5, offset: 9296},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 365, col: 5, offset: 9296},
							val:        "filter",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 365, col: 15, offset: 9306},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 365, col: 17, offset: 9308},
							label: "expr",
							expr: &ruleRefExpr{
								pos:  position{line: 365, col: 22, offset: 9313},
								name: "searchExpr",
							},
						},
					},
				},
			},
		},
		{
			name: "uniq",
			pos:  position{line: 368, col: 1, offset: 9371},
			expr: &choiceExpr{
				pos: position{line: 369, col: 5, offset: 9380},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 369, col: 5, offset: 9380},
						run: (*parser).callonuniq2,
						expr: &seqExpr{
							pos: position{line: 369, col: 5, offset: 9380},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 369, col: 5, offset: 9380},
									val:        "uniq",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 369, col: 13, offset: 9388},
									name: "_",
								},
								&litMatcher{
									pos:        position{line: 369, col: 15, offset: 9390},
									val:        "-c",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 372, col: 5, offset: 9444},
						run: (*parser).callonuniq7,
						expr: &litMatcher{
							pos:        position{line: 372, col: 5, offset: 9444},
							val:        "uniq",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "duration",
			pos:  position{line: 376, col: 1, offset: 9499},
			expr: &choiceExpr{
				pos: position{line: 377, col: 5, offset: 9512},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 377, col: 5, offset: 9512},
						name: "seconds",
					},
					&ruleRefExpr{
						pos:  position{line: 378, col: 5, offset: 9524},
						name: "minutes",
					},
					&ruleRefExpr{
						pos:  position{line: 379, col: 5, offset: 9536},
						name: "hours",
					},
					&seqExpr{
						pos: position{line: 380, col: 5, offset: 9546},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 380, col: 5, offset: 9546},
								name: "hours",
							},
							&ruleRefExpr{
								pos:  position{line: 380, col: 11, offset: 9552},
								name: "_",
							},
							&litMatcher{
								pos:        position{line: 380, col: 13, offset: 9554},
								val:        "and",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 380, col: 19, offset: 9560},
								name: "_",
							},
							&ruleRefExpr{
								pos:  position{line: 380, col: 21, offset: 9562},
								name: "minutes",
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 381, col: 5, offset: 9574},
						name: "days",
					},
					&ruleRefExpr{
						pos:  position{line: 382, col: 5, offset: 9583},
						name: "weeks",
					},
				},
			},
		},
		{
			name: "sec_abbrev",
			pos:  position{line: 384, col: 1, offset: 9590},
			expr: &choiceExpr{
				pos: position{line: 385, col: 5, offset: 9605},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 385, col: 5, offset: 9605},
						val:        "seconds",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 386, col: 5, offset: 9619},
						val:        "second",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 387, col: 5, offset: 9632},
						val:        "secs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 388, col: 5, offset: 9643},
						val:        "sec",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 389, col: 5, offset: 9653},
						val:        "s",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "min_abbrev",
			pos:  position{line: 391, col: 1, offset: 9658},
			expr: &choiceExpr{
				pos: position{line: 392, col: 5, offset: 9673},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 392, col: 5, offset: 9673},
						val:        "minutes",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 393, col: 5, offset: 9687},
						val:        "minute",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 394, col: 5, offset: 9700},
						val:        "mins",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 395, col: 5, offset: 9711},
						val:        "min",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 396, col: 5, offset: 9721},
						val:        "m",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "hour_abbrev",
			pos:  position{line: 398, col: 1, offset: 9726},
			expr: &choiceExpr{
				pos: position{line: 399, col: 5, offset: 9742},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 399, col: 5, offset: 9742},
						val:        "hours",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 400, col: 5, offset: 9754},
						val:        "hrs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 401, col: 5, offset: 9764},
						val:        "hr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 402, col: 5, offset: 9773},
						val:        "h",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 403, col: 5, offset: 9781},
						val:        "hour",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "day_abbrev",
			pos:  position{line: 405, col: 1, offset: 9789},
			expr: &choiceExpr{
				pos: position{line: 405, col: 14, offset: 9802},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 405, col: 14, offset: 9802},
						val:        "days",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 405, col: 21, offset: 9809},
						val:        "day",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 405, col: 27, offset: 9815},
						val:        "d",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "week_abbrev",
			pos:  position{line: 406, col: 1, offset: 9819},
			expr: &choiceExpr{
				pos: position{line: 406, col: 15, offset: 9833},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 406, col: 15, offset: 9833},
						val:        "weeks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 406, col: 23, offset: 9841},
						val:        "week",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 406, col: 30, offset: 9848},
						val:        "wks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 406, col: 36, offset: 9854},
						val:        "wk",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 406, col: 41, offset: 9859},
						val:        "w",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "seconds",
			pos:  position{line: 408, col: 1, offset: 9864},
			expr: &choiceExpr{
				pos: position{line: 409, col: 5, offset: 9876},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 409, col: 5, offset: 9876},
						run: (*parser).callonseconds2,
						expr: &litMatcher{
							pos:        position{line: 409, col: 5, offset: 9876},
							val:        "second",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 410, col: 5, offset: 9921},
						run: (*parser).callonseconds4,
						expr: &seqExpr{
							pos: position{line: 410, col: 5, offset: 9921},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 410, col: 5, offset: 9921},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 410, col: 9, offset: 9925},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 410, col: 16, offset: 9932},
									expr: &ruleRefExpr{
										pos:  position{line: 410, col: 16, offset: 9932},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 410, col: 19, offset: 9935},
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
			pos:  position{line: 412, col: 1, offset: 9981},
			expr: &choiceExpr{
				pos: position{line: 413, col: 5, offset: 9993},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 413, col: 5, offset: 9993},
						run: (*parser).callonminutes2,
						expr: &litMatcher{
							pos:        position{line: 413, col: 5, offset: 9993},
							val:        "minute",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 414, col: 5, offset: 10039},
						run: (*parser).callonminutes4,
						expr: &seqExpr{
							pos: position{line: 414, col: 5, offset: 10039},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 414, col: 5, offset: 10039},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 414, col: 9, offset: 10043},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 414, col: 16, offset: 10050},
									expr: &ruleRefExpr{
										pos:  position{line: 414, col: 16, offset: 10050},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 414, col: 19, offset: 10053},
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
			pos:  position{line: 416, col: 1, offset: 10108},
			expr: &choiceExpr{
				pos: position{line: 417, col: 5, offset: 10118},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 417, col: 5, offset: 10118},
						run: (*parser).callonhours2,
						expr: &litMatcher{
							pos:        position{line: 417, col: 5, offset: 10118},
							val:        "hour",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 418, col: 5, offset: 10164},
						run: (*parser).callonhours4,
						expr: &seqExpr{
							pos: position{line: 418, col: 5, offset: 10164},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 418, col: 5, offset: 10164},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 418, col: 9, offset: 10168},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 418, col: 16, offset: 10175},
									expr: &ruleRefExpr{
										pos:  position{line: 418, col: 16, offset: 10175},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 418, col: 19, offset: 10178},
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
			pos:  position{line: 420, col: 1, offset: 10236},
			expr: &choiceExpr{
				pos: position{line: 421, col: 5, offset: 10245},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 421, col: 5, offset: 10245},
						run: (*parser).callondays2,
						expr: &litMatcher{
							pos:        position{line: 421, col: 5, offset: 10245},
							val:        "day",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 422, col: 5, offset: 10293},
						run: (*parser).callondays4,
						expr: &seqExpr{
							pos: position{line: 422, col: 5, offset: 10293},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 422, col: 5, offset: 10293},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 422, col: 9, offset: 10297},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 422, col: 16, offset: 10304},
									expr: &ruleRefExpr{
										pos:  position{line: 422, col: 16, offset: 10304},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 422, col: 19, offset: 10307},
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
			pos:  position{line: 424, col: 1, offset: 10367},
			expr: &actionExpr{
				pos: position{line: 425, col: 5, offset: 10377},
				run: (*parser).callonweeks1,
				expr: &seqExpr{
					pos: position{line: 425, col: 5, offset: 10377},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 425, col: 5, offset: 10377},
							label: "num",
							expr: &ruleRefExpr{
								pos:  position{line: 425, col: 9, offset: 10381},
								name: "number",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 425, col: 16, offset: 10388},
							expr: &ruleRefExpr{
								pos:  position{line: 425, col: 16, offset: 10388},
								name: "_",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 425, col: 19, offset: 10391},
							name: "week_abbrev",
						},
					},
				},
			},
		},
		{
			name: "number",
			pos:  position{line: 427, col: 1, offset: 10454},
			expr: &ruleRefExpr{
				pos:  position{line: 427, col: 10, offset: 10463},
				name: "integer",
			},
		},
		{
			name: "addr",
			pos:  position{line: 431, col: 1, offset: 10501},
			expr: &actionExpr{
				pos: position{line: 432, col: 5, offset: 10510},
				run: (*parser).callonaddr1,
				expr: &labeledExpr{
					pos:   position{line: 432, col: 5, offset: 10510},
					label: "a",
					expr: &seqExpr{
						pos: position{line: 432, col: 8, offset: 10513},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 432, col: 8, offset: 10513},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 432, col: 16, offset: 10521},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 432, col: 20, offset: 10525},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 432, col: 28, offset: 10533},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 432, col: 32, offset: 10537},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 432, col: 40, offset: 10545},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 432, col: 44, offset: 10549},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "port",
			pos:  position{line: 434, col: 1, offset: 10590},
			expr: &actionExpr{
				pos: position{line: 435, col: 5, offset: 10599},
				run: (*parser).callonport1,
				expr: &seqExpr{
					pos: position{line: 435, col: 5, offset: 10599},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 435, col: 5, offset: 10599},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 435, col: 9, offset: 10603},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 435, col: 11, offset: 10605},
								name: "sinteger",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6addr",
			pos:  position{line: 439, col: 1, offset: 10764},
			expr: &choiceExpr{
				pos: position{line: 440, col: 5, offset: 10776},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 440, col: 5, offset: 10776},
						run: (*parser).callonip6addr2,
						expr: &seqExpr{
							pos: position{line: 440, col: 5, offset: 10776},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 440, col: 5, offset: 10776},
									label: "a",
									expr: &oneOrMoreExpr{
										pos: position{line: 440, col: 7, offset: 10778},
										expr: &ruleRefExpr{
											pos:  position{line: 440, col: 8, offset: 10779},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 440, col: 20, offset: 10791},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 440, col: 22, offset: 10793},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 443, col: 5, offset: 10857},
						run: (*parser).callonip6addr9,
						expr: &seqExpr{
							pos: position{line: 443, col: 5, offset: 10857},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 443, col: 5, offset: 10857},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 443, col: 7, offset: 10859},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 443, col: 11, offset: 10863},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 443, col: 13, offset: 10865},
										expr: &ruleRefExpr{
											pos:  position{line: 443, col: 14, offset: 10866},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 443, col: 25, offset: 10877},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 443, col: 30, offset: 10882},
									label: "d",
									expr: &zeroOrMoreExpr{
										pos: position{line: 443, col: 32, offset: 10884},
										expr: &ruleRefExpr{
											pos:  position{line: 443, col: 33, offset: 10885},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 443, col: 45, offset: 10897},
									label: "e",
									expr: &ruleRefExpr{
										pos:  position{line: 443, col: 47, offset: 10899},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 446, col: 5, offset: 10998},
						run: (*parser).callonip6addr22,
						expr: &seqExpr{
							pos: position{line: 446, col: 5, offset: 10998},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 446, col: 5, offset: 10998},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 446, col: 10, offset: 11003},
									label: "a",
									expr: &zeroOrMoreExpr{
										pos: position{line: 446, col: 12, offset: 11005},
										expr: &ruleRefExpr{
											pos:  position{line: 446, col: 13, offset: 11006},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 446, col: 25, offset: 11018},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 446, col: 27, offset: 11020},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 449, col: 5, offset: 11091},
						run: (*parser).callonip6addr30,
						expr: &seqExpr{
							pos: position{line: 449, col: 5, offset: 11091},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 449, col: 5, offset: 11091},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 449, col: 7, offset: 11093},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 449, col: 11, offset: 11097},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 449, col: 13, offset: 11099},
										expr: &ruleRefExpr{
											pos:  position{line: 449, col: 14, offset: 11100},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 449, col: 25, offset: 11111},
									val:        "::",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 452, col: 5, offset: 11179},
						run: (*parser).callonip6addr38,
						expr: &litMatcher{
							pos:        position{line: 452, col: 5, offset: 11179},
							val:        "::",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "ip6tail",
			pos:  position{line: 456, col: 1, offset: 11216},
			expr: &choiceExpr{
				pos: position{line: 457, col: 5, offset: 11228},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 457, col: 5, offset: 11228},
						name: "addr",
					},
					&ruleRefExpr{
						pos:  position{line: 458, col: 5, offset: 11237},
						name: "h16",
					},
				},
			},
		},
		{
			name: "h_append",
			pos:  position{line: 460, col: 1, offset: 11242},
			expr: &actionExpr{
				pos: position{line: 460, col: 12, offset: 11253},
				run: (*parser).callonh_append1,
				expr: &seqExpr{
					pos: position{line: 460, col: 12, offset: 11253},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 460, col: 12, offset: 11253},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 460, col: 16, offset: 11257},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 460, col: 18, offset: 11259},
								name: "h16",
							},
						},
					},
				},
			},
		},
		{
			name: "h_prepend",
			pos:  position{line: 461, col: 1, offset: 11296},
			expr: &actionExpr{
				pos: position{line: 461, col: 13, offset: 11308},
				run: (*parser).callonh_prepend1,
				expr: &seqExpr{
					pos: position{line: 461, col: 13, offset: 11308},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 461, col: 13, offset: 11308},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 461, col: 15, offset: 11310},
								name: "h16",
							},
						},
						&litMatcher{
							pos:        position{line: 461, col: 19, offset: 11314},
							val:        ":",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "sub_addr",
			pos:  position{line: 463, col: 1, offset: 11352},
			expr: &choiceExpr{
				pos: position{line: 464, col: 5, offset: 11365},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 464, col: 5, offset: 11365},
						name: "addr",
					},
					&actionExpr{
						pos: position{line: 465, col: 5, offset: 11374},
						run: (*parser).callonsub_addr3,
						expr: &labeledExpr{
							pos:   position{line: 465, col: 5, offset: 11374},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 465, col: 8, offset: 11377},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 465, col: 8, offset: 11377},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 465, col: 16, offset: 11385},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 465, col: 20, offset: 11389},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 465, col: 28, offset: 11397},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 465, col: 32, offset: 11401},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 466, col: 5, offset: 11453},
						run: (*parser).callonsub_addr11,
						expr: &labeledExpr{
							pos:   position{line: 466, col: 5, offset: 11453},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 466, col: 8, offset: 11456},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 466, col: 8, offset: 11456},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 466, col: 16, offset: 11464},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 466, col: 20, offset: 11468},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 467, col: 5, offset: 11522},
						run: (*parser).callonsub_addr17,
						expr: &labeledExpr{
							pos:   position{line: 467, col: 5, offset: 11522},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 467, col: 7, offset: 11524},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "subnet",
			pos:  position{line: 469, col: 1, offset: 11575},
			expr: &actionExpr{
				pos: position{line: 470, col: 5, offset: 11586},
				run: (*parser).callonsubnet1,
				expr: &seqExpr{
					pos: position{line: 470, col: 5, offset: 11586},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 470, col: 5, offset: 11586},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 470, col: 7, offset: 11588},
								name: "sub_addr",
							},
						},
						&litMatcher{
							pos:        position{line: 470, col: 16, offset: 11597},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 470, col: 20, offset: 11601},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 470, col: 22, offset: 11603},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6subnet",
			pos:  position{line: 474, col: 1, offset: 11679},
			expr: &actionExpr{
				pos: position{line: 475, col: 5, offset: 11693},
				run: (*parser).callonip6subnet1,
				expr: &seqExpr{
					pos: position{line: 475, col: 5, offset: 11693},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 475, col: 5, offset: 11693},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 475, col: 7, offset: 11695},
								name: "ip6addr",
							},
						},
						&litMatcher{
							pos:        position{line: 475, col: 15, offset: 11703},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 475, col: 19, offset: 11707},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 475, col: 21, offset: 11709},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "integer",
			pos:  position{line: 479, col: 1, offset: 11775},
			expr: &actionExpr{
				pos: position{line: 480, col: 5, offset: 11787},
				run: (*parser).calloninteger1,
				expr: &labeledExpr{
					pos:   position{line: 480, col: 5, offset: 11787},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 480, col: 7, offset: 11789},
						name: "sinteger",
					},
				},
			},
		},
		{
			name: "sinteger",
			pos:  position{line: 484, col: 1, offset: 11833},
			expr: &actionExpr{
				pos: position{line: 485, col: 5, offset: 11846},
				run: (*parser).callonsinteger1,
				expr: &labeledExpr{
					pos:   position{line: 485, col: 5, offset: 11846},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 485, col: 11, offset: 11852},
						expr: &charClassMatcher{
							pos:        position{line: 485, col: 11, offset: 11852},
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
			pos:  position{line: 489, col: 1, offset: 11897},
			expr: &actionExpr{
				pos: position{line: 490, col: 5, offset: 11908},
				run: (*parser).callondouble1,
				expr: &labeledExpr{
					pos:   position{line: 490, col: 5, offset: 11908},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 490, col: 7, offset: 11910},
						name: "sdouble",
					},
				},
			},
		},
		{
			name: "sdouble",
			pos:  position{line: 494, col: 1, offset: 11957},
			expr: &choiceExpr{
				pos: position{line: 495, col: 5, offset: 11969},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 495, col: 5, offset: 11969},
						run: (*parser).callonsdouble2,
						expr: &seqExpr{
							pos: position{line: 495, col: 5, offset: 11969},
							exprs: []interface{}{
								&oneOrMoreExpr{
									pos: position{line: 495, col: 5, offset: 11969},
									expr: &ruleRefExpr{
										pos:  position{line: 495, col: 5, offset: 11969},
										name: "doubleInteger",
									},
								},
								&litMatcher{
									pos:        position{line: 495, col: 20, offset: 11984},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 495, col: 24, offset: 11988},
									expr: &ruleRefExpr{
										pos:  position{line: 495, col: 24, offset: 11988},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 495, col: 37, offset: 12001},
									expr: &ruleRefExpr{
										pos:  position{line: 495, col: 37, offset: 12001},
										name: "exponentPart",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 498, col: 5, offset: 12060},
						run: (*parser).callonsdouble11,
						expr: &seqExpr{
							pos: position{line: 498, col: 5, offset: 12060},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 498, col: 5, offset: 12060},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 498, col: 9, offset: 12064},
									expr: &ruleRefExpr{
										pos:  position{line: 498, col: 9, offset: 12064},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 498, col: 22, offset: 12077},
									expr: &ruleRefExpr{
										pos:  position{line: 498, col: 22, offset: 12077},
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
			pos:  position{line: 502, col: 1, offset: 12133},
			expr: &choiceExpr{
				pos: position{line: 503, col: 5, offset: 12151},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 503, col: 5, offset: 12151},
						val:        "0",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 504, col: 5, offset: 12159},
						exprs: []interface{}{
							&charClassMatcher{
								pos:        position{line: 504, col: 5, offset: 12159},
								val:        "[1-9]",
								ranges:     []rune{'1', '9'},
								ignoreCase: false,
								inverted:   false,
							},
							&zeroOrMoreExpr{
								pos: position{line: 504, col: 11, offset: 12165},
								expr: &charClassMatcher{
									pos:        position{line: 504, col: 11, offset: 12165},
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
			pos:  position{line: 506, col: 1, offset: 12173},
			expr: &charClassMatcher{
				pos:        position{line: 506, col: 15, offset: 12187},
				val:        "[0-9]",
				ranges:     []rune{'0', '9'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "signedInteger",
			pos:  position{line: 508, col: 1, offset: 12194},
			expr: &seqExpr{
				pos: position{line: 508, col: 17, offset: 12210},
				exprs: []interface{}{
					&zeroOrOneExpr{
						pos: position{line: 508, col: 17, offset: 12210},
						expr: &charClassMatcher{
							pos:        position{line: 508, col: 17, offset: 12210},
							val:        "[+-]",
							chars:      []rune{'+', '-'},
							ignoreCase: false,
							inverted:   false,
						},
					},
					&oneOrMoreExpr{
						pos: position{line: 508, col: 23, offset: 12216},
						expr: &ruleRefExpr{
							pos:  position{line: 508, col: 23, offset: 12216},
							name: "doubleDigit",
						},
					},
				},
			},
		},
		{
			name: "exponentPart",
			pos:  position{line: 510, col: 1, offset: 12230},
			expr: &seqExpr{
				pos: position{line: 510, col: 16, offset: 12245},
				exprs: []interface{}{
					&litMatcher{
						pos:        position{line: 510, col: 16, offset: 12245},
						val:        "e",
						ignoreCase: true,
					},
					&ruleRefExpr{
						pos:  position{line: 510, col: 21, offset: 12250},
						name: "signedInteger",
					},
				},
			},
		},
		{
			name: "h16",
			pos:  position{line: 512, col: 1, offset: 12265},
			expr: &actionExpr{
				pos: position{line: 512, col: 7, offset: 12271},
				run: (*parser).callonh161,
				expr: &labeledExpr{
					pos:   position{line: 512, col: 7, offset: 12271},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 512, col: 13, offset: 12277},
						expr: &ruleRefExpr{
							pos:  position{line: 512, col: 13, offset: 12277},
							name: "hexdigit",
						},
					},
				},
			},
		},
		{
			name: "hexdigit",
			pos:  position{line: 514, col: 1, offset: 12319},
			expr: &charClassMatcher{
				pos:        position{line: 514, col: 12, offset: 12330},
				val:        "[0-9a-fA-F]",
				ranges:     []rune{'0', '9', 'a', 'f', 'A', 'F'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name:        "boomWord",
			displayName: "\"boomWord\"",
			pos:         position{line: 516, col: 1, offset: 12343},
			expr: &actionExpr{
				pos: position{line: 516, col: 23, offset: 12365},
				run: (*parser).callonboomWord1,
				expr: &labeledExpr{
					pos:   position{line: 516, col: 23, offset: 12365},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 516, col: 29, offset: 12371},
						expr: &ruleRefExpr{
							pos:  position{line: 516, col: 29, offset: 12371},
							name: "boomWordPart",
						},
					},
				},
			},
		},
		{
			name: "boomWordPart",
			pos:  position{line: 518, col: 1, offset: 12419},
			expr: &choiceExpr{
				pos: position{line: 519, col: 5, offset: 12436},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 519, col: 5, offset: 12436},
						run: (*parser).callonboomWordPart2,
						expr: &seqExpr{
							pos: position{line: 519, col: 5, offset: 12436},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 519, col: 5, offset: 12436},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 519, col: 10, offset: 12441},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 519, col: 12, offset: 12443},
										name: "escapeSequence",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 520, col: 5, offset: 12481},
						run: (*parser).callonboomWordPart7,
						expr: &seqExpr{
							pos: position{line: 520, col: 5, offset: 12481},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 520, col: 5, offset: 12481},
									expr: &ruleRefExpr{
										pos:  position{line: 520, col: 6, offset: 12482},
										name: "ws",
									},
								},
								&anyMatcher{
									line: 520, col: 9, offset: 12485,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "quotedString",
			pos:  position{line: 522, col: 1, offset: 12519},
			expr: &choiceExpr{
				pos: position{line: 523, col: 5, offset: 12536},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 523, col: 5, offset: 12536},
						run: (*parser).callonquotedString2,
						expr: &seqExpr{
							pos: position{line: 523, col: 5, offset: 12536},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 523, col: 5, offset: 12536},
									val:        "\"",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 523, col: 9, offset: 12540},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 523, col: 11, offset: 12542},
										expr: &ruleRefExpr{
											pos:  position{line: 523, col: 11, offset: 12542},
											name: "doubleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 523, col: 29, offset: 12560},
									val:        "\"",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 524, col: 5, offset: 12597},
						run: (*parser).callonquotedString9,
						expr: &seqExpr{
							pos: position{line: 524, col: 5, offset: 12597},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 524, col: 5, offset: 12597},
									val:        "'",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 524, col: 9, offset: 12601},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 524, col: 11, offset: 12603},
										expr: &ruleRefExpr{
											pos:  position{line: 524, col: 11, offset: 12603},
											name: "singleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 524, col: 29, offset: 12621},
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
			pos:  position{line: 526, col: 1, offset: 12655},
			expr: &choiceExpr{
				pos: position{line: 527, col: 5, offset: 12676},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 527, col: 5, offset: 12676},
						run: (*parser).callondoubleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 527, col: 5, offset: 12676},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 527, col: 5, offset: 12676},
									expr: &choiceExpr{
										pos: position{line: 527, col: 7, offset: 12678},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 527, col: 7, offset: 12678},
												val:        "\"",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 527, col: 13, offset: 12684},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 527, col: 26, offset: 12697,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 528, col: 5, offset: 12734},
						run: (*parser).callondoubleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 528, col: 5, offset: 12734},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 528, col: 5, offset: 12734},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 528, col: 10, offset: 12739},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 528, col: 12, offset: 12741},
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
			pos:  position{line: 530, col: 1, offset: 12775},
			expr: &choiceExpr{
				pos: position{line: 531, col: 5, offset: 12796},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 531, col: 5, offset: 12796},
						run: (*parser).callonsingleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 531, col: 5, offset: 12796},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 531, col: 5, offset: 12796},
									expr: &choiceExpr{
										pos: position{line: 531, col: 7, offset: 12798},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 531, col: 7, offset: 12798},
												val:        "'",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 531, col: 13, offset: 12804},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 531, col: 26, offset: 12817,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 532, col: 5, offset: 12854},
						run: (*parser).callonsingleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 532, col: 5, offset: 12854},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 532, col: 5, offset: 12854},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 532, col: 10, offset: 12859},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 532, col: 12, offset: 12861},
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
			pos:  position{line: 534, col: 1, offset: 12895},
			expr: &choiceExpr{
				pos: position{line: 535, col: 5, offset: 12914},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 535, col: 5, offset: 12914},
						run: (*parser).callonescapeSequence2,
						expr: &seqExpr{
							pos: position{line: 535, col: 5, offset: 12914},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 535, col: 5, offset: 12914},
									val:        "x",
									ignoreCase: false,
								},
								&ruleRefExpr{
									pos:  position{line: 535, col: 9, offset: 12918},
									name: "hexdigit",
								},
								&ruleRefExpr{
									pos:  position{line: 535, col: 18, offset: 12927},
									name: "hexdigit",
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 536, col: 5, offset: 12978},
						name: "singleCharEscape",
					},
					&ruleRefExpr{
						pos:  position{line: 537, col: 5, offset: 12999},
						name: "unicodeEscape",
					},
				},
			},
		},
		{
			name: "singleCharEscape",
			pos:  position{line: 539, col: 1, offset: 13014},
			expr: &choiceExpr{
				pos: position{line: 540, col: 5, offset: 13035},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 540, col: 5, offset: 13035},
						val:        "'",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 541, col: 5, offset: 13043},
						val:        "\"",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 542, col: 5, offset: 13051},
						val:        "\\",
						ignoreCase: false,
					},
					&actionExpr{
						pos: position{line: 543, col: 5, offset: 13060},
						run: (*parser).callonsingleCharEscape5,
						expr: &litMatcher{
							pos:        position{line: 543, col: 5, offset: 13060},
							val:        "b",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 544, col: 5, offset: 13089},
						run: (*parser).callonsingleCharEscape7,
						expr: &litMatcher{
							pos:        position{line: 544, col: 5, offset: 13089},
							val:        "f",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 545, col: 5, offset: 13118},
						run: (*parser).callonsingleCharEscape9,
						expr: &litMatcher{
							pos:        position{line: 545, col: 5, offset: 13118},
							val:        "n",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 546, col: 5, offset: 13147},
						run: (*parser).callonsingleCharEscape11,
						expr: &litMatcher{
							pos:        position{line: 546, col: 5, offset: 13147},
							val:        "r",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 547, col: 5, offset: 13176},
						run: (*parser).callonsingleCharEscape13,
						expr: &litMatcher{
							pos:        position{line: 547, col: 5, offset: 13176},
							val:        "t",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 548, col: 5, offset: 13205},
						run: (*parser).callonsingleCharEscape15,
						expr: &litMatcher{
							pos:        position{line: 548, col: 5, offset: 13205},
							val:        "v",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 549, col: 5, offset: 13234},
						run: (*parser).callonsingleCharEscape17,
						expr: &litMatcher{
							pos:        position{line: 549, col: 5, offset: 13234},
							val:        "=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 550, col: 5, offset: 13264},
						run: (*parser).callonsingleCharEscape19,
						expr: &litMatcher{
							pos:        position{line: 550, col: 5, offset: 13264},
							val:        "*",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "unicodeEscape",
			pos:  position{line: 552, col: 1, offset: 13291},
			expr: &choiceExpr{
				pos: position{line: 553, col: 5, offset: 13309},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 553, col: 5, offset: 13309},
						run: (*parser).callonunicodeEscape2,
						expr: &seqExpr{
							pos: position{line: 553, col: 5, offset: 13309},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 553, col: 5, offset: 13309},
									val:        "u",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 553, col: 9, offset: 13313},
									label: "chars",
									expr: &seqExpr{
										pos: position{line: 553, col: 16, offset: 13320},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 553, col: 16, offset: 13320},
												name: "hexdigit",
											},
											&ruleRefExpr{
												pos:  position{line: 553, col: 25, offset: 13329},
												name: "hexdigit",
											},
											&ruleRefExpr{
												pos:  position{line: 553, col: 34, offset: 13338},
												name: "hexdigit",
											},
											&ruleRefExpr{
												pos:  position{line: 553, col: 43, offset: 13347},
												name: "hexdigit",
											},
										},
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 556, col: 5, offset: 13410},
						run: (*parser).callonunicodeEscape11,
						expr: &seqExpr{
							pos: position{line: 556, col: 5, offset: 13410},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 556, col: 5, offset: 13410},
									val:        "u",
									ignoreCase: false,
								},
								&litMatcher{
									pos:        position{line: 556, col: 9, offset: 13414},
									val:        "{",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 556, col: 13, offset: 13418},
									label: "chars",
									expr: &seqExpr{
										pos: position{line: 556, col: 20, offset: 13425},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 556, col: 20, offset: 13425},
												name: "hexdigit",
											},
											&zeroOrOneExpr{
												pos: position{line: 556, col: 29, offset: 13434},
												expr: &ruleRefExpr{
													pos:  position{line: 556, col: 29, offset: 13434},
													name: "hexdigit",
												},
											},
											&zeroOrOneExpr{
												pos: position{line: 556, col: 39, offset: 13444},
												expr: &ruleRefExpr{
													pos:  position{line: 556, col: 39, offset: 13444},
													name: "hexdigit",
												},
											},
											&zeroOrOneExpr{
												pos: position{line: 556, col: 49, offset: 13454},
												expr: &ruleRefExpr{
													pos:  position{line: 556, col: 49, offset: 13454},
													name: "hexdigit",
												},
											},
											&zeroOrOneExpr{
												pos: position{line: 556, col: 59, offset: 13464},
												expr: &ruleRefExpr{
													pos:  position{line: 556, col: 59, offset: 13464},
													name: "hexdigit",
												},
											},
											&zeroOrOneExpr{
												pos: position{line: 556, col: 69, offset: 13474},
												expr: &ruleRefExpr{
													pos:  position{line: 556, col: 69, offset: 13474},
													name: "hexdigit",
												},
											},
										},
									},
								},
								&litMatcher{
									pos:        position{line: 556, col: 80, offset: 13485},
									val:        "}",
									ignoreCase: false,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "reString",
			pos:  position{line: 560, col: 1, offset: 13539},
			expr: &actionExpr{
				pos: position{line: 561, col: 5, offset: 13552},
				run: (*parser).callonreString1,
				expr: &seqExpr{
					pos: position{line: 561, col: 5, offset: 13552},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 561, col: 5, offset: 13552},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 561, col: 9, offset: 13556},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 561, col: 11, offset: 13558},
								name: "reBody",
							},
						},
						&litMatcher{
							pos:        position{line: 561, col: 18, offset: 13565},
							val:        "/",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reBody",
			pos:  position{line: 563, col: 1, offset: 13588},
			expr: &actionExpr{
				pos: position{line: 564, col: 5, offset: 13599},
				run: (*parser).callonreBody1,
				expr: &oneOrMoreExpr{
					pos: position{line: 564, col: 5, offset: 13599},
					expr: &choiceExpr{
						pos: position{line: 564, col: 6, offset: 13600},
						alternatives: []interface{}{
							&charClassMatcher{
								pos:        position{line: 564, col: 6, offset: 13600},
								val:        "[^/\\\\]",
								chars:      []rune{'/', '\\'},
								ignoreCase: false,
								inverted:   true,
							},
							&litMatcher{
								pos:        position{line: 564, col: 13, offset: 13607},
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
			pos:  position{line: 566, col: 1, offset: 13647},
			expr: &charClassMatcher{
				pos:        position{line: 567, col: 5, offset: 13663},
				val:        "[\\x00-\\x1f\\\\]",
				chars:      []rune{'\\'},
				ranges:     []rune{'\x00', '\x1f'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "ws",
			pos:  position{line: 569, col: 1, offset: 13678},
			expr: &choiceExpr{
				pos: position{line: 570, col: 5, offset: 13685},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 570, col: 5, offset: 13685},
						val:        "\t",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 571, col: 5, offset: 13694},
						val:        "\v",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 572, col: 5, offset: 13703},
						val:        "\f",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 573, col: 5, offset: 13712},
						val:        " ",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 574, col: 5, offset: 13720},
						val:        "\u00a0",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 575, col: 5, offset: 13733},
						val:        "\ufeff",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name:        "_",
			displayName: "\"whitespace\"",
			pos:         position{line: 577, col: 1, offset: 13743},
			expr: &oneOrMoreExpr{
				pos: position{line: 577, col: 18, offset: 13760},
				expr: &ruleRefExpr{
					pos:  position{line: 577, col: 18, offset: 13760},
					name: "ws",
				},
			},
		},
		{
			name: "EOF",
			pos:  position{line: 579, col: 1, offset: 13765},
			expr: &notExpr{
				pos: position{line: 579, col: 7, offset: 13771},
				expr: &anyMatcher{
					line: 579, col: 8, offset: 13772,
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

func (c *current) onsearchPred24(f, fieldComparator, v interface{}) (interface{}, error) {
	return makeCompareField(fieldComparator, f, v), nil

}

func (p *parser) callonsearchPred24() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchPred24(stack["f"], stack["fieldComparator"], stack["v"])
}

func (c *current) onsearchPred36(v interface{}) (interface{}, error) {
	return makeCompareAny("in", false, v), nil

}

func (p *parser) callonsearchPred36() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchPred36(stack["v"])
}

func (c *current) onsearchPred46(v, f interface{}) (interface{}, error) {
	return makeCompareField("in", f, v), nil

}

func (p *parser) callonsearchPred46() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchPred46(stack["v"], stack["f"])
}

func (c *current) onsearchPred57(v interface{}) (interface{}, error) {
	if getValueType(v) == "string" {
		return makeOrChain(makeCompareAny("search", true, v), []interface{}{makeCompareAny("searchin", true, v)}), nil
	}
	if getValueType(v) == "regexp" {
		if string(c.text) == "*" {
			return makeBooleanLiteral(true), nil
		}
		return makeOrChain(makeCompareAny("eql", true, v), []interface{}{makeCompareAny("in", true, v)}), nil
	}

	return makeOrChain(makeCompareAny("search", true, makeTypedValue("string", string(c.text))), []interface{}{makeCompareAny("searchin", true, makeTypedValue("string", string(c.text))), makeCompareAny("eql", true, v), makeCompareAny("in", true, v)}), nil

}

func (p *parser) callonsearchPred57() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchPred57(stack["v"])
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
	if reglob.IsGlobby(v.(string)) {
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

func (c *current) ontop6(n interface{}) (interface{}, error) {
	return n, nil
}

func (p *parser) callontop6() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.ontop6(stack["n"])
}

func (c *current) ontop18(f interface{}) (interface{}, error) {
	return f, nil
}

func (p *parser) callontop18() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.ontop18(stack["f"])
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

func (c *current) onsingleCharEscape17() (interface{}, error) {
	return "\\=", nil
}

func (p *parser) callonsingleCharEscape17() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsingleCharEscape17()
}

func (c *current) onsingleCharEscape19() (interface{}, error) {
	return "\\*", nil
}

func (p *parser) callonsingleCharEscape19() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsingleCharEscape19()
}

func (c *current) onunicodeEscape2(chars interface{}) (interface{}, error) {
	return makeUnicodeChar(chars), nil

}

func (p *parser) callonunicodeEscape2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onunicodeEscape2(stack["chars"])
}

func (c *current) onunicodeEscape11(chars interface{}) (interface{}, error) {
	return makeUnicodeChar(chars), nil

}

func (p *parser) callonunicodeEscape11() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onunicodeEscape11(stack["chars"])
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

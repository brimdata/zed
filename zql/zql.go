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
			pos:  position{line: 97, col: 1, offset: 2784},
			expr: &choiceExpr{
				pos: position{line: 98, col: 5, offset: 2800},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 98, col: 5, offset: 2800},
						run: (*parser).callonsearchValue2,
						expr: &labeledExpr{
							pos:   position{line: 98, col: 5, offset: 2800},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 98, col: 7, offset: 2802},
								name: "quotedString",
							},
						},
					},
					&actionExpr{
						pos: position{line: 101, col: 5, offset: 2870},
						run: (*parser).callonsearchValue5,
						expr: &labeledExpr{
							pos:   position{line: 101, col: 5, offset: 2870},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 101, col: 7, offset: 2872},
								name: "reString",
							},
						},
					},
					&actionExpr{
						pos: position{line: 104, col: 5, offset: 2936},
						run: (*parser).callonsearchValue8,
						expr: &labeledExpr{
							pos:   position{line: 104, col: 5, offset: 2936},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 104, col: 7, offset: 2938},
								name: "port",
							},
						},
					},
					&actionExpr{
						pos: position{line: 107, col: 5, offset: 2994},
						run: (*parser).callonsearchValue11,
						expr: &labeledExpr{
							pos:   position{line: 107, col: 5, offset: 2994},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 107, col: 7, offset: 2996},
								name: "ip6subnet",
							},
						},
					},
					&actionExpr{
						pos: position{line: 110, col: 5, offset: 3061},
						run: (*parser).callonsearchValue14,
						expr: &labeledExpr{
							pos:   position{line: 110, col: 5, offset: 3061},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 110, col: 7, offset: 3063},
								name: "ip6addr",
							},
						},
					},
					&actionExpr{
						pos: position{line: 113, col: 5, offset: 3124},
						run: (*parser).callonsearchValue17,
						expr: &labeledExpr{
							pos:   position{line: 113, col: 5, offset: 3124},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 113, col: 7, offset: 3126},
								name: "subnet",
							},
						},
					},
					&actionExpr{
						pos: position{line: 116, col: 5, offset: 3188},
						run: (*parser).callonsearchValue20,
						expr: &labeledExpr{
							pos:   position{line: 116, col: 5, offset: 3188},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 116, col: 7, offset: 3190},
								name: "addr",
							},
						},
					},
					&actionExpr{
						pos: position{line: 119, col: 5, offset: 3248},
						run: (*parser).callonsearchValue23,
						expr: &labeledExpr{
							pos:   position{line: 119, col: 5, offset: 3248},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 119, col: 7, offset: 3250},
								name: "sdouble",
							},
						},
					},
					&actionExpr{
						pos: position{line: 122, col: 5, offset: 3313},
						run: (*parser).callonsearchValue26,
						expr: &seqExpr{
							pos: position{line: 122, col: 5, offset: 3313},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 122, col: 5, offset: 3313},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 122, col: 7, offset: 3315},
										name: "sinteger",
									},
								},
								&notExpr{
									pos: position{line: 122, col: 16, offset: 3324},
									expr: &ruleRefExpr{
										pos:  position{line: 122, col: 17, offset: 3325},
										name: "boomWord",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 125, col: 5, offset: 3386},
						run: (*parser).callonsearchValue32,
						expr: &seqExpr{
							pos: position{line: 125, col: 5, offset: 3386},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 125, col: 5, offset: 3386},
									expr: &seqExpr{
										pos: position{line: 125, col: 7, offset: 3388},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 125, col: 7, offset: 3388},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 125, col: 22, offset: 3403},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 125, col: 25, offset: 3406},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 125, col: 27, offset: 3408},
										name: "booleanLiteral",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 126, col: 5, offset: 3445},
						run: (*parser).callonsearchValue40,
						expr: &seqExpr{
							pos: position{line: 126, col: 5, offset: 3445},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 126, col: 5, offset: 3445},
									expr: &seqExpr{
										pos: position{line: 126, col: 7, offset: 3447},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 126, col: 7, offset: 3447},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 126, col: 22, offset: 3462},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 126, col: 25, offset: 3465},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 126, col: 27, offset: 3467},
										name: "unsetLiteral",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 127, col: 5, offset: 3502},
						run: (*parser).callonsearchValue48,
						expr: &seqExpr{
							pos: position{line: 127, col: 5, offset: 3502},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 127, col: 5, offset: 3502},
									expr: &seqExpr{
										pos: position{line: 127, col: 7, offset: 3504},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 127, col: 8, offset: 3505},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 127, col: 24, offset: 3521},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 127, col: 27, offset: 3524},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 127, col: 29, offset: 3526},
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
			pos:  position{line: 135, col: 1, offset: 3723},
			expr: &choiceExpr{
				pos: position{line: 136, col: 5, offset: 3742},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 136, col: 5, offset: 3742},
						name: "andToken",
					},
					&ruleRefExpr{
						pos:  position{line: 137, col: 5, offset: 3755},
						name: "orToken",
					},
					&ruleRefExpr{
						pos:  position{line: 138, col: 5, offset: 3767},
						name: "inToken",
					},
				},
			},
		},
		{
			name: "booleanLiteral",
			pos:  position{line: 140, col: 1, offset: 3776},
			expr: &choiceExpr{
				pos: position{line: 141, col: 5, offset: 3795},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 141, col: 5, offset: 3795},
						run: (*parser).callonbooleanLiteral2,
						expr: &litMatcher{
							pos:        position{line: 141, col: 5, offset: 3795},
							val:        "true",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 142, col: 5, offset: 3860},
						run: (*parser).callonbooleanLiteral4,
						expr: &litMatcher{
							pos:        position{line: 142, col: 5, offset: 3860},
							val:        "false",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "unsetLiteral",
			pos:  position{line: 144, col: 1, offset: 3923},
			expr: &actionExpr{
				pos: position{line: 145, col: 5, offset: 3940},
				run: (*parser).callonunsetLiteral1,
				expr: &litMatcher{
					pos:        position{line: 145, col: 5, offset: 3940},
					val:        "nil",
					ignoreCase: false,
				},
			},
		},
		{
			name: "procList",
			pos:  position{line: 147, col: 1, offset: 3998},
			expr: &actionExpr{
				pos: position{line: 148, col: 5, offset: 4011},
				run: (*parser).callonprocList1,
				expr: &seqExpr{
					pos: position{line: 148, col: 5, offset: 4011},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 148, col: 5, offset: 4011},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 148, col: 11, offset: 4017},
								name: "procChain",
							},
						},
						&labeledExpr{
							pos:   position{line: 148, col: 21, offset: 4027},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 148, col: 26, offset: 4032},
								expr: &ruleRefExpr{
									pos:  position{line: 148, col: 26, offset: 4032},
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
			pos:  position{line: 157, col: 1, offset: 4256},
			expr: &actionExpr{
				pos: position{line: 158, col: 5, offset: 4274},
				run: (*parser).callonparallelChain1,
				expr: &seqExpr{
					pos: position{line: 158, col: 5, offset: 4274},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 158, col: 5, offset: 4274},
							expr: &ruleRefExpr{
								pos:  position{line: 158, col: 5, offset: 4274},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 158, col: 8, offset: 4277},
							val:        ";",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 158, col: 12, offset: 4281},
							expr: &ruleRefExpr{
								pos:  position{line: 158, col: 12, offset: 4281},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 158, col: 15, offset: 4284},
							label: "ch",
							expr: &ruleRefExpr{
								pos:  position{line: 158, col: 18, offset: 4287},
								name: "procChain",
							},
						},
					},
				},
			},
		},
		{
			name: "proc",
			pos:  position{line: 160, col: 1, offset: 4337},
			expr: &choiceExpr{
				pos: position{line: 161, col: 5, offset: 4346},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 161, col: 5, offset: 4346},
						name: "simpleProc",
					},
					&ruleRefExpr{
						pos:  position{line: 162, col: 5, offset: 4361},
						name: "reducerProc",
					},
					&actionExpr{
						pos: position{line: 163, col: 5, offset: 4377},
						run: (*parser).callonproc4,
						expr: &seqExpr{
							pos: position{line: 163, col: 5, offset: 4377},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 163, col: 5, offset: 4377},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 163, col: 9, offset: 4381},
									expr: &ruleRefExpr{
										pos:  position{line: 163, col: 9, offset: 4381},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 163, col: 12, offset: 4384},
									label: "proc",
									expr: &ruleRefExpr{
										pos:  position{line: 163, col: 17, offset: 4389},
										name: "procList",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 163, col: 26, offset: 4398},
									expr: &ruleRefExpr{
										pos:  position{line: 163, col: 26, offset: 4398},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 163, col: 29, offset: 4401},
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
			pos:  position{line: 167, col: 1, offset: 4437},
			expr: &actionExpr{
				pos: position{line: 168, col: 5, offset: 4449},
				run: (*parser).callongroupBy1,
				expr: &seqExpr{
					pos: position{line: 168, col: 5, offset: 4449},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 168, col: 5, offset: 4449},
							val:        "by",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 168, col: 11, offset: 4455},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 168, col: 13, offset: 4457},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 168, col: 18, offset: 4462},
								name: "fieldExprList",
							},
						},
					},
				},
			},
		},
		{
			name: "everyDur",
			pos:  position{line: 170, col: 1, offset: 4498},
			expr: &actionExpr{
				pos: position{line: 171, col: 5, offset: 4511},
				run: (*parser).calloneveryDur1,
				expr: &seqExpr{
					pos: position{line: 171, col: 5, offset: 4511},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 171, col: 5, offset: 4511},
							val:        "every",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 171, col: 14, offset: 4520},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 171, col: 16, offset: 4522},
							label: "dur",
							expr: &ruleRefExpr{
								pos:  position{line: 171, col: 20, offset: 4526},
								name: "duration",
							},
						},
					},
				},
			},
		},
		{
			name: "equalityToken",
			pos:  position{line: 173, col: 1, offset: 4556},
			expr: &choiceExpr{
				pos: position{line: 174, col: 5, offset: 4574},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 174, col: 5, offset: 4574},
						run: (*parser).callonequalityToken2,
						expr: &litMatcher{
							pos:        position{line: 174, col: 5, offset: 4574},
							val:        "=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 175, col: 5, offset: 4604},
						run: (*parser).callonequalityToken4,
						expr: &litMatcher{
							pos:        position{line: 175, col: 5, offset: 4604},
							val:        "!=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 176, col: 5, offset: 4636},
						run: (*parser).callonequalityToken6,
						expr: &litMatcher{
							pos:        position{line: 176, col: 5, offset: 4636},
							val:        "<=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 177, col: 5, offset: 4667},
						run: (*parser).callonequalityToken8,
						expr: &litMatcher{
							pos:        position{line: 177, col: 5, offset: 4667},
							val:        ">=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 178, col: 5, offset: 4698},
						run: (*parser).callonequalityToken10,
						expr: &litMatcher{
							pos:        position{line: 178, col: 5, offset: 4698},
							val:        "<",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 179, col: 5, offset: 4727},
						run: (*parser).callonequalityToken12,
						expr: &litMatcher{
							pos:        position{line: 179, col: 5, offset: 4727},
							val:        ">",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "types",
			pos:  position{line: 181, col: 1, offset: 4753},
			expr: &choiceExpr{
				pos: position{line: 182, col: 5, offset: 4763},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 182, col: 5, offset: 4763},
						val:        "bool",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 183, col: 5, offset: 4774},
						val:        "int",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 184, col: 5, offset: 4784},
						val:        "count",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 185, col: 5, offset: 4796},
						val:        "double",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 186, col: 5, offset: 4809},
						val:        "string",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 187, col: 5, offset: 4822},
						val:        "addr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 188, col: 5, offset: 4833},
						val:        "subnet",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 189, col: 5, offset: 4846},
						val:        "port",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "dash",
			pos:  position{line: 191, col: 1, offset: 4854},
			expr: &choiceExpr{
				pos: position{line: 191, col: 8, offset: 4861},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 191, col: 8, offset: 4861},
						val:        "-",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 191, col: 14, offset: 4867},
						val:        "—",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 191, col: 25, offset: 4878},
						val:        "–",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 191, col: 36, offset: 4889},
						exprs: []interface{}{
							&litMatcher{
								pos:        position{line: 191, col: 36, offset: 4889},
								val:        "to",
								ignoreCase: false,
							},
							&andExpr{
								pos: position{line: 191, col: 40, offset: 4893},
								expr: &ruleRefExpr{
									pos:  position{line: 191, col: 42, offset: 4895},
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
			pos:  position{line: 193, col: 1, offset: 4899},
			expr: &litMatcher{
				pos:        position{line: 193, col: 12, offset: 4910},
				val:        "and",
				ignoreCase: false,
			},
		},
		{
			name: "orToken",
			pos:  position{line: 194, col: 1, offset: 4916},
			expr: &litMatcher{
				pos:        position{line: 194, col: 11, offset: 4926},
				val:        "or",
				ignoreCase: false,
			},
		},
		{
			name: "inToken",
			pos:  position{line: 195, col: 1, offset: 4931},
			expr: &litMatcher{
				pos:        position{line: 195, col: 11, offset: 4941},
				val:        "in",
				ignoreCase: false,
			},
		},
		{
			name: "notToken",
			pos:  position{line: 196, col: 1, offset: 4946},
			expr: &litMatcher{
				pos:        position{line: 196, col: 12, offset: 4957},
				val:        "not",
				ignoreCase: false,
			},
		},
		{
			name: "fieldName",
			pos:  position{line: 198, col: 1, offset: 4964},
			expr: &actionExpr{
				pos: position{line: 198, col: 13, offset: 4976},
				run: (*parser).callonfieldName1,
				expr: &seqExpr{
					pos: position{line: 198, col: 13, offset: 4976},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 198, col: 13, offset: 4976},
							name: "fieldNameStart",
						},
						&zeroOrMoreExpr{
							pos: position{line: 198, col: 28, offset: 4991},
							expr: &ruleRefExpr{
								pos:  position{line: 198, col: 28, offset: 4991},
								name: "fieldNameRest",
							},
						},
					},
				},
			},
		},
		{
			name: "fieldNameStart",
			pos:  position{line: 200, col: 1, offset: 5038},
			expr: &charClassMatcher{
				pos:        position{line: 200, col: 18, offset: 5055},
				val:        "[A-Za-z_$]",
				chars:      []rune{'_', '$'},
				ranges:     []rune{'A', 'Z', 'a', 'z'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "fieldNameRest",
			pos:  position{line: 201, col: 1, offset: 5066},
			expr: &choiceExpr{
				pos: position{line: 201, col: 17, offset: 5082},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 201, col: 17, offset: 5082},
						name: "fieldNameStart",
					},
					&charClassMatcher{
						pos:        position{line: 201, col: 34, offset: 5099},
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
			pos:  position{line: 203, col: 1, offset: 5106},
			expr: &actionExpr{
				pos: position{line: 204, col: 4, offset: 5124},
				run: (*parser).callonfieldReference1,
				expr: &seqExpr{
					pos: position{line: 204, col: 4, offset: 5124},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 204, col: 4, offset: 5124},
							label: "base",
							expr: &ruleRefExpr{
								pos:  position{line: 204, col: 9, offset: 5129},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 204, col: 19, offset: 5139},
							label: "derefs",
							expr: &zeroOrMoreExpr{
								pos: position{line: 204, col: 26, offset: 5146},
								expr: &choiceExpr{
									pos: position{line: 205, col: 8, offset: 5155},
									alternatives: []interface{}{
										&actionExpr{
											pos: position{line: 205, col: 8, offset: 5155},
											run: (*parser).callonfieldReference8,
											expr: &seqExpr{
												pos: position{line: 205, col: 8, offset: 5155},
												exprs: []interface{}{
													&litMatcher{
														pos:        position{line: 205, col: 8, offset: 5155},
														val:        ".",
														ignoreCase: false,
													},
													&labeledExpr{
														pos:   position{line: 205, col: 12, offset: 5159},
														label: "field",
														expr: &ruleRefExpr{
															pos:  position{line: 205, col: 18, offset: 5165},
															name: "fieldName",
														},
													},
												},
											},
										},
										&actionExpr{
											pos: position{line: 206, col: 8, offset: 5246},
											run: (*parser).callonfieldReference13,
											expr: &seqExpr{
												pos: position{line: 206, col: 8, offset: 5246},
												exprs: []interface{}{
													&litMatcher{
														pos:        position{line: 206, col: 8, offset: 5246},
														val:        "[",
														ignoreCase: false,
													},
													&labeledExpr{
														pos:   position{line: 206, col: 12, offset: 5250},
														label: "index",
														expr: &ruleRefExpr{
															pos:  position{line: 206, col: 18, offset: 5256},
															name: "sinteger",
														},
													},
													&litMatcher{
														pos:        position{line: 206, col: 27, offset: 5265},
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
			pos:  position{line: 211, col: 1, offset: 5381},
			expr: &choiceExpr{
				pos: position{line: 212, col: 5, offset: 5395},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 212, col: 5, offset: 5395},
						run: (*parser).callonfieldExpr2,
						expr: &seqExpr{
							pos: position{line: 212, col: 5, offset: 5395},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 212, col: 5, offset: 5395},
									label: "op",
									expr: &ruleRefExpr{
										pos:  position{line: 212, col: 8, offset: 5398},
										name: "fieldOp",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 212, col: 16, offset: 5406},
									expr: &ruleRefExpr{
										pos:  position{line: 212, col: 16, offset: 5406},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 212, col: 19, offset: 5409},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 212, col: 23, offset: 5413},
									expr: &ruleRefExpr{
										pos:  position{line: 212, col: 23, offset: 5413},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 212, col: 26, offset: 5416},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 212, col: 32, offset: 5422},
										name: "fieldReference",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 212, col: 47, offset: 5437},
									expr: &ruleRefExpr{
										pos:  position{line: 212, col: 47, offset: 5437},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 212, col: 50, offset: 5440},
									val:        ")",
									ignoreCase: false,
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 215, col: 5, offset: 5504},
						name: "fieldReference",
					},
				},
			},
		},
		{
			name: "fieldOp",
			pos:  position{line: 217, col: 1, offset: 5520},
			expr: &actionExpr{
				pos: position{line: 218, col: 5, offset: 5532},
				run: (*parser).callonfieldOp1,
				expr: &litMatcher{
					pos:        position{line: 218, col: 5, offset: 5532},
					val:        "len",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldExprList",
			pos:  position{line: 220, col: 1, offset: 5562},
			expr: &actionExpr{
				pos: position{line: 221, col: 5, offset: 5580},
				run: (*parser).callonfieldExprList1,
				expr: &seqExpr{
					pos: position{line: 221, col: 5, offset: 5580},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 221, col: 5, offset: 5580},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 221, col: 11, offset: 5586},
								name: "fieldExpr",
							},
						},
						&labeledExpr{
							pos:   position{line: 221, col: 21, offset: 5596},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 221, col: 26, offset: 5601},
								expr: &seqExpr{
									pos: position{line: 221, col: 27, offset: 5602},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 221, col: 27, offset: 5602},
											expr: &ruleRefExpr{
												pos:  position{line: 221, col: 27, offset: 5602},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 221, col: 30, offset: 5605},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 221, col: 34, offset: 5609},
											expr: &ruleRefExpr{
												pos:  position{line: 221, col: 34, offset: 5609},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 221, col: 37, offset: 5612},
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
			pos:  position{line: 231, col: 1, offset: 5807},
			expr: &actionExpr{
				pos: position{line: 232, col: 5, offset: 5827},
				run: (*parser).callonfieldRefDotOnly1,
				expr: &seqExpr{
					pos: position{line: 232, col: 5, offset: 5827},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 232, col: 5, offset: 5827},
							label: "base",
							expr: &ruleRefExpr{
								pos:  position{line: 232, col: 10, offset: 5832},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 232, col: 20, offset: 5842},
							label: "refs",
							expr: &zeroOrMoreExpr{
								pos: position{line: 232, col: 25, offset: 5847},
								expr: &actionExpr{
									pos: position{line: 232, col: 26, offset: 5848},
									run: (*parser).callonfieldRefDotOnly7,
									expr: &seqExpr{
										pos: position{line: 232, col: 26, offset: 5848},
										exprs: []interface{}{
											&litMatcher{
												pos:        position{line: 232, col: 26, offset: 5848},
												val:        ".",
												ignoreCase: false,
											},
											&labeledExpr{
												pos:   position{line: 232, col: 30, offset: 5852},
												label: "field",
												expr: &ruleRefExpr{
													pos:  position{line: 232, col: 36, offset: 5858},
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
			pos:  position{line: 236, col: 1, offset: 5983},
			expr: &actionExpr{
				pos: position{line: 237, col: 5, offset: 6007},
				run: (*parser).callonfieldRefDotOnlyList1,
				expr: &seqExpr{
					pos: position{line: 237, col: 5, offset: 6007},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 237, col: 5, offset: 6007},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 237, col: 11, offset: 6013},
								name: "fieldRefDotOnly",
							},
						},
						&labeledExpr{
							pos:   position{line: 237, col: 27, offset: 6029},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 237, col: 32, offset: 6034},
								expr: &actionExpr{
									pos: position{line: 237, col: 33, offset: 6035},
									run: (*parser).callonfieldRefDotOnlyList7,
									expr: &seqExpr{
										pos: position{line: 237, col: 33, offset: 6035},
										exprs: []interface{}{
											&zeroOrOneExpr{
												pos: position{line: 237, col: 33, offset: 6035},
												expr: &ruleRefExpr{
													pos:  position{line: 237, col: 33, offset: 6035},
													name: "_",
												},
											},
											&litMatcher{
												pos:        position{line: 237, col: 36, offset: 6038},
												val:        ",",
												ignoreCase: false,
											},
											&zeroOrOneExpr{
												pos: position{line: 237, col: 40, offset: 6042},
												expr: &ruleRefExpr{
													pos:  position{line: 237, col: 40, offset: 6042},
													name: "_",
												},
											},
											&labeledExpr{
												pos:   position{line: 237, col: 43, offset: 6045},
												label: "ref",
												expr: &ruleRefExpr{
													pos:  position{line: 237, col: 47, offset: 6049},
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
			pos:  position{line: 245, col: 1, offset: 6229},
			expr: &actionExpr{
				pos: position{line: 246, col: 5, offset: 6247},
				run: (*parser).callonfieldNameList1,
				expr: &seqExpr{
					pos: position{line: 246, col: 5, offset: 6247},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 246, col: 5, offset: 6247},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 246, col: 11, offset: 6253},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 246, col: 21, offset: 6263},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 246, col: 26, offset: 6268},
								expr: &seqExpr{
									pos: position{line: 246, col: 27, offset: 6269},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 246, col: 27, offset: 6269},
											expr: &ruleRefExpr{
												pos:  position{line: 246, col: 27, offset: 6269},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 246, col: 30, offset: 6272},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 246, col: 34, offset: 6276},
											expr: &ruleRefExpr{
												pos:  position{line: 246, col: 34, offset: 6276},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 246, col: 37, offset: 6279},
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
			pos:  position{line: 254, col: 1, offset: 6472},
			expr: &actionExpr{
				pos: position{line: 255, col: 5, offset: 6484},
				run: (*parser).calloncountOp1,
				expr: &litMatcher{
					pos:        position{line: 255, col: 5, offset: 6484},
					val:        "count",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldReducerOp",
			pos:  position{line: 257, col: 1, offset: 6518},
			expr: &choiceExpr{
				pos: position{line: 258, col: 5, offset: 6537},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 258, col: 5, offset: 6537},
						run: (*parser).callonfieldReducerOp2,
						expr: &litMatcher{
							pos:        position{line: 258, col: 5, offset: 6537},
							val:        "sum",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 259, col: 5, offset: 6571},
						run: (*parser).callonfieldReducerOp4,
						expr: &litMatcher{
							pos:        position{line: 259, col: 5, offset: 6571},
							val:        "avg",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 260, col: 5, offset: 6605},
						run: (*parser).callonfieldReducerOp6,
						expr: &litMatcher{
							pos:        position{line: 260, col: 5, offset: 6605},
							val:        "stdev",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 261, col: 5, offset: 6642},
						run: (*parser).callonfieldReducerOp8,
						expr: &litMatcher{
							pos:        position{line: 261, col: 5, offset: 6642},
							val:        "sd",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 262, col: 5, offset: 6678},
						run: (*parser).callonfieldReducerOp10,
						expr: &litMatcher{
							pos:        position{line: 262, col: 5, offset: 6678},
							val:        "var",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 263, col: 5, offset: 6712},
						run: (*parser).callonfieldReducerOp12,
						expr: &litMatcher{
							pos:        position{line: 263, col: 5, offset: 6712},
							val:        "entropy",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 264, col: 5, offset: 6753},
						run: (*parser).callonfieldReducerOp14,
						expr: &litMatcher{
							pos:        position{line: 264, col: 5, offset: 6753},
							val:        "min",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 265, col: 5, offset: 6787},
						run: (*parser).callonfieldReducerOp16,
						expr: &litMatcher{
							pos:        position{line: 265, col: 5, offset: 6787},
							val:        "max",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 266, col: 5, offset: 6821},
						run: (*parser).callonfieldReducerOp18,
						expr: &litMatcher{
							pos:        position{line: 266, col: 5, offset: 6821},
							val:        "first",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 267, col: 5, offset: 6859},
						run: (*parser).callonfieldReducerOp20,
						expr: &litMatcher{
							pos:        position{line: 267, col: 5, offset: 6859},
							val:        "last",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 268, col: 5, offset: 6895},
						run: (*parser).callonfieldReducerOp22,
						expr: &litMatcher{
							pos:        position{line: 268, col: 5, offset: 6895},
							val:        "countdistinct",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "paddedFieldName",
			pos:  position{line: 270, col: 1, offset: 6945},
			expr: &actionExpr{
				pos: position{line: 270, col: 19, offset: 6963},
				run: (*parser).callonpaddedFieldName1,
				expr: &seqExpr{
					pos: position{line: 270, col: 19, offset: 6963},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 270, col: 19, offset: 6963},
							expr: &ruleRefExpr{
								pos:  position{line: 270, col: 19, offset: 6963},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 270, col: 22, offset: 6966},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 270, col: 28, offset: 6972},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 270, col: 38, offset: 6982},
							expr: &ruleRefExpr{
								pos:  position{line: 270, col: 38, offset: 6982},
								name: "_",
							},
						},
					},
				},
			},
		},
		{
			name: "countReducer",
			pos:  position{line: 272, col: 1, offset: 7008},
			expr: &actionExpr{
				pos: position{line: 273, col: 5, offset: 7025},
				run: (*parser).calloncountReducer1,
				expr: &seqExpr{
					pos: position{line: 273, col: 5, offset: 7025},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 273, col: 5, offset: 7025},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 273, col: 8, offset: 7028},
								name: "countOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 273, col: 16, offset: 7036},
							expr: &ruleRefExpr{
								pos:  position{line: 273, col: 16, offset: 7036},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 273, col: 19, offset: 7039},
							val:        "(",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 273, col: 23, offset: 7043},
							label: "field",
							expr: &zeroOrOneExpr{
								pos: position{line: 273, col: 29, offset: 7049},
								expr: &ruleRefExpr{
									pos:  position{line: 273, col: 29, offset: 7049},
									name: "paddedFieldName",
								},
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 273, col: 47, offset: 7067},
							expr: &ruleRefExpr{
								pos:  position{line: 273, col: 47, offset: 7067},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 273, col: 50, offset: 7070},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "fieldReducer",
			pos:  position{line: 277, col: 1, offset: 7129},
			expr: &actionExpr{
				pos: position{line: 278, col: 5, offset: 7146},
				run: (*parser).callonfieldReducer1,
				expr: &seqExpr{
					pos: position{line: 278, col: 5, offset: 7146},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 278, col: 5, offset: 7146},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 278, col: 8, offset: 7149},
								name: "fieldReducerOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 278, col: 23, offset: 7164},
							expr: &ruleRefExpr{
								pos:  position{line: 278, col: 23, offset: 7164},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 278, col: 26, offset: 7167},
							val:        "(",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 278, col: 30, offset: 7171},
							expr: &ruleRefExpr{
								pos:  position{line: 278, col: 30, offset: 7171},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 278, col: 33, offset: 7174},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 278, col: 39, offset: 7180},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 278, col: 50, offset: 7191},
							expr: &ruleRefExpr{
								pos:  position{line: 278, col: 50, offset: 7191},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 278, col: 53, offset: 7194},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reducerProc",
			pos:  position{line: 282, col: 1, offset: 7261},
			expr: &actionExpr{
				pos: position{line: 283, col: 5, offset: 7277},
				run: (*parser).callonreducerProc1,
				expr: &seqExpr{
					pos: position{line: 283, col: 5, offset: 7277},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 283, col: 5, offset: 7277},
							label: "every",
							expr: &zeroOrOneExpr{
								pos: position{line: 283, col: 11, offset: 7283},
								expr: &seqExpr{
									pos: position{line: 283, col: 12, offset: 7284},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 283, col: 12, offset: 7284},
											name: "everyDur",
										},
										&ruleRefExpr{
											pos:  position{line: 283, col: 21, offset: 7293},
											name: "_",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 283, col: 25, offset: 7297},
							label: "reducers",
							expr: &ruleRefExpr{
								pos:  position{line: 283, col: 34, offset: 7306},
								name: "reducerList",
							},
						},
						&labeledExpr{
							pos:   position{line: 283, col: 46, offset: 7318},
							label: "keys",
							expr: &zeroOrOneExpr{
								pos: position{line: 283, col: 51, offset: 7323},
								expr: &seqExpr{
									pos: position{line: 283, col: 52, offset: 7324},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 283, col: 52, offset: 7324},
											name: "_",
										},
										&ruleRefExpr{
											pos:  position{line: 283, col: 54, offset: 7326},
											name: "groupBy",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 283, col: 64, offset: 7336},
							label: "limit",
							expr: &zeroOrOneExpr{
								pos: position{line: 283, col: 70, offset: 7342},
								expr: &ruleRefExpr{
									pos:  position{line: 283, col: 70, offset: 7342},
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
			pos:  position{line: 301, col: 1, offset: 7699},
			expr: &actionExpr{
				pos: position{line: 302, col: 5, offset: 7712},
				run: (*parser).callonasClause1,
				expr: &seqExpr{
					pos: position{line: 302, col: 5, offset: 7712},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 302, col: 5, offset: 7712},
							val:        "as",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 302, col: 11, offset: 7718},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 302, col: 13, offset: 7720},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 302, col: 15, offset: 7722},
								name: "fieldName",
							},
						},
					},
				},
			},
		},
		{
			name: "reducerExpr",
			pos:  position{line: 304, col: 1, offset: 7751},
			expr: &choiceExpr{
				pos: position{line: 305, col: 5, offset: 7767},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 305, col: 5, offset: 7767},
						run: (*parser).callonreducerExpr2,
						expr: &seqExpr{
							pos: position{line: 305, col: 5, offset: 7767},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 305, col: 5, offset: 7767},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 305, col: 11, offset: 7773},
										name: "fieldName",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 305, col: 21, offset: 7783},
									expr: &ruleRefExpr{
										pos:  position{line: 305, col: 21, offset: 7783},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 305, col: 24, offset: 7786},
									val:        "=",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 305, col: 28, offset: 7790},
									expr: &ruleRefExpr{
										pos:  position{line: 305, col: 28, offset: 7790},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 305, col: 31, offset: 7793},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 305, col: 33, offset: 7795},
										name: "reducer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 308, col: 5, offset: 7858},
						run: (*parser).callonreducerExpr13,
						expr: &seqExpr{
							pos: position{line: 308, col: 5, offset: 7858},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 308, col: 5, offset: 7858},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 308, col: 7, offset: 7860},
										name: "reducer",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 308, col: 15, offset: 7868},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 308, col: 17, offset: 7870},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 308, col: 23, offset: 7876},
										name: "asClause",
									},
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 311, col: 5, offset: 7940},
						name: "reducer",
					},
				},
			},
		},
		{
			name: "reducer",
			pos:  position{line: 313, col: 1, offset: 7949},
			expr: &choiceExpr{
				pos: position{line: 314, col: 5, offset: 7961},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 314, col: 5, offset: 7961},
						name: "countReducer",
					},
					&ruleRefExpr{
						pos:  position{line: 315, col: 5, offset: 7978},
						name: "fieldReducer",
					},
				},
			},
		},
		{
			name: "reducerList",
			pos:  position{line: 317, col: 1, offset: 7992},
			expr: &actionExpr{
				pos: position{line: 318, col: 5, offset: 8008},
				run: (*parser).callonreducerList1,
				expr: &seqExpr{
					pos: position{line: 318, col: 5, offset: 8008},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 318, col: 5, offset: 8008},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 318, col: 11, offset: 8014},
								name: "reducerExpr",
							},
						},
						&labeledExpr{
							pos:   position{line: 318, col: 23, offset: 8026},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 318, col: 28, offset: 8031},
								expr: &seqExpr{
									pos: position{line: 318, col: 29, offset: 8032},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 318, col: 29, offset: 8032},
											expr: &ruleRefExpr{
												pos:  position{line: 318, col: 29, offset: 8032},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 318, col: 32, offset: 8035},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 318, col: 36, offset: 8039},
											expr: &ruleRefExpr{
												pos:  position{line: 318, col: 36, offset: 8039},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 318, col: 39, offset: 8042},
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
			pos:  position{line: 326, col: 1, offset: 8239},
			expr: &choiceExpr{
				pos: position{line: 327, col: 5, offset: 8254},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 327, col: 5, offset: 8254},
						name: "sort",
					},
					&ruleRefExpr{
						pos:  position{line: 328, col: 5, offset: 8263},
						name: "top",
					},
					&ruleRefExpr{
						pos:  position{line: 329, col: 5, offset: 8271},
						name: "cut",
					},
					&ruleRefExpr{
						pos:  position{line: 330, col: 5, offset: 8279},
						name: "head",
					},
					&ruleRefExpr{
						pos:  position{line: 331, col: 5, offset: 8288},
						name: "tail",
					},
					&ruleRefExpr{
						pos:  position{line: 332, col: 5, offset: 8297},
						name: "filter",
					},
					&ruleRefExpr{
						pos:  position{line: 333, col: 5, offset: 8308},
						name: "uniq",
					},
				},
			},
		},
		{
			name: "sort",
			pos:  position{line: 335, col: 1, offset: 8314},
			expr: &actionExpr{
				pos: position{line: 336, col: 5, offset: 8323},
				run: (*parser).callonsort1,
				expr: &seqExpr{
					pos: position{line: 336, col: 5, offset: 8323},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 336, col: 5, offset: 8323},
							val:        "sort",
							ignoreCase: true,
						},
						&labeledExpr{
							pos:   position{line: 336, col: 13, offset: 8331},
							label: "args",
							expr: &ruleRefExpr{
								pos:  position{line: 336, col: 18, offset: 8336},
								name: "sortArgs",
							},
						},
						&labeledExpr{
							pos:   position{line: 336, col: 27, offset: 8345},
							label: "list",
							expr: &zeroOrOneExpr{
								pos: position{line: 336, col: 32, offset: 8350},
								expr: &actionExpr{
									pos: position{line: 336, col: 33, offset: 8351},
									run: (*parser).callonsort8,
									expr: &seqExpr{
										pos: position{line: 336, col: 33, offset: 8351},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 336, col: 33, offset: 8351},
												name: "_",
											},
											&labeledExpr{
												pos:   position{line: 336, col: 35, offset: 8353},
												label: "l",
												expr: &ruleRefExpr{
													pos:  position{line: 336, col: 37, offset: 8355},
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
			name: "sortArgs",
			pos:  position{line: 340, col: 1, offset: 8432},
			expr: &zeroOrMoreExpr{
				pos: position{line: 340, col: 12, offset: 8443},
				expr: &actionExpr{
					pos: position{line: 340, col: 13, offset: 8444},
					run: (*parser).callonsortArgs2,
					expr: &seqExpr{
						pos: position{line: 340, col: 13, offset: 8444},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 340, col: 13, offset: 8444},
								name: "_",
							},
							&labeledExpr{
								pos:   position{line: 340, col: 15, offset: 8446},
								label: "a",
								expr: &ruleRefExpr{
									pos:  position{line: 340, col: 17, offset: 8448},
									name: "sortArg",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "sortArg",
			pos:  position{line: 342, col: 1, offset: 8477},
			expr: &choiceExpr{
				pos: position{line: 343, col: 5, offset: 8489},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 343, col: 5, offset: 8489},
						run: (*parser).callonsortArg2,
						expr: &seqExpr{
							pos: position{line: 343, col: 5, offset: 8489},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 343, col: 5, offset: 8489},
									val:        "-limit",
									ignoreCase: false,
								},
								&ruleRefExpr{
									pos:  position{line: 343, col: 14, offset: 8498},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 343, col: 16, offset: 8500},
									label: "limit",
									expr: &ruleRefExpr{
										pos:  position{line: 343, col: 22, offset: 8506},
										name: "sinteger",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 344, col: 5, offset: 8559},
						run: (*parser).callonsortArg8,
						expr: &litMatcher{
							pos:        position{line: 344, col: 5, offset: 8559},
							val:        "-r",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 345, col: 5, offset: 8602},
						run: (*parser).callonsortArg10,
						expr: &seqExpr{
							pos: position{line: 345, col: 5, offset: 8602},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 345, col: 5, offset: 8602},
									val:        "-nulls",
									ignoreCase: false,
								},
								&ruleRefExpr{
									pos:  position{line: 345, col: 14, offset: 8611},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 345, col: 16, offset: 8613},
									label: "where",
									expr: &actionExpr{
										pos: position{line: 345, col: 23, offset: 8620},
										run: (*parser).callonsortArg15,
										expr: &choiceExpr{
											pos: position{line: 345, col: 24, offset: 8621},
											alternatives: []interface{}{
												&litMatcher{
													pos:        position{line: 345, col: 24, offset: 8621},
													val:        "first",
													ignoreCase: false,
												},
												&litMatcher{
													pos:        position{line: 345, col: 34, offset: 8631},
													val:        "last",
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
		{
			name: "top",
			pos:  position{line: 347, col: 1, offset: 8713},
			expr: &actionExpr{
				pos: position{line: 348, col: 5, offset: 8721},
				run: (*parser).callontop1,
				expr: &seqExpr{
					pos: position{line: 348, col: 5, offset: 8721},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 348, col: 5, offset: 8721},
							val:        "top",
							ignoreCase: true,
						},
						&labeledExpr{
							pos:   position{line: 348, col: 12, offset: 8728},
							label: "limit",
							expr: &zeroOrOneExpr{
								pos: position{line: 348, col: 18, offset: 8734},
								expr: &actionExpr{
									pos: position{line: 348, col: 19, offset: 8735},
									run: (*parser).callontop6,
									expr: &seqExpr{
										pos: position{line: 348, col: 19, offset: 8735},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 348, col: 19, offset: 8735},
												name: "_",
											},
											&labeledExpr{
												pos:   position{line: 348, col: 21, offset: 8737},
												label: "n",
												expr: &ruleRefExpr{
													pos:  position{line: 348, col: 23, offset: 8739},
													name: "integer",
												},
											},
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 348, col: 50, offset: 8766},
							label: "flush",
							expr: &zeroOrOneExpr{
								pos: position{line: 348, col: 56, offset: 8772},
								expr: &seqExpr{
									pos: position{line: 348, col: 57, offset: 8773},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 348, col: 57, offset: 8773},
											name: "_",
										},
										&litMatcher{
											pos:        position{line: 348, col: 59, offset: 8775},
											val:        "-flush",
											ignoreCase: false,
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 348, col: 70, offset: 8786},
							label: "list",
							expr: &zeroOrOneExpr{
								pos: position{line: 348, col: 75, offset: 8791},
								expr: &actionExpr{
									pos: position{line: 348, col: 76, offset: 8792},
									run: (*parser).callontop18,
									expr: &seqExpr{
										pos: position{line: 348, col: 76, offset: 8792},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 348, col: 76, offset: 8792},
												name: "_",
											},
											&labeledExpr{
												pos:   position{line: 348, col: 78, offset: 8794},
												label: "f",
												expr: &ruleRefExpr{
													pos:  position{line: 348, col: 80, offset: 8796},
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
			pos:  position{line: 352, col: 1, offset: 8885},
			expr: &actionExpr{
				pos: position{line: 353, col: 5, offset: 8902},
				run: (*parser).callonprocLimitArg1,
				expr: &seqExpr{
					pos: position{line: 353, col: 5, offset: 8902},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 353, col: 5, offset: 8902},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 353, col: 7, offset: 8904},
							val:        "-limit",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 353, col: 16, offset: 8913},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 353, col: 18, offset: 8915},
							label: "limit",
							expr: &ruleRefExpr{
								pos:  position{line: 353, col: 24, offset: 8921},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "cut",
			pos:  position{line: 355, col: 1, offset: 8952},
			expr: &actionExpr{
				pos: position{line: 356, col: 5, offset: 8960},
				run: (*parser).calloncut1,
				expr: &seqExpr{
					pos: position{line: 356, col: 5, offset: 8960},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 356, col: 5, offset: 8960},
							val:        "cut",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 356, col: 12, offset: 8967},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 356, col: 14, offset: 8969},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 356, col: 19, offset: 8974},
								name: "fieldRefDotOnlyList",
							},
						},
					},
				},
			},
		},
		{
			name: "head",
			pos:  position{line: 357, col: 1, offset: 9028},
			expr: &choiceExpr{
				pos: position{line: 358, col: 5, offset: 9037},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 358, col: 5, offset: 9037},
						run: (*parser).callonhead2,
						expr: &seqExpr{
							pos: position{line: 358, col: 5, offset: 9037},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 358, col: 5, offset: 9037},
									val:        "head",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 358, col: 13, offset: 9045},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 358, col: 15, offset: 9047},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 358, col: 21, offset: 9053},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 359, col: 5, offset: 9101},
						run: (*parser).callonhead8,
						expr: &litMatcher{
							pos:        position{line: 359, col: 5, offset: 9101},
							val:        "head",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "tail",
			pos:  position{line: 360, col: 1, offset: 9141},
			expr: &choiceExpr{
				pos: position{line: 361, col: 5, offset: 9150},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 361, col: 5, offset: 9150},
						run: (*parser).callontail2,
						expr: &seqExpr{
							pos: position{line: 361, col: 5, offset: 9150},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 361, col: 5, offset: 9150},
									val:        "tail",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 361, col: 13, offset: 9158},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 361, col: 15, offset: 9160},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 361, col: 21, offset: 9166},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 362, col: 5, offset: 9214},
						run: (*parser).callontail8,
						expr: &litMatcher{
							pos:        position{line: 362, col: 5, offset: 9214},
							val:        "tail",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "filter",
			pos:  position{line: 364, col: 1, offset: 9255},
			expr: &actionExpr{
				pos: position{line: 365, col: 5, offset: 9266},
				run: (*parser).callonfilter1,
				expr: &seqExpr{
					pos: position{line: 365, col: 5, offset: 9266},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 365, col: 5, offset: 9266},
							val:        "filter",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 365, col: 15, offset: 9276},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 365, col: 17, offset: 9278},
							label: "expr",
							expr: &ruleRefExpr{
								pos:  position{line: 365, col: 22, offset: 9283},
								name: "searchExpr",
							},
						},
					},
				},
			},
		},
		{
			name: "uniq",
			pos:  position{line: 368, col: 1, offset: 9341},
			expr: &choiceExpr{
				pos: position{line: 369, col: 5, offset: 9350},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 369, col: 5, offset: 9350},
						run: (*parser).callonuniq2,
						expr: &seqExpr{
							pos: position{line: 369, col: 5, offset: 9350},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 369, col: 5, offset: 9350},
									val:        "uniq",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 369, col: 13, offset: 9358},
									name: "_",
								},
								&litMatcher{
									pos:        position{line: 369, col: 15, offset: 9360},
									val:        "-c",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 372, col: 5, offset: 9414},
						run: (*parser).callonuniq7,
						expr: &litMatcher{
							pos:        position{line: 372, col: 5, offset: 9414},
							val:        "uniq",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "duration",
			pos:  position{line: 376, col: 1, offset: 9469},
			expr: &choiceExpr{
				pos: position{line: 377, col: 5, offset: 9482},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 377, col: 5, offset: 9482},
						name: "seconds",
					},
					&ruleRefExpr{
						pos:  position{line: 378, col: 5, offset: 9494},
						name: "minutes",
					},
					&ruleRefExpr{
						pos:  position{line: 379, col: 5, offset: 9506},
						name: "hours",
					},
					&seqExpr{
						pos: position{line: 380, col: 5, offset: 9516},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 380, col: 5, offset: 9516},
								name: "hours",
							},
							&ruleRefExpr{
								pos:  position{line: 380, col: 11, offset: 9522},
								name: "_",
							},
							&litMatcher{
								pos:        position{line: 380, col: 13, offset: 9524},
								val:        "and",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 380, col: 19, offset: 9530},
								name: "_",
							},
							&ruleRefExpr{
								pos:  position{line: 380, col: 21, offset: 9532},
								name: "minutes",
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 381, col: 5, offset: 9544},
						name: "days",
					},
					&ruleRefExpr{
						pos:  position{line: 382, col: 5, offset: 9553},
						name: "weeks",
					},
				},
			},
		},
		{
			name: "sec_abbrev",
			pos:  position{line: 384, col: 1, offset: 9560},
			expr: &choiceExpr{
				pos: position{line: 385, col: 5, offset: 9575},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 385, col: 5, offset: 9575},
						val:        "seconds",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 386, col: 5, offset: 9589},
						val:        "second",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 387, col: 5, offset: 9602},
						val:        "secs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 388, col: 5, offset: 9613},
						val:        "sec",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 389, col: 5, offset: 9623},
						val:        "s",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "min_abbrev",
			pos:  position{line: 391, col: 1, offset: 9628},
			expr: &choiceExpr{
				pos: position{line: 392, col: 5, offset: 9643},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 392, col: 5, offset: 9643},
						val:        "minutes",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 393, col: 5, offset: 9657},
						val:        "minute",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 394, col: 5, offset: 9670},
						val:        "mins",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 395, col: 5, offset: 9681},
						val:        "min",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 396, col: 5, offset: 9691},
						val:        "m",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "hour_abbrev",
			pos:  position{line: 398, col: 1, offset: 9696},
			expr: &choiceExpr{
				pos: position{line: 399, col: 5, offset: 9712},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 399, col: 5, offset: 9712},
						val:        "hours",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 400, col: 5, offset: 9724},
						val:        "hrs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 401, col: 5, offset: 9734},
						val:        "hr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 402, col: 5, offset: 9743},
						val:        "h",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 403, col: 5, offset: 9751},
						val:        "hour",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "day_abbrev",
			pos:  position{line: 405, col: 1, offset: 9759},
			expr: &choiceExpr{
				pos: position{line: 405, col: 14, offset: 9772},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 405, col: 14, offset: 9772},
						val:        "days",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 405, col: 21, offset: 9779},
						val:        "day",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 405, col: 27, offset: 9785},
						val:        "d",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "week_abbrev",
			pos:  position{line: 406, col: 1, offset: 9789},
			expr: &choiceExpr{
				pos: position{line: 406, col: 15, offset: 9803},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 406, col: 15, offset: 9803},
						val:        "weeks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 406, col: 23, offset: 9811},
						val:        "week",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 406, col: 30, offset: 9818},
						val:        "wks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 406, col: 36, offset: 9824},
						val:        "wk",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 406, col: 41, offset: 9829},
						val:        "w",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "seconds",
			pos:  position{line: 408, col: 1, offset: 9834},
			expr: &choiceExpr{
				pos: position{line: 409, col: 5, offset: 9846},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 409, col: 5, offset: 9846},
						run: (*parser).callonseconds2,
						expr: &litMatcher{
							pos:        position{line: 409, col: 5, offset: 9846},
							val:        "second",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 410, col: 5, offset: 9891},
						run: (*parser).callonseconds4,
						expr: &seqExpr{
							pos: position{line: 410, col: 5, offset: 9891},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 410, col: 5, offset: 9891},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 410, col: 9, offset: 9895},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 410, col: 16, offset: 9902},
									expr: &ruleRefExpr{
										pos:  position{line: 410, col: 16, offset: 9902},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 410, col: 19, offset: 9905},
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
			pos:  position{line: 412, col: 1, offset: 9951},
			expr: &choiceExpr{
				pos: position{line: 413, col: 5, offset: 9963},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 413, col: 5, offset: 9963},
						run: (*parser).callonminutes2,
						expr: &litMatcher{
							pos:        position{line: 413, col: 5, offset: 9963},
							val:        "minute",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 414, col: 5, offset: 10009},
						run: (*parser).callonminutes4,
						expr: &seqExpr{
							pos: position{line: 414, col: 5, offset: 10009},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 414, col: 5, offset: 10009},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 414, col: 9, offset: 10013},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 414, col: 16, offset: 10020},
									expr: &ruleRefExpr{
										pos:  position{line: 414, col: 16, offset: 10020},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 414, col: 19, offset: 10023},
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
			pos:  position{line: 416, col: 1, offset: 10078},
			expr: &choiceExpr{
				pos: position{line: 417, col: 5, offset: 10088},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 417, col: 5, offset: 10088},
						run: (*parser).callonhours2,
						expr: &litMatcher{
							pos:        position{line: 417, col: 5, offset: 10088},
							val:        "hour",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 418, col: 5, offset: 10134},
						run: (*parser).callonhours4,
						expr: &seqExpr{
							pos: position{line: 418, col: 5, offset: 10134},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 418, col: 5, offset: 10134},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 418, col: 9, offset: 10138},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 418, col: 16, offset: 10145},
									expr: &ruleRefExpr{
										pos:  position{line: 418, col: 16, offset: 10145},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 418, col: 19, offset: 10148},
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
			pos:  position{line: 420, col: 1, offset: 10206},
			expr: &choiceExpr{
				pos: position{line: 421, col: 5, offset: 10215},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 421, col: 5, offset: 10215},
						run: (*parser).callondays2,
						expr: &litMatcher{
							pos:        position{line: 421, col: 5, offset: 10215},
							val:        "day",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 422, col: 5, offset: 10263},
						run: (*parser).callondays4,
						expr: &seqExpr{
							pos: position{line: 422, col: 5, offset: 10263},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 422, col: 5, offset: 10263},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 422, col: 9, offset: 10267},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 422, col: 16, offset: 10274},
									expr: &ruleRefExpr{
										pos:  position{line: 422, col: 16, offset: 10274},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 422, col: 19, offset: 10277},
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
			pos:  position{line: 424, col: 1, offset: 10337},
			expr: &actionExpr{
				pos: position{line: 425, col: 5, offset: 10347},
				run: (*parser).callonweeks1,
				expr: &seqExpr{
					pos: position{line: 425, col: 5, offset: 10347},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 425, col: 5, offset: 10347},
							label: "num",
							expr: &ruleRefExpr{
								pos:  position{line: 425, col: 9, offset: 10351},
								name: "number",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 425, col: 16, offset: 10358},
							expr: &ruleRefExpr{
								pos:  position{line: 425, col: 16, offset: 10358},
								name: "_",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 425, col: 19, offset: 10361},
							name: "week_abbrev",
						},
					},
				},
			},
		},
		{
			name: "number",
			pos:  position{line: 427, col: 1, offset: 10424},
			expr: &ruleRefExpr{
				pos:  position{line: 427, col: 10, offset: 10433},
				name: "integer",
			},
		},
		{
			name: "addr",
			pos:  position{line: 431, col: 1, offset: 10471},
			expr: &actionExpr{
				pos: position{line: 432, col: 5, offset: 10480},
				run: (*parser).callonaddr1,
				expr: &labeledExpr{
					pos:   position{line: 432, col: 5, offset: 10480},
					label: "a",
					expr: &seqExpr{
						pos: position{line: 432, col: 8, offset: 10483},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 432, col: 8, offset: 10483},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 432, col: 16, offset: 10491},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 432, col: 20, offset: 10495},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 432, col: 28, offset: 10503},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 432, col: 32, offset: 10507},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 432, col: 40, offset: 10515},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 432, col: 44, offset: 10519},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "port",
			pos:  position{line: 434, col: 1, offset: 10560},
			expr: &actionExpr{
				pos: position{line: 435, col: 5, offset: 10569},
				run: (*parser).callonport1,
				expr: &seqExpr{
					pos: position{line: 435, col: 5, offset: 10569},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 435, col: 5, offset: 10569},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 435, col: 9, offset: 10573},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 435, col: 11, offset: 10575},
								name: "sinteger",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6addr",
			pos:  position{line: 439, col: 1, offset: 10734},
			expr: &choiceExpr{
				pos: position{line: 440, col: 5, offset: 10746},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 440, col: 5, offset: 10746},
						run: (*parser).callonip6addr2,
						expr: &seqExpr{
							pos: position{line: 440, col: 5, offset: 10746},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 440, col: 5, offset: 10746},
									label: "a",
									expr: &oneOrMoreExpr{
										pos: position{line: 440, col: 7, offset: 10748},
										expr: &ruleRefExpr{
											pos:  position{line: 440, col: 8, offset: 10749},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 440, col: 20, offset: 10761},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 440, col: 22, offset: 10763},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 443, col: 5, offset: 10827},
						run: (*parser).callonip6addr9,
						expr: &seqExpr{
							pos: position{line: 443, col: 5, offset: 10827},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 443, col: 5, offset: 10827},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 443, col: 7, offset: 10829},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 443, col: 11, offset: 10833},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 443, col: 13, offset: 10835},
										expr: &ruleRefExpr{
											pos:  position{line: 443, col: 14, offset: 10836},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 443, col: 25, offset: 10847},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 443, col: 30, offset: 10852},
									label: "d",
									expr: &zeroOrMoreExpr{
										pos: position{line: 443, col: 32, offset: 10854},
										expr: &ruleRefExpr{
											pos:  position{line: 443, col: 33, offset: 10855},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 443, col: 45, offset: 10867},
									label: "e",
									expr: &ruleRefExpr{
										pos:  position{line: 443, col: 47, offset: 10869},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 446, col: 5, offset: 10968},
						run: (*parser).callonip6addr22,
						expr: &seqExpr{
							pos: position{line: 446, col: 5, offset: 10968},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 446, col: 5, offset: 10968},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 446, col: 10, offset: 10973},
									label: "a",
									expr: &zeroOrMoreExpr{
										pos: position{line: 446, col: 12, offset: 10975},
										expr: &ruleRefExpr{
											pos:  position{line: 446, col: 13, offset: 10976},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 446, col: 25, offset: 10988},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 446, col: 27, offset: 10990},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 449, col: 5, offset: 11061},
						run: (*parser).callonip6addr30,
						expr: &seqExpr{
							pos: position{line: 449, col: 5, offset: 11061},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 449, col: 5, offset: 11061},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 449, col: 7, offset: 11063},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 449, col: 11, offset: 11067},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 449, col: 13, offset: 11069},
										expr: &ruleRefExpr{
											pos:  position{line: 449, col: 14, offset: 11070},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 449, col: 25, offset: 11081},
									val:        "::",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 452, col: 5, offset: 11149},
						run: (*parser).callonip6addr38,
						expr: &litMatcher{
							pos:        position{line: 452, col: 5, offset: 11149},
							val:        "::",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "ip6tail",
			pos:  position{line: 456, col: 1, offset: 11186},
			expr: &choiceExpr{
				pos: position{line: 457, col: 5, offset: 11198},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 457, col: 5, offset: 11198},
						name: "addr",
					},
					&ruleRefExpr{
						pos:  position{line: 458, col: 5, offset: 11207},
						name: "h16",
					},
				},
			},
		},
		{
			name: "h_append",
			pos:  position{line: 460, col: 1, offset: 11212},
			expr: &actionExpr{
				pos: position{line: 460, col: 12, offset: 11223},
				run: (*parser).callonh_append1,
				expr: &seqExpr{
					pos: position{line: 460, col: 12, offset: 11223},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 460, col: 12, offset: 11223},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 460, col: 16, offset: 11227},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 460, col: 18, offset: 11229},
								name: "h16",
							},
						},
					},
				},
			},
		},
		{
			name: "h_prepend",
			pos:  position{line: 461, col: 1, offset: 11266},
			expr: &actionExpr{
				pos: position{line: 461, col: 13, offset: 11278},
				run: (*parser).callonh_prepend1,
				expr: &seqExpr{
					pos: position{line: 461, col: 13, offset: 11278},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 461, col: 13, offset: 11278},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 461, col: 15, offset: 11280},
								name: "h16",
							},
						},
						&litMatcher{
							pos:        position{line: 461, col: 19, offset: 11284},
							val:        ":",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "sub_addr",
			pos:  position{line: 463, col: 1, offset: 11322},
			expr: &choiceExpr{
				pos: position{line: 464, col: 5, offset: 11335},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 464, col: 5, offset: 11335},
						name: "addr",
					},
					&actionExpr{
						pos: position{line: 465, col: 5, offset: 11344},
						run: (*parser).callonsub_addr3,
						expr: &labeledExpr{
							pos:   position{line: 465, col: 5, offset: 11344},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 465, col: 8, offset: 11347},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 465, col: 8, offset: 11347},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 465, col: 16, offset: 11355},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 465, col: 20, offset: 11359},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 465, col: 28, offset: 11367},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 465, col: 32, offset: 11371},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 466, col: 5, offset: 11423},
						run: (*parser).callonsub_addr11,
						expr: &labeledExpr{
							pos:   position{line: 466, col: 5, offset: 11423},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 466, col: 8, offset: 11426},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 466, col: 8, offset: 11426},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 466, col: 16, offset: 11434},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 466, col: 20, offset: 11438},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 467, col: 5, offset: 11492},
						run: (*parser).callonsub_addr17,
						expr: &labeledExpr{
							pos:   position{line: 467, col: 5, offset: 11492},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 467, col: 7, offset: 11494},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "subnet",
			pos:  position{line: 469, col: 1, offset: 11545},
			expr: &actionExpr{
				pos: position{line: 470, col: 5, offset: 11556},
				run: (*parser).callonsubnet1,
				expr: &seqExpr{
					pos: position{line: 470, col: 5, offset: 11556},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 470, col: 5, offset: 11556},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 470, col: 7, offset: 11558},
								name: "sub_addr",
							},
						},
						&litMatcher{
							pos:        position{line: 470, col: 16, offset: 11567},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 470, col: 20, offset: 11571},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 470, col: 22, offset: 11573},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6subnet",
			pos:  position{line: 474, col: 1, offset: 11649},
			expr: &actionExpr{
				pos: position{line: 475, col: 5, offset: 11663},
				run: (*parser).callonip6subnet1,
				expr: &seqExpr{
					pos: position{line: 475, col: 5, offset: 11663},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 475, col: 5, offset: 11663},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 475, col: 7, offset: 11665},
								name: "ip6addr",
							},
						},
						&litMatcher{
							pos:        position{line: 475, col: 15, offset: 11673},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 475, col: 19, offset: 11677},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 475, col: 21, offset: 11679},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "integer",
			pos:  position{line: 479, col: 1, offset: 11745},
			expr: &actionExpr{
				pos: position{line: 480, col: 5, offset: 11757},
				run: (*parser).calloninteger1,
				expr: &labeledExpr{
					pos:   position{line: 480, col: 5, offset: 11757},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 480, col: 7, offset: 11759},
						name: "sinteger",
					},
				},
			},
		},
		{
			name: "sinteger",
			pos:  position{line: 484, col: 1, offset: 11803},
			expr: &actionExpr{
				pos: position{line: 485, col: 5, offset: 11816},
				run: (*parser).callonsinteger1,
				expr: &labeledExpr{
					pos:   position{line: 485, col: 5, offset: 11816},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 485, col: 11, offset: 11822},
						expr: &charClassMatcher{
							pos:        position{line: 485, col: 11, offset: 11822},
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
			pos:  position{line: 489, col: 1, offset: 11867},
			expr: &actionExpr{
				pos: position{line: 490, col: 5, offset: 11878},
				run: (*parser).callondouble1,
				expr: &labeledExpr{
					pos:   position{line: 490, col: 5, offset: 11878},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 490, col: 7, offset: 11880},
						name: "sdouble",
					},
				},
			},
		},
		{
			name: "sdouble",
			pos:  position{line: 494, col: 1, offset: 11927},
			expr: &choiceExpr{
				pos: position{line: 495, col: 5, offset: 11939},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 495, col: 5, offset: 11939},
						run: (*parser).callonsdouble2,
						expr: &seqExpr{
							pos: position{line: 495, col: 5, offset: 11939},
							exprs: []interface{}{
								&oneOrMoreExpr{
									pos: position{line: 495, col: 5, offset: 11939},
									expr: &ruleRefExpr{
										pos:  position{line: 495, col: 5, offset: 11939},
										name: "doubleInteger",
									},
								},
								&litMatcher{
									pos:        position{line: 495, col: 20, offset: 11954},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 495, col: 24, offset: 11958},
									expr: &ruleRefExpr{
										pos:  position{line: 495, col: 24, offset: 11958},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 495, col: 37, offset: 11971},
									expr: &ruleRefExpr{
										pos:  position{line: 495, col: 37, offset: 11971},
										name: "exponentPart",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 498, col: 5, offset: 12030},
						run: (*parser).callonsdouble11,
						expr: &seqExpr{
							pos: position{line: 498, col: 5, offset: 12030},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 498, col: 5, offset: 12030},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 498, col: 9, offset: 12034},
									expr: &ruleRefExpr{
										pos:  position{line: 498, col: 9, offset: 12034},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 498, col: 22, offset: 12047},
									expr: &ruleRefExpr{
										pos:  position{line: 498, col: 22, offset: 12047},
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
			pos:  position{line: 502, col: 1, offset: 12103},
			expr: &choiceExpr{
				pos: position{line: 503, col: 5, offset: 12121},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 503, col: 5, offset: 12121},
						val:        "0",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 504, col: 5, offset: 12129},
						exprs: []interface{}{
							&charClassMatcher{
								pos:        position{line: 504, col: 5, offset: 12129},
								val:        "[1-9]",
								ranges:     []rune{'1', '9'},
								ignoreCase: false,
								inverted:   false,
							},
							&zeroOrMoreExpr{
								pos: position{line: 504, col: 11, offset: 12135},
								expr: &charClassMatcher{
									pos:        position{line: 504, col: 11, offset: 12135},
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
			pos:  position{line: 506, col: 1, offset: 12143},
			expr: &charClassMatcher{
				pos:        position{line: 506, col: 15, offset: 12157},
				val:        "[0-9]",
				ranges:     []rune{'0', '9'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "signedInteger",
			pos:  position{line: 508, col: 1, offset: 12164},
			expr: &seqExpr{
				pos: position{line: 508, col: 17, offset: 12180},
				exprs: []interface{}{
					&zeroOrOneExpr{
						pos: position{line: 508, col: 17, offset: 12180},
						expr: &charClassMatcher{
							pos:        position{line: 508, col: 17, offset: 12180},
							val:        "[+-]",
							chars:      []rune{'+', '-'},
							ignoreCase: false,
							inverted:   false,
						},
					},
					&oneOrMoreExpr{
						pos: position{line: 508, col: 23, offset: 12186},
						expr: &ruleRefExpr{
							pos:  position{line: 508, col: 23, offset: 12186},
							name: "doubleDigit",
						},
					},
				},
			},
		},
		{
			name: "exponentPart",
			pos:  position{line: 510, col: 1, offset: 12200},
			expr: &seqExpr{
				pos: position{line: 510, col: 16, offset: 12215},
				exprs: []interface{}{
					&litMatcher{
						pos:        position{line: 510, col: 16, offset: 12215},
						val:        "e",
						ignoreCase: true,
					},
					&ruleRefExpr{
						pos:  position{line: 510, col: 21, offset: 12220},
						name: "signedInteger",
					},
				},
			},
		},
		{
			name: "h16",
			pos:  position{line: 512, col: 1, offset: 12235},
			expr: &actionExpr{
				pos: position{line: 512, col: 7, offset: 12241},
				run: (*parser).callonh161,
				expr: &labeledExpr{
					pos:   position{line: 512, col: 7, offset: 12241},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 512, col: 13, offset: 12247},
						expr: &ruleRefExpr{
							pos:  position{line: 512, col: 13, offset: 12247},
							name: "hexdigit",
						},
					},
				},
			},
		},
		{
			name: "hexdigit",
			pos:  position{line: 514, col: 1, offset: 12289},
			expr: &charClassMatcher{
				pos:        position{line: 514, col: 12, offset: 12300},
				val:        "[0-9a-fA-F]",
				ranges:     []rune{'0', '9', 'a', 'f', 'A', 'F'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name:        "boomWord",
			displayName: "\"boomWord\"",
			pos:         position{line: 516, col: 1, offset: 12313},
			expr: &actionExpr{
				pos: position{line: 516, col: 23, offset: 12335},
				run: (*parser).callonboomWord1,
				expr: &labeledExpr{
					pos:   position{line: 516, col: 23, offset: 12335},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 516, col: 29, offset: 12341},
						expr: &ruleRefExpr{
							pos:  position{line: 516, col: 29, offset: 12341},
							name: "boomWordPart",
						},
					},
				},
			},
		},
		{
			name: "boomWordPart",
			pos:  position{line: 518, col: 1, offset: 12389},
			expr: &choiceExpr{
				pos: position{line: 519, col: 5, offset: 12406},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 519, col: 5, offset: 12406},
						run: (*parser).callonboomWordPart2,
						expr: &seqExpr{
							pos: position{line: 519, col: 5, offset: 12406},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 519, col: 5, offset: 12406},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 519, col: 10, offset: 12411},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 519, col: 12, offset: 12413},
										name: "escapeSequence",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 520, col: 5, offset: 12451},
						run: (*parser).callonboomWordPart7,
						expr: &seqExpr{
							pos: position{line: 520, col: 5, offset: 12451},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 520, col: 5, offset: 12451},
									expr: &choiceExpr{
										pos: position{line: 520, col: 7, offset: 12453},
										alternatives: []interface{}{
											&charClassMatcher{
												pos:        position{line: 520, col: 7, offset: 12453},
												val:        "[\\x00-\\x1F\\x5C(),!><=\\x22|\\x27;]",
												chars:      []rune{'\\', '(', ')', ',', '!', '>', '<', '=', '"', '|', '\'', ';'},
												ranges:     []rune{'\x00', '\x1f'},
												ignoreCase: false,
												inverted:   false,
											},
											&ruleRefExpr{
												pos:  position{line: 520, col: 42, offset: 12488},
												name: "ws",
											},
										},
									},
								},
								&anyMatcher{
									line: 520, col: 46, offset: 12492,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "quotedString",
			pos:  position{line: 522, col: 1, offset: 12526},
			expr: &choiceExpr{
				pos: position{line: 523, col: 5, offset: 12543},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 523, col: 5, offset: 12543},
						run: (*parser).callonquotedString2,
						expr: &seqExpr{
							pos: position{line: 523, col: 5, offset: 12543},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 523, col: 5, offset: 12543},
									val:        "\"",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 523, col: 9, offset: 12547},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 523, col: 11, offset: 12549},
										expr: &ruleRefExpr{
											pos:  position{line: 523, col: 11, offset: 12549},
											name: "doubleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 523, col: 29, offset: 12567},
									val:        "\"",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 524, col: 5, offset: 12604},
						run: (*parser).callonquotedString9,
						expr: &seqExpr{
							pos: position{line: 524, col: 5, offset: 12604},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 524, col: 5, offset: 12604},
									val:        "'",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 524, col: 9, offset: 12608},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 524, col: 11, offset: 12610},
										expr: &ruleRefExpr{
											pos:  position{line: 524, col: 11, offset: 12610},
											name: "singleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 524, col: 29, offset: 12628},
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
			pos:  position{line: 526, col: 1, offset: 12662},
			expr: &choiceExpr{
				pos: position{line: 527, col: 5, offset: 12683},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 527, col: 5, offset: 12683},
						run: (*parser).callondoubleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 527, col: 5, offset: 12683},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 527, col: 5, offset: 12683},
									expr: &choiceExpr{
										pos: position{line: 527, col: 7, offset: 12685},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 527, col: 7, offset: 12685},
												val:        "\"",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 527, col: 13, offset: 12691},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 527, col: 26, offset: 12704,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 528, col: 5, offset: 12741},
						run: (*parser).callondoubleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 528, col: 5, offset: 12741},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 528, col: 5, offset: 12741},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 528, col: 10, offset: 12746},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 528, col: 12, offset: 12748},
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
			pos:  position{line: 530, col: 1, offset: 12782},
			expr: &choiceExpr{
				pos: position{line: 531, col: 5, offset: 12803},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 531, col: 5, offset: 12803},
						run: (*parser).callonsingleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 531, col: 5, offset: 12803},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 531, col: 5, offset: 12803},
									expr: &choiceExpr{
										pos: position{line: 531, col: 7, offset: 12805},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 531, col: 7, offset: 12805},
												val:        "'",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 531, col: 13, offset: 12811},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 531, col: 26, offset: 12824,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 532, col: 5, offset: 12861},
						run: (*parser).callonsingleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 532, col: 5, offset: 12861},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 532, col: 5, offset: 12861},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 532, col: 10, offset: 12866},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 532, col: 12, offset: 12868},
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
			pos:  position{line: 534, col: 1, offset: 12902},
			expr: &choiceExpr{
				pos: position{line: 535, col: 5, offset: 12921},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 535, col: 5, offset: 12921},
						run: (*parser).callonescapeSequence2,
						expr: &seqExpr{
							pos: position{line: 535, col: 5, offset: 12921},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 535, col: 5, offset: 12921},
									val:        "x",
									ignoreCase: false,
								},
								&ruleRefExpr{
									pos:  position{line: 535, col: 9, offset: 12925},
									name: "hexdigit",
								},
								&ruleRefExpr{
									pos:  position{line: 535, col: 18, offset: 12934},
									name: "hexdigit",
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 536, col: 5, offset: 12985},
						name: "singleCharEscape",
					},
					&ruleRefExpr{
						pos:  position{line: 537, col: 5, offset: 13006},
						name: "unicodeEscape",
					},
				},
			},
		},
		{
			name: "singleCharEscape",
			pos:  position{line: 539, col: 1, offset: 13021},
			expr: &choiceExpr{
				pos: position{line: 540, col: 5, offset: 13042},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 540, col: 5, offset: 13042},
						val:        "'",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 541, col: 5, offset: 13050},
						val:        "\"",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 542, col: 5, offset: 13058},
						val:        "\\",
						ignoreCase: false,
					},
					&actionExpr{
						pos: position{line: 543, col: 5, offset: 13067},
						run: (*parser).callonsingleCharEscape5,
						expr: &litMatcher{
							pos:        position{line: 543, col: 5, offset: 13067},
							val:        "b",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 544, col: 5, offset: 13096},
						run: (*parser).callonsingleCharEscape7,
						expr: &litMatcher{
							pos:        position{line: 544, col: 5, offset: 13096},
							val:        "f",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 545, col: 5, offset: 13125},
						run: (*parser).callonsingleCharEscape9,
						expr: &litMatcher{
							pos:        position{line: 545, col: 5, offset: 13125},
							val:        "n",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 546, col: 5, offset: 13154},
						run: (*parser).callonsingleCharEscape11,
						expr: &litMatcher{
							pos:        position{line: 546, col: 5, offset: 13154},
							val:        "r",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 547, col: 5, offset: 13183},
						run: (*parser).callonsingleCharEscape13,
						expr: &litMatcher{
							pos:        position{line: 547, col: 5, offset: 13183},
							val:        "t",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 548, col: 5, offset: 13212},
						run: (*parser).callonsingleCharEscape15,
						expr: &litMatcher{
							pos:        position{line: 548, col: 5, offset: 13212},
							val:        "v",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 549, col: 5, offset: 13241},
						run: (*parser).callonsingleCharEscape17,
						expr: &litMatcher{
							pos:        position{line: 549, col: 5, offset: 13241},
							val:        "=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 550, col: 5, offset: 13269},
						run: (*parser).callonsingleCharEscape19,
						expr: &litMatcher{
							pos:        position{line: 550, col: 5, offset: 13269},
							val:        "*",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "unicodeEscape",
			pos:  position{line: 552, col: 1, offset: 13296},
			expr: &choiceExpr{
				pos: position{line: 553, col: 5, offset: 13314},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 553, col: 5, offset: 13314},
						run: (*parser).callonunicodeEscape2,
						expr: &seqExpr{
							pos: position{line: 553, col: 5, offset: 13314},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 553, col: 5, offset: 13314},
									val:        "u",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 553, col: 9, offset: 13318},
									label: "chars",
									expr: &seqExpr{
										pos: position{line: 553, col: 16, offset: 13325},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 553, col: 16, offset: 13325},
												name: "hexdigit",
											},
											&ruleRefExpr{
												pos:  position{line: 553, col: 25, offset: 13334},
												name: "hexdigit",
											},
											&ruleRefExpr{
												pos:  position{line: 553, col: 34, offset: 13343},
												name: "hexdigit",
											},
											&ruleRefExpr{
												pos:  position{line: 553, col: 43, offset: 13352},
												name: "hexdigit",
											},
										},
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 556, col: 5, offset: 13415},
						run: (*parser).callonunicodeEscape11,
						expr: &seqExpr{
							pos: position{line: 556, col: 5, offset: 13415},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 556, col: 5, offset: 13415},
									val:        "u",
									ignoreCase: false,
								},
								&litMatcher{
									pos:        position{line: 556, col: 9, offset: 13419},
									val:        "{",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 556, col: 13, offset: 13423},
									label: "chars",
									expr: &seqExpr{
										pos: position{line: 556, col: 20, offset: 13430},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 556, col: 20, offset: 13430},
												name: "hexdigit",
											},
											&zeroOrOneExpr{
												pos: position{line: 556, col: 29, offset: 13439},
												expr: &ruleRefExpr{
													pos:  position{line: 556, col: 29, offset: 13439},
													name: "hexdigit",
												},
											},
											&zeroOrOneExpr{
												pos: position{line: 556, col: 39, offset: 13449},
												expr: &ruleRefExpr{
													pos:  position{line: 556, col: 39, offset: 13449},
													name: "hexdigit",
												},
											},
											&zeroOrOneExpr{
												pos: position{line: 556, col: 49, offset: 13459},
												expr: &ruleRefExpr{
													pos:  position{line: 556, col: 49, offset: 13459},
													name: "hexdigit",
												},
											},
											&zeroOrOneExpr{
												pos: position{line: 556, col: 59, offset: 13469},
												expr: &ruleRefExpr{
													pos:  position{line: 556, col: 59, offset: 13469},
													name: "hexdigit",
												},
											},
											&zeroOrOneExpr{
												pos: position{line: 556, col: 69, offset: 13479},
												expr: &ruleRefExpr{
													pos:  position{line: 556, col: 69, offset: 13479},
													name: "hexdigit",
												},
											},
										},
									},
								},
								&litMatcher{
									pos:        position{line: 556, col: 80, offset: 13490},
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
			pos:  position{line: 560, col: 1, offset: 13544},
			expr: &actionExpr{
				pos: position{line: 561, col: 5, offset: 13557},
				run: (*parser).callonreString1,
				expr: &seqExpr{
					pos: position{line: 561, col: 5, offset: 13557},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 561, col: 5, offset: 13557},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 561, col: 9, offset: 13561},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 561, col: 11, offset: 13563},
								name: "reBody",
							},
						},
						&litMatcher{
							pos:        position{line: 561, col: 18, offset: 13570},
							val:        "/",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reBody",
			pos:  position{line: 563, col: 1, offset: 13593},
			expr: &actionExpr{
				pos: position{line: 564, col: 5, offset: 13604},
				run: (*parser).callonreBody1,
				expr: &oneOrMoreExpr{
					pos: position{line: 564, col: 5, offset: 13604},
					expr: &choiceExpr{
						pos: position{line: 564, col: 6, offset: 13605},
						alternatives: []interface{}{
							&charClassMatcher{
								pos:        position{line: 564, col: 6, offset: 13605},
								val:        "[^/\\\\]",
								chars:      []rune{'/', '\\'},
								ignoreCase: false,
								inverted:   true,
							},
							&litMatcher{
								pos:        position{line: 564, col: 13, offset: 13612},
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
			pos:  position{line: 566, col: 1, offset: 13652},
			expr: &charClassMatcher{
				pos:        position{line: 567, col: 5, offset: 13668},
				val:        "[\\x00-\\x1f\\\\]",
				chars:      []rune{'\\'},
				ranges:     []rune{'\x00', '\x1f'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "ws",
			pos:  position{line: 569, col: 1, offset: 13683},
			expr: &choiceExpr{
				pos: position{line: 570, col: 5, offset: 13690},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 570, col: 5, offset: 13690},
						val:        "\t",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 571, col: 5, offset: 13699},
						val:        "\v",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 572, col: 5, offset: 13708},
						val:        "\f",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 573, col: 5, offset: 13717},
						val:        " ",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 574, col: 5, offset: 13725},
						val:        "\u00a0",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 575, col: 5, offset: 13738},
						val:        "\ufeff",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name:        "_",
			displayName: "\"whitespace\"",
			pos:         position{line: 577, col: 1, offset: 13748},
			expr: &oneOrMoreExpr{
				pos: position{line: 577, col: 18, offset: 13765},
				expr: &ruleRefExpr{
					pos:  position{line: 577, col: 18, offset: 13765},
					name: "ws",
				},
			},
		},
		{
			name: "EOF",
			pos:  position{line: 579, col: 1, offset: 13770},
			expr: &notExpr{
				pos: position{line: 579, col: 7, offset: 13776},
				expr: &anyMatcher{
					line: 579, col: 8, offset: 13777,
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

	return makeOrChain(makeCompareAny("search", true, makeLiteral("string", string(c.text))), []interface{}{makeCompareAny("searchin", true, makeLiteral("string", string(c.text))), makeCompareAny("eql", true, v), makeCompareAny("in", true, v)}), nil

}

func (p *parser) callonsearchPred57() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchPred57(stack["v"])
}

func (c *current) onsearchValue2(v interface{}) (interface{}, error) {
	return makeLiteral("string", v), nil

}

func (p *parser) callonsearchValue2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchValue2(stack["v"])
}

func (c *current) onsearchValue5(v interface{}) (interface{}, error) {
	return makeLiteral("regexp", v), nil

}

func (p *parser) callonsearchValue5() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchValue5(stack["v"])
}

func (c *current) onsearchValue8(v interface{}) (interface{}, error) {
	return makeLiteral("port", v), nil

}

func (p *parser) callonsearchValue8() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchValue8(stack["v"])
}

func (c *current) onsearchValue11(v interface{}) (interface{}, error) {
	return makeLiteral("subnet", v), nil

}

func (p *parser) callonsearchValue11() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchValue11(stack["v"])
}

func (c *current) onsearchValue14(v interface{}) (interface{}, error) {
	return makeLiteral("addr", v), nil

}

func (p *parser) callonsearchValue14() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchValue14(stack["v"])
}

func (c *current) onsearchValue17(v interface{}) (interface{}, error) {
	return makeLiteral("subnet", v), nil

}

func (p *parser) callonsearchValue17() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchValue17(stack["v"])
}

func (c *current) onsearchValue20(v interface{}) (interface{}, error) {
	return makeLiteral("addr", v), nil

}

func (p *parser) callonsearchValue20() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchValue20(stack["v"])
}

func (c *current) onsearchValue23(v interface{}) (interface{}, error) {
	return makeLiteral("double", v), nil

}

func (p *parser) callonsearchValue23() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchValue23(stack["v"])
}

func (c *current) onsearchValue26(v interface{}) (interface{}, error) {
	return makeLiteral("int", v), nil

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
		return makeLiteral("regexp", re), nil
	}
	return makeLiteral("string", v), nil

}

func (p *parser) callonsearchValue48() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchValue48(stack["v"])
}

func (c *current) onbooleanLiteral2() (interface{}, error) {
	return makeLiteral("bool", "true"), nil
}

func (p *parser) callonbooleanLiteral2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onbooleanLiteral2()
}

func (c *current) onbooleanLiteral4() (interface{}, error) {
	return makeLiteral("bool", "false"), nil
}

func (p *parser) callonbooleanLiteral4() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onbooleanLiteral4()
}

func (c *current) onunsetLiteral1() (interface{}, error) {
	return makeLiteral("unset", ""), nil
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

func (c *current) onsort8(l interface{}) (interface{}, error) {
	return l, nil
}

func (p *parser) callonsort8() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsort8(stack["l"])
}

func (c *current) onsort1(args, list interface{}) (interface{}, error) {
	return makeSortProc(args, list)

}

func (p *parser) callonsort1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsort1(stack["args"], stack["list"])
}

func (c *current) onsortArgs2(a interface{}) (interface{}, error) {
	return a, nil
}

func (p *parser) callonsortArgs2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsortArgs2(stack["a"])
}

func (c *current) onsortArg2(limit interface{}) (interface{}, error) {
	return makeArg("limit", limit), nil
}

func (p *parser) callonsortArg2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsortArg2(stack["limit"])
}

func (c *current) onsortArg8() (interface{}, error) {
	return makeArg("r", nil), nil
}

func (p *parser) callonsortArg8() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsortArg8()
}

func (c *current) onsortArg15() (interface{}, error) {
	return string(c.text), nil
}

func (p *parser) callonsortArg15() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsortArg15()
}

func (c *current) onsortArg10(where interface{}) (interface{}, error) {
	return makeArg("nulls", where), nil
}

func (p *parser) callonsortArg10() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsortArg10(stack["where"])
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
	return "=", nil
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

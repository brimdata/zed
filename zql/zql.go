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
								name: "query",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 12, col: 22, offset: 90},
							expr: &ruleRefExpr{
								pos:  position{line: 12, col: 22, offset: 90},
								name: "_",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 12, col: 25, offset: 93},
							name: "EOF",
						},
					},
				},
			},
		},
		{
			name: "query",
			pos:  position{line: 14, col: 1, offset: 118},
			expr: &choiceExpr{
				pos: position{line: 15, col: 5, offset: 128},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 15, col: 5, offset: 128},
						run: (*parser).callonquery2,
						expr: &labeledExpr{
							pos:   position{line: 15, col: 5, offset: 128},
							label: "procs",
							expr: &ruleRefExpr{
								pos:  position{line: 15, col: 11, offset: 134},
								name: "procChain",
							},
						},
					},
					&actionExpr{
						pos: position{line: 19, col: 5, offset: 307},
						run: (*parser).callonquery5,
						expr: &seqExpr{
							pos: position{line: 19, col: 5, offset: 307},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 19, col: 5, offset: 307},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 19, col: 7, offset: 309},
										name: "search",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 19, col: 14, offset: 316},
									expr: &ruleRefExpr{
										pos:  position{line: 19, col: 14, offset: 316},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 19, col: 17, offset: 319},
									label: "rest",
									expr: &zeroOrMoreExpr{
										pos: position{line: 19, col: 22, offset: 324},
										expr: &ruleRefExpr{
											pos:  position{line: 19, col: 22, offset: 324},
											name: "chainedProc",
										},
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 26, col: 5, offset: 534},
						run: (*parser).callonquery14,
						expr: &labeledExpr{
							pos:   position{line: 26, col: 5, offset: 534},
							label: "s",
							expr: &ruleRefExpr{
								pos:  position{line: 26, col: 7, offset: 536},
								name: "search",
							},
						},
					},
				},
			},
		},
		{
			name: "procChain",
			pos:  position{line: 30, col: 1, offset: 607},
			expr: &actionExpr{
				pos: position{line: 31, col: 5, offset: 621},
				run: (*parser).callonprocChain1,
				expr: &seqExpr{
					pos: position{line: 31, col: 5, offset: 621},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 31, col: 5, offset: 621},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 31, col: 11, offset: 627},
								name: "proc",
							},
						},
						&labeledExpr{
							pos:   position{line: 31, col: 16, offset: 632},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 31, col: 21, offset: 637},
								expr: &ruleRefExpr{
									pos:  position{line: 31, col: 21, offset: 637},
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
			pos:  position{line: 39, col: 1, offset: 823},
			expr: &actionExpr{
				pos: position{line: 39, col: 15, offset: 837},
				run: (*parser).callonchainedProc1,
				expr: &seqExpr{
					pos: position{line: 39, col: 15, offset: 837},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 39, col: 15, offset: 837},
							expr: &ruleRefExpr{
								pos:  position{line: 39, col: 15, offset: 837},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 39, col: 18, offset: 840},
							val:        "|",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 39, col: 22, offset: 844},
							expr: &ruleRefExpr{
								pos:  position{line: 39, col: 22, offset: 844},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 39, col: 25, offset: 847},
							label: "p",
							expr: &ruleRefExpr{
								pos:  position{line: 39, col: 27, offset: 849},
								name: "proc",
							},
						},
					},
				},
			},
		},
		{
			name: "search",
			pos:  position{line: 41, col: 1, offset: 873},
			expr: &actionExpr{
				pos: position{line: 42, col: 5, offset: 884},
				run: (*parser).callonsearch1,
				expr: &labeledExpr{
					pos:   position{line: 42, col: 5, offset: 884},
					label: "expr",
					expr: &ruleRefExpr{
						pos:  position{line: 42, col: 10, offset: 889},
						name: "searchExpr",
					},
				},
			},
		},
		{
			name: "searchExpr",
			pos:  position{line: 46, col: 1, offset: 948},
			expr: &actionExpr{
				pos: position{line: 47, col: 5, offset: 963},
				run: (*parser).callonsearchExpr1,
				expr: &seqExpr{
					pos: position{line: 47, col: 5, offset: 963},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 47, col: 5, offset: 963},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 47, col: 11, offset: 969},
								name: "searchTerm",
							},
						},
						&labeledExpr{
							pos:   position{line: 47, col: 22, offset: 980},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 47, col: 27, offset: 985},
								expr: &ruleRefExpr{
									pos:  position{line: 47, col: 27, offset: 985},
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
			pos:  position{line: 51, col: 1, offset: 1053},
			expr: &actionExpr{
				pos: position{line: 51, col: 18, offset: 1070},
				run: (*parser).callonoredSearchTerm1,
				expr: &seqExpr{
					pos: position{line: 51, col: 18, offset: 1070},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 51, col: 18, offset: 1070},
							name: "_",
						},
						&ruleRefExpr{
							pos:  position{line: 51, col: 20, offset: 1072},
							name: "orToken",
						},
						&ruleRefExpr{
							pos:  position{line: 51, col: 28, offset: 1080},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 51, col: 30, offset: 1082},
							label: "t",
							expr: &ruleRefExpr{
								pos:  position{line: 51, col: 32, offset: 1084},
								name: "searchTerm",
							},
						},
					},
				},
			},
		},
		{
			name: "searchTerm",
			pos:  position{line: 53, col: 1, offset: 1114},
			expr: &actionExpr{
				pos: position{line: 54, col: 5, offset: 1129},
				run: (*parser).callonsearchTerm1,
				expr: &seqExpr{
					pos: position{line: 54, col: 5, offset: 1129},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 54, col: 5, offset: 1129},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 54, col: 11, offset: 1135},
								name: "searchFactor",
							},
						},
						&labeledExpr{
							pos:   position{line: 54, col: 24, offset: 1148},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 54, col: 29, offset: 1153},
								expr: &ruleRefExpr{
									pos:  position{line: 54, col: 29, offset: 1153},
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
			pos:  position{line: 58, col: 1, offset: 1223},
			expr: &actionExpr{
				pos: position{line: 58, col: 19, offset: 1241},
				run: (*parser).callonandedSearchTerm1,
				expr: &seqExpr{
					pos: position{line: 58, col: 19, offset: 1241},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 58, col: 19, offset: 1241},
							name: "_",
						},
						&zeroOrOneExpr{
							pos: position{line: 58, col: 21, offset: 1243},
							expr: &seqExpr{
								pos: position{line: 58, col: 22, offset: 1244},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 58, col: 22, offset: 1244},
										name: "andToken",
									},
									&ruleRefExpr{
										pos:  position{line: 58, col: 31, offset: 1253},
										name: "_",
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 58, col: 35, offset: 1257},
							label: "f",
							expr: &ruleRefExpr{
								pos:  position{line: 58, col: 37, offset: 1259},
								name: "searchFactor",
							},
						},
					},
				},
			},
		},
		{
			name: "searchFactor",
			pos:  position{line: 60, col: 1, offset: 1291},
			expr: &choiceExpr{
				pos: position{line: 61, col: 5, offset: 1308},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 61, col: 5, offset: 1308},
						run: (*parser).callonsearchFactor2,
						expr: &seqExpr{
							pos: position{line: 61, col: 5, offset: 1308},
							exprs: []interface{}{
								&choiceExpr{
									pos: position{line: 61, col: 6, offset: 1309},
									alternatives: []interface{}{
										&seqExpr{
											pos: position{line: 61, col: 6, offset: 1309},
											exprs: []interface{}{
												&ruleRefExpr{
													pos:  position{line: 61, col: 6, offset: 1309},
													name: "notToken",
												},
												&ruleRefExpr{
													pos:  position{line: 61, col: 15, offset: 1318},
													name: "_",
												},
											},
										},
										&seqExpr{
											pos: position{line: 61, col: 19, offset: 1322},
											exprs: []interface{}{
												&litMatcher{
													pos:        position{line: 61, col: 19, offset: 1322},
													val:        "!",
													ignoreCase: false,
												},
												&zeroOrOneExpr{
													pos: position{line: 61, col: 23, offset: 1326},
													expr: &ruleRefExpr{
														pos:  position{line: 61, col: 23, offset: 1326},
														name: "_",
													},
												},
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 61, col: 27, offset: 1330},
									label: "e",
									expr: &ruleRefExpr{
										pos:  position{line: 61, col: 29, offset: 1332},
										name: "searchExpr",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 64, col: 5, offset: 1391},
						run: (*parser).callonsearchFactor14,
						expr: &seqExpr{
							pos: position{line: 64, col: 5, offset: 1391},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 64, col: 5, offset: 1391},
									expr: &litMatcher{
										pos:        position{line: 64, col: 7, offset: 1393},
										val:        "-",
										ignoreCase: false,
									},
								},
								&labeledExpr{
									pos:   position{line: 64, col: 12, offset: 1398},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 64, col: 14, offset: 1400},
										name: "searchPred",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 65, col: 5, offset: 1433},
						run: (*parser).callonsearchFactor20,
						expr: &seqExpr{
							pos: position{line: 65, col: 5, offset: 1433},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 65, col: 5, offset: 1433},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 65, col: 9, offset: 1437},
									expr: &ruleRefExpr{
										pos:  position{line: 65, col: 9, offset: 1437},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 65, col: 12, offset: 1440},
									label: "expr",
									expr: &ruleRefExpr{
										pos:  position{line: 65, col: 17, offset: 1445},
										name: "searchExpr",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 65, col: 28, offset: 1456},
									expr: &ruleRefExpr{
										pos:  position{line: 65, col: 28, offset: 1456},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 65, col: 31, offset: 1459},
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
			pos:  position{line: 67, col: 1, offset: 1485},
			expr: &choiceExpr{
				pos: position{line: 68, col: 5, offset: 1500},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 68, col: 5, offset: 1500},
						run: (*parser).callonsearchPred2,
						expr: &seqExpr{
							pos: position{line: 68, col: 5, offset: 1500},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 68, col: 5, offset: 1500},
									val:        "*",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 68, col: 9, offset: 1504},
									expr: &ruleRefExpr{
										pos:  position{line: 68, col: 9, offset: 1504},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 68, col: 12, offset: 1507},
									label: "fieldComparator",
									expr: &ruleRefExpr{
										pos:  position{line: 68, col: 28, offset: 1523},
										name: "equalityToken",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 68, col: 42, offset: 1537},
									expr: &ruleRefExpr{
										pos:  position{line: 68, col: 42, offset: 1537},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 68, col: 45, offset: 1540},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 68, col: 47, offset: 1542},
										name: "searchValue",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 71, col: 5, offset: 1626},
						run: (*parser).callonsearchPred13,
						expr: &seqExpr{
							pos: position{line: 71, col: 5, offset: 1626},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 71, col: 5, offset: 1626},
									val:        "**",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 71, col: 10, offset: 1631},
									expr: &ruleRefExpr{
										pos:  position{line: 71, col: 10, offset: 1631},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 71, col: 13, offset: 1634},
									label: "fieldComparator",
									expr: &ruleRefExpr{
										pos:  position{line: 71, col: 29, offset: 1650},
										name: "equalityToken",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 71, col: 43, offset: 1664},
									expr: &ruleRefExpr{
										pos:  position{line: 71, col: 43, offset: 1664},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 71, col: 46, offset: 1667},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 71, col: 48, offset: 1669},
										name: "searchValue",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 74, col: 5, offset: 1752},
						run: (*parser).callonsearchPred24,
						expr: &seqExpr{
							pos: position{line: 74, col: 5, offset: 1752},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 74, col: 5, offset: 1752},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 74, col: 7, offset: 1754},
										name: "fieldExpr",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 74, col: 17, offset: 1764},
									expr: &ruleRefExpr{
										pos:  position{line: 74, col: 17, offset: 1764},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 74, col: 20, offset: 1767},
									label: "fieldComparator",
									expr: &ruleRefExpr{
										pos:  position{line: 74, col: 36, offset: 1783},
										name: "equalityToken",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 74, col: 50, offset: 1797},
									expr: &ruleRefExpr{
										pos:  position{line: 74, col: 50, offset: 1797},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 74, col: 53, offset: 1800},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 74, col: 55, offset: 1802},
										name: "searchValue",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 77, col: 5, offset: 1884},
						run: (*parser).callonsearchPred36,
						expr: &seqExpr{
							pos: position{line: 77, col: 5, offset: 1884},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 77, col: 5, offset: 1884},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 77, col: 7, offset: 1886},
										name: "searchValue",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 77, col: 19, offset: 1898},
									expr: &ruleRefExpr{
										pos:  position{line: 77, col: 19, offset: 1898},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 77, col: 22, offset: 1901},
									name: "inToken",
								},
								&zeroOrOneExpr{
									pos: position{line: 77, col: 30, offset: 1909},
									expr: &ruleRefExpr{
										pos:  position{line: 77, col: 30, offset: 1909},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 77, col: 33, offset: 1912},
									val:        "*",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 80, col: 5, offset: 1977},
						run: (*parser).callonsearchPred46,
						expr: &seqExpr{
							pos: position{line: 80, col: 5, offset: 1977},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 80, col: 5, offset: 1977},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 80, col: 7, offset: 1979},
										name: "searchValue",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 80, col: 19, offset: 1991},
									expr: &ruleRefExpr{
										pos:  position{line: 80, col: 19, offset: 1991},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 80, col: 22, offset: 1994},
									name: "inToken",
								},
								&zeroOrOneExpr{
									pos: position{line: 80, col: 30, offset: 2002},
									expr: &ruleRefExpr{
										pos:  position{line: 80, col: 30, offset: 2002},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 80, col: 33, offset: 2005},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 80, col: 35, offset: 2007},
										name: "fieldReference",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 83, col: 5, offset: 2081},
						run: (*parser).callonsearchPred57,
						expr: &labeledExpr{
							pos:   position{line: 83, col: 5, offset: 2081},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 83, col: 7, offset: 2083},
								name: "searchValue",
							},
						},
					},
				},
			},
		},
		{
			name: "searchValue",
			pos:  position{line: 97, col: 1, offset: 2772},
			expr: &choiceExpr{
				pos: position{line: 98, col: 5, offset: 2788},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 98, col: 5, offset: 2788},
						run: (*parser).callonsearchValue2,
						expr: &labeledExpr{
							pos:   position{line: 98, col: 5, offset: 2788},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 98, col: 7, offset: 2790},
								name: "quotedString",
							},
						},
					},
					&actionExpr{
						pos: position{line: 101, col: 5, offset: 2858},
						run: (*parser).callonsearchValue5,
						expr: &labeledExpr{
							pos:   position{line: 101, col: 5, offset: 2858},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 101, col: 7, offset: 2860},
								name: "reString",
							},
						},
					},
					&actionExpr{
						pos: position{line: 104, col: 5, offset: 2924},
						run: (*parser).callonsearchValue8,
						expr: &labeledExpr{
							pos:   position{line: 104, col: 5, offset: 2924},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 104, col: 7, offset: 2926},
								name: "port",
							},
						},
					},
					&actionExpr{
						pos: position{line: 107, col: 5, offset: 2982},
						run: (*parser).callonsearchValue11,
						expr: &labeledExpr{
							pos:   position{line: 107, col: 5, offset: 2982},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 107, col: 7, offset: 2984},
								name: "ip6subnet",
							},
						},
					},
					&actionExpr{
						pos: position{line: 110, col: 5, offset: 3049},
						run: (*parser).callonsearchValue14,
						expr: &labeledExpr{
							pos:   position{line: 110, col: 5, offset: 3049},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 110, col: 7, offset: 3051},
								name: "ip6addr",
							},
						},
					},
					&actionExpr{
						pos: position{line: 113, col: 5, offset: 3112},
						run: (*parser).callonsearchValue17,
						expr: &labeledExpr{
							pos:   position{line: 113, col: 5, offset: 3112},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 113, col: 7, offset: 3114},
								name: "subnet",
							},
						},
					},
					&actionExpr{
						pos: position{line: 116, col: 5, offset: 3176},
						run: (*parser).callonsearchValue20,
						expr: &labeledExpr{
							pos:   position{line: 116, col: 5, offset: 3176},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 116, col: 7, offset: 3178},
								name: "addr",
							},
						},
					},
					&actionExpr{
						pos: position{line: 119, col: 5, offset: 3236},
						run: (*parser).callonsearchValue23,
						expr: &labeledExpr{
							pos:   position{line: 119, col: 5, offset: 3236},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 119, col: 7, offset: 3238},
								name: "sdouble",
							},
						},
					},
					&actionExpr{
						pos: position{line: 122, col: 5, offset: 3301},
						run: (*parser).callonsearchValue26,
						expr: &seqExpr{
							pos: position{line: 122, col: 5, offset: 3301},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 122, col: 5, offset: 3301},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 122, col: 7, offset: 3303},
										name: "sinteger",
									},
								},
								&notExpr{
									pos: position{line: 122, col: 16, offset: 3312},
									expr: &ruleRefExpr{
										pos:  position{line: 122, col: 17, offset: 3313},
										name: "searchWord",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 125, col: 5, offset: 3376},
						run: (*parser).callonsearchValue32,
						expr: &seqExpr{
							pos: position{line: 125, col: 5, offset: 3376},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 125, col: 5, offset: 3376},
									expr: &seqExpr{
										pos: position{line: 125, col: 7, offset: 3378},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 125, col: 7, offset: 3378},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 125, col: 22, offset: 3393},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 125, col: 25, offset: 3396},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 125, col: 27, offset: 3398},
										name: "booleanLiteral",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 126, col: 5, offset: 3435},
						run: (*parser).callonsearchValue40,
						expr: &seqExpr{
							pos: position{line: 126, col: 5, offset: 3435},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 126, col: 5, offset: 3435},
									expr: &seqExpr{
										pos: position{line: 126, col: 7, offset: 3437},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 126, col: 7, offset: 3437},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 126, col: 22, offset: 3452},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 126, col: 25, offset: 3455},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 126, col: 27, offset: 3457},
										name: "nullLiteral",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 127, col: 5, offset: 3491},
						run: (*parser).callonsearchValue48,
						expr: &seqExpr{
							pos: position{line: 127, col: 5, offset: 3491},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 127, col: 5, offset: 3491},
									expr: &seqExpr{
										pos: position{line: 127, col: 7, offset: 3493},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 127, col: 8, offset: 3494},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 127, col: 24, offset: 3510},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 127, col: 27, offset: 3513},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 127, col: 29, offset: 3515},
										name: "searchWord",
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
			pos:  position{line: 135, col: 1, offset: 3714},
			expr: &choiceExpr{
				pos: position{line: 136, col: 5, offset: 3733},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 136, col: 5, offset: 3733},
						name: "andToken",
					},
					&ruleRefExpr{
						pos:  position{line: 137, col: 5, offset: 3746},
						name: "orToken",
					},
					&ruleRefExpr{
						pos:  position{line: 138, col: 5, offset: 3758},
						name: "inToken",
					},
				},
			},
		},
		{
			name: "booleanLiteral",
			pos:  position{line: 140, col: 1, offset: 3767},
			expr: &choiceExpr{
				pos: position{line: 141, col: 5, offset: 3786},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 141, col: 5, offset: 3786},
						run: (*parser).callonbooleanLiteral2,
						expr: &litMatcher{
							pos:        position{line: 141, col: 5, offset: 3786},
							val:        "true",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 142, col: 5, offset: 3851},
						run: (*parser).callonbooleanLiteral4,
						expr: &litMatcher{
							pos:        position{line: 142, col: 5, offset: 3851},
							val:        "false",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "nullLiteral",
			pos:  position{line: 144, col: 1, offset: 3914},
			expr: &actionExpr{
				pos: position{line: 145, col: 5, offset: 3930},
				run: (*parser).callonnullLiteral1,
				expr: &litMatcher{
					pos:        position{line: 145, col: 5, offset: 3930},
					val:        "null",
					ignoreCase: false,
				},
			},
		},
		{
			name: "procList",
			pos:  position{line: 147, col: 1, offset: 3988},
			expr: &actionExpr{
				pos: position{line: 148, col: 5, offset: 4001},
				run: (*parser).callonprocList1,
				expr: &seqExpr{
					pos: position{line: 148, col: 5, offset: 4001},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 148, col: 5, offset: 4001},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 148, col: 11, offset: 4007},
								name: "procChain",
							},
						},
						&labeledExpr{
							pos:   position{line: 148, col: 21, offset: 4017},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 148, col: 26, offset: 4022},
								expr: &ruleRefExpr{
									pos:  position{line: 148, col: 26, offset: 4022},
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
			pos:  position{line: 157, col: 1, offset: 4246},
			expr: &actionExpr{
				pos: position{line: 158, col: 5, offset: 4264},
				run: (*parser).callonparallelChain1,
				expr: &seqExpr{
					pos: position{line: 158, col: 5, offset: 4264},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 158, col: 5, offset: 4264},
							expr: &ruleRefExpr{
								pos:  position{line: 158, col: 5, offset: 4264},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 158, col: 8, offset: 4267},
							val:        ";",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 158, col: 12, offset: 4271},
							expr: &ruleRefExpr{
								pos:  position{line: 158, col: 12, offset: 4271},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 158, col: 15, offset: 4274},
							label: "ch",
							expr: &ruleRefExpr{
								pos:  position{line: 158, col: 18, offset: 4277},
								name: "procChain",
							},
						},
					},
				},
			},
		},
		{
			name: "proc",
			pos:  position{line: 160, col: 1, offset: 4327},
			expr: &choiceExpr{
				pos: position{line: 161, col: 5, offset: 4336},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 161, col: 5, offset: 4336},
						name: "simpleProc",
					},
					&ruleRefExpr{
						pos:  position{line: 162, col: 5, offset: 4351},
						name: "reducerProc",
					},
					&actionExpr{
						pos: position{line: 163, col: 5, offset: 4367},
						run: (*parser).callonproc4,
						expr: &seqExpr{
							pos: position{line: 163, col: 5, offset: 4367},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 163, col: 5, offset: 4367},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 163, col: 9, offset: 4371},
									expr: &ruleRefExpr{
										pos:  position{line: 163, col: 9, offset: 4371},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 163, col: 12, offset: 4374},
									label: "proc",
									expr: &ruleRefExpr{
										pos:  position{line: 163, col: 17, offset: 4379},
										name: "procList",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 163, col: 26, offset: 4388},
									expr: &ruleRefExpr{
										pos:  position{line: 163, col: 26, offset: 4388},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 163, col: 29, offset: 4391},
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
			pos:  position{line: 167, col: 1, offset: 4427},
			expr: &actionExpr{
				pos: position{line: 168, col: 5, offset: 4439},
				run: (*parser).callongroupBy1,
				expr: &seqExpr{
					pos: position{line: 168, col: 5, offset: 4439},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 168, col: 5, offset: 4439},
							val:        "by",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 168, col: 11, offset: 4445},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 168, col: 13, offset: 4447},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 168, col: 18, offset: 4452},
								name: "fieldExprList",
							},
						},
					},
				},
			},
		},
		{
			name: "everyDur",
			pos:  position{line: 170, col: 1, offset: 4488},
			expr: &actionExpr{
				pos: position{line: 171, col: 5, offset: 4501},
				run: (*parser).calloneveryDur1,
				expr: &seqExpr{
					pos: position{line: 171, col: 5, offset: 4501},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 171, col: 5, offset: 4501},
							val:        "every",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 171, col: 14, offset: 4510},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 171, col: 16, offset: 4512},
							label: "dur",
							expr: &ruleRefExpr{
								pos:  position{line: 171, col: 20, offset: 4516},
								name: "duration",
							},
						},
					},
				},
			},
		},
		{
			name: "equalityToken",
			pos:  position{line: 173, col: 1, offset: 4546},
			expr: &choiceExpr{
				pos: position{line: 174, col: 5, offset: 4564},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 174, col: 5, offset: 4564},
						run: (*parser).callonequalityToken2,
						expr: &litMatcher{
							pos:        position{line: 174, col: 5, offset: 4564},
							val:        "=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 175, col: 5, offset: 4594},
						run: (*parser).callonequalityToken4,
						expr: &litMatcher{
							pos:        position{line: 175, col: 5, offset: 4594},
							val:        "!=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 176, col: 5, offset: 4626},
						run: (*parser).callonequalityToken6,
						expr: &litMatcher{
							pos:        position{line: 176, col: 5, offset: 4626},
							val:        "<=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 177, col: 5, offset: 4657},
						run: (*parser).callonequalityToken8,
						expr: &litMatcher{
							pos:        position{line: 177, col: 5, offset: 4657},
							val:        ">=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 178, col: 5, offset: 4688},
						run: (*parser).callonequalityToken10,
						expr: &litMatcher{
							pos:        position{line: 178, col: 5, offset: 4688},
							val:        "<",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 179, col: 5, offset: 4717},
						run: (*parser).callonequalityToken12,
						expr: &litMatcher{
							pos:        position{line: 179, col: 5, offset: 4717},
							val:        ">",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "types",
			pos:  position{line: 181, col: 1, offset: 4743},
			expr: &choiceExpr{
				pos: position{line: 182, col: 5, offset: 4753},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 182, col: 5, offset: 4753},
						val:        "bool",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 183, col: 5, offset: 4764},
						val:        "int",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 184, col: 5, offset: 4774},
						val:        "count",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 185, col: 5, offset: 4786},
						val:        "double",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 186, col: 5, offset: 4799},
						val:        "string",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 187, col: 5, offset: 4812},
						val:        "addr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 188, col: 5, offset: 4823},
						val:        "subnet",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 189, col: 5, offset: 4836},
						val:        "port",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "dash",
			pos:  position{line: 191, col: 1, offset: 4844},
			expr: &choiceExpr{
				pos: position{line: 191, col: 8, offset: 4851},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 191, col: 8, offset: 4851},
						val:        "-",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 191, col: 14, offset: 4857},
						val:        "—",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 191, col: 25, offset: 4868},
						val:        "–",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 191, col: 36, offset: 4879},
						exprs: []interface{}{
							&litMatcher{
								pos:        position{line: 191, col: 36, offset: 4879},
								val:        "to",
								ignoreCase: false,
							},
							&andExpr{
								pos: position{line: 191, col: 40, offset: 4883},
								expr: &ruleRefExpr{
									pos:  position{line: 191, col: 42, offset: 4885},
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
			pos:  position{line: 193, col: 1, offset: 4889},
			expr: &litMatcher{
				pos:        position{line: 193, col: 12, offset: 4900},
				val:        "and",
				ignoreCase: false,
			},
		},
		{
			name: "orToken",
			pos:  position{line: 194, col: 1, offset: 4906},
			expr: &litMatcher{
				pos:        position{line: 194, col: 11, offset: 4916},
				val:        "or",
				ignoreCase: false,
			},
		},
		{
			name: "inToken",
			pos:  position{line: 195, col: 1, offset: 4921},
			expr: &litMatcher{
				pos:        position{line: 195, col: 11, offset: 4931},
				val:        "in",
				ignoreCase: false,
			},
		},
		{
			name: "notToken",
			pos:  position{line: 196, col: 1, offset: 4936},
			expr: &litMatcher{
				pos:        position{line: 196, col: 12, offset: 4947},
				val:        "not",
				ignoreCase: false,
			},
		},
		{
			name: "fieldName",
			pos:  position{line: 198, col: 1, offset: 4954},
			expr: &actionExpr{
				pos: position{line: 198, col: 13, offset: 4966},
				run: (*parser).callonfieldName1,
				expr: &seqExpr{
					pos: position{line: 198, col: 13, offset: 4966},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 198, col: 13, offset: 4966},
							name: "fieldNameStart",
						},
						&zeroOrMoreExpr{
							pos: position{line: 198, col: 28, offset: 4981},
							expr: &ruleRefExpr{
								pos:  position{line: 198, col: 28, offset: 4981},
								name: "fieldNameRest",
							},
						},
					},
				},
			},
		},
		{
			name: "fieldNameStart",
			pos:  position{line: 200, col: 1, offset: 5028},
			expr: &charClassMatcher{
				pos:        position{line: 200, col: 18, offset: 5045},
				val:        "[A-Za-z_$]",
				chars:      []rune{'_', '$'},
				ranges:     []rune{'A', 'Z', 'a', 'z'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "fieldNameRest",
			pos:  position{line: 201, col: 1, offset: 5056},
			expr: &choiceExpr{
				pos: position{line: 201, col: 17, offset: 5072},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 201, col: 17, offset: 5072},
						name: "fieldNameStart",
					},
					&charClassMatcher{
						pos:        position{line: 201, col: 34, offset: 5089},
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
			pos:  position{line: 203, col: 1, offset: 5096},
			expr: &actionExpr{
				pos: position{line: 204, col: 4, offset: 5114},
				run: (*parser).callonfieldReference1,
				expr: &seqExpr{
					pos: position{line: 204, col: 4, offset: 5114},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 204, col: 4, offset: 5114},
							label: "base",
							expr: &ruleRefExpr{
								pos:  position{line: 204, col: 9, offset: 5119},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 204, col: 19, offset: 5129},
							label: "derefs",
							expr: &zeroOrMoreExpr{
								pos: position{line: 204, col: 26, offset: 5136},
								expr: &choiceExpr{
									pos: position{line: 205, col: 8, offset: 5145},
									alternatives: []interface{}{
										&actionExpr{
											pos: position{line: 205, col: 8, offset: 5145},
											run: (*parser).callonfieldReference8,
											expr: &seqExpr{
												pos: position{line: 205, col: 8, offset: 5145},
												exprs: []interface{}{
													&litMatcher{
														pos:        position{line: 205, col: 8, offset: 5145},
														val:        ".",
														ignoreCase: false,
													},
													&labeledExpr{
														pos:   position{line: 205, col: 12, offset: 5149},
														label: "field",
														expr: &ruleRefExpr{
															pos:  position{line: 205, col: 18, offset: 5155},
															name: "fieldName",
														},
													},
												},
											},
										},
										&actionExpr{
											pos: position{line: 206, col: 8, offset: 5236},
											run: (*parser).callonfieldReference13,
											expr: &seqExpr{
												pos: position{line: 206, col: 8, offset: 5236},
												exprs: []interface{}{
													&litMatcher{
														pos:        position{line: 206, col: 8, offset: 5236},
														val:        "[",
														ignoreCase: false,
													},
													&labeledExpr{
														pos:   position{line: 206, col: 12, offset: 5240},
														label: "index",
														expr: &ruleRefExpr{
															pos:  position{line: 206, col: 18, offset: 5246},
															name: "sinteger",
														},
													},
													&litMatcher{
														pos:        position{line: 206, col: 27, offset: 5255},
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
			pos:  position{line: 211, col: 1, offset: 5371},
			expr: &choiceExpr{
				pos: position{line: 212, col: 5, offset: 5385},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 212, col: 5, offset: 5385},
						run: (*parser).callonfieldExpr2,
						expr: &seqExpr{
							pos: position{line: 212, col: 5, offset: 5385},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 212, col: 5, offset: 5385},
									label: "op",
									expr: &ruleRefExpr{
										pos:  position{line: 212, col: 8, offset: 5388},
										name: "fieldOp",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 212, col: 16, offset: 5396},
									expr: &ruleRefExpr{
										pos:  position{line: 212, col: 16, offset: 5396},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 212, col: 19, offset: 5399},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 212, col: 23, offset: 5403},
									expr: &ruleRefExpr{
										pos:  position{line: 212, col: 23, offset: 5403},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 212, col: 26, offset: 5406},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 212, col: 32, offset: 5412},
										name: "fieldReference",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 212, col: 47, offset: 5427},
									expr: &ruleRefExpr{
										pos:  position{line: 212, col: 47, offset: 5427},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 212, col: 50, offset: 5430},
									val:        ")",
									ignoreCase: false,
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 215, col: 5, offset: 5494},
						name: "fieldReference",
					},
				},
			},
		},
		{
			name: "fieldOp",
			pos:  position{line: 217, col: 1, offset: 5510},
			expr: &actionExpr{
				pos: position{line: 218, col: 5, offset: 5522},
				run: (*parser).callonfieldOp1,
				expr: &litMatcher{
					pos:        position{line: 218, col: 5, offset: 5522},
					val:        "len",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldExprList",
			pos:  position{line: 220, col: 1, offset: 5552},
			expr: &actionExpr{
				pos: position{line: 221, col: 5, offset: 5570},
				run: (*parser).callonfieldExprList1,
				expr: &seqExpr{
					pos: position{line: 221, col: 5, offset: 5570},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 221, col: 5, offset: 5570},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 221, col: 11, offset: 5576},
								name: "fieldExpr",
							},
						},
						&labeledExpr{
							pos:   position{line: 221, col: 21, offset: 5586},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 221, col: 26, offset: 5591},
								expr: &seqExpr{
									pos: position{line: 221, col: 27, offset: 5592},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 221, col: 27, offset: 5592},
											expr: &ruleRefExpr{
												pos:  position{line: 221, col: 27, offset: 5592},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 221, col: 30, offset: 5595},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 221, col: 34, offset: 5599},
											expr: &ruleRefExpr{
												pos:  position{line: 221, col: 34, offset: 5599},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 221, col: 37, offset: 5602},
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
			pos:  position{line: 231, col: 1, offset: 5797},
			expr: &actionExpr{
				pos: position{line: 232, col: 5, offset: 5817},
				run: (*parser).callonfieldRefDotOnly1,
				expr: &seqExpr{
					pos: position{line: 232, col: 5, offset: 5817},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 232, col: 5, offset: 5817},
							label: "base",
							expr: &ruleRefExpr{
								pos:  position{line: 232, col: 10, offset: 5822},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 232, col: 20, offset: 5832},
							label: "refs",
							expr: &zeroOrMoreExpr{
								pos: position{line: 232, col: 25, offset: 5837},
								expr: &actionExpr{
									pos: position{line: 232, col: 26, offset: 5838},
									run: (*parser).callonfieldRefDotOnly7,
									expr: &seqExpr{
										pos: position{line: 232, col: 26, offset: 5838},
										exprs: []interface{}{
											&litMatcher{
												pos:        position{line: 232, col: 26, offset: 5838},
												val:        ".",
												ignoreCase: false,
											},
											&labeledExpr{
												pos:   position{line: 232, col: 30, offset: 5842},
												label: "field",
												expr: &ruleRefExpr{
													pos:  position{line: 232, col: 36, offset: 5848},
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
			pos:  position{line: 236, col: 1, offset: 5973},
			expr: &actionExpr{
				pos: position{line: 237, col: 5, offset: 5997},
				run: (*parser).callonfieldRefDotOnlyList1,
				expr: &seqExpr{
					pos: position{line: 237, col: 5, offset: 5997},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 237, col: 5, offset: 5997},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 237, col: 11, offset: 6003},
								name: "fieldRefDotOnly",
							},
						},
						&labeledExpr{
							pos:   position{line: 237, col: 27, offset: 6019},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 237, col: 32, offset: 6024},
								expr: &actionExpr{
									pos: position{line: 237, col: 33, offset: 6025},
									run: (*parser).callonfieldRefDotOnlyList7,
									expr: &seqExpr{
										pos: position{line: 237, col: 33, offset: 6025},
										exprs: []interface{}{
											&zeroOrOneExpr{
												pos: position{line: 237, col: 33, offset: 6025},
												expr: &ruleRefExpr{
													pos:  position{line: 237, col: 33, offset: 6025},
													name: "_",
												},
											},
											&litMatcher{
												pos:        position{line: 237, col: 36, offset: 6028},
												val:        ",",
												ignoreCase: false,
											},
											&zeroOrOneExpr{
												pos: position{line: 237, col: 40, offset: 6032},
												expr: &ruleRefExpr{
													pos:  position{line: 237, col: 40, offset: 6032},
													name: "_",
												},
											},
											&labeledExpr{
												pos:   position{line: 237, col: 43, offset: 6035},
												label: "ref",
												expr: &ruleRefExpr{
													pos:  position{line: 237, col: 47, offset: 6039},
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
			pos:  position{line: 245, col: 1, offset: 6219},
			expr: &actionExpr{
				pos: position{line: 246, col: 5, offset: 6237},
				run: (*parser).callonfieldNameList1,
				expr: &seqExpr{
					pos: position{line: 246, col: 5, offset: 6237},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 246, col: 5, offset: 6237},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 246, col: 11, offset: 6243},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 246, col: 21, offset: 6253},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 246, col: 26, offset: 6258},
								expr: &seqExpr{
									pos: position{line: 246, col: 27, offset: 6259},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 246, col: 27, offset: 6259},
											expr: &ruleRefExpr{
												pos:  position{line: 246, col: 27, offset: 6259},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 246, col: 30, offset: 6262},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 246, col: 34, offset: 6266},
											expr: &ruleRefExpr{
												pos:  position{line: 246, col: 34, offset: 6266},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 246, col: 37, offset: 6269},
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
			pos:  position{line: 254, col: 1, offset: 6462},
			expr: &actionExpr{
				pos: position{line: 255, col: 5, offset: 6474},
				run: (*parser).calloncountOp1,
				expr: &litMatcher{
					pos:        position{line: 255, col: 5, offset: 6474},
					val:        "count",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldReducerOp",
			pos:  position{line: 257, col: 1, offset: 6508},
			expr: &choiceExpr{
				pos: position{line: 258, col: 5, offset: 6527},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 258, col: 5, offset: 6527},
						run: (*parser).callonfieldReducerOp2,
						expr: &litMatcher{
							pos:        position{line: 258, col: 5, offset: 6527},
							val:        "sum",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 259, col: 5, offset: 6561},
						run: (*parser).callonfieldReducerOp4,
						expr: &litMatcher{
							pos:        position{line: 259, col: 5, offset: 6561},
							val:        "avg",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 260, col: 5, offset: 6595},
						run: (*parser).callonfieldReducerOp6,
						expr: &litMatcher{
							pos:        position{line: 260, col: 5, offset: 6595},
							val:        "stdev",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 261, col: 5, offset: 6632},
						run: (*parser).callonfieldReducerOp8,
						expr: &litMatcher{
							pos:        position{line: 261, col: 5, offset: 6632},
							val:        "sd",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 262, col: 5, offset: 6668},
						run: (*parser).callonfieldReducerOp10,
						expr: &litMatcher{
							pos:        position{line: 262, col: 5, offset: 6668},
							val:        "var",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 263, col: 5, offset: 6702},
						run: (*parser).callonfieldReducerOp12,
						expr: &litMatcher{
							pos:        position{line: 263, col: 5, offset: 6702},
							val:        "entropy",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 264, col: 5, offset: 6743},
						run: (*parser).callonfieldReducerOp14,
						expr: &litMatcher{
							pos:        position{line: 264, col: 5, offset: 6743},
							val:        "min",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 265, col: 5, offset: 6777},
						run: (*parser).callonfieldReducerOp16,
						expr: &litMatcher{
							pos:        position{line: 265, col: 5, offset: 6777},
							val:        "max",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 266, col: 5, offset: 6811},
						run: (*parser).callonfieldReducerOp18,
						expr: &litMatcher{
							pos:        position{line: 266, col: 5, offset: 6811},
							val:        "first",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 267, col: 5, offset: 6849},
						run: (*parser).callonfieldReducerOp20,
						expr: &litMatcher{
							pos:        position{line: 267, col: 5, offset: 6849},
							val:        "last",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 268, col: 5, offset: 6885},
						run: (*parser).callonfieldReducerOp22,
						expr: &litMatcher{
							pos:        position{line: 268, col: 5, offset: 6885},
							val:        "countdistinct",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "paddedFieldName",
			pos:  position{line: 270, col: 1, offset: 6935},
			expr: &actionExpr{
				pos: position{line: 270, col: 19, offset: 6953},
				run: (*parser).callonpaddedFieldName1,
				expr: &seqExpr{
					pos: position{line: 270, col: 19, offset: 6953},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 270, col: 19, offset: 6953},
							expr: &ruleRefExpr{
								pos:  position{line: 270, col: 19, offset: 6953},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 270, col: 22, offset: 6956},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 270, col: 28, offset: 6962},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 270, col: 38, offset: 6972},
							expr: &ruleRefExpr{
								pos:  position{line: 270, col: 38, offset: 6972},
								name: "_",
							},
						},
					},
				},
			},
		},
		{
			name: "countReducer",
			pos:  position{line: 272, col: 1, offset: 6998},
			expr: &actionExpr{
				pos: position{line: 273, col: 5, offset: 7015},
				run: (*parser).calloncountReducer1,
				expr: &seqExpr{
					pos: position{line: 273, col: 5, offset: 7015},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 273, col: 5, offset: 7015},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 273, col: 8, offset: 7018},
								name: "countOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 273, col: 16, offset: 7026},
							expr: &ruleRefExpr{
								pos:  position{line: 273, col: 16, offset: 7026},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 273, col: 19, offset: 7029},
							val:        "(",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 273, col: 23, offset: 7033},
							label: "field",
							expr: &zeroOrOneExpr{
								pos: position{line: 273, col: 29, offset: 7039},
								expr: &ruleRefExpr{
									pos:  position{line: 273, col: 29, offset: 7039},
									name: "paddedFieldName",
								},
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 273, col: 47, offset: 7057},
							expr: &ruleRefExpr{
								pos:  position{line: 273, col: 47, offset: 7057},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 273, col: 50, offset: 7060},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "fieldReducer",
			pos:  position{line: 277, col: 1, offset: 7119},
			expr: &actionExpr{
				pos: position{line: 278, col: 5, offset: 7136},
				run: (*parser).callonfieldReducer1,
				expr: &seqExpr{
					pos: position{line: 278, col: 5, offset: 7136},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 278, col: 5, offset: 7136},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 278, col: 8, offset: 7139},
								name: "fieldReducerOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 278, col: 23, offset: 7154},
							expr: &ruleRefExpr{
								pos:  position{line: 278, col: 23, offset: 7154},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 278, col: 26, offset: 7157},
							val:        "(",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 278, col: 30, offset: 7161},
							expr: &ruleRefExpr{
								pos:  position{line: 278, col: 30, offset: 7161},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 278, col: 33, offset: 7164},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 278, col: 39, offset: 7170},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 278, col: 50, offset: 7181},
							expr: &ruleRefExpr{
								pos:  position{line: 278, col: 50, offset: 7181},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 278, col: 53, offset: 7184},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reducerProc",
			pos:  position{line: 282, col: 1, offset: 7251},
			expr: &actionExpr{
				pos: position{line: 283, col: 5, offset: 7267},
				run: (*parser).callonreducerProc1,
				expr: &seqExpr{
					pos: position{line: 283, col: 5, offset: 7267},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 283, col: 5, offset: 7267},
							label: "every",
							expr: &zeroOrOneExpr{
								pos: position{line: 283, col: 11, offset: 7273},
								expr: &seqExpr{
									pos: position{line: 283, col: 12, offset: 7274},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 283, col: 12, offset: 7274},
											name: "everyDur",
										},
										&ruleRefExpr{
											pos:  position{line: 283, col: 21, offset: 7283},
											name: "_",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 283, col: 25, offset: 7287},
							label: "reducers",
							expr: &ruleRefExpr{
								pos:  position{line: 283, col: 34, offset: 7296},
								name: "reducerList",
							},
						},
						&labeledExpr{
							pos:   position{line: 283, col: 46, offset: 7308},
							label: "keys",
							expr: &zeroOrOneExpr{
								pos: position{line: 283, col: 51, offset: 7313},
								expr: &seqExpr{
									pos: position{line: 283, col: 52, offset: 7314},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 283, col: 52, offset: 7314},
											name: "_",
										},
										&ruleRefExpr{
											pos:  position{line: 283, col: 54, offset: 7316},
											name: "groupBy",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 283, col: 64, offset: 7326},
							label: "limit",
							expr: &zeroOrOneExpr{
								pos: position{line: 283, col: 70, offset: 7332},
								expr: &ruleRefExpr{
									pos:  position{line: 283, col: 70, offset: 7332},
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
			pos:  position{line: 301, col: 1, offset: 7689},
			expr: &actionExpr{
				pos: position{line: 302, col: 5, offset: 7702},
				run: (*parser).callonasClause1,
				expr: &seqExpr{
					pos: position{line: 302, col: 5, offset: 7702},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 302, col: 5, offset: 7702},
							val:        "as",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 302, col: 11, offset: 7708},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 302, col: 13, offset: 7710},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 302, col: 15, offset: 7712},
								name: "fieldName",
							},
						},
					},
				},
			},
		},
		{
			name: "reducerExpr",
			pos:  position{line: 304, col: 1, offset: 7741},
			expr: &choiceExpr{
				pos: position{line: 305, col: 5, offset: 7757},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 305, col: 5, offset: 7757},
						run: (*parser).callonreducerExpr2,
						expr: &seqExpr{
							pos: position{line: 305, col: 5, offset: 7757},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 305, col: 5, offset: 7757},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 305, col: 11, offset: 7763},
										name: "fieldName",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 305, col: 21, offset: 7773},
									expr: &ruleRefExpr{
										pos:  position{line: 305, col: 21, offset: 7773},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 305, col: 24, offset: 7776},
									val:        "=",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 305, col: 28, offset: 7780},
									expr: &ruleRefExpr{
										pos:  position{line: 305, col: 28, offset: 7780},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 305, col: 31, offset: 7783},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 305, col: 33, offset: 7785},
										name: "reducer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 308, col: 5, offset: 7848},
						run: (*parser).callonreducerExpr13,
						expr: &seqExpr{
							pos: position{line: 308, col: 5, offset: 7848},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 308, col: 5, offset: 7848},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 308, col: 7, offset: 7850},
										name: "reducer",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 308, col: 15, offset: 7858},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 308, col: 17, offset: 7860},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 308, col: 23, offset: 7866},
										name: "asClause",
									},
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 311, col: 5, offset: 7930},
						name: "reducer",
					},
				},
			},
		},
		{
			name: "reducer",
			pos:  position{line: 313, col: 1, offset: 7939},
			expr: &choiceExpr{
				pos: position{line: 314, col: 5, offset: 7951},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 314, col: 5, offset: 7951},
						name: "countReducer",
					},
					&ruleRefExpr{
						pos:  position{line: 315, col: 5, offset: 7968},
						name: "fieldReducer",
					},
				},
			},
		},
		{
			name: "reducerList",
			pos:  position{line: 317, col: 1, offset: 7982},
			expr: &actionExpr{
				pos: position{line: 318, col: 5, offset: 7998},
				run: (*parser).callonreducerList1,
				expr: &seqExpr{
					pos: position{line: 318, col: 5, offset: 7998},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 318, col: 5, offset: 7998},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 318, col: 11, offset: 8004},
								name: "reducerExpr",
							},
						},
						&labeledExpr{
							pos:   position{line: 318, col: 23, offset: 8016},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 318, col: 28, offset: 8021},
								expr: &seqExpr{
									pos: position{line: 318, col: 29, offset: 8022},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 318, col: 29, offset: 8022},
											expr: &ruleRefExpr{
												pos:  position{line: 318, col: 29, offset: 8022},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 318, col: 32, offset: 8025},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 318, col: 36, offset: 8029},
											expr: &ruleRefExpr{
												pos:  position{line: 318, col: 36, offset: 8029},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 318, col: 39, offset: 8032},
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
			pos:  position{line: 326, col: 1, offset: 8229},
			expr: &choiceExpr{
				pos: position{line: 327, col: 5, offset: 8244},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 327, col: 5, offset: 8244},
						name: "sort",
					},
					&ruleRefExpr{
						pos:  position{line: 328, col: 5, offset: 8253},
						name: "top",
					},
					&ruleRefExpr{
						pos:  position{line: 329, col: 5, offset: 8261},
						name: "cut",
					},
					&ruleRefExpr{
						pos:  position{line: 330, col: 5, offset: 8269},
						name: "head",
					},
					&ruleRefExpr{
						pos:  position{line: 331, col: 5, offset: 8278},
						name: "tail",
					},
					&ruleRefExpr{
						pos:  position{line: 332, col: 5, offset: 8287},
						name: "filter",
					},
					&ruleRefExpr{
						pos:  position{line: 333, col: 5, offset: 8298},
						name: "uniq",
					},
				},
			},
		},
		{
			name: "sort",
			pos:  position{line: 335, col: 1, offset: 8304},
			expr: &actionExpr{
				pos: position{line: 336, col: 5, offset: 8313},
				run: (*parser).callonsort1,
				expr: &seqExpr{
					pos: position{line: 336, col: 5, offset: 8313},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 336, col: 5, offset: 8313},
							val:        "sort",
							ignoreCase: true,
						},
						&labeledExpr{
							pos:   position{line: 336, col: 13, offset: 8321},
							label: "args",
							expr: &ruleRefExpr{
								pos:  position{line: 336, col: 18, offset: 8326},
								name: "sortArgs",
							},
						},
						&labeledExpr{
							pos:   position{line: 336, col: 27, offset: 8335},
							label: "list",
							expr: &zeroOrOneExpr{
								pos: position{line: 336, col: 32, offset: 8340},
								expr: &actionExpr{
									pos: position{line: 336, col: 33, offset: 8341},
									run: (*parser).callonsort8,
									expr: &seqExpr{
										pos: position{line: 336, col: 33, offset: 8341},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 336, col: 33, offset: 8341},
												name: "_",
											},
											&labeledExpr{
												pos:   position{line: 336, col: 35, offset: 8343},
												label: "l",
												expr: &ruleRefExpr{
													pos:  position{line: 336, col: 37, offset: 8345},
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
			pos:  position{line: 340, col: 1, offset: 8422},
			expr: &zeroOrMoreExpr{
				pos: position{line: 340, col: 12, offset: 8433},
				expr: &actionExpr{
					pos: position{line: 340, col: 13, offset: 8434},
					run: (*parser).callonsortArgs2,
					expr: &seqExpr{
						pos: position{line: 340, col: 13, offset: 8434},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 340, col: 13, offset: 8434},
								name: "_",
							},
							&labeledExpr{
								pos:   position{line: 340, col: 15, offset: 8436},
								label: "a",
								expr: &ruleRefExpr{
									pos:  position{line: 340, col: 17, offset: 8438},
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
			pos:  position{line: 342, col: 1, offset: 8467},
			expr: &choiceExpr{
				pos: position{line: 343, col: 5, offset: 8479},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 343, col: 5, offset: 8479},
						run: (*parser).callonsortArg2,
						expr: &seqExpr{
							pos: position{line: 343, col: 5, offset: 8479},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 343, col: 5, offset: 8479},
									val:        "-limit",
									ignoreCase: false,
								},
								&ruleRefExpr{
									pos:  position{line: 343, col: 14, offset: 8488},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 343, col: 16, offset: 8490},
									label: "limit",
									expr: &ruleRefExpr{
										pos:  position{line: 343, col: 22, offset: 8496},
										name: "sinteger",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 344, col: 5, offset: 8549},
						run: (*parser).callonsortArg8,
						expr: &litMatcher{
							pos:        position{line: 344, col: 5, offset: 8549},
							val:        "-r",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 345, col: 5, offset: 8592},
						run: (*parser).callonsortArg10,
						expr: &seqExpr{
							pos: position{line: 345, col: 5, offset: 8592},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 345, col: 5, offset: 8592},
									val:        "-nulls",
									ignoreCase: false,
								},
								&ruleRefExpr{
									pos:  position{line: 345, col: 14, offset: 8601},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 345, col: 16, offset: 8603},
									label: "where",
									expr: &actionExpr{
										pos: position{line: 345, col: 23, offset: 8610},
										run: (*parser).callonsortArg15,
										expr: &choiceExpr{
											pos: position{line: 345, col: 24, offset: 8611},
											alternatives: []interface{}{
												&litMatcher{
													pos:        position{line: 345, col: 24, offset: 8611},
													val:        "first",
													ignoreCase: false,
												},
												&litMatcher{
													pos:        position{line: 345, col: 34, offset: 8621},
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
			pos:  position{line: 347, col: 1, offset: 8703},
			expr: &actionExpr{
				pos: position{line: 348, col: 5, offset: 8711},
				run: (*parser).callontop1,
				expr: &seqExpr{
					pos: position{line: 348, col: 5, offset: 8711},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 348, col: 5, offset: 8711},
							val:        "top",
							ignoreCase: true,
						},
						&labeledExpr{
							pos:   position{line: 348, col: 12, offset: 8718},
							label: "limit",
							expr: &zeroOrOneExpr{
								pos: position{line: 348, col: 18, offset: 8724},
								expr: &actionExpr{
									pos: position{line: 348, col: 19, offset: 8725},
									run: (*parser).callontop6,
									expr: &seqExpr{
										pos: position{line: 348, col: 19, offset: 8725},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 348, col: 19, offset: 8725},
												name: "_",
											},
											&labeledExpr{
												pos:   position{line: 348, col: 21, offset: 8727},
												label: "n",
												expr: &ruleRefExpr{
													pos:  position{line: 348, col: 23, offset: 8729},
													name: "integer",
												},
											},
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 348, col: 50, offset: 8756},
							label: "flush",
							expr: &zeroOrOneExpr{
								pos: position{line: 348, col: 56, offset: 8762},
								expr: &seqExpr{
									pos: position{line: 348, col: 57, offset: 8763},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 348, col: 57, offset: 8763},
											name: "_",
										},
										&litMatcher{
											pos:        position{line: 348, col: 59, offset: 8765},
											val:        "-flush",
											ignoreCase: false,
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 348, col: 70, offset: 8776},
							label: "list",
							expr: &zeroOrOneExpr{
								pos: position{line: 348, col: 75, offset: 8781},
								expr: &actionExpr{
									pos: position{line: 348, col: 76, offset: 8782},
									run: (*parser).callontop18,
									expr: &seqExpr{
										pos: position{line: 348, col: 76, offset: 8782},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 348, col: 76, offset: 8782},
												name: "_",
											},
											&labeledExpr{
												pos:   position{line: 348, col: 78, offset: 8784},
												label: "f",
												expr: &ruleRefExpr{
													pos:  position{line: 348, col: 80, offset: 8786},
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
			pos:  position{line: 352, col: 1, offset: 8875},
			expr: &actionExpr{
				pos: position{line: 353, col: 5, offset: 8892},
				run: (*parser).callonprocLimitArg1,
				expr: &seqExpr{
					pos: position{line: 353, col: 5, offset: 8892},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 353, col: 5, offset: 8892},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 353, col: 7, offset: 8894},
							val:        "-limit",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 353, col: 16, offset: 8903},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 353, col: 18, offset: 8905},
							label: "limit",
							expr: &ruleRefExpr{
								pos:  position{line: 353, col: 24, offset: 8911},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "cut",
			pos:  position{line: 355, col: 1, offset: 8942},
			expr: &actionExpr{
				pos: position{line: 356, col: 5, offset: 8950},
				run: (*parser).calloncut1,
				expr: &seqExpr{
					pos: position{line: 356, col: 5, offset: 8950},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 356, col: 5, offset: 8950},
							val:        "cut",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 356, col: 12, offset: 8957},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 356, col: 14, offset: 8959},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 356, col: 19, offset: 8964},
								name: "fieldRefDotOnlyList",
							},
						},
					},
				},
			},
		},
		{
			name: "head",
			pos:  position{line: 357, col: 1, offset: 9018},
			expr: &choiceExpr{
				pos: position{line: 358, col: 5, offset: 9027},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 358, col: 5, offset: 9027},
						run: (*parser).callonhead2,
						expr: &seqExpr{
							pos: position{line: 358, col: 5, offset: 9027},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 358, col: 5, offset: 9027},
									val:        "head",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 358, col: 13, offset: 9035},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 358, col: 15, offset: 9037},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 358, col: 21, offset: 9043},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 359, col: 5, offset: 9091},
						run: (*parser).callonhead8,
						expr: &litMatcher{
							pos:        position{line: 359, col: 5, offset: 9091},
							val:        "head",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "tail",
			pos:  position{line: 360, col: 1, offset: 9131},
			expr: &choiceExpr{
				pos: position{line: 361, col: 5, offset: 9140},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 361, col: 5, offset: 9140},
						run: (*parser).callontail2,
						expr: &seqExpr{
							pos: position{line: 361, col: 5, offset: 9140},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 361, col: 5, offset: 9140},
									val:        "tail",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 361, col: 13, offset: 9148},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 361, col: 15, offset: 9150},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 361, col: 21, offset: 9156},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 362, col: 5, offset: 9204},
						run: (*parser).callontail8,
						expr: &litMatcher{
							pos:        position{line: 362, col: 5, offset: 9204},
							val:        "tail",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "filter",
			pos:  position{line: 364, col: 1, offset: 9245},
			expr: &actionExpr{
				pos: position{line: 365, col: 5, offset: 9256},
				run: (*parser).callonfilter1,
				expr: &seqExpr{
					pos: position{line: 365, col: 5, offset: 9256},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 365, col: 5, offset: 9256},
							val:        "filter",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 365, col: 15, offset: 9266},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 365, col: 17, offset: 9268},
							label: "expr",
							expr: &ruleRefExpr{
								pos:  position{line: 365, col: 22, offset: 9273},
								name: "searchExpr",
							},
						},
					},
				},
			},
		},
		{
			name: "uniq",
			pos:  position{line: 368, col: 1, offset: 9331},
			expr: &choiceExpr{
				pos: position{line: 369, col: 5, offset: 9340},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 369, col: 5, offset: 9340},
						run: (*parser).callonuniq2,
						expr: &seqExpr{
							pos: position{line: 369, col: 5, offset: 9340},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 369, col: 5, offset: 9340},
									val:        "uniq",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 369, col: 13, offset: 9348},
									name: "_",
								},
								&litMatcher{
									pos:        position{line: 369, col: 15, offset: 9350},
									val:        "-c",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 372, col: 5, offset: 9404},
						run: (*parser).callonuniq7,
						expr: &litMatcher{
							pos:        position{line: 372, col: 5, offset: 9404},
							val:        "uniq",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "duration",
			pos:  position{line: 376, col: 1, offset: 9459},
			expr: &choiceExpr{
				pos: position{line: 377, col: 5, offset: 9472},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 377, col: 5, offset: 9472},
						name: "seconds",
					},
					&ruleRefExpr{
						pos:  position{line: 378, col: 5, offset: 9484},
						name: "minutes",
					},
					&ruleRefExpr{
						pos:  position{line: 379, col: 5, offset: 9496},
						name: "hours",
					},
					&seqExpr{
						pos: position{line: 380, col: 5, offset: 9506},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 380, col: 5, offset: 9506},
								name: "hours",
							},
							&ruleRefExpr{
								pos:  position{line: 380, col: 11, offset: 9512},
								name: "_",
							},
							&litMatcher{
								pos:        position{line: 380, col: 13, offset: 9514},
								val:        "and",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 380, col: 19, offset: 9520},
								name: "_",
							},
							&ruleRefExpr{
								pos:  position{line: 380, col: 21, offset: 9522},
								name: "minutes",
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 381, col: 5, offset: 9534},
						name: "days",
					},
					&ruleRefExpr{
						pos:  position{line: 382, col: 5, offset: 9543},
						name: "weeks",
					},
				},
			},
		},
		{
			name: "sec_abbrev",
			pos:  position{line: 384, col: 1, offset: 9550},
			expr: &choiceExpr{
				pos: position{line: 385, col: 5, offset: 9565},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 385, col: 5, offset: 9565},
						val:        "seconds",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 386, col: 5, offset: 9579},
						val:        "second",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 387, col: 5, offset: 9592},
						val:        "secs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 388, col: 5, offset: 9603},
						val:        "sec",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 389, col: 5, offset: 9613},
						val:        "s",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "min_abbrev",
			pos:  position{line: 391, col: 1, offset: 9618},
			expr: &choiceExpr{
				pos: position{line: 392, col: 5, offset: 9633},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 392, col: 5, offset: 9633},
						val:        "minutes",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 393, col: 5, offset: 9647},
						val:        "minute",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 394, col: 5, offset: 9660},
						val:        "mins",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 395, col: 5, offset: 9671},
						val:        "min",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 396, col: 5, offset: 9681},
						val:        "m",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "hour_abbrev",
			pos:  position{line: 398, col: 1, offset: 9686},
			expr: &choiceExpr{
				pos: position{line: 399, col: 5, offset: 9702},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 399, col: 5, offset: 9702},
						val:        "hours",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 400, col: 5, offset: 9714},
						val:        "hrs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 401, col: 5, offset: 9724},
						val:        "hr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 402, col: 5, offset: 9733},
						val:        "h",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 403, col: 5, offset: 9741},
						val:        "hour",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "day_abbrev",
			pos:  position{line: 405, col: 1, offset: 9749},
			expr: &choiceExpr{
				pos: position{line: 405, col: 14, offset: 9762},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 405, col: 14, offset: 9762},
						val:        "days",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 405, col: 21, offset: 9769},
						val:        "day",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 405, col: 27, offset: 9775},
						val:        "d",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "week_abbrev",
			pos:  position{line: 406, col: 1, offset: 9779},
			expr: &choiceExpr{
				pos: position{line: 406, col: 15, offset: 9793},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 406, col: 15, offset: 9793},
						val:        "weeks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 406, col: 23, offset: 9801},
						val:        "week",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 406, col: 30, offset: 9808},
						val:        "wks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 406, col: 36, offset: 9814},
						val:        "wk",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 406, col: 41, offset: 9819},
						val:        "w",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "seconds",
			pos:  position{line: 408, col: 1, offset: 9824},
			expr: &choiceExpr{
				pos: position{line: 409, col: 5, offset: 9836},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 409, col: 5, offset: 9836},
						run: (*parser).callonseconds2,
						expr: &litMatcher{
							pos:        position{line: 409, col: 5, offset: 9836},
							val:        "second",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 410, col: 5, offset: 9881},
						run: (*parser).callonseconds4,
						expr: &seqExpr{
							pos: position{line: 410, col: 5, offset: 9881},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 410, col: 5, offset: 9881},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 410, col: 9, offset: 9885},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 410, col: 16, offset: 9892},
									expr: &ruleRefExpr{
										pos:  position{line: 410, col: 16, offset: 9892},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 410, col: 19, offset: 9895},
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
			pos:  position{line: 412, col: 1, offset: 9941},
			expr: &choiceExpr{
				pos: position{line: 413, col: 5, offset: 9953},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 413, col: 5, offset: 9953},
						run: (*parser).callonminutes2,
						expr: &litMatcher{
							pos:        position{line: 413, col: 5, offset: 9953},
							val:        "minute",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 414, col: 5, offset: 9999},
						run: (*parser).callonminutes4,
						expr: &seqExpr{
							pos: position{line: 414, col: 5, offset: 9999},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 414, col: 5, offset: 9999},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 414, col: 9, offset: 10003},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 414, col: 16, offset: 10010},
									expr: &ruleRefExpr{
										pos:  position{line: 414, col: 16, offset: 10010},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 414, col: 19, offset: 10013},
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
			pos:  position{line: 416, col: 1, offset: 10068},
			expr: &choiceExpr{
				pos: position{line: 417, col: 5, offset: 10078},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 417, col: 5, offset: 10078},
						run: (*parser).callonhours2,
						expr: &litMatcher{
							pos:        position{line: 417, col: 5, offset: 10078},
							val:        "hour",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 418, col: 5, offset: 10124},
						run: (*parser).callonhours4,
						expr: &seqExpr{
							pos: position{line: 418, col: 5, offset: 10124},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 418, col: 5, offset: 10124},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 418, col: 9, offset: 10128},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 418, col: 16, offset: 10135},
									expr: &ruleRefExpr{
										pos:  position{line: 418, col: 16, offset: 10135},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 418, col: 19, offset: 10138},
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
			pos:  position{line: 420, col: 1, offset: 10196},
			expr: &choiceExpr{
				pos: position{line: 421, col: 5, offset: 10205},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 421, col: 5, offset: 10205},
						run: (*parser).callondays2,
						expr: &litMatcher{
							pos:        position{line: 421, col: 5, offset: 10205},
							val:        "day",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 422, col: 5, offset: 10253},
						run: (*parser).callondays4,
						expr: &seqExpr{
							pos: position{line: 422, col: 5, offset: 10253},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 422, col: 5, offset: 10253},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 422, col: 9, offset: 10257},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 422, col: 16, offset: 10264},
									expr: &ruleRefExpr{
										pos:  position{line: 422, col: 16, offset: 10264},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 422, col: 19, offset: 10267},
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
			pos:  position{line: 424, col: 1, offset: 10327},
			expr: &actionExpr{
				pos: position{line: 425, col: 5, offset: 10337},
				run: (*parser).callonweeks1,
				expr: &seqExpr{
					pos: position{line: 425, col: 5, offset: 10337},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 425, col: 5, offset: 10337},
							label: "num",
							expr: &ruleRefExpr{
								pos:  position{line: 425, col: 9, offset: 10341},
								name: "number",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 425, col: 16, offset: 10348},
							expr: &ruleRefExpr{
								pos:  position{line: 425, col: 16, offset: 10348},
								name: "_",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 425, col: 19, offset: 10351},
							name: "week_abbrev",
						},
					},
				},
			},
		},
		{
			name: "number",
			pos:  position{line: 427, col: 1, offset: 10414},
			expr: &ruleRefExpr{
				pos:  position{line: 427, col: 10, offset: 10423},
				name: "integer",
			},
		},
		{
			name: "addr",
			pos:  position{line: 431, col: 1, offset: 10461},
			expr: &actionExpr{
				pos: position{line: 432, col: 5, offset: 10470},
				run: (*parser).callonaddr1,
				expr: &labeledExpr{
					pos:   position{line: 432, col: 5, offset: 10470},
					label: "a",
					expr: &seqExpr{
						pos: position{line: 432, col: 8, offset: 10473},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 432, col: 8, offset: 10473},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 432, col: 16, offset: 10481},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 432, col: 20, offset: 10485},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 432, col: 28, offset: 10493},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 432, col: 32, offset: 10497},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 432, col: 40, offset: 10505},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 432, col: 44, offset: 10509},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "port",
			pos:  position{line: 434, col: 1, offset: 10550},
			expr: &actionExpr{
				pos: position{line: 435, col: 5, offset: 10559},
				run: (*parser).callonport1,
				expr: &seqExpr{
					pos: position{line: 435, col: 5, offset: 10559},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 435, col: 5, offset: 10559},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 435, col: 9, offset: 10563},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 435, col: 11, offset: 10565},
								name: "sinteger",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6addr",
			pos:  position{line: 439, col: 1, offset: 10724},
			expr: &choiceExpr{
				pos: position{line: 440, col: 5, offset: 10736},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 440, col: 5, offset: 10736},
						run: (*parser).callonip6addr2,
						expr: &seqExpr{
							pos: position{line: 440, col: 5, offset: 10736},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 440, col: 5, offset: 10736},
									label: "a",
									expr: &oneOrMoreExpr{
										pos: position{line: 440, col: 7, offset: 10738},
										expr: &ruleRefExpr{
											pos:  position{line: 440, col: 8, offset: 10739},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 440, col: 20, offset: 10751},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 440, col: 22, offset: 10753},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 443, col: 5, offset: 10817},
						run: (*parser).callonip6addr9,
						expr: &seqExpr{
							pos: position{line: 443, col: 5, offset: 10817},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 443, col: 5, offset: 10817},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 443, col: 7, offset: 10819},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 443, col: 11, offset: 10823},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 443, col: 13, offset: 10825},
										expr: &ruleRefExpr{
											pos:  position{line: 443, col: 14, offset: 10826},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 443, col: 25, offset: 10837},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 443, col: 30, offset: 10842},
									label: "d",
									expr: &zeroOrMoreExpr{
										pos: position{line: 443, col: 32, offset: 10844},
										expr: &ruleRefExpr{
											pos:  position{line: 443, col: 33, offset: 10845},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 443, col: 45, offset: 10857},
									label: "e",
									expr: &ruleRefExpr{
										pos:  position{line: 443, col: 47, offset: 10859},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 446, col: 5, offset: 10958},
						run: (*parser).callonip6addr22,
						expr: &seqExpr{
							pos: position{line: 446, col: 5, offset: 10958},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 446, col: 5, offset: 10958},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 446, col: 10, offset: 10963},
									label: "a",
									expr: &zeroOrMoreExpr{
										pos: position{line: 446, col: 12, offset: 10965},
										expr: &ruleRefExpr{
											pos:  position{line: 446, col: 13, offset: 10966},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 446, col: 25, offset: 10978},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 446, col: 27, offset: 10980},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 449, col: 5, offset: 11051},
						run: (*parser).callonip6addr30,
						expr: &seqExpr{
							pos: position{line: 449, col: 5, offset: 11051},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 449, col: 5, offset: 11051},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 449, col: 7, offset: 11053},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 449, col: 11, offset: 11057},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 449, col: 13, offset: 11059},
										expr: &ruleRefExpr{
											pos:  position{line: 449, col: 14, offset: 11060},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 449, col: 25, offset: 11071},
									val:        "::",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 452, col: 5, offset: 11139},
						run: (*parser).callonip6addr38,
						expr: &litMatcher{
							pos:        position{line: 452, col: 5, offset: 11139},
							val:        "::",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "ip6tail",
			pos:  position{line: 456, col: 1, offset: 11176},
			expr: &choiceExpr{
				pos: position{line: 457, col: 5, offset: 11188},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 457, col: 5, offset: 11188},
						name: "addr",
					},
					&ruleRefExpr{
						pos:  position{line: 458, col: 5, offset: 11197},
						name: "h16",
					},
				},
			},
		},
		{
			name: "h_append",
			pos:  position{line: 460, col: 1, offset: 11202},
			expr: &actionExpr{
				pos: position{line: 460, col: 12, offset: 11213},
				run: (*parser).callonh_append1,
				expr: &seqExpr{
					pos: position{line: 460, col: 12, offset: 11213},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 460, col: 12, offset: 11213},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 460, col: 16, offset: 11217},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 460, col: 18, offset: 11219},
								name: "h16",
							},
						},
					},
				},
			},
		},
		{
			name: "h_prepend",
			pos:  position{line: 461, col: 1, offset: 11256},
			expr: &actionExpr{
				pos: position{line: 461, col: 13, offset: 11268},
				run: (*parser).callonh_prepend1,
				expr: &seqExpr{
					pos: position{line: 461, col: 13, offset: 11268},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 461, col: 13, offset: 11268},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 461, col: 15, offset: 11270},
								name: "h16",
							},
						},
						&litMatcher{
							pos:        position{line: 461, col: 19, offset: 11274},
							val:        ":",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "sub_addr",
			pos:  position{line: 463, col: 1, offset: 11312},
			expr: &choiceExpr{
				pos: position{line: 464, col: 5, offset: 11325},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 464, col: 5, offset: 11325},
						name: "addr",
					},
					&actionExpr{
						pos: position{line: 465, col: 5, offset: 11334},
						run: (*parser).callonsub_addr3,
						expr: &labeledExpr{
							pos:   position{line: 465, col: 5, offset: 11334},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 465, col: 8, offset: 11337},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 465, col: 8, offset: 11337},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 465, col: 16, offset: 11345},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 465, col: 20, offset: 11349},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 465, col: 28, offset: 11357},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 465, col: 32, offset: 11361},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 466, col: 5, offset: 11413},
						run: (*parser).callonsub_addr11,
						expr: &labeledExpr{
							pos:   position{line: 466, col: 5, offset: 11413},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 466, col: 8, offset: 11416},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 466, col: 8, offset: 11416},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 466, col: 16, offset: 11424},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 466, col: 20, offset: 11428},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 467, col: 5, offset: 11482},
						run: (*parser).callonsub_addr17,
						expr: &labeledExpr{
							pos:   position{line: 467, col: 5, offset: 11482},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 467, col: 7, offset: 11484},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "subnet",
			pos:  position{line: 469, col: 1, offset: 11535},
			expr: &actionExpr{
				pos: position{line: 470, col: 5, offset: 11546},
				run: (*parser).callonsubnet1,
				expr: &seqExpr{
					pos: position{line: 470, col: 5, offset: 11546},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 470, col: 5, offset: 11546},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 470, col: 7, offset: 11548},
								name: "sub_addr",
							},
						},
						&litMatcher{
							pos:        position{line: 470, col: 16, offset: 11557},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 470, col: 20, offset: 11561},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 470, col: 22, offset: 11563},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6subnet",
			pos:  position{line: 474, col: 1, offset: 11639},
			expr: &actionExpr{
				pos: position{line: 475, col: 5, offset: 11653},
				run: (*parser).callonip6subnet1,
				expr: &seqExpr{
					pos: position{line: 475, col: 5, offset: 11653},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 475, col: 5, offset: 11653},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 475, col: 7, offset: 11655},
								name: "ip6addr",
							},
						},
						&litMatcher{
							pos:        position{line: 475, col: 15, offset: 11663},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 475, col: 19, offset: 11667},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 475, col: 21, offset: 11669},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "integer",
			pos:  position{line: 479, col: 1, offset: 11735},
			expr: &actionExpr{
				pos: position{line: 480, col: 5, offset: 11747},
				run: (*parser).calloninteger1,
				expr: &labeledExpr{
					pos:   position{line: 480, col: 5, offset: 11747},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 480, col: 7, offset: 11749},
						name: "sinteger",
					},
				},
			},
		},
		{
			name: "sinteger",
			pos:  position{line: 484, col: 1, offset: 11793},
			expr: &actionExpr{
				pos: position{line: 485, col: 5, offset: 11806},
				run: (*parser).callonsinteger1,
				expr: &labeledExpr{
					pos:   position{line: 485, col: 5, offset: 11806},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 485, col: 11, offset: 11812},
						expr: &charClassMatcher{
							pos:        position{line: 485, col: 11, offset: 11812},
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
			pos:  position{line: 489, col: 1, offset: 11857},
			expr: &actionExpr{
				pos: position{line: 490, col: 5, offset: 11868},
				run: (*parser).callondouble1,
				expr: &labeledExpr{
					pos:   position{line: 490, col: 5, offset: 11868},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 490, col: 7, offset: 11870},
						name: "sdouble",
					},
				},
			},
		},
		{
			name: "sdouble",
			pos:  position{line: 494, col: 1, offset: 11917},
			expr: &choiceExpr{
				pos: position{line: 495, col: 5, offset: 11929},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 495, col: 5, offset: 11929},
						run: (*parser).callonsdouble2,
						expr: &seqExpr{
							pos: position{line: 495, col: 5, offset: 11929},
							exprs: []interface{}{
								&oneOrMoreExpr{
									pos: position{line: 495, col: 5, offset: 11929},
									expr: &ruleRefExpr{
										pos:  position{line: 495, col: 5, offset: 11929},
										name: "doubleInteger",
									},
								},
								&litMatcher{
									pos:        position{line: 495, col: 20, offset: 11944},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 495, col: 24, offset: 11948},
									expr: &ruleRefExpr{
										pos:  position{line: 495, col: 24, offset: 11948},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 495, col: 37, offset: 11961},
									expr: &ruleRefExpr{
										pos:  position{line: 495, col: 37, offset: 11961},
										name: "exponentPart",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 498, col: 5, offset: 12020},
						run: (*parser).callonsdouble11,
						expr: &seqExpr{
							pos: position{line: 498, col: 5, offset: 12020},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 498, col: 5, offset: 12020},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 498, col: 9, offset: 12024},
									expr: &ruleRefExpr{
										pos:  position{line: 498, col: 9, offset: 12024},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 498, col: 22, offset: 12037},
									expr: &ruleRefExpr{
										pos:  position{line: 498, col: 22, offset: 12037},
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
			pos:  position{line: 502, col: 1, offset: 12093},
			expr: &choiceExpr{
				pos: position{line: 503, col: 5, offset: 12111},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 503, col: 5, offset: 12111},
						val:        "0",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 504, col: 5, offset: 12119},
						exprs: []interface{}{
							&charClassMatcher{
								pos:        position{line: 504, col: 5, offset: 12119},
								val:        "[1-9]",
								ranges:     []rune{'1', '9'},
								ignoreCase: false,
								inverted:   false,
							},
							&zeroOrMoreExpr{
								pos: position{line: 504, col: 11, offset: 12125},
								expr: &charClassMatcher{
									pos:        position{line: 504, col: 11, offset: 12125},
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
			pos:  position{line: 506, col: 1, offset: 12133},
			expr: &charClassMatcher{
				pos:        position{line: 506, col: 15, offset: 12147},
				val:        "[0-9]",
				ranges:     []rune{'0', '9'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "signedInteger",
			pos:  position{line: 508, col: 1, offset: 12154},
			expr: &seqExpr{
				pos: position{line: 508, col: 17, offset: 12170},
				exprs: []interface{}{
					&zeroOrOneExpr{
						pos: position{line: 508, col: 17, offset: 12170},
						expr: &charClassMatcher{
							pos:        position{line: 508, col: 17, offset: 12170},
							val:        "[+-]",
							chars:      []rune{'+', '-'},
							ignoreCase: false,
							inverted:   false,
						},
					},
					&oneOrMoreExpr{
						pos: position{line: 508, col: 23, offset: 12176},
						expr: &ruleRefExpr{
							pos:  position{line: 508, col: 23, offset: 12176},
							name: "doubleDigit",
						},
					},
				},
			},
		},
		{
			name: "exponentPart",
			pos:  position{line: 510, col: 1, offset: 12190},
			expr: &seqExpr{
				pos: position{line: 510, col: 16, offset: 12205},
				exprs: []interface{}{
					&litMatcher{
						pos:        position{line: 510, col: 16, offset: 12205},
						val:        "e",
						ignoreCase: true,
					},
					&ruleRefExpr{
						pos:  position{line: 510, col: 21, offset: 12210},
						name: "signedInteger",
					},
				},
			},
		},
		{
			name: "h16",
			pos:  position{line: 512, col: 1, offset: 12225},
			expr: &actionExpr{
				pos: position{line: 512, col: 7, offset: 12231},
				run: (*parser).callonh161,
				expr: &labeledExpr{
					pos:   position{line: 512, col: 7, offset: 12231},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 512, col: 13, offset: 12237},
						expr: &ruleRefExpr{
							pos:  position{line: 512, col: 13, offset: 12237},
							name: "hexdigit",
						},
					},
				},
			},
		},
		{
			name: "hexdigit",
			pos:  position{line: 514, col: 1, offset: 12279},
			expr: &charClassMatcher{
				pos:        position{line: 514, col: 12, offset: 12290},
				val:        "[0-9a-fA-F]",
				ranges:     []rune{'0', '9', 'a', 'f', 'A', 'F'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "searchWord",
			pos:  position{line: 516, col: 1, offset: 12303},
			expr: &actionExpr{
				pos: position{line: 517, col: 5, offset: 12318},
				run: (*parser).callonsearchWord1,
				expr: &labeledExpr{
					pos:   position{line: 517, col: 5, offset: 12318},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 517, col: 11, offset: 12324},
						expr: &ruleRefExpr{
							pos:  position{line: 517, col: 11, offset: 12324},
							name: "searchWordPart",
						},
					},
				},
			},
		},
		{
			name: "searchWordPart",
			pos:  position{line: 519, col: 1, offset: 12374},
			expr: &choiceExpr{
				pos: position{line: 520, col: 5, offset: 12393},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 520, col: 5, offset: 12393},
						run: (*parser).callonsearchWordPart2,
						expr: &seqExpr{
							pos: position{line: 520, col: 5, offset: 12393},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 520, col: 5, offset: 12393},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 520, col: 10, offset: 12398},
									label: "s",
									expr: &choiceExpr{
										pos: position{line: 520, col: 13, offset: 12401},
										alternatives: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 520, col: 13, offset: 12401},
												name: "escapeSequence",
											},
											&ruleRefExpr{
												pos:  position{line: 520, col: 30, offset: 12418},
												name: "searchEscape",
											},
										},
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 521, col: 5, offset: 12455},
						run: (*parser).callonsearchWordPart9,
						expr: &seqExpr{
							pos: position{line: 521, col: 5, offset: 12455},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 521, col: 5, offset: 12455},
									expr: &choiceExpr{
										pos: position{line: 521, col: 7, offset: 12457},
										alternatives: []interface{}{
											&charClassMatcher{
												pos:        position{line: 521, col: 7, offset: 12457},
												val:        "[\\x00-\\x1F\\x5C(),!><=\\x22|\\x27;]",
												chars:      []rune{'\\', '(', ')', ',', '!', '>', '<', '=', '"', '|', '\'', ';'},
												ranges:     []rune{'\x00', '\x1f'},
												ignoreCase: false,
												inverted:   false,
											},
											&ruleRefExpr{
												pos:  position{line: 521, col: 42, offset: 12492},
												name: "ws",
											},
										},
									},
								},
								&anyMatcher{
									line: 521, col: 46, offset: 12496,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "quotedString",
			pos:  position{line: 523, col: 1, offset: 12530},
			expr: &choiceExpr{
				pos: position{line: 524, col: 5, offset: 12547},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 524, col: 5, offset: 12547},
						run: (*parser).callonquotedString2,
						expr: &seqExpr{
							pos: position{line: 524, col: 5, offset: 12547},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 524, col: 5, offset: 12547},
									val:        "\"",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 524, col: 9, offset: 12551},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 524, col: 11, offset: 12553},
										expr: &ruleRefExpr{
											pos:  position{line: 524, col: 11, offset: 12553},
											name: "doubleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 524, col: 29, offset: 12571},
									val:        "\"",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 525, col: 5, offset: 12608},
						run: (*parser).callonquotedString9,
						expr: &seqExpr{
							pos: position{line: 525, col: 5, offset: 12608},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 525, col: 5, offset: 12608},
									val:        "'",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 525, col: 9, offset: 12612},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 525, col: 11, offset: 12614},
										expr: &ruleRefExpr{
											pos:  position{line: 525, col: 11, offset: 12614},
											name: "singleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 525, col: 29, offset: 12632},
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
			pos:  position{line: 527, col: 1, offset: 12666},
			expr: &choiceExpr{
				pos: position{line: 528, col: 5, offset: 12687},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 528, col: 5, offset: 12687},
						run: (*parser).callondoubleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 528, col: 5, offset: 12687},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 528, col: 5, offset: 12687},
									expr: &choiceExpr{
										pos: position{line: 528, col: 7, offset: 12689},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 528, col: 7, offset: 12689},
												val:        "\"",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 528, col: 13, offset: 12695},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 528, col: 26, offset: 12708,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 529, col: 5, offset: 12745},
						run: (*parser).callondoubleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 529, col: 5, offset: 12745},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 529, col: 5, offset: 12745},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 529, col: 10, offset: 12750},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 529, col: 12, offset: 12752},
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
			pos:  position{line: 531, col: 1, offset: 12786},
			expr: &choiceExpr{
				pos: position{line: 532, col: 5, offset: 12807},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 532, col: 5, offset: 12807},
						run: (*parser).callonsingleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 532, col: 5, offset: 12807},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 532, col: 5, offset: 12807},
									expr: &choiceExpr{
										pos: position{line: 532, col: 7, offset: 12809},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 532, col: 7, offset: 12809},
												val:        "'",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 532, col: 13, offset: 12815},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 532, col: 26, offset: 12828,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 533, col: 5, offset: 12865},
						run: (*parser).callonsingleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 533, col: 5, offset: 12865},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 533, col: 5, offset: 12865},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 533, col: 10, offset: 12870},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 533, col: 12, offset: 12872},
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
			pos:  position{line: 535, col: 1, offset: 12906},
			expr: &choiceExpr{
				pos: position{line: 536, col: 5, offset: 12925},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 536, col: 5, offset: 12925},
						run: (*parser).callonescapeSequence2,
						expr: &seqExpr{
							pos: position{line: 536, col: 5, offset: 12925},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 536, col: 5, offset: 12925},
									val:        "x",
									ignoreCase: false,
								},
								&ruleRefExpr{
									pos:  position{line: 536, col: 9, offset: 12929},
									name: "hexdigit",
								},
								&ruleRefExpr{
									pos:  position{line: 536, col: 18, offset: 12938},
									name: "hexdigit",
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 537, col: 5, offset: 12989},
						name: "singleCharEscape",
					},
					&ruleRefExpr{
						pos:  position{line: 538, col: 5, offset: 13010},
						name: "unicodeEscape",
					},
				},
			},
		},
		{
			name: "singleCharEscape",
			pos:  position{line: 540, col: 1, offset: 13025},
			expr: &choiceExpr{
				pos: position{line: 541, col: 5, offset: 13046},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 541, col: 5, offset: 13046},
						val:        "'",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 542, col: 5, offset: 13054},
						val:        "\"",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 543, col: 5, offset: 13062},
						val:        "\\",
						ignoreCase: false,
					},
					&actionExpr{
						pos: position{line: 544, col: 5, offset: 13071},
						run: (*parser).callonsingleCharEscape5,
						expr: &litMatcher{
							pos:        position{line: 544, col: 5, offset: 13071},
							val:        "b",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 545, col: 5, offset: 13100},
						run: (*parser).callonsingleCharEscape7,
						expr: &litMatcher{
							pos:        position{line: 545, col: 5, offset: 13100},
							val:        "f",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 546, col: 5, offset: 13129},
						run: (*parser).callonsingleCharEscape9,
						expr: &litMatcher{
							pos:        position{line: 546, col: 5, offset: 13129},
							val:        "n",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 547, col: 5, offset: 13158},
						run: (*parser).callonsingleCharEscape11,
						expr: &litMatcher{
							pos:        position{line: 547, col: 5, offset: 13158},
							val:        "r",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 548, col: 5, offset: 13187},
						run: (*parser).callonsingleCharEscape13,
						expr: &litMatcher{
							pos:        position{line: 548, col: 5, offset: 13187},
							val:        "t",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 549, col: 5, offset: 13216},
						run: (*parser).callonsingleCharEscape15,
						expr: &litMatcher{
							pos:        position{line: 549, col: 5, offset: 13216},
							val:        "v",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "searchEscape",
			pos:  position{line: 551, col: 1, offset: 13242},
			expr: &choiceExpr{
				pos: position{line: 552, col: 5, offset: 13259},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 552, col: 5, offset: 13259},
						run: (*parser).callonsearchEscape2,
						expr: &litMatcher{
							pos:        position{line: 552, col: 5, offset: 13259},
							val:        "=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 553, col: 5, offset: 13287},
						run: (*parser).callonsearchEscape4,
						expr: &litMatcher{
							pos:        position{line: 553, col: 5, offset: 13287},
							val:        "*",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "unicodeEscape",
			pos:  position{line: 555, col: 1, offset: 13314},
			expr: &choiceExpr{
				pos: position{line: 556, col: 5, offset: 13332},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 556, col: 5, offset: 13332},
						run: (*parser).callonunicodeEscape2,
						expr: &seqExpr{
							pos: position{line: 556, col: 5, offset: 13332},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 556, col: 5, offset: 13332},
									val:        "u",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 556, col: 9, offset: 13336},
									label: "chars",
									expr: &seqExpr{
										pos: position{line: 556, col: 16, offset: 13343},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 556, col: 16, offset: 13343},
												name: "hexdigit",
											},
											&ruleRefExpr{
												pos:  position{line: 556, col: 25, offset: 13352},
												name: "hexdigit",
											},
											&ruleRefExpr{
												pos:  position{line: 556, col: 34, offset: 13361},
												name: "hexdigit",
											},
											&ruleRefExpr{
												pos:  position{line: 556, col: 43, offset: 13370},
												name: "hexdigit",
											},
										},
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 559, col: 5, offset: 13433},
						run: (*parser).callonunicodeEscape11,
						expr: &seqExpr{
							pos: position{line: 559, col: 5, offset: 13433},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 559, col: 5, offset: 13433},
									val:        "u",
									ignoreCase: false,
								},
								&litMatcher{
									pos:        position{line: 559, col: 9, offset: 13437},
									val:        "{",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 559, col: 13, offset: 13441},
									label: "chars",
									expr: &seqExpr{
										pos: position{line: 559, col: 20, offset: 13448},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 559, col: 20, offset: 13448},
												name: "hexdigit",
											},
											&zeroOrOneExpr{
												pos: position{line: 559, col: 29, offset: 13457},
												expr: &ruleRefExpr{
													pos:  position{line: 559, col: 29, offset: 13457},
													name: "hexdigit",
												},
											},
											&zeroOrOneExpr{
												pos: position{line: 559, col: 39, offset: 13467},
												expr: &ruleRefExpr{
													pos:  position{line: 559, col: 39, offset: 13467},
													name: "hexdigit",
												},
											},
											&zeroOrOneExpr{
												pos: position{line: 559, col: 49, offset: 13477},
												expr: &ruleRefExpr{
													pos:  position{line: 559, col: 49, offset: 13477},
													name: "hexdigit",
												},
											},
											&zeroOrOneExpr{
												pos: position{line: 559, col: 59, offset: 13487},
												expr: &ruleRefExpr{
													pos:  position{line: 559, col: 59, offset: 13487},
													name: "hexdigit",
												},
											},
											&zeroOrOneExpr{
												pos: position{line: 559, col: 69, offset: 13497},
												expr: &ruleRefExpr{
													pos:  position{line: 559, col: 69, offset: 13497},
													name: "hexdigit",
												},
											},
										},
									},
								},
								&litMatcher{
									pos:        position{line: 559, col: 80, offset: 13508},
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
			pos:  position{line: 563, col: 1, offset: 13562},
			expr: &actionExpr{
				pos: position{line: 564, col: 5, offset: 13575},
				run: (*parser).callonreString1,
				expr: &seqExpr{
					pos: position{line: 564, col: 5, offset: 13575},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 564, col: 5, offset: 13575},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 564, col: 9, offset: 13579},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 564, col: 11, offset: 13581},
								name: "reBody",
							},
						},
						&litMatcher{
							pos:        position{line: 564, col: 18, offset: 13588},
							val:        "/",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reBody",
			pos:  position{line: 566, col: 1, offset: 13611},
			expr: &actionExpr{
				pos: position{line: 567, col: 5, offset: 13622},
				run: (*parser).callonreBody1,
				expr: &oneOrMoreExpr{
					pos: position{line: 567, col: 5, offset: 13622},
					expr: &choiceExpr{
						pos: position{line: 567, col: 6, offset: 13623},
						alternatives: []interface{}{
							&charClassMatcher{
								pos:        position{line: 567, col: 6, offset: 13623},
								val:        "[^/\\\\]",
								chars:      []rune{'/', '\\'},
								ignoreCase: false,
								inverted:   true,
							},
							&litMatcher{
								pos:        position{line: 567, col: 13, offset: 13630},
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
			pos:  position{line: 569, col: 1, offset: 13670},
			expr: &charClassMatcher{
				pos:        position{line: 570, col: 5, offset: 13686},
				val:        "[\\x00-\\x1f\\\\]",
				chars:      []rune{'\\'},
				ranges:     []rune{'\x00', '\x1f'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "ws",
			pos:  position{line: 572, col: 1, offset: 13701},
			expr: &choiceExpr{
				pos: position{line: 573, col: 5, offset: 13708},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 573, col: 5, offset: 13708},
						val:        "\t",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 574, col: 5, offset: 13717},
						val:        "\v",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 575, col: 5, offset: 13726},
						val:        "\f",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 576, col: 5, offset: 13735},
						val:        " ",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 577, col: 5, offset: 13743},
						val:        "\u00a0",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 578, col: 5, offset: 13756},
						val:        "\ufeff",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name:        "_",
			displayName: "\"whitespace\"",
			pos:         position{line: 580, col: 1, offset: 13766},
			expr: &oneOrMoreExpr{
				pos: position{line: 580, col: 18, offset: 13783},
				expr: &ruleRefExpr{
					pos:  position{line: 580, col: 18, offset: 13783},
					name: "ws",
				},
			},
		},
		{
			name: "EOF",
			pos:  position{line: 582, col: 1, offset: 13788},
			expr: &notExpr{
				pos: position{line: 582, col: 7, offset: 13794},
				expr: &anyMatcher{
					line: 582, col: 8, offset: 13795,
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

func (c *current) onquery2(procs interface{}) (interface{}, error) {
	filt := makeFilterProc(makeBooleanLiteral(true))
	return makeSequentialProc(append([]interface{}{filt}, (procs.([]interface{}))...)), nil

}

func (p *parser) callonquery2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onquery2(stack["procs"])
}

func (c *current) onquery5(s, rest interface{}) (interface{}, error) {
	if len(rest.([]interface{})) == 0 {
		return s, nil
	} else {
		return makeSequentialProc(append([]interface{}{s}, (rest.([]interface{}))...)), nil
	}

}

func (p *parser) callonquery5() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onquery5(stack["s"], stack["rest"])
}

func (c *current) onquery14(s interface{}) (interface{}, error) {
	return makeSequentialProc([]interface{}{s}), nil

}

func (p *parser) callonquery14() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onquery14(stack["s"])
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

func (c *current) onnullLiteral1() (interface{}, error) {
	return makeLiteral("null", ""), nil
}

func (p *parser) callonnullLiteral1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onnullLiteral1()
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

func (c *current) onsearchWord1(chars interface{}) (interface{}, error) {
	return joinChars(chars), nil
}

func (p *parser) callonsearchWord1() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchWord1(stack["chars"])
}

func (c *current) onsearchWordPart2(s interface{}) (interface{}, error) {
	return s, nil
}

func (p *parser) callonsearchWordPart2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchWordPart2(stack["s"])
}

func (c *current) onsearchWordPart9() (interface{}, error) {
	return string(c.text), nil
}

func (p *parser) callonsearchWordPart9() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchWordPart9()
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

func (c *current) onsearchEscape2() (interface{}, error) {
	return "=", nil
}

func (p *parser) callonsearchEscape2() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchEscape2()
}

func (c *current) onsearchEscape4() (interface{}, error) {
	return "\\*", nil
}

func (p *parser) callonsearchEscape4() (interface{}, error) {
	stack := p.vstack[len(p.vstack)-1]
	_ = stack
	return p.cur.onsearchEscape4()
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

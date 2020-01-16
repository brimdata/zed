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
										name: "unsetLiteral",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 127, col: 5, offset: 3492},
						run: (*parser).callonsearchValue48,
						expr: &seqExpr{
							pos: position{line: 127, col: 5, offset: 3492},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 127, col: 5, offset: 3492},
									expr: &seqExpr{
										pos: position{line: 127, col: 7, offset: 3494},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 127, col: 8, offset: 3495},
												name: "searchKeywords",
											},
											&ruleRefExpr{
												pos:  position{line: 127, col: 24, offset: 3511},
												name: "_",
											},
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 127, col: 27, offset: 3514},
									label: "v",
									expr: &ruleRefExpr{
										pos:  position{line: 127, col: 29, offset: 3516},
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
			pos:  position{line: 135, col: 1, offset: 3715},
			expr: &choiceExpr{
				pos: position{line: 136, col: 5, offset: 3734},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 136, col: 5, offset: 3734},
						name: "andToken",
					},
					&ruleRefExpr{
						pos:  position{line: 137, col: 5, offset: 3747},
						name: "orToken",
					},
					&ruleRefExpr{
						pos:  position{line: 138, col: 5, offset: 3759},
						name: "inToken",
					},
				},
			},
		},
		{
			name: "booleanLiteral",
			pos:  position{line: 140, col: 1, offset: 3768},
			expr: &choiceExpr{
				pos: position{line: 141, col: 5, offset: 3787},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 141, col: 5, offset: 3787},
						run: (*parser).callonbooleanLiteral2,
						expr: &litMatcher{
							pos:        position{line: 141, col: 5, offset: 3787},
							val:        "true",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 142, col: 5, offset: 3852},
						run: (*parser).callonbooleanLiteral4,
						expr: &litMatcher{
							pos:        position{line: 142, col: 5, offset: 3852},
							val:        "false",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "unsetLiteral",
			pos:  position{line: 144, col: 1, offset: 3915},
			expr: &actionExpr{
				pos: position{line: 145, col: 5, offset: 3932},
				run: (*parser).callonunsetLiteral1,
				expr: &litMatcher{
					pos:        position{line: 145, col: 5, offset: 3932},
					val:        "nil",
					ignoreCase: false,
				},
			},
		},
		{
			name: "procList",
			pos:  position{line: 147, col: 1, offset: 3990},
			expr: &actionExpr{
				pos: position{line: 148, col: 5, offset: 4003},
				run: (*parser).callonprocList1,
				expr: &seqExpr{
					pos: position{line: 148, col: 5, offset: 4003},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 148, col: 5, offset: 4003},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 148, col: 11, offset: 4009},
								name: "procChain",
							},
						},
						&labeledExpr{
							pos:   position{line: 148, col: 21, offset: 4019},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 148, col: 26, offset: 4024},
								expr: &ruleRefExpr{
									pos:  position{line: 148, col: 26, offset: 4024},
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
			pos:  position{line: 157, col: 1, offset: 4248},
			expr: &actionExpr{
				pos: position{line: 158, col: 5, offset: 4266},
				run: (*parser).callonparallelChain1,
				expr: &seqExpr{
					pos: position{line: 158, col: 5, offset: 4266},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 158, col: 5, offset: 4266},
							expr: &ruleRefExpr{
								pos:  position{line: 158, col: 5, offset: 4266},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 158, col: 8, offset: 4269},
							val:        ";",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 158, col: 12, offset: 4273},
							expr: &ruleRefExpr{
								pos:  position{line: 158, col: 12, offset: 4273},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 158, col: 15, offset: 4276},
							label: "ch",
							expr: &ruleRefExpr{
								pos:  position{line: 158, col: 18, offset: 4279},
								name: "procChain",
							},
						},
					},
				},
			},
		},
		{
			name: "proc",
			pos:  position{line: 160, col: 1, offset: 4329},
			expr: &choiceExpr{
				pos: position{line: 161, col: 5, offset: 4338},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 161, col: 5, offset: 4338},
						name: "simpleProc",
					},
					&ruleRefExpr{
						pos:  position{line: 162, col: 5, offset: 4353},
						name: "reducerProc",
					},
					&actionExpr{
						pos: position{line: 163, col: 5, offset: 4369},
						run: (*parser).callonproc4,
						expr: &seqExpr{
							pos: position{line: 163, col: 5, offset: 4369},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 163, col: 5, offset: 4369},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 163, col: 9, offset: 4373},
									expr: &ruleRefExpr{
										pos:  position{line: 163, col: 9, offset: 4373},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 163, col: 12, offset: 4376},
									label: "proc",
									expr: &ruleRefExpr{
										pos:  position{line: 163, col: 17, offset: 4381},
										name: "procList",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 163, col: 26, offset: 4390},
									expr: &ruleRefExpr{
										pos:  position{line: 163, col: 26, offset: 4390},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 163, col: 29, offset: 4393},
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
			pos:  position{line: 167, col: 1, offset: 4429},
			expr: &actionExpr{
				pos: position{line: 168, col: 5, offset: 4441},
				run: (*parser).callongroupBy1,
				expr: &seqExpr{
					pos: position{line: 168, col: 5, offset: 4441},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 168, col: 5, offset: 4441},
							val:        "by",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 168, col: 11, offset: 4447},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 168, col: 13, offset: 4449},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 168, col: 18, offset: 4454},
								name: "fieldExprList",
							},
						},
					},
				},
			},
		},
		{
			name: "everyDur",
			pos:  position{line: 170, col: 1, offset: 4490},
			expr: &actionExpr{
				pos: position{line: 171, col: 5, offset: 4503},
				run: (*parser).calloneveryDur1,
				expr: &seqExpr{
					pos: position{line: 171, col: 5, offset: 4503},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 171, col: 5, offset: 4503},
							val:        "every",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 171, col: 14, offset: 4512},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 171, col: 16, offset: 4514},
							label: "dur",
							expr: &ruleRefExpr{
								pos:  position{line: 171, col: 20, offset: 4518},
								name: "duration",
							},
						},
					},
				},
			},
		},
		{
			name: "equalityToken",
			pos:  position{line: 173, col: 1, offset: 4548},
			expr: &choiceExpr{
				pos: position{line: 174, col: 5, offset: 4566},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 174, col: 5, offset: 4566},
						run: (*parser).callonequalityToken2,
						expr: &litMatcher{
							pos:        position{line: 174, col: 5, offset: 4566},
							val:        "=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 175, col: 5, offset: 4596},
						run: (*parser).callonequalityToken4,
						expr: &litMatcher{
							pos:        position{line: 175, col: 5, offset: 4596},
							val:        "!=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 176, col: 5, offset: 4628},
						run: (*parser).callonequalityToken6,
						expr: &litMatcher{
							pos:        position{line: 176, col: 5, offset: 4628},
							val:        "<=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 177, col: 5, offset: 4659},
						run: (*parser).callonequalityToken8,
						expr: &litMatcher{
							pos:        position{line: 177, col: 5, offset: 4659},
							val:        ">=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 178, col: 5, offset: 4690},
						run: (*parser).callonequalityToken10,
						expr: &litMatcher{
							pos:        position{line: 178, col: 5, offset: 4690},
							val:        "<",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 179, col: 5, offset: 4719},
						run: (*parser).callonequalityToken12,
						expr: &litMatcher{
							pos:        position{line: 179, col: 5, offset: 4719},
							val:        ">",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "types",
			pos:  position{line: 181, col: 1, offset: 4745},
			expr: &choiceExpr{
				pos: position{line: 182, col: 5, offset: 4755},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 182, col: 5, offset: 4755},
						val:        "bool",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 183, col: 5, offset: 4766},
						val:        "int",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 184, col: 5, offset: 4776},
						val:        "count",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 185, col: 5, offset: 4788},
						val:        "double",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 186, col: 5, offset: 4801},
						val:        "string",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 187, col: 5, offset: 4814},
						val:        "addr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 188, col: 5, offset: 4825},
						val:        "subnet",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 189, col: 5, offset: 4838},
						val:        "port",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "dash",
			pos:  position{line: 191, col: 1, offset: 4846},
			expr: &choiceExpr{
				pos: position{line: 191, col: 8, offset: 4853},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 191, col: 8, offset: 4853},
						val:        "-",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 191, col: 14, offset: 4859},
						val:        "—",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 191, col: 25, offset: 4870},
						val:        "–",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 191, col: 36, offset: 4881},
						exprs: []interface{}{
							&litMatcher{
								pos:        position{line: 191, col: 36, offset: 4881},
								val:        "to",
								ignoreCase: false,
							},
							&andExpr{
								pos: position{line: 191, col: 40, offset: 4885},
								expr: &ruleRefExpr{
									pos:  position{line: 191, col: 42, offset: 4887},
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
			pos:  position{line: 193, col: 1, offset: 4891},
			expr: &litMatcher{
				pos:        position{line: 193, col: 12, offset: 4902},
				val:        "and",
				ignoreCase: false,
			},
		},
		{
			name: "orToken",
			pos:  position{line: 194, col: 1, offset: 4908},
			expr: &litMatcher{
				pos:        position{line: 194, col: 11, offset: 4918},
				val:        "or",
				ignoreCase: false,
			},
		},
		{
			name: "inToken",
			pos:  position{line: 195, col: 1, offset: 4923},
			expr: &litMatcher{
				pos:        position{line: 195, col: 11, offset: 4933},
				val:        "in",
				ignoreCase: false,
			},
		},
		{
			name: "notToken",
			pos:  position{line: 196, col: 1, offset: 4938},
			expr: &litMatcher{
				pos:        position{line: 196, col: 12, offset: 4949},
				val:        "not",
				ignoreCase: false,
			},
		},
		{
			name: "fieldName",
			pos:  position{line: 198, col: 1, offset: 4956},
			expr: &actionExpr{
				pos: position{line: 198, col: 13, offset: 4968},
				run: (*parser).callonfieldName1,
				expr: &seqExpr{
					pos: position{line: 198, col: 13, offset: 4968},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 198, col: 13, offset: 4968},
							name: "fieldNameStart",
						},
						&zeroOrMoreExpr{
							pos: position{line: 198, col: 28, offset: 4983},
							expr: &ruleRefExpr{
								pos:  position{line: 198, col: 28, offset: 4983},
								name: "fieldNameRest",
							},
						},
					},
				},
			},
		},
		{
			name: "fieldNameStart",
			pos:  position{line: 200, col: 1, offset: 5030},
			expr: &charClassMatcher{
				pos:        position{line: 200, col: 18, offset: 5047},
				val:        "[A-Za-z_$]",
				chars:      []rune{'_', '$'},
				ranges:     []rune{'A', 'Z', 'a', 'z'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "fieldNameRest",
			pos:  position{line: 201, col: 1, offset: 5058},
			expr: &choiceExpr{
				pos: position{line: 201, col: 17, offset: 5074},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 201, col: 17, offset: 5074},
						name: "fieldNameStart",
					},
					&charClassMatcher{
						pos:        position{line: 201, col: 34, offset: 5091},
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
			pos:  position{line: 203, col: 1, offset: 5098},
			expr: &actionExpr{
				pos: position{line: 204, col: 4, offset: 5116},
				run: (*parser).callonfieldReference1,
				expr: &seqExpr{
					pos: position{line: 204, col: 4, offset: 5116},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 204, col: 4, offset: 5116},
							label: "base",
							expr: &ruleRefExpr{
								pos:  position{line: 204, col: 9, offset: 5121},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 204, col: 19, offset: 5131},
							label: "derefs",
							expr: &zeroOrMoreExpr{
								pos: position{line: 204, col: 26, offset: 5138},
								expr: &choiceExpr{
									pos: position{line: 205, col: 8, offset: 5147},
									alternatives: []interface{}{
										&actionExpr{
											pos: position{line: 205, col: 8, offset: 5147},
											run: (*parser).callonfieldReference8,
											expr: &seqExpr{
												pos: position{line: 205, col: 8, offset: 5147},
												exprs: []interface{}{
													&litMatcher{
														pos:        position{line: 205, col: 8, offset: 5147},
														val:        ".",
														ignoreCase: false,
													},
													&labeledExpr{
														pos:   position{line: 205, col: 12, offset: 5151},
														label: "field",
														expr: &ruleRefExpr{
															pos:  position{line: 205, col: 18, offset: 5157},
															name: "fieldName",
														},
													},
												},
											},
										},
										&actionExpr{
											pos: position{line: 206, col: 8, offset: 5238},
											run: (*parser).callonfieldReference13,
											expr: &seqExpr{
												pos: position{line: 206, col: 8, offset: 5238},
												exprs: []interface{}{
													&litMatcher{
														pos:        position{line: 206, col: 8, offset: 5238},
														val:        "[",
														ignoreCase: false,
													},
													&labeledExpr{
														pos:   position{line: 206, col: 12, offset: 5242},
														label: "index",
														expr: &ruleRefExpr{
															pos:  position{line: 206, col: 18, offset: 5248},
															name: "sinteger",
														},
													},
													&litMatcher{
														pos:        position{line: 206, col: 27, offset: 5257},
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
			pos:  position{line: 211, col: 1, offset: 5373},
			expr: &choiceExpr{
				pos: position{line: 212, col: 5, offset: 5387},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 212, col: 5, offset: 5387},
						run: (*parser).callonfieldExpr2,
						expr: &seqExpr{
							pos: position{line: 212, col: 5, offset: 5387},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 212, col: 5, offset: 5387},
									label: "op",
									expr: &ruleRefExpr{
										pos:  position{line: 212, col: 8, offset: 5390},
										name: "fieldOp",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 212, col: 16, offset: 5398},
									expr: &ruleRefExpr{
										pos:  position{line: 212, col: 16, offset: 5398},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 212, col: 19, offset: 5401},
									val:        "(",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 212, col: 23, offset: 5405},
									expr: &ruleRefExpr{
										pos:  position{line: 212, col: 23, offset: 5405},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 212, col: 26, offset: 5408},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 212, col: 32, offset: 5414},
										name: "fieldReference",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 212, col: 47, offset: 5429},
									expr: &ruleRefExpr{
										pos:  position{line: 212, col: 47, offset: 5429},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 212, col: 50, offset: 5432},
									val:        ")",
									ignoreCase: false,
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 215, col: 5, offset: 5496},
						name: "fieldReference",
					},
				},
			},
		},
		{
			name: "fieldOp",
			pos:  position{line: 217, col: 1, offset: 5512},
			expr: &actionExpr{
				pos: position{line: 218, col: 5, offset: 5524},
				run: (*parser).callonfieldOp1,
				expr: &litMatcher{
					pos:        position{line: 218, col: 5, offset: 5524},
					val:        "len",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldExprList",
			pos:  position{line: 220, col: 1, offset: 5554},
			expr: &actionExpr{
				pos: position{line: 221, col: 5, offset: 5572},
				run: (*parser).callonfieldExprList1,
				expr: &seqExpr{
					pos: position{line: 221, col: 5, offset: 5572},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 221, col: 5, offset: 5572},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 221, col: 11, offset: 5578},
								name: "fieldExpr",
							},
						},
						&labeledExpr{
							pos:   position{line: 221, col: 21, offset: 5588},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 221, col: 26, offset: 5593},
								expr: &seqExpr{
									pos: position{line: 221, col: 27, offset: 5594},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 221, col: 27, offset: 5594},
											expr: &ruleRefExpr{
												pos:  position{line: 221, col: 27, offset: 5594},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 221, col: 30, offset: 5597},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 221, col: 34, offset: 5601},
											expr: &ruleRefExpr{
												pos:  position{line: 221, col: 34, offset: 5601},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 221, col: 37, offset: 5604},
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
			pos:  position{line: 231, col: 1, offset: 5799},
			expr: &actionExpr{
				pos: position{line: 232, col: 5, offset: 5819},
				run: (*parser).callonfieldRefDotOnly1,
				expr: &seqExpr{
					pos: position{line: 232, col: 5, offset: 5819},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 232, col: 5, offset: 5819},
							label: "base",
							expr: &ruleRefExpr{
								pos:  position{line: 232, col: 10, offset: 5824},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 232, col: 20, offset: 5834},
							label: "refs",
							expr: &zeroOrMoreExpr{
								pos: position{line: 232, col: 25, offset: 5839},
								expr: &actionExpr{
									pos: position{line: 232, col: 26, offset: 5840},
									run: (*parser).callonfieldRefDotOnly7,
									expr: &seqExpr{
										pos: position{line: 232, col: 26, offset: 5840},
										exprs: []interface{}{
											&litMatcher{
												pos:        position{line: 232, col: 26, offset: 5840},
												val:        ".",
												ignoreCase: false,
											},
											&labeledExpr{
												pos:   position{line: 232, col: 30, offset: 5844},
												label: "field",
												expr: &ruleRefExpr{
													pos:  position{line: 232, col: 36, offset: 5850},
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
			pos:  position{line: 236, col: 1, offset: 5975},
			expr: &actionExpr{
				pos: position{line: 237, col: 5, offset: 5999},
				run: (*parser).callonfieldRefDotOnlyList1,
				expr: &seqExpr{
					pos: position{line: 237, col: 5, offset: 5999},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 237, col: 5, offset: 5999},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 237, col: 11, offset: 6005},
								name: "fieldRefDotOnly",
							},
						},
						&labeledExpr{
							pos:   position{line: 237, col: 27, offset: 6021},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 237, col: 32, offset: 6026},
								expr: &actionExpr{
									pos: position{line: 237, col: 33, offset: 6027},
									run: (*parser).callonfieldRefDotOnlyList7,
									expr: &seqExpr{
										pos: position{line: 237, col: 33, offset: 6027},
										exprs: []interface{}{
											&zeroOrOneExpr{
												pos: position{line: 237, col: 33, offset: 6027},
												expr: &ruleRefExpr{
													pos:  position{line: 237, col: 33, offset: 6027},
													name: "_",
												},
											},
											&litMatcher{
												pos:        position{line: 237, col: 36, offset: 6030},
												val:        ",",
												ignoreCase: false,
											},
											&zeroOrOneExpr{
												pos: position{line: 237, col: 40, offset: 6034},
												expr: &ruleRefExpr{
													pos:  position{line: 237, col: 40, offset: 6034},
													name: "_",
												},
											},
											&labeledExpr{
												pos:   position{line: 237, col: 43, offset: 6037},
												label: "ref",
												expr: &ruleRefExpr{
													pos:  position{line: 237, col: 47, offset: 6041},
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
			pos:  position{line: 245, col: 1, offset: 6221},
			expr: &actionExpr{
				pos: position{line: 246, col: 5, offset: 6239},
				run: (*parser).callonfieldNameList1,
				expr: &seqExpr{
					pos: position{line: 246, col: 5, offset: 6239},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 246, col: 5, offset: 6239},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 246, col: 11, offset: 6245},
								name: "fieldName",
							},
						},
						&labeledExpr{
							pos:   position{line: 246, col: 21, offset: 6255},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 246, col: 26, offset: 6260},
								expr: &seqExpr{
									pos: position{line: 246, col: 27, offset: 6261},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 246, col: 27, offset: 6261},
											expr: &ruleRefExpr{
												pos:  position{line: 246, col: 27, offset: 6261},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 246, col: 30, offset: 6264},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 246, col: 34, offset: 6268},
											expr: &ruleRefExpr{
												pos:  position{line: 246, col: 34, offset: 6268},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 246, col: 37, offset: 6271},
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
			pos:  position{line: 254, col: 1, offset: 6464},
			expr: &actionExpr{
				pos: position{line: 255, col: 5, offset: 6476},
				run: (*parser).calloncountOp1,
				expr: &litMatcher{
					pos:        position{line: 255, col: 5, offset: 6476},
					val:        "count",
					ignoreCase: true,
				},
			},
		},
		{
			name: "fieldReducerOp",
			pos:  position{line: 257, col: 1, offset: 6510},
			expr: &choiceExpr{
				pos: position{line: 258, col: 5, offset: 6529},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 258, col: 5, offset: 6529},
						run: (*parser).callonfieldReducerOp2,
						expr: &litMatcher{
							pos:        position{line: 258, col: 5, offset: 6529},
							val:        "sum",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 259, col: 5, offset: 6563},
						run: (*parser).callonfieldReducerOp4,
						expr: &litMatcher{
							pos:        position{line: 259, col: 5, offset: 6563},
							val:        "avg",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 260, col: 5, offset: 6597},
						run: (*parser).callonfieldReducerOp6,
						expr: &litMatcher{
							pos:        position{line: 260, col: 5, offset: 6597},
							val:        "stdev",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 261, col: 5, offset: 6634},
						run: (*parser).callonfieldReducerOp8,
						expr: &litMatcher{
							pos:        position{line: 261, col: 5, offset: 6634},
							val:        "sd",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 262, col: 5, offset: 6670},
						run: (*parser).callonfieldReducerOp10,
						expr: &litMatcher{
							pos:        position{line: 262, col: 5, offset: 6670},
							val:        "var",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 263, col: 5, offset: 6704},
						run: (*parser).callonfieldReducerOp12,
						expr: &litMatcher{
							pos:        position{line: 263, col: 5, offset: 6704},
							val:        "entropy",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 264, col: 5, offset: 6745},
						run: (*parser).callonfieldReducerOp14,
						expr: &litMatcher{
							pos:        position{line: 264, col: 5, offset: 6745},
							val:        "min",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 265, col: 5, offset: 6779},
						run: (*parser).callonfieldReducerOp16,
						expr: &litMatcher{
							pos:        position{line: 265, col: 5, offset: 6779},
							val:        "max",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 266, col: 5, offset: 6813},
						run: (*parser).callonfieldReducerOp18,
						expr: &litMatcher{
							pos:        position{line: 266, col: 5, offset: 6813},
							val:        "first",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 267, col: 5, offset: 6851},
						run: (*parser).callonfieldReducerOp20,
						expr: &litMatcher{
							pos:        position{line: 267, col: 5, offset: 6851},
							val:        "last",
							ignoreCase: true,
						},
					},
					&actionExpr{
						pos: position{line: 268, col: 5, offset: 6887},
						run: (*parser).callonfieldReducerOp22,
						expr: &litMatcher{
							pos:        position{line: 268, col: 5, offset: 6887},
							val:        "countdistinct",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "paddedFieldName",
			pos:  position{line: 270, col: 1, offset: 6937},
			expr: &actionExpr{
				pos: position{line: 270, col: 19, offset: 6955},
				run: (*parser).callonpaddedFieldName1,
				expr: &seqExpr{
					pos: position{line: 270, col: 19, offset: 6955},
					exprs: []interface{}{
						&zeroOrOneExpr{
							pos: position{line: 270, col: 19, offset: 6955},
							expr: &ruleRefExpr{
								pos:  position{line: 270, col: 19, offset: 6955},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 270, col: 22, offset: 6958},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 270, col: 28, offset: 6964},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 270, col: 38, offset: 6974},
							expr: &ruleRefExpr{
								pos:  position{line: 270, col: 38, offset: 6974},
								name: "_",
							},
						},
					},
				},
			},
		},
		{
			name: "countReducer",
			pos:  position{line: 272, col: 1, offset: 7000},
			expr: &actionExpr{
				pos: position{line: 273, col: 5, offset: 7017},
				run: (*parser).calloncountReducer1,
				expr: &seqExpr{
					pos: position{line: 273, col: 5, offset: 7017},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 273, col: 5, offset: 7017},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 273, col: 8, offset: 7020},
								name: "countOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 273, col: 16, offset: 7028},
							expr: &ruleRefExpr{
								pos:  position{line: 273, col: 16, offset: 7028},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 273, col: 19, offset: 7031},
							val:        "(",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 273, col: 23, offset: 7035},
							label: "field",
							expr: &zeroOrOneExpr{
								pos: position{line: 273, col: 29, offset: 7041},
								expr: &ruleRefExpr{
									pos:  position{line: 273, col: 29, offset: 7041},
									name: "paddedFieldName",
								},
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 273, col: 47, offset: 7059},
							expr: &ruleRefExpr{
								pos:  position{line: 273, col: 47, offset: 7059},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 273, col: 50, offset: 7062},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "fieldReducer",
			pos:  position{line: 277, col: 1, offset: 7121},
			expr: &actionExpr{
				pos: position{line: 278, col: 5, offset: 7138},
				run: (*parser).callonfieldReducer1,
				expr: &seqExpr{
					pos: position{line: 278, col: 5, offset: 7138},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 278, col: 5, offset: 7138},
							label: "op",
							expr: &ruleRefExpr{
								pos:  position{line: 278, col: 8, offset: 7141},
								name: "fieldReducerOp",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 278, col: 23, offset: 7156},
							expr: &ruleRefExpr{
								pos:  position{line: 278, col: 23, offset: 7156},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 278, col: 26, offset: 7159},
							val:        "(",
							ignoreCase: false,
						},
						&zeroOrOneExpr{
							pos: position{line: 278, col: 30, offset: 7163},
							expr: &ruleRefExpr{
								pos:  position{line: 278, col: 30, offset: 7163},
								name: "_",
							},
						},
						&labeledExpr{
							pos:   position{line: 278, col: 33, offset: 7166},
							label: "field",
							expr: &ruleRefExpr{
								pos:  position{line: 278, col: 39, offset: 7172},
								name: "fieldName",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 278, col: 50, offset: 7183},
							expr: &ruleRefExpr{
								pos:  position{line: 278, col: 50, offset: 7183},
								name: "_",
							},
						},
						&litMatcher{
							pos:        position{line: 278, col: 53, offset: 7186},
							val:        ")",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reducerProc",
			pos:  position{line: 282, col: 1, offset: 7253},
			expr: &actionExpr{
				pos: position{line: 283, col: 5, offset: 7269},
				run: (*parser).callonreducerProc1,
				expr: &seqExpr{
					pos: position{line: 283, col: 5, offset: 7269},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 283, col: 5, offset: 7269},
							label: "every",
							expr: &zeroOrOneExpr{
								pos: position{line: 283, col: 11, offset: 7275},
								expr: &seqExpr{
									pos: position{line: 283, col: 12, offset: 7276},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 283, col: 12, offset: 7276},
											name: "everyDur",
										},
										&ruleRefExpr{
											pos:  position{line: 283, col: 21, offset: 7285},
											name: "_",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 283, col: 25, offset: 7289},
							label: "reducers",
							expr: &ruleRefExpr{
								pos:  position{line: 283, col: 34, offset: 7298},
								name: "reducerList",
							},
						},
						&labeledExpr{
							pos:   position{line: 283, col: 46, offset: 7310},
							label: "keys",
							expr: &zeroOrOneExpr{
								pos: position{line: 283, col: 51, offset: 7315},
								expr: &seqExpr{
									pos: position{line: 283, col: 52, offset: 7316},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 283, col: 52, offset: 7316},
											name: "_",
										},
										&ruleRefExpr{
											pos:  position{line: 283, col: 54, offset: 7318},
											name: "groupBy",
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 283, col: 64, offset: 7328},
							label: "limit",
							expr: &zeroOrOneExpr{
								pos: position{line: 283, col: 70, offset: 7334},
								expr: &ruleRefExpr{
									pos:  position{line: 283, col: 70, offset: 7334},
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
			pos:  position{line: 301, col: 1, offset: 7691},
			expr: &actionExpr{
				pos: position{line: 302, col: 5, offset: 7704},
				run: (*parser).callonasClause1,
				expr: &seqExpr{
					pos: position{line: 302, col: 5, offset: 7704},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 302, col: 5, offset: 7704},
							val:        "as",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 302, col: 11, offset: 7710},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 302, col: 13, offset: 7712},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 302, col: 15, offset: 7714},
								name: "fieldName",
							},
						},
					},
				},
			},
		},
		{
			name: "reducerExpr",
			pos:  position{line: 304, col: 1, offset: 7743},
			expr: &choiceExpr{
				pos: position{line: 305, col: 5, offset: 7759},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 305, col: 5, offset: 7759},
						run: (*parser).callonreducerExpr2,
						expr: &seqExpr{
							pos: position{line: 305, col: 5, offset: 7759},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 305, col: 5, offset: 7759},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 305, col: 11, offset: 7765},
										name: "fieldName",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 305, col: 21, offset: 7775},
									expr: &ruleRefExpr{
										pos:  position{line: 305, col: 21, offset: 7775},
										name: "_",
									},
								},
								&litMatcher{
									pos:        position{line: 305, col: 24, offset: 7778},
									val:        "=",
									ignoreCase: false,
								},
								&zeroOrOneExpr{
									pos: position{line: 305, col: 28, offset: 7782},
									expr: &ruleRefExpr{
										pos:  position{line: 305, col: 28, offset: 7782},
										name: "_",
									},
								},
								&labeledExpr{
									pos:   position{line: 305, col: 31, offset: 7785},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 305, col: 33, offset: 7787},
										name: "reducer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 308, col: 5, offset: 7850},
						run: (*parser).callonreducerExpr13,
						expr: &seqExpr{
							pos: position{line: 308, col: 5, offset: 7850},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 308, col: 5, offset: 7850},
									label: "f",
									expr: &ruleRefExpr{
										pos:  position{line: 308, col: 7, offset: 7852},
										name: "reducer",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 308, col: 15, offset: 7860},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 308, col: 17, offset: 7862},
									label: "field",
									expr: &ruleRefExpr{
										pos:  position{line: 308, col: 23, offset: 7868},
										name: "asClause",
									},
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 311, col: 5, offset: 7932},
						name: "reducer",
					},
				},
			},
		},
		{
			name: "reducer",
			pos:  position{line: 313, col: 1, offset: 7941},
			expr: &choiceExpr{
				pos: position{line: 314, col: 5, offset: 7953},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 314, col: 5, offset: 7953},
						name: "countReducer",
					},
					&ruleRefExpr{
						pos:  position{line: 315, col: 5, offset: 7970},
						name: "fieldReducer",
					},
				},
			},
		},
		{
			name: "reducerList",
			pos:  position{line: 317, col: 1, offset: 7984},
			expr: &actionExpr{
				pos: position{line: 318, col: 5, offset: 8000},
				run: (*parser).callonreducerList1,
				expr: &seqExpr{
					pos: position{line: 318, col: 5, offset: 8000},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 318, col: 5, offset: 8000},
							label: "first",
							expr: &ruleRefExpr{
								pos:  position{line: 318, col: 11, offset: 8006},
								name: "reducerExpr",
							},
						},
						&labeledExpr{
							pos:   position{line: 318, col: 23, offset: 8018},
							label: "rest",
							expr: &zeroOrMoreExpr{
								pos: position{line: 318, col: 28, offset: 8023},
								expr: &seqExpr{
									pos: position{line: 318, col: 29, offset: 8024},
									exprs: []interface{}{
										&zeroOrOneExpr{
											pos: position{line: 318, col: 29, offset: 8024},
											expr: &ruleRefExpr{
												pos:  position{line: 318, col: 29, offset: 8024},
												name: "_",
											},
										},
										&litMatcher{
											pos:        position{line: 318, col: 32, offset: 8027},
											val:        ",",
											ignoreCase: false,
										},
										&zeroOrOneExpr{
											pos: position{line: 318, col: 36, offset: 8031},
											expr: &ruleRefExpr{
												pos:  position{line: 318, col: 36, offset: 8031},
												name: "_",
											},
										},
										&ruleRefExpr{
											pos:  position{line: 318, col: 39, offset: 8034},
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
			pos:  position{line: 326, col: 1, offset: 8231},
			expr: &choiceExpr{
				pos: position{line: 327, col: 5, offset: 8246},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 327, col: 5, offset: 8246},
						name: "sort",
					},
					&ruleRefExpr{
						pos:  position{line: 328, col: 5, offset: 8255},
						name: "top",
					},
					&ruleRefExpr{
						pos:  position{line: 329, col: 5, offset: 8263},
						name: "cut",
					},
					&ruleRefExpr{
						pos:  position{line: 330, col: 5, offset: 8271},
						name: "head",
					},
					&ruleRefExpr{
						pos:  position{line: 331, col: 5, offset: 8280},
						name: "tail",
					},
					&ruleRefExpr{
						pos:  position{line: 332, col: 5, offset: 8289},
						name: "filter",
					},
					&ruleRefExpr{
						pos:  position{line: 333, col: 5, offset: 8300},
						name: "uniq",
					},
				},
			},
		},
		{
			name: "sort",
			pos:  position{line: 335, col: 1, offset: 8306},
			expr: &actionExpr{
				pos: position{line: 336, col: 5, offset: 8315},
				run: (*parser).callonsort1,
				expr: &seqExpr{
					pos: position{line: 336, col: 5, offset: 8315},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 336, col: 5, offset: 8315},
							val:        "sort",
							ignoreCase: true,
						},
						&labeledExpr{
							pos:   position{line: 336, col: 13, offset: 8323},
							label: "args",
							expr: &ruleRefExpr{
								pos:  position{line: 336, col: 18, offset: 8328},
								name: "sortArgs",
							},
						},
						&labeledExpr{
							pos:   position{line: 336, col: 27, offset: 8337},
							label: "list",
							expr: &zeroOrOneExpr{
								pos: position{line: 336, col: 32, offset: 8342},
								expr: &actionExpr{
									pos: position{line: 336, col: 33, offset: 8343},
									run: (*parser).callonsort8,
									expr: &seqExpr{
										pos: position{line: 336, col: 33, offset: 8343},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 336, col: 33, offset: 8343},
												name: "_",
											},
											&labeledExpr{
												pos:   position{line: 336, col: 35, offset: 8345},
												label: "l",
												expr: &ruleRefExpr{
													pos:  position{line: 336, col: 37, offset: 8347},
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
			pos:  position{line: 340, col: 1, offset: 8424},
			expr: &zeroOrMoreExpr{
				pos: position{line: 340, col: 12, offset: 8435},
				expr: &actionExpr{
					pos: position{line: 340, col: 13, offset: 8436},
					run: (*parser).callonsortArgs2,
					expr: &seqExpr{
						pos: position{line: 340, col: 13, offset: 8436},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 340, col: 13, offset: 8436},
								name: "_",
							},
							&labeledExpr{
								pos:   position{line: 340, col: 15, offset: 8438},
								label: "a",
								expr: &ruleRefExpr{
									pos:  position{line: 340, col: 17, offset: 8440},
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
			pos:  position{line: 342, col: 1, offset: 8469},
			expr: &choiceExpr{
				pos: position{line: 343, col: 5, offset: 8481},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 343, col: 5, offset: 8481},
						run: (*parser).callonsortArg2,
						expr: &seqExpr{
							pos: position{line: 343, col: 5, offset: 8481},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 343, col: 5, offset: 8481},
									val:        "-limit",
									ignoreCase: false,
								},
								&ruleRefExpr{
									pos:  position{line: 343, col: 14, offset: 8490},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 343, col: 16, offset: 8492},
									label: "limit",
									expr: &ruleRefExpr{
										pos:  position{line: 343, col: 22, offset: 8498},
										name: "sinteger",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 344, col: 5, offset: 8551},
						run: (*parser).callonsortArg8,
						expr: &litMatcher{
							pos:        position{line: 344, col: 5, offset: 8551},
							val:        "-r",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 345, col: 5, offset: 8594},
						run: (*parser).callonsortArg10,
						expr: &seqExpr{
							pos: position{line: 345, col: 5, offset: 8594},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 345, col: 5, offset: 8594},
									val:        "-nulls",
									ignoreCase: false,
								},
								&ruleRefExpr{
									pos:  position{line: 345, col: 14, offset: 8603},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 345, col: 16, offset: 8605},
									label: "where",
									expr: &actionExpr{
										pos: position{line: 345, col: 23, offset: 8612},
										run: (*parser).callonsortArg15,
										expr: &choiceExpr{
											pos: position{line: 345, col: 24, offset: 8613},
											alternatives: []interface{}{
												&litMatcher{
													pos:        position{line: 345, col: 24, offset: 8613},
													val:        "first",
													ignoreCase: false,
												},
												&litMatcher{
													pos:        position{line: 345, col: 34, offset: 8623},
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
			pos:  position{line: 347, col: 1, offset: 8705},
			expr: &actionExpr{
				pos: position{line: 348, col: 5, offset: 8713},
				run: (*parser).callontop1,
				expr: &seqExpr{
					pos: position{line: 348, col: 5, offset: 8713},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 348, col: 5, offset: 8713},
							val:        "top",
							ignoreCase: true,
						},
						&labeledExpr{
							pos:   position{line: 348, col: 12, offset: 8720},
							label: "limit",
							expr: &zeroOrOneExpr{
								pos: position{line: 348, col: 18, offset: 8726},
								expr: &actionExpr{
									pos: position{line: 348, col: 19, offset: 8727},
									run: (*parser).callontop6,
									expr: &seqExpr{
										pos: position{line: 348, col: 19, offset: 8727},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 348, col: 19, offset: 8727},
												name: "_",
											},
											&labeledExpr{
												pos:   position{line: 348, col: 21, offset: 8729},
												label: "n",
												expr: &ruleRefExpr{
													pos:  position{line: 348, col: 23, offset: 8731},
													name: "integer",
												},
											},
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 348, col: 50, offset: 8758},
							label: "flush",
							expr: &zeroOrOneExpr{
								pos: position{line: 348, col: 56, offset: 8764},
								expr: &seqExpr{
									pos: position{line: 348, col: 57, offset: 8765},
									exprs: []interface{}{
										&ruleRefExpr{
											pos:  position{line: 348, col: 57, offset: 8765},
											name: "_",
										},
										&litMatcher{
											pos:        position{line: 348, col: 59, offset: 8767},
											val:        "-flush",
											ignoreCase: false,
										},
									},
								},
							},
						},
						&labeledExpr{
							pos:   position{line: 348, col: 70, offset: 8778},
							label: "list",
							expr: &zeroOrOneExpr{
								pos: position{line: 348, col: 75, offset: 8783},
								expr: &actionExpr{
									pos: position{line: 348, col: 76, offset: 8784},
									run: (*parser).callontop18,
									expr: &seqExpr{
										pos: position{line: 348, col: 76, offset: 8784},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 348, col: 76, offset: 8784},
												name: "_",
											},
											&labeledExpr{
												pos:   position{line: 348, col: 78, offset: 8786},
												label: "f",
												expr: &ruleRefExpr{
													pos:  position{line: 348, col: 80, offset: 8788},
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
			pos:  position{line: 352, col: 1, offset: 8877},
			expr: &actionExpr{
				pos: position{line: 353, col: 5, offset: 8894},
				run: (*parser).callonprocLimitArg1,
				expr: &seqExpr{
					pos: position{line: 353, col: 5, offset: 8894},
					exprs: []interface{}{
						&ruleRefExpr{
							pos:  position{line: 353, col: 5, offset: 8894},
							name: "_",
						},
						&litMatcher{
							pos:        position{line: 353, col: 7, offset: 8896},
							val:        "-limit",
							ignoreCase: false,
						},
						&ruleRefExpr{
							pos:  position{line: 353, col: 16, offset: 8905},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 353, col: 18, offset: 8907},
							label: "limit",
							expr: &ruleRefExpr{
								pos:  position{line: 353, col: 24, offset: 8913},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "cut",
			pos:  position{line: 355, col: 1, offset: 8944},
			expr: &actionExpr{
				pos: position{line: 356, col: 5, offset: 8952},
				run: (*parser).calloncut1,
				expr: &seqExpr{
					pos: position{line: 356, col: 5, offset: 8952},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 356, col: 5, offset: 8952},
							val:        "cut",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 356, col: 12, offset: 8959},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 356, col: 14, offset: 8961},
							label: "list",
							expr: &ruleRefExpr{
								pos:  position{line: 356, col: 19, offset: 8966},
								name: "fieldRefDotOnlyList",
							},
						},
					},
				},
			},
		},
		{
			name: "head",
			pos:  position{line: 357, col: 1, offset: 9020},
			expr: &choiceExpr{
				pos: position{line: 358, col: 5, offset: 9029},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 358, col: 5, offset: 9029},
						run: (*parser).callonhead2,
						expr: &seqExpr{
							pos: position{line: 358, col: 5, offset: 9029},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 358, col: 5, offset: 9029},
									val:        "head",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 358, col: 13, offset: 9037},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 358, col: 15, offset: 9039},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 358, col: 21, offset: 9045},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 359, col: 5, offset: 9093},
						run: (*parser).callonhead8,
						expr: &litMatcher{
							pos:        position{line: 359, col: 5, offset: 9093},
							val:        "head",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "tail",
			pos:  position{line: 360, col: 1, offset: 9133},
			expr: &choiceExpr{
				pos: position{line: 361, col: 5, offset: 9142},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 361, col: 5, offset: 9142},
						run: (*parser).callontail2,
						expr: &seqExpr{
							pos: position{line: 361, col: 5, offset: 9142},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 361, col: 5, offset: 9142},
									val:        "tail",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 361, col: 13, offset: 9150},
									name: "_",
								},
								&labeledExpr{
									pos:   position{line: 361, col: 15, offset: 9152},
									label: "count",
									expr: &ruleRefExpr{
										pos:  position{line: 361, col: 21, offset: 9158},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 362, col: 5, offset: 9206},
						run: (*parser).callontail8,
						expr: &litMatcher{
							pos:        position{line: 362, col: 5, offset: 9206},
							val:        "tail",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "filter",
			pos:  position{line: 364, col: 1, offset: 9247},
			expr: &actionExpr{
				pos: position{line: 365, col: 5, offset: 9258},
				run: (*parser).callonfilter1,
				expr: &seqExpr{
					pos: position{line: 365, col: 5, offset: 9258},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 365, col: 5, offset: 9258},
							val:        "filter",
							ignoreCase: true,
						},
						&ruleRefExpr{
							pos:  position{line: 365, col: 15, offset: 9268},
							name: "_",
						},
						&labeledExpr{
							pos:   position{line: 365, col: 17, offset: 9270},
							label: "expr",
							expr: &ruleRefExpr{
								pos:  position{line: 365, col: 22, offset: 9275},
								name: "searchExpr",
							},
						},
					},
				},
			},
		},
		{
			name: "uniq",
			pos:  position{line: 368, col: 1, offset: 9333},
			expr: &choiceExpr{
				pos: position{line: 369, col: 5, offset: 9342},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 369, col: 5, offset: 9342},
						run: (*parser).callonuniq2,
						expr: &seqExpr{
							pos: position{line: 369, col: 5, offset: 9342},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 369, col: 5, offset: 9342},
									val:        "uniq",
									ignoreCase: true,
								},
								&ruleRefExpr{
									pos:  position{line: 369, col: 13, offset: 9350},
									name: "_",
								},
								&litMatcher{
									pos:        position{line: 369, col: 15, offset: 9352},
									val:        "-c",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 372, col: 5, offset: 9406},
						run: (*parser).callonuniq7,
						expr: &litMatcher{
							pos:        position{line: 372, col: 5, offset: 9406},
							val:        "uniq",
							ignoreCase: true,
						},
					},
				},
			},
		},
		{
			name: "duration",
			pos:  position{line: 376, col: 1, offset: 9461},
			expr: &choiceExpr{
				pos: position{line: 377, col: 5, offset: 9474},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 377, col: 5, offset: 9474},
						name: "seconds",
					},
					&ruleRefExpr{
						pos:  position{line: 378, col: 5, offset: 9486},
						name: "minutes",
					},
					&ruleRefExpr{
						pos:  position{line: 379, col: 5, offset: 9498},
						name: "hours",
					},
					&seqExpr{
						pos: position{line: 380, col: 5, offset: 9508},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 380, col: 5, offset: 9508},
								name: "hours",
							},
							&ruleRefExpr{
								pos:  position{line: 380, col: 11, offset: 9514},
								name: "_",
							},
							&litMatcher{
								pos:        position{line: 380, col: 13, offset: 9516},
								val:        "and",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 380, col: 19, offset: 9522},
								name: "_",
							},
							&ruleRefExpr{
								pos:  position{line: 380, col: 21, offset: 9524},
								name: "minutes",
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 381, col: 5, offset: 9536},
						name: "days",
					},
					&ruleRefExpr{
						pos:  position{line: 382, col: 5, offset: 9545},
						name: "weeks",
					},
				},
			},
		},
		{
			name: "sec_abbrev",
			pos:  position{line: 384, col: 1, offset: 9552},
			expr: &choiceExpr{
				pos: position{line: 385, col: 5, offset: 9567},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 385, col: 5, offset: 9567},
						val:        "seconds",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 386, col: 5, offset: 9581},
						val:        "second",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 387, col: 5, offset: 9594},
						val:        "secs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 388, col: 5, offset: 9605},
						val:        "sec",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 389, col: 5, offset: 9615},
						val:        "s",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "min_abbrev",
			pos:  position{line: 391, col: 1, offset: 9620},
			expr: &choiceExpr{
				pos: position{line: 392, col: 5, offset: 9635},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 392, col: 5, offset: 9635},
						val:        "minutes",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 393, col: 5, offset: 9649},
						val:        "minute",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 394, col: 5, offset: 9662},
						val:        "mins",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 395, col: 5, offset: 9673},
						val:        "min",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 396, col: 5, offset: 9683},
						val:        "m",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "hour_abbrev",
			pos:  position{line: 398, col: 1, offset: 9688},
			expr: &choiceExpr{
				pos: position{line: 399, col: 5, offset: 9704},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 399, col: 5, offset: 9704},
						val:        "hours",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 400, col: 5, offset: 9716},
						val:        "hrs",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 401, col: 5, offset: 9726},
						val:        "hr",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 402, col: 5, offset: 9735},
						val:        "h",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 403, col: 5, offset: 9743},
						val:        "hour",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "day_abbrev",
			pos:  position{line: 405, col: 1, offset: 9751},
			expr: &choiceExpr{
				pos: position{line: 405, col: 14, offset: 9764},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 405, col: 14, offset: 9764},
						val:        "days",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 405, col: 21, offset: 9771},
						val:        "day",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 405, col: 27, offset: 9777},
						val:        "d",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "week_abbrev",
			pos:  position{line: 406, col: 1, offset: 9781},
			expr: &choiceExpr{
				pos: position{line: 406, col: 15, offset: 9795},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 406, col: 15, offset: 9795},
						val:        "weeks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 406, col: 23, offset: 9803},
						val:        "week",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 406, col: 30, offset: 9810},
						val:        "wks",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 406, col: 36, offset: 9816},
						val:        "wk",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 406, col: 41, offset: 9821},
						val:        "w",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name: "seconds",
			pos:  position{line: 408, col: 1, offset: 9826},
			expr: &choiceExpr{
				pos: position{line: 409, col: 5, offset: 9838},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 409, col: 5, offset: 9838},
						run: (*parser).callonseconds2,
						expr: &litMatcher{
							pos:        position{line: 409, col: 5, offset: 9838},
							val:        "second",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 410, col: 5, offset: 9883},
						run: (*parser).callonseconds4,
						expr: &seqExpr{
							pos: position{line: 410, col: 5, offset: 9883},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 410, col: 5, offset: 9883},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 410, col: 9, offset: 9887},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 410, col: 16, offset: 9894},
									expr: &ruleRefExpr{
										pos:  position{line: 410, col: 16, offset: 9894},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 410, col: 19, offset: 9897},
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
			pos:  position{line: 412, col: 1, offset: 9943},
			expr: &choiceExpr{
				pos: position{line: 413, col: 5, offset: 9955},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 413, col: 5, offset: 9955},
						run: (*parser).callonminutes2,
						expr: &litMatcher{
							pos:        position{line: 413, col: 5, offset: 9955},
							val:        "minute",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 414, col: 5, offset: 10001},
						run: (*parser).callonminutes4,
						expr: &seqExpr{
							pos: position{line: 414, col: 5, offset: 10001},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 414, col: 5, offset: 10001},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 414, col: 9, offset: 10005},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 414, col: 16, offset: 10012},
									expr: &ruleRefExpr{
										pos:  position{line: 414, col: 16, offset: 10012},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 414, col: 19, offset: 10015},
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
			pos:  position{line: 416, col: 1, offset: 10070},
			expr: &choiceExpr{
				pos: position{line: 417, col: 5, offset: 10080},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 417, col: 5, offset: 10080},
						run: (*parser).callonhours2,
						expr: &litMatcher{
							pos:        position{line: 417, col: 5, offset: 10080},
							val:        "hour",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 418, col: 5, offset: 10126},
						run: (*parser).callonhours4,
						expr: &seqExpr{
							pos: position{line: 418, col: 5, offset: 10126},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 418, col: 5, offset: 10126},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 418, col: 9, offset: 10130},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 418, col: 16, offset: 10137},
									expr: &ruleRefExpr{
										pos:  position{line: 418, col: 16, offset: 10137},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 418, col: 19, offset: 10140},
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
			pos:  position{line: 420, col: 1, offset: 10198},
			expr: &choiceExpr{
				pos: position{line: 421, col: 5, offset: 10207},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 421, col: 5, offset: 10207},
						run: (*parser).callondays2,
						expr: &litMatcher{
							pos:        position{line: 421, col: 5, offset: 10207},
							val:        "day",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 422, col: 5, offset: 10255},
						run: (*parser).callondays4,
						expr: &seqExpr{
							pos: position{line: 422, col: 5, offset: 10255},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 422, col: 5, offset: 10255},
									label: "num",
									expr: &ruleRefExpr{
										pos:  position{line: 422, col: 9, offset: 10259},
										name: "number",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 422, col: 16, offset: 10266},
									expr: &ruleRefExpr{
										pos:  position{line: 422, col: 16, offset: 10266},
										name: "_",
									},
								},
								&ruleRefExpr{
									pos:  position{line: 422, col: 19, offset: 10269},
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
			pos:  position{line: 424, col: 1, offset: 10329},
			expr: &actionExpr{
				pos: position{line: 425, col: 5, offset: 10339},
				run: (*parser).callonweeks1,
				expr: &seqExpr{
					pos: position{line: 425, col: 5, offset: 10339},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 425, col: 5, offset: 10339},
							label: "num",
							expr: &ruleRefExpr{
								pos:  position{line: 425, col: 9, offset: 10343},
								name: "number",
							},
						},
						&zeroOrOneExpr{
							pos: position{line: 425, col: 16, offset: 10350},
							expr: &ruleRefExpr{
								pos:  position{line: 425, col: 16, offset: 10350},
								name: "_",
							},
						},
						&ruleRefExpr{
							pos:  position{line: 425, col: 19, offset: 10353},
							name: "week_abbrev",
						},
					},
				},
			},
		},
		{
			name: "number",
			pos:  position{line: 427, col: 1, offset: 10416},
			expr: &ruleRefExpr{
				pos:  position{line: 427, col: 10, offset: 10425},
				name: "integer",
			},
		},
		{
			name: "addr",
			pos:  position{line: 431, col: 1, offset: 10463},
			expr: &actionExpr{
				pos: position{line: 432, col: 5, offset: 10472},
				run: (*parser).callonaddr1,
				expr: &labeledExpr{
					pos:   position{line: 432, col: 5, offset: 10472},
					label: "a",
					expr: &seqExpr{
						pos: position{line: 432, col: 8, offset: 10475},
						exprs: []interface{}{
							&ruleRefExpr{
								pos:  position{line: 432, col: 8, offset: 10475},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 432, col: 16, offset: 10483},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 432, col: 20, offset: 10487},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 432, col: 28, offset: 10495},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 432, col: 32, offset: 10499},
								name: "integer",
							},
							&litMatcher{
								pos:        position{line: 432, col: 40, offset: 10507},
								val:        ".",
								ignoreCase: false,
							},
							&ruleRefExpr{
								pos:  position{line: 432, col: 44, offset: 10511},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "port",
			pos:  position{line: 434, col: 1, offset: 10552},
			expr: &actionExpr{
				pos: position{line: 435, col: 5, offset: 10561},
				run: (*parser).callonport1,
				expr: &seqExpr{
					pos: position{line: 435, col: 5, offset: 10561},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 435, col: 5, offset: 10561},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 435, col: 9, offset: 10565},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 435, col: 11, offset: 10567},
								name: "sinteger",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6addr",
			pos:  position{line: 439, col: 1, offset: 10726},
			expr: &choiceExpr{
				pos: position{line: 440, col: 5, offset: 10738},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 440, col: 5, offset: 10738},
						run: (*parser).callonip6addr2,
						expr: &seqExpr{
							pos: position{line: 440, col: 5, offset: 10738},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 440, col: 5, offset: 10738},
									label: "a",
									expr: &oneOrMoreExpr{
										pos: position{line: 440, col: 7, offset: 10740},
										expr: &ruleRefExpr{
											pos:  position{line: 440, col: 8, offset: 10741},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 440, col: 20, offset: 10753},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 440, col: 22, offset: 10755},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 443, col: 5, offset: 10819},
						run: (*parser).callonip6addr9,
						expr: &seqExpr{
							pos: position{line: 443, col: 5, offset: 10819},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 443, col: 5, offset: 10819},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 443, col: 7, offset: 10821},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 443, col: 11, offset: 10825},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 443, col: 13, offset: 10827},
										expr: &ruleRefExpr{
											pos:  position{line: 443, col: 14, offset: 10828},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 443, col: 25, offset: 10839},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 443, col: 30, offset: 10844},
									label: "d",
									expr: &zeroOrMoreExpr{
										pos: position{line: 443, col: 32, offset: 10846},
										expr: &ruleRefExpr{
											pos:  position{line: 443, col: 33, offset: 10847},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 443, col: 45, offset: 10859},
									label: "e",
									expr: &ruleRefExpr{
										pos:  position{line: 443, col: 47, offset: 10861},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 446, col: 5, offset: 10960},
						run: (*parser).callonip6addr22,
						expr: &seqExpr{
							pos: position{line: 446, col: 5, offset: 10960},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 446, col: 5, offset: 10960},
									val:        "::",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 446, col: 10, offset: 10965},
									label: "a",
									expr: &zeroOrMoreExpr{
										pos: position{line: 446, col: 12, offset: 10967},
										expr: &ruleRefExpr{
											pos:  position{line: 446, col: 13, offset: 10968},
											name: "h_prepend",
										},
									},
								},
								&labeledExpr{
									pos:   position{line: 446, col: 25, offset: 10980},
									label: "b",
									expr: &ruleRefExpr{
										pos:  position{line: 446, col: 27, offset: 10982},
										name: "ip6tail",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 449, col: 5, offset: 11053},
						run: (*parser).callonip6addr30,
						expr: &seqExpr{
							pos: position{line: 449, col: 5, offset: 11053},
							exprs: []interface{}{
								&labeledExpr{
									pos:   position{line: 449, col: 5, offset: 11053},
									label: "a",
									expr: &ruleRefExpr{
										pos:  position{line: 449, col: 7, offset: 11055},
										name: "h16",
									},
								},
								&labeledExpr{
									pos:   position{line: 449, col: 11, offset: 11059},
									label: "b",
									expr: &zeroOrMoreExpr{
										pos: position{line: 449, col: 13, offset: 11061},
										expr: &ruleRefExpr{
											pos:  position{line: 449, col: 14, offset: 11062},
											name: "h_append",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 449, col: 25, offset: 11073},
									val:        "::",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 452, col: 5, offset: 11141},
						run: (*parser).callonip6addr38,
						expr: &litMatcher{
							pos:        position{line: 452, col: 5, offset: 11141},
							val:        "::",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "ip6tail",
			pos:  position{line: 456, col: 1, offset: 11178},
			expr: &choiceExpr{
				pos: position{line: 457, col: 5, offset: 11190},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 457, col: 5, offset: 11190},
						name: "addr",
					},
					&ruleRefExpr{
						pos:  position{line: 458, col: 5, offset: 11199},
						name: "h16",
					},
				},
			},
		},
		{
			name: "h_append",
			pos:  position{line: 460, col: 1, offset: 11204},
			expr: &actionExpr{
				pos: position{line: 460, col: 12, offset: 11215},
				run: (*parser).callonh_append1,
				expr: &seqExpr{
					pos: position{line: 460, col: 12, offset: 11215},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 460, col: 12, offset: 11215},
							val:        ":",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 460, col: 16, offset: 11219},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 460, col: 18, offset: 11221},
								name: "h16",
							},
						},
					},
				},
			},
		},
		{
			name: "h_prepend",
			pos:  position{line: 461, col: 1, offset: 11258},
			expr: &actionExpr{
				pos: position{line: 461, col: 13, offset: 11270},
				run: (*parser).callonh_prepend1,
				expr: &seqExpr{
					pos: position{line: 461, col: 13, offset: 11270},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 461, col: 13, offset: 11270},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 461, col: 15, offset: 11272},
								name: "h16",
							},
						},
						&litMatcher{
							pos:        position{line: 461, col: 19, offset: 11276},
							val:        ":",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "sub_addr",
			pos:  position{line: 463, col: 1, offset: 11314},
			expr: &choiceExpr{
				pos: position{line: 464, col: 5, offset: 11327},
				alternatives: []interface{}{
					&ruleRefExpr{
						pos:  position{line: 464, col: 5, offset: 11327},
						name: "addr",
					},
					&actionExpr{
						pos: position{line: 465, col: 5, offset: 11336},
						run: (*parser).callonsub_addr3,
						expr: &labeledExpr{
							pos:   position{line: 465, col: 5, offset: 11336},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 465, col: 8, offset: 11339},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 465, col: 8, offset: 11339},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 465, col: 16, offset: 11347},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 465, col: 20, offset: 11351},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 465, col: 28, offset: 11359},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 465, col: 32, offset: 11363},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 466, col: 5, offset: 11415},
						run: (*parser).callonsub_addr11,
						expr: &labeledExpr{
							pos:   position{line: 466, col: 5, offset: 11415},
							label: "a",
							expr: &seqExpr{
								pos: position{line: 466, col: 8, offset: 11418},
								exprs: []interface{}{
									&ruleRefExpr{
										pos:  position{line: 466, col: 8, offset: 11418},
										name: "integer",
									},
									&litMatcher{
										pos:        position{line: 466, col: 16, offset: 11426},
										val:        ".",
										ignoreCase: false,
									},
									&ruleRefExpr{
										pos:  position{line: 466, col: 20, offset: 11430},
										name: "integer",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 467, col: 5, offset: 11484},
						run: (*parser).callonsub_addr17,
						expr: &labeledExpr{
							pos:   position{line: 467, col: 5, offset: 11484},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 467, col: 7, offset: 11486},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "subnet",
			pos:  position{line: 469, col: 1, offset: 11537},
			expr: &actionExpr{
				pos: position{line: 470, col: 5, offset: 11548},
				run: (*parser).callonsubnet1,
				expr: &seqExpr{
					pos: position{line: 470, col: 5, offset: 11548},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 470, col: 5, offset: 11548},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 470, col: 7, offset: 11550},
								name: "sub_addr",
							},
						},
						&litMatcher{
							pos:        position{line: 470, col: 16, offset: 11559},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 470, col: 20, offset: 11563},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 470, col: 22, offset: 11565},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "ip6subnet",
			pos:  position{line: 474, col: 1, offset: 11641},
			expr: &actionExpr{
				pos: position{line: 475, col: 5, offset: 11655},
				run: (*parser).callonip6subnet1,
				expr: &seqExpr{
					pos: position{line: 475, col: 5, offset: 11655},
					exprs: []interface{}{
						&labeledExpr{
							pos:   position{line: 475, col: 5, offset: 11655},
							label: "a",
							expr: &ruleRefExpr{
								pos:  position{line: 475, col: 7, offset: 11657},
								name: "ip6addr",
							},
						},
						&litMatcher{
							pos:        position{line: 475, col: 15, offset: 11665},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 475, col: 19, offset: 11669},
							label: "m",
							expr: &ruleRefExpr{
								pos:  position{line: 475, col: 21, offset: 11671},
								name: "integer",
							},
						},
					},
				},
			},
		},
		{
			name: "integer",
			pos:  position{line: 479, col: 1, offset: 11737},
			expr: &actionExpr{
				pos: position{line: 480, col: 5, offset: 11749},
				run: (*parser).calloninteger1,
				expr: &labeledExpr{
					pos:   position{line: 480, col: 5, offset: 11749},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 480, col: 7, offset: 11751},
						name: "sinteger",
					},
				},
			},
		},
		{
			name: "sinteger",
			pos:  position{line: 484, col: 1, offset: 11795},
			expr: &actionExpr{
				pos: position{line: 485, col: 5, offset: 11808},
				run: (*parser).callonsinteger1,
				expr: &labeledExpr{
					pos:   position{line: 485, col: 5, offset: 11808},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 485, col: 11, offset: 11814},
						expr: &charClassMatcher{
							pos:        position{line: 485, col: 11, offset: 11814},
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
			pos:  position{line: 489, col: 1, offset: 11859},
			expr: &actionExpr{
				pos: position{line: 490, col: 5, offset: 11870},
				run: (*parser).callondouble1,
				expr: &labeledExpr{
					pos:   position{line: 490, col: 5, offset: 11870},
					label: "s",
					expr: &ruleRefExpr{
						pos:  position{line: 490, col: 7, offset: 11872},
						name: "sdouble",
					},
				},
			},
		},
		{
			name: "sdouble",
			pos:  position{line: 494, col: 1, offset: 11919},
			expr: &choiceExpr{
				pos: position{line: 495, col: 5, offset: 11931},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 495, col: 5, offset: 11931},
						run: (*parser).callonsdouble2,
						expr: &seqExpr{
							pos: position{line: 495, col: 5, offset: 11931},
							exprs: []interface{}{
								&oneOrMoreExpr{
									pos: position{line: 495, col: 5, offset: 11931},
									expr: &ruleRefExpr{
										pos:  position{line: 495, col: 5, offset: 11931},
										name: "doubleInteger",
									},
								},
								&litMatcher{
									pos:        position{line: 495, col: 20, offset: 11946},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 495, col: 24, offset: 11950},
									expr: &ruleRefExpr{
										pos:  position{line: 495, col: 24, offset: 11950},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 495, col: 37, offset: 11963},
									expr: &ruleRefExpr{
										pos:  position{line: 495, col: 37, offset: 11963},
										name: "exponentPart",
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 498, col: 5, offset: 12022},
						run: (*parser).callonsdouble11,
						expr: &seqExpr{
							pos: position{line: 498, col: 5, offset: 12022},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 498, col: 5, offset: 12022},
									val:        ".",
									ignoreCase: false,
								},
								&oneOrMoreExpr{
									pos: position{line: 498, col: 9, offset: 12026},
									expr: &ruleRefExpr{
										pos:  position{line: 498, col: 9, offset: 12026},
										name: "doubleDigit",
									},
								},
								&zeroOrOneExpr{
									pos: position{line: 498, col: 22, offset: 12039},
									expr: &ruleRefExpr{
										pos:  position{line: 498, col: 22, offset: 12039},
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
			pos:  position{line: 502, col: 1, offset: 12095},
			expr: &choiceExpr{
				pos: position{line: 503, col: 5, offset: 12113},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 503, col: 5, offset: 12113},
						val:        "0",
						ignoreCase: false,
					},
					&seqExpr{
						pos: position{line: 504, col: 5, offset: 12121},
						exprs: []interface{}{
							&charClassMatcher{
								pos:        position{line: 504, col: 5, offset: 12121},
								val:        "[1-9]",
								ranges:     []rune{'1', '9'},
								ignoreCase: false,
								inverted:   false,
							},
							&zeroOrMoreExpr{
								pos: position{line: 504, col: 11, offset: 12127},
								expr: &charClassMatcher{
									pos:        position{line: 504, col: 11, offset: 12127},
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
			pos:  position{line: 506, col: 1, offset: 12135},
			expr: &charClassMatcher{
				pos:        position{line: 506, col: 15, offset: 12149},
				val:        "[0-9]",
				ranges:     []rune{'0', '9'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "signedInteger",
			pos:  position{line: 508, col: 1, offset: 12156},
			expr: &seqExpr{
				pos: position{line: 508, col: 17, offset: 12172},
				exprs: []interface{}{
					&zeroOrOneExpr{
						pos: position{line: 508, col: 17, offset: 12172},
						expr: &charClassMatcher{
							pos:        position{line: 508, col: 17, offset: 12172},
							val:        "[+-]",
							chars:      []rune{'+', '-'},
							ignoreCase: false,
							inverted:   false,
						},
					},
					&oneOrMoreExpr{
						pos: position{line: 508, col: 23, offset: 12178},
						expr: &ruleRefExpr{
							pos:  position{line: 508, col: 23, offset: 12178},
							name: "doubleDigit",
						},
					},
				},
			},
		},
		{
			name: "exponentPart",
			pos:  position{line: 510, col: 1, offset: 12192},
			expr: &seqExpr{
				pos: position{line: 510, col: 16, offset: 12207},
				exprs: []interface{}{
					&litMatcher{
						pos:        position{line: 510, col: 16, offset: 12207},
						val:        "e",
						ignoreCase: true,
					},
					&ruleRefExpr{
						pos:  position{line: 510, col: 21, offset: 12212},
						name: "signedInteger",
					},
				},
			},
		},
		{
			name: "h16",
			pos:  position{line: 512, col: 1, offset: 12227},
			expr: &actionExpr{
				pos: position{line: 512, col: 7, offset: 12233},
				run: (*parser).callonh161,
				expr: &labeledExpr{
					pos:   position{line: 512, col: 7, offset: 12233},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 512, col: 13, offset: 12239},
						expr: &ruleRefExpr{
							pos:  position{line: 512, col: 13, offset: 12239},
							name: "hexdigit",
						},
					},
				},
			},
		},
		{
			name: "hexdigit",
			pos:  position{line: 514, col: 1, offset: 12281},
			expr: &charClassMatcher{
				pos:        position{line: 514, col: 12, offset: 12292},
				val:        "[0-9a-fA-F]",
				ranges:     []rune{'0', '9', 'a', 'f', 'A', 'F'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "searchWord",
			pos:  position{line: 516, col: 1, offset: 12305},
			expr: &actionExpr{
				pos: position{line: 517, col: 5, offset: 12320},
				run: (*parser).callonsearchWord1,
				expr: &labeledExpr{
					pos:   position{line: 517, col: 5, offset: 12320},
					label: "chars",
					expr: &oneOrMoreExpr{
						pos: position{line: 517, col: 11, offset: 12326},
						expr: &ruleRefExpr{
							pos:  position{line: 517, col: 11, offset: 12326},
							name: "searchWordPart",
						},
					},
				},
			},
		},
		{
			name: "searchWordPart",
			pos:  position{line: 519, col: 1, offset: 12376},
			expr: &choiceExpr{
				pos: position{line: 520, col: 5, offset: 12395},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 520, col: 5, offset: 12395},
						run: (*parser).callonsearchWordPart2,
						expr: &seqExpr{
							pos: position{line: 520, col: 5, offset: 12395},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 520, col: 5, offset: 12395},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 520, col: 10, offset: 12400},
									label: "s",
									expr: &choiceExpr{
										pos: position{line: 520, col: 13, offset: 12403},
										alternatives: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 520, col: 13, offset: 12403},
												name: "escapeSequence",
											},
											&ruleRefExpr{
												pos:  position{line: 520, col: 30, offset: 12420},
												name: "searchEscape",
											},
										},
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 521, col: 5, offset: 12457},
						run: (*parser).callonsearchWordPart9,
						expr: &seqExpr{
							pos: position{line: 521, col: 5, offset: 12457},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 521, col: 5, offset: 12457},
									expr: &choiceExpr{
										pos: position{line: 521, col: 7, offset: 12459},
										alternatives: []interface{}{
											&charClassMatcher{
												pos:        position{line: 521, col: 7, offset: 12459},
												val:        "[\\x00-\\x1F\\x5C(),!><=\\x22|\\x27;]",
												chars:      []rune{'\\', '(', ')', ',', '!', '>', '<', '=', '"', '|', '\'', ';'},
												ranges:     []rune{'\x00', '\x1f'},
												ignoreCase: false,
												inverted:   false,
											},
											&ruleRefExpr{
												pos:  position{line: 521, col: 42, offset: 12494},
												name: "ws",
											},
										},
									},
								},
								&anyMatcher{
									line: 521, col: 46, offset: 12498,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "quotedString",
			pos:  position{line: 523, col: 1, offset: 12532},
			expr: &choiceExpr{
				pos: position{line: 524, col: 5, offset: 12549},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 524, col: 5, offset: 12549},
						run: (*parser).callonquotedString2,
						expr: &seqExpr{
							pos: position{line: 524, col: 5, offset: 12549},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 524, col: 5, offset: 12549},
									val:        "\"",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 524, col: 9, offset: 12553},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 524, col: 11, offset: 12555},
										expr: &ruleRefExpr{
											pos:  position{line: 524, col: 11, offset: 12555},
											name: "doubleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 524, col: 29, offset: 12573},
									val:        "\"",
									ignoreCase: false,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 525, col: 5, offset: 12610},
						run: (*parser).callonquotedString9,
						expr: &seqExpr{
							pos: position{line: 525, col: 5, offset: 12610},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 525, col: 5, offset: 12610},
									val:        "'",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 525, col: 9, offset: 12614},
									label: "v",
									expr: &zeroOrMoreExpr{
										pos: position{line: 525, col: 11, offset: 12616},
										expr: &ruleRefExpr{
											pos:  position{line: 525, col: 11, offset: 12616},
											name: "singleQuotedChar",
										},
									},
								},
								&litMatcher{
									pos:        position{line: 525, col: 29, offset: 12634},
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
			pos:  position{line: 527, col: 1, offset: 12668},
			expr: &choiceExpr{
				pos: position{line: 528, col: 5, offset: 12689},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 528, col: 5, offset: 12689},
						run: (*parser).callondoubleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 528, col: 5, offset: 12689},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 528, col: 5, offset: 12689},
									expr: &choiceExpr{
										pos: position{line: 528, col: 7, offset: 12691},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 528, col: 7, offset: 12691},
												val:        "\"",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 528, col: 13, offset: 12697},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 528, col: 26, offset: 12710,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 529, col: 5, offset: 12747},
						run: (*parser).callondoubleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 529, col: 5, offset: 12747},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 529, col: 5, offset: 12747},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 529, col: 10, offset: 12752},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 529, col: 12, offset: 12754},
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
			pos:  position{line: 531, col: 1, offset: 12788},
			expr: &choiceExpr{
				pos: position{line: 532, col: 5, offset: 12809},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 532, col: 5, offset: 12809},
						run: (*parser).callonsingleQuotedChar2,
						expr: &seqExpr{
							pos: position{line: 532, col: 5, offset: 12809},
							exprs: []interface{}{
								&notExpr{
									pos: position{line: 532, col: 5, offset: 12809},
									expr: &choiceExpr{
										pos: position{line: 532, col: 7, offset: 12811},
										alternatives: []interface{}{
											&litMatcher{
												pos:        position{line: 532, col: 7, offset: 12811},
												val:        "'",
												ignoreCase: false,
											},
											&ruleRefExpr{
												pos:  position{line: 532, col: 13, offset: 12817},
												name: "escapedChar",
											},
										},
									},
								},
								&anyMatcher{
									line: 532, col: 26, offset: 12830,
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 533, col: 5, offset: 12867},
						run: (*parser).callonsingleQuotedChar9,
						expr: &seqExpr{
							pos: position{line: 533, col: 5, offset: 12867},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 533, col: 5, offset: 12867},
									val:        "\\",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 533, col: 10, offset: 12872},
									label: "s",
									expr: &ruleRefExpr{
										pos:  position{line: 533, col: 12, offset: 12874},
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
			pos:  position{line: 535, col: 1, offset: 12908},
			expr: &choiceExpr{
				pos: position{line: 536, col: 5, offset: 12927},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 536, col: 5, offset: 12927},
						run: (*parser).callonescapeSequence2,
						expr: &seqExpr{
							pos: position{line: 536, col: 5, offset: 12927},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 536, col: 5, offset: 12927},
									val:        "x",
									ignoreCase: false,
								},
								&ruleRefExpr{
									pos:  position{line: 536, col: 9, offset: 12931},
									name: "hexdigit",
								},
								&ruleRefExpr{
									pos:  position{line: 536, col: 18, offset: 12940},
									name: "hexdigit",
								},
							},
						},
					},
					&ruleRefExpr{
						pos:  position{line: 537, col: 5, offset: 12991},
						name: "singleCharEscape",
					},
					&ruleRefExpr{
						pos:  position{line: 538, col: 5, offset: 13012},
						name: "unicodeEscape",
					},
				},
			},
		},
		{
			name: "singleCharEscape",
			pos:  position{line: 540, col: 1, offset: 13027},
			expr: &choiceExpr{
				pos: position{line: 541, col: 5, offset: 13048},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 541, col: 5, offset: 13048},
						val:        "'",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 542, col: 5, offset: 13056},
						val:        "\"",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 543, col: 5, offset: 13064},
						val:        "\\",
						ignoreCase: false,
					},
					&actionExpr{
						pos: position{line: 544, col: 5, offset: 13073},
						run: (*parser).callonsingleCharEscape5,
						expr: &litMatcher{
							pos:        position{line: 544, col: 5, offset: 13073},
							val:        "b",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 545, col: 5, offset: 13102},
						run: (*parser).callonsingleCharEscape7,
						expr: &litMatcher{
							pos:        position{line: 545, col: 5, offset: 13102},
							val:        "f",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 546, col: 5, offset: 13131},
						run: (*parser).callonsingleCharEscape9,
						expr: &litMatcher{
							pos:        position{line: 546, col: 5, offset: 13131},
							val:        "n",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 547, col: 5, offset: 13160},
						run: (*parser).callonsingleCharEscape11,
						expr: &litMatcher{
							pos:        position{line: 547, col: 5, offset: 13160},
							val:        "r",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 548, col: 5, offset: 13189},
						run: (*parser).callonsingleCharEscape13,
						expr: &litMatcher{
							pos:        position{line: 548, col: 5, offset: 13189},
							val:        "t",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 549, col: 5, offset: 13218},
						run: (*parser).callonsingleCharEscape15,
						expr: &litMatcher{
							pos:        position{line: 549, col: 5, offset: 13218},
							val:        "v",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "searchEscape",
			pos:  position{line: 551, col: 1, offset: 13244},
			expr: &choiceExpr{
				pos: position{line: 552, col: 5, offset: 13261},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 552, col: 5, offset: 13261},
						run: (*parser).callonsearchEscape2,
						expr: &litMatcher{
							pos:        position{line: 552, col: 5, offset: 13261},
							val:        "=",
							ignoreCase: false,
						},
					},
					&actionExpr{
						pos: position{line: 553, col: 5, offset: 13289},
						run: (*parser).callonsearchEscape4,
						expr: &litMatcher{
							pos:        position{line: 553, col: 5, offset: 13289},
							val:        "*",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "unicodeEscape",
			pos:  position{line: 555, col: 1, offset: 13316},
			expr: &choiceExpr{
				pos: position{line: 556, col: 5, offset: 13334},
				alternatives: []interface{}{
					&actionExpr{
						pos: position{line: 556, col: 5, offset: 13334},
						run: (*parser).callonunicodeEscape2,
						expr: &seqExpr{
							pos: position{line: 556, col: 5, offset: 13334},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 556, col: 5, offset: 13334},
									val:        "u",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 556, col: 9, offset: 13338},
									label: "chars",
									expr: &seqExpr{
										pos: position{line: 556, col: 16, offset: 13345},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 556, col: 16, offset: 13345},
												name: "hexdigit",
											},
											&ruleRefExpr{
												pos:  position{line: 556, col: 25, offset: 13354},
												name: "hexdigit",
											},
											&ruleRefExpr{
												pos:  position{line: 556, col: 34, offset: 13363},
												name: "hexdigit",
											},
											&ruleRefExpr{
												pos:  position{line: 556, col: 43, offset: 13372},
												name: "hexdigit",
											},
										},
									},
								},
							},
						},
					},
					&actionExpr{
						pos: position{line: 559, col: 5, offset: 13435},
						run: (*parser).callonunicodeEscape11,
						expr: &seqExpr{
							pos: position{line: 559, col: 5, offset: 13435},
							exprs: []interface{}{
								&litMatcher{
									pos:        position{line: 559, col: 5, offset: 13435},
									val:        "u",
									ignoreCase: false,
								},
								&litMatcher{
									pos:        position{line: 559, col: 9, offset: 13439},
									val:        "{",
									ignoreCase: false,
								},
								&labeledExpr{
									pos:   position{line: 559, col: 13, offset: 13443},
									label: "chars",
									expr: &seqExpr{
										pos: position{line: 559, col: 20, offset: 13450},
										exprs: []interface{}{
											&ruleRefExpr{
												pos:  position{line: 559, col: 20, offset: 13450},
												name: "hexdigit",
											},
											&zeroOrOneExpr{
												pos: position{line: 559, col: 29, offset: 13459},
												expr: &ruleRefExpr{
													pos:  position{line: 559, col: 29, offset: 13459},
													name: "hexdigit",
												},
											},
											&zeroOrOneExpr{
												pos: position{line: 559, col: 39, offset: 13469},
												expr: &ruleRefExpr{
													pos:  position{line: 559, col: 39, offset: 13469},
													name: "hexdigit",
												},
											},
											&zeroOrOneExpr{
												pos: position{line: 559, col: 49, offset: 13479},
												expr: &ruleRefExpr{
													pos:  position{line: 559, col: 49, offset: 13479},
													name: "hexdigit",
												},
											},
											&zeroOrOneExpr{
												pos: position{line: 559, col: 59, offset: 13489},
												expr: &ruleRefExpr{
													pos:  position{line: 559, col: 59, offset: 13489},
													name: "hexdigit",
												},
											},
											&zeroOrOneExpr{
												pos: position{line: 559, col: 69, offset: 13499},
												expr: &ruleRefExpr{
													pos:  position{line: 559, col: 69, offset: 13499},
													name: "hexdigit",
												},
											},
										},
									},
								},
								&litMatcher{
									pos:        position{line: 559, col: 80, offset: 13510},
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
			pos:  position{line: 563, col: 1, offset: 13564},
			expr: &actionExpr{
				pos: position{line: 564, col: 5, offset: 13577},
				run: (*parser).callonreString1,
				expr: &seqExpr{
					pos: position{line: 564, col: 5, offset: 13577},
					exprs: []interface{}{
						&litMatcher{
							pos:        position{line: 564, col: 5, offset: 13577},
							val:        "/",
							ignoreCase: false,
						},
						&labeledExpr{
							pos:   position{line: 564, col: 9, offset: 13581},
							label: "v",
							expr: &ruleRefExpr{
								pos:  position{line: 564, col: 11, offset: 13583},
								name: "reBody",
							},
						},
						&litMatcher{
							pos:        position{line: 564, col: 18, offset: 13590},
							val:        "/",
							ignoreCase: false,
						},
					},
				},
			},
		},
		{
			name: "reBody",
			pos:  position{line: 566, col: 1, offset: 13613},
			expr: &actionExpr{
				pos: position{line: 567, col: 5, offset: 13624},
				run: (*parser).callonreBody1,
				expr: &oneOrMoreExpr{
					pos: position{line: 567, col: 5, offset: 13624},
					expr: &choiceExpr{
						pos: position{line: 567, col: 6, offset: 13625},
						alternatives: []interface{}{
							&charClassMatcher{
								pos:        position{line: 567, col: 6, offset: 13625},
								val:        "[^/\\\\]",
								chars:      []rune{'/', '\\'},
								ignoreCase: false,
								inverted:   true,
							},
							&litMatcher{
								pos:        position{line: 567, col: 13, offset: 13632},
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
			pos:  position{line: 569, col: 1, offset: 13672},
			expr: &charClassMatcher{
				pos:        position{line: 570, col: 5, offset: 13688},
				val:        "[\\x00-\\x1f\\\\]",
				chars:      []rune{'\\'},
				ranges:     []rune{'\x00', '\x1f'},
				ignoreCase: false,
				inverted:   false,
			},
		},
		{
			name: "ws",
			pos:  position{line: 572, col: 1, offset: 13703},
			expr: &choiceExpr{
				pos: position{line: 573, col: 5, offset: 13710},
				alternatives: []interface{}{
					&litMatcher{
						pos:        position{line: 573, col: 5, offset: 13710},
						val:        "\t",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 574, col: 5, offset: 13719},
						val:        "\v",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 575, col: 5, offset: 13728},
						val:        "\f",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 576, col: 5, offset: 13737},
						val:        " ",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 577, col: 5, offset: 13745},
						val:        "\u00a0",
						ignoreCase: false,
					},
					&litMatcher{
						pos:        position{line: 578, col: 5, offset: 13758},
						val:        "\ufeff",
						ignoreCase: false,
					},
				},
			},
		},
		{
			name:        "_",
			displayName: "\"whitespace\"",
			pos:         position{line: 580, col: 1, offset: 13768},
			expr: &oneOrMoreExpr{
				pos: position{line: 580, col: 18, offset: 13785},
				expr: &ruleRefExpr{
					pos:  position{line: 580, col: 18, offset: 13785},
					name: "ws",
				},
			},
		},
		{
			name: "EOF",
			pos:  position{line: 582, col: 1, offset: 13790},
			expr: &notExpr{
				pos: position{line: 582, col: 7, offset: 13796},
				expr: &anyMatcher{
					line: 582, col: 8, offset: 13797,
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

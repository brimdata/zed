package ast_test

import (
	"testing"

	"github.com/mccanne/zq/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// boom ast 'a=1 foo (b=2 or c=3) | cut d, e | every day sum(j) as j, min(l) as m by n, o | sort -r p | head 1' | jq .proc | pbcopy
var testCopyJSON = []byte(`
{
  "op": "SequentialProc",
  "procs": [
    {
      "op": "FilterProc",
      "filter": {
        "op": "LogicalAnd",
        "left": {
          "op": "LogicalAnd",
          "left": {
            "op": "CompareField",
            "comparator": "eql",
            "field": {
              "op": "FieldRead",
              "field": "a"
            },
            "value": {
              "type": "int",
              "value": "1"
            }
          },
          "right": {
            "op": "LogicalOr",
            "left": {
              "op": "CompareAny",
              "comparator": "search",
              "recursive": true,
              "value": {
                "type": "string",
                "value": "foo"
              }
            },
            "right": {
              "op": "CompareAny",
              "comparator": "searchin",
              "recursive": true,
              "value": {
                "type": "string",
                "value": "foo"
              }
            }
          }
        },
        "right": {
          "op": "LogicalOr",
          "left": {
            "op": "CompareField",
            "comparator": "eql",
            "field": {
              "op": "FieldRead",
              "field": "b"
            },
            "value": {
              "type": "int",
              "value": "2"
            }
          },
          "right": {
            "op": "CompareField",
            "comparator": "eql",
            "field": {
              "op": "FieldRead",
              "field": "c"
            },
            "value": {
              "type": "int",
              "value": "3"
            }
          }
        }
      }
    },
    {
      "op": "CutProc",
      "fields": [
        {
          "op": "FieldRead",
          "field": "d"
        },
        {
          "op": "FieldRead",
          "field": "e"
        }
      ]
    },
    {
      "op": "GroupByProc",
      "duration": {
        "seconds": 86400
      },
      "update_interval": {
        "seconds": 0
      },
      "keys": [
        {
          "op": "FieldRead",
          "field": "n"
        },
        {
          "op": "FieldRead",
          "field": "o"
        }
      ],
      "reducers": [
        {
          "op": "Sum",
          "var": "j",
          "field": {
            "op": "FieldRead",
            "field": "j"
          }
        },
        {
          "op": "Min",
          "var": "m",
          "field": {
            "op": "FieldRead",
            "field": "l"
          }
        }
      ]
    },
    {
      "op": "SortProc",
      "fields": [
        {
          "op": "FieldRead",
          "field": "p"
        }
      ],
      "sortdir": -1
    },
    {
      "op": "HeadProc",
      "count": 1
    }
  ]
}`)

// boom ast 'a=1 foo (b=2 or d=4) | cut z, e | every hour sum(k) as j, min(l) as m by n, p | sort   p | head 2' | jq .proc | pbcopy
var testCopyJSONExpected = []byte(`
{
  "op": "SequentialProc",
  "procs": [
    {
      "op": "FilterProc",
      "filter": {
        "op": "LogicalAnd",
        "left": {
          "op": "LogicalAnd",
          "left": {
            "op": "CompareField",
            "comparator": "eql",
            "field": {
              "op": "FieldRead",
              "field": "a"
            },
            "value": {
              "type": "int",
              "value": "1"
            }
          },
          "right": {
            "op": "LogicalOr",
            "left": {
              "op": "CompareAny",
              "comparator": "search",
              "recursive": true,
              "value": {
                "type": "string",
                "value": "foo"
              }
            },
            "right": {
              "op": "CompareAny",
              "comparator": "searchin",
              "recursive": true,
              "value": {
                "type": "string",
                "value": "foo"
              }
            }
          }
        },
        "right": {
          "op": "LogicalOr",
          "left": {
            "op": "CompareField",
            "comparator": "eql",
            "field": {
              "op": "FieldRead",
              "field": "b"
            },
            "value": {
              "type": "int",
              "value": "2"
            }
          },
          "right": {
            "op": "CompareField",
            "comparator": "eql",
            "field": {
              "op": "FieldRead",
              "field": "d"
            },
            "value": {
              "type": "int",
              "value": "4"
            }
          }
        }
      }
    },
    {
      "op": "CutProc",
      "fields": [
        {
          "op": "FieldRead",
          "field": "z"
        },
        {
          "op": "FieldRead",
          "field": "e"
        }
      ]
    },
    {
      "op": "GroupByProc",
      "duration": {
        "seconds": 3600
      },
      "update_interval": {
        "seconds": 0
      },
      "keys": [
        {
          "op": "FieldRead",
          "field": "n"
        },
        {
          "op": "FieldRead",
          "field": "p"
        }
      ],
      "reducers": [
        {
          "op": "Sum",
          "var": "j",
          "field": {
            "op": "FieldRead",
            "field": "k"
          }
        },
        {
          "op": "Min",
          "var": "m",
          "field": {
            "op": "FieldRead",
            "field": "l"
          }
        }
      ]
    },
    {
      "op": "SortProc",
      "fields": [
        {
          "op": "FieldRead",
          "field": "p"
        }
      ],
      "sortdir": 1
    },
    {
      "op": "HeadProc",
      "count": 2
    }
  ]
}
`)

func TestCopyAST(t *testing.T) {
	initialProc, err := ast.UnpackProc(nil, testCopyJSON)
	require.NoError(t, err)

	// modify AST (and check result as indended)
	copy := initialProc.Copy()
	copy.(*ast.SequentialProc).Procs[0].(*ast.FilterProc).Filter.(*ast.LogicalAnd).Right.(*ast.LogicalOr).Right.(*ast.CompareField).Field.(*ast.FieldRead).Field = "d"
	copy.(*ast.SequentialProc).Procs[0].(*ast.FilterProc).Filter.(*ast.LogicalAnd).Right.(*ast.LogicalOr).Right.(*ast.CompareField).Value.Value = "4"
	copy.(*ast.SequentialProc).Procs[1].(*ast.CutProc).Fields[0].(*ast.FieldRead).Field = "z"
	copy.(*ast.SequentialProc).Procs[2].(*ast.GroupByProc).Duration.Seconds = 3600
	copy.(*ast.SequentialProc).Procs[2].(*ast.GroupByProc).Reducers[0].Field.(*ast.FieldRead).Field = "k"
	copy.(*ast.SequentialProc).Procs[2].(*ast.GroupByProc).Keys[1].(*ast.FieldRead).Field = "p"
	copy.(*ast.SequentialProc).Procs[3].(*ast.SortProc).SortDir = 1
	copy.(*ast.SequentialProc).Procs[4].(*ast.HeadProc).Count = 2
	expectedProc, err := ast.UnpackProc(nil, testCopyJSONExpected)
	require.NoError(t, err)
	assert.Exactly(t, expectedProc, copy, "Wrong AST")

	// ensure our copy is deep by verifying that initial AST
	// wasn't modified
	p, err := ast.UnpackProc(nil, testCopyJSON)
	require.NoError(t, err)
	assert.Exactly(t, initialProc, p, "Initial AST modified (Copy not deep?) ")
}

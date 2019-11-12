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
      "filter": {
        "left": {
          "left": {
            "comparator": "eql",
            "field": {
              "op": "FieldRead",
              "field": "a"
            },
            "op": "CompareField",
            "value": {
              "type": "int",
              "value": "1"
            }
          },
          "op": "LogicalAnd",
          "right": {
            "op": "SearchString",
            "value": {
              "type": "string",
              "value": "foo"
            }
          }
        },
        "op": "LogicalAnd",
        "right": {
          "left": {
            "comparator": "eql",
            "field": {
              "op": "FieldRead",
              "field": "b"
            },
            "op": "CompareField",
            "value": {
              "type": "int",
              "value": "2"
            }
          },
          "op": "LogicalOr",
          "right": {
            "comparator": "eql",
            "field": {
              "op": "FieldRead",
              "field": "c"
            },
            "op": "CompareField",
            "value": {
              "type": "int",
              "value": "3"
            }
          }
        }
      },
      "op": "FilterProc"
    },
    {
      "fields": [
        "d",
        "e"
      ],
      "op": "CutProc"
    },
    {
      "duration": {
        "seconds": 86400,
        "type": "Duration"
      },
      "keys": [
        "n",
        "o"
      ],
      "op": "GroupByProc",
      "reducers": [
        {
          "field": "j",
          "op": "Sum",
          "var": "j"
        },
        {
          "field": "l",
          "op": "Min",
          "var": "m"
        }
      ]
    },
    {
      "fields": [
        "p"
      ],
      "op": "SortProc",
      "sortdir": -1
    },
    {
      "count": 1,
      "op": "HeadProc"
    }
  ]
}
`)

// boom ast 'a=1 foo (b=2 or d=4) | cut z, e | every hour sum(k) as j, min(l) as m by n, p | sort   p | head 2' | pbcopy
var testCopyJSONExpected = []byte(`
{
  "op": "SequentialProc",
  "procs": [
    {
      "filter": {
        "left": {
          "left": {
            "comparator": "eql",
            "field": {
              "op": "FieldRead",
              "field": "a"
            },
            "op": "CompareField",
            "value": {
              "type": "int",
              "value": "1"
            }
          },
          "op": "LogicalAnd",
          "right": {
            "op": "SearchString",
            "value": {
              "type": "string",
              "value": "foo"
            }
          }
        },
        "op": "LogicalAnd",
        "right": {
          "left": {
            "comparator": "eql",
            "field": {
              "op": "FieldRead",
              "field": "b"
            },
            "op": "CompareField",
            "value": {
              "type": "int",
              "value": "2"
            }
          },
          "op": "LogicalOr",
          "right": {
            "comparator": "eql",
            "field": {
              "op": "FieldRead",
              "field": "d"
            },
            "op": "CompareField",
            "value": {
              "type": "int",
              "value": "4"
            }
          }
        }
      },
      "op": "FilterProc"
    },
    {
      "fields": [
        "z",
        "e"
      ],
      "op": "CutProc"
    },
    {
      "duration": {
        "seconds": 3600,
        "type": "Duration"
      },
      "keys": [
        "n",
        "p"
      ],
      "op": "GroupByProc",
      "reducers": [
        {
          "field": "k",
          "op": "Sum",
          "var": "j"
        },
        {
          "field": "l",
          "op": "Min",
          "var": "m"
        }
      ]
    },
    {
      "fields": [
        "p"
      ],
      "op": "SortProc",
      "sortdir": 1
    },
    {
      "count": 2,
      "op": "HeadProc"
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
	copy.(*ast.SequentialProc).Procs[1].(*ast.CutProc).Fields[0] = "z"
	copy.(*ast.SequentialProc).Procs[2].(*ast.GroupByProc).Duration.Seconds = 3600
	copy.(*ast.SequentialProc).Procs[2].(*ast.GroupByProc).Reducers[0].Field = "k"
	copy.(*ast.SequentialProc).Procs[2].(*ast.GroupByProc).Keys[1] = "p"
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

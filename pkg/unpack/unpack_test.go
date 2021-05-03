package unpack_test

import (
	"testing"

	"github.com/brimdata/zed/pkg/unpack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Expr interface{}

type BinaryExpr struct {
	Op  string `json:"op" unpack:""`
	LHS Expr   `json:"lhs"`
	RHS Expr   `json:"rhs"`
}

type UnaryExpr struct {
	Op      string `json:"op" unpack:""`
	Operand Expr   `json:"operand"`
}

type List struct {
	Op    string `json:"op" unpack:""`
	Exprs []Expr `json:"exprs"`
}

type Terminal struct {
	Op   string `json:"op" unpack:""`
	Body string `json:"body"`
}

const binaryExprJSON = `
{
	"op":"BinaryExpr",
	"lhs": { "op": "Terminal", "body": "foo" } ,
	"rhs": { "op": "Terminal", "body": "bar" }
}`

var binaryExprExpected = &BinaryExpr{
	Op:  "BinaryExpr",
	LHS: &Terminal{Op: "Terminal", Body: "foo"},
	RHS: &Terminal{Op: "Terminal", Body: "bar"},
}

func TestUnpackBinaryExpr(t *testing.T) {
	reflector := unpack.New(
		BinaryExpr{},
		UnaryExpr{},
		Terminal{},
		List{},
	)
	actual, err := reflector.Unpack(binaryExprJSON)
	require.NoError(t, err)
	assert.Equal(t, binaryExprExpected, actual)
}

type BinaryExpr2 struct {
	Op  string `json:"op" unpack:"FooExpr"`
	LHS Expr   `json:"lhs"`
	RHS Expr   `json:"rhs"`
}

const typeTagJSON = `
{
	"op":"FooExpr",
	"lhs": { "op": "Terminal", "body": "foo" } ,
	"rhs": { "op": "Terminal", "body": "bar" }
}`

var typeTagExpected = &BinaryExpr2{
	Op:  "FooExpr",
	LHS: &Terminal{Op: "Terminal", Body: "foo"},
	RHS: &Terminal{Op: "Terminal", Body: "bar"},
}

func TestUnpackTypeTag(t *testing.T) {
	reflector := unpack.New(
		BinaryExpr2{},
		Terminal{},
	)
	actual, err := reflector.Unpack(typeTagJSON)
	require.NoError(t, err)
	assert.Equal(t, typeTagExpected, actual)
}

const nestedJSON = `
{
	"op":"BinaryExpr",
	"lhs": {
		"op": "UnaryExpr",
		"operand": { "op": "Terminal",  "body": "foo" }
	},
	"rhs": {
		"op": "BinaryExpr",
		"lhs": { "op": "Terminal", "body": "bar" },
		"rhs": { "op": "Terminal", "body": "baz" }
	}
}`

var nestedExpected = &BinaryExpr{
	Op: "BinaryExpr",
	LHS: &UnaryExpr{
		Op:      "UnaryExpr",
		Operand: &Terminal{"Terminal", "foo"},
	},
	RHS: &BinaryExpr{
		Op:  "BinaryExpr",
		LHS: &Terminal{"Terminal", "bar"},
		RHS: &Terminal{"Terminal", "baz"},
	},
}

func TestUnpackNested(t *testing.T) {
	reflector := unpack.New(
		BinaryExpr{},
		UnaryExpr{},
		Terminal{},
		List{},
	)
	actual, err := reflector.Unpack(nestedJSON)
	require.NoError(t, err)
	assert.Equal(t, nestedExpected, actual)
}

// Embedded is not a decodable object itself but has embedded stuff in it.
// That said, it still needs an Op so the decoder knows how to make it.
type Embedded struct {
	Op   string `json:"op" unpack:""`
	Root Pair   `json:"root"`
	// Ptr is handled by mapstructure
	Ptr *Pair `json:"ptr"`
}

// Pair has no Op because it only appears in things that have Ops.
type Pair struct {
	A Expr `json:"a"`
	B Expr `json:"b"`
}

const embeddedJSON = `
{
	"op": "Embedded",
	"root": {
	    "a": { "op": "Terminal", "body": "a" },
	    "b": { "op": "Terminal", "body": "b" }
         },
	"ptr": {
	    "a": { "op": "Terminal", "body": "c" },
	    "b": { "op": "Terminal", "body": "d" }
         }
}`

var embeddedExpected = &Embedded{
	Op: "Embedded",
	Root: Pair{
		A: &Terminal{"Terminal", "a"},
		B: &Terminal{"Terminal", "b"},
	},
	Ptr: &Pair{
		A: &Terminal{"Terminal", "c"},
		B: &Terminal{"Terminal", "d"},
	},
}

func TestUnpackEmbedded(t *testing.T) {
	reflector := unpack.New(
		BinaryExpr{},
		UnaryExpr{},
		Terminal{},
		List{},
		Embedded{},
	)
	actual, err := reflector.Unpack(embeddedJSON)
	require.NoError(t, err)
	assert.Equal(t, embeddedExpected, actual)
}

const listJSON = `
{
	"op": "List",
	"exprs": [ { "op": "Terminal", "body": "elem" } ]
}`

var listExpected = &List{
	Op: "List",
	Exprs: []Expr{
		&Terminal{Op: "Terminal", Body: "elem"},
	},
}

func TestUnpackList(t *testing.T) {
	reflector := unpack.New(
		BinaryExpr{},
		UnaryExpr{},
		Terminal{},
		List{},
	)
	actual, err := reflector.Unpack(listJSON)
	require.NoError(t, err)
	assert.Equal(t, listExpected, actual)
}

type PairList struct {
	Op    string `json:"op" unpack:""`
	Pairs []Pair `json:"pairs"`
}

const pairListJSON = `
{
	"op": "PairList",
	"pairs": [
	{
		"a": { "op": "Terminal", "body": "a1" },
		"b": { "op": "Terminal", "body": "b1" }
	},
	{
		"a": { "op": "Terminal", "body": "a2" },
		"b": { "op": "Terminal", "body": "b2" }
	} ]
}`

var pairListExpected = &PairList{
	Op: "PairList",
	Pairs: []Pair{
		{
			A: &Terminal{"Terminal", "a1"},
			B: &Terminal{"Terminal", "b1"},
		},
		{
			A: &Terminal{"Terminal", "a2"},
			B: &Terminal{"Terminal", "b2"},
		},
	},
}

func TestUnpackPairList(t *testing.T) {
	reflector := unpack.New(
		BinaryExpr{},
		UnaryExpr{},
		Terminal{},
		PairList{},
	)
	actual, err := reflector.Unpack(pairListJSON)
	require.NoError(t, err)
	assert.Equal(t, pairListExpected, actual)
}

type CutProc struct {
	Op     string       `json:"op" unpack:""`
	Fields []Assignment `json:"fields"`
}

type Assignment struct {
	Op  string `json:"op" unpack:""`
	LHS Expr   `json:"lhs"`
	RHS Expr   `json:"rhs"`
}

type Identifier struct {
	Op   string `json:"op" unpack:""`
	Name string `json:"name"`
}

const cutJSON = `
        {
            "fields": [
                {
                    "lhs": null,
                    "op": "Assignment",
                    "rhs": {
                        "name": "ts",
                        "op": "Identifier"
                    }
                },
                {
                    "lhs": {
                        "name": "foo",
                        "op": "Identifier"
                    },
                    "op": "Assignment",
                    "rhs": {
                        "name": "x",
                        "op": "Identifier"
                    }
                }
            ],
            "op": "CutProc"
        }

`

var cutExpected = &CutProc{
	Op: "CutProc",
	Fields: []Assignment{
		{
			Op: "Assignment",
			RHS: &Identifier{
				Op:   "Identifier",
				Name: "ts",
			},
		},
		{
			Op: "Assignment",
			LHS: &Identifier{
				Op:   "Identifier",
				Name: "foo",
			},
			RHS: &Identifier{
				Op:   "Identifier",
				Name: "x",
			},
		},
	},
}

func TestUnpackCut(t *testing.T) {
	reflector := unpack.New(
		CutProc{},
		Identifier{},
		Assignment{},
	)
	actual, err := reflector.Unpack(cutJSON)
	require.NoError(t, err)
	assert.Equal(t, cutExpected, actual)
}

const skipJSON = `
{
	"op":"BinaryExpr",
	"lhs": { "op": "Terminal", "body": "foo" } ,
	"rhs": { "op": "Terminal", "body": "bar" }
}`

type BinaryExpr3 struct {
	Op  string `json:"op" unpack:"BinaryExpr,skip"`
	LHS Expr   `json:"lhs"`
	RHS Expr   `json:"rhs"`
}

var skipExpected = map[string]interface{}{
	"lhs": map[string]interface{}{
		"body": "foo", "op": "Terminal"},
	"op": "BinaryExpr",
	"rhs": map[string]interface{}{
		"body": "bar",
		"op":   "Terminal"}}

func TestUnpackSkip(t *testing.T) {
	reflector := unpack.New(
		BinaryExpr3{},
		Terminal{},
	)
	actual, err := reflector.Unpack(skipJSON)
	require.NoError(t, err)
	assert.Equal(t, skipExpected, actual)
}

package unpack_test

import (
	"testing"

	"github.com/brimsec/zq/pkg/unpack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Expr interface {
	Which() string
}

type BinaryExpr struct {
	Op  string `json:"op"`
	LHS Expr   `json:"lhs"`
	RHS Expr   `json:"rhs"`
}

type UnaryExpr struct {
	Op      string `json:"op"`
	Operand Expr   `json:"operand"`
}

type List struct {
	Op    string `json:"op"`
	Exprs []Expr `json:"exprs"`
}

type Terminal struct {
	Body string `json:"body"`
}

func (t *Terminal) Which() string {
	return t.Body
}

func (*BinaryExpr) Which() string {
	return "BinaryExpr"
}

func (*UnaryExpr) Which() string {
	return "UnaryExpr"
}

const binaryExprJSON = `
{
	"op":"BinaryExpr",
	"lhs": { "op": "Terminal", "body": "foo" } ,
	"rhs": { "op": "Terminal", "body": "bar" }
}`

var binaryExprExpected = &BinaryExpr{
	Op:  "BinaryExpr",
	LHS: &Terminal{Body: "foo"},
	RHS: &Terminal{Body: "bar"},
}

func TestUnpackBinaryExpr(t *testing.T) {
	reflector := unpack.New().Init(
		BinaryExpr{},
		UnaryExpr{},
		Terminal{},
		List{},
	)
	actual, err := reflector.Unpack("op", binaryExprJSON)
	require.NoError(t, err)
	assert.Equal(t, binaryExprExpected, actual)
}

const withAsJSON = `
{
	"op":"WithAs",
	"lhs": { "op": "Terminal", "body": "foo" } ,
	"rhs": { "op": "Terminal", "body": "bar" }
}`

var withAsExpected = &BinaryExpr{
	Op:  "WithAs",
	LHS: &Terminal{Body: "foo"},
	RHS: &Terminal{Body: "bar"},
}

func TestUnpackWithAs(t *testing.T) {
	reflector := unpack.New().Init(Terminal{}).AddAs(BinaryExpr{}, "WithAs")
	actual, err := reflector.Unpack("op", withAsJSON)
	require.NoError(t, err)
	assert.Equal(t, withAsExpected, actual)
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
		Operand: &Terminal{"foo"},
	},
	RHS: &BinaryExpr{
		Op:  "BinaryExpr",
		LHS: &Terminal{"bar"},
		RHS: &Terminal{"baz"},
	},
}

func TestUnpackNested(t *testing.T) {
	reflector := unpack.New().Init(
		BinaryExpr{},
		UnaryExpr{},
		Terminal{},
		List{},
	)
	actual, err := reflector.Unpack("op", nestedJSON)
	require.NoError(t, err)
	assert.Equal(t, nestedExpected, actual)
}

// Embedded is not a decodable object itself but has embedded stuff in it.
// That said, it still needs an Op so the decoder knows how to make it.
type Embedded struct {
	Op   string `json:"op"`
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
		A: &Terminal{"a"},
		B: &Terminal{"b"},
	},
	Ptr: &Pair{
		A: &Terminal{"c"},
		B: &Terminal{"d"},
	},
}

func TestUnpackEmbedded(t *testing.T) {
	reflector := unpack.New().Init(
		BinaryExpr{},
		UnaryExpr{},
		Terminal{},
		List{},
		Embedded{},
	)
	actual, err := reflector.Unpack("op", embeddedJSON)
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
		&Terminal{Body: "elem"},
	},
}

func TestUnpackList(t *testing.T) {
	reflector := unpack.New().Init(
		BinaryExpr{},
		UnaryExpr{},
		Terminal{},
		List{},
	)
	actual, err := reflector.Unpack("op", listJSON)
	require.NoError(t, err)
	assert.Equal(t, listExpected, actual)
}

type PairList struct {
	Op    string `json:"op"`
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
		Pair{
			A: &Terminal{"a1"},
			B: &Terminal{"b1"},
		},
		Pair{
			A: &Terminal{"a2"},
			B: &Terminal{"b2"},
		},
	},
}

func TestUnpackPairList(t *testing.T) {
	reflector := unpack.New().Init(
		BinaryExpr{},
		UnaryExpr{},
		Terminal{},
		PairList{},
	)
	actual, err := reflector.Unpack("op", pairListJSON)
	require.NoError(t, err)
	assert.Equal(t, pairListExpected, actual)
}

type CutProc struct {
	Op     string       `json:"op"`
	Fields []Assignment `json:"fields"`
}

type Assignment struct {
	Op  string `json:"op"`
	LHS Expr   `json:"lhs"`
	RHS Expr   `json:"rhs"`
}

func (*Assignment) Which() string { return "Assignment" }

type Identifier struct {
	Op   string `json:"op"`
	Name string `json:"name"`
}

func (*Identifier) Which() string { return "Identifier" }

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
		Assignment{
			Op: "Assignment",
			RHS: &Identifier{
				Op:   "Identifier",
				Name: "ts",
			},
		},
		Assignment{
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
	reflector := unpack.New().Init(
		CutProc{},
		Identifier{},
		Assignment{},
	)
	actual, err := reflector.Unpack("op", cutJSON)
	require.NoError(t, err)
	assert.Equal(t, cutExpected, actual)
}

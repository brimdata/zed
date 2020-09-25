package expr

import (
	"errors"
	"fmt"
	"math"
	"net"
	"regexp"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/reglob"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

type Evaluator interface {
	Eval(*zng.Record) (zng.Value, error)
}

var ErrNoSuchField = errors.New("field is not present")
var ErrIncompatibleTypes = errors.New("incompatible types")
var ErrIndexOutOfBounds = errors.New("array index out of bounds")
var ErrNoSuchFunction = errors.New("no such function")
var ErrNotContainer = errors.New("cannot apply in to a non-container")
var ErrBadCast = errors.New("bad cast")

// CompileExpr compiles the given Expression into an object
// that evalutes the expression against a provided Record.  It returns an
// error if compilation fails for any reason.
//
// This is the "intepreted slow path" of the analytics engine.
// It is completely adaptive to the dynamic type system of zng so there
// are a lot of types checks and run-tme decisions made according to type.
//
// Eventually, we will optimize this by adding a "fast path" that dynamically
// generates byte codes (which an in turn be JIT assembled into machine code)
// for each zng TypeRecord encountered.  Once you know the type record,
// you can generate code using strong typing just as an OLAP system does
// due to its schemas defined up-front in its relational tables.  Here,
// each record type is like a schema and as we encounter them, we can compile
// optimized code for the now-static types within that record type.
//
// The keep flag, if true, says to return zng.Value that are safe to stash
// and will be garbage collected.  Otherwise, the expression will produce
// values into temporary buffers may be modified on subsequent calls to Eval.
// This is intended to minimize the garbage collection needs of the inner loop
// by not allocating memory on a per-Eval basis.  For uses like filtering and
// aggregations, where the results are immediately use, this is desirable and
// "keep" should be false in these cases. For use cases like storing the results
// as groupby keys, the results must bae nonvolatile and "keep" should be true.
func CompileExpr(node ast.Expression, keep bool) (Evaluator, error) {
	return compileExpr(node, true, keep)
}

func compileExpr(node ast.Expression, root, keep bool) (Evaluator, error) {
	switch n := node.(type) {
	case *ast.Literal:
		return NewLiteral(*n)
	case *ast.Field:
		return newFieldNode(n.Field, nil, root), nil
	case *ast.UnaryExpression:
		return compileUnary(*n, keep)

	case *ast.BinaryExpression:
		if n.Operator == "." {
			return compileDotExpr(n.LHS, n.RHS, keep)
		}
		lhs, err := compileExpr(n.LHS, true, keep)
		if err != nil {
			return nil, err
		}
		rhs, err := compileExpr(n.RHS, true, keep)
		if err != nil {
			return nil, err
		}
		switch n.Operator {
		case "AND", "OR":
			return compileLogical(lhs, rhs, n.Operator)
		case "in":
			return compileIn(lhs, rhs)
		case "=", "!=":
			return compileCompareEquality(lhs, rhs, n.Operator, keep)
		case "=~", "!~":
			return compilePatternMatch(lhs, rhs, n.Operator)
		case "<", "<=", ">", ">=":
			return compileCompareRelative(lhs, rhs, n.Operator, keep)
		case "+", "-", "*", "/":
			return compileArithmetic(lhs, rhs, n.Operator, keep)
		case "[":
			return compileIndexExpr(lhs, rhs)
		default:
			return nil, fmt.Errorf("invalid binary operator %s", n.Operator)
		}

	case *ast.ConditionalExpression:
		return compileConditional(*n, keep)

	case *ast.FunctionCall:
		return compileCall(*n, keep)

	case *ast.CastExpression:
		return compileCast(*n, keep)

	default:
		return nil, fmt.Errorf("invalid expression type %T", node)
	}
}

func CompileExprs(nodes []ast.Expression, keep bool) ([]Evaluator, error) {
	var exprs []Evaluator
	for k := range nodes {
		e, err := compileExpr(nodes[k], true, keep)
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, e)
	}
	return exprs, nil
}

type Not struct {
	expr Evaluator
}

func compileUnary(node ast.UnaryExpression, keep bool) (Evaluator, error) {
	if node.Operator != "!" {
		return nil, fmt.Errorf("unknown unary operator %s\n", node.Operator)
	}
	e, err := compileExpr(node.Operand, true, keep)
	if err != nil {
		return nil, err
	}
	return &Not{e}, nil
}

func (n *Not) Eval(rec *zng.Record) (zng.Value, error) {
	zv, err := evalBool(n.expr, rec)
	if err != nil {
		return zv, err
	}
	if zng.IsTrue(zv.Bytes) {
		return zng.False, nil
	}
	return zng.True, nil
}

type And struct {
	lhs Evaluator
	rhs Evaluator
}

type Or struct {
	lhs Evaluator
	rhs Evaluator
}

func compileLogical(lhs, rhs Evaluator, operator string) (Evaluator, error) {
	switch operator {
	case "AND":
		return &And{lhs, rhs}, nil
	case "OR":
		return &Or{lhs, rhs}, nil
	default:
		return nil, fmt.Errorf("unknown logical operator: %s", operator)
	}
}

func evalBool(e Evaluator, rec *zng.Record) (zng.Value, error) {
	zv, err := e.Eval(rec)
	if err != nil {
		return zv, err
	}
	if zv.Type != zng.TypeBool {
		err = ErrIncompatibleTypes
	}
	return zv, err
}

func (a *And) Eval(rec *zng.Record) (zng.Value, error) {
	lhs, err := evalBool(a.lhs, rec)
	if err != nil {
		return lhs, err
	}
	if !zng.IsTrue(lhs.Bytes) {
		return zng.False, nil
	}
	rhs, err := evalBool(a.rhs, rec)
	if err != nil {
		return lhs, err
	}
	if !zng.IsTrue(rhs.Bytes) {
		return zng.False, nil
	}
	return zng.True, nil
}

func (o *Or) Eval(rec *zng.Record) (zng.Value, error) {
	lhs, err := evalBool(o.lhs, rec)
	if err != nil {
		return lhs, err
	}
	if zng.IsTrue(lhs.Bytes) {
		return zng.True, nil
	}
	rhs, err := evalBool(o.rhs, rec)
	if err != nil {
		return lhs, err
	}
	if zng.IsTrue(rhs.Bytes) {
		return zng.True, nil
	}
	return zng.False, nil
}

type In struct {
	elem      Evaluator
	container Evaluator
	vals      Coercion
}

func compileIn(elem, container Evaluator) (Evaluator, error) {
	return &In{
		elem:      elem,
		container: container,
	}, nil
}

func (i *In) Eval(rec *zng.Record) (zng.Value, error) {
	container, err := i.container.Eval(rec)
	if err != nil {
		return container, err
	}
	typ := zng.InnerType(container.Type)
	if typ == nil {
		return zng.Value{}, ErrNotContainer
	}
	elem, err := i.elem.Eval(rec)
	if err != nil {
		return elem, err
	}
	iter := zcode.Iter(container.Bytes)
	for {
		if iter.Done() {
			return zng.False, nil
		}
		zv, _, err := iter.Next()
		if err != nil {
			return zng.Value{}, err
		}
		_, err = i.vals.coerce(elem, zng.Value{typ, zv})
		if err != nil {
			return zng.Value{}, err
		}
		if i.vals.equal() {
			return zng.True, nil
		}
	}
}

func floatToInt64(f float64) (int64, bool) {
	i := int64(f)
	if float64(i) == f {
		return i, true
	}
	return 0, false
}

func floatToUint64(f float64) (uint64, bool) {
	u := uint64(f)
	if float64(u) == f {
		return u, true
	}
	return 0, false
}

type Equal struct {
	numeric
	equality bool
}

func compileCompareEquality(lhs, rhs Evaluator, operator string, keep bool) (Evaluator, error) {
	e := &Equal{numeric: newNumeric(lhs, rhs, keep)}
	switch operator {
	case "=":
		e.equality = true
	case "!=":
	default:
		return nil, fmt.Errorf("unknown equlity operator: %s", operator)
	}
	return e, nil
}

func (e *Equal) Eval(rec *zng.Record) (zng.Value, error) {
	_, err := e.numeric.eval(rec)
	if err != nil {
		if err == ErrOverflow {
			// If there was overflow converting one to the other,
			// we know they can't be equal.
			if e.equality {
				return zng.False, nil
			}
			return zng.True, nil
		}
		return zng.Value{}, err
	}
	result := e.vals.equal()
	if !e.equality {
		result = !result
	}
	if result {
		return zng.True, nil
	}
	return zng.False, nil
}

type Match struct {
	equality bool
	lhs      Evaluator
	rhs      Evaluator
}

func compilePatternMatch(lhs, rhs Evaluator, op string) (Evaluator, error) {
	equality := true
	if op == "!~" {
		equality = false
	}
	return &Match{
		equality: equality,
		lhs:      lhs,
		rhs:      rhs,
	}, nil
}

func (m *Match) Eval(rec *zng.Record) (zng.Value, error) {
	lhs, err := m.lhs.Eval(rec)
	if err != nil {
		return zng.Value{}, err
	}
	rhs, err := m.rhs.Eval(rec)
	if err != nil {
		return zng.Value{}, err
	}
	var result bool
	rid := rhs.Type.ID()
	lid := lhs.Type.ID()
	if zng.IsStringy(rid) {
		if !zng.IsStringy(lid) {
			return zng.Value{}, ErrIncompatibleTypes
		}
		pattern := reglob.Reglob(string(rhs.Bytes))
		result, err = regexp.MatchString(pattern, string(lhs.Bytes))
		if err != nil {
			return zng.Value{}, fmt.Errorf("error comparing pattern: %w", err)
		}
	} else if rid == zng.IdNet && lid == zng.IdIP {
		addr, _ := zng.DecodeIP(lhs.Bytes)
		net, _ := zng.DecodeNet(rhs.Bytes)
		result = net.IP.Equal(addr.Mask(net.Mask))
	} else {
		return zng.Value{}, ErrIncompatibleTypes
	}
	if !m.equality {
		result = !result
	}
	if result {
		return zng.True, nil
	}
	return zng.False, nil
}

type numeric struct {
	lhs  Evaluator
	rhs  Evaluator
	vals Coercion
}

func newNumeric(lhs, rhs Evaluator, keep bool) numeric {
	return numeric{
		lhs:  lhs,
		rhs:  rhs,
		vals: newCoercion(keep),
	}
}

func (n *numeric) eval(rec *zng.Record) (int, error) {
	lhs, err := n.lhs.Eval(rec)
	if err != nil {
		return 0, err
	}
	rhs, err := n.rhs.Eval(rec)
	if err != nil {
		return 0, err
	}
	return n.vals.coerce(lhs, rhs)
}

type Compare struct {
	numeric
	convert func(int) bool
}

func compileCompareRelative(lhs, rhs Evaluator, operator string, keep bool) (Evaluator, error) {
	c := &Compare{numeric: newNumeric(lhs, rhs, keep)}
	switch operator {
	case "<":
		c.convert = func(v int) bool { return v < 0 }
	case "<=":
		c.convert = func(v int) bool { return v <= 0 }
	case ">":
		c.convert = func(v int) bool { return v > 0 }
	case ">=":
		c.convert = func(v int) bool { return v >= 0 }
	default:
		return nil, fmt.Errorf("unknown comparison operator: %s", operator)
	}
	return c, nil
}

func (c *Compare) result(result int) zng.Value {
	if c.convert(result) {
		return zng.True
	}
	return zng.False
}

func (c *Compare) Eval(rec *zng.Record) (zng.Value, error) {
	lhs, err := c.lhs.Eval(rec)
	if err != nil {
		return zng.Value{}, err
	}
	rhs, err := c.rhs.Eval(rec)
	if err != nil {
		return zng.Value{}, err
	}
	id, err := c.vals.coerce(lhs, rhs)
	if err != nil {
		// If coercion fails due to overflow, then we know there is a
		// mixed signed and unsigned situation and either the unsigned
		// value couldn't be converted to an int64 because it was too big,
		// or the signed value couldn't be converted to a uint64 because
		// it was negative.  In either case, the unsigned value is bigger
		// than the signed value.
		if err == ErrOverflow {
			result := 1
			if zng.IsSigned(lhs.Type.ID()) {
				result = -1
			}
			return c.result(result), nil
		}
		return zng.False, err
	}
	var result int
	if !c.vals.equal() {
		if zng.IsFloat(id) {
			v1, _ := zng.DecodeFloat64(c.vals.a)
			v2, _ := zng.DecodeFloat64(c.vals.b)
			if v1 < v2 {
				result = -1
			} else {
				result = 1
			}
		} else if zng.IsSigned(id) {
			v1, _ := zng.DecodeInt(c.vals.a)
			v2, _ := zng.DecodeInt(c.vals.b)
			if v1 < v2 {
				result = -1
			} else {
				result = 1
			}
		} else if zng.IsNumber(id) {
			v1, _ := zng.DecodeUint(c.vals.a)
			v2, _ := zng.DecodeUint(c.vals.b)
			if v1 < v2 {
				result = -1
			} else {
				result = 1
			}
		} else if zng.IsStringy(id) {
			v1, _ := zng.DecodeString(c.vals.a)
			v2, _ := zng.DecodeString(c.vals.b)
			if v1 < v2 {
				result = -1
			} else {
				result = 1
			}
		} else {
			return zng.Value{}, fmt.Errorf("bad comparison type ID: %d", id)
		}
	}
	if c.convert(result) {
		return zng.True, nil
	}
	return zng.False, nil
}

type Add struct {
	numeric
}

type Subtract struct {
	numeric
}

type Multiply struct {
	numeric
}

type Divide struct {
	numeric
}

// compileArithmetic compiles an expression of the form "expr1 op expr2"
// for the arithmetic operators +, -, *, /
func compileArithmetic(lhs, rhs Evaluator, op string, keep bool) (Evaluator, error) {
	n := newNumeric(lhs, rhs, keep)
	switch op {
	case "+":
		return &Add{n}, nil
	case "-":
		return &Subtract{n}, nil
	case "*":
		return &Multiply{n}, nil
	case "/":
		return &Divide{n}, nil
	}
	return nil, fmt.Errorf("unknown arithmetic operator: %s", op)
}

func (a *Add) Eval(rec *zng.Record) (zng.Value, error) {
	id, err := a.eval(rec)
	if err != nil {
		return zng.Value{}, err
	}
	typ := zng.LookupPrimitiveById(id)
	if zng.IsFloat(id) {
		v1, _ := zng.DecodeFloat64(a.vals.a)
		v2, _ := zng.DecodeFloat64(a.vals.b)
		return zng.Value{typ, a.vals.Float64(v1 + v2)}, nil
	}
	if zng.IsSigned(id) {
		v1, _ := zng.DecodeInt(a.vals.a)
		v2, _ := zng.DecodeInt(a.vals.b)
		return zng.Value{typ, a.vals.Int(v1 + v2)}, nil
	}
	if zng.IsNumber(id) {
		v1, _ := zng.DecodeUint(a.vals.a)
		v2, _ := zng.DecodeUint(a.vals.b)
		return zng.Value{typ, a.vals.Uint(v1 + v2)}, nil
	}
	if zng.IsStringy(id) {
		v1, _ := zng.DecodeString(a.vals.a)
		v2, _ := zng.DecodeString(a.vals.b)
		// XXX GC
		return zng.Value{typ, zng.EncodeString(v1 + v2)}, nil
	}
	return zng.Value{}, ErrIncompatibleTypes
}

func (s *Subtract) Eval(rec *zng.Record) (zng.Value, error) {
	id, err := s.eval(rec)
	if err != nil {
		return zng.Value{}, err
	}
	typ := zng.LookupPrimitiveById(id)
	if zng.IsFloat(id) {
		v1, _ := zng.DecodeFloat64(s.vals.a)
		v2, _ := zng.DecodeFloat64(s.vals.b)
		return zng.Value{typ, s.vals.Float64(v1 - v2)}, nil
	}
	if zng.IsSigned(id) {
		v1, _ := zng.DecodeInt(s.vals.a)
		v2, _ := zng.DecodeInt(s.vals.b)
		return zng.Value{typ, s.vals.Int(v1 - v2)}, nil
	}
	if zng.IsNumber(id) {
		v1, _ := zng.DecodeUint(s.vals.a)
		v2, _ := zng.DecodeUint(s.vals.b)
		return zng.Value{typ, s.vals.Uint(v1 - v2)}, nil
	}
	return zng.Value{}, ErrIncompatibleTypes
}

func (m *Multiply) Eval(rec *zng.Record) (zng.Value, error) {
	id, err := m.eval(rec)
	if err != nil {
		return zng.Value{}, err
	}
	typ := zng.LookupPrimitiveById(id)
	if zng.IsFloat(id) {
		v1, _ := zng.DecodeFloat64(m.vals.a)
		v2, _ := zng.DecodeFloat64(m.vals.b)
		return zng.Value{typ, m.vals.Float64(v1 * v2)}, nil
	}
	if zng.IsSigned(id) {
		v1, _ := zng.DecodeInt(m.vals.a)
		v2, _ := zng.DecodeInt(m.vals.b)
		return zng.Value{typ, m.vals.Int(v1 * v2)}, nil
	}
	if zng.IsNumber(id) {
		v1, _ := zng.DecodeUint(m.vals.a)
		v2, _ := zng.DecodeUint(m.vals.b)
		return zng.Value{typ, m.vals.Uint(v1 * v2)}, nil
	}
	return zng.Value{}, ErrIncompatibleTypes
}

func (d *Divide) Eval(rec *zng.Record) (zng.Value, error) {
	id, err := d.eval(rec)
	if err != nil {
		return zng.Value{}, err
	}
	typ := zng.LookupPrimitiveById(id)
	if zng.IsFloat(id) {
		v1, _ := zng.DecodeFloat64(d.vals.a)
		v2, _ := zng.DecodeFloat64(d.vals.b)
		if v2 == 0 {
			// XXX change to error type in subsequent PR
			return zng.Value{zng.TypeString, zng.EncodeString("floating point divide by 0")}, nil
		}
		return zng.Value{typ, d.vals.Float64(v1 / v2)}, nil
	}
	if zng.IsSigned(id) {
		v1, _ := zng.DecodeInt(d.vals.a)
		v2, _ := zng.DecodeInt(d.vals.b)
		if v2 == 0 {
			// XXX change to error type in subsequent PR
			return zng.Value{zng.TypeString, zng.EncodeString("signed integer divide by 0")}, nil
		}
		return zng.Value{typ, d.vals.Int(v1 / v2)}, nil
	}
	if zng.IsNumber(id) {
		v1, _ := zng.DecodeUint(d.vals.a)
		v2, _ := zng.DecodeUint(d.vals.b)
		if v2 == 0 {
			// XXX change to error type in subsequent PR
			return zng.Value{zng.TypeString, zng.EncodeString("unsigned integer divide by 0")}, nil
		}
		return zng.Value{typ, d.vals.Uint(v1 / v2)}, nil
	}
	return zng.Value{}, ErrIncompatibleTypes
}

func getNthFromContainer(container zcode.Bytes, idx uint) (zcode.Bytes, error) {
	iter := zcode.Iter(container)
	var i uint = 0
	for ; !iter.Done(); i++ {
		zv, _, err := iter.Next()
		if err != nil {
			return nil, err
		}
		if i == idx {
			return zv, nil
		}
	}
	return nil, ErrIndexOutOfBounds
}

// Index represents an index operator "container[index]" where container is
// either an array (with index type integer) or a record (with index type string).
type Index struct {
	container Evaluator
	index     Evaluator
}

func compileIndexExpr(container, index Evaluator) (Evaluator, error) {
	return &Index{container, index}, nil
}

func (i *Index) Eval(rec *zng.Record) (zng.Value, error) {
	array, err := i.container.Eval(rec)
	if err != nil {
		return zng.Value{}, err
	}
	//XXX add support for records
	typ, ok := array.Type.(*zng.TypeArray)
	if !ok {
		//XXX this operator should be used for record accesses for
		// things like foo["@type"]
		return zng.Value{}, errors.New("indexed value is not an array")
	}
	index, err := i.index.Eval(rec)
	if err != nil {
		return zng.Value{}, err
	}
	id := index.Type.ID()
	if !zng.IsInteger(id) {
		return zng.Value{}, errors.New("array index is not an integer")
	}
	var idx uint
	if zng.IsSigned(id) {
		v, _ := zng.DecodeInt(index.Bytes)
		if idx < 0 {
			return zng.Value{}, ErrIndexOutOfBounds
		}
		idx = uint(v)
	} else {
		v, _ := zng.DecodeUint(index.Bytes)
		idx = uint(v)
	}
	zv, err := getNthFromContainer(array.Bytes, idx)
	if err != nil {
		return zng.Value{}, err
	}
	return zng.Value{typ.Type, zv}, nil
}

type Conditional struct {
	predicate Evaluator
	thenExpr  Evaluator
	elseExpr  Evaluator
}

func compileConditional(node ast.ConditionalExpression, keep bool) (Evaluator, error) {
	var err error
	predicate, err := compileExpr(node.Condition, true, keep)
	if err != nil {
		return nil, err
	}
	thenExpr, err := compileExpr(node.Then, true, keep)
	if err != nil {
		return nil, err
	}
	elseExpr, err := compileExpr(node.Else, true, keep)
	if err != nil {
		return nil, err
	}
	return &Conditional{
		predicate: predicate,
		thenExpr:  thenExpr,
		elseExpr:  elseExpr,
	}, nil
}

func (c *Conditional) Eval(rec *zng.Record) (zng.Value, error) {
	val, err := c.predicate.Eval(rec)
	if err != nil {
		return zng.Value{}, err
	}
	if val.Type.ID() != zng.IdBool {
		return zng.Value{}, ErrIncompatibleTypes
	}
	if zng.IsTrue(val.Bytes) {
		return c.thenExpr.Eval(rec)
	}
	return c.elseExpr.Eval(rec)
}

func compileDotExpr(lhs, rhs ast.Expression, keep bool) (*FieldExpr, error) {
	record, err := compileExpr(lhs, true, keep)
	if err != nil {
		return nil, err
	}
	field, err := compileExpr(rhs, false, keep)
	if err != nil {
		return nil, err
	}
	return &FieldExpr{field, record, false}, nil
}

type Call struct {
	function Function
	exprs    []Evaluator
	args     *Args
}

func compileCall(node ast.FunctionCall, keep bool) (Evaluator, error) {
	fn, ok := allFns[node.Function]
	if !ok {
		return nil, fmt.Errorf("%s: %w", node.Function, ErrNoSuchFunction)
	}
	nargs := len(node.Args)
	if fn.minArgs >= 0 && nargs < fn.minArgs {
		return nil, fmt.Errorf("%s: %w", node.Function, ErrTooFewArgs)
	}
	if fn.maxArgs >= 0 && nargs > fn.maxArgs {
		return nil, fmt.Errorf("%s: %w", node.Function, ErrTooManyArgs)
	}
	exprs := make([]Evaluator, 0, nargs)
	for _, expr := range node.Args {
		e, err := compileExpr(expr, true, keep)
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, e)
	}

	return &Call{
		function: fn.impl,
		exprs:    exprs,
		args:     NewArgs(nargs, keep),
	}, nil
}

func (c *Call) Eval(rec *zng.Record) (zng.Value, error) {
	for k, e := range c.exprs {
		val, err := e.Eval(rec)
		if err != nil {
			return zng.Value{}, err
		}
		c.args.vals[k] = val
	}
	return c.function(c.args)
}

func compileCast(node ast.CastExpression, keep bool) (Evaluator, error) {
	expr, err := compileExpr(node.Expr, true, keep)
	if err != nil {
		return nil, err
	}
	// XXX should handle alias casts... need type context.
	// compile is going to need a local type context to create literals
	// of complex types?
	switch node.Type {
	case "int8":
		return &IntCast{expr, zng.TypeInt8, math.MinInt8, math.MaxInt8}, nil
	case "int16":
		return &IntCast{expr, zng.TypeInt16, math.MinInt16, math.MaxInt16}, nil
	case "int32":
		return &IntCast{expr, zng.TypeInt32, math.MinInt32, math.MaxInt32}, nil
	case "int64":
		return &IntCast{expr, zng.TypeInt64, 0, 0}, nil
	case "uint8":
		return &UintCast{expr, zng.TypeUint8, math.MaxUint8}, nil
	case "uint16":
		return &UintCast{expr, zng.TypeUint16, math.MaxUint16}, nil
	case "uint32":
		return &UintCast{expr, zng.TypeUint32, math.MaxUint32}, nil
	case "uint64":
		return &UintCast{expr, zng.TypeUint64, 0}, nil
	case "float64":
		return &Float64Cast{expr}, nil
	case "ip":
		return &IPCast{expr}, nil
	case "time":
		return &TimeCast{expr}, nil
	case "string":
		return &StringCast{expr, zng.TypeString}, nil
	case "bstring":
		return &StringCast{expr, zng.TypeBstring}, nil
	default:
		return nil, fmt.Errorf("cast to %s not implemeneted", node.Type)
	}
}

type IntCast struct {
	expr Evaluator
	typ  zng.Type
	min  int64
	max  int64
}

func (i *IntCast) Eval(rec *zng.Record) (zng.Value, error) {
	zv, err := i.expr.Eval(rec)
	if err != nil {
		return zng.Value{}, err
	}
	v, ok := CoerceToInt(zv)
	// XXX better error message
	if !ok || (i.min != 0 && (v < i.min || v > i.max)) {
		return zng.Value{}, ErrBadCast
	}
	// XXX GC
	return zng.Value{i.typ, zng.EncodeInt(v)}, nil
}

type UintCast struct {
	expr Evaluator
	typ  zng.Type
	max  uint64
}

func (u *UintCast) Eval(rec *zng.Record) (zng.Value, error) {
	zv, err := u.expr.Eval(rec)
	if err != nil {
		return zng.Value{}, err
	}
	v, ok := CoerceToUint(zv)
	// XXX better error message
	if !ok || (u.max != 0 && v > u.max) {
		return zng.Value{}, ErrBadCast
	}
	// XXX GC
	return zng.Value{u.typ, zng.EncodeUint(v)}, nil
}

type Float64Cast struct {
	expr Evaluator
}

func (i *Float64Cast) Eval(rec *zng.Record) (zng.Value, error) {
	zv, err := i.expr.Eval(rec)
	if err != nil {
		return zng.Value{}, err
	}
	f, ok := CoerceToFloat(zv)
	if !ok {
		return zng.Value{}, ErrBadCast
	}
	return zng.Value{zng.TypeFloat64, zng.EncodeFloat64(f)}, nil
}

type IPCast struct {
	expr Evaluator
}

func (i *IPCast) Eval(rec *zng.Record) (zng.Value, error) {
	zv, err := i.expr.Eval(rec)
	if err != nil {
		return zng.Value{}, err
	}
	if !isStringy(zv) {
		return zng.Value{}, ErrBadCast
	}
	ip := net.ParseIP(string(zv.Bytes))
	if ip == nil {
		return zng.Value{}, ErrBadCast
	}
	// XXX GC
	return zng.Value{zng.TypeIP, zng.EncodeIP(ip)}, nil
}

type TimeCast struct {
	expr Evaluator
}

func (t *TimeCast) Eval(rec *zng.Record) (zng.Value, error) {
	zv, err := t.expr.Eval(rec)
	if err != nil {
		return zng.Value{}, err
	}
	if zng.IsFloat(zv.Type.ID()) {
		f, _ := zng.DecodeFloat64(zv.Bytes)
		ts := nano.FloatToTs(f)
		// XXX GC
		return zng.Value{zng.TypeTime, zng.EncodeTime(ts)}, nil
	}
	ns, ok := CoerceToInt(zv)
	if !ok {
		return zng.Value{}, ErrBadCast
	}
	return zng.Value{zng.TypeTime, zng.EncodeTime(nano.Ts(ns * 1_000_000_000))}, nil
}

type StringCast struct {
	expr Evaluator
	typ  zng.Type
}

func (s *StringCast) Eval(rec *zng.Record) (zng.Value, error) {
	zv, err := s.expr.Eval(rec)
	if err != nil {
		return zng.Value{}, err
	}
	return zng.Value{s.typ, zng.EncodeString(zv.String())}, nil
}

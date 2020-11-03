package expr

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"net"
	"regexp"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/reglob"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
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
// that evaluates the expression against a provided Record.  It returns an
// error if compilation fails for any reason.
//
// This is the "intepreted slow path" of the analytics engine.  Because it
// handles dynamic typinig at runtime, overheads are incurrend due to
// various type checks and coercions that determine different computational
// outcomes based on type.  There is nothing here that optimizes analytics
// for native machine types; these optimizations (will) happen in the pushdown
// predicate processing engine in the zst columnar scanner.
//
// Eventually, we will optimize this zst "fast path" by dynamically
// generating byte codes (which an in turn be JIT assembled into machine code)
// for each zng TypeRecord encountered.  Once you know the type record,
// you can generate code using strong typing just as an OLAP system does
// due to its schemas defined up-front in its relational tables.  Here,
// each record type is like a schema and as we encounter them, we can compile
// optimized code for the now-static types within that record type.
//
// The Evaluator return by CompilExpr produces zng.Values that are stored
// in temporary buffers and may be modified on subsequent calls to Eval.
// This is intended to minimize the garbage collection needs of the inner loop
// by not allocating memory on a per-Eval basis.  For uses like filtering and
// aggregations, where the results are immediately use, this is desirable and
// efficient but for use cases like storing the results as groupby keys, the
// resulting zng.Value should be copied (e.g., via zng.Value.Copy()).
//
// TBD: string values and net.IP address do not need to be copied because they
// are allocated by go libraries and temporary buffers are not used.  This will
// change down the road when we implement no-allocation string and IP conversion.
func CompileExpr(zctx *resolver.Context, node ast.Expression) (Evaluator, error) {
	return compileExpr(zctx, node)
}

func compileExpr(zctx *resolver.Context, node ast.Expression) (Evaluator, error) {
	switch n := node.(type) {
	case *ast.Literal:
		return NewLiteral(*n)
	case *ast.Identifier:
		return nil, fmt.Errorf("stray identifier in AST: %s", n.Name)
	case *ast.RootRecord:
		return &RootRecord{}, nil
	case *ast.UnaryExpression:
		return compileUnary(zctx, *n)

	case *ast.BinaryExpression:
		if n.Operator == "." {
			return compileDotExpr(zctx, n.LHS, n.RHS)
		}
		lhs, err := compileExpr(zctx, n.LHS)
		if err != nil {
			return nil, err
		}
		rhs, err := compileExpr(zctx, n.RHS)
		if err != nil {
			return nil, err
		}
		switch n.Operator {
		case "AND", "OR":
			return compileLogical(lhs, rhs, n.Operator)
		case "in":
			return compileIn(lhs, rhs)
		case "=", "!=":
			return compileCompareEquality(zctx, lhs, rhs, n.Operator)
		case "=~", "!~":
			return compilePatternMatch(lhs, rhs, n.Operator)
		case "<", "<=", ">", ">=":
			return compileCompareRelative(zctx, lhs, rhs, n.Operator)
		case "+", "-", "*", "/":
			return compileArithmetic(zctx, lhs, rhs, n.Operator)
		case "[":
			return compileIndexExpr(zctx, lhs, rhs)
		default:
			return nil, fmt.Errorf("invalid binary operator %s", n.Operator)
		}

	case *ast.ConditionalExpression:
		return compileConditional(zctx, *n)

	case *ast.FunctionCall:
		return compileCall(zctx, *n)

	case *ast.CastExpression:
		return compileCast(zctx, *n)

	default:
		return nil, fmt.Errorf("invalid expression type %T", node)
	}
}

func CompileExprs(zctx *resolver.Context, nodes []ast.Expression) ([]Evaluator, error) {
	var exprs []Evaluator
	for k := range nodes {
		e, err := compileExpr(zctx, nodes[k])
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

func compileUnary(zctx *resolver.Context, node ast.UnaryExpression) (Evaluator, error) {
	if node.Operator != "!" {
		return nil, fmt.Errorf("unknown unary operator %s\n", node.Operator)
	}
	e, err := compileExpr(zctx, node.Operand)
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
	iter := container.Bytes.Iter()
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

type Equal struct {
	numeric
	equality bool
}

func compileCompareEquality(zctx *resolver.Context, lhs, rhs Evaluator, operator string) (Evaluator, error) {
	e := &Equal{numeric: newNumeric(zctx, lhs, rhs)}
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
	zctx *resolver.Context
	lhs  Evaluator
	rhs  Evaluator
	vals Coercion
}

func newNumeric(zctx *resolver.Context, lhs, rhs Evaluator) numeric {
	return numeric{
		zctx: zctx,
		lhs:  lhs,
		rhs:  rhs,
	}
}

func enumify(v zng.Value) (zng.Value, error) {
	// automatically convert an enum to its value when coercing
	if typ, ok := v.Type.(*zng.TypeEnum); ok {
		selector, err := zng.DecodeUint(v.Bytes)
		if err != nil {
			return zng.Value{}, err
		}
		elem, err := typ.Element(int(selector))
		if err != nil {
			return zng.Value{}, err
		}
		return zng.Value{typ.Type, elem.Value}, nil
	}
	return v, nil
}

func (n *numeric) eval(rec *zng.Record) (int, error) {
	lhs, err := n.lhs.Eval(rec)
	if err != nil {
		return 0, err
	}
	lhs, err = enumify(lhs)
	if err != nil {
		return 0, err
	}
	rhs, err := n.rhs.Eval(rec)
	if err != nil {
		return 0, err
	}
	rhs, err = enumify(rhs)
	if err != nil {
		return 0, err
	}
	return n.vals.coerce(lhs, rhs)
}

type Compare struct {
	numeric
	convert func(int) bool
}

func compileCompareRelative(zctx *resolver.Context, lhs, rhs Evaluator, operator string) (Evaluator, error) {
	c := &Compare{numeric: newNumeric(zctx, lhs, rhs)}
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
		switch {
		case zng.IsFloat(id):
			v1, _ := zng.DecodeFloat64(c.vals.a)
			v2, _ := zng.DecodeFloat64(c.vals.b)
			if v1 < v2 {
				result = -1
			} else {
				result = 1
			}
		case zng.IsSigned(id):
			v1, _ := zng.DecodeInt(c.vals.a)
			v2, _ := zng.DecodeInt(c.vals.b)
			if v1 < v2 {
				result = -1
			} else {
				result = 1
			}
		case zng.IsNumber(id):
			v1, _ := zng.DecodeUint(c.vals.a)
			v2, _ := zng.DecodeUint(c.vals.b)
			if v1 < v2 {
				result = -1
			} else {
				result = 1
			}
		case zng.IsStringy(id):
			v1, _ := zng.DecodeString(c.vals.a)
			v2, _ := zng.DecodeString(c.vals.b)
			if v1 < v2 {
				result = -1
			} else {
				result = 1
			}
		default:
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
func compileArithmetic(zctx *resolver.Context, lhs, rhs Evaluator, op string) (Evaluator, error) {
	n := newNumeric(zctx, lhs, rhs)
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
	switch {
	case zng.IsFloat(id):
		v1, _ := zng.DecodeFloat64(a.vals.a)
		v2, _ := zng.DecodeFloat64(a.vals.b)
		return zng.Value{typ, a.vals.Float64(v1 + v2)}, nil
	case zng.IsSigned(id):
		v1, _ := zng.DecodeInt(a.vals.a)
		v2, _ := zng.DecodeInt(a.vals.b)
		return zng.Value{typ, a.vals.Int(v1 + v2)}, nil
	case zng.IsNumber(id):
		v1, _ := zng.DecodeUint(a.vals.a)
		v2, _ := zng.DecodeUint(a.vals.b)
		return zng.Value{typ, a.vals.Uint(v1 + v2)}, nil
	case zng.IsStringy(id):
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
	switch {
	case zng.IsFloat(id):
		v1, _ := zng.DecodeFloat64(s.vals.a)
		v2, _ := zng.DecodeFloat64(s.vals.b)
		return zng.Value{typ, s.vals.Float64(v1 - v2)}, nil
	case zng.IsSigned(id):
		v1, _ := zng.DecodeInt(s.vals.a)
		v2, _ := zng.DecodeInt(s.vals.b)
		return zng.Value{typ, s.vals.Int(v1 - v2)}, nil
	case zng.IsNumber(id):
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
	switch {
	case zng.IsFloat(id):
		v1, _ := zng.DecodeFloat64(m.vals.a)
		v2, _ := zng.DecodeFloat64(m.vals.b)
		return zng.Value{typ, m.vals.Float64(v1 * v2)}, nil
	case zng.IsSigned(id):
		v1, _ := zng.DecodeInt(m.vals.a)
		v2, _ := zng.DecodeInt(m.vals.b)
		return zng.Value{typ, m.vals.Int(v1 * v2)}, nil
	case zng.IsNumber(id):
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
	switch {
	case zng.IsFloat(id):
		v1, _ := zng.DecodeFloat64(d.vals.a)
		v2, _ := zng.DecodeFloat64(d.vals.b)
		if v2 == 0 {
			return d.zctx.NewError("floating point divide by 0")
		}
		return zng.Value{typ, d.vals.Float64(v1 / v2)}, nil
	case zng.IsSigned(id):
		v1, _ := zng.DecodeInt(d.vals.a)
		v2, _ := zng.DecodeInt(d.vals.b)
		if v2 == 0 {
			return d.zctx.NewError("signed integer divide by 0")
		}
		return zng.Value{typ, d.vals.Int(v1 / v2)}, nil
	case zng.IsNumber(id):
		v1, _ := zng.DecodeUint(d.vals.a)
		v2, _ := zng.DecodeUint(d.vals.b)
		if v2 == 0 {
			return d.zctx.NewError("unsigned integer divide by 0")
		}
		return zng.Value{typ, d.vals.Uint(v1 / v2)}, nil
	}
	return zng.Value{}, ErrIncompatibleTypes
}

func getNthFromContainer(container zcode.Bytes, idx uint) (zcode.Bytes, error) {
	iter := container.Iter()
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

func lookupKey(mapBytes, target zcode.Bytes) (zcode.Bytes, bool) {
	iter := mapBytes.Iter()
	for !iter.Done() {
		key, _, err := iter.Next()
		if err != nil {
			return nil, false
		}
		val, _, err := iter.Next()
		if err != nil {
			return nil, false
		}
		if bytes.Compare(key, target) == 0 {
			return val, true
		}
	}
	return nil, false
}

// Index represents an index operator "container[index]" where container is
// either an array (with index type integer) or a record (with index type string).
type Index struct {
	zctx      *resolver.Context
	container Evaluator
	index     Evaluator
}

func compileIndexExpr(zctx *resolver.Context, container, index Evaluator) (Evaluator, error) {
	return &Index{zctx, container, index}, nil
}

func (i *Index) Eval(rec *zng.Record) (zng.Value, error) {
	container, err := i.container.Eval(rec)
	if err != nil {
		return zng.Value{}, err
	}
	index, err := i.index.Eval(rec)
	if err != nil {
		return zng.Value{}, err
	}
	switch typ := container.Type.(type) {
	case *zng.TypeArray:
		return indexArray(i.zctx, typ, container.Bytes, index)
	case *zng.TypeRecord:
		return indexRecord(i.zctx, typ, container.Bytes, index)
	case *zng.TypeMap:
		return indexMap(i.zctx, typ, container.Bytes, index)
	default:
		return zng.Value{}, fmt.Errorf("cannot index type \"%s\" with key \"%s\"", typ, index)
	}
}

func indexArray(zctx *resolver.Context, typ *zng.TypeArray, array zcode.Bytes, index zng.Value) (zng.Value, error) {
	id := index.Type.ID()
	if !zng.IsInteger(id) {
		return zctx.NewError("array index is not an integer")
	}
	var idx uint
	if zng.IsSigned(id) {
		v, _ := zng.DecodeInt(index.Bytes)
		if idx < 0 {
			return zctx.NewError("array index out of bounds")
		}
		idx = uint(v)
	} else {
		v, _ := zng.DecodeUint(index.Bytes)
		idx = uint(v)
	}
	zv, err := getNthFromContainer(array, idx)
	if err != nil {
		return zng.Value{}, err
	}
	return zng.Value{typ.Type, zv}, nil
}

func indexRecord(zctx *resolver.Context, typ *zng.TypeRecord, record zcode.Bytes, index zng.Value) (zng.Value, error) {
	id := index.Type.ID()
	if !zng.IsStringy(id) {
		return zctx.NewError("record index is not a string")
	}
	field, _ := zng.DecodeString(index.Bytes)
	result, err := zng.NewRecord(typ, record).ValueByField(string(field))
	if err != nil {
		return zctx.NewError(err.Error())
	}
	return result, nil
}

func indexMap(zctx *resolver.Context, typ *zng.TypeMap, mapBytes zcode.Bytes, key zng.Value) (zng.Value, error) {
	if key.Type != typ.KeyType {
		//XXX should try coercing?
		return zctx.NewError("map key type does not match index type")
	}
	if valBytes, ok := lookupKey(mapBytes, key.Bytes); ok {
		return zng.Value{typ.ValType, valBytes}, nil
	}
	return zctx.NewError("key not found in map: %s", key)
}

type Conditional struct {
	predicate Evaluator
	thenExpr  Evaluator
	elseExpr  Evaluator
}

func compileConditional(zctx *resolver.Context, node ast.ConditionalExpression) (Evaluator, error) {
	var err error
	predicate, err := compileExpr(zctx, node.Condition)
	if err != nil {
		return nil, err
	}
	thenExpr, err := compileExpr(zctx, node.Then)
	if err != nil {
		return nil, err
	}
	elseExpr, err := compileExpr(zctx, node.Else)
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

func compileDotExpr(zctx *resolver.Context, lhs, rhs ast.Expression) (*DotExpr, error) {
	id, ok := rhs.(*ast.Identifier)
	if !ok {
		return nil, errors.New("rhs of dot expression is not an identifier")
	}
	record, err := compileExpr(zctx, lhs)
	if err != nil {
		return nil, err
	}
	return &DotExpr{record, id.Name}, nil
}

type Call struct {
	zctx     *resolver.Context
	name     string
	function Function
	exprs    []Evaluator
	args     *Args
}

func compileCall(zctx *resolver.Context, node ast.FunctionCall) (Evaluator, error) {
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
		e, err := compileExpr(zctx, expr)
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, e)
	}

	return &Call{
		zctx:     zctx,
		name:     node.Function,
		function: fn.impl,
		exprs:    exprs,
		args:     NewArgs(nargs),
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
	return c.function(c.zctx, c.args)
}

func compileCast(zctx *resolver.Context, node ast.CastExpression) (Evaluator, error) {
	expr, err := compileExpr(zctx, node.Expr)
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
		return &StringCast{zctx, expr, zng.TypeString}, nil
	case "bstring":
		return &StringCast{zctx, expr, zng.TypeBstring}, nil
	case "bytes":
		return &BytesCast{expr}, nil
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
	zctx *resolver.Context
	expr Evaluator
	typ  zng.Type
}

func (s *StringCast) Eval(rec *zng.Record) (zng.Value, error) {
	zv, err := s.expr.Eval(rec)
	if err != nil {
		return zng.Value{}, err
	}
	if zv.Type.ID() == zng.IdBytes {
		return zng.Value{s.typ, zng.EncodeString(string(zv.Bytes))}, nil
	}
	if enum, ok := zv.Type.(*zng.TypeEnum); ok {
		selector, _ := zng.DecodeUint(zv.Bytes)
		element, err := enum.Element(int(selector))
		if err != nil {
			return s.zctx.NewError(err.Error())
		}
		return zng.Value{s.typ, zng.EncodeString(element.Name)}, nil
	}
	return zng.Value{s.typ, zng.EncodeString(zv.String())}, nil
}

type BytesCast struct {
	expr Evaluator
}

func (s *BytesCast) Eval(rec *zng.Record) (zng.Value, error) {
	zv, err := s.expr.Eval(rec)
	if err != nil {
		return zng.Value{}, err
	}
	return zng.Value{zng.TypeBytes, zng.EncodeBytes(zv.Bytes)}, nil
}

func CompileLval(node ast.Expression) (field.Static, error) {
	switch node := node.(type) {
	case *ast.RootRecord:
		return field.NewRoot(), nil
	// XXX We need to allow index operators at some point, but for now
	// we have been assuming only dotted field lvalues.  See Issue #1462.
	case *ast.BinaryExpression:
		if node.Operator != "." {
			break
		}
		id, ok := node.RHS.(*ast.Identifier)
		if !ok {
			return nil, errors.New("rhs of dot operator is not an identifier")
		}
		lhs, err := CompileLval(node.LHS)
		if err != nil {
			return nil, err
		}
		return append(lhs, id.Name), nil
	}
	return nil, errors.New("invalid expression on lhs of assignment")
}

func NewRootField(name string) Evaluator {
	return NewDotExpr(field.New(name))
}

var ErrInference = errors.New("assigment name could not be inferred from rhs expressioin")

func CompileAssignment(zctx *resolver.Context, node *ast.Assignment) (field.Static, Evaluator, error) {
	rhs, err := CompileExpr(zctx, node.RHS)
	if err != nil {
		return nil, nil, fmt.Errorf("rhs of assigment expression: %w", err)
	}
	var lhs field.Static
	if node.LHS != nil {
		lhs, err = CompileLval(node.LHS)
		if err != nil {
			return nil, nil, fmt.Errorf("lhs of assigment expression: %w", err)
		}
	} else {
		switch rhs := node.RHS.(type) {
		case *ast.RootRecord:
			lhs = field.New(".")
		case *ast.FunctionCall:
			lhs = field.New(rhs.Function)
		case *ast.BinaryExpression:
			// This can be a dotted record or some other expression.
			// In the latter case, it might be nice to infer a name,
			// e.g., forr "count() by a+b" we could infer "sum" for
			// the name, i,e., "count() by sum=a+b".  But for now,
			// we'll just catch this as an error.
			lhs, err = CompileLval(rhs)
			if err != nil {
				err = ErrInference
			}
		default:
			err = ErrInference
		}
	}
	return lhs, rhs, err
}

func CompileAssignments(dsts []field.Static, srcs []field.Static) ([]field.Static, []Evaluator) {
	if len(srcs) != len(dsts) {
		panic("CompileAssignmentFromStrings argument mismatch")
	}
	var resolvers []Evaluator
	var fields []field.Static
	for k, dst := range dsts {
		fields = append(fields, dst)
		resolvers = append(resolvers, NewDotExpr(srcs[k]))
	}
	return fields, resolvers
}

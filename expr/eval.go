package expr

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr/coerce"
	"github.com/brimdata/zed/expr/function"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/zcode"
)

var ErrIncompatibleTypes = coerce.ErrIncompatibleTypes
var ErrIndexOutOfBounds = errors.New("array index out of bounds")
var ErrNotContainer = errors.New("cannot apply in to a non-container")
var ErrBadCast = errors.New("bad cast")

type Evaluator interface {
	Eval(*zed.Value) (zed.Value, error)
}

type Not struct {
	expr Evaluator
}

func NewLogicalNot(e Evaluator) *Not {
	return &Not{e}
}

func (n *Not) Eval(rec *zed.Value) (zed.Value, error) {
	zv, err := evalBool(n.expr, rec)
	if err != nil {
		return zv, err
	}
	if zed.IsTrue(zv.Bytes) {
		return zed.False, nil
	}
	return zed.True, nil
}

type And struct {
	lhs Evaluator
	rhs Evaluator
}

func NewLogicalAnd(lhs, rhs Evaluator) *And {
	return &And{lhs, rhs}
}

type Or struct {
	lhs Evaluator
	rhs Evaluator
}

func NewLogicalOr(lhs, rhs Evaluator) *Or {
	return &Or{lhs, rhs}
}

func evalBool(e Evaluator, rec *zed.Value) (zed.Value, error) {
	zv, err := e.Eval(rec)
	if err != nil {
		return zv, err
	}
	if zv.Type != zed.TypeBool {
		err = ErrIncompatibleTypes
	}
	return zv, err
}

func (a *And) Eval(rec *zed.Value) (zed.Value, error) {
	lhs, err := evalBool(a.lhs, rec)
	if err != nil {
		return lhs, err
	}
	if !zed.IsTrue(lhs.Bytes) {
		return zed.False, nil
	}
	rhs, err := evalBool(a.rhs, rec)
	if err != nil {
		return lhs, err
	}
	if !zed.IsTrue(rhs.Bytes) {
		return zed.False, nil
	}
	return zed.True, nil
}

func (o *Or) Eval(rec *zed.Value) (zed.Value, error) {
	lhs, err := evalBool(o.lhs, rec)
	if err != nil {
		return lhs, err
	}
	if zed.IsTrue(lhs.Bytes) {
		return zed.True, nil
	}
	rhs, err := evalBool(o.rhs, rec)
	if err != nil {
		return lhs, err
	}
	if zed.IsTrue(rhs.Bytes) {
		return zed.True, nil
	}
	return zed.False, nil
}

type In struct {
	elem      Evaluator
	container Evaluator
	vals      coerce.Pair
}

func NewIn(elem, container Evaluator) *In {
	return &In{
		elem:      elem,
		container: container,
	}
}

func (i *In) Eval(rec *zed.Value) (zed.Value, error) {
	elem, err := i.elem.Eval(rec)
	if err != nil {
		return elem, err
	}
	container, err := i.container.Eval(rec)
	if err != nil {
		return container, err
	}
	switch typ := zed.AliasOf(container.Type).(type) {
	case *zed.TypeOfNet:
		return inNet(elem, container)
	case *zed.TypeArray:
		return i.inContainer(zed.AliasOf(typ.Type), elem, container)
	case *zed.TypeSet:
		return i.inContainer(zed.AliasOf(typ.Type), elem, container)
	case *zed.TypeMap:
		return i.inMap(typ, elem, container)
	default:
		return zed.NewErrorf("'in' operator applied to non-container type"), nil
	}
}

func inNet(elem, net zed.Value) (zed.Value, error) {
	n, err := zed.DecodeNet(net.Bytes)
	if err != nil {
		return zed.Value{}, err
	}
	if typ := zed.AliasOf(elem.Type); typ != zed.TypeIP {
		return zed.NewErrorf("'in' operator applied to non-container type"), nil
	}
	a, err := zed.DecodeIP(elem.Bytes)
	if err != nil {
		return zed.Value{}, err
	}
	if n.IP.Equal(a.Mask(n.Mask)) {
		return zed.True, nil
	}
	return zed.False, nil
}

func (i *In) inContainer(typ zed.Type, elem, container zed.Value) (zed.Value, error) {
	iter := container.Bytes.Iter()
	for {
		if iter.Done() {
			return zed.False, nil
		}
		zv, _, err := iter.Next()
		if err != nil {
			return zed.Value{}, err
		}
		_, err = i.vals.Coerce(elem, zed.Value{typ, zv})
		if err == nil && i.vals.Equal() {
			return zed.True, nil
		}
	}
}

func (i *In) inMap(typ *zed.TypeMap, elem, container zed.Value) (zed.Value, error) {
	keyType := zed.AliasOf(typ.KeyType)
	valType := zed.AliasOf(typ.ValType)
	iter := container.Bytes.Iter()
	for !iter.Done() {
		zv, _, err := iter.Next()
		if err != nil {
			return zed.Value{}, err
		}
		_, err = i.vals.Coerce(elem, zed.Value{keyType, zv})
		if err == nil && i.vals.Equal() {
			return zed.True, nil
		}
		zv, _, err = iter.Next()
		if err != nil {
			return zed.Value{}, err
		}
		_, err = i.vals.Coerce(elem, zed.Value{valType, zv})
		if err == nil && i.vals.Equal() {
			return zed.True, nil
		}
	}
	return zed.False, nil
}

type Equal struct {
	numeric
	equality bool
}

func NewCompareEquality(lhs, rhs Evaluator, operator string) (*Equal, error) {
	e := &Equal{numeric: newNumeric(lhs, rhs)} //XXX
	switch operator {
	case "=":
		e.equality = true
	case "!=":
	default:
		return nil, fmt.Errorf("unknown equality operator: %s", operator)
	}
	return e, nil
}

func (e *Equal) Eval(rec *zed.Value) (zed.Value, error) {
	_, err := e.numeric.eval(rec)
	if err != nil {
		if err == coerce.ErrOverflow {
			// If there was overflow converting one to the other,
			// we know they can't be equal.
			if e.equality {
				return zed.False, nil
			}
			return zed.True, nil
		}
		return zed.Value{}, err
	}
	result := e.vals.Equal()
	if !e.equality {
		result = !result
	}
	if result {
		return zed.True, nil
	}
	return zed.False, nil
}

type RegexpMatch struct {
	re   *regexp.Regexp
	expr Evaluator
}

func NewRegexpMatch(re *regexp.Regexp, e Evaluator) *RegexpMatch {
	return &RegexpMatch{re, e}
}

func (r *RegexpMatch) Eval(rec *zed.Value) (zed.Value, error) {
	zv, err := r.expr.Eval(rec)
	if err != nil {
		return zed.Value{}, err
	}
	if !zed.IsStringy(zv.Type.ID()) {
		return zed.Value{}, zed.ErrMissing
	}
	if r.re.Match(zv.Bytes) {
		return zed.True, nil
	}
	return zed.False, nil
}

type numeric struct {
	zctx *zed.Context
	lhs  Evaluator
	rhs  Evaluator
	vals coerce.Pair
}

func newNumeric(lhs, rhs Evaluator) numeric {
	return numeric{
		lhs: lhs,
		rhs: rhs,
	}
}

func enumify(v zed.Value) (zed.Value, error) {
	// automatically convert an enum to its index value when coercing
	if _, ok := v.Type.(*zed.TypeEnum); ok {
		return zed.Value{zed.TypeUint64, v.Bytes}, nil
	}
	return v, nil
}

func (n *numeric) eval(rec *zed.Value) (int, error) {
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
	return n.vals.Coerce(lhs, rhs)
}

type Compare struct {
	numeric
	convert func(int) bool
}

func NewCompareRelative(lhs, rhs Evaluator, operator string) (*Compare, error) {
	c := &Compare{numeric: newNumeric(lhs, rhs)}
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

func (c *Compare) result(result int) zed.Value {
	if c.convert(result) {
		return zed.True
	}
	return zed.False
}

func (c *Compare) Eval(rec *zed.Value) (zed.Value, error) {
	lhs, err := c.lhs.Eval(rec)
	if err != nil {
		return zed.Value{}, err
	}
	rhs, err := c.rhs.Eval(rec)
	if err != nil {
		return zed.Value{}, err
	}
	id, err := c.vals.Coerce(lhs, rhs)
	if err != nil {
		// If coercion fails due to overflow, then we know there is a
		// mixed signed and unsigned situation and either the unsigned
		// value couldn't be converted to an int64 because it was too big,
		// or the signed value couldn't be converted to a uint64 because
		// it was negative.  In either case, the unsigned value is bigger
		// than the signed value.
		if err == coerce.ErrOverflow {
			result := 1
			if zed.IsSigned(lhs.Type.ID()) {
				result = -1
			}
			return c.result(result), nil
		}
		return zed.False, err
	}
	var result int
	if !c.vals.Equal() {
		switch {
		case c.vals.A == nil || c.vals.B == nil:
			return zed.False, nil
		case zed.IsFloat(id):
			v1, _ := zed.DecodeFloat(c.vals.A)
			v2, _ := zed.DecodeFloat(c.vals.B)
			if v1 < v2 {
				result = -1
			} else {
				result = 1
			}
		case zed.IsSigned(id):
			v1, _ := zed.DecodeInt(c.vals.A)
			v2, _ := zed.DecodeInt(c.vals.B)
			if v1 < v2 {
				result = -1
			} else {
				result = 1
			}
		case zed.IsNumber(id):
			v1, _ := zed.DecodeUint(c.vals.A)
			v2, _ := zed.DecodeUint(c.vals.B)
			if v1 < v2 {
				result = -1
			} else {
				result = 1
			}
		case zed.IsStringy(id):
			v1, _ := zed.DecodeString(c.vals.A)
			v2, _ := zed.DecodeString(c.vals.B)
			if v1 < v2 {
				result = -1
			} else {
				result = 1
			}
		default:
			return zed.Value{}, fmt.Errorf("bad comparison type ID: %d", id)
		}
	}
	if c.convert(result) {
		return zed.True, nil
	}
	return zed.False, nil
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

type Modulo struct {
	numeric
}

// NewArithmetic compiles an expression of the form "expr1 op expr2"
// for the arithmetic operators +, -, *, /
func NewArithmetic(lhs, rhs Evaluator, op string) (Evaluator, error) {
	n := newNumeric(lhs, rhs)
	switch op {
	case "+":
		return &Add{n}, nil
	case "-":
		return &Subtract{n}, nil
	case "*":
		return &Multiply{n}, nil
	case "/":
		return &Divide{n}, nil
	case "%":
		return &Modulo{n}, nil
	}
	return nil, fmt.Errorf("unknown arithmetic operator: %s", op)
}

func (a *Add) Eval(rec *zed.Value) (zed.Value, error) {
	id, err := a.eval(rec)
	if err != nil {
		return zed.Value{}, err
	}
	typ := zed.LookupPrimitiveByID(id)
	switch {
	case zed.IsFloat(id):
		v1, _ := zed.DecodeFloat64(a.vals.A)
		v2, _ := zed.DecodeFloat64(a.vals.B)
		return zed.Value{typ, a.vals.Float64(v1 + v2)}, nil
	case zed.IsSigned(id):
		v1, _ := zed.DecodeInt(a.vals.A)
		v2, _ := zed.DecodeInt(a.vals.B)
		return zed.Value{typ, a.vals.Int(v1 + v2)}, nil
	case zed.IsNumber(id):
		v1, _ := zed.DecodeUint(a.vals.A)
		v2, _ := zed.DecodeUint(a.vals.B)
		return zed.Value{typ, a.vals.Uint(v1 + v2)}, nil
	case zed.IsStringy(id):
		v1, _ := zed.DecodeString(a.vals.A)
		v2, _ := zed.DecodeString(a.vals.B)
		// XXX GC
		return zed.Value{typ, zed.EncodeString(v1 + v2)}, nil
	}
	return zed.Value{}, ErrIncompatibleTypes
}

func (s *Subtract) Eval(rec *zed.Value) (zed.Value, error) {
	id, err := s.eval(rec)
	if err != nil {
		return zed.Value{}, err
	}
	typ := zed.LookupPrimitiveByID(id)
	switch {
	case zed.IsFloat(id):
		v1, _ := zed.DecodeFloat64(s.vals.A)
		v2, _ := zed.DecodeFloat64(s.vals.B)
		return zed.Value{typ, s.vals.Float64(v1 - v2)}, nil
	case zed.IsSigned(id):
		v1, _ := zed.DecodeInt(s.vals.A)
		v2, _ := zed.DecodeInt(s.vals.B)
		return zed.Value{typ, s.vals.Int(v1 - v2)}, nil
	case zed.IsNumber(id):
		v1, _ := zed.DecodeUint(s.vals.A)
		v2, _ := zed.DecodeUint(s.vals.B)
		return zed.Value{typ, s.vals.Uint(v1 - v2)}, nil
	}
	return zed.Value{}, ErrIncompatibleTypes
}

func (m *Multiply) Eval(rec *zed.Value) (zed.Value, error) {
	id, err := m.eval(rec)
	if err != nil {
		return zed.Value{}, err
	}
	typ := zed.LookupPrimitiveByID(id)
	switch {
	case zed.IsFloat(id):
		v1, _ := zed.DecodeFloat64(m.vals.A)
		v2, _ := zed.DecodeFloat64(m.vals.B)
		return zed.Value{typ, m.vals.Float64(v1 * v2)}, nil
	case zed.IsSigned(id):
		v1, _ := zed.DecodeInt(m.vals.A)
		v2, _ := zed.DecodeInt(m.vals.B)
		return zed.Value{typ, m.vals.Int(v1 * v2)}, nil
	case zed.IsNumber(id):
		v1, _ := zed.DecodeUint(m.vals.A)
		v2, _ := zed.DecodeUint(m.vals.B)
		return zed.Value{typ, m.vals.Uint(v1 * v2)}, nil
	}
	return zed.Value{}, ErrIncompatibleTypes
}

func (d *Divide) Eval(rec *zed.Value) (zed.Value, error) {
	id, err := d.eval(rec)
	if err != nil {
		return zed.Value{}, err
	}
	typ := zed.LookupPrimitiveByID(id)
	switch {
	case zed.IsFloat(id):
		v1, _ := zed.DecodeFloat64(d.vals.A)
		v2, _ := zed.DecodeFloat64(d.vals.B)
		if v2 == 0 {
			return zed.NewErrorf("floating point divide by 0"), nil
		}
		return zed.Value{typ, d.vals.Float64(v1 / v2)}, nil
	case zed.IsSigned(id):
		v1, _ := zed.DecodeInt(d.vals.A)
		v2, _ := zed.DecodeInt(d.vals.B)
		if v2 == 0 {
			return zed.NewErrorf("signed integer divide by 0"), nil
		}
		return zed.Value{typ, d.vals.Int(v1 / v2)}, nil
	case zed.IsNumber(id):
		v1, _ := zed.DecodeUint(d.vals.A)
		v2, _ := zed.DecodeUint(d.vals.B)
		if v2 == 0 {
			return zed.NewErrorf("unsigned integer divide by 0"), nil
		}
		return zed.Value{typ, d.vals.Uint(v1 / v2)}, nil
	}
	return zed.Value{}, ErrIncompatibleTypes
}

func (m *Modulo) Eval(zv *zed.Value) (zed.Value, error) {
	id, err := m.eval(zv)
	if err != nil {
		return zed.Value{}, err
	}
	typ := zed.LookupPrimitiveByID(id)
	if zed.IsFloat(id) || !zed.IsNumber(id) {
		return zed.NewErrorf("operator %% not defined on type %s", typ), nil
	}
	if zed.IsSigned(id) {
		x, _ := zed.DecodeInt(m.vals.A)
		y, _ := zed.DecodeInt(m.vals.B)
		if y == 0 {
			return zed.NewErrorf("divide by zero"), nil
		}
		return zed.Value{typ, m.vals.Int(x % y)}, nil
	}
	x, _ := zed.DecodeUint(m.vals.A)
	y, _ := zed.DecodeUint(m.vals.B)
	if y == 0 {
		return zed.NewErrorf("divide by zero"), nil
	}
	return zed.Value{typ, m.vals.Uint(x % y)}, nil
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
	return nil, zed.ErrMissing
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
	zctx      *zed.Context
	container Evaluator
	index     Evaluator
}

func NewIndexExpr(zctx *zed.Context, container, index Evaluator) (Evaluator, error) {
	return &Index{zctx, container, index}, nil
}

func (i *Index) Eval(rec *zed.Value) (zed.Value, error) {
	container, err := i.container.Eval(rec)
	if err != nil {
		return zed.Value{}, err
	}
	index, err := i.index.Eval(rec)
	if err != nil {
		return zed.Value{}, err
	}
	switch typ := container.Type.(type) {
	case *zed.TypeArray:
		return indexArray(typ, container.Bytes, index)
	case *zed.TypeRecord:
		return indexRecord(typ, container.Bytes, index)
	case *zed.TypeMap:
		return indexMap(typ, container.Bytes, index)
	default:
		return zed.Value{}, zed.ErrMissing
	}
}

func indexArray(typ *zed.TypeArray, array zcode.Bytes, index zed.Value) (zed.Value, error) {
	id := index.Type.ID()
	if !zed.IsInteger(id) {
		return zed.NewErrorf("array index is not an integer"), nil
	}
	var idx uint
	if zed.IsSigned(id) {
		v, _ := zed.DecodeInt(index.Bytes)
		if idx < 0 {
			return zed.Value{}, zed.ErrMissing
		}
		idx = uint(v)
	} else {
		v, _ := zed.DecodeUint(index.Bytes)
		idx = uint(v)
	}
	zv, err := getNthFromContainer(array, idx)
	if err != nil {
		return zed.Value{}, err
	}
	return zed.Value{typ.Type, zv}, nil
}

func indexRecord(typ *zed.TypeRecord, record zcode.Bytes, index zed.Value) (zed.Value, error) {
	id := index.Type.ID()
	if !zed.IsStringy(id) {
		return zed.NewErrorf("record index is not a string"), nil
	}
	field, _ := zed.DecodeString(index.Bytes)
	result, err := zed.NewValue(typ, record).ValueByField(string(field))
	if err != nil {
		return zed.NewError(err), nil
	}
	return result, nil
}

func indexMap(typ *zed.TypeMap, mapBytes zcode.Bytes, key zed.Value) (zed.Value, error) {
	if key.Type != typ.KeyType {
		//XXX should try coercing?
		return zed.NewErrorf("map key type does not match index type"), nil
	}
	if valBytes, ok := lookupKey(mapBytes, key.Bytes); ok {
		return zed.Value{typ.ValType, valBytes}, nil
	}
	return zed.Value{}, zed.ErrMissing
}

type Conditional struct {
	predicate Evaluator
	thenExpr  Evaluator
	elseExpr  Evaluator
}

func NewConditional(predicate, thenExpr, elseExpr Evaluator) *Conditional {
	return &Conditional{
		predicate: predicate,
		thenExpr:  thenExpr,
		elseExpr:  elseExpr,
	}
}

func (c *Conditional) Eval(rec *zed.Value) (zed.Value, error) {
	val, err := c.predicate.Eval(rec)
	if err != nil {
		return zed.Value{}, err
	}
	if val.Type.ID() != zed.IDBool {
		return zed.Value{}, ErrIncompatibleTypes
	}
	if zed.IsTrue(val.Bytes) {
		return c.thenExpr.Eval(rec)
	}
	return c.elseExpr.Eval(rec)
}

type Call struct {
	zctx    *zed.Context
	fn      function.Interface
	exprs   []Evaluator
	args    []zed.Value
	AddRoot bool
}

func NewCall(zctx *zed.Context, fn function.Interface, exprs []Evaluator) *Call {
	return &Call{
		zctx:  zctx,
		fn:    fn,
		exprs: exprs,
		args:  make([]zed.Value, len(exprs)),
	}
}

func (c *Call) Eval(rec *zed.Value) (zed.Value, error) {
	for k, e := range c.exprs {
		val, err := e.Eval(rec)
		if err != nil {
			return zed.Value{}, err
		}
		c.args[k] = val
	}
	return c.fn.Call(c.args)
}

// A TyepFunc returns a type value of the named type (where the name is
// a Zed typedef).  It returns MISSING if the name doesn't exist.
type TypeFunc struct {
	name string
	zctx *zed.Context
	zv   zed.Value
}

func NewTypeFunc(zctx *zed.Context, name string) *TypeFunc {
	return &TypeFunc{
		name: name,
		zctx: zctx,
	}
}

func (t *TypeFunc) Eval(rec *zed.Value) (zed.Value, error) {
	if t.zv.Bytes == nil {
		typ := t.zctx.LookupTypeDef(t.name)
		if typ == nil {
			return zed.Missing, nil
		}
		t.zv = zed.NewTypeValue(typ)
	}
	return t.zv, nil
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#has
type Has struct {
	exprs []Evaluator
}

func NewHas(exprs []Evaluator) *Has {
	return &Has{exprs}
}

func (h *Has) Eval(rec *zed.Value) (zed.Value, error) {
	for _, e := range h.exprs {
		zv, err := e.Eval(rec)
		if errors.Is(err, zed.ErrMissing) || zed.IsMissing(zv) {
			return zed.False, nil
		}
		if err != nil {
			return zed.Value{}, err
		}
		if zv.Type == zed.TypeError {
			return zv, nil
		}
	}
	return zed.True, nil
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#missing
type Missing struct {
	has *Has
}

func NewMissing(exprs []Evaluator) *Missing {
	return &Missing{NewHas(exprs)}
}

func (m *Missing) Eval(rec *zed.Value) (zed.Value, error) {
	zv, err := m.has.Eval(rec)
	if zv.Type == zed.TypeBool {
		zv = zed.Not(zv.Bytes)
	}
	return zv, err
}

func NewCast(expr Evaluator, typ zed.Type) (Evaluator, error) {
	// XXX should handle alias casts... need type context.
	// compile is going to need a local type context to create literals
	// of complex types?
	c := LookupPrimitiveCaster(typ)
	if c == nil {
		// XXX See issue #1572.   To implement aliascast here.
		return nil, fmt.Errorf("cast to '%s' not implemented", typ)
	}
	return &evalCast{expr, c, typ}, nil
}

type evalCast struct {
	expr   Evaluator
	caster PrimitiveCaster
	typ    zed.Type
}

func (c *evalCast) Eval(rec *zed.Value) (zed.Value, error) {
	zv, err := c.expr.Eval(rec)
	if err != nil {
		return zed.Value{}, err
	}
	if zv.Bytes == nil {
		// Take care of null here so the casters don't have to
		// worry about it.  Any value can be null after all.
		return zed.Value{c.typ, nil}, nil
	}
	return c.caster(zv)
}

func NewRootField(name string) Evaluator {
	return NewDotExpr(field.New(name))
}

type Assignment struct {
	LHS field.Path
	RHS Evaluator
}

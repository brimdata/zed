package expr

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr/coerce"
	"github.com/brimdata/zed/expr/function"
	"github.com/brimdata/zed/expr/result"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

//XXX remove ErrBadCast and other unused Errs

//XXX git rid of these?
var ErrIncompatibleTypes = coerce.ErrIncompatibleTypes
var ErrIndexOutOfBounds = errors.New("array index out of bounds")
var ErrNotContainer = errors.New("cannot apply in to a non-container")
var ErrBadCast = errors.New("bad cast")

type Evaluator interface {
	Eval(Context, *zed.Value) *zed.Value
}

type Not struct {
	expr Evaluator
}

var _ Evaluator = (*Not)(nil)

func NewLogicalNot(e Evaluator) *Not {
	return &Not{e}
}

func (n *Not) Eval(ctx Context, val *zed.Value) *zed.Value {
	zv := evalBool(ctx, n.expr, val)
	if zed.IsTrue(zv.Bytes) {
		return zed.False
	}
	return zed.True
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

func evalBool(ctx Context, e Evaluator, rec *zed.Value) *zed.Value {
	val := e.Eval(ctx, rec)
	if zed.AliasOf(val.Type) != zed.TypeBool {
		return ctx.CopyValue(zed.NewErrorf("not a boolean: %s", zson.MustFormatValue(*val)))
	}
	return val
}

func (a *And) Eval(ctx Context, this *zed.Value) *zed.Value {
	lhs := evalBool(ctx, a.lhs, this)
	if lhs.IsError() {
		return lhs
	}
	if !zed.IsTrue(lhs.Bytes) {
		return zed.False
	}
	return evalBool(ctx, a.rhs, this)
}

func (o *Or) Eval(ctx Context, this *zed.Value) *zed.Value {
	lhs := evalBool(ctx, o.lhs, this)
	if lhs.IsError() {
		return lhs
	}
	if zed.IsTrue(lhs.Bytes) {
		return zed.True
	}
	return evalBool(ctx, o.rhs, this)
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

func (i *In) Eval(ctx Context, this *zed.Value) *zed.Value {
	elem := i.elem.Eval(ctx, this)
	if elem.IsError() {
		return elem
	}
	container := i.container.Eval(ctx, this)
	if container.IsError() {
		return container
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
		//XXX
		v := zed.NewErrorf("'in' operator applied to non-container type")
		return &v
	}
}

func inNet(elem, net *zed.Value) *zed.Value {
	n, err := zed.DecodeNet(net.Bytes)
	if err != nil {
		panic(err)
	}
	if typ := zed.AliasOf(elem.Type); typ != zed.TypeIP {
		//XXX
		v := zed.NewErrorf("'in' operator applied to non-container type")
		return &v
	}
	a, err := zed.DecodeIP(elem.Bytes)
	if err != nil {
		panic(err)
	}
	if n.IP.Equal(a.Mask(n.Mask)) {
		return zed.True
	}
	return zed.False
}

func (i *In) inContainer(typ zed.Type, elem, container *zed.Value) *zed.Value {
	iter := container.Bytes.Iter()
	for {
		if iter.Done() {
			return zed.False
		}
		zv, _, err := iter.Next()
		if err != nil {
			panic(err)
		}
		_, err = i.vals.Coerce(elem, &zed.Value{typ, zv})
		if err == nil && i.vals.Equal() {
			return zed.True
		}
	}
}

func (i *In) inMap(typ *zed.TypeMap, elem, container *zed.Value) *zed.Value {
	keyType := zed.AliasOf(typ.KeyType)
	valType := zed.AliasOf(typ.ValType)
	iter := container.Bytes.Iter()
	for !iter.Done() {
		zv, _, err := iter.Next()
		if err != nil {
			panic(err)
		}
		_, err = i.vals.Coerce(elem, &zed.Value{keyType, zv})
		if err == nil && i.vals.Equal() {
			return zed.True
		}
		zv, _, err = iter.Next()
		if err != nil {
			panic(err)
		}
		_, err = i.vals.Coerce(elem, &zed.Value{valType, zv})
		if err == nil && i.vals.Equal() {
			return zed.True
		}
	}
	return zed.False
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

func (e *Equal) Eval(ctx Context, this *zed.Value) *zed.Value {
	_, err := e.numeric.eval(ctx, this)
	if err != nil {
		//XXX need to compare have coerce return zed error?
		if err == coerce.ErrOverflow {
			// If there was overflow converting one to the other,
			// we know they can't be equal.
			if e.equality {
				return zed.False
			}
			return zed.True
		}
		//XXX panic?
		return zed.False
	}
	result := e.vals.Equal()
	if !e.equality {
		result = !result
	}
	if result {
		return zed.True
	}
	return zed.False
}

type RegexpMatch struct {
	re   *regexp.Regexp
	expr Evaluator
}

func NewRegexpMatch(re *regexp.Regexp, e Evaluator) *RegexpMatch {
	return &RegexpMatch{re, e}
}

func (r *RegexpMatch) Eval(ctx Context, this *zed.Value) *zed.Value {
	zv := r.expr.Eval(ctx, this)
	if !zed.IsStringy(zv.Type.ID()) {
		//XXX change from missing to false right?
		return zed.False
	}
	if r.re.Match(zv.Bytes) {
		return zed.True
	}
	return zed.False
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

func enumify(v *zed.Value) *zed.Value {
	// automatically convert an enum to its index value when coercing
	if _, ok := v.Type.(*zed.TypeEnum); ok {
		return &zed.Value{zed.TypeUint64, v.Bytes}
	}
	return v
}

func (n *numeric) eval(ctx Context, this *zed.Value) (int, error) {
	//XXX need valOf too...
	lhs := n.lhs.Eval(ctx, this)
	if lhs == zed.Missing {
		return 0, zed.ErrMissing
	}
	lhs = enumify(lhs)
	rhs := n.rhs.Eval(ctx, this)
	if rhs == zed.Missing {
		return 0, zed.ErrMissing
	}
	rhs = enumify(rhs)
	return n.vals.Coerce(lhs, rhs)
}

func (n *numeric) floats() (float64, float64) {
	a, err := zed.DecodeFloat(n.vals.A)
	if err != nil {
		panic(err)
	}
	b, err := zed.DecodeFloat(n.vals.B)
	if err != nil {
		panic(err)
	}
	return a, b
}

func (n *numeric) ints() (int64, int64) {
	a, err := zed.DecodeInt(n.vals.A)
	if err != nil {
		panic(err)
	}
	b, err := zed.DecodeInt(n.vals.B)
	if err != nil {
		panic(err)
	}
	return a, b
}

func (n *numeric) uints() (uint64, uint64) {
	a, err := zed.DecodeUint(n.vals.A)
	if err != nil {
		panic(err)
	}
	b, err := zed.DecodeUint(n.vals.B)
	if err != nil {
		panic(err)
	}
	return a, b
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

func (c *Compare) result(result int) *zed.Value {
	if c.convert(result) {
		return zed.True
	}
	return zed.False
}

func (c *Compare) Eval(ctx Context, this *zed.Value) *zed.Value {
	lhs := c.lhs.Eval(ctx, this)
	if lhs.IsError() {
		return lhs
	}
	rhs := c.rhs.Eval(ctx, this)
	if rhs.IsError() {
		return lhs
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
			return c.result(result)
		}
		//XXX what about error?
		return zed.False
	}
	var result int
	if !c.vals.Equal() {
		switch {
		case c.vals.A == nil || c.vals.B == nil:
			return zed.False
		case zed.IsFloat(id):
			v1, v2 := c.floats()
			if v1 < v2 {
				result = -1
			} else {
				result = 1
			}
		case zed.IsSigned(id):
			v1, v2 := c.ints()
			if v1 < v2 {
				result = -1
			} else {
				result = 1
			}
		case zed.IsNumber(id):
			v1, v2 := c.uints()
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
			//XXX
			v := zed.NewErrorf("bad comparison type ID: %d", id)
			return &v
		}
	}
	if c.convert(result) {
		return zed.True
	}
	return zed.False
}

type Add struct {
	operands numeric
	result   result.Value
}

type Subtract struct {
	operands numeric
	result   result.Value
}

type Multiply struct {
	operands numeric
	result   result.Value
}

type Divide struct {
	operands numeric
	result   result.Value
}

type Modulo struct {
	operands numeric
	result   result.Value
}

// XXX put error singletons in one place
var DivideByZero = &zed.Value{Type: zed.TypeError, Bytes: []byte("divide by zero")}

// NewArithmetic compiles an expression of the form "expr1 op expr2"
// for the arithmetic operators +, -, *, /
func NewArithmetic(lhs, rhs Evaluator, op string) (Evaluator, error) {
	n := newNumeric(lhs, rhs)
	switch op {
	case "+":
		return &Add{operands: n}, nil
	case "-":
		return &Subtract{operands: n}, nil
	case "*":
		return &Multiply{operands: n}, nil
	case "/":
		return &Divide{operands: n}, nil
	case "%":
		return &Modulo{operands: n}, nil
	}
	return nil, fmt.Errorf("unknown arithmetic operator: %s", op)
}

func (a *Add) Eval(ctx Context, this *zed.Value) *zed.Value {
	id, err := a.operands.eval(ctx, this)
	if err != nil {
		if err == zed.ErrMissing {
			return zed.Missing
		}
		return a.result.Error(err)
	}
	typ := zed.LookupPrimitiveByID(id)
	switch {
	case zed.IsFloat(id):
		v1, v2 := a.operands.floats()
		return a.result.Float(typ, v1+v2)
	case zed.IsSigned(id):
		v1, v2 := a.operands.ints()
		return a.result.Int(typ, v1+v2)
	case zed.IsNumber(id):
		v1, v2 := a.operands.uints()
		return a.result.Uint(typ, v1+v2)
	case zed.IsStringy(id):
		v1, _ := zed.DecodeString(a.operands.vals.A)
		v2, _ := zed.DecodeString(a.operands.vals.B)
		//XXX stringy going away with structure errors and no bstring
		// XXX GC
		return a.result.String(v1 + v2)
	}
	return a.result.Errorf("type %s incompatible with '+' operator", typ)
}

func (s *Subtract) Eval(ctx Context, this *zed.Value) *zed.Value {
	id, err := s.operands.eval(ctx, this)
	if err != nil {
		if err == zed.ErrMissing {
			return zed.Missing
		}
		return s.result.Error(err)
	}
	typ := zed.LookupPrimitiveByID(id)
	switch {
	case zed.IsFloat(id):
		v1, v2 := s.operands.floats()
		return s.result.Float(typ, v1-v2)
	case zed.IsSigned(id):
		v1, v2 := s.operands.ints()
		return s.result.Int(typ, v1-v2)
	case zed.IsNumber(id):
		v1, v2 := s.operands.uints()
		return s.result.Uint(typ, v1-v2)
	}
	return s.result.Errorf("type %s incompatible with '-' operator", typ)
}

func (m *Multiply) Eval(ctx Context, this *zed.Value) *zed.Value {
	id, err := m.operands.eval(ctx, this)
	if err != nil {
		if err == zed.ErrMissing {
			return zed.Missing
		}
		return m.result.Error(err)
	}
	typ := zed.LookupPrimitiveByID(id)
	switch {
	case zed.IsFloat(id):
		v1, v2 := m.operands.floats()
		return m.result.Float(typ, v1*v2)
	case zed.IsSigned(id):
		v1, v2 := m.operands.ints()
		return m.result.Int(typ, v1*v2)
	case zed.IsNumber(id):
		v1, v2 := m.operands.uints()
		return m.result.Uint(typ, v1*v2)
	}
	return m.result.Errorf("type %s incompatible with '*' operator", typ)
}

func (d *Divide) Eval(ctx Context, this *zed.Value) *zed.Value {
	id, err := d.operands.eval(ctx, this)
	if err != nil {
		if err == zed.ErrMissing {
			return zed.Missing
		}
		return d.result.Error(err)
	}
	typ := zed.LookupPrimitiveByID(id)
	switch {
	case zed.IsFloat(id):
		v1, v2 := d.operands.floats()
		if v2 == 0 {
			return DivideByZero
		}
		return d.result.Float(typ, v1/v2)
	case zed.IsSigned(id):
		v1, v2 := d.operands.ints()
		if v2 == 0 {
			return DivideByZero
		}
		return d.result.Int(typ, v1/v2)
	case zed.IsNumber(id):
		v1, v2 := d.operands.uints()
		if v2 == 0 {
			return DivideByZero
		}
		return d.result.Uint(typ, v1/v2)
	}
	return d.result.Errorf("type %s incompatible with '/' operator", typ)
}

func (m *Modulo) Eval(ctx Context, this *zed.Value) *zed.Value {
	id, err := m.operands.eval(ctx, this)
	if err != nil {
		if err == zed.ErrMissing {
			return zed.Missing
		}
		return m.result.Error(err)
	}
	typ := zed.LookupPrimitiveByID(id)
	if zed.IsFloat(id) || !zed.IsNumber(id) {
		return m.result.Errorf("type %s incompatible with '%%' operator", typ)
	}
	if zed.IsSigned(id) {
		x, y := m.operands.ints()
		if y == 0 {
			return DivideByZero
		}
		return m.result.Int(typ, x%y)
	}
	x, y := m.operands.uints()
	if y == 0 {
		return DivideByZero
	}
	return m.result.Uint(typ, x%y)
}

func getNthFromContainer(container zcode.Bytes, idx uint) zcode.Bytes {
	iter := container.Iter()
	var i uint = 0
	for ; !iter.Done(); i++ {
		zv, _, err := iter.Next()
		if err != nil {
			panic(err)
		}
		if i == idx {
			return zv
		}
	}
	return nil
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

func NewIndexExpr(zctx *zed.Context, container, index Evaluator) Evaluator {
	return &Index{zctx, container, index}
}

func (i *Index) Eval(ctx Context, this *zed.Value) *zed.Value {
	container := i.container.Eval(ctx, this)
	index := i.index.Eval(ctx, this)
	switch typ := container.Type.(type) {
	case *zed.TypeArray:
		return indexArray(ctx, typ, container.Bytes, index)
	case *zed.TypeRecord:
		return indexRecord(ctx, typ, container.Bytes, index)
	case *zed.TypeMap:
		return indexMap(ctx, typ, container.Bytes, index)
	default:
		return zed.Missing
	}
}

func indexArray(ctx Context, typ *zed.TypeArray, array zcode.Bytes, index *zed.Value) *zed.Value {
	id := index.Type.ID()
	if !zed.IsInteger(id) {
		return ctx.CopyValue(zed.NewErrorf("array index is not an integer"))
	}
	var idx uint
	if zed.IsSigned(id) {
		v, _ := zed.DecodeInt(index.Bytes)
		if idx < 0 {
			return zed.Missing
		}
		idx = uint(v)
	} else {
		v, err := zed.DecodeUint(index.Bytes)
		if err != nil {
			panic(err)
		}
		idx = uint(v)
	}
	zv := getNthFromContainer(array, idx)
	if zv == nil {
		return zed.Missing
	}
	return ctx.NewValue(typ.Type, zv)
}

func indexRecord(ctx Context, typ *zed.TypeRecord, record zcode.Bytes, index *zed.Value) *zed.Value {
	id := index.Type.ID()
	if !zed.IsStringy(id) {
		return ctx.CopyValue(zed.NewErrorf("record index is not a string"))
	}
	field, _ := zed.DecodeString(index.Bytes)
	val, err := zed.NewValue(typ, record).ValueByField(string(field))
	if err != nil {
		return zed.Missing
	}
	return ctx.CopyValue(val)
}

func indexMap(ctx Context, typ *zed.TypeMap, mapBytes zcode.Bytes, key *zed.Value) *zed.Value {
	if key == zed.Missing {
		return zed.Missing
	}
	if key.Type != typ.KeyType {
		//XXX coerce numeric index?
		//XXX seems like we should jut return missing here as
		// a wrong-type key is simply not present in the map
		return ctx.CopyValue(zed.NewErrorf("incompatible map key type"))
	}
	if valBytes, ok := lookupKey(mapBytes, key.Bytes); ok {
		return ctx.NewValue(typ.ValType, valBytes)
	}
	return zed.Missing
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

func (c *Conditional) Eval(ctx Context, this *zed.Value) *zed.Value {
	val := c.predicate.Eval(ctx, this)
	if val.Type.ID() != zed.IDBool {
		val := zed.NewErrorf("?-operator: bool predicate required")
		return &val
	}
	if zed.IsTrue(val.Bytes) {
		return c.thenExpr.Eval(ctx, this)
	}
	return c.elseExpr.Eval(ctx, this)
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

func (c *Call) Eval(ctx Context, this *zed.Value) *zed.Value {
	for k, e := range c.exprs {
		c.args[k] = *e.Eval(ctx, this)
	}
	return c.fn.Call(ctx, c.args)
}

// A TyepFunc returns a type value of the named type (where the name is
// a Zed typedef).  It returns MISSING if the name doesn't exist.
type TypeFunc struct {
	name string
	zctx *zed.Context
	val  zed.Value
}

func NewTypeFunc(zctx *zed.Context, name string) *TypeFunc {
	return &TypeFunc{
		name: name,
		zctx: zctx,
	}
}

func (t *TypeFunc) Eval(ctx Context, this *zed.Value) *zed.Value {
	if t.val.Bytes == nil {
		typ := t.zctx.LookupTypeDef(t.name)
		if typ == nil {
			return zed.Missing
		}
		t.val = zed.NewTypeValue(typ)
	}
	return &t.val
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#has
type Has struct {
	exprs []Evaluator
}

func NewHas(exprs []Evaluator) *Has {
	return &Has{exprs}
}

func (h *Has) Eval(ctx Context, this *zed.Value) *zed.Value {
	for _, e := range h.exprs {
		val := e.Eval(ctx, this)
		if val.IsError() {
			if val.IsMissing() || val.IsQuiet() {
				return zed.False
			}
			return val
		}
	}
	return zed.True
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#missing
type Missing struct {
	has *Has
}

func NewMissing(exprs []Evaluator) *Missing {
	//XXX not has
	return &Missing{NewHas(exprs)}
}

func (m *Missing) Eval(ctx Context, this *zed.Value) *zed.Value {
	val := m.has.Eval(ctx, this)
	if val.Type == zed.TypeBool {
		val = zed.Not(val.Bytes)
	}
	return val
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
	caster Caster
	typ    zed.Type
}

func (c *evalCast) Eval(ctx Context, this *zed.Value) *zed.Value {
	val := c.expr.Eval(ctx, this)
	if val.IsNull() {
		// Take care of null here so the casters don't have to
		// worry about it.  Any value can be null after all.
		return ctx.NewValue(c.typ, nil)
	}
	return c.caster(val)
}

func NewRootField(name string) Evaluator {
	return NewDottedExpr(field.New(name))
}

type Assignment struct {
	LHS field.Path
	RHS Evaluator
}

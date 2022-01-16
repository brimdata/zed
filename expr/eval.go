package expr

import (
	"bytes"
	"fmt"
	"regexp"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr/coerce"
	"github.com/brimdata/zed/expr/function"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

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

func (n *Not) Eval(ectx Context, this *zed.Value) *zed.Value {
	val, ok := EvalBool(ectx, this, n.expr)
	if !ok {
		return val
	}
	if val.Bytes != nil && zed.IsTrue(val.Bytes) {
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

// EvalBool evaluates e with this and if the result is a Zed bool, returns the
// result and true.  Otherwise, a Zed error (inclusive of missing) and false
// are returned.
func EvalBool(ectx Context, this *zed.Value, e Evaluator) (*zed.Value, bool) {
	val := e.Eval(ectx, this)
	if zed.TypeUnder(val.Type) == zed.TypeBool {
		return val, true
	}
	if val.IsError() {
		return val, false
	}
	return ectx.CopyValue(*zed.NewErrorf("not type bool: %s", zson.MustFormatValue(*val))), false
}

func (a *And) Eval(ectx Context, this *zed.Value) *zed.Value {
	lhs, ok := EvalBool(ectx, this, a.lhs)
	if !ok {
		return lhs
	}
	if lhs.Bytes == nil || !zed.IsTrue(lhs.Bytes) {
		return zed.False
	}
	rhs, ok := EvalBool(ectx, this, a.rhs)
	if !ok {
		return rhs
	}
	if rhs.Bytes == nil || !zed.IsTrue(rhs.Bytes) {
		return zed.False
	}
	return zed.True
}

func (o *Or) Eval(ectx Context, this *zed.Value) *zed.Value {
	lhs, ok := EvalBool(ectx, this, o.lhs)
	if ok && lhs.Bytes != nil && zed.IsTrue(lhs.Bytes) {
		return zed.True
	}
	if lhs.IsError() && !lhs.IsMissing() {
		return lhs
	}
	rhs, ok := EvalBool(ectx, this, o.rhs)
	if ok {
		if rhs.Bytes != nil && zed.IsTrue(rhs.Bytes) {
			return zed.True
		}
		return zed.False
	}
	return rhs
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

func (i *In) Eval(ectx Context, this *zed.Value) *zed.Value {
	elem := i.elem.Eval(ectx, this)
	if elem.IsError() {
		return elem
	}
	container := i.container.Eval(ectx, this)
	if container.IsError() {
		return container
	}
	switch typ := zed.TypeUnder(container.Type).(type) {
	case *zed.TypeOfNet:
		return inNet(ectx, elem, container)
	case *zed.TypeArray:
		return i.inContainer(zed.TypeUnder(typ.Type), elem, container)
	case *zed.TypeSet:
		return i.inContainer(zed.TypeUnder(typ.Type), elem, container)
	case *zed.TypeMap:
		return i.inMap(typ, elem, container)
	default:
		return ectx.CopyValue(*zed.NewErrorf("'in' operator applied to non-container type"))
	}
}

func inNet(ectx Context, elem, net *zed.Value) *zed.Value {
	n, err := zed.DecodeNet(net.Bytes)
	if err != nil {
		panic(err)
	}
	if typ := zed.TypeUnder(elem.Type); typ != zed.TypeIP {
		return ectx.CopyValue(*zed.NewErrorf("'in' operator applied to non-container type"))
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
		zv, _ := iter.Next()
		if _, errVal := i.vals.Coerce(elem, &zed.Value{typ, zv}); errVal != nil {
			if errVal != coerce.IncompatibleTypes {
				return errVal
			}
		} else if i.vals.Equal() {
			return zed.True
		}
	}
}

func (i *In) inMap(typ *zed.TypeMap, elem, container *zed.Value) *zed.Value {
	keyType := zed.TypeUnder(typ.KeyType)
	valType := zed.TypeUnder(typ.ValType)
	iter := container.Bytes.Iter()
	for !iter.Done() {
		zv, _ := iter.Next()
		if _, errVal := i.vals.Coerce(elem, &zed.Value{keyType, zv}); errVal != nil {
			if errVal != coerce.IncompatibleTypes {
				return errVal
			}
		} else if i.vals.Equal() {
			return zed.True
		}
		zv, _ = iter.Next()
		if _, errVal := i.vals.Coerce(elem, &zed.Value{valType, zv}); errVal != nil {
			if errVal != coerce.IncompatibleTypes {
				return errVal
			}
		} else if i.vals.Equal() {
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

func (e *Equal) Eval(ectx Context, this *zed.Value) *zed.Value {
	if _, err := e.numeric.eval(ectx, this); err != nil {
		switch err {
		case coerce.Overflow:
			// If there was overflow converting one to the other,
			// we know they can't be equal.
			if e.equality {
				return zed.False
			}
			return zed.True
		case coerce.IncompatibleTypes:
			return zed.False
		default:
			return err
		}
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

func (r *RegexpMatch) Eval(ectx Context, this *zed.Value) *zed.Value {
	val := r.expr.Eval(ectx, this)
	if zed.IsStringy(val.Type.ID()) && r.re.Match(val.Bytes) {
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

func enumify(val *zed.Value) *zed.Value {
	// automatically convert an enum to its index value when coercing
	if _, ok := val.Type.(*zed.TypeEnum); ok {
		return &zed.Value{zed.TypeUint64, val.Bytes}
	}
	return val
}

func (n *numeric) eval(ectx Context, this *zed.Value) (int, *zed.Value) {
	lhs := n.lhs.Eval(ectx, this)
	if lhs.IsError() {
		return 0, lhs
	}
	lhs = enumify(lhs)
	rhs := n.rhs.Eval(ectx, this)
	if rhs.IsError() {
		return 0, rhs
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

func (c *Compare) Eval(ectx Context, this *zed.Value) *zed.Value {
	lhs := c.lhs.Eval(ectx, this)
	if lhs.IsError() {
		return lhs

	}
	rhs := c.rhs.Eval(ectx, this)
	if rhs.IsError() {
		return rhs
	}
	id, err := c.vals.Coerce(lhs, rhs)
	if err != nil {
		// If coercion fails due to overflow, then we know there is a
		// mixed signed and unsigned situation and either the unsigned
		// value couldn't be converted to an int64 because it was too big,
		// or the signed value couldn't be converted to a uint64 because
		// it was negative.  In either case, the unsigned value is bigger
		// than the signed value.
		if err == coerce.Overflow {
			result := 1
			if zed.IsSigned(lhs.Type.ID()) {
				result = -1
			}
			return c.result(result)
		}
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
			return ectx.CopyValue(*zed.NewErrorf("bad comparison type ID: %d", id))
		}
	}
	if c.convert(result) {
		return zed.True
	}
	return zed.False
}

type Add struct {
	operands numeric
}

type Subtract struct {
	operands numeric
}

type Multiply struct {
	operands numeric
}

type Divide struct {
	operands numeric
}

type Modulo struct {
	operands numeric
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

func (a *Add) Eval(ectx Context, this *zed.Value) *zed.Value {
	id, err := a.operands.eval(ectx, this)
	if err != nil {
		return err
	}
	typ := zed.LookupPrimitiveByID(id)
	switch {
	case zed.IsFloat(id):
		v1, v2 := a.operands.floats()
		return ectx.NewValue(typ, zed.EncodeFloat64(v1+v2))
	case zed.IsSigned(id):
		v1, v2 := a.operands.ints()
		return ectx.NewValue(typ, zed.EncodeInt(v1+v2))
	case zed.IsNumber(id):
		v1, v2 := a.operands.uints()
		return ectx.NewValue(typ, zed.EncodeUint(v1+v2))
	case zed.IsStringy(id):
		v1, _ := zed.DecodeString(a.operands.vals.A)
		v2, _ := zed.DecodeString(a.operands.vals.B)
		//XXX stringy going away with structure errors
		// XXX GC
		return ectx.NewValue(typ, zed.EncodeString(v1+v2))
	}
	return ectx.CopyValue(*zed.NewErrorf("type %s incompatible with '+' operator", typ))
}

func (s *Subtract) Eval(ectx Context, this *zed.Value) *zed.Value {
	id, err := s.operands.eval(ectx, this)
	if err != nil {
		return err
	}
	typ := zed.LookupPrimitiveByID(id)
	switch {
	case zed.IsFloat(id):
		v1, v2 := s.operands.floats()
		return ectx.NewValue(typ, zed.EncodeFloat64(v1-v2))
	case zed.IsSigned(id):
		v1, v2 := s.operands.ints()
		return ectx.NewValue(typ, zed.EncodeInt(v1-v2))
	case zed.IsNumber(id):
		v1, v2 := s.operands.uints()
		return ectx.NewValue(typ, zed.EncodeUint(v1-v2))
	}
	return ectx.CopyValue(*zed.NewErrorf("type %s incompatible with '-' operator", typ))
}

func (m *Multiply) Eval(ectx Context, this *zed.Value) *zed.Value {
	id, err := m.operands.eval(ectx, this)
	if err != nil {
		return err
	}
	typ := zed.LookupPrimitiveByID(id)
	switch {
	case zed.IsFloat(id):
		v1, v2 := m.operands.floats()
		return ectx.NewValue(typ, zed.EncodeFloat64(v1*v2))
	case zed.IsSigned(id):
		v1, v2 := m.operands.ints()
		return ectx.NewValue(typ, zed.EncodeInt(v1*v2))
	case zed.IsNumber(id):
		v1, v2 := m.operands.uints()
		return ectx.NewValue(typ, zed.EncodeUint(v1*v2))
	}
	return ectx.CopyValue(*zed.NewErrorf("type %s incompatible with '*' operator", typ))
}

func (d *Divide) Eval(ectx Context, this *zed.Value) *zed.Value {
	id, err := d.operands.eval(ectx, this)
	if err != nil {
		return err
	}
	typ := zed.LookupPrimitiveByID(id)
	switch {
	case zed.IsFloat(id):
		v1, v2 := d.operands.floats()
		if v2 == 0 {
			return DivideByZero
		}
		return ectx.NewValue(typ, zed.EncodeFloat64(v1/v2))
	case zed.IsSigned(id):
		v1, v2 := d.operands.ints()
		if v2 == 0 {
			return DivideByZero
		}
		return ectx.NewValue(typ, zed.EncodeInt(v1/v2))
	case zed.IsNumber(id):
		v1, v2 := d.operands.uints()
		if v2 == 0 {
			return DivideByZero
		}
		return ectx.NewValue(typ, zed.EncodeUint(v1/v2))
	}
	return ectx.CopyValue(*zed.NewErrorf("type %s incompatible with '/' operator", typ))
}

func (m *Modulo) Eval(ectx Context, this *zed.Value) *zed.Value {
	id, err := m.operands.eval(ectx, this)
	if err != nil {
		return err
	}
	typ := zed.LookupPrimitiveByID(id)
	if zed.IsFloat(id) || !zed.IsNumber(id) {
		return ectx.CopyValue(*zed.NewErrorf("type %s incompatible with '%%' operator", typ))
	}
	if zed.IsSigned(id) {
		x, y := m.operands.ints()
		if y == 0 {
			return DivideByZero
		}
		return ectx.NewValue(typ, zed.EncodeInt(x%y))
	}
	x, y := m.operands.uints()
	if y == 0 {
		return DivideByZero
	}
	return ectx.NewValue(typ, zed.EncodeUint(x%y))
}

func getNthFromContainer(container zcode.Bytes, idx uint) zcode.Bytes {
	iter := container.Iter()
	var i uint = 0
	for ; !iter.Done(); i++ {
		zv, _ := iter.Next()
		if i == idx {
			return zv
		}
	}
	return nil
}

func lookupKey(mapBytes, target zcode.Bytes) (zcode.Bytes, bool) {
	iter := mapBytes.Iter()
	for !iter.Done() {
		key, _ := iter.Next()
		val, _ := iter.Next()
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

func (i *Index) Eval(ectx Context, this *zed.Value) *zed.Value {
	container := i.container.Eval(ectx, this)
	index := i.index.Eval(ectx, this)
	switch typ := container.Type.(type) {
	case *zed.TypeArray:
		return indexArray(ectx, typ, container.Bytes, index)
	case *zed.TypeRecord:
		return indexRecord(ectx, typ, container.Bytes, index)
	case *zed.TypeMap:
		return indexMap(ectx, typ, container.Bytes, index)
	default:
		return zed.Missing
	}
}

func indexArray(ectx Context, typ *zed.TypeArray, array zcode.Bytes, index *zed.Value) *zed.Value {
	id := index.Type.ID()
	if !zed.IsInteger(id) {
		return ectx.CopyValue(*zed.NewErrorf("array index is not an integer"))
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
	return ectx.NewValue(typ.Type, zv)
}

func indexRecord(ectx Context, typ *zed.TypeRecord, record zcode.Bytes, index *zed.Value) *zed.Value {
	id := index.Type.ID()
	if !zed.IsStringy(id) {
		return ectx.CopyValue(*zed.NewErrorf("record index is not a string"))
	}
	field, _ := zed.DecodeString(index.Bytes)
	val, err := zed.NewValue(typ, record).ValueByField(string(field))
	if err != nil {
		return zed.Missing
	}
	return ectx.CopyValue(val)
}

func indexMap(ectx Context, typ *zed.TypeMap, mapBytes zcode.Bytes, key *zed.Value) *zed.Value {
	if key.IsMissing() {
		return zed.Missing
	}
	if key.Type != typ.KeyType {
		// XXX issue #3360
		return zed.Missing
	}
	if valBytes, ok := lookupKey(mapBytes, key.Bytes); ok {
		return ectx.NewValue(typ.ValType, valBytes)
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

func (c *Conditional) Eval(ectx Context, this *zed.Value) *zed.Value {
	val := c.predicate.Eval(ectx, this)
	if val.Type.ID() != zed.IDBool {
		val := *zed.NewErrorf("?-operator: bool predicate required")
		return &val
	}
	if zed.IsTrue(val.Bytes) {
		return c.thenExpr.Eval(ectx, this)
	}
	return c.elseExpr.Eval(ectx, this)
}

type Call struct {
	zctx  *zed.Context
	fn    function.Interface
	exprs []Evaluator
	args  []zed.Value
}

func NewCall(zctx *zed.Context, fn function.Interface, exprs []Evaluator) *Call {
	return &Call{
		zctx:  zctx,
		fn:    fn,
		exprs: exprs,
		args:  make([]zed.Value, len(exprs)),
	}
}

func (c *Call) Eval(ectx Context, this *zed.Value) *zed.Value {
	for k, e := range c.exprs {
		c.args[k] = *e.Eval(ectx, this)
	}
	return c.fn.Call(ectx, c.args)
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#has
type Has struct {
	exprs []Evaluator
}

func NewHas(exprs []Evaluator) *Has {
	return &Has{exprs}
}

func (h *Has) Eval(ectx Context, this *zed.Value) *zed.Value {
	for _, e := range h.exprs {
		val := e.Eval(ectx, this)
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
	return &Missing{NewHas(exprs)}
}

func (m *Missing) Eval(ectx Context, this *zed.Value) *zed.Value {
	val := m.has.Eval(ectx, this)
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

func (c *evalCast) Eval(ectx Context, this *zed.Value) *zed.Value {
	val := c.expr.Eval(ectx, this)
	if val.IsNull() {
		// Take care of null here so the casters don't have to
		// worry about it.  Any value can be null after all.
		return ectx.NewValue(c.typ, nil)
	}
	return c.caster(ectx, val)
}

type Assignment struct {
	LHS field.Path
	RHS Evaluator
}

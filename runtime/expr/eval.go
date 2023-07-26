package expr

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"regexp"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime/expr/coerce"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
	"golang.org/x/exp/constraints"
)

type Evaluator interface {
	Eval(Context, *zed.Value) *zed.Value
}

type Function interface {
	Call(zed.Allocator, []zed.Value) *zed.Value
}

type Not struct {
	zctx *zed.Context
	expr Evaluator
}

var _ Evaluator = (*Not)(nil)

func NewLogicalNot(zctx *zed.Context, e Evaluator) *Not {
	return &Not{zctx, e}
}

func (n *Not) Eval(ectx Context, this *zed.Value) *zed.Value {
	val, ok := EvalBool(n.zctx, ectx, this, n.expr)
	if !ok {
		return val
	}
	if val.Bool() {
		return zed.False
	}
	return zed.True
}

type And struct {
	zctx *zed.Context
	lhs  Evaluator
	rhs  Evaluator
}

func NewLogicalAnd(zctx *zed.Context, lhs, rhs Evaluator) *And {
	return &And{zctx, lhs, rhs}
}

type Or struct {
	zctx *zed.Context
	lhs  Evaluator
	rhs  Evaluator
}

func NewLogicalOr(zctx *zed.Context, lhs, rhs Evaluator) *Or {
	return &Or{zctx, lhs, rhs}
}

// EvalBool evaluates e with this and if the result is a Zed bool, returns the
// result and true.  Otherwise, a Zed error (inclusive of missing) and false
// are returned.
func EvalBool(zctx *zed.Context, ectx Context, this *zed.Value, e Evaluator) (*zed.Value, bool) {
	val := e.Eval(ectx, this)
	if zed.TypeUnder(val.Type) == zed.TypeBool {
		return val, true
	}
	if val.IsError() {
		return val, false
	}
	return ectx.CopyValue(*zctx.WrapError("not type bool", val)), false
}

func (a *And) Eval(ectx Context, this *zed.Value) *zed.Value {
	lhs, ok := EvalBool(a.zctx, ectx, this, a.lhs)
	if !ok {
		return lhs
	}
	if !lhs.Bool() {
		return zed.False
	}
	rhs, ok := EvalBool(a.zctx, ectx, this, a.rhs)
	if !ok {
		return rhs
	}
	if !rhs.Bool() {
		return zed.False
	}
	return zed.True
}

func (o *Or) Eval(ectx Context, this *zed.Value) *zed.Value {
	lhs, ok := EvalBool(o.zctx, ectx, this, o.lhs)
	if ok && lhs.Bool() {
		return zed.True
	}
	if lhs.IsError() && !lhs.IsMissing() {
		return lhs
	}
	rhs, ok := EvalBool(o.zctx, ectx, this, o.rhs)
	if ok {
		if rhs.Bool() {
			return zed.True
		}
		return zed.False
	}
	return rhs
}

type In struct {
	zctx      *zed.Context
	elem      Evaluator
	container Evaluator
}

func NewIn(zctx *zed.Context, elem, container Evaluator) *In {
	return &In{
		zctx:      zctx,
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
	tmpVal := ectx.NewValue(nil, nil)
	err := container.Walk(func(typ zed.Type, body zcode.Bytes) error {
		*tmpVal = *zed.NewValue(typ, body)
		if coerce.Equal(elem, tmpVal) {
			return errMatch
		}
		return nil
	})
	switch err {
	case errMatch:
		return zed.True
	case nil:
		return zed.False
	default:
		return i.zctx.NewError(err)
	}
}

type Equal struct {
	numeric
	equality bool
}

func NewCompareEquality(zctx *zed.Context, lhs, rhs Evaluator, operator string) (*Equal, error) {
	e := &Equal{numeric: newNumeric(zctx, lhs, rhs)} //XXX
	switch operator {
	case "==":
		e.equality = true
	case "!=":
	default:
		return nil, fmt.Errorf("unknown equality operator: %s", operator)
	}
	return e, nil
}

func (e *Equal) Eval(ectx Context, this *zed.Value) *zed.Value {
	lhsVal, rhsVal, errVal := e.numeric.eval(ectx, this)
	if errVal != nil {
		return errVal
	}
	result := coerce.Equal(lhsVal, rhsVal)
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
	if val.Type.ID() == zed.IDString && r.re.Match(val.Bytes()) {
		return zed.True
	}
	return zed.False
}

type numeric struct {
	zctx *zed.Context
	lhs  Evaluator
	rhs  Evaluator
}

func newNumeric(zctx *zed.Context, lhs, rhs Evaluator) numeric {
	return numeric{
		zctx: zctx,
		lhs:  lhs,
		rhs:  rhs,
	}
}

func (n *numeric) evalAndPromote(ectx Context, this *zed.Value) (*zed.Value, *zed.Value, int, zed.Type, *zed.Value) {
	lhsVal, rhsVal, errVal := n.eval(ectx, this)
	if errVal != nil {
		return nil, nil, 0, nil, errVal
	}
	id, typ, errVal := n.promote(ectx, lhsVal, rhsVal)
	if errVal != nil {
		return nil, nil, 0, nil, errVal
	}
	return lhsVal, rhsVal, id, typ, nil
}

func enumify(ectx Context, val *zed.Value) *zed.Value {
	// automatically convert an enum to its index value when coercing
	if _, ok := val.Type.(*zed.TypeEnum); ok {
		return ectx.NewValue(zed.TypeUint64, val.Bytes())
	}
	return val
}

func (n *numeric) eval(ectx Context, this *zed.Value) (*zed.Value, *zed.Value, *zed.Value) {
	lhs := n.lhs.Eval(ectx, this)
	if lhs.IsError() {
		return nil, nil, lhs
	}
	rhs := n.rhs.Eval(ectx, this)
	if rhs.IsError() {
		return nil, nil, rhs
	}
	return enumify(ectx, lhs), enumify(ectx, rhs), nil
}

func (n *numeric) promote(ectx Context, lhsVal, rhsVal *zed.Value) (int, zed.Type, *zed.Value) {
	id, err := coerce.Promote(lhsVal, rhsVal)
	if err != nil {
		return 0, nil, ectx.CopyValue(*n.zctx.NewError(err))
	}
	typ, err := zed.LookupPrimitiveByID(id)
	if err != nil {
		return 0, nil, ectx.CopyValue(*n.zctx.NewError(err))
	}
	return id, typ, nil
}

type Compare struct {
	zctx *zed.Context
	numeric
	convert func(int) bool
}

func NewCompareRelative(zctx *zed.Context, lhs, rhs Evaluator, operator string) (*Compare, error) {
	c := &Compare{zctx: zctx, numeric: newNumeric(zctx, lhs, rhs)}
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

	if lhs.IsNull() {
		if rhs.IsNull() {
			return c.result(0)
		}
		return zed.False
	} else if rhs.IsNull() {
		// We know lhs isn't null.
		return zed.False
	}

	switch lid, rid := lhs.Type.ID(), rhs.Type.ID(); {
	case zed.IsNumber(lid) && zed.IsNumber(rid):
		return c.result(compareNumbers(lhs, rhs, lid, rid))
	// case lid == zed.IDBool && rid == zed.IDBool:
	case lid == zed.IDBytes && rid == zed.IDBytes:
		return c.result(bytes.Compare(zed.DecodeBytes(lhs.Bytes()), zed.DecodeBytes(rhs.Bytes())))
	case lid == zed.IDString && rid == zed.IDString:
		return c.result(compare(zed.DecodeString(lhs.Bytes()), zed.DecodeString(lhs.Bytes())))
	case lid == rid:
		if bytes.Equal(lhs.Bytes(), rhs.Bytes()) {
			return c.result(0)
		}
	}
	return zed.False
}

func compareNumbers(a, b *zed.Value, aid, bid int) int {
	switch {
	case zed.IsFloat(aid):
		return compare(a.Float(), toFloat(b))
	case zed.IsFloat(bid):
		return compare(toFloat(a), b.Float())
	case zed.IsSigned(aid):
		av := a.Int()
		if zed.IsUnsigned(bid) {
			if av < 0 {
				return -1
			}
			return compare(uint64(av), b.Uint())
		}
		return compare(av, b.Int())
	case zed.IsSigned(bid):
		bv := b.Int()
		if zed.IsUnsigned(aid) {
			if bv < 0 {
				return 1
			}
			return compare(a.Uint(), uint64(bv))
		}
		return compare(a.Int(), bv)
	}
	return compare(a.Uint(), b.Uint())
}

func compare[T constraints.Ordered](a, b T) int {
	if a < b {
		return -1
	} else if a > b {
		return 1
	}
	return 0
}

func toFloat(val *zed.Value) float64 { return coerce.ToNumeric[float64](val) }
func toInt(val *zed.Value) int64     { return coerce.ToNumeric[int64](val) }

type Add struct {
	zctx     *zed.Context
	operands numeric
}

type Subtract struct {
	zctx     *zed.Context
	operands numeric
}

type Multiply struct {
	zctx     *zed.Context
	operands numeric
}

type Divide struct {
	zctx     *zed.Context
	operands numeric
}

type Modulo struct {
	zctx     *zed.Context
	operands numeric
}

var DivideByZero = errors.New("divide by zero")

// NewArithmetic compiles an expression of the form "expr1 op expr2"
// for the arithmetic operators +, -, *, /
func NewArithmetic(zctx *zed.Context, lhs, rhs Evaluator, op string) (Evaluator, error) {
	n := newNumeric(zctx, lhs, rhs)
	switch op {
	case "+":
		return &Add{zctx: zctx, operands: n}, nil
	case "-":
		return &Subtract{zctx: zctx, operands: n}, nil
	case "*":
		return &Multiply{zctx: zctx, operands: n}, nil
	case "/":
		return &Divide{zctx: zctx, operands: n}, nil
	case "%":
		return &Modulo{zctx: zctx, operands: n}, nil
	}
	return nil, fmt.Errorf("unknown arithmetic operator: %s", op)
}

func (a *Add) Eval(ectx Context, this *zed.Value) *zed.Value {
	lhsVal, rhsVal, id, typ, errVal := a.operands.evalAndPromote(ectx, this)
	if errVal != nil {
		return errVal
	}
	switch {
	case zed.IsUnsigned(id):
		return ectx.CopyValue(*zed.NewUint(typ, lhsVal.Uint()+rhsVal.Uint()))
	case zed.IsSigned(id):
		return ectx.CopyValue(*zed.NewInt(typ, toInt(lhsVal)+toInt(rhsVal)))
	case zed.IsFloat(id):
		return ectx.CopyValue(*zed.NewFloat(typ, toFloat(lhsVal)+toFloat(rhsVal)))
	case id == zed.IDString:
		v1, v2 := zed.DecodeString(lhsVal.Bytes()), zed.DecodeString(rhsVal.Bytes())
		// XXX GC
		return ectx.NewValue(typ, zed.EncodeString(v1+v2))
	}
	return ectx.CopyValue(*a.zctx.NewErrorf("type %s incompatible with '+' operator", zson.FormatType(typ)))
}

func (s *Subtract) Eval(ectx Context, this *zed.Value) *zed.Value {
	lhsVal, rhsVal, id, typ, errVal := s.operands.evalAndPromote(ectx, this)
	if errVal != nil {
		return errVal
	}
	switch {
	case zed.IsUnsigned(id):
		return ectx.CopyValue(*zed.NewUint(typ, lhsVal.Uint()-rhsVal.Uint()))
	case zed.IsSigned(id):
		if id == zed.IDTime {
			// Return the difference of two times as a duration.
			typ = zed.TypeDuration
		}
		return ectx.CopyValue(*zed.NewInt(typ, toInt(lhsVal)-toInt(rhsVal)))
	case zed.IsFloat(id):
		return ectx.CopyValue(*zed.NewFloat(typ, toFloat(lhsVal)-toFloat(rhsVal)))
	}
	return ectx.CopyValue(*s.zctx.NewErrorf("type %s incompatible with '-' operator", zson.FormatType(typ)))
}

func (m *Multiply) Eval(ectx Context, this *zed.Value) *zed.Value {
	lhsVal, rhsVal, id, typ, errVal := m.operands.evalAndPromote(ectx, this)
	if errVal != nil {
		return errVal
	}
	switch {
	case zed.IsUnsigned(id):
		return ectx.CopyValue(*zed.NewUint(typ, lhsVal.Uint()*rhsVal.Uint()))
	case zed.IsSigned(id):
		return ectx.CopyValue(*zed.NewInt(typ, toInt(lhsVal)*toInt(rhsVal)))
	case zed.IsFloat(id):
		return ectx.CopyValue(*zed.NewFloat(typ, toFloat(lhsVal)*toFloat(rhsVal)))
	}
	return ectx.CopyValue(*m.zctx.NewErrorf("type %s incompatible with '*' operator", zson.FormatType(typ)))
}

func (d *Divide) Eval(ectx Context, this *zed.Value) *zed.Value {
	lhsVal, rhsVal, id, typ, errVal := d.operands.evalAndPromote(ectx, this)
	if errVal != nil {
		return errVal
	}
	switch {
	case zed.IsUnsigned(id):
		v := rhsVal.Uint()
		if v == 0 {
			return d.zctx.NewError(DivideByZero)
		}
		return ectx.CopyValue(*zed.NewUint(typ, lhsVal.Uint()/v))
	case zed.IsSigned(id):
		v := toInt(rhsVal)
		if v == 0 {
			return d.zctx.NewError(DivideByZero)
		}
		return ectx.CopyValue(*zed.NewInt(typ, toInt(lhsVal)/v))
	case zed.IsFloat(id):
		v := toFloat(rhsVal)
		if v == 0 {
			return d.zctx.NewError(DivideByZero)
		}
		return ectx.CopyValue(*zed.NewFloat(typ, toFloat(lhsVal)/v))
	}
	return ectx.CopyValue(*d.zctx.NewErrorf("type %s incompatible with '/' operator", zson.FormatType(typ)))
}

func (m *Modulo) Eval(ectx Context, this *zed.Value) *zed.Value {
	lhsVal, rhsVal, id, typ, errVal := m.operands.evalAndPromote(ectx, this)
	if errVal != nil {
		return errVal
	}
	switch {
	case zed.IsUnsigned(id):
		v := rhsVal.Uint()
		if v == 0 {
			return m.zctx.NewError(DivideByZero)
		}
		return ectx.CopyValue(*zed.NewUint(typ, lhsVal.Uint()%v))
	case zed.IsSigned(id):
		v := toInt(rhsVal)
		if v == 0 {
			return m.zctx.NewError(DivideByZero)
		}
		return ectx.CopyValue(*zed.NewInt(typ, toInt(lhsVal)%v))
	}
	return ectx.CopyValue(*m.zctx.NewErrorf("type %s incompatible with '%%' operator", zson.FormatType(typ)))
}

type UnaryMinus struct {
	zctx *zed.Context
	expr Evaluator
}

func NewUnaryMinus(zctx *zed.Context, e Evaluator) *UnaryMinus {
	return &UnaryMinus{
		zctx: zctx,
		expr: e,
	}
}

func (u *UnaryMinus) Eval(ectx Context, this *zed.Value) *zed.Value {
	val := u.expr.Eval(ectx, this)
	typ := val.Type
	if val.IsNull() && zed.IsNumber(typ.ID()) {
		return val
	}
	switch typ.ID() {
	case zed.IDFloat16, zed.IDFloat32, zed.IDFloat64:
		return ectx.CopyValue(*zed.NewFloat(typ, -val.Float()))
	case zed.IDInt8:
		v := val.Int()
		if v == math.MinInt8 {
			return ectx.CopyValue(*u.zctx.WrapError("unary '-' underflow", val))
		}
		return ectx.CopyValue(*zed.NewInt8(int8(-v)))
	case zed.IDInt16:
		v := val.Int()
		if v == math.MinInt16 {
			return ectx.CopyValue(*u.zctx.WrapError("unary '-' underflow", val))
		}
		return ectx.CopyValue(*zed.NewInt16(int16(-v)))
	case zed.IDInt32:
		v := val.Int()
		if v == math.MinInt32 {
			return ectx.CopyValue(*u.zctx.WrapError("unary '-' underflow", val))
		}
		return ectx.CopyValue(*zed.NewInt32(int32(-v)))
	case zed.IDInt64:
		v := val.Int()
		if v == math.MinInt64 {
			return ectx.CopyValue(*u.zctx.WrapError("unary '-' underflow", val))
		}
		return ectx.CopyValue(*zed.NewInt64(-v))
	case zed.IDUint8:
		v := val.Uint()
		if v > math.MaxInt8 {
			return ectx.CopyValue(*u.zctx.WrapError("unary '-' overflow", val))
		}
		return ectx.CopyValue(*zed.NewInt8(int8(-v)))
	case zed.IDUint16:
		v := val.Uint()
		if v > math.MaxInt16 {
			return ectx.CopyValue(*u.zctx.WrapError("unary '-' overflow", val))
		}
		return ectx.CopyValue(*zed.NewInt16(int16(-v)))
	case zed.IDUint32:
		v := val.Uint()
		if v > math.MaxInt32 {
			return ectx.CopyValue(*u.zctx.WrapError("unary '-' overflow", val))
		}
		return ectx.CopyValue(*zed.NewInt32(int32(-v)))
	case zed.IDUint64:
		v := val.Uint()
		if v > math.MaxInt64 {
			return ectx.CopyValue(*u.zctx.WrapError("unary '-' overflow", val))
		}
		return ectx.CopyValue(*zed.NewInt64(int64(-v)))
	}
	return u.zctx.WrapError("type incompatible with unary '-' operator", val)
}

func getNthFromContainer(container zcode.Bytes, idx int) zcode.Bytes {
	if idx < 0 {
		var length int
		for it := container.Iter(); !it.Done(); it.Next() {
			length++
		}
		idx = length + idx
		if idx < 0 || idx >= length {
			return nil
		}
	}
	for i, it := 0, container.Iter(); !it.Done(); i++ {
		zv := it.Next()
		if i == idx {
			return zv
		}
	}
	return nil
}

func lookupKey(mapBytes, target zcode.Bytes) (zcode.Bytes, bool) {
	for it := mapBytes.Iter(); !it.Done(); {
		key := it.Next()
		val := it.Next()
		if bytes.Equal(key, target) {
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
	switch typ := zed.TypeUnder(container.Type).(type) {
	case *zed.TypeArray, *zed.TypeSet:
		return indexVector(i.zctx, ectx, zed.InnerType(typ), container.Bytes(), index)
	case *zed.TypeRecord:
		return indexRecord(i.zctx, ectx, typ, container.Bytes(), index)
	case *zed.TypeMap:
		return indexMap(i.zctx, ectx, typ, container.Bytes(), index)
	default:
		return i.zctx.Missing()
	}
}

func indexVector(zctx *zed.Context, ectx Context, inner zed.Type, vector zcode.Bytes, index *zed.Value) *zed.Value {
	id := index.Type.ID()
	if !zed.IsInteger(id) {
		return ectx.CopyValue(*zctx.WrapError("array index is not an integer", index))
	}
	var idx int
	if zed.IsSigned(id) {
		idx = int(index.Int())
	} else {
		idx = int(index.Uint())
	}
	zv := getNthFromContainer(vector, idx)
	if zv == nil {
		return zctx.Missing()
	}
	return deunion(ectx, inner, zv)
}

func indexRecord(zctx *zed.Context, ectx Context, typ *zed.TypeRecord, record zcode.Bytes, index *zed.Value) *zed.Value {
	id := index.Type.ID()
	if id != zed.IDString {
		return ectx.CopyValue(*zctx.WrapError("record index is not a string", index))
	}
	field := zed.DecodeString(index.Bytes())
	val := ectx.NewValue(typ, record).Deref(field)
	if val == nil {
		return zctx.Missing()
	}
	return ectx.CopyValue(*val)
}

func indexMap(zctx *zed.Context, ectx Context, typ *zed.TypeMap, mapBytes zcode.Bytes, key *zed.Value) *zed.Value {
	if key.IsMissing() {
		return zctx.Missing()
	}
	if key.Type != typ.KeyType {
		if union, ok := zed.TypeUnder(typ.KeyType).(*zed.TypeUnion); ok {
			if tag := union.TagOf(key.Type); tag >= 0 {
				var b zcode.Builder
				zed.BuildUnion(&b, union.TagOf(key.Type), key.Bytes())
				if valBytes, ok := lookupKey(mapBytes, b.Bytes().Body()); ok {
					return deunion(ectx, typ.ValType, valBytes)
				}
			}
		}
		return zctx.Missing()
	}
	if valBytes, ok := lookupKey(mapBytes, key.Bytes()); ok {
		return deunion(ectx, typ.ValType, valBytes)
	}
	return zctx.Missing()
}

func deunion(ectx Context, typ zed.Type, b zcode.Bytes) *zed.Value {
	if union, ok := typ.(*zed.TypeUnion); ok {
		typ, b = union.Untag(b)
	}
	return ectx.NewValue(typ, b)
}

type Conditional struct {
	zctx      *zed.Context
	predicate Evaluator
	thenExpr  Evaluator
	elseExpr  Evaluator
}

func NewConditional(zctx *zed.Context, predicate, thenExpr, elseExpr Evaluator) *Conditional {
	return &Conditional{
		zctx:      zctx,
		predicate: predicate,
		thenExpr:  thenExpr,
		elseExpr:  elseExpr,
	}
}

func (c *Conditional) Eval(ectx Context, this *zed.Value) *zed.Value {
	val := c.predicate.Eval(ectx, this)
	if val.Type.ID() != zed.IDBool {
		val := *c.zctx.WrapError("?-operator: bool predicate required", val)
		return &val
	}
	if val.Bool() {
		return c.thenExpr.Eval(ectx, this)
	}
	return c.elseExpr.Eval(ectx, this)
}

type Call struct {
	zctx  *zed.Context
	fn    Function
	exprs []Evaluator
	args  []zed.Value
}

func NewCall(zctx *zed.Context, fn Function, exprs []Evaluator) *Call {
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

func NewCast(zctx *zed.Context, expr Evaluator, typ zed.Type) (Evaluator, error) {
	// XXX should handle named type casts. need type context.
	// compile is going to need a local type context to create literals
	// of complex types?
	c := LookupPrimitiveCaster(zctx, typ)
	if c == nil {
		// XXX See issue #1572.  To implement named cast here.
		return nil, fmt.Errorf("cast to %q not implemented", zson.FormatType(typ))
	}
	return &evalCast{expr, c, typ}, nil
}

type evalCast struct {
	expr   Evaluator
	caster Evaluator
	typ    zed.Type
}

func (c *evalCast) Eval(ectx Context, this *zed.Value) *zed.Value {
	val := c.expr.Eval(ectx, this)
	if val.IsNull() || val.Type == c.typ {
		// If value is null or the type won't change, just return a
		// copy of the value.
		return ectx.NewValue(c.typ, val.Bytes())
	}
	return c.caster.Eval(ectx, val)
}

type Assignment struct {
	LHS field.Path
	RHS Evaluator
}

func NewAssignments(zctx *zed.Context, dsts field.List, srcs field.List) (field.List, []Evaluator) {
	if len(srcs) != len(dsts) {
		panic("NewAssignments: argument mismatch")
	}
	var resolvers []Evaluator
	var fields field.List
	for k, dst := range dsts {
		fields = append(fields, dst)
		resolvers = append(resolvers, NewDottedExpr(zctx, srcs[k]))
	}
	return fields, resolvers
}

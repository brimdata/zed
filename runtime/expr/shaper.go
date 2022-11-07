package expr

import (
	"fmt"
	"sort"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
	"golang.org/x/exp/slices"
)

// A ShaperTransform represents one of the different transforms that a
// shaper can apply.  The transforms are represented as bit flags that
// can be bitwise-ored together to create a single shaping operator that
// represents the composition of all operators.  This composition is efficient
// as it is created once per incoming type and then the resulting
// operator is run for every value of that type.
type ShaperTransform int

const (
	Cast ShaperTransform = 1 << iota
	Crop
	Fill
	Order
)

type Shaper struct {
	zctx       *zed.Context
	expr       Evaluator
	typeExpr   Evaluator
	transforms ShaperTransform

	shapers map[zed.Type]*ConstShaper
}

// NewShaper returns a shaper that will shape the result of expr
// to the type returned by typeExpr according to tf.
func NewShaper(zctx *zed.Context, expr, typeExpr Evaluator, tf ShaperTransform) *Shaper {
	return &Shaper{
		zctx:       zctx,
		expr:       expr,
		typeExpr:   typeExpr,
		transforms: tf,
		shapers:    make(map[zed.Type]*ConstShaper),
	}
}

func (s *Shaper) Eval(ectx Context, this *zed.Value) *zed.Value {
	//XXX should have a fast path for constant types
	typeVal := s.typeExpr.Eval(ectx, this)
	if typeVal.IsError() {
		return typeVal
	}
	if typeVal.Type == zed.TypeString {
		typ, _ := s.zctx.LookupTypeNamed(string(typeVal.Bytes), this.Type)
		return ectx.NewValue(typ, this.Bytes)
	}
	//XXX TypeUnder?
	if typeVal.Type != zed.TypeType {
		return ectx.CopyValue(s.zctx.NewErrorf(
			"shaper type argument is not a type: %s", zson.MustFormatValue(typeVal)))
	}
	shapeTo, err := s.zctx.LookupByValue(typeVal.Bytes)
	if err != nil {
		panic(err)
	}
	shaper, ok := s.shapers[shapeTo]
	if !ok {
		//XXX we should check if this is a cast-only function and
		// and allocate a primitive caster if warranted
		shaper = NewConstShaper(s.zctx, s.expr, shapeTo, s.transforms)
		s.shapers[shapeTo] = shaper
	}
	return shaper.Eval(ectx, this)
}

type ConstShaper struct {
	zctx       *zed.Context
	expr       Evaluator
	shapeTo    zed.Type
	transforms ShaperTransform

	b       zcode.Builder
	shapers map[int]*shaper // map from input type ID to shaper
}

// NewConstShaper returns a shaper that will shape the result of expr
// to the provided shapeTo type.
func NewConstShaper(zctx *zed.Context, expr Evaluator, shapeTo zed.Type, tf ShaperTransform) *ConstShaper {
	return &ConstShaper{
		zctx:       zctx,
		expr:       expr,
		shapeTo:    shapeTo,
		transforms: tf,
		shapers:    make(map[int]*shaper),
	}
}

func (c *ConstShaper) Eval(ectx Context, this *zed.Value) *zed.Value {
	val := c.expr.Eval(ectx, this)
	if val.IsError() {
		return val
	}
	id := val.Type.ID()
	s, ok := c.shapers[id]
	if !ok {
		var err error
		s, err = newShaper(c.zctx, c.transforms, val.Type, c.shapeTo)
		if err != nil {
			return ectx.CopyValue(c.zctx.NewError(err))
		}
		c.shapers[id] = s
	}
	if s.typ.ID() == id {
		return ectx.NewValue(s.typ, val.Bytes)
	}
	c.b.Reset()
	typ := s.step.build(c.zctx, ectx, val.Bytes, &c.b)
	return ectx.NewValue(typ, c.b.Bytes().Body())
}

// A shaper is a per-input type ID "spec" that contains the output
// type and the op to create an output value.
type shaper struct {
	typ  zed.Type
	step step
}

func newShaper(zctx *zed.Context, tf ShaperTransform, in, out zed.Type) (*shaper, error) {
	typ, err := shaperType(zctx, tf, in, out)
	if err != nil {
		return nil, err
	}
	step, err := newStep(zctx, in, typ)
	return &shaper{typ, step}, err
}

func shaperType(zctx *zed.Context, tf ShaperTransform, in, out zed.Type) (zed.Type, error) {
	inUnder, outUnder := zed.TypeUnder(in), zed.TypeUnder(out)
	if tf&Cast != 0 {
		if inUnder == outUnder || inUnder == zed.TypeNull {
			return out, nil
		}
		if isMap(outUnder) {
			return nil, fmt.Errorf("cannot yet use maps in shaping functions (issue #2894)")
		}
		if zed.IsPrimitiveType(inUnder) && zed.IsPrimitiveType(outUnder) {
			// Matching field is a primitive: output type is cast type.
			if LookupPrimitiveCaster(zctx, outUnder) == nil {
				return nil, fmt.Errorf("cast to %s not implemented", zson.FormatType(out))
			}
			return out, nil
		}
		if in, ok := inUnder.(*zed.TypeUnion); ok {
			for _, t := range in.Types {
				if _, err := shaperType(zctx, tf, t, out); err != nil {
					return nil, fmt.Errorf("cannot cast union %q to %q due to %q",
						zson.FormatType(in), zson.FormatType(out), zson.FormatType(t))
				}
			}
			return out, nil
		}
		if bestUnionTag(in, outUnder) > -1 {
			return out, nil
		}
	} else if inUnder == outUnder {
		return in, nil
	}
	if inRec, ok := inUnder.(*zed.TypeRecord); ok {
		if outRec, ok := outUnder.(*zed.TypeRecord); ok {
			cols, err := shaperColumns(zctx, tf, inRec, outRec)
			if err != nil {
				return nil, err
			}
			if tf&Cast != 0 {
				if equalColumns(cols, outRec.Columns) {
					return out, nil
				}
			} else if equalColumns(cols, inRec.Columns) {
				return in, nil
			}
			return zctx.LookupTypeRecord(cols)
		}
	}
	inInner, outInner := zed.InnerType(inUnder), zed.InnerType(outUnder)
	if inInner != nil && outInner != nil && (tf&Cast != 0 || isArray(inUnder) == isArray(outUnder)) {
		t, err := shaperType(zctx, tf, inInner, outInner)
		if err != nil {
			return nil, err
		}
		if tf&Cast != 0 {
			if t == outInner {
				return out, nil
			}
		} else if t == inInner {
			return in, nil
		}
		if isArray(outUnder) {
			return zctx.LookupTypeArray(t), nil
		}
		return zctx.LookupTypeSet(t), nil
	}
	return in, nil
}

func shaperColumns(zctx *zed.Context, tf ShaperTransform, in, out *zed.TypeRecord) ([]zed.Column, error) {
	crop, fill := tf&Crop != 0, tf&Fill != 0
	if tf&Order == 0 {
		crop, fill = !fill, !crop
		out, in = in, out
	}
	var cols []zed.Column
	for _, outCol := range out.Columns {
		if inColType, ok := in.TypeOfField(outCol.Name); ok {
			outColType := outCol.Type
			if tf&Order == 0 {
				// Counteract the swap of in and out above.
				outColType, inColType = inColType, outColType
			}
			t, err := shaperType(zctx, tf, inColType, outColType)
			if err != nil {
				return nil, err
			}
			cols = append(cols, zed.Column{Name: outCol.Name, Type: t})
		} else if fill {
			cols = append(cols, outCol)
		}
	}
	if !crop {
		inColumns := in.Columns
		if tf&Order != 0 {
			// Order appends unknown fields in lexicographic order.
			inColumns = slices.Clone(inColumns)
			sort.Slice(inColumns, func(i, j int) bool {
				return inColumns[i].Name < inColumns[j].Name
			})
		}
		for _, inCol := range inColumns {
			if !out.HasField(inCol.Name) {
				cols = append(cols, inCol)
			}
		}
	}
	return cols, nil
}

// bestUnionTag tries to return the most specific union tag for in
// within out.  It returns -1 if out is not a union or contains no type
// compatible with in.  (Types are compatible if they have the same underlying
// type.)  If out contains in, bestUnionTag returns its tag.
// Otherwise, if out contains in's underlying type, bestUnionTag returns
// its tag.  Finally, bestUnionTag returns the smallest tag in
// out whose type is compatible with in.
func bestUnionTag(in, out zed.Type) int {
	outUnion, ok := zed.TypeUnder(out).(*zed.TypeUnion)
	if !ok {
		return -1
	}
	typeUnderIn := zed.TypeUnder(in)
	underlying := -1
	compatible := -1
	for i, t := range outUnion.Types {
		if t == in {
			return i
		}
		if t == typeUnderIn && underlying == -1 {
			underlying = i
		}
		if zed.TypeUnder(t) == typeUnderIn && compatible == -1 {
			compatible = i
		}
	}
	if underlying != -1 {
		return underlying
	}
	return compatible
}

func equalColumns(a, b []zed.Column) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isArray(t zed.Type) bool {
	_, ok := t.(*zed.TypeArray)
	return ok
}

func isMap(t zed.Type) bool {
	_, ok := t.(*zed.TypeMap)
	return ok
}

type op int

const (
	copyOp        op = iota // copy field fromIndex from input record
	castPrimitive           // cast field fromIndex from fromType to toType
	castFromUnion           // cast union value with tag s using children[s]
	castToUnion             // cast non-union fromType to union toType with tag toTag
	null                    // write null
	array                   // build array
	set                     // build set
	record                  // build record
)

// A step is a recursive data structure encoding a series of
// copy/cast steps to be carried out over an input record.
type step struct {
	op        op
	caster    Evaluator // for castPrimitive
	fromIndex int       // for children of a record step
	fromType  zed.Type  // for castPrimitive and castToUnion
	toTag     int       // for castToUnion
	toType    zed.Type
	// if op == record, contains one op for each column.
	// if op == array, contains one op for all array elements.
	// if op == castFromUnion, contains one op per union tag.
	children []step

	types       []zed.Type
	uniqueTypes []zed.Type
}

func newStep(zctx *zed.Context, in, out zed.Type) (step, error) {
Switch:
	switch {
	case in.ID() == zed.IDNull:
		return step{op: null, toType: out}, nil
	case in.ID() == out.ID():
		return step{op: copyOp, toType: out}, nil
	case zed.IsRecordType(in) && zed.IsRecordType(out):
		return newRecordStep(zctx, zed.TypeRecordOf(in), out)
	case zed.IsPrimitiveType(in) && zed.IsPrimitiveType(out):
		caster := LookupPrimitiveCaster(zctx, zed.TypeUnder(out))
		return step{op: castPrimitive, caster: caster, fromType: in, toType: out}, nil
	case zed.InnerType(in) != nil:
		if k := out.Kind(); k == zed.ArrayKind {
			return newArrayOrSetStep(zctx, array, zed.InnerType(in), out)
		} else if k == zed.SetKind {
			return newArrayOrSetStep(zctx, set, zed.InnerType(in), out)
		}
	case zed.IsUnionType(in):
		var steps []step
		for _, t := range zed.TypeUnder(in).(*zed.TypeUnion).Types {
			s, err := newStep(zctx, t, out)
			if err != nil {
				break Switch
			}
			steps = append(steps, s)
		}
		return step{op: castFromUnion, toType: out, children: steps}, nil
	}
	if tag := bestUnionTag(in, out); tag != -1 {
		return step{op: castToUnion, fromType: in, toTag: tag, toType: out}, nil
	}
	return step{}, fmt.Errorf("createStep: incompatible types %s and %s", zson.FormatType(in), zson.FormatType(out))
}

// newRecordStep returns a step that will build a record of type out from a
// record of type in. The two types must be compatible, meaning that
// the input type must be an unordered subset of the input type
// (where 'unordered' means that if the output type has record fields
// [a b] and the input type has fields [b a] that is ok). It is also
// ok for leaf primitive types to be different; if they are a casting
// step is inserted.
func newRecordStep(zctx *zed.Context, in *zed.TypeRecord, out zed.Type) (step, error) {
	var children []step
	for _, outCol := range zed.TypeRecordOf(out).Columns {
		ind, ok := in.ColumnOfField(outCol.Name)
		if !ok {
			children = append(children, step{op: null, toType: outCol.Type})
			continue
		}
		child, err := newStep(zctx, in.Columns[ind].Type, outCol.Type)
		if err != nil {
			return step{}, err
		}
		child.fromIndex = ind
		children = append(children, child)
	}
	return step{op: record, toType: out, children: children}, nil
}

func newArrayOrSetStep(zctx *zed.Context, op op, inInner, out zed.Type) (step, error) {
	innerStep, err := newStep(zctx, inInner, zed.InnerType(out))
	if err != nil {
		return step{}, err
	}
	return step{op: op, toType: out, children: []step{innerStep}}, nil
}

// build applies the operation described by s to in, appends the resulting bytes
// to b, and returns the resulting type.  The type is usually s.toType but can
// differ if a primitive cast fails.
func (s *step) build(zctx *zed.Context, ectx Context, in zcode.Bytes, b *zcode.Builder) zed.Type {
	if in == nil || s.op == copyOp {
		b.Append(in)
		return s.toType
	}
	switch s.op {
	case castPrimitive:
		// For a successful cast, v.Type == zed.TypeUnder(s.toType).
		// For a failed cast, v.Type is a zed.TypeError.
		v := s.caster.Eval(ectx, zed.NewValue(s.fromType, in))
		b.Append(v.Bytes)
		if zed.TypeUnder(v.Type) == zed.TypeUnder(s.toType) {
			// Prefer s.toType in case it's a named type.
			return s.toType
		}
		return v.Type
	case castFromUnion:
		it := in.Iter()
		tag := int(zed.DecodeInt(it.Next()))
		return s.children[tag].build(zctx, ectx, it.Next(), b)
	case castToUnion:
		zed.BuildUnion(b, s.toTag, in)
		return s.toType
	case array, set:
		return s.buildArrayOrSet(zctx, ectx, s.op, in, b)
	case record:
		return s.buildRecord(zctx, ectx, in, b)
	default:
		panic(fmt.Sprintf("unknown step.op %v", s.op))
	}
}

func (s *step) buildArrayOrSet(zctx *zed.Context, ectx Context, op op, in zcode.Bytes, b *zcode.Builder) zed.Type {
	b.BeginContainer()
	defer b.EndContainer()
	s.types = s.types[:0]
	for it := in.Iter(); !it.Done(); {
		typ := s.children[0].build(zctx, ectx, it.Next(), b)
		s.types = append(s.types, typ)
	}
	s.uniqueTypes = append(s.uniqueTypes[:0], s.types...)
	s.uniqueTypes = zed.UniqueTypes(s.uniqueTypes)
	var inner zed.Type
	switch len(s.uniqueTypes) {
	case 0:
		return s.toType
	case 1:
		inner = s.uniqueTypes[0]
	default:
		union := zctx.LookupTypeUnion(s.uniqueTypes)
		// Convert each container element to the union type.
		b.TransformContainer(func(bytes zcode.Bytes) zcode.Bytes {
			var b2 zcode.Builder
			for i, it := 0, bytes.Iter(); !it.Done(); i++ {
				zed.BuildUnion(&b2, union.TagOf(s.types[i]), it.Next())
			}
			return b2.Bytes()
		})
		inner = union
	}
	if op == set {
		b.TransformContainer(zed.NormalizeSet)
	}
	if zed.TypeUnder(inner) == zed.TypeUnder(zed.InnerType(s.toType)) {
		// Prefer s.toType in case it or its inner type is a named type.
		return s.toType
	}
	if op == set {
		return zctx.LookupTypeSet(inner)
	}
	return zctx.LookupTypeArray(inner)
}

func (s *step) buildRecord(zctx *zed.Context, ectx Context, in zcode.Bytes, b *zcode.Builder) zed.Type {
	b.BeginContainer()
	defer b.EndContainer()
	s.types = s.types[:0]
	var needNewRecordType bool
	for _, child := range s.children {
		if child.op == null {
			b.Append(nil)
			s.types = append(s.types, child.toType)
			continue
		}
		// Using getNthFromContainer means we iterate from the
		// beginning of the record for each field. An
		// optimization (for shapes that don't require field
		// reordering) would be make direct use of a
		// zcode.Iter along with keeping track of our
		// position.
		bytes := getNthFromContainer(in, child.fromIndex)
		typ := child.build(zctx, ectx, bytes, b)
		if zed.TypeUnder(typ) == zed.TypeUnder(child.toType) {
			// Prefer child.toType in case it's a named type.
			typ = child.toType
		} else {
			// This field's type differs from the corresponding
			// field in s.toType, so we'll need to look up a new
			// record type below.
			needNewRecordType = true
		}
		s.types = append(s.types, typ)
	}
	if needNewRecordType {
		fields := slices.Clone(zed.TypeUnder(s.toType).(*zed.TypeRecord).Columns)
		for i, t := range s.types {
			fields[i].Type = t
		}
		return zctx.MustLookupTypeRecord(fields)
	}
	return s.toType
}

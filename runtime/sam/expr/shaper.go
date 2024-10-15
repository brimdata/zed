package expr

import (
	"fmt"
	"slices"
	"sort"

	"github.com/brimdata/super"
	"github.com/brimdata/super/zcode"
	"github.com/brimdata/super/zson"
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

func NewShaperTransform(s string) ShaperTransform {
	switch s {
	case "cast":
		return Cast
	case "crop":
		return Crop
	case "fill":
		return Fill
	case "fit":
		return Crop | Fill
	case "order":
		return Order
	case "shape":
		return Cast | Fill | Order
	}
	return 0
}

// NewShaper returns a shaper that will shape the result of expr
// to the type returned by typeExpr according to tf.
func NewShaper(zctx *zed.Context, expr, typeExpr Evaluator, tf ShaperTransform) (Evaluator, error) {
	if l, ok := typeExpr.(*Literal); ok {
		typeVal := l.val
		switch id := typeVal.Type().ID(); {
		case id == zed.IDType:
			typ, err := zctx.LookupByValue(typeVal.Bytes())
			if err != nil {
				return nil, err
			}
			return NewConstShaper(zctx, expr, typ, tf), nil
		case id == zed.IDString && tf == Cast:
			name := zed.DecodeString(typeVal.Bytes())
			if _, err := zed.NewContext().LookupTypeNamed(name, zed.TypeNull); err != nil {
				return nil, err
			}
			return &casterNamedType{zctx, expr, name}, nil
		}
		return nil, fmt.Errorf("shaper type argument is not a type: %s", zson.FormatValue(typeVal))
	}
	return &Shaper{
		zctx:       zctx,
		expr:       expr,
		typeExpr:   typeExpr,
		transforms: tf,
		shapers:    make(map[zed.Type]*ConstShaper),
	}, nil
}

type Shaper struct {
	zctx       *zed.Context
	expr       Evaluator
	typeExpr   Evaluator
	transforms ShaperTransform

	shapers map[zed.Type]*ConstShaper
}

func (s *Shaper) Eval(ectx Context, this zed.Value) zed.Value {
	typeVal := s.typeExpr.Eval(ectx, this)
	switch id := typeVal.Type().ID(); {
	case id == zed.IDType:
		typ, err := s.zctx.LookupByValue(typeVal.Bytes())
		if err != nil {
			return s.zctx.NewError(err)
		}
		shaper, ok := s.shapers[typ]
		if !ok {
			shaper = NewConstShaper(s.zctx, s.expr, typ, s.transforms)
			s.shapers[typ] = shaper
		}
		return shaper.Eval(ectx, this)
	case id == zed.IDString && s.transforms == Cast:
		name := zed.DecodeString(typeVal.Bytes())
		return (&casterNamedType{s.zctx, s.expr, name}).Eval(ectx, this)
	}
	return s.zctx.WrapError("shaper type argument is not a type", typeVal)
}

type ConstShaper struct {
	zctx       *zed.Context
	expr       Evaluator
	shapeTo    zed.Type
	transforms ShaperTransform

	b       zcode.Builder
	caster  Evaluator       // used when shapeTo is a primitive type
	shapers map[int]*shaper // map from input type ID to shaper
}

// NewConstShaper returns a shaper that will shape the result of expr
// to the provided shapeTo type.
func NewConstShaper(zctx *zed.Context, expr Evaluator, shapeTo zed.Type, tf ShaperTransform) *ConstShaper {
	var caster Evaluator
	if tf == Cast {
		// Use a caster since it's faster.
		caster = LookupPrimitiveCaster(zctx, zed.TypeUnder(shapeTo))
	}
	return &ConstShaper{
		zctx:       zctx,
		expr:       expr,
		shapeTo:    shapeTo,
		transforms: tf,
		caster:     caster,
		shapers:    make(map[int]*shaper),
	}
}

func (c *ConstShaper) Eval(ectx Context, this zed.Value) zed.Value {
	val := c.expr.Eval(ectx, this)
	if val.IsError() {
		return val
	}
	if val.IsNull() {
		// Null values can be shaped to any type.
		return zed.NewValue(c.shapeTo, nil)
	}
	id, shapeToID := val.Type().ID(), c.shapeTo.ID()
	if id == shapeToID {
		// Same underlying types but one or both are named.
		return zed.NewValue(c.shapeTo, val.Bytes())
	}
	if c.caster != nil && !zed.IsUnionType(val.Type()) {
		val = c.caster.Eval(ectx, val)
		if val.Type() != c.shapeTo && val.Type().ID() == shapeToID {
			// Same underlying types but one or both are named.
			return zed.NewValue(c.shapeTo, val.Bytes())
		}
		return val
	}
	s, ok := c.shapers[id]
	if !ok {
		var err error
		s, err = newShaper(c.zctx, c.transforms, val.Type(), c.shapeTo)
		if err != nil {
			return c.zctx.NewError(err)
		}
		c.shapers[id] = s
	}
	c.b.Reset()
	typ := s.step.build(c.zctx, ectx, val.Bytes(), &c.b)
	return zed.NewValue(typ, c.b.Bytes().Body())
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
			fields, err := shaperFields(zctx, tf, inRec, outRec)
			if err != nil {
				return nil, err
			}
			if tf&Cast != 0 {
				if slices.Equal(fields, outRec.Fields) {
					return out, nil
				}
			} else if slices.Equal(fields, inRec.Fields) {
				return in, nil
			}
			return zctx.LookupTypeRecord(fields)
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

func shaperFields(zctx *zed.Context, tf ShaperTransform, in, out *zed.TypeRecord) ([]zed.Field, error) {
	crop, fill := tf&Crop != 0, tf&Fill != 0
	if tf&Order == 0 {
		crop, fill = !fill, !crop
		out, in = in, out
	}
	var fields []zed.Field
	for _, outField := range out.Fields {
		if inFieldType, ok := in.TypeOfField(outField.Name); ok {
			outFieldType := outField.Type
			if tf&Order == 0 {
				// Counteract the swap of in and out above.
				outFieldType, inFieldType = inFieldType, outFieldType
			}
			t, err := shaperType(zctx, tf, inFieldType, outFieldType)
			if err != nil {
				return nil, err
			}
			fields = append(fields, zed.NewField(outField.Name, t))
		} else if fill {
			fields = append(fields, outField)
		}
	}
	if !crop {
		inFields := in.Fields
		if tf&Order != 0 {
			// Order appends unknown fields in lexicographic order.
			inFields = slices.Clone(inFields)
			sort.Slice(inFields, func(i, j int) bool {
				return inFields[i].Name < inFields[j].Name
			})
		}
		for _, f := range inFields {
			if !out.HasField(f.Name) {
				fields = append(fields, f)
			}
		}
	}
	return fields, nil
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
	// if op == record, contains one op for each field.
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
	for _, outField := range zed.TypeRecordOf(out).Fields {
		ind, ok := in.IndexOfField(outField.Name)
		if !ok {
			children = append(children, step{op: null, toType: outField.Type})
			continue
		}
		child, err := newStep(zctx, in.Fields[ind].Type, outField.Type)
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
		b.Append(v.Bytes())
		if zed.TypeUnder(v.Type()) == zed.TypeUnder(s.toType) {
			// Prefer s.toType in case it's a named type.
			return s.toType
		}
		return v.Type()
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
		fields := slices.Clone(zed.TypeUnder(s.toType).(*zed.TypeRecord).Fields)
		for i, t := range s.types {
			fields[i].Type = t
		}
		return zctx.MustLookupTypeRecord(fields)
	}
	return s.toType
}

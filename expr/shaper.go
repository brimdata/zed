package expr

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

// A ShaperTransform represents one of the different transforms that a
// shaper can apply.  The transforms are represented as a bit flags that
// can be bitwise-ored together to create a single shaping operator that
// represents the composition of all operators.  This composition is efficient
// as it is carried once per incoming type signature and then the resulting
// operator is run for every value of that type.
type ShaperTransform int

const (
	Cast ShaperTransform = 1 << iota
	Fill
	Crop
	Order
)

type Shaper struct {
	zctx       *zed.Context
	typExpr    Evaluator
	expr       Evaluator
	shapers    map[zed.Type]*ConstShaper
	transforms ShaperTransform
}

// NewShaper returns a shaper that will shape the result of expr
// to the type returned by typExpr.
func NewShaper(zctx *zed.Context, expr, typExpr Evaluator, tf ShaperTransform) *Shaper {
	return &Shaper{
		zctx:       zctx,
		typExpr:    typExpr,
		expr:       expr,
		shapers:    make(map[zed.Type]*ConstShaper),
		transforms: tf,
	}
}

func (s *Shaper) Eval(ectx Context, this *zed.Value) *zed.Value {
	//XXX should have a fast path for constant types
	typVal := s.typExpr.Eval(ectx, this)
	if typVal.IsError() {
		return typVal
	}
	//XXX aliasof?
	if typVal.Type != zed.TypeType {
		return ectx.CopyValue(*s.zctx.NewErrorf(
			"shaper type argument is not a type: %s", zson.MustFormatValue(*typVal)))
	}
	shapeTo, err := s.zctx.LookupByValue(typVal.Bytes)
	if err != nil {
		panic(err)
	}
	shaper, ok := s.shapers[shapeTo]
	if !ok {
		//XXX we should check if this is a cast-only function and
		// and allocate a primitive caster if warranted
		if zed.TypeRecordOf(shapeTo) == nil {
			//XXX use structured error
			return ectx.CopyValue(*s.zctx.NewErrorf(
				"shaper function type argument is not a record type: %q", shapeTo))
		}
		shaper = NewConstShaper(s.zctx, s.expr, shapeTo, s.transforms)
		s.shapers[shapeTo] = shaper
	}
	return shaper.Eval(ectx, this)
}

type ConstShaper struct {
	zctx       *zed.Context
	b          zcode.Builder
	expr       Evaluator
	shapeTo    zed.Type
	shapers    map[int]*shaper // map from input type ID to shaper
	transforms ShaperTransform
}

// NewConstShaper returns a shaper that will shape the result of expr
// to the provided shapeTo type.
func NewConstShaper(zctx *zed.Context, expr Evaluator, shapeTo zed.Type, tf ShaperTransform) *ConstShaper {
	return &ConstShaper{
		zctx:       zctx,
		expr:       expr,
		shapeTo:    shapeTo,
		shapers:    make(map[int]*shaper),
		transforms: tf,
	}
}

func (s *ConstShaper) Apply(ectx Context, this *zed.Value) *zed.Value {
	val := s.Eval(ectx, this)
	if !zed.IsRecordType(val.Type) {
		// XXX use structured error
		return ectx.CopyValue(*s.zctx.NewErrorf(
			"shaper returned non-record value %s", zson.MustFormatValue(*val)))
	}
	return val
}

func (c *ConstShaper) Eval(ectx Context, this *zed.Value) *zed.Value {
	val := c.expr.Eval(ectx, this)
	if val.IsError() {
		return val
	}
	id := this.Type.ID()
	s, ok := c.shapers[id]
	if !ok {
		var err error
		s, err = createShaper(c.zctx, c.transforms, c.shapeTo, val.Type)
		if err != nil {
			return ectx.CopyValue(*c.zctx.NewError(err))
		}
		c.shapers[id] = s
	}
	if s.typ.ID() == id {
		return ectx.NewValue(s.typ, val.Bytes)
	}
	c.b.Reset()
	if zerr := s.step.buildRecord(c.zctx, ectx, val.Bytes, &c.b); zerr != nil {
		return zerr
	}
	return ectx.NewValue(s.typ, c.b.Bytes())
}

// A shaper is a per-input type ID "spec" that contains the output
// type and the op to create an output record.
type shaper struct {
	typ  zed.Type
	step step
}

func createShaper(zctx *zed.Context, tf ShaperTransform, spec, in zed.Type) (*shaper, error) {
	typ, err := shaperType(zctx, tf, spec, in)
	if err != nil {
		return nil, err
	}
	step, err := createStepRecord(zed.TypeRecordOf(in), zed.TypeRecordOf(typ))
	return &shaper{typ, step}, err
}

func shaperType(zctx *zed.Context, tf ShaperTransform, spec, in zed.Type) (zed.Type, error) {
	inUnder, specUnder := zed.TypeUnder(in), zed.TypeUnder(spec)
	if tf&Cast > 0 {
		if inUnder == specUnder || inUnder == zed.TypeNull {
			return spec, nil
		}
		if isMap(specUnder) {
			return nil, fmt.Errorf("cannot yet use maps in shaping functions (issue #2894)")
		}
		if zed.IsPrimitiveType(inUnder) && zed.IsPrimitiveType(specUnder) {
			// Matching field is a primitive: output type is cast type.
			if LookupPrimitiveCaster(zctx, specUnder) == nil {
				return nil, fmt.Errorf("cast to %s not implemented", spec)
			}
			return spec, nil
		}
		if bestUnionSelector(in, specUnder) > -1 {
			return spec, nil
		}
	} else if inUnder == specUnder {
		return in, nil
	}
	if inRec, ok := inUnder.(*zed.TypeRecord); ok {
		if specRec, ok := specUnder.(*zed.TypeRecord); ok {
			cols, err := shaperColumns(zctx, tf, specRec, inRec)
			if err != nil {
				return nil, err
			}
			if tf&Cast > 0 {
				if equalColumns(cols, specRec.Columns) {
					return spec, nil
				}
			} else if equalColumns(cols, inRec.Columns) {
				return in, nil
			}
			return zctx.LookupTypeRecord(cols)
		}
	}
	inInner, specInner := zed.InnerType(inUnder), zed.InnerType(specUnder)
	if inInner != nil && specInner != nil && (tf&Cast > 0 || isArray(inUnder) == isArray(specUnder)) {
		t, err := shaperType(zctx, tf, specInner, inInner)
		if err != nil {
			return nil, err
		}
		if tf&Cast > 0 {
			if t == specInner {
				return spec, nil
			}
		} else if t == inInner {
			return in, nil
		}
		if isArray(specUnder) {
			return zctx.LookupTypeArray(t), nil
		}
		return zctx.LookupTypeSet(t), nil
	}
	return in, nil
}

func shaperColumns(zctx *zed.Context, tf ShaperTransform, specRec, inRec *zed.TypeRecord) ([]zed.Column, error) {
	crop, fill := tf&Crop > 0, tf&Fill > 0
	if tf&Order == 0 {
		crop, fill = !fill, !crop
		specRec, inRec = inRec, specRec
	}
	var cols []zed.Column
	for _, specCol := range specRec.Columns {
		if inColType, ok := inRec.TypeOfField(specCol.Name); ok {
			specColType := specCol.Type
			if tf&Order == 0 {
				// Counteract the swap of specRec and inRec above.
				specColType, inColType = inColType, specColType
			}
			t, err := shaperType(zctx, tf, specColType, inColType)
			if err != nil {
				return nil, err
			}
			cols = append(cols, zed.Column{Name: specCol.Name, Type: t})
		} else if fill {
			cols = append(cols, specCol)
		}
	}
	if !crop {
		for _, inCol := range inRec.Columns {
			if !specRec.HasField(inCol.Name) {
				cols = append(cols, inCol)
			}
		}
	}
	return cols, nil
}

// bestUnionSelector tries to return the most specific union selector for in
// within spec.  It returns -1 if spec is not a union or contains no type
// compatible with in.  (Types are compatible if they have the same underlying
// type.)  If spec contains in, bestUnionSelector returns its selector.
// Otherwise, if spec contains in's underlying type, bestUnionSelector returns
// its selector.  Finally, bestUnionSelector returns the smallest selector in
// spec whose type is compatible with in.
func bestUnionSelector(in, spec zed.Type) int {
	specUnion, ok := zed.TypeUnder(spec).(*zed.TypeUnion)
	if !ok {
		return -1
	}
	aliasOfIn := zed.TypeUnder(in)
	underlying := -1
	compatible := -1
	for i, t := range specUnion.Types {
		if t == in {
			return i
		}
		if t == aliasOfIn && underlying == -1 {
			underlying = i
		}
		if zed.TypeUnder(t) == aliasOfIn && compatible == -1 {
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
	copyPrimitive op = iota // copy field fromIndex from input record
	copyContainer
	castPrimitive // cast field fromIndex from fromType to toType
	castUnion     // cast field fromIndex from fromType to union with selector toSelector
	null          // write null
	array         // build array
	set           // build set
	record        // build record
)

// A step is a recursive data structure encoding a series of
// copy/cast steps to be carried out over an input record.
type step struct {
	op         op
	fromIndex  int
	fromType   zed.Type // for castPrimitive and castUnion
	toSelector int      // for castUnion
	toType     zed.Type // for castPrimitive
	// if op == record, contains one op for each column.
	// if op == array, contains one op for all array elements.
	children []step
}

// create the step needed to build a record of type out from a
// record of type in. The two types must be compatible, meaning that
// the input type must be an unordered subset of the input type
// (where 'unordered' means that if the output type has record fields
// [a b] and the input type has fields [b a] that is ok). It is also
// ok for leaf primitive types to be different; if they are a casting
// step is inserted.
func createStepRecord(in, out *zed.TypeRecord) (step, error) {
	s := step{op: record}
	for _, outCol := range out.Columns {
		ind, ok := in.ColumnOfField(outCol.Name)
		if !ok {
			s.append(step{op: null})
			continue
		}
		inCol := in.Columns[ind]
		child, err := createStep(inCol.Type, outCol.Type)
		if err != nil {
			return step{}, err
		}
		child.fromIndex = ind
		s.append(child)
	}
	return s, nil
}

func createStep(in, out zed.Type) (step, error) {
	switch {
	case in.ID() == zed.IDNull:
		return step{op: null}, nil
	case in.ID() == out.ID():
		if zed.IsContainerType(in) {
			return step{op: copyContainer}, nil
		} else {
			return step{op: copyPrimitive}, nil
		}
	case zed.IsRecordType(in) && zed.IsRecordType(out):
		return createStepRecord(zed.TypeRecordOf(in), zed.TypeRecordOf(out))
	case zed.IsPrimitiveType(in) && zed.IsPrimitiveType(out):
		return step{op: castPrimitive, fromType: in, toType: out}, nil
	case isCollectionType(in):
		if _, ok := zed.TypeUnder(out).(*zed.TypeArray); ok {
			return createStepArray(zed.InnerType(in), zed.InnerType(out))
		}
		if _, ok := zed.TypeUnder(out).(*zed.TypeSet); ok {
			return createStepSet(zed.InnerType(in), zed.InnerType(out))
		}
	}
	if s := bestUnionSelector(in, out); s != -1 {
		return step{op: castUnion, fromType: in, toSelector: s}, nil
	}
	return step{}, fmt.Errorf("createStep: incompatible types %s and %s", in, out)
}

func isCollectionType(t zed.Type) bool {
	switch zed.TypeUnder(t).(type) {
	case *zed.TypeArray, *zed.TypeSet:
		return true
	}
	return false
}

func createStepArray(in, out zed.Type) (step, error) {
	s := step{op: array}
	innerOp, err := createStep(in, out)
	if err != nil {
		return step{}, nil
	}
	s.append(innerOp)
	return s, nil
}

func createStepSet(in, out zed.Type) (step, error) {
	s := step{op: set}
	innerOp, err := createStep(in, out)
	if err != nil {
		return step{}, nil
	}
	s.append(innerOp)
	return s, nil
}

func (s *step) append(step step) {
	s.children = append(s.children, step)
}

func (s *step) buildRecord(zctx *zed.Context, ectx Context, in zcode.Bytes, b *zcode.Builder) *zed.Value {
	for _, step := range s.children {
		switch step.op {
		case null:
			b.AppendNull()
			continue
		}
		// Using getNthFromContainer means we iterate from the
		// beginning of the record for each field. An
		// optimization (for shapes that don't require field
		// reordering) would be make direct use of a
		// zcode.Iter along with keeping track of our
		// position.
		bytes := getNthFromContainer(in, uint(step.fromIndex))
		if zerr := step.build(zctx, ectx, bytes, b); zerr != nil {
			return zerr
		}
	}
	return nil
}

func (s *step) build(zctx *zed.Context, ectx Context, in zcode.Bytes, b *zcode.Builder) *zed.Value {
	switch s.op {
	case copyPrimitive:
		b.AppendPrimitive(in)
	case copyContainer:
		b.AppendContainer(in)
	case castPrimitive:
		if zerr := s.castPrimitive(zctx, ectx, in, b); zerr != nil {
			return zerr
		}
	case castUnion:
		zed.BuildUnion(b, s.toSelector, in, zed.IsContainerType(s.fromType))
	case record:
		if in == nil {
			b.AppendNull()
			return nil
		}
		b.BeginContainer()
		if zerr := s.buildRecord(zctx, ectx, in, b); zerr != nil {
			return zerr
		}
		b.EndContainer()
	case array, set:
		if in == nil {
			b.AppendNull()
			return nil
		}
		b.BeginContainer()
		iter := in.Iter()
		for !iter.Done() {
			zv, _ := iter.Next()
			if zerr := s.children[0].build(zctx, ectx, zv, b); zerr != nil {
				return zerr
			}
		}
		if s.op == set {
			b.TransformContainer(zed.NormalizeSet)
		}
		b.EndContainer()
	}
	return nil
}

func (s *step) castPrimitive(zctx *zed.Context, ectx Context, in zcode.Bytes, b *zcode.Builder) *zed.Value {
	if in == nil {
		b.AppendNull()
		return nil
	}
	toType := zed.TypeUnder(s.toType)
	//XXX We should cache these allocations. See issue #3456.
	caster := LookupPrimitiveCaster(zctx, toType)
	v := caster.Eval(ectx, &zed.Value{s.fromType, in})
	if v.Type != toType {
		// v isn't the "to" type, so we can't safely append v.Bytes to
		// the builder. See https://github.com/brimdata/zed/issues/2710.
		if v.IsError() {
			return v
		}
		panic(fmt.Sprintf("expr: got %T from primitive caster, expected %T", v.Type, toType))
	}
	b.AppendPrimitive(v.Bytes)
	return nil
}

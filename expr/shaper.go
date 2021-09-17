package expr

import (
	"fmt"

	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zng"
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
	zctx       *zson.Context
	typExpr    Evaluator
	expr       Evaluator
	shapers    map[zng.Type]*ConstShaper
	transforms ShaperTransform
}

// NewShaper returns a shaper that will shape the result of expr
// to the type returned by typExpr.
func NewShaper(zctx *zson.Context, expr, typExpr Evaluator, tf ShaperTransform) *Shaper {
	return &Shaper{
		zctx:       zctx,
		typExpr:    typExpr,
		expr:       expr,
		shapers:    make(map[zng.Type]*ConstShaper),
		transforms: tf,
	}
}

func (s *Shaper) Eval(rec *zng.Record) (zng.Value, error) {
	typVal, err := s.typExpr.Eval(rec)
	if err != nil {
		return zng.Value{}, err
	}
	if typVal.Type != zng.TypeType {
		return zng.NewErrorf("shaper function type argument is not a type"), nil
	}
	shapeTo, err := s.zctx.LookupByValue(typVal.Bytes)
	if err != nil {
		return zng.NewErrorf("shaper encountered unknown type value: %s", err), nil
	}
	shaper, ok := s.shapers[shapeTo]
	if !ok {
		if zng.TypeRecordOf(shapeTo) == nil {
			return zng.NewErrorf("shaper function type argument is not a record type: %q", shapeTo), nil
		}
		shaper = NewConstShaper(s.zctx, s.expr, shapeTo, s.transforms)
		s.shapers[shapeTo] = shaper
	}
	return shaper.Eval(rec)
}

type ConstShaper struct {
	zctx       *zson.Context
	b          zcode.Builder
	expr       Evaluator
	shapeTo    zng.Type
	shapers    map[int]*shaper // map from input type ID to shaper
	transforms ShaperTransform
}

// NewConstShaper returns a shaper that will shape the result of expr
// to the provided shapeTo type.
func NewConstShaper(zctx *zson.Context, expr Evaluator, shapeTo zng.Type, tf ShaperTransform) *ConstShaper {
	return &ConstShaper{
		zctx:       zctx,
		expr:       expr,
		shapeTo:    shapeTo,
		shapers:    make(map[int]*shaper),
		transforms: tf,
	}
}

func (s *ConstShaper) Apply(in *zng.Record) (*zng.Record, error) {
	v, err := s.Eval(in)
	if err != nil {
		return nil, err
	}
	if !zng.IsRecordType(v.Type) {
		return nil, fmt.Errorf("shaper returned non-record value %s", zson.String(v))
	}
	return zng.NewRecord(v.Type, v.Bytes), nil
}

func (c *ConstShaper) Eval(in *zng.Record) (zng.Value, error) {
	inVal, err := c.expr.Eval(in)
	if err != nil {
		return zng.Value{}, err
	}
	id := in.Type.ID()
	s, ok := c.shapers[id]
	if !ok {
		s, err = createShaper(c.zctx, c.transforms, c.shapeTo, inVal.Type)
		if err != nil {
			return zng.Value{}, err
		}
		c.shapers[id] = s
	}
	if s.typ.ID() == id {
		return zng.Value{s.typ, inVal.Bytes}, nil
	}
	c.b.Reset()
	if zerr := s.step.buildRecord(inVal.Bytes, &c.b); zerr != nil {
		typ, err := c.zctx.LookupTypeRecord([]zng.Column{{Name: "error", Type: zerr.Type}})
		if err != nil {
			return zng.Value{}, err
		}
		c.b.AppendPrimitive(zerr.Bytes)
		return zng.Value{typ, c.b.Bytes()}, nil
	}
	return zng.Value{s.typ, c.b.Bytes()}, nil
}

// A shaper is a per-input type ID "spec" that contains the output
// type and the op to create an output record.
type shaper struct {
	typ  zng.Type
	step step
}

func createShaper(zctx *zson.Context, tf ShaperTransform, spec, in zng.Type) (*shaper, error) {
	typ, err := shaperType(zctx, tf, spec, in)
	if err != nil {
		return nil, err
	}
	step, err := createStepRecord(zng.TypeRecordOf(in), zng.TypeRecordOf(typ))
	return &shaper{typ, step}, err
}

func shaperType(zctx *zson.Context, tf ShaperTransform, spec, in zng.Type) (zng.Type, error) {
	inUnder, specUnder := zng.AliasOf(in), zng.AliasOf(spec)
	if tf&Cast > 0 {
		if inUnder == specUnder || inUnder == zng.TypeNull {
			return spec, nil
		}
		if isMap(specUnder) {
			return nil, fmt.Errorf("cannot yet use maps in shaping functions (issue #2894)")
		}
		if zng.IsPrimitiveType(inUnder) && zng.IsPrimitiveType(specUnder) {
			// Matching field is a primitive: output type is cast type.
			if LookupPrimitiveCaster(specUnder) == nil {
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
	if inRec, ok := inUnder.(*zng.TypeRecord); ok {
		if specRec, ok := specUnder.(*zng.TypeRecord); ok {
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
	inInner, specInner := zng.InnerType(inUnder), zng.InnerType(specUnder)
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

func shaperColumns(zctx *zson.Context, tf ShaperTransform, specRec, inRec *zng.TypeRecord) ([]zng.Column, error) {
	crop, fill := tf&Crop > 0, tf&Fill > 0
	if tf&Order == 0 {
		crop, fill = !fill, !crop
		specRec, inRec = inRec, specRec
	}
	var cols []zng.Column
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
			cols = append(cols, zng.Column{Name: specCol.Name, Type: t})
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
func bestUnionSelector(in, spec zng.Type) int {
	specUnion, ok := zng.AliasOf(spec).(*zng.TypeUnion)
	if !ok {
		return -1
	}
	aliasOfIn := zng.AliasOf(in)
	underlying := -1
	compatible := -1
	for i, t := range specUnion.Types {
		if t == in {
			return i
		}
		if t == aliasOfIn && underlying == -1 {
			underlying = i
		}
		if zng.AliasOf(t) == aliasOfIn && compatible == -1 {
			compatible = i
		}
	}
	if underlying != -1 {
		return underlying
	}
	return compatible
}

func equalColumns(a, b []zng.Column) bool {
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

func isArray(t zng.Type) bool {
	_, ok := t.(*zng.TypeArray)
	return ok
}

func isMap(t zng.Type) bool {
	_, ok := t.(*zng.TypeMap)
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
	fromType   zng.Type // for castPrimitive and castUnion
	toSelector int      // for castUnion
	toType     zng.Type // for castPrimitive
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
func createStepRecord(in, out *zng.TypeRecord) (step, error) {
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

func createStep(in, out zng.Type) (step, error) {
	switch {
	case in.ID() == zng.IDNull:
		return step{op: null}, nil
	case in.ID() == out.ID():
		if zng.IsContainerType(in) {
			return step{op: copyContainer}, nil
		} else {
			return step{op: copyPrimitive}, nil
		}
	case zng.IsRecordType(in) && zng.IsRecordType(out):
		return createStepRecord(zng.TypeRecordOf(in), zng.TypeRecordOf(out))
	case zng.IsPrimitiveType(in) && zng.IsPrimitiveType(out):
		return step{op: castPrimitive, fromType: in, toType: out}, nil
	case isCollectionType(in):
		if _, ok := zng.AliasOf(out).(*zng.TypeArray); ok {
			return createStepArray(zng.InnerType(in), zng.InnerType(out))
		}
		if _, ok := zng.AliasOf(out).(*zng.TypeSet); ok {
			return createStepSet(zng.InnerType(in), zng.InnerType(out))
		}
	}
	if s := bestUnionSelector(in, out); s != -1 {
		return step{op: castUnion, fromType: in, toSelector: s}, nil
	}
	return step{}, fmt.Errorf("createStep: incompatible types %s and %s", in, out)
}

func isCollectionType(t zng.Type) bool {
	switch zng.AliasOf(t).(type) {
	case *zng.TypeArray, *zng.TypeSet:
		return true
	}
	return false
}

func createStepArray(in, out zng.Type) (step, error) {
	s := step{op: array}
	innerOp, err := createStep(in, out)
	if err != nil {
		return step{}, nil
	}
	s.append(innerOp)
	return s, nil
}

func createStepSet(in, out zng.Type) (step, error) {
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

func (s *step) buildRecord(in zcode.Bytes, b *zcode.Builder) *zng.Value {
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
		bytes, err := getNthFromContainer(in, uint(step.fromIndex))
		if err != nil {
			panic(err)
		}
		if zerr := step.build(bytes, b); zerr != nil {
			return zerr
		}
	}
	return nil
}

func (s *step) build(in zcode.Bytes, b *zcode.Builder) *zng.Value {
	switch s.op {
	case copyPrimitive:
		b.AppendPrimitive(in)
	case copyContainer:
		b.AppendContainer(in)
	case castPrimitive:
		if zerr := s.castPrimitive(in, b); zerr != nil {
			return zerr
		}
	case castUnion:
		zng.BuildUnion(b, s.toSelector, in, zng.IsContainerType(s.fromType))
	case record:
		if in == nil {
			b.AppendNull()
			return nil
		}
		b.BeginContainer()
		if zerr := s.buildRecord(in, b); zerr != nil {
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
			zv, _, err := iter.Next()
			if err != nil {
				panic(err)
			}
			if zerr := s.children[0].build(zv, b); zerr != nil {
				return zerr
			}
		}
		if s.op == set {
			b.TransformContainer(zng.NormalizeSet)
		}
		b.EndContainer()
	}
	return nil
}

func (s *step) castPrimitive(in zcode.Bytes, b *zcode.Builder) *zng.Value {
	if in == nil {
		b.AppendNull()
		return nil
	}
	toType := zng.AliasOf(s.toType)
	pc := LookupPrimitiveCaster(toType)
	v, err := pc(zng.Value{s.fromType, in})
	if err != nil {
		b.AppendNull()
		return nil
	}
	if v.Type != toType {
		// v isn't the "to" type, so we can't safely append v.Bytes to
		// the builder. See https://github.com/brimdata/zed/issues/2710.
		if v.Type == zng.TypeError {
			return &v
		}
		panic(fmt.Sprintf("expr: got %T from primitive caster, expected %T", v.Type, toType))
	}
	b.AppendPrimitive(v.Bytes)
	return nil
}

package expr

import (
	"fmt"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zson"
)

// A ShaperTransform represents one of the different transforms that a
// shaper can apply.
type ShaperTransform int

const (
	Cast ShaperTransform = 1 << iota
	Fill
	Crop
	Order
)

type op int

const (
	copyPrimitive op = iota // copy field fromIndex from input record
	copyContainer
	castPrimitive // cast field fromIndex from input record
	null          // write null
	array         // build array
	set           // build set
	record        // build record
)

// A step is a recursive data structure encoding a series of
// copy/cast steps to be carried out over an input record.
type step struct {
	op        op
	fromIndex int
	castTypes struct{ from, to zng.Type } // for op == castPrimitive
	// if op == record, contains one op for each column.
	// if op == array, contains one op for all array elements.
	children []step
}

func (s *step) append(step step) {
	s.children = append(s.children, step)
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

func isCollectionType(t zng.Type) bool {
	if _, ok := t.(*zng.TypeArray); ok {
		return true
	}
	if _, ok := t.(*zng.TypeSet); ok {
		return true
	}
	return false
}

func innerType(t zng.Type) zng.Type {
	switch t := t.(type) {
	case *zng.TypeArray:
		return t.Type
	case *zng.TypeSet:
		return t.Type
	}
	return nil
}

func createStep(in, out zng.Type) (step, error) {
	switch {
	case in.ID() == out.ID():
		if zng.IsContainerType(in) {
			return step{op: copyContainer}, nil
		} else {
			return step{op: copyPrimitive}, nil
		}
	case zng.IsRecordType(in) && zng.IsRecordType(out):
		return createStepRecord(in.(*zng.TypeRecord), out.(*zng.TypeRecord))
	case zng.IsPrimitiveType(in) && zng.IsPrimitiveType(out):
		return step{op: castPrimitive, castTypes: struct{ from, to zng.Type }{in, out}}, nil
	case isCollectionType(in):
		if _, ok := out.(*zng.TypeArray); ok {
			return createStepArray(innerType(in), innerType(out))
		}
		if _, ok := out.(*zng.TypeSet); ok {
			return createStepSet(innerType(in), innerType(out))
		}
		fallthrough
	default:
		return step{}, fmt.Errorf("createStep incompatible column types %s and %s\n", in, out)
	}
}

func (s *step) castPrimitive(in zcode.Bytes, b *zcode.Builder) {
	if in == nil {
		b.AppendNull()
		return
	}
	pc := LookupPrimitiveCaster(s.castTypes.to)
	v, err := pc(zng.Value{s.castTypes.from, in})
	if err != nil {
		b.AppendNull()
		return
	}
	b.AppendPrimitive(v.Bytes)
}

func (s *step) build(in zcode.Bytes, b *zcode.Builder) {
	switch s.op {
	case copyPrimitive:
		b.AppendPrimitive(in)
	case copyContainer:
		b.AppendContainer(in)
	case castPrimitive:
		s.castPrimitive(in, b)
	case record:
		if in == nil {
			b.AppendNull()
			return
		}
		b.BeginContainer()
		s.buildRecord(in, b)
		b.EndContainer()
	case array:
		fallthrough
	case set:
		if in == nil {
			b.AppendNull()
			return
		}
		b.BeginContainer()
		iter := in.Iter()
		for !iter.Done() {
			zv, _, err := iter.Next()
			if err != nil {
				panic(err)
			}
			s.children[0].build(zv, b)
		}
		if s.op == set {
			b.TransformContainer(zng.NormalizeSet)
		}
		b.EndContainer()
	}
}

func (s *step) buildRecord(in zcode.Bytes, b *zcode.Builder) {
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
		step.build(bytes, b)
	}
}

// A shapeSpec is a per-input type ID "spec" that contains the output
// type and the op to create an output record.
type shapeSpec struct {
	typ  *zng.TypeRecord
	step step
}

type Shaper struct {
	zctx       *resolver.Context
	b          zcode.Builder
	fieldExpr  Evaluator
	typ        *zng.TypeRecord
	shapeSpecs map[int]shapeSpec // map from type ID to shapeSpec
	transforms ShaperTransform
}

// NewShaperType returns a shaper that will shape the result of fieldExpr
// to the provided typExpr. (typExpr should evaluate to a type value,
// e.g. a value of type TypeType).
func NewShaperType(zctx *resolver.Context, fieldExpr Evaluator, typ *zng.TypeRecord, tf ShaperTransform) (*Shaper, error) {
	return &Shaper{
		zctx:       zctx,
		fieldExpr:  fieldExpr,
		typ:        typ,
		shapeSpecs: make(map[int]shapeSpec),
		transforms: tf,
	}, nil
}

// NewShaper returns a shaper that will shape the result of fieldExpr
// to the provided typExpr. (typExpr should evaluate to a type value,
// e.g. a value of type TypeType).
func NewShaper(zctx *resolver.Context, fieldExpr, typExpr Evaluator, tf ShaperTransform) (*Shaper, error) {
	lit, ok := typExpr.(*Literal)
	if !ok {
		return nil, fmt.Errorf("shaping functions (crop, fill, cast, order) take a literal as second parameter")
	}

	// Z doesn't yet have type value literals, so we accept a
	// string literal and parse it. When type value lits are in,
	// this only change will be to require zng.TypeType rather
	// than zng.TypeString. Since a TypeType value holds the type
	// as zson, parsing with the zson.TypeTable below will be
	// identical.
	//
	// if lit.zv.Type != zng.TypeType {
	if lit.zv.Type != zng.TypeString {
		return nil, fmt.Errorf("shaper needs a type value as second parameter")
	}
	tt := zson.NewTypeTable(zctx)
	shapeToType, err := tt.LookupType(string(lit.zv.Bytes))
	if err != nil {
		return nil, fmt.Errorf("shaper could not parse type value literal: %s", err)
	}

	recType, isRecord := shapeToType.(*zng.TypeRecord)
	if !isRecord {
		return nil, fmt.Errorf("shaper needs a record type value as second parameter")
	}
	return NewShaperType(zctx, fieldExpr, recType, tf)
}

func (s *Shaper) Apply(in *zng.Record) (*zng.Record, error) {
	v, err := s.Eval(in)
	if err != nil {
		return nil, err
	}
	return zng.NewRecord(v.Type.(*zng.TypeRecord), v.Bytes), nil
}

func (s *Shaper) Eval(in *zng.Record) (zng.Value, error) {
	inVal, err := s.fieldExpr.Eval(in)
	if err != nil {
		return zng.Value{}, err
	}
	inType, ok := inVal.Type.(*zng.TypeRecord)
	if !ok {
		return inVal, nil
	}
	if _, ok := s.shapeSpecs[in.Type.ID()]; !ok {
		spec, err := s.createShapeSpec(inType, s.typ)
		if err != nil {
			return zng.Value{}, err
		}
		s.shapeSpecs[in.Type.ID()] = spec
	}
	spec := s.shapeSpecs[in.Type.ID()]
	if spec.typ.ID() == in.Type.ID() {
		return inVal, nil
	}

	s.b.Reset()
	spec.step.buildRecord(inVal.Bytes, &s.b)
	return zng.Value{spec.typ, s.b.Bytes()}, nil
}

func (s *Shaper) createShapeSpec(inType, spec *zng.TypeRecord) (shapeSpec, error) {
	var err error
	typ := inType
	if s.transforms&Cast > 0 {
		typ, err = s.castRecordType(typ, spec)
		if err != nil {
			return shapeSpec{}, err
		}
	}
	if s.transforms&Crop > 0 {
		typ, err = s.cropRecordType(typ, spec)
		if err != nil {
			return shapeSpec{}, err
		}
	}
	if s.transforms&Fill > 0 {
		typ, err = s.fillRecordType(typ, spec)
		if err != nil {
			return shapeSpec{}, err
		}
	}
	if s.transforms&Order > 0 {
		typ, err = s.orderRecordType(typ, spec)
		if err != nil {
			return shapeSpec{}, err
		}
	}
	step, err := createStepRecord(inType, typ)
	return shapeSpec{typ, step}, err
}

// cropRecordType applies a crop (as specified by the record type 'spec')
// to a record type and returns the resulting record type.
func (s *Shaper) cropRecordType(input, spec *zng.TypeRecord) (*zng.TypeRecord, error) {
	cols := make([]zng.Column, 0)
	for _, inCol := range input.Columns {
		ind, ok := spec.ColumnOfField(inCol.Name)
		if !ok {
			// 1. Field not in crop: drop.
			continue
		}

		specCol := spec.Columns[ind]
		switch {
		case !zng.IsRecordType(inCol.Type):
			// 2. Field is non-record in input: keep (regardless of crop record-ness)
			cols = append(cols, inCol)
		case zng.IsRecordType(specCol.Type):
			// 3. Both records: recurse
			out, err := s.cropRecordType(inCol.Type.(*zng.TypeRecord), specCol.Type.(*zng.TypeRecord))
			if err != nil {
				return nil, err
			}
			cols = append(cols, zng.Column{inCol.Name, out})
		default:
			// 4. record input but non-record in crop: keep crop
			cols = append(cols, specCol)

		}
	}
	return s.zctx.LookupTypeRecord(cols)
}

// orderRecordType applies a field order (as specified by the record type 'spec')
// to a record type and returns the resulting record type.
func (s *Shaper) orderRecordType(input, spec *zng.TypeRecord) (*zng.TypeRecord, error) {
	cols := make([]zng.Column, 0)
	// Simple order algorithm creates a list with all specified
	// 'order' fields present in input, followed by any other
	// fields that were in input but not specified in order. Two
	// examples:
	//
	// 1. (a b c d) order (c e b a) -> (c b a d)
	//
	// 2. (a b c d) order (d e c b) -> (d c b a)
	//
	// The second example with 'a' moving to the back suggests
	// that we may want to use a different algorithm where
	// unspecified fields "stay where they were". Specifically we
	// might prefer that the result be (a d c b). We will learn
	// with use, so starting with simpler algorithm for now.
	//
	for _, specCol := range spec.Columns {
		if ind, ok := input.ColumnOfField(specCol.Name); ok {
			inCol := input.Columns[ind]
			if zng.IsRecordType(inCol.Type) && zng.IsRecordType(specCol.Type) {
				if nested, err := s.orderRecordType(inCol.Type.(*zng.TypeRecord), specCol.Type.(*zng.TypeRecord)); err != nil {
					return nil, err
				} else {
					cols = append(cols, zng.Column{specCol.Name, nested})
				}
				continue
			}
			if isCollectionType(inCol.Type) && isCollectionType(specCol.Type) && zng.IsRecordType(innerType(inCol.Type)) && zng.IsRecordType(innerType(specCol.Type)) {
				if inner, err := s.orderRecordType(innerType(inCol.Type).(*zng.TypeRecord), innerType(specCol.Type).(*zng.TypeRecord)); err != nil {
					return nil, err
				} else {
					var err error
					var t zng.Type
					if _, ok := inCol.Type.(*zng.TypeArray); ok {
						t, err = s.zctx.LookupTypeArray(inner), nil
					} else {
						t, err = s.zctx.LookupTypeSet(inner), nil
					}
					if err != nil {
						return nil, err
					}
					cols = append(cols, zng.Column{specCol.Name, t})
				}
				continue
			}
			cols = append(cols, inCol)
		}
	}
	for _, inCol := range input.Columns {
		if !spec.HasField(inCol.Name) {
			cols = append(cols, inCol)
		}
	}
	return s.zctx.LookupTypeRecord(cols)
}

// fillRecordType applies a fill (as specified by the record type 'spec')
// to a record type and returns the resulting record type.
func (s *Shaper) fillRecordType(input, spec *zng.TypeRecord) (*zng.TypeRecord, error) {

	cols := make([]zng.Column, len(input.Columns), len(input.Columns)+len(spec.Columns))
	copy(cols, input.Columns)
	for _, specCol := range spec.Columns {
		if i, ok := input.ColumnOfField(specCol.Name); ok {
			// Field is present both in input and spec: recurse if
			// both records, or select appropriate type if not.
			if specRecType, ok := specCol.Type.(*zng.TypeRecord); ok {
				if inRecType, ok := input.Columns[i].Type.(*zng.TypeRecord); ok {
					filled, err := s.fillRecordType(inRecType, specRecType)
					if err != nil {
						return nil, err
					}
					cols[i] = zng.Column{specCol.Name, filled}
				} else {
					cols[i] = specCol
				}
			}
		} else {
			cols = append(cols, specCol)
		}
	}
	return s.zctx.LookupTypeRecord(cols)
}

// castRecordType applies a cast (as specified by the record type 'spec')
// to a record type and returns the resulting record type.
func (s *Shaper) castRecordType(input, spec *zng.TypeRecord) (*zng.TypeRecord, error) {
	cols := make([]zng.Column, 0)

	for _, inCol := range input.Columns {
		// For each input column, check if we have a matching
		// name in the cast spec.
		ind, ok := spec.ColumnOfField(inCol.Name)
		if !ok {
			// 1. No match: output type unmodified.
			cols = append(cols, inCol)
			continue
		}
		specCol := spec.Columns[ind]

		if _, ok := specCol.Type.(*zng.TypeMap); ok {
			return nil, fmt.Errorf("cannot yet use maps in shaping functions")
		}

		if inCol.Type.ID() == specCol.Type.ID() {
			// Field has same type in cast: output type unmodified.
			cols = append(cols, inCol)
			continue
		}
		castType, err := s.castType(inCol.Type, specCol.Type)
		if err != nil {
			return nil, err
		}
		cols = append(cols, zng.Column{inCol.Name, castType})

	}
	return s.zctx.LookupTypeRecord(cols)
}

func (c *Shaper) castType(inType, specType zng.Type) (zng.Type, error) {
	switch {
	case zng.IsRecordType(inType) && zng.IsRecordType(specType):
		// Matching field is a record: recurse.
		inRec := inType.(*zng.TypeRecord)
		castRec := specType.(*zng.TypeRecord)
		return c.castRecordType(inRec, castRec)
	case zng.IsPrimitiveType(inType) && zng.IsPrimitiveType(specType):
		// Matching field is a primitive: output type is cast type.
		if LookupPrimitiveCaster(specType) == nil {
			return nil, fmt.Errorf("cast to %s not implemented", specType)
		}
		return specType, nil
	case isCollectionType(inType) && isCollectionType(specType):
		out, err := c.castType(innerType(inType), innerType(specType))
		if err != nil {
			return nil, err
		}
		if _, ok := specType.(*zng.TypeArray); ok {
			return c.zctx.LookupTypeArray(out), nil
		}
		return c.zctx.LookupTypeSet(out), nil
	default:
		// Non-castable type pair with at least one
		// (non-record) container: output column is left
		// unchanged.
		return inType, nil
	}
}

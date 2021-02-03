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

type step int

const (
	copyPrimitive step = iota // copy field fromIndex from input record
	copyContainer
	castPrimitive // cast field fromIndex from input record
	null          // write null
	record        // record into record below us
)

// A op is a recursive data structure encoding a series of
// copy/cast steps to be carried out over an input record.
type op struct {
	op        step
	fromIndex int
	castTypes struct{ from, to zng.Type } // for op == castPrimitive
	record    []op                        // for op == record
}

func (s *op) append(step op) {
	s.record = append(s.record, step)
}

// create the op needed to build a record of type out from a
// record of type in. The two types must be compatible, meaning that
// the input type must be an unordered subset of the input type
// (where 'unordered' means that if the output type has record fields
// [a b] and the input type has fields [b a] that is ok). It is also
// ok for leaf primitive types to be different; if they are a casting
// step is inserted.
func createOp(in, out *zng.TypeRecord) (op, error) {
	o := op{op: record}
	for _, outCol := range out.Columns {
		ind, ok := in.ColumnOfField(outCol.Name)
		if !ok {
			o.append(op{op: null})
			continue
		}

		inCol := in.Columns[ind]

		switch {
		case inCol.Type.ID() == outCol.Type.ID():
			if zng.IsContainerType(inCol.Type) {
				o.append(op{fromIndex: ind, op: copyContainer})
			} else {
				o.append(op{fromIndex: ind, op: copyPrimitive})
			}
		case zng.IsRecordType(inCol.Type) && zng.IsRecordType(outCol.Type):
			step, err := createOp(inCol.Type.(*zng.TypeRecord), outCol.Type.(*zng.TypeRecord))
			if err != nil {
				return op{}, err
			}
			step.fromIndex = ind
			o.append(step)
		case zng.IsPrimitiveType(inCol.Type) && zng.IsPrimitiveType(outCol.Type):
			step := op{fromIndex: ind, op: castPrimitive, castTypes: struct{ from, to zng.Type }{inCol.Type, outCol.Type}}
			o.append(step)
		default:
			return op{}, fmt.Errorf("createOp incompatible column types %s and %s\n", inCol.Type, outCol.Type)
		}
	}
	return o, nil
}

func (s op) castPrimitive(in zcode.Bytes, b *zcode.Builder) {
	pc := LookupPrimitiveCaster(s.castTypes.to)
	v, err := pc(zng.Value{s.castTypes.from, in})
	if err != nil {
		b.AppendNull()
		return
	}
	b.AppendPrimitive(v.Bytes)
}

func (s op) buildRecord(in zcode.Bytes, b *zcode.Builder) {
	if s.op != record {
		panic("bad op")
	}
	for _, step := range s.record {

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

		switch step.op {
		case copyPrimitive:
			b.AppendPrimitive(bytes)
		case copyContainer:
			b.AppendContainer(bytes)
		case castPrimitive:
			step.castPrimitive(bytes, b)
		case record:
			b.BeginContainer()
			step.buildRecord(bytes, b)
			b.EndContainer()
		}
	}
}

// A shapeSpec is a per-input type ID "spec" that contains the output
// type and the op to create an output record.
type shapeSpec struct {
	typ *zng.TypeRecord
	op  op
}

type Shaper struct {
	zctx       *resolver.Context
	b          zcode.Builder
	fieldExpr  Evaluator
	typ        *zng.TypeRecord
	shapeSpecs map[int]shapeSpec // map from type ID to shapeSpec
	transforms ShaperTransform
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
	return &Shaper{
		zctx:       zctx,
		fieldExpr:  fieldExpr,
		typ:        recType,
		shapeSpecs: make(map[int]shapeSpec),
		transforms: tf,
	}, nil
}

func (c *Shaper) Eval(in *zng.Record) (zng.Value, error) {
	inVal, err := c.fieldExpr.Eval(in)
	if err != nil {
		return zng.Value{}, err
	}
	inType, ok := inVal.Type.(*zng.TypeRecord)
	if !ok {
		return inVal, nil
	}
	if _, ok := c.shapeSpecs[in.Type.ID()]; !ok {
		spec, err := c.createShapeSpec(inType, c.typ)
		if err != nil {
			return zng.Value{}, err
		}
		c.shapeSpecs[in.Type.ID()] = spec
	}
	spec := c.shapeSpecs[in.Type.ID()]
	if spec.typ.ID() == in.Type.ID() {
		return inVal, nil
	}

	c.b.Reset()
	spec.op.buildRecord(inVal.Bytes, &c.b)
	return zng.Value{spec.typ, c.b.Bytes()}, nil
}

func (c *Shaper) createShapeSpec(inType, spec *zng.TypeRecord) (shapeSpec, error) {
	var err error
	typ := inType
	if c.transforms&Cast > 0 {
		typ, err = c.castRecordType(typ, spec)
		if err != nil {
			return shapeSpec{}, err
		}
	}
	if c.transforms&Crop > 0 {
		typ, err = c.cropRecordType(typ, spec)
		if err != nil {
			return shapeSpec{}, err
		}
	}
	if c.transforms&Fill > 0 {
		typ, err = c.fillRecordType(typ, spec)
		if err != nil {
			return shapeSpec{}, err
		}
	}
	if c.transforms&Order > 0 {
		typ, err = c.orderRecordType(typ, spec)
		if err != nil {
			return shapeSpec{}, err
		}
	}
	op, err := createOp(inType, typ)
	return shapeSpec{typ, op}, err
}

// cropRecordType applies a crop (as specified by the record type 'spec')
// to a record type and returns the resulting record type.
func (c *Shaper) cropRecordType(input, spec *zng.TypeRecord) (*zng.TypeRecord, error) {
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
			out, err := c.cropRecordType(inCol.Type.(*zng.TypeRecord), specCol.Type.(*zng.TypeRecord))
			if err != nil {
				return nil, err
			}
			cols = append(cols, zng.Column{inCol.Name, out})
		default:
			// 4. record input but non-record in crop: keep crop
			cols = append(cols, specCol)

		}
	}
	return c.zctx.LookupTypeRecord(cols)
}

// orderRecordType applies a field order (as specified by the record type 'spec')
// to a record type and returns the resulting record type.
func (c *Shaper) orderRecordType(input, spec *zng.TypeRecord) (*zng.TypeRecord, error) {
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
				if nested, err := c.orderRecordType(inCol.Type.(*zng.TypeRecord), specCol.Type.(*zng.TypeRecord)); err != nil {
					return nil, err
				} else {
					cols = append(cols, zng.Column{specCol.Name, nested})
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
	return c.zctx.LookupTypeRecord(cols)
}

// fillRecordType applies a fill (as specified by the record type 'spec')
// to a record type and returns the resulting record type.
func (c *Shaper) fillRecordType(input, spec *zng.TypeRecord) (*zng.TypeRecord, error) {
	cols := make([]zng.Column, 0)
	names := make([]string, 0)
	// Compute union list of fields. This takes all the input
	// fields and adds the filled fields at the end. For
	// standalone uses of 'fill()' (as opposed to shape() which
	// includes order)', this might be surprising and it would
	// likely be preferable to have the filled fields be woven
	// into the input records. This can come later as needed.
	for _, col := range input.Columns {
		names = append(names, col.Name)
	}
	for _, specCol := range spec.Columns {
		if !input.HasField(specCol.Name) {
			names = append(names, specCol.Name)
		}
	}
	// Now, figure out filled type for the fields.
	for _, name := range names {
		var inCol, specCol zng.Column
		i, inputOk := input.ColumnOfField(name)
		if inputOk {
			inCol = input.Columns[i]
		}
		i, specOk := spec.ColumnOfField(name)
		if specOk {
			specCol = spec.Columns[i]
		}
		if !inputOk {
			// Field not in input: fill.
			cols = append(cols, specCol)
			continue
		}
		if !specOk {
			// Field not in spec: input passes through.
			cols = append(cols, inCol)
			continue
		}

		// Field is present both in input and spec: recurse if
		// both records, or select appropriate type if not.
		_, inIsRec := inCol.Type.(*zng.TypeRecord)
		_, specIsRec := specCol.Type.(*zng.TypeRecord)
		switch {
		case inIsRec && specIsRec:
			col, err := c.fillRecordType(inCol.Type.(*zng.TypeRecord), specCol.Type.(*zng.TypeRecord))
			if err != nil {
				return nil, err
			}
			cols = append(cols, zng.Column{inCol.Name, col})

		case !inIsRec && !specIsRec:
			cols = append(cols, inCol)

		case inIsRec && !specIsRec:
			cols = append(cols, inCol)

		case !inIsRec && specIsRec:
			cols = append(cols, specCol)

		}
	}
	return c.zctx.LookupTypeRecord(cols)
}

// castRecordType applies a cast (as specified by the record type 'spec')
// to a record type and returns the resulting record type.
func (c *Shaper) castRecordType(input, spec *zng.TypeRecord) (*zng.TypeRecord, error) {
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

		inRec, inIsRec := inCol.Type.(*zng.TypeRecord)
		castRec, castIsRec := specCol.Type.(*zng.TypeRecord)

		switch {
		case inCol.Type.ID() == specCol.Type.ID():
			// 2. Field has same type in cast: output type unmodified.
			cols = append(cols, inCol)
		case inIsRec && castIsRec:
			// 3. Matching field is a record: recurse.
			out, err := c.castRecordType(inRec, castRec)
			if err != nil {
				return nil, err
			}
			cols = append(cols, zng.Column{inCol.Name, out})
		case zng.IsPrimitiveType(inCol.Type) && zng.IsPrimitiveType(specCol.Type):
			// 4. Matching field is a primitive: output type is cast type.
			if LookupPrimitiveCaster(specCol.Type) == nil {
				return nil, fmt.Errorf("cast to %s not implemented", specCol.Type)
			}
			cols = append(cols, zng.Column{inCol.Name, specCol.Type})
		default:
			// 5. Non-castable type pair with at least one
			// (non-record) container: output column is left
			// unchanged.  Note that eventually, we should
			// recognize and cast e.g. array[string] to array[ip].
			// xxx before merge: file issue for this and mention
			// it here.
			cols = append(cols, inCol)
		}
	}
	return c.zctx.LookupTypeRecord(cols)
}

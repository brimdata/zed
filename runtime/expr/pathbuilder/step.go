package pathbuilder

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Step interface {
	Build(*zed.Context, *zcode.Builder, zcode.Bytes, []zed.Value) (zed.Type, error)
}

type recordStep struct {
	isBase bool
	getter getter
	fields []recordField
}

type recordField struct {
	index int
	name  string
	step  Step
}

func newRecordStep(fields []zed.Field) *recordStep {
	var s recordStep
	for i, f := range fields {
		f := recordField{
			index: i,
			name:  f.Name,
			step:  newLeafStep(f.Type, -1),
		}
		s.fields = append(s.fields, f)
	}
	return &s
}

func (s *recordStep) lookup(name string) int {
	i := slices.IndexFunc(s.fields, func(f recordField) bool {
		return f.name == name
	})
	if i >= 0 {
		return i
	}
	n := len(s.fields)
	s.fields = append(s.fields, recordField{name: name, index: -1})
	return n
}

func (s *recordStep) Build(zctx *zed.Context, b *zcode.Builder, in zcode.Bytes, vals []zed.Value) (zed.Type, error) {
	if !s.isBase {
		b.BeginContainer()
		defer b.EndContainer()
	}
	s.getter = newGetter(in)
	fields := make([]zed.Field, 0, len(s.fields))
	for _, field := range s.fields {
		var vb zcode.Bytes
		if field.index != -1 {
			var err error
			if vb, err = s.getter.nth(field.index); err != nil {
				return nil, err
			}
		}
		typ, err := field.step.Build(zctx, b, vb, vals)
		if err != nil {
			return nil, err
		}
		fields = append(fields, zed.NewField(field.name, typ))
	}
	// XXX If there are no downstream vector or map elements we can cache this
	// result.
	return zctx.LookupTypeRecord(fields)
}

type vectorStep struct {
	elems  []vectorElem
	getter getter
	inner  zed.Type
	isSet  bool
	isBase bool
}

type vectorElem struct {
	index int
	step  Step
}

func (s *vectorStep) lookup(i int) int {
	elem := vectorElem{index: i}
	at, ok := slices.BinarySearchFunc(s.elems, elem, func(a, b vectorElem) int {
		return cmp.Compare(a.index, b.index)
	})
	if !ok {
		s.elems = slices.Insert(s.elems, at, elem)
	}
	return at
}

func (s *vectorStep) Build(zctx *zed.Context, b *zcode.Builder, in zcode.Bytes, vals []zed.Value) (zed.Type, error) {
	if !s.isBase {
		b.BeginContainer()
		defer b.EndContainer()
	}
	elems := s.elems
	it := in.Iter()
	var types []zed.Type
	for i := 0; !it.Done(); i++ {
		typ, vb := s.inner, it.Next()
		if len(elems) > 0 && i == elems[0].index {
			var err error
			typ, err = elems[0].step.Build(zctx, b, vb, vals)
			if err != nil {
				return nil, err
			}
			elems = elems[1:]
		} else {
			b.Append(vb)
		}
		types = append(types, typ)
	}
	if len(elems) > 0 {
		return nil, fmt.Errorf("element out of bounds %d", elems[0].index)
	}
	inner := normalizeVectorElems(zctx, types, b)
	if s.isSet {
		b.TransformContainer(zed.NormalizeSet)
		return zctx.LookupTypeSet(inner), nil
	}
	return zctx.LookupTypeArray(inner), nil
}

func normalizeVectorElems(zctx *zed.Context, types []zed.Type, b *zcode.Builder) zed.Type {
	i := slices.IndexFunc(types, func(t zed.Type) bool {
		_, ok := zed.TypeUnder(t).(*zed.TypeUnion)
		return ok
	})
	if i >= 0 {
		// Untag union values.
		b.TransformContainer(func(bytes zcode.Bytes) zcode.Bytes {
			var b2 zcode.Builder
			for i, it := 0, bytes.Iter(); !it.Done(); i++ {
				vb := it.Next()
				if union, ok := zed.TypeUnder(types[i]).(*zed.TypeUnion); ok {
					types[i], vb = union.Untag(vb)
				}
				b2.Append(vb)
			}
			return b2.Bytes()
		})
	}
	unique := zed.UniqueTypes(slices.Clone(types))
	if len(unique) == 1 {
		return unique[0]
	}
	union := zctx.LookupTypeUnion(unique)
	b.TransformContainer(func(bytes zcode.Bytes) zcode.Bytes {
		var b2 zcode.Builder
		for i, it := 0, bytes.Iter(); !it.Done(); i++ {
			zed.BuildUnion(&b2, union.TagOf(types[i]), it.Next())
		}
		return b2.Bytes()
	})
	return union
}

type leafStep struct {
	inputIndex int
	typ        zed.Type
}

func newLeafStep(typ zed.Type, inputIndex int) *leafStep {
	return &leafStep{typ: typ, inputIndex: inputIndex}
}

func (s *leafStep) Build(zctx *zed.Context, b *zcode.Builder, in zcode.Bytes, vals []zed.Value) (zed.Type, error) {
	if s.inputIndex != -1 {
		b.Append(vals[s.inputIndex].Bytes())
	} else {
		b.Append(in)
	}
	return s.typ, nil
}

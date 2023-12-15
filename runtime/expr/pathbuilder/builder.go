package pathbuilder

import (
	"errors"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/expr/dynfield"
)

type builder struct {
	inputCount int
	base       Step
}

func New(base zed.Type, paths []dynfield.Path, leafs []zed.Value) (Step, error) {
	if len(paths) != len(leafs) {
		return nil, errors.New("paths and leafs must be the same length")
	}
	b := &builder{base: newLeafStep(base, -1)}
	for i, p := range paths {
		if err := b.Put(p, leafs[i].Type); err != nil {
			return nil, err
		}
	}
	return b.base, nil
}

func (m *builder) Put(p dynfield.Path, leaf zed.Type) error {
	defer func() { m.inputCount++ }()
	return m.put(&m.base, p, leaf)
}

func (m *builder) put(parent *Step, p dynfield.Path, typ zed.Type) error {
	// Actually let's do this differently. If current is a string then we are
	// putting to a record. When we support maps we'll need to check for that.
	if p[0].IsString() {
		return m.putRecord(parent, p, typ)
	}
	// This could be for a map or a set but keep it simple for now.
	if zed.IsInteger(p[0].Type.ID()) {
		return m.putVector(parent, p, typ)
	}
	// if zed.TypeUnder(parent.typeof())
	return errors.New("unsupported types")
}

func (m *builder) putRecord(s *Step, p dynfield.Path, typ zed.Type) error {
	current, p := p[0], p[1:]
	rstep, ok := (*s).(*recordStep)
	if !ok {
		// If this is a leafStep with a type of record than we need to
		// initialize a recordStep with fields, otherwise just replace this will
		// a recordStep.
		var fields []zed.Field
		if lstep, ok := (*s).(*leafStep); ok && zed.TypeRecordOf(lstep.typ) != nil {
			fields = zed.TypeRecordOf(lstep.typ).Fields
		}
		rstep = newRecordStep(fields)
		if *s == m.base {
			rstep.isBase = true
		}
		*s = rstep
	}
	i := rstep.lookup(current.AsString())
	field := &rstep.fields[i]
	if len(p) == 0 {
		field.step = newLeafStep(typ, m.inputCount)
		return nil
	}
	return m.put(&field.step, p, typ)
}

func (m *builder) putVector(s *Step, p dynfield.Path, typ zed.Type) error {
	current, p := p[0], p[1:]
	vstep, ok := (*s).(*vectorStep)
	if !ok {
		// If this is a leafStep with a type of array than we need to
		// initialize a arrayStep with fields, otherwise just replace this with
		// an arrayStep.
		vstep = &vectorStep{}
		if lstep, ok := (*s).(*leafStep); ok && zed.InnerType(lstep.typ) != nil {
			vstep.inner = zed.InnerType(lstep.typ)
			_, vstep.isSet = zed.TypeUnder(lstep.typ).(*zed.TypeSet)
		}
		if *s == m.base {
			vstep.isBase = true
		}
		*s = vstep
	}
	at := vstep.lookup(int(current.AsInt()))
	elem := &vstep.elems[at]
	if len(p) == 0 {
		elem.step = newLeafStep(typ, m.inputCount)
		return nil
	}
	return m.put(&elem.step, p, typ)
}

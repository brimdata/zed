package tzngio

import (
	"errors"
	"fmt"
	"strings"

	"github.com/brimdata/zq/zcode"
	"github.com/brimdata/zq/zng"
)

// XXX this comes from the old zng.Builder code and should be replaced
// with a parse of Z literals using the PEG parser.  Currently, just the
// microindex logic uses this to parse key parameters.

type builder struct {
	*zcode.Builder
}

func ParseKeys(typ *zng.TypeRecord, vals ...string) (*zng.Record, error) {
	b := &builder{zcode.NewBuilder()}
	out, err := b.parseRecord(typ, vals)
	if err != nil && err != zng.ErrIncomplete {
		return nil, err
	}
	if len(out) != 0 {
		return nil, fmt.Errorf("too many values (%d) supplied for type: %s", len(vals), typ.ZSON())
	}
	// Note that t.rec.nonvolatile is false so anything downstream
	// will have to copy the record and we can re-use the record value
	// between subsequent calls.
	r := zng.NewRecord(typ, b.Bytes())
	// We do a final type check to make sure everything is good.  In particular,
	// there are no checks below to ensure that set vals comply with the
	// ordering constraint.  XXX we could order them automatically if they
	// are not sorted, but we don't do this yet.
	if typErr := r.TypeCheck(); typErr != nil {
		// type error overrides ErrIncomplete
		err = typErr
	}
	return r, err
}

func (b *builder) parseRecord(typ *zng.TypeRecord, in []string) ([]string, error) {
	var err error
	for _, col := range typ.Columns {
		if len(in) == 0 {
			err = zng.ErrIncomplete
			b.appendUnset(col.Type)
			continue
		}

		switch v := zng.AliasOf(col.Type).(type) {
		case *zng.TypeRecord:
			b.BeginContainer()
			in, err = b.parseRecord(v, in)
			if err != nil && err != zng.ErrIncomplete {
				return nil, err
			}
			b.EndContainer()
		case *zng.TypeArray:
			b.BeginContainer()
			if err := b.parseArray(v, in[0]); err != nil {
				return nil, err
			}
			b.EndContainer()
			in = in[1:]
		case *zng.TypeSet:
			b.BeginContainer()
			if err := b.parseSet(v, in[0]); err != nil {
				return nil, err
			}
			b.EndContainer()
			in = in[1:]
		case *zng.TypeUnion:
			// XXX need a value syntax that indicates which type to use
			return nil, errors.New("union values not yet supported")
		default:
			if err := b.parsePrimitive(v, in[0]); err != nil {
				return nil, err
			}
			in = in[1:]
		}
	}
	return in, err
}

func (b *builder) parseArray(typ *zng.TypeArray, in string) error {
	inner := zng.InnerType(zng.AliasOf(typ))
	if len(in) == 0 {
		return nil
	}
	if in[0] != '[' || in[len(in)-1] != ']' {
		return ErrBadFormat
	}
	in = in[1 : len(in)-1]
	if len(in) == 0 {
		// empty array
		b.appendUnset(inner)
		return nil
	}
	//XXX for now just use simple comma rule, which means comman
	// cannot be embedded in a set value here.  we need a recursive
	// descent parser like the type parser to d this correctly or
	// change all this to built from an AST literal object parsed
	// by the zql parser.
	for _, val := range strings.Split(in, ",") {
		switch v := inner.(type) {
		case *zng.TypeRecord:
			// XXX need a syntax for this, and recursive descent
			return errors.New("array of records value not yet supported")
		case *zng.TypeArray:
			// XXX need a syntax for this, and recursive descent
			return errors.New("array of arrays value not yet supported")
		case *zng.TypeSet:
			// XXX need a syntax for this, and recursive descent
			return errors.New("array of sets value not yet supported")
		case *zng.TypeUnion:
			// XXX need a syntax for this, and recursive descent
			return errors.New("array of unions value not yet supported")
		default:
			b.parsePrimitive(v, val)
		}
	}
	return nil
}

func (b *builder) parseSet(typ *zng.TypeSet, in string) error {
	inner := zng.InnerType(zng.AliasOf(typ))
	if len(in) == 0 {
		return nil
	}
	if in[0] != '[' || in[len(in)-1] != ']' {
		return ErrBadFormat
	}
	in = in[1 : len(in)-1]
	if len(in) == 0 {
		// empty set
		b.appendUnset(inner)
		return nil
	}
	//XXX for now just use simple comma rule, which means comman
	// cannot be embedded in a set value here.  we need a recursive
	// descent parser like the type parser to d this correctly
	for _, val := range strings.Split(in, ",") {
		// we a don't enforce ordering here but rely on top-level
		// here doing a recordCheck
		if err := b.parsePrimitive(inner, val); err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) appendUnset(typ zng.Type) {
	if zng.IsContainerType(typ) {
		b.AppendContainer(nil)
	} else {
		b.AppendPrimitive(nil)
	}
}

func (b *builder) parsePrimitive(typ zng.Type, val string) error {
	body, err := ParseValue(typ, []byte(val))
	if err != nil {
		return err
	}
	b.AppendPrimitive(body)
	return nil
}

package shape

import (
	"errors"
	"hash/maphash"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/sam/op/spill"
	"github.com/brimdata/zed/zcode"
)

type Shaper struct {
	zctx        *zed.Context
	memMaxBytes int

	nbytes     int
	typeAnchor map[zed.Type]*anchor
	anchors    map[uint64]*anchor
	recode     map[zed.Type]*zed.TypeRecord
	spiller    *spill.File
	hash       maphash.Hash
	val        zed.Value
	vals       []zed.Value
	readArena  *zed.Arena
	valsArena  *zed.Arena
}

type anchor struct {
	typ      *zed.TypeRecord
	fields   []zed.Field
	integers []integer
	next     *anchor
}

type integer struct {
	signed   bool
	unsigned bool
}

func nulltype(t zed.Type) bool {
	return zed.TypeUnder(t) == zed.TypeNull
}

func (a *anchor) match(fields []zed.Field) bool {
	if len(fields) != len(a.fields) {
		return false
	}
	for k, f := range a.fields {
		in := fields[k]
		if f.Type != in.Type && !nulltype(f.Type) && !nulltype(in.Type) {
			return false
		}
	}
	return true
}

func (a *anchor) mixIn(fields []zed.Field) {
	for k, f := range a.fields {
		if nulltype(f.Type) {
			a.fields[k].Type = fields[k].Type
		}
	}
}

func (i *integer) check(val zed.Value) {
	id := val.Type().ID()
	if zed.IsInteger(id) || id == zed.IDNull {
		return
	}
	if !zed.IsFloat(id) {
		i.signed = false
		i.unsigned = false
		return
	}
	f := val.Float()
	//XXX We could track signed vs unsigned and overflow,
	// but for now, we leave it as float64 unless we can
	// guarantee int64.
	// for now, we don't handle these corner cases
	if i.signed && f != float64(int64(f)) {
		i.signed = false
	}
	if i.unsigned && f != float64(uint64(f)) {
		i.unsigned = false
	}
}

func (a *anchor) updateInts(rec *zed.Value) error {
	arena := zed.NewArena()
	defer arena.Unref()
	it := rec.Bytes().Iter()
	for k, f := range rec.Fields() {
		a.integers[k].check(arena.New(f.Type, it.Next()))
	}
	return nil
}

func (a *anchor) recodeType() []zed.Field {
	var fields []zed.Field
	for k, f := range a.typ.Fields {
		if i := a.integers[k]; i.signed {
			f.Type = zed.TypeInt64
		} else if i.unsigned {
			f.Type = zed.TypeUint64
		}
		fields = append(fields, f)
	}
	return fields
}

func (a *anchor) needRecode() []zed.Field {
	for _, i := range a.integers {
		if i.signed || i.unsigned {
			return a.recodeType()
		}
	}
	return nil
}

func NewShaper(zctx *zed.Context, memMaxBytes int) *Shaper {
	return &Shaper{
		zctx:        zctx,
		memMaxBytes: memMaxBytes,
		anchors:     make(map[uint64]*anchor),
		typeAnchor:  make(map[zed.Type]*anchor),
		recode:      make(map[zed.Type]*zed.TypeRecord),
		readArena:   zed.NewArena(),
		valsArena:   zed.NewArena(),
	}
}

// Close removes the receiver's temporary file if it created one.
func (s *Shaper) Close() error {
	if s.spiller != nil {
		return s.spiller.CloseAndRemove()
	}
	return nil
}

func hash(h *maphash.Hash, fields []zed.Field) uint64 {
	h.Reset()
	for _, f := range fields {
		h.WriteString(f.Name)
	}
	return h.Sum64()
}

func (s *Shaper) lookupAnchor(fields []zed.Field) *anchor {
	h := hash(&s.hash, fields)
	for a := s.anchors[h]; a != nil; a = a.next {
		if a.match(fields) {
			return a
		}
	}
	return nil
}

func (s *Shaper) newAnchor(fields []zed.Field) *anchor {
	h := hash(&s.hash, fields)
	a := &anchor{
		fields:   fields,
		integers: make([]integer, len(fields)),
		next:     s.anchors[h],
	}
	s.anchors[h] = a
	for k := range a.integers {
		// start off as int64 and invalidate when we see
		// a value that doesn't fit.
		a.integers[k].unsigned = true
		a.integers[k].signed = true
	}
	return a
}

func (s *Shaper) update(rec *zed.Value) {
	if a, ok := s.typeAnchor[rec.Type()]; ok {
		a.updateInts(rec)
		return
	}
	fields := rec.Fields()
	a := s.lookupAnchor(fields)
	if a == nil {
		a = s.newAnchor(fields)
	} else {
		a.mixIn(fields)
	}
	a.updateInts(rec)
	s.typeAnchor[rec.Type()] = a
}

func (s *Shaper) needRecode(typ zed.Type) (*zed.TypeRecord, error) {
	target, ok := s.recode[typ]
	if !ok {
		a := s.typeAnchor[typ]
		fields := a.needRecode()
		if fields != nil {
			var err error
			target, err = s.zctx.LookupTypeRecord(fields)
			if err != nil {
				return nil, err
			}
		}
		s.recode[typ] = target
	}
	return target, nil
}

func (s *Shaper) lookupType(in zed.Type) (*zed.TypeRecord, error) {
	a, ok := s.typeAnchor[in]
	if !ok {
		return nil, errors.New("Shaper: unencountered type (this is a bug)")
	}
	typ := a.typ
	if typ == nil {
		var err error
		typ, err = s.zctx.LookupTypeRecord(a.fields)
		if err != nil {
			return nil, err
		}
		a.typ = typ
	}
	return typ, nil
}

// Write buffers rec. If called after Read, Write panics.
func (s *Shaper) Write(rec zed.Value) error {
	if s.spiller != nil {
		return s.spiller.Write(rec)
	}
	if err := s.stash(rec); err != nil {
		return err
	}
	s.update(&rec)
	return nil
}

func (s *Shaper) stash(rec zed.Value) error {
	s.nbytes += len(rec.Bytes())
	if s.nbytes >= s.memMaxBytes {
		var err error
		s.spiller, err = spill.NewTempFile()
		if err != nil {
			return err
		}
		for _, rec := range s.vals {
			if err := s.spiller.Write(rec); err != nil {
				return err
			}
		}
		s.vals = nil
		return s.spiller.Write(rec)
	}
	s.vals = append(s.vals, rec.Copy(s.valsArena))
	return nil
}

func (s *Shaper) Read() (*zed.Value, error) {
	rec, err := s.next()
	if rec == nil || err != nil {
		return nil, err
	}
	typ, err := s.lookupType(rec.Type())
	if err != nil {
		return nil, err
	}
	bytes := rec.Bytes()
	targetType, err := s.needRecode(rec.Type())
	if err != nil {
		return nil, err
	}
	if targetType != nil {
		if bytes, err = recode(typ.Fields, targetType.Fields, bytes); err != nil {
			return nil, err
		}
		typ = targetType
	}
	s.readArena.Reset()
	s.val = s.readArena.New(typ, bytes)
	return &s.val, nil
}

func recode(from, to []zed.Field, bytes zcode.Bytes) (zcode.Bytes, error) {
	out := make(zcode.Bytes, 0, len(bytes))
	it := bytes.Iter()
	for k, fromField := range from {
		b := it.Next()
		toType := to[k].Type
		if fromField.Type != toType && b != nil {
			if fromField.Type != zed.TypeFloat64 {
				return nil, errors.New("shape: can't recode from non float64")
			}
			f := zed.DecodeFloat64(b)
			if toType == zed.TypeInt64 {
				b = zed.EncodeInt(int64(f))
			} else if toType == zed.TypeUint64 {
				b = zed.EncodeUint(uint64(f))
			} else {
				return nil, errors.New("internal error: can't recode from to non-integer")
			}
		}
		out = zcode.Append(out, b)
	}
	return out, nil
}

func (s *Shaper) next() (*zed.Value, error) {
	if s.spiller != nil {
		return s.spiller.Read()
	}
	var rec *zed.Value
	if len(s.vals) > 0 {
		rec = &s.vals[0]
		s.vals = s.vals[1:]
	}
	return rec, nil

}

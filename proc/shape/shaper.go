package shape

import (
	"errors"
	"hash/maphash"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/proc/spill"
	"github.com/brimdata/zed/zcode"
)

type Shaper struct {
	zctx        *zed.Context
	memMaxBytes int

	nbytes     int
	queue      []*zed.Value
	typeAnchor map[zed.Type]*anchor
	anchors    map[uint64]*anchor
	recode     map[zed.Type]*zed.TypeRecord
	spiller    *spill.File
	hash       maphash.Hash
	vals       []*zed.Value
}

type anchor struct {
	typ      *zed.TypeRecord
	columns  []zed.Column
	integers []integer
	next     *anchor
}

type integer struct {
	signed   bool
	unsigned bool
}

func nulltype(t zed.Type) bool {
	return zed.AliasOf(t) == zed.TypeNull
}

func (a *anchor) match(cols []zed.Column) bool {
	if len(cols) != len(a.columns) {
		return false
	}
	for k, c := range a.columns {
		in := cols[k]
		if c.Type != in.Type && !nulltype(c.Type) && !nulltype(in.Type) {
			return false
		}
	}
	return true
}

func (a *anchor) mixIn(cols []zed.Column) {
	for k, c := range a.columns {
		if nulltype(c.Type) {
			a.columns[k].Type = cols[k].Type
		}
	}
}

func (i *integer) check(zv zed.Value) {
	id := zv.Type.ID()
	if zed.IsInteger(id) || id == zed.IDNull {
		return
	}
	if !zed.IsFloat(id) {
		i.signed = false
		i.unsigned = false
		return
	}
	f, _ := zed.DecodeFloat64(zv.Bytes)
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
	it := rec.Bytes.Iter()
	for k, c := range rec.Columns() {
		bytes, _ := it.Next()
		zv := zed.Value{c.Type, bytes}
		a.integers[k].check(zv)
	}
	return nil
}

func (a *anchor) recodeType() []zed.Column {
	var cols []zed.Column
	for k, c := range a.typ.Columns {
		if i := a.integers[k]; i.signed {
			c.Type = zed.TypeInt64
		} else if i.unsigned {
			c.Type = zed.TypeUint64
		}
		cols = append(cols, c)
	}
	return cols
}

func (a *anchor) needRecode() []zed.Column {
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
	}
}

// Close removes the receiver's temporary file if it created one.
func (s *Shaper) Close() error {
	if s.spiller != nil {
		return s.spiller.CloseAndRemove()
	}
	return nil
}

func hash(h *maphash.Hash, cols []zed.Column) uint64 {
	h.Reset()
	for _, c := range cols {
		h.WriteString(c.Name)
	}
	return h.Sum64()
}

func (s *Shaper) lookupAnchor(columns []zed.Column) *anchor {
	h := hash(&s.hash, columns)
	for a := s.anchors[h]; a != nil; a = a.next {
		if a.match(columns) {
			return a
		}
	}
	return nil
}

func (s *Shaper) newAnchor(columns []zed.Column) *anchor {
	h := hash(&s.hash, columns)
	a := &anchor{
		columns:  columns,
		integers: make([]integer, len(columns)),
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
	if a, ok := s.typeAnchor[rec.Type]; ok {
		a.updateInts(rec)
		return
	}
	columns := rec.Columns()
	a := s.lookupAnchor(columns)
	if a == nil {
		a = s.newAnchor(columns)
	} else {
		a.mixIn(columns)
	}
	a.updateInts(rec)
	s.typeAnchor[rec.Type] = a
}

func (s *Shaper) needRecode(typ zed.Type) (*zed.TypeRecord, error) {
	target, ok := s.recode[typ]
	if !ok {
		a := s.typeAnchor[typ]
		cols := a.needRecode()
		if cols != nil {
			var err error
			target, err = s.zctx.LookupTypeRecord(cols)
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
		typ, err = s.zctx.LookupTypeRecord(a.columns)
		if err != nil {
			return nil, err
		}
		a.typ = typ
	}
	return typ, nil
}

// Write buffers rec. If called after Read, Write panics.
func (s *Shaper) Write(rec *zed.Value) error {
	if s.spiller != nil {
		return s.spiller.Write(rec)
	}
	if err := s.stash(rec); err != nil {
		return err
	}
	s.update(rec)
	return nil
}

func (s *Shaper) stash(rec *zed.Value) error {
	s.nbytes += len(rec.Bytes)
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
	s.vals = append(s.vals, rec.Copy())
	return nil
}

func (s *Shaper) Read() (*zed.Value, error) {
	rec, err := s.next()
	if rec == nil || err != nil {
		return nil, err
	}
	typ, err := s.lookupType(rec.Type)
	if err != nil {
		return nil, err
	}
	bytes := rec.Bytes
	targetType, err := s.needRecode(rec.Type)
	if err != nil {
		return nil, err
	}
	if targetType != nil {
		if bytes, err = recode(typ.Columns, targetType.Columns, bytes); err != nil {
			return nil, err
		}
		typ = targetType
	}
	return zed.NewValue(typ, bytes), nil
}

func recode(from, to []zed.Column, bytes zcode.Bytes) (zcode.Bytes, error) {
	out := make(zcode.Bytes, 0, len(bytes))
	it := bytes.Iter()
	for k, fromCol := range from {
		b, container := it.Next()
		toType := to[k].Type
		if fromCol.Type != toType && b != nil {
			if fromCol.Type != zed.TypeFloat64 {
				return nil, errors.New("shape: can't recode from non float64")
			}
			f, _ := zed.DecodeFloat64(b)
			if toType == zed.TypeInt64 {
				b = zed.EncodeInt(int64(f))
			} else if toType == zed.TypeUint64 {
				b = zed.EncodeUint(uint64(f))
			} else {
				return nil, errors.New("internal error: can't recode from to non-integer")
			}
		}
		out = zcode.AppendAs(out, container, b)
	}
	return out, nil
}

func (s *Shaper) next() (*zed.Value, error) {
	if s.spiller != nil {
		return s.spiller.Read()
	}
	var rec *zed.Value
	if len(s.vals) > 0 {
		rec = s.vals[0]
		s.vals = s.vals[1:]
	}
	return rec, nil

}

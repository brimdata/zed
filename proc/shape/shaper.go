package shape

import (
	"errors"
	"hash/maphash"

	"github.com/brimsec/zq/proc/spill"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Shaper struct {
	zctx        *resolver.Context
	memMaxBytes int

	nbytes     int
	queue      []*zng.Record
	typeAnchor map[zng.Type]*anchor
	anchors    map[uint64]*anchor
	recode     map[zng.Type]*zng.TypeRecord
	spiller    *spill.File
	hash       maphash.Hash
	recs       []*zng.Record
}

type anchor struct {
	typ      *zng.TypeRecord
	columns  []zng.Column
	integers []integer
	next     *anchor
}

type integer struct {
	signed   bool
	unsigned bool
}

func nulltype(t zng.Type) bool {
	return zng.AliasedType(t) == zng.TypeNull
}

func (a *anchor) match(cols []zng.Column) bool {
	if len(cols) != len(a.columns) {
		return false
	}
	for k, c := range a.columns {
		in := cols[k]
		if c.Type == in.Type || nulltype(c.Type) || nulltype(in.Type) {
			continue
		}
		return false
	}
	return true
}

func (a *anchor) mixIn(cols []zng.Column) {
	for k, c := range a.columns {
		if nulltype(c.Type) {
			a.columns[k].Type = cols[k].Type
		}
	}
}

func (i *integer) check(zv zng.Value) {
	id := zv.Type.ID()
	if zng.IsInteger(id) || nulltype(zv.Type) {
		return
	}
	if !zng.IsFloat(id) {
		i.signed = false
		i.unsigned = false
	}
	f, _ := zng.DecodeFloat64(zv.Bytes)
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

func (a *anchor) updateInts(rec *zng.Record) error {
	it := rec.Raw.Iter()
	for k := range a.columns {
		bytes, _, err := it.Next()
		if err != nil {
			return err
		}
		zv := zng.Value{rec.Type.Columns[k].Type, bytes}
		a.integers[k].check(zv)
	}
	return nil
}

func (a *anchor) recodeType() []zng.Column {
	var cols []zng.Column
	for k, c := range a.typ.Columns {
		i := a.integers[k]
		if i.signed {
			c.Type = zng.TypeInt64
		} else if i.unsigned {
			c.Type = zng.TypeUint64
		}
		cols = append(cols, c)
	}
	return cols
}

func (a *anchor) needRecode() []zng.Column {
	for k := range a.typ.Columns {
		i := a.integers[k]
		if i.signed || i.unsigned {
			return a.recodeType()
		}
	}
	return nil
}

func NewShaper(zctx *resolver.Context, memMaxBytes int) *Shaper {
	return &Shaper{
		zctx:        zctx,
		memMaxBytes: memMaxBytes,
		anchors:     make(map[uint64]*anchor),
		typeAnchor:  make(map[zng.Type]*anchor),
		recode:      make(map[zng.Type]*zng.TypeRecord),
	}
}

// Close removes the receiver's temporary file if it created one.
func (s *Shaper) Close() error {
	if s.spiller != nil {
		return s.spiller.CloseAndRemove()
	}
	return nil
}

func hash(h *maphash.Hash, cols []zng.Column) uint64 {
	h.Reset()
	for _, c := range cols {
		h.WriteString(c.Name)
	}
	return h.Sum64()
}

func (s *Shaper) lookupAnchor(columns []zng.Column) *anchor {
	h := hash(&s.hash, columns)
	for a := s.anchors[h]; a != nil; a = a.next {
		if a.match(columns) {
			return a
		}
	}
	return nil
}

func (s *Shaper) newAnchor(columns []zng.Column) *anchor {
	h := hash(&s.hash, columns)
	a := &anchor{columns: columns}
	a.next = s.anchors[h]
	s.anchors[h] = a
	a.integers = make([]integer, len(columns))
	for k := range columns {
		// start off as int64 and invalidate when we see
		// a value that doesn't fit.
		a.integers[k].unsigned = true
		a.integers[k].signed = true
	}
	return a
}

func (s *Shaper) update(rec *zng.Record) {
	if a, ok := s.typeAnchor[rec.Type]; ok {
		a.updateInts(rec)
		return
	}
	a := s.lookupAnchor(rec.Type.Columns)
	if a == nil {
		a = s.newAnchor(rec.Type.Columns)
	} else {
		a.mixIn(rec.Type.Columns)
	}
	a.updateInts(rec)
	s.typeAnchor[rec.Type] = a
}

func (s *Shaper) needRecode(typ *zng.TypeRecord) (*zng.TypeRecord, error) {
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

func (s *Shaper) lookupType(in *zng.TypeRecord) (*zng.TypeRecord, error) {
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
func (s *Shaper) Write(rec *zng.Record) error {
	if s.spiller != nil {
		return s.spiller.Write(rec)
	}
	if err := s.stash(rec); err != nil {
		return err
	}
	s.update(rec)
	return nil
}

func (s *Shaper) stash(rec *zng.Record) error {
	s.nbytes += len(rec.Raw)
	if s.nbytes >= s.memMaxBytes {
		var err error
		s.spiller, err = spill.NewTempFile()
		if err != nil {
			return err
		}
		for _, rec := range s.recs {
			if err := s.spiller.Write(rec); err != nil {
				return err
			}
		}
		s.recs = nil
		return s.spiller.Write(rec)
	}
	rec = rec.Keep()
	s.recs = append(s.recs, rec)
	return nil
}

func (s *Shaper) Read() (*zng.Record, error) {
	rec, err := s.next()
	if rec == nil || err != nil {
		return nil, err
	}
	typ, err := s.lookupType(rec.Type)
	if err != nil {
		return nil, err
	}
	bytes := rec.Raw
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
	return zng.NewRecordFromType(typ, bytes), nil
}

func recode(from, to []zng.Column, bytes zcode.Bytes) (zcode.Bytes, error) {
	out := make(zcode.Bytes, 0, len(bytes))
	it := bytes.Iter()
	for k, fromCol := range from {
		b, container, err := it.Next()
		if err != nil {
			return nil, err
		}
		toType := to[k].Type
		if fromCol.Type != toType && b != nil {
			if fromCol.Type != zng.TypeFloat64 {
				return nil, errors.New("shape: can't recode from non float64")
			}
			f, _ := zng.DecodeFloat64(b)
			if toType == zng.TypeInt64 {
				b = zng.EncodeInt(int64(f))
			} else if toType == zng.TypeUint64 {
				b = zng.EncodeUint(uint64(f))
			} else {
				return nil, errors.New("internal error: can't recode from to non-integer")
			}
		}
		out = zcode.AppendAs(out, container, b)
	}
	return out, nil
}

func (s *Shaper) next() (*zng.Record, error) {
	if s.spiller != nil {
		return s.spiller.Read()
	}
	var rec *zng.Record
	if len(s.recs) > 0 {
		rec = s.recs[0]
		s.recs = s.recs[1:]
	}
	return rec, nil

}

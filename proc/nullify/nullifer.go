package nullify

import (
	"errors"
	"hash/maphash"

	"github.com/brimsec/zq/proc/spill"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Nullifier struct {
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

func NewNullifier(zctx *resolver.Context, memMaxBytes int) *Nullifier {
	return &Nullifier{
		zctx:        zctx,
		memMaxBytes: memMaxBytes,
		anchors:     make(map[uint64]*anchor),
		typeAnchor:  make(map[zng.Type]*anchor),
		recode:      make(map[zng.Type]*zng.TypeRecord),
	}
}

// Close removes the receiver's temporary file if it created one.
func (n *Nullifier) Close() error {
	if n.spiller != nil {
		return n.spiller.CloseAndRemove()
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

func (n *Nullifier) lookupAnchor(columns []zng.Column) *anchor {
	h := hash(&n.hash, columns)
	for a := n.anchors[h]; a != nil; a = a.next {
		if a.match(columns) {
			return a
		}
	}
	return nil
}

func (n *Nullifier) newAnchor(columns []zng.Column) *anchor {
	h := hash(&n.hash, columns)
	a := &anchor{columns: columns}
	a.next = n.anchors[h]
	n.anchors[h] = a
	a.integers = make([]integer, len(columns))
	for k := range columns {
		// start off as int64 and invalidate when we see
		// a value that doesn't fit.
		a.integers[k].unsigned = true
		a.integers[k].signed = true
	}
	return a
}

func (n *Nullifier) update(rec *zng.Record) {
	if a, ok := n.typeAnchor[rec.Type]; ok {
		a.updateInts(rec)
		return
	}
	a := n.lookupAnchor(rec.Type.Columns)
	if a == nil {
		a = n.newAnchor(rec.Type.Columns)
	} else {
		a.mixIn(rec.Type.Columns)
	}
	a.updateInts(rec)
	n.typeAnchor[rec.Type] = a
}

func (n *Nullifier) needRecode(typ *zng.TypeRecord) (*zng.TypeRecord, error) {
	target, ok := n.recode[typ]
	if !ok {
		a := n.typeAnchor[typ]
		cols := a.needRecode()
		if cols != nil {
			var err error
			target, err = n.zctx.LookupTypeRecord(cols)
			if err != nil {
				return nil, err
			}
		}
		n.recode[typ] = target
	}
	return target, nil
}

func (n *Nullifier) lookupType(in *zng.TypeRecord) (*zng.TypeRecord, error) {
	a, ok := n.typeAnchor[in]
	if !ok {
		return nil, errors.New("nullifier: unencountered type (this is a bug)")
	}
	typ := a.typ
	if typ == nil {
		var err error
		typ, err = n.zctx.LookupTypeRecord(a.columns)
		if err != nil {
			return nil, err
		}
		a.typ = typ
	}
	return typ, nil
}

// Write buffers rec. If called after Read, Write panics.
func (n *Nullifier) Write(rec *zng.Record) error {
	if n.spiller != nil {
		return n.spiller.Write(rec)
	}
	if err := n.stash(rec); err != nil {
		return err
	}
	n.update(rec)
	return nil
}

func (n *Nullifier) stash(rec *zng.Record) error {
	n.nbytes += len(rec.Raw)
	if n.nbytes >= n.memMaxBytes {
		var err error
		n.spiller, err = spill.NewTempFile()
		if err != nil {
			return err
		}
		for _, rec := range n.recs {
			if err := n.spiller.Write(rec); err != nil {
				return err
			}
		}
		n.recs = nil
		return n.spiller.Write(rec)
	}
	rec = rec.Keep()
	n.recs = append(n.recs, rec)
	return nil
}

func (n *Nullifier) Read() (*zng.Record, error) {
	rec, err := n.next()
	if rec == nil || err != nil {
		return nil, err
	}
	typ, err := n.lookupType(rec.Type)
	if err != nil {
		return nil, err
	}
	bytes := rec.Raw
	targetType, err := n.needRecode(rec.Type)
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
				return nil, errors.New("nullify: can't recode from non float64")
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

func (n *Nullifier) next() (*zng.Record, error) {
	if n.spiller != nil {
		return n.spiller.Read()
	}
	var rec *zng.Record
	if len(n.recs) > 0 {
		rec = n.recs[0]
		n.recs = n.recs[1:]
	}
	return rec, nil

}

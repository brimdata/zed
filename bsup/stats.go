package bsup

import (
	"encoding/binary"
	"errors"
	"slices"

	"github.com/superdb/super"
	"github.com/superdb/super/pkg/field"
	"github.com/superdb/super/scode"
)

var errBadStatsSection = errors.New("corrupt stats section")

type CollectStatsFunc func(path field.Path, min, max super.Value)

type ColumnStats struct {
	Path field.Path
	Min  super.Value
	Max  super.Value
}

func statsValue(sctx *super.Context, projection field.List, stats []ColumnStats) super.Value {
	stats = slices.DeleteFunc(stats, func(stat ColumnStats) bool {
		return !projection.Has(append(stat.Path, "min")) && !projection.Has(append(stat.Path, "max"))
	})
	var paths field.List
	var bytes [][]byte
	var types []super.Type
	for _, s := range stats {
		paths = append(paths, append(slices.Clone(s.Path), "min"), append(slices.Clone(s.Path), "max"))
		types = append(types, s.Min.Type(), s.Max.Type())
		bytes = append(bytes, s.Min.Bytes(), s.Max.Bytes())
	}
	b, err := super.NewRecordBuilder(sctx, paths)
	if err != nil {
		panic(err)
	}
	for _, leaf := range bytes {
		b.Append(leaf)
	}
	bb, err := b.Encode()
	if err != nil {
		panic(err)
	}
	return super.NewValue(b.Type(types), bb)
}

func DecodeStats(b []byte) ([]uint64, [][]ColumnStats, error) {
	it := scode.Bytes(b).Iter()
	var frameStats [][]ColumnStats
	var frameLocs []uint64
	for !it.Done() {
		loc, stats, err := decodeFrameStats(it.Next())
		if err != nil {
			return nil, nil, err
		}
		frameLocs = append(frameLocs, loc)
		frameStats = append(frameStats, stats)
	}
	return frameLocs, frameStats, nil
}

func decodeFrameStats(b scode.Bytes) (uint64, []ColumnStats, error) {
	it := b.Iter()
	loc := binary.LittleEndian.Uint64(it.Next())
	var stats []ColumnStats
	for !it.Done() {
		path := decodePath(it.Next())
		min, max, err := decodeMinMax(&it)
		if err != nil {
			return loc, nil, err
		}
		stats = append(stats, ColumnStats{path, min, max})
	}
	return loc, stats, nil
}

func decodePath(b scode.Bytes) field.Path {
	var p field.Path
	it := b.Iter()
	for !it.Done() {
		p = append(p, string(it.Next()))
	}
	return p
}

func decodeMinMax(it *scode.Iter) (super.Value, super.Value, error) {
	if it.Done() {
		return super.None, super.None, errBadStatsSection
	}
	id := binary.LittleEndian.Uint16(it.Next())
	typ, err := super.LookupPrimitiveByID(int(id))
	if it.Done() || err != nil {
		return super.None, super.None, errBadStatsSection
	}
	min := super.NewValue(typ, it.Next())
	if it.Done() {
		return super.None, super.None, errBadStatsSection
	}
	id = binary.LittleEndian.Uint16(it.Next())
	typ, err = super.LookupPrimitiveByID(int(id))
	if it.Done() || err != nil {
		return super.None, super.None, errBadStatsSection
	}
	max := super.NewValue(typ, it.Next())
	return min, max, nil
}

func encodeStats(frameLocs []uint64, frameStats [][]ColumnStats) []byte {
	b := scode.NewBuilder()
	for i := range frameLocs {
		b.BeginContainer()
		encodeFrameStats(b, frameLocs[i], frameStats[i])
		b.EndContainer()
	}
	return b.Bytes()
}

func encodeFrameStats(b *scode.Builder, frameLoc uint64, frameStats []ColumnStats) {
	var tmp [8]byte
	binary.LittleEndian.PutUint64(tmp[:], frameLoc)
	b.Append(tmp[:])
	for _, s := range frameStats {
		encodePath(b, s.Path)
		encodeMinMax(b, s.Min, s.Max)
	}
}

func encodePath(b *scode.Builder, path field.Path) {
	b.BeginContainer()
	for _, n := range path {
		b.Append([]byte(n))
	}
	b.EndContainer()
}

func encodeMinMax(b *scode.Builder, min, max super.Value) {
	var id [2]byte
	binary.LittleEndian.PutUint16(id[:], uint16(min.Type().ID()))
	b.Append(id[:])
	b.Append(min.Bytes())
	binary.LittleEndian.PutUint16(id[:], uint16(max.Type().ID()))
	b.Append(id[:])
	b.Append(max.Bytes())
}

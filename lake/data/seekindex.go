package data

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake/seekindex"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
)

func LookupSeekRange(ctx context.Context, engine storage.Engine, path *storage.URI,
	obj *Object, pruner expr.Evaluator) ([]seekindex.Range, error) {
	if pruner == nil {
		// scan whole object
		return nil, nil
	}
	r, err := engine.Get(ctx, obj.SeekIndexURI(path))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var ranges []seekindex.Range
	var rg *seekindex.Range
	unmarshaler := zson.NewZNGUnmarshaler()
	reader := zngio.NewReader(zed.NewContext(), r)
	defer reader.Close()
	ectx := expr.NewContext()
	for {
		val, err := reader.Read()
		if val == nil || err != nil {
			return ranges, err
		}
		result := pruner.Eval(ectx, val)
		if result.Type == zed.TypeBool && zed.IsTrue(result.Bytes) {
			rg = nil
			continue
		}
		var entry seekindex.Entry
		if err := unmarshaler.Unmarshal(val, &entry); err != nil {
			return nil, fmt.Errorf("corrupt seek index entry for %q at value: %q (%w)", obj.ID.String(), zson.String(val), err)
		}
		if rg == nil {
			ranges = append(ranges, seekindex.Range{Offset: int64(entry.Offset)})
			rg = &ranges[len(ranges)-1]
		}
		rg.Length += int64(entry.Length)
	}
}

func FetchSeekIndex(ctx context.Context, engine storage.Engine, path *storage.URI, obj *Object) ([]seekindex.Entry, error) {
	r, err := engine.Get(ctx, obj.SeekIndexURI(path))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// If there is no seek just return a single entry representing the
			// entire object.
			return []seekindex.Entry{{
				Min:    &obj.Min,
				Max:    &obj.Max,
				ValOff: 0,
				ValCnt: obj.Count,
				Offset: 0,
				Length: uint64(obj.Size),
			}}, nil
		}
		return nil, err
	}
	defer r.Close()
	unmarshaler := zson.NewZNGUnmarshaler()
	reader := zngio.NewReader(zed.NewContext(), r)
	defer reader.Close()
	var entries []seekindex.Entry
	for {
		val, err := reader.Read()
		if val == nil || err != nil {
			return entries, err
		}
		var entry seekindex.Entry
		if err := unmarshaler.Unmarshal(val, &entry); err != nil {
			return nil, fmt.Errorf("corrupt seek index entry for %q at value: %q (%w)", obj.ID.String(), zson.String(val), err)
		}
		entries = append(entries, entry)
	}
}

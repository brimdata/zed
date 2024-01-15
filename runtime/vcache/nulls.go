package vcache

import (
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/vector"
	"github.com/brimdata/zed/vng"
)

func (l *loader) loadNulls(any *vector.Any, typ zed.Type, path field.Path, m *vng.Nulls) (vector.Any, error) {
	// The runlengths are typically small so we load them with the metadata
	// and don't bother waiting for a reference.
	runlens := vng.NewInt64Decoder(m.Runs, l.r) //XXX 32-bit reader?
	var null bool
	var off int
	var slots []uint32
	// In zed, nulls are generally bad and not really needed because we don't
	// need super-wide uber schemas with lots of nulls.
	for {
		run, err := runlens.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if null {
			for i := 0; int64(i) < run; i++ {
				slots = append(slots, uint32(off+i))
			}
		}
		off += int(run)
		null = !null
	}
	var values vector.Any
	if _, err := l.loadVector(&values, typ, path, m.Values); err != nil {
		return nil, err
	}
	*any = vector.NewNulls(slots, off, values)
	return *any, nil
}

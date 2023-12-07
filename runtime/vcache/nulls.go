package vcache

import (
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/vector"
	meta "github.com/brimdata/zed/vng/vector"
)

func loadNulls(any *vector.Any, typ zed.Type, path field.Path, m *meta.Nulls, r io.ReaderAt) (vector.Any, error) {
	// The runlengths are typically small so we load them with the metadata
	// and don't bother waiting for a reference.
	runlens := meta.NewInt64Reader(m.Runs, r) //XXX 32-bit reader?
	var off, nulls uint32
	null := true
	//XXX finish this loop... need to remove slots covered by nulls and subtract
	// cumulative number of nulls for each surviving value slot.
	// In zed, nulls are generally bad and not really needed because we don't
	// need super-wide uber schemas with lots of nulls.
	for {
		//XXX need nullslots array to build vector.Nullmask and need a way to pass down Nullmask XXX
		run, err := runlens.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		off += uint32(run)
		if null {
			nulls += uint32(run)
		}
		null = !null
	}
	//newSlots := slots //XXX need to create this above
	return loadVector(any, typ, path, m.Values, r)
}

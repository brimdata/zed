package op

import (
	"bytes"

	"github.com/brimdata/super/runtime"
	"github.com/brimdata/super/runtime/sam/expr"
	"github.com/brimdata/super/runtime/sam/op/sort"
	"github.com/brimdata/super/runtime/vam"
	"github.com/brimdata/super/runtime/vcache"
	"github.com/brimdata/super/vector"
	"github.com/brimdata/super/vng"
	"github.com/brimdata/super/zbuf"
	"github.com/brimdata/super/zio"
)

type Sort struct {
	rctx    *runtime.Context
	samsort *sort.Op
}

func NewSort(rctx *runtime.Context, parent vector.Puller, fields []expr.SortEvaluator, nullsFirst, reverse bool, resetter expr.Resetter) *Sort {
	materializer := vam.NewMaterializer(parent)
	s := sort.New(rctx, materializer, fields, nullsFirst, reverse, resetter)
	return &Sort{rctx: rctx, samsort: s}
}

func (s *Sort) Pull(done bool) (vector.Any, error) {
	batch, err := s.samsort.Pull(done)
	if batch == nil || err != nil {
		return nil, err
	}
	return s.convertBatchToVec(batch)
}

func (s *Sort) convertBatchToVec(batch zbuf.Batch) (vector.Any, error) {
	var buf bytes.Buffer
	w := vng.NewWriter(zio.NopCloser(&buf))
	for _, val := range batch.Values() {
		w.Write(val)
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	o, err := vng.NewObject(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return nil, err
	}
	return vcache.NewObjectFromVNG(o).Fetch(s.rctx.Zctx, nil)
}

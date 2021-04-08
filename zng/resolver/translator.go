package resolver

import (
	"sync"

	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

type Translator struct {
	mu sync.Mutex
	Slice
	inputCtx  *zson.Context
	outputCtx *zson.Context
}

func NewTranslator(in, out *zson.Context) *Translator {
	return &Translator{
		inputCtx:  in,
		outputCtx: out,
	}
}

// Lookup implements zng.Resolver
func (t *Translator) Lookup(id int) (zng.Type, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	outputType := t.Slice.Lookup(id)
	var err error
	if outputType == nil {
		inputType := t.inputCtx.Lookup(id)
		if inputType == nil {
			return nil, nil
		}
		outputType, err = t.outputCtx.TranslateTypeRecord(inputType)
		if err != nil {
			return nil, err
		}
		t.Slice.Enter(id, outputType)
	}
	return outputType, nil
}
